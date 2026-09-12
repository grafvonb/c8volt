// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
)

// Service adapts Camunda 8.10 process-definition endpoints to the version-neutral API.
type Service struct {
	cc  GenProcessDefinitionClientCamunda
	cfg *config.Config
	log *slog.Logger
}

// ClientCamunda returns the generated Camunda process-definition client used by this service.
func (s *Service) ClientCamunda() GenProcessDefinitionClientCamunda { return s.cc }

// Config returns the normalized configuration used by this service.
func (s *Service) Config() *config.Config { return s.cfg }

// Logger returns the logger used by this service.
func (s *Service) Logger() *slog.Logger { return s.log }

// Option customizes v8.10 process-definition service construction.
type Option func(*Service)

// WithClientCamunda overrides the generated Camunda client, primarily for service tests.
func WithClientCamunda(c GenProcessDefinitionClientCamunda) Option {
	return func(s *Service) {
		if c != nil {
			s.cc = c
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

// New prepares a v8.10 process-definition service with the generated Camunda v2 client.
func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger, opts ...Option) (*Service, error) {
	deps, err := common.PrepareServiceDeps(cfg, httpClient, log)
	if err != nil {
		return nil, err
	}
	c, err := camundav810.NewClientWithResponses(
		deps.Config.APIs.Camunda.BaseURL,
		camundav810.WithHTTPClient(deps.HTTPClient),
	)
	if err != nil {
		return nil, err
	}
	s := &Service{cc: c, cfg: deps.Config, log: deps.Logger}
	for _, opt := range opts {
		opt(s)
	}
	logger, err := common.EnsureLoggerAndClients(s.log, s.cc)
	if err != nil {
		return nil, err
	}
	s.log = logger
	return s, nil
}

// SearchProcessDefinitions returns one sorted page of matching process definitions.
func (s *Service) SearchProcessDefinitions(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	cCfg := services.ApplyCallOptions(opts)
	page, err := s.SearchProcessDefinitionsPage(ctx, filter, d.ProcessDefinitionPageRequest{Size: size}, opts...)
	if err != nil {
		return nil, err
	}
	out := page.Items
	// Stats are attached to each definition, so final canonical sorting moves
	// the enriched definition as one value and preserves per-key association.
	d.SortProcessDefinitionsCanonical(out)

	common.VerboseLog(ctx, cCfg, s.log, "found process definitions", "count", len(out))
	return out, nil
}

// SearchProcessDefinitionsPage returns one backend process-definition search page without mutating page order.
func (s *Service) SearchProcessDefinitionsPage(ctx context.Context, filter d.ProcessDefinitionFilter, pageReq d.ProcessDefinitionPageRequest, opts ...services.CallOption) (d.ProcessDefinitionPage, error) {
	cCfg := services.ApplyCallOptions(opts)
	tenantID := common.EffectiveTenant(s.cfg)
	if cCfg.IgnoreTenant {
		tenantID = ""
	}
	body, err := searchProcessDefinitionsRequest(tenantID, filter, pageReq)
	if err != nil {
		return d.ProcessDefinitionPage{}, fmt.Errorf("building process definition search request: %w", err)
	}
	common.VerboseLog(ctx, cCfg, s.log, "searching process definitions", "baseURL", s.cfg.APIs.Camunda.BaseURL, "body", body)
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return d.ProcessDefinitionPage{}, err
	}
	resp, err := s.cc.SearchProcessDefinitionsWithBodyWithResponse(ctx, "application/json", bytes.NewReader(bodyJSON))
	if err != nil {
		return d.ProcessDefinitionPage{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.ProcessDefinitionPage{}, err
	}
	result, err := decodeSearchProcessDefinitionsResponse(resp.Body, payload)
	if err != nil {
		return d.ProcessDefinitionPage{}, err
	}
	items := toolx.MapSlice(result.Items, fromProcessDefinitionResult)
	if cCfg.WithStat {
		for i := range items {
			if items[i].Key == "" {
				continue
			}
			if err = s.retrieveProcessDefinitionStats(ctx, &items[i], opts...); err != nil {
				return d.ProcessDefinitionPage{}, err
			}
		}
	}
	return d.ProcessDefinitionPage{
		Items:         items,
		Request:       pageReq,
		OverflowState: pickProcessDefinitionOverflowState(result.Page, pageReq, len(items)),
		ReportedTotal: newProcessDefinitionReportedTotal(result.Page),
		EndCursor:     processDefinitionEndCursor(result.Page),
	}, nil
}

// SearchProcessDefinitionsLatest returns the latest visible process definition per BPMN process ID.
func (s *Service) SearchProcessDefinitionsLatest(ctx context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	filter.IsLatestVersion = true
	return s.SearchProcessDefinitions(ctx, filter, 1000, opts...)
}

// GetProcessDefinition retrieves a single process definition and optional statistics.
func (s *Service) GetProcessDefinition(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
	cCfg := services.ApplyCallOptions(opts)
	common.VerboseLog(ctx, cCfg, s.log, "retrieving process definition", "key", key)
	resp, err := s.cc.GetProcessDefinitionWithResponse(ctx, key)
	if err != nil {
		return d.ProcessDefinition{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.ProcessDefinition{}, err
	}
	pd := fromProcessDefinitionResult(*payload)
	if cCfg.WithStat {
		if err := s.retrieveProcessDefinitionStats(ctx, &pd, opts...); err != nil {
			return d.ProcessDefinition{}, err
		}
	}
	common.VerboseLog(ctx, cCfg, s.log, "process definition retrieved", "bpmnProcessId", pd.BpmnProcessId, "version", pd.ProcessVersion)
	return pd, nil
}

// GetProcessDefinitionXML retrieves the BPMN XML for one process definition.
func (s *Service) GetProcessDefinitionXML(ctx context.Context, key string, opts ...services.CallOption) (string, error) {
	cCfg := services.ApplyCallOptions(opts)
	common.VerboseLog(ctx, cCfg, s.log, "retrieving process definition xml", "key", key)
	resp, err := s.cc.GetProcessDefinitionXMLWithResponse(ctx, key)
	if err != nil {
		return "", err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.XML200)
	if err != nil {
		return "", err
	}
	if len(bytes.TrimSpace([]byte(*payload))) == 0 && len(bytes.TrimSpace(resp.Body)) > 0 {
		return string(resp.Body), nil
	}
	common.VerboseLog(ctx, cCfg, s.log, "process definition xml retrieved", "key", key)
	return *payload, nil
}

// retrieveProcessDefinitionStats populates exact process-instance statistics for one process definition.
func (s *Service) retrieveProcessDefinitionStats(ctx context.Context, pd *d.ProcessDefinition, opts ...services.CallOption) error {
	s.log.Debug(fmt.Sprintf("getting pd %s stats", pd.Key))
	stopActivity := logging.StartActivity(ctx, common.ProcessDefinitionStatsActivity(pd.BpmnProcessId, pd.Key))
	defer stopActivity()

	tenantID := common.EffectiveTenant(s.cfg)
	if services.ApplyCallOptions(opts).IgnoreTenant {
		tenantID = ""
	}
	active, err := s.countProcessInstancesForProcessDefinitionState(ctx, tenantID, *pd, "active", camundav810.ProcessInstanceStateEnumACTIVE)
	if err != nil {
		return err
	}
	logging.UpdateActivity(ctx, fmt.Sprintf("pd stats %s (%s): active %d", pd.BpmnProcessId, pd.Key, active))
	completed, err := s.countProcessInstancesForProcessDefinitionState(ctx, tenantID, *pd, "completed", camundav810.ProcessInstanceStateEnumCOMPLETED)
	if err != nil {
		return err
	}
	logging.UpdateActivity(ctx, fmt.Sprintf("pd stats %s (%s): completed %d", pd.BpmnProcessId, pd.Key, completed))
	canceled, err := s.countProcessInstancesForProcessDefinitionState(ctx, tenantID, *pd, "canceled", camundav810.ProcessInstanceStateEnum(d.StateTerminated))
	if err != nil {
		return err
	}
	logging.UpdateActivity(ctx, fmt.Sprintf("pd stats %s (%s): canceled %d", pd.BpmnProcessId, pd.Key, canceled))
	incidents, err := s.countProcessInstancesWithIncidentsForProcessDefinition(ctx, tenantID, *pd)
	if err != nil {
		return err
	}
	logging.UpdateActivity(ctx, fmt.Sprintf("pd stats %s (%s): incidents %d", pd.BpmnProcessId, pd.Key, incidents))
	ret := d.ProcessDefinitionStatistics{
		Active:    active,
		Completed: completed,
		Canceled:  canceled,
		Incidents: incidents,
	}
	ret.IncidentCountSupported = true
	pd.Statistics = &ret
	return nil
}

// countProcessInstancesWithIncidentsForProcessDefinition counts incident-bearing instances for one process definition.
func (s *Service) countProcessInstancesWithIncidentsForProcessDefinition(ctx context.Context, tenantID string, pd d.ProcessDefinition) (int64, error) {
	if pd.Key == "" {
		return 0, nil
	}
	req, err := searchProcessInstancesForDefinitionIncidentRequest(tenantID, pd.Key)
	if err != nil {
		return 0, err
	}
	return s.countProcessInstancesForDefinitionSearch(ctx, pd, "incidents", req)
}

// countProcessInstancesForProcessDefinitionState counts instances for one process-definition state bucket.
func (s *Service) countProcessInstancesForProcessDefinitionState(ctx context.Context, tenantID string, pd d.ProcessDefinition, label string, state camundav810.ProcessInstanceStateEnum) (int64, error) {
	if pd.Key == "" {
		return 0, nil
	}
	req, err := searchProcessInstancesForDefinitionStateRequest(tenantID, pd.Key, state)
	if err != nil {
		return 0, err
	}
	return s.countProcessInstancesForDefinitionSearch(ctx, pd, label, req)
}

// countProcessInstancesForDefinitionSearch returns an exact count, paging when Camunda reports a capped total.
func (s *Service) countProcessInstancesForDefinitionSearch(ctx context.Context, pd d.ProcessDefinition, label string, req camundav810.SearchProcessInstancesJSONRequestBody) (int64, error) {
	resp, err := s.searchProcessInstancesForDefinitionStatsPage(ctx, req)
	if err != nil {
		return 0, err
	}
	s.logProcessDefinitionStatsPage(ctx, pd, label, req, resp, 0)
	if !resp.JSON200.Page.HasMoreTotalItems {
		return resp.JSON200.Page.TotalItems, nil
	}

	total := int64(len(resp.JSON200.Items))
	cursor := processDefinitionStatsEndCursor(resp.JSON200.Page)
	offset := int32(total)
	for len(resp.JSON200.Items) > 0 {
		req.Page = processDefinitionStatsNextPage(cursor, offset)
		resp, err = s.searchProcessInstancesForDefinitionStatsPage(ctx, req)
		if err != nil {
			return 0, err
		}
		s.logProcessDefinitionStatsPage(ctx, pd, label, req, resp, total)
		total += int64(len(resp.JSON200.Items))
		if len(resp.JSON200.Items) == 0 {
			return total, nil
		}
		if next := processDefinitionStatsEndCursor(resp.JSON200.Page); next != "" {
			if next == cursor {
				return total, nil
			}
			cursor = next
			continue
		}
		if cursor != "" {
			return total, nil
		}
		offset += int32(len(resp.JSON200.Items))
	}
	return total, nil
}

// logProcessDefinitionStatsPage records debug details for one stats-count page.
func (s *Service) logProcessDefinitionStatsPage(ctx context.Context, pd d.ProcessDefinition, label string, req camundav810.SearchProcessInstancesJSONRequestBody, resp *camundav810.SearchProcessInstancesResponse, totalBefore int64) {
	if s.log == nil || resp == nil || resp.JSON200 == nil {
		return
	}
	mode, from, after, limit := describeProcessDefinitionStatsPageRequest(req.Page)
	items := len(resp.JSON200.Items)
	page := resp.JSON200.Page
	s.log.DebugContext(ctx, fmt.Sprintf(
		"pd stats page; key %s, bpmn %s, bucket %s, mode %s, from %d, after %q, limit %d, items %d, total before %d, total after %d, reported total %d, has more %t, end cursor %q",
		pd.Key,
		pd.BpmnProcessId,
		label,
		mode,
		from,
		after,
		limit,
		items,
		totalBefore,
		totalBefore+int64(items),
		page.TotalItems,
		page.HasMoreTotalItems,
		processDefinitionStatsEndCursor(page),
	))
}

// describeProcessDefinitionStatsPageRequest extracts stable debug fields from the stats page request.
func describeProcessDefinitionStatsPageRequest(page *camundav810.SearchQueryPageRequest) (string, int32, string, int32) {
	if page == nil {
		return "none", 0, "", 0
	}
	if cursor, err := page.AsCursorForwardPagination(); err == nil {
		return "cursor", 0, string(toolx.Deref(cursor.After, "")), toolx.Deref(cursor.Limit, 0)
	}
	if offset, err := page.AsOffsetPagination(); err == nil {
		return "offset", toolx.Deref(offset.From, 0), "", toolx.Deref(offset.Limit, 0)
	}
	return "unknown", 0, "", 0
}

// searchProcessInstancesForDefinitionStatsPage fetches and validates one stats-count process-instance page.
func (s *Service) searchProcessInstancesForDefinitionStatsPage(ctx context.Context, req camundav810.SearchProcessInstancesJSONRequestBody) (*camundav810.SearchProcessInstancesResponse, error) {
	bodyJSON, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := s.cc.SearchProcessInstancesWithBodyWithResponse(ctx, "application/json", bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		return nil, err
	}
	if resp.JSON200 == nil {
		return nil, d.ErrMalformedResponse
	}
	return resp, nil
}

// processDefinitionStatsNextPage builds the next stats-count page request from cursor or offset progress.
func processDefinitionStatsNextPage(after string, offset int32) *camundav810.SearchQueryPageRequest {
	limit := consts.MaxPISearchSize
	page := camundav810.SearchQueryPageRequest{}
	if after != "" {
		afterCursor := camundav810.EndCursor(after)
		_ = page.FromCursorForwardPagination(camundav810.CursorForwardPagination{
			After: &afterCursor,
			Limit: &limit,
		})
		return &page
	}
	_ = page.FromOffsetPagination(camundav810.OffsetPagination{
		From:  &offset,
		Limit: &limit,
	})
	return &page
}

// processDefinitionStatsEndCursor returns the page cursor used to continue stats-count paging.
func processDefinitionStatsEndCursor(page camundav810.SearchQueryPageResponse) string {
	if page.EndCursor == nil {
		return ""
	}
	return string(*page.EndCursor)
}

// processDefinitionEndCursor returns the page cursor used to continue process-definition paging.
func processDefinitionEndCursor(page camundav810.SearchQueryPageResponse) string {
	if page.EndCursor == nil {
		return ""
	}
	return string(*page.EndCursor)
}

// newProcessDefinitionReportedTotal converts v8.10 page metadata into domain total metadata.
func newProcessDefinitionReportedTotal(page camundav810.SearchQueryPageResponse) *d.ProcessDefinitionReportedTotal {
	if page.TotalItems == 0 {
		return nil
	}
	kind := d.ProcessDefinitionReportedTotalKindExact
	if page.HasMoreTotalItems {
		kind = d.ProcessDefinitionReportedTotalKindLowerBound
	}
	return &d.ProcessDefinitionReportedTotal{Count: page.TotalItems, Kind: kind}
}

// pickProcessDefinitionOverflowState classifies whether a process-definition page can continue.
func pickProcessDefinitionOverflowState(page camundav810.SearchQueryPageResponse, req d.ProcessDefinitionPageRequest, itemCount int) d.ProcessInstanceOverflowState {
	if itemCount == 0 {
		return d.ProcessInstanceOverflowStateNoMore
	}
	if req.After != "" {
		if page.HasMoreTotalItems {
			return d.ProcessInstanceOverflowStateHasMore
		}
		return d.ProcessInstanceOverflowStateNoMore
	}
	visibleCount := int64(req.From) + int64(itemCount)
	if page.TotalItems > visibleCount {
		return d.ProcessInstanceOverflowStateHasMore
	}
	if page.HasMoreTotalItems && req.Size > 0 && itemCount >= int(req.Size) {
		return d.ProcessInstanceOverflowStateHasMore
	}
	if page.TotalItems == 0 {
		return d.ProcessInstanceOverflowStateIndeterminate
	}
	return d.ProcessInstanceOverflowStateNoMore
}

// newProcessDefinitionSearchPageRequest builds the v8.10 page request, using cursor pagination when requested.
func newProcessDefinitionSearchPageRequest(pageReq d.ProcessDefinitionPageRequest, preferCursor bool) camundav810.SearchQueryPageRequest {
	page := camundav810.SearchQueryPageRequest{}
	if preferCursor || pageReq.After != "" {
		after := camundav810.EndCursor(pageReq.After)
		_ = page.FromCursorForwardPagination(camundav810.CursorForwardPagination{
			After: &after,
			Limit: &pageReq.Size,
		})
		return page
	}
	_ = page.FromOffsetPagination(camundav810.OffsetPagination{
		From:  &pageReq.From,
		Limit: &pageReq.Size,
	})
	return page
}

func searchProcessDefinitionsRequest(tenantID string, filter d.ProcessDefinitionFilter, pageReq d.ProcessDefinitionPageRequest) (processDefinitionSearchQuery, error) {
	processDefinitionIDFilter, err := newStringEqFilterPtr(filter.BpmnProcessId)
	if err != nil {
		return processDefinitionSearchQuery{}, err
	}
	bodyFilter := &processDefinitionFilter{
		ProcessDefinitionId: processDefinitionIDFilter,
		TenantId:            toolx.PtrIf(tenantID, ""),
		Version:             toolx.PtrIfNonZero(filter.ProcessVersion),
		VersionTag:          toolx.PtrIf(filter.ProcessVersionTag, ""),
		IsLatestVersion:     toolx.PtrIf(filter.IsLatestVersion, false),
	}
	page := newProcessDefinitionSearchPageRequest(pageReq, filter.IsLatestVersion)
	sort := []camundav810.ProcessDefinitionSearchQuerySortRequest{}
	if filter.IsLatestVersion {
		asc := camundav810.ASC
		sort = append(sort,
			camundav810.ProcessDefinitionSearchQuerySortRequest{
				Field: camundav810.ProcessDefinitionSearchQuerySortRequestFieldTenantId,
				Order: &asc,
			},
			camundav810.ProcessDefinitionSearchQuerySortRequest{
				Field: camundav810.ProcessDefinitionSearchQuerySortRequestFieldProcessDefinitionId,
				Order: &asc,
			},
		)
	} else {
		desc := camundav810.DESC
		asc := camundav810.ASC
		sort = append(sort,
			camundav810.ProcessDefinitionSearchQuerySortRequest{
				Field: camundav810.ProcessDefinitionSearchQuerySortRequestFieldTenantId,
				Order: &asc,
			},
			camundav810.ProcessDefinitionSearchQuerySortRequest{
				Field: camundav810.ProcessDefinitionSearchQuerySortRequestFieldProcessDefinitionId,
				Order: &asc,
			},
			camundav810.ProcessDefinitionSearchQuerySortRequest{
				Field: camundav810.ProcessDefinitionSearchQuerySortRequestFieldVersion,
				Order: &desc,
			},
			camundav810.ProcessDefinitionSearchQuerySortRequest{
				Field: camundav810.ProcessDefinitionSearchQuerySortRequestFieldProcessDefinitionKey,
				Order: &asc,
			},
		)
	}
	return processDefinitionSearchQuery{
		Filter: bodyFilter,
		Page:   &page,
		Sort:   &sort,
	}, nil
}

func searchProcessInstancesForDefinitionIncidentRequest(tenantID, processDefinitionKey string) (camundav810.SearchProcessInstancesJSONRequestBody, error) {
	processDefinitionKeyFilter, err := newProcessDefinitionKeyEqFilterPtr(processDefinitionKey)
	if err != nil {
		return camundav810.SearchProcessInstancesJSONRequestBody{}, err
	}
	tenantIDFilter, err := newStringEqFilterPtr(tenantID)
	if err != nil {
		return camundav810.SearchProcessInstancesJSONRequestBody{}, err
	}
	hasIncident := true
	var from int32
	var limit int32 = 1
	page := camundav810.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav810.OffsetPagination{
		From:  &from,
		Limit: &limit,
	})
	sort := processDefinitionStatsPISort()
	return camundav810.SearchProcessInstancesJSONRequestBody{
		Filter: &camundav810.ProcessInstanceFilter{
			ProcessDefinitionKey: processDefinitionKeyFilter,
			TenantId:             tenantIDFilter,
			HasIncident:          &hasIncident,
		},
		Page: &page,
		Sort: sort,
	}, nil
}

func searchProcessInstancesForDefinitionStateRequest(tenantID, processDefinitionKey string, state camundav810.ProcessInstanceStateEnum) (camundav810.SearchProcessInstancesJSONRequestBody, error) {
	processDefinitionKeyFilter, err := newProcessDefinitionKeyEqFilterPtr(processDefinitionKey)
	if err != nil {
		return camundav810.SearchProcessInstancesJSONRequestBody{}, err
	}
	stateFilter, err := newProcessInstanceStateEqFilterPtr(state)
	if err != nil {
		return camundav810.SearchProcessInstancesJSONRequestBody{}, err
	}
	tenantIDFilter, err := newStringEqFilterPtr(tenantID)
	if err != nil {
		return camundav810.SearchProcessInstancesJSONRequestBody{}, err
	}
	var from int32
	var limit int32 = 1
	page := camundav810.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav810.OffsetPagination{
		From:  &from,
		Limit: &limit,
	})
	sort := processDefinitionStatsPISort()
	return camundav810.SearchProcessInstancesJSONRequestBody{
		Filter: &camundav810.ProcessInstanceFilter{
			ProcessDefinitionKey: processDefinitionKeyFilter,
			State:                stateFilter,
			TenantId:             tenantIDFilter,
		},
		Page: &page,
		Sort: sort,
	}, nil
}

func processDefinitionStatsPISort() *[]camundav810.ProcessInstanceSearchQuerySortRequest {
	asc := camundav810.ASC
	return &[]camundav810.ProcessInstanceSearchQuerySortRequest{
		{
			Field: camundav810.ProcessInstanceSearchQuerySortRequestFieldProcessInstanceKey,
			Order: &asc,
		},
	}
}

type processDefinitionSearchQuery struct {
	Filter *processDefinitionFilter                               `json:"filter,omitempty"`
	Page   *camundav810.SearchQueryPageRequest                    `json:"page,omitempty"`
	Sort   *[]camundav810.ProcessDefinitionSearchQuerySortRequest `json:"sort,omitempty"`
}

type processDefinitionFilter struct {
	ProcessDefinitionId *camundav810.StringFilterProperty `json:"processDefinitionId,omitempty"`
	TenantId            *camundav810.TenantId             `json:"tenantId,omitempty"`
	Version             *int32                            `json:"version,omitempty"`
	VersionTag          *string                           `json:"versionTag,omitempty"`
	IsLatestVersion     *bool                             `json:"isLatestVersion,omitempty"`
}

type processDefinitionSearchQueryResult struct {
	Items []camundav810.ProcessDefinitionResult `json:"items"`
	Page  camundav810.SearchQueryPageResponse   `json:"page"`
}

func decodeSearchProcessDefinitionsResponse(body []byte, page *camundav810.ProcessDefinitionSearchQueryResult) (processDefinitionSearchQueryResult, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return processDefinitionSearchQueryResult{}, d.ErrMalformedResponse
	}
	var result processDefinitionSearchQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return processDefinitionSearchQueryResult{}, err
	}
	if result.Page.TotalItems == 0 && result.Page.EndCursor == nil && page != nil {
		result.Page = page.Page
	}
	return result, nil
}

// newStringEqFilterPtr builds a v8.10 string equality filter when a value is set.
func newStringEqFilterPtr(v string) (*camundav810.StringFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav810.StringFilterProperty
	if err := f.FromStringFilterProperty0(v); err != nil {
		return nil, err
	}
	return new(f), nil
}

// newProcessDefinitionKeyEqFilterPtr builds a v8.10 process-definition-key equality filter when a key is set.
func newProcessDefinitionKeyEqFilterPtr(v string) (*camundav810.ProcessDefinitionKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var filter camundav810.ProcessDefinitionKeyFilterProperty
	if err := filter.FromProcessDefinitionKeyFilterProperty0(camundav810.ProcessDefinitionKey(v)); err != nil {
		return nil, err
	}
	return &filter, nil
}

// newProcessInstanceStateEqFilterPtr builds a v8.10 process-instance-state equality filter.
func newProcessInstanceStateEqFilterPtr(v camundav810.ProcessInstanceStateEnum) (*camundav810.ProcessInstanceStateFilterProperty, error) {
	var filter camundav810.ProcessInstanceStateFilterProperty
	if err := filter.FromProcessInstanceStateFilterProperty0(v); err != nil {
		return nil, err
	}
	return &filter, nil
}
