// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/internal/services/job/waiter"
)

type Service struct {
	c   GenJobClient
	cfg *config.Config
	log *slog.Logger
}

func (s *Service) Client() GenJobClient   { return s.c }
func (s *Service) Config() *config.Config { return s.cfg }
func (s *Service) Logger() *slog.Logger   { return s.log }

type Option func(*Service)

func WithClient(c GenJobClient) Option {
	return func(s *Service) {
		if c != nil {
			s.c = c
		}
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.log = logger
		}
	}
}

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	deps, err := common.PrepareServiceDeps(cfg, httpClient, log)
	if err != nil {
		return nil, err
	}
	c, err := camundav89.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav89.WithHTTPClient(deps.HTTPClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{c: c, cfg: deps.Config, log: deps.Logger}
	for _, opt := range opts {
		opt(s)
	}
	logger, err := common.EnsureLoggerAndClients(s.log, s.c)
	if err != nil {
		return nil, err
	}
	s.log = logger
	return s, nil
}

func (s *Service) GetJob(ctx context.Context, key string, opts ...services.CallOption) (d.Job, error) {
	_ = services.ApplyCallOptions(opts)

	jobKeyFilter, err := newJobKeyEqFilterPtr(key)
	if err != nil {
		return d.Job{}, fmt.Errorf("building job key filter: %w", err)
	}
	page := newSearchQueryPageRequest(0, 2)
	resp, err := s.c.SearchJobsWithResponse(ctx, camundav89.SearchJobsJSONRequestBody{
		Filter: &camundav89.JobFilter{
			JobKey: jobKeyFilter,
		},
		Page: &page,
	})
	if err != nil {
		return d.Job{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.Job{}, err
	}
	return requireSingleJob(payload.Items, key)
}

// SearchJobs collects all selected job pages using service-owned offset
// traversal and returns the mapped rows.
func (s *Service) SearchJobs(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (d.JobSearchResult, error) {
	result, err := s.SearchJobsPages(ctx, query, nil, opts...)
	if err != nil {
		return d.JobSearchResult{}, err
	}
	return d.JobSearchResult{
		Items: result.Items,
		Limit: result.Limit,
	}, nil
}

// SearchJobsPages owns raw cursor/offset advancement, per-page size capping, and user
// limit trimming while allowing callers to render or prompt after each page.
func (s *Service) SearchJobsPages(ctx context.Context, query d.JobSearchQuery, visitor d.JobSearchPageVisitor, opts ...services.CallOption) (d.JobSearchPagesResult, error) {
	_ = services.ApplyCallOptions(opts)
	batchSize := query.BatchSize
	if batchSize <= 0 {
		batchSize = consts.MaxPISearchSize
	}
	limit := query.Limit
	items := make([]d.Job, 0, minPositiveJobSearchSize(batchSize, limit))
	from := int32(0)
	after := ""
	seenCursors := make(map[string]bool)
	backoff := s.cfg.App.Backoff
	_ = backoff.Normalize()
	pageCtx := ctx
	var cancelAmbiguity context.CancelFunc
	defer func() {
		if cancelAmbiguity != nil {
			cancelAmbiguity()
		}
	}()
	pages := int32(0)
	for {
		if err := ctx.Err(); err != nil {
			return d.JobSearchPagesResult{}, err
		}
		pageLimit := nextJobSearchPageLimit(batchSize, limit, int32(len(items)))
		if pageLimit <= 0 {
			break
		}
		page, err := s.SearchJobsPage(pageCtx, query, d.JobPageRequest{From: from, Size: pageLimit, After: after}, opts...)
		if err != nil {
			if pageCtx.Err() != nil {
				return d.JobSearchPagesResult{}, fmt.Errorf("%w: incomplete job search: %w", d.ErrMalformedResponse, pageCtx.Err())
			}
			return d.JobSearchPagesResult{}, err
		}
		if err := pageCtx.Err(); err != nil {
			return d.JobSearchPagesResult{}, fmt.Errorf("%w: incomplete job search: %w", d.ErrMalformedResponse, err)
		}
		if page.RawItemCount > 0 && cancelAmbiguity != nil {
			cancelAmbiguity()
			cancelAmbiguity, pageCtx = nil, ctx
		}
		if page.EndCursor != "" {
			if seenCursors[page.EndCursor] {
				return d.JobSearchPagesResult{}, fmt.Errorf("%w: incomplete job search: repeated cursor", d.ErrMalformedResponse)
			}
			seenCursors[page.EndCursor] = true
		}
		items = append(items, page.Items...)
		pages++
		limitReached := limit > 0 && int32(len(items)) >= limit
		if visitor != nil {
			action, err := visitor(d.JobSearchPageStep{
				Page:            page,
				CumulativeCount: int32(len(items)),
				LimitReached:    limitReached,
			})
			if err != nil {
				return d.JobSearchPagesResult{}, err
			}
			if action == d.JobSearchPageActionStop {
				break
			}
		}
		if limitReached {
			break
		}
		if page.OverflowState != d.ProcessInstanceOverflowStateHasMore {
			break
		}
		if page.RawItemCount == 0 && cancelAmbiguity == nil {
			// Advancing cursors can bridge empty pages, but cannot defer evidence forever.
			pageCtx, cancelAmbiguity = context.WithTimeout(ctx, backoff.Timeout)
		}
		from, after = page.NextFrom, page.EndCursor
	}
	return d.JobSearchPagesResult{
		Items: items,
		Limit: limit,
		Pages: pages,
	}, nil
}

// SearchJobsPage normalizes raw continuation before defensively filtering tenant rows.
// Unsupported or contradictory continuation fails explicitly instead of claiming exhaustion.
func (s *Service) SearchJobsPage(ctx context.Context, query d.JobSearchQuery, pageReq d.JobPageRequest, opts ...services.CallOption) (d.JobSearchPage, error) {
	_ = services.ApplyCallOptions(opts)

	filter, err := newJobSearchFilter(query, common.EffectiveTenant(s.cfg))
	if err != nil {
		return d.JobSearchPage{}, err
	}
	pageSize := pageReq.Size
	if pageSize <= 0 {
		pageSize = consts.MaxPISearchSize
	}
	if pageReq.From < 0 {
		return d.JobSearchPage{}, fmt.Errorf("%w: incomplete job search: negative offset", d.ErrMalformedResponse)
	}
	page := newSearchQueryPageRequest(pageReq.From, pageSize)
	if pageReq.After != "" {
		if err := page.FromCursorForwardPagination(camundav89.CursorForwardPagination{After: pageReq.After, Limit: &pageSize}); err != nil {
			return d.JobSearchPage{}, err
		}
	}
	body := camundav89.SearchJobsJSONRequestBody{Filter: filter, Page: &page}
	if query.SortByKey {
		order := camundav89.SortOrderEnum("ASC")
		body.Sort = &[]camundav89.JobSearchQuerySortRequest{{Field: "jobKey", Order: &order}}
	}
	resp, err := s.c.SearchJobsWithResponse(ctx, body)
	if err != nil {
		return d.JobSearchPage{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.JobSearchPage{}, err
	}
	items := trimJobSearchPageResults(payload.Items, payload.Page, pageReq.From, pageSize)
	nextFrom := int64(pageReq.From) + int64(len(items))
	if payload.Page.TotalItems < 0 || (!payload.Page.HasMoreTotalItems && nextFrom > payload.Page.TotalItems) || nextFrom > math.MaxInt32 {
		return d.JobSearchPage{}, fmt.Errorf("%w: incomplete job search: inconsistent total or offset", d.ErrMalformedResponse)
	}
	overflow := pickJobSearchOverflowState(payload.Page, pageReq.From, len(items))
	endCursor := stringPtrValue(payload.Page.EndCursor)
	if len(payload.Items) > len(items) {
		// An overfull response cursor points past rows we deliberately did not consume.
		endCursor = ""
	}
	if (endCursor != "" && endCursor == pageReq.After) ||
		(overflow == d.ProcessInstanceOverflowStateHasMore && endCursor == "" && len(items) == 0) {
		return d.JobSearchPage{}, fmt.Errorf("%w: incomplete job search: nonadvancing continuation", d.ErrMalformedResponse)
	}
	selected := fromJobSearchResults(items)
	if tenant := common.EffectiveTenant(s.cfg); tenant != "" {
		scoped := make([]d.Job, 0, len(selected))
		for _, item := range selected {
			if item.TenantId == tenant {
				scoped = append(scoped, item)
			}
		}
		selected = scoped
	}
	return d.JobSearchPage{
		Items:         selected,
		Request:       d.JobPageRequest{From: pageReq.From, Size: pageSize, After: pageReq.After},
		OverflowState: overflow,
		ReportedTotal: newJobReportedTotal(payload.Page.TotalItems, payload.Page.HasMoreTotalItems),
		RawItemCount:  int32(len(payload.Items)),
		NextFrom:      int32(nextFrom),
		EndCursor:     endCursor,
	}, nil
}

// SearchJobsTotal uses exact unfiltered totals, otherwise counting selected pages
// so defensive tenant filtering cannot inflate a scoped count.
func (s *Service) SearchJobsTotal(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (int64, error) {
	var total int64
	_, err := s.SearchJobsPages(ctx, query, func(step d.JobSearchPageStep) (d.JobSearchPageAction, error) {
		if common.EffectiveTenant(s.cfg) == "" && step.Page.ReportedTotal != nil && step.Page.ReportedTotal.Kind == d.JobReportedTotalKindExact {
			total = step.Page.ReportedTotal.Count
			return d.JobSearchPageActionStop, nil
		}
		total += int64(len(step.Page.Items))
		return d.JobSearchPageActionContinue, nil
	}, opts...)
	if err != nil {
		return 0, err
	}
	return total, nil
}

// minPositiveJobSearchSize sizes the initial result slice for bounded searches.
func minPositiveJobSearchSize(batchSize int32, limit int32) int {
	if limit > 0 && limit < batchSize {
		return int(limit)
	}
	return int(batchSize)
}

// nextJobSearchPageLimit caps the next page request at the remaining caller limit.
func nextJobSearchPageLimit(batchSize int32, limit int32, loaded int32) int32 {
	if limit <= 0 {
		return batchSize
	}
	remaining := limit - loaded
	if remaining < batchSize {
		return remaining
	}
	return batchSize
}

// trimJobSearchPageResults protects callers from backend responses that include
// more rows than the requested page size.
func trimJobSearchPageResults(items []camundav89.JobSearchResult, page camundav89.SearchQueryPageResponse, from int32, pageSize int32) []camundav89.JobSearchResult {
	if pageSize <= 0 || len(items) <= int(pageSize) {
		return items
	}
	if page.TotalItems == int64(len(items)) && from > 0 {
		start := int(from)
		if start >= len(items) {
			return nil
		}
		items = items[start:]
	}
	if len(items) > int(pageSize) {
		return items[:pageSize]
	}
	return items
}

// pickJobSearchOverflowState normalizes Camunda total metadata into the
// version-neutral continuation state used by higher layers.
func pickJobSearchOverflowState(page camundav89.SearchQueryPageResponse, from int32, itemCount int) d.ProcessInstanceOverflowState {
	// A capped total still permits exhaustion after its lower bound has been passed.
	// Empty pages below that bound (or with a cursor) need continuation evidence.
	if itemCount == 0 && page.HasMoreTotalItems && int64(from) > page.TotalItems && stringPtrValue(page.EndCursor) == "" {
		return d.ProcessInstanceOverflowStateNoMore
	}
	nextFrom := int64(from) + int64(itemCount)
	if page.TotalItems > nextFrom || page.HasMoreTotalItems {
		return d.ProcessInstanceOverflowStateHasMore
	}
	return d.ProcessInstanceOverflowStateNoMore
}

// newJobReportedTotal preserves exact vs lower-bound total metadata.
func newJobReportedTotal(count int64, lowerBound bool) *d.JobReportedTotal {
	kind := d.JobReportedTotalKindExact
	if lowerBound {
		kind = d.JobReportedTotalKindLowerBound
	}
	return &d.JobReportedTotal{Count: count, Kind: kind}
}

func (s *Service) UpdateJob(ctx context.Context, request d.JobUpdateRequest, opts ...services.CallOption) (d.JobUpdateResult, error) {
	cCfg := services.ApplyCallOptions(opts)
	result := d.JobUpdateResult{
		Key:                  request.Key,
		SubmittedRetries:     request.Retries,
		SubmittedTimeoutMS:   request.TimeoutMillis,
		ConfirmationStatus:   "not_applicable",
		UnsupportedOperation: false,
	}
	body := camundav89.UpdateJobJSONRequestBody{
		Changeset: camundav89.JobChangeset{
			Retries: request.Retries,
			Timeout: request.TimeoutMillis,
		},
	}
	resp, err := services.RetryCamundaMutation(ctx, s.log, "update job", func(ctx context.Context) (*camundav89.UpdateJobResponse, *http.Response, []byte, error) {
		resp, err := s.c.UpdateJobWithResponse(ctx, camundav89.JobKey(request.Key), body)
		if resp == nil {
			return resp, nil, nil, err
		}
		return resp, resp.HTTPResponse, resp.Body, err
	})
	if err != nil {
		result.MutationError = err.Error()
		return result, err
	}
	result.MutationAccepted = true
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		result.MutationAccepted = false
		result.MutationError = err.Error()
		return result, err
	}
	if cCfg.NoWait || request.SkipConfirmation || !request.ConfirmRetries || request.Retries == nil {
		result.ConfirmationStatus = "skipped"
		return result, nil
	}
	confirmed, err := waiter.WaitForRetries(ctx, s, s.cfg, s.log, request.Key, *request.Retries, opts...)
	if err != nil {
		result.ConfirmationStatus = "failed"
		result.ConfirmationError = err.Error()
		return result, nil
	}
	result.ConfirmationStatus = "confirmed"
	result.ConfirmedRetries = &confirmed.Retries
	return result, nil
}

func (s *Service) SubmitJobWorkerOutcome(ctx context.Context, request d.JobWorkerOutcomeRequest, opts ...services.CallOption) (d.JobWorkerOutcomeResult, error) {
	_ = services.ApplyCallOptions(opts)
	switch request.Mode {
	case d.JobWorkerOutcomeTechnicalFailure:
		return s.submitJobTechnicalFailure(ctx, request)
	case d.JobWorkerOutcomeBPMNError:
		return s.submitJobBPMNError(ctx, request)
	case d.JobWorkerOutcomeCompletion:
		return s.submitJobCompletion(ctx, request)
	default:
		return d.JobWorkerOutcomeResult{}, fmt.Errorf("%w: job worker outcome service implementation is pending", d.ErrUnsupported)
	}
}

func (s *Service) submitJobTechnicalFailure(ctx context.Context, request d.JobWorkerOutcomeRequest) (d.JobWorkerOutcomeResult, error) {
	result := d.JobWorkerOutcomeResult{
		Key:                request.Key,
		Mode:               request.Mode,
		SubmittedRetries:   request.Retries,
		SubmittedBackoffMS: request.RetryBackoffMillis,
		ConfirmationStatus: "not_applicable",
	}
	body := newFailJobRequestBody(request)
	resp, err := services.RetryCamundaMutation(ctx, s.log, "fail job", func(ctx context.Context) (*camundav89.FailJobResponse, *http.Response, []byte, error) {
		resp, err := s.c.FailJobWithResponse(ctx, camundav89.JobKey(request.Key), body)
		if resp == nil {
			return resp, nil, nil, err
		}
		return resp, resp.HTTPResponse, resp.Body, err
	})
	if err != nil {
		result.MutationError = err.Error()
		return result, err
	}
	result.MutationAccepted = true
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		result.MutationAccepted = false
		result.MutationError = err.Error()
		return result, err
	}
	result.ConfirmationStatus = "skipped"
	return result, nil
}

func (s *Service) submitJobBPMNError(ctx context.Context, request d.JobWorkerOutcomeRequest) (d.JobWorkerOutcomeResult, error) {
	result := d.JobWorkerOutcomeResult{
		Key:                request.Key,
		Mode:               request.Mode,
		SubmittedErrorCode: request.ErrorCode,
		ConfirmationStatus: "not_applicable",
	}
	body := newThrowJobErrorRequestBody(request)
	resp, err := services.RetryCamundaMutation(ctx, s.log, "throw job BPMN error", func(ctx context.Context) (*camundav89.ThrowJobErrorResponse, *http.Response, []byte, error) {
		resp, err := s.c.ThrowJobErrorWithResponse(ctx, camundav89.JobKey(request.Key), body)
		if resp == nil {
			return resp, nil, nil, err
		}
		return resp, resp.HTTPResponse, resp.Body, err
	})
	if err != nil {
		result.MutationError = err.Error()
		return result, err
	}
	result.MutationAccepted = true
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		result.MutationAccepted = false
		result.MutationError = err.Error()
		return result, err
	}
	result.ConfirmationStatus = "skipped"
	return result, nil
}

func (s *Service) submitJobCompletion(ctx context.Context, request d.JobWorkerOutcomeRequest) (d.JobWorkerOutcomeResult, error) {
	result := d.JobWorkerOutcomeResult{
		Key:                request.Key,
		Mode:               request.Mode,
		ConfirmationStatus: "not_applicable",
	}
	body := newCompleteJobRequestBody(request)
	resp, err := services.RetryCamundaMutation(ctx, s.log, "complete job", func(ctx context.Context) (*camundav89.CompleteJobResponse, *http.Response, []byte, error) {
		resp, err := s.c.CompleteJobWithResponse(ctx, camundav89.JobKey(request.Key), body)
		if resp == nil {
			return resp, nil, nil, err
		}
		return resp, resp.HTTPResponse, resp.Body, err
	})
	if err != nil {
		result.MutationError = err.Error()
		return result, err
	}
	result.MutationAccepted = true
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		result.MutationAccepted = false
		result.MutationError = err.Error()
		return result, err
	}
	result.ConfirmationStatus = "skipped"
	return result, nil
}

// newFailJobRequestBody keeps generated v8.9 failure request details inside the
// service adapter while preserving explicit zero retries.
func newFailJobRequestBody(request d.JobWorkerOutcomeRequest) camundav89.FailJobJSONRequestBody {
	body := camundav89.FailJobJSONRequestBody{
		Retries:      request.Retries,
		RetryBackOff: request.RetryBackoffMillis,
	}
	if request.Message != "" {
		body.ErrorMessage = &request.Message
	}
	if request.Variables != nil {
		body.Variables = &request.Variables
	}
	return body
}

func newThrowJobErrorRequestBody(request d.JobWorkerOutcomeRequest) camundav89.ThrowJobErrorJSONRequestBody {
	body := camundav89.ThrowJobErrorJSONRequestBody{
		ErrorCode: request.ErrorCode,
	}
	if request.Message != "" {
		body.ErrorMessage = &request.Message
	}
	if request.Variables != nil {
		body.Variables = &request.Variables
	}
	return body
}

func newCompleteJobRequestBody(request d.JobWorkerOutcomeRequest) camundav89.CompleteJobJSONRequestBody {
	variables := request.Variables
	if variables == nil {
		// Camunda's generated completion request requires a variables object even when no variables are added.
		variables = map[string]any{}
	}
	return camundav89.CompleteJobJSONRequestBody{
		Variables: &variables,
	}
}

var _ waiter.JobGetter = (*Service)(nil)
