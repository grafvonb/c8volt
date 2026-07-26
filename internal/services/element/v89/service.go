// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
)

// Service adapts Camunda 8.9 runtime element endpoints to the internal API.
type Service struct {
	c   GenElementClient
	cfg *config.Config
	log *slog.Logger
}

// Client returns the generated Camunda element client used by the v8.9 service.
func (s *Service) Client() GenElementClient { return s.c }

// Config returns the normalized service configuration used by the v8.9 element service.
func (s *Service) Config() *config.Config { return s.cfg }

// Logger returns the service logger used by the v8.9 element service.
func (s *Service) Logger() *slog.Logger { return s.log }

// Option customizes the v8.9 element service during construction.
type Option func(*Service)

// WithClient overrides the generated Camunda element client, primarily for service tests.
func WithClient(c GenElementClient) Option {
	return func(s *Service) {
		if c != nil {
			s.c = c
		}
	}
}

// WithLogger overrides the default logger for tests and callers that need custom logging.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Service) {
		if logger != nil {
			s.log = logger
		}
	}
}

// New prepares a v8.9 element service with the generated Camunda v2 client.
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

// GetElement will use the generated direct lookup endpoint once US1 fills the adapter.
func (s *Service) GetElement(ctx context.Context, key string, opts ...services.CallOption) (d.Element, error) {
	_ = services.ApplyCallOptions(opts)

	resp, err := s.c.GetElementInstanceWithResponse(ctx, camundav89.ElementInstanceKey(key))
	if err != nil {
		return d.Element{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.Element{}, err
	}
	return fromElementInstanceResult(*payload), nil
}

// SearchElements collects all selected element pages using service-owned offset
// traversal and returns the mapped rows.
func (s *Service) SearchElements(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error) {
	result, err := s.SearchElementsPages(ctx, query, nil, opts...)
	if err != nil {
		return d.ElementSearchResult{}, err
	}
	return d.ElementSearchResult{
		Items: result.Items,
		Total: result.Total,
	}, nil
}

// SearchElementsPages owns offset advancement, per-page size capping, and user
// limit trimming while allowing callers to render or prompt after each page.
func (s *Service) SearchElementsPages(ctx context.Context, query d.ElementSearchQuery, visitor d.ElementSearchPageVisitor, opts ...services.CallOption) (d.ElementSearchPagesResult, error) {
	_ = services.ApplyCallOptions(opts)

	batchSize := query.BatchSize
	if batchSize <= 0 {
		batchSize = consts.MaxPISearchSize
	}
	limit := query.Limit
	items := make([]d.Element, 0, minPositiveElementSearchSize(batchSize, limit))
	from := int32(0)
	pages := int32(0)
	for {
		pageLimit := nextElementSearchPageLimit(batchSize, limit, int32(len(items)))
		if pageLimit <= 0 {
			break
		}
		page, err := s.SearchElementsPage(ctx, query, d.ElementPageRequest{From: from, Size: pageLimit}, opts...)
		if err != nil {
			return d.ElementSearchPagesResult{}, err
		}
		items = append(items, page.Items...)
		pages++
		limitReached := limit > 0 && int32(len(items)) >= limit
		if visitor != nil {
			action, err := visitor(d.ElementSearchPageStep{
				Page:            page,
				CumulativeCount: int32(len(items)),
				LimitReached:    limitReached,
			})
			if err != nil {
				return d.ElementSearchPagesResult{}, err
			}
			if action == d.ElementSearchPageActionStop {
				break
			}
		}
		if limitReached {
			break
		}
		if page.OverflowState != d.ProcessInstanceOverflowStateHasMore {
			break
		}
		from = nextElementSearchPageOffset(from, page)
	}
	return d.ElementSearchPagesResult{
		Items: items,
		Total: int32(len(items)),
		Pages: pages,
	}, nil
}

// SearchElementsPage fetches one runtime element search page.
func (s *Service) SearchElementsPage(ctx context.Context, query d.ElementSearchQuery, pageReq d.ElementPageRequest, opts ...services.CallOption) (d.ElementSearchPage, error) {
	_ = services.ApplyCallOptions(opts)

	filter, err := newElementSearchFilter(query)
	if err != nil {
		return d.ElementSearchPage{}, err
	}
	pageSize := pageReq.Size
	if pageSize <= 0 {
		pageSize = consts.MaxPISearchSize
	}
	page := newSearchQueryPageRequest(pageReq.From, pageSize)
	resp, err := s.c.SearchElementInstancesWithResponse(ctx, camundav89.SearchElementInstancesJSONRequestBody{
		Filter: filter,
		Page:   &page,
	})
	if err != nil {
		return d.ElementSearchPage{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.ElementSearchPage{}, err
	}
	items := trimElementSearchPageResults(payload.Items, payload.Page, pageReq.From, pageSize)
	return d.ElementSearchPage{
		Items:         fromElementInstanceResults(items),
		Request:       d.ElementPageRequest{From: pageReq.From, Size: pageSize},
		OverflowState: pickElementSearchOverflowState(payload.Page, pageReq.From, len(items), pageSize),
		ReportedTotal: newElementReportedTotal(payload.Page.TotalItems, payload.Page.HasMoreTotalItems),
	}, nil
}

// SearchElementsTotal returns the exact backend total when available and
// otherwise falls back to service-owned page counting.
func (s *Service) SearchElementsTotal(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (int64, error) {
	var total int64
	_, err := s.SearchElementsPages(ctx, query, func(step d.ElementSearchPageStep) (d.ElementSearchPageAction, error) {
		if step.Page.ReportedTotal != nil && step.Page.ReportedTotal.Kind == d.ElementReportedTotalKindExact {
			total = step.Page.ReportedTotal.Count
			return d.ElementSearchPageActionStop, nil
		}
		total += int64(len(step.Page.Items))
		return d.ElementSearchPageActionContinue, nil
	}, opts...)
	if err != nil {
		return 0, err
	}
	return total, nil
}

// nextElementSearchPageOffset advances offset searches without looping forever
// on empty pages that still advertise more data.
func nextElementSearchPageOffset(from int32, page d.ElementSearchPage) int32 {
	if len(page.Items) == 0 {
		return from + page.Request.Size
	}
	return from + int32(len(page.Items))
}

// minPositiveElementSearchSize sizes the initial result slice for bounded searches.
func minPositiveElementSearchSize(batchSize int32, limit int32) int {
	if limit > 0 && limit < batchSize {
		return int(limit)
	}
	return int(batchSize)
}

// nextElementSearchPageLimit caps the next page request at the remaining caller limit.
func nextElementSearchPageLimit(batchSize int32, limit int32, loaded int32) int32 {
	if limit <= 0 {
		return batchSize
	}
	remaining := limit - loaded
	if remaining < batchSize {
		return remaining
	}
	return batchSize
}

// trimElementSearchPageResults protects callers from backend responses that
// include more rows than the requested page size.
func trimElementSearchPageResults(items []camundav89.ElementInstanceResult, page camundav89.SearchQueryPageResponse, from int32, pageSize int32) []camundav89.ElementInstanceResult {
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

// pickElementSearchOverflowState normalizes Camunda total metadata into the
// version-neutral continuation state used by higher layers.
func pickElementSearchOverflowState(page camundav89.SearchQueryPageResponse, from int32, itemCount int, pageSize int32) d.ProcessInstanceOverflowState {
	if itemCount == 0 {
		return d.ProcessInstanceOverflowStateNoMore
	}
	nextFrom := int64(from) + int64(itemCount)
	if page.TotalItems > nextFrom {
		return d.ProcessInstanceOverflowStateHasMore
	}
	if page.HasMoreTotalItems && itemCount >= int(pageSize) {
		return d.ProcessInstanceOverflowStateHasMore
	}
	return d.ProcessInstanceOverflowStateNoMore
}

// newElementReportedTotal preserves exact vs lower-bound total metadata.
func newElementReportedTotal(count int64, lowerBound bool) *d.ElementReportedTotal {
	kind := d.ElementReportedTotalKindExact
	if lowerBound {
		kind = d.ElementReportedTotalKindLowerBound
	}
	return &d.ElementReportedTotal{Count: count, Kind: kind}
}
