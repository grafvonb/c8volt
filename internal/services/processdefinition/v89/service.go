// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
)

type Service struct {
	cc  GenProcessDefinitionClientCamunda
	cfg *config.Config
	log *slog.Logger
}

func (s *Service) ClientCamunda() GenProcessDefinitionClientCamunda { return s.cc }
func (s *Service) Config() *config.Config                           { return s.cfg }
func (s *Service) Logger() *slog.Logger                             { return s.log }

type Option func(*Service)

func WithClientCamunda(c GenProcessDefinitionClientCamunda) Option {
	return func(s *Service) {
		if c != nil {
			s.cc = c
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

func (s *Service) SearchProcessDefinitions(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	cCfg := services.ApplyCallOptions(opts)
	body, err := searchProcessDefinitionsRequest(common.EffectiveTenant(s.cfg), filter, size)
	if err != nil {
		return nil, fmt.Errorf("building process definition search request: %w", err)
	}
	common.VerboseLog(ctx, cCfg, s.log, "searching process definitions", "baseURL", s.cfg.APIs.Camunda.BaseURL, "body", body)
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := s.cc.SearchProcessDefinitionsWithBodyWithResponse(ctx, "application/json", bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return nil, err
	}
	result, err := decodeSearchProcessDefinitionsResponse(resp.Body, payload)
	if err != nil {
		return nil, err
	}
	out := toolx.MapSlice(result.Items, fromProcessDefinitionResult)
	d.SortByBpmnProcessIdAscThenByVersionDesc(out)

	if cCfg.WithStat {
		for i := range out {
			if out[i].Key == "" {
				continue
			}
			if err = s.retrieveProcessDefinitionStats(ctx, &out[i]); err != nil {
				return nil, err
			}
		}
	}
	common.VerboseLog(ctx, cCfg, s.log, "found process definitions", "count", len(out))
	return out, nil
}

func (s *Service) SearchProcessDefinitionsLatest(ctx context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	filter.IsLatestVersion = true
	return s.SearchProcessDefinitions(ctx, filter, 1000, opts...)
}

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
		if err := s.retrieveProcessDefinitionStats(ctx, &pd); err != nil {
			return d.ProcessDefinition{}, err
		}
	}
	common.VerboseLog(ctx, cCfg, s.log, "process definition retrieved", "bpmnProcessId", pd.BpmnProcessId, "version", pd.ProcessVersion)
	return pd, nil
}

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
func (s *Service) retrieveProcessDefinitionStats(ctx context.Context, pd *d.ProcessDefinition) error {
	s.log.Debug(fmt.Sprintf("retrieving process definition stats for key %q", pd.Key))
	stopActivity := logging.StartActivity(ctx, common.ProcessDefinitionStatsActivity(pd.BpmnProcessId, pd.Key))
	defer stopActivity()

	active, err := s.countProcessInstancesForProcessDefinitionState(ctx, *pd, "active", camundav89.ProcessInstanceStateEnumACTIVE)
	if err != nil {
		return err
	}
	completed, err := s.countProcessInstancesForProcessDefinitionState(ctx, *pd, "completed", camundav89.ProcessInstanceStateEnumCOMPLETED)
	if err != nil {
		return err
	}
	canceled, err := s.countProcessInstancesForProcessDefinitionState(ctx, *pd, "canceled", camundav89.ProcessInstanceStateEnum(d.StateTerminated))
	if err != nil {
		return err
	}
	incidents, err := s.countProcessInstancesWithIncidentsForProcessDefinition(ctx, *pd)
	if err != nil {
		return err
	}
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
func (s *Service) countProcessInstancesWithIncidentsForProcessDefinition(ctx context.Context, pd d.ProcessDefinition) (int64, error) {
	if pd.Key == "" {
		return 0, nil
	}
	req, err := searchProcessInstancesForDefinitionIncidentRequest(common.EffectiveTenant(s.cfg), pd.Key)
	if err != nil {
		return 0, err
	}
	return s.countProcessInstancesForDefinitionSearch(ctx, pd, "incidents", req)
}

// countProcessInstancesForProcessDefinitionState counts instances for one process-definition state bucket.
func (s *Service) countProcessInstancesForProcessDefinitionState(ctx context.Context, pd d.ProcessDefinition, label string, state camundav89.ProcessInstanceStateEnum) (int64, error) {
	if pd.Key == "" {
		return 0, nil
	}
	req, err := searchProcessInstancesForDefinitionStateRequest(common.EffectiveTenant(s.cfg), pd.Key, state)
	if err != nil {
		return 0, err
	}
	return s.countProcessInstancesForDefinitionSearch(ctx, pd, label, req)
}

// countProcessInstancesForDefinitionSearch returns an exact count, paging when Camunda reports a capped total.
func (s *Service) countProcessInstancesForDefinitionSearch(ctx context.Context, pd d.ProcessDefinition, label string, req camundav89.SearchProcessInstancesJSONRequestBody) (int64, error) {
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
func (s *Service) logProcessDefinitionStatsPage(ctx context.Context, pd d.ProcessDefinition, label string, req camundav89.SearchProcessInstancesJSONRequestBody, resp *camundav89.SearchProcessInstancesResponse, totalBefore int64) {
	if s.log == nil || resp == nil || resp.JSON200 == nil {
		return
	}
	mode, from, after, limit := describeProcessDefinitionStatsPageRequest(req.Page)
	items := len(resp.JSON200.Items)
	page := resp.JSON200.Page
	s.log.DebugContext(ctx, fmt.Sprintf(
		"process-definition stats page: process definition key=%s, bpmn process id=%s, bucket=%s, mode=%s, from=%d, after=%q, limit=%d, items=%d, total before=%d, total after=%d, reported total=%d, has more total items=%t, end cursor=%q",
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
func describeProcessDefinitionStatsPageRequest(page *camundav89.SearchQueryPageRequest) (string, int32, string, int32) {
	if page == nil {
		return "none", 0, "", 0
	}
	if cursor, err := page.AsCursorForwardPagination(); err == nil {
		return "cursor", 0, string(cursor.After), toolx.Deref(cursor.Limit, 0)
	}
	if offset, err := page.AsOffsetPagination(); err == nil {
		return "offset", toolx.Deref(offset.From, 0), "", toolx.Deref(offset.Limit, 0)
	}
	return "unknown", 0, "", 0
}

// searchProcessInstancesForDefinitionStatsPage fetches and validates one stats-count process-instance page.
func (s *Service) searchProcessInstancesForDefinitionStatsPage(ctx context.Context, req camundav89.SearchProcessInstancesJSONRequestBody) (*camundav89.SearchProcessInstancesResponse, error) {
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
func processDefinitionStatsNextPage(after string, offset int32) *camundav89.SearchQueryPageRequest {
	limit := consts.MaxPISearchSize
	page := camundav89.SearchQueryPageRequest{}
	if after != "" {
		_ = page.FromCursorForwardPagination(camundav89.CursorForwardPagination{
			After: camundav89.EndCursor(after),
			Limit: &limit,
		})
		return &page
	}
	_ = page.FromOffsetPagination(camundav89.OffsetPagination{
		From:  &offset,
		Limit: &limit,
	})
	return &page
}

// processDefinitionStatsEndCursor returns the page cursor used to continue stats-count paging.
func processDefinitionStatsEndCursor(page camundav89.SearchQueryPageResponse) string {
	if page.EndCursor == nil {
		return ""
	}
	return string(*page.EndCursor)
}

func searchProcessDefinitionsRequest(tenantID string, filter d.ProcessDefinitionFilter, size int32) (processDefinitionSearchQuery, error) {
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
	page := camundav89.SearchQueryPageRequest{}
	sort := []camundav89.ProcessDefinitionSearchQuerySortRequest{}
	if filter.IsLatestVersion {
		after := camundav89.EndCursor("")
		_ = page.FromCursorForwardPagination(camundav89.CursorForwardPagination{
			After: after,
			Limit: &size,
		})
		asc := camundav89.ASC
		sort = append(sort,
			camundav89.ProcessDefinitionSearchQuerySortRequest{
				Field: camundav89.ProcessDefinitionSearchQuerySortRequestFieldProcessDefinitionId,
				Order: &asc,
			},
			camundav89.ProcessDefinitionSearchQuerySortRequest{
				Field: camundav89.ProcessDefinitionSearchQuerySortRequestFieldTenantId,
				Order: &asc,
			},
		)
	} else {
		_ = page.FromOffsetPagination(camundav89.OffsetPagination{
			From:  new(int32),
			Limit: &size,
		})
		desc := camundav89.DESC
		asc := camundav89.ASC
		sort = append(sort,
			camundav89.ProcessDefinitionSearchQuerySortRequest{
				Field: camundav89.ProcessDefinitionSearchQuerySortRequestFieldVersion,
				Order: &desc,
			},
			camundav89.ProcessDefinitionSearchQuerySortRequest{
				Field: camundav89.ProcessDefinitionSearchQuerySortRequestFieldName,
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

func searchProcessInstancesForDefinitionIncidentRequest(tenantID, processDefinitionKey string) (camundav89.SearchProcessInstancesJSONRequestBody, error) {
	processDefinitionKeyFilter, err := newProcessDefinitionKeyEqFilterPtr(processDefinitionKey)
	if err != nil {
		return camundav89.SearchProcessInstancesJSONRequestBody{}, err
	}
	tenantIDFilter, err := newStringEqFilterPtr(tenantID)
	if err != nil {
		return camundav89.SearchProcessInstancesJSONRequestBody{}, err
	}
	hasIncident := true
	var from int32
	var limit int32 = 1
	page := camundav89.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav89.OffsetPagination{
		From:  &from,
		Limit: &limit,
	})
	sort := processDefinitionStatsPISort()
	return camundav89.SearchProcessInstancesJSONRequestBody{
		Filter: &camundav89.ProcessInstanceFilter{
			ProcessDefinitionKey: processDefinitionKeyFilter,
			TenantId:             tenantIDFilter,
			HasIncident:          &hasIncident,
		},
		Page: &page,
		Sort: sort,
	}, nil
}

func searchProcessInstancesForDefinitionStateRequest(tenantID, processDefinitionKey string, state camundav89.ProcessInstanceStateEnum) (camundav89.SearchProcessInstancesJSONRequestBody, error) {
	processDefinitionKeyFilter, err := newProcessDefinitionKeyEqFilterPtr(processDefinitionKey)
	if err != nil {
		return camundav89.SearchProcessInstancesJSONRequestBody{}, err
	}
	stateFilter, err := newProcessInstanceStateEqFilterPtr(state)
	if err != nil {
		return camundav89.SearchProcessInstancesJSONRequestBody{}, err
	}
	tenantIDFilter, err := newStringEqFilterPtr(tenantID)
	if err != nil {
		return camundav89.SearchProcessInstancesJSONRequestBody{}, err
	}
	var from int32
	var limit int32 = 1
	page := camundav89.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav89.OffsetPagination{
		From:  &from,
		Limit: &limit,
	})
	sort := processDefinitionStatsPISort()
	return camundav89.SearchProcessInstancesJSONRequestBody{
		Filter: &camundav89.ProcessInstanceFilter{
			ProcessDefinitionKey: processDefinitionKeyFilter,
			State:                stateFilter,
			TenantId:             tenantIDFilter,
		},
		Page: &page,
		Sort: sort,
	}, nil
}

func processDefinitionStatsPISort() *[]camundav89.ProcessInstanceSearchQuerySortRequest {
	asc := camundav89.ASC
	return &[]camundav89.ProcessInstanceSearchQuerySortRequest{
		{
			Field: camundav89.ProcessInstanceSearchQuerySortRequestFieldProcessInstanceKey,
			Order: &asc,
		},
	}
}

type processDefinitionSearchQuery struct {
	Filter *processDefinitionFilter                              `json:"filter,omitempty"`
	Page   *camundav89.SearchQueryPageRequest                    `json:"page,omitempty"`
	Sort   *[]camundav89.ProcessDefinitionSearchQuerySortRequest `json:"sort,omitempty"`
}

type processDefinitionFilter struct {
	ProcessDefinitionId *camundav89.StringFilterProperty `json:"processDefinitionId,omitempty"`
	TenantId            *camundav89.TenantId             `json:"tenantId,omitempty"`
	Version             *int32                           `json:"version,omitempty"`
	VersionTag          *string                          `json:"versionTag,omitempty"`
	IsLatestVersion     *bool                            `json:"isLatestVersion,omitempty"`
}

type processDefinitionSearchQueryResult struct {
	Items []camundav89.ProcessDefinitionResult `json:"items"`
	Page  camundav89.SearchQueryPageResponse   `json:"page"`
}

func decodeSearchProcessDefinitionsResponse(body []byte, page *camundav89.ProcessDefinitionSearchQueryResult) (processDefinitionSearchQueryResult, error) {
	if len(bytes.TrimSpace(body)) == 0 {
		return processDefinitionSearchQueryResult{}, d.ErrMalformedResponse
	}
	var result processDefinitionSearchQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return processDefinitionSearchQueryResult{}, err
	}
	result.Page = page.Page
	return result, nil
}

// newStringEqFilterPtr builds a v8.9 string equality filter when a value is set.
func newStringEqFilterPtr(v string) (*camundav89.StringFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav89.StringFilterProperty
	if err := f.FromStringFilterProperty0(v); err != nil {
		return nil, err
	}
	return new(f), nil
}

// newProcessDefinitionKeyEqFilterPtr builds a v8.9 process-definition-key equality filter when a key is set.
func newProcessDefinitionKeyEqFilterPtr(v string) (*camundav89.ProcessDefinitionKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var filter camundav89.ProcessDefinitionKeyFilterProperty
	if err := filter.FromProcessDefinitionKeyFilterProperty0(camundav89.ProcessDefinitionKey(v)); err != nil {
		return nil, err
	}
	return &filter, nil
}

// newProcessInstanceStateEqFilterPtr builds a v8.9 process-instance-state equality filter.
func newProcessInstanceStateEqFilterPtr(v camundav89.ProcessInstanceStateEnum) (*camundav89.ProcessInstanceStateFilterProperty, error) {
	var filter camundav89.ProcessInstanceStateFilterProperty
	if err := filter.FromProcessInstanceStateFilterProperty0(v); err != nil {
		return nil, err
	}
	return &filter, nil
}
