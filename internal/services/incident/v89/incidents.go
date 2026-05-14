// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"fmt"
	"strings"
	"time"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/internal/services/incident/waiter"
	"github.com/grafvonb/c8volt/internal/services/incidentfilter"
	"github.com/grafvonb/c8volt/toolx"
)

// GetIncident loads a single incident by key for direct resolution planning and confirmation.
func (s *Service) GetIncident(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("getting incident %s", key))
	resp, err := s.cc.GetIncidentWithResponse(ctx, key)
	if err != nil {
		return d.ProcessInstanceIncidentDetail{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.ProcessInstanceIncidentDetail{}, err
	}
	return fromIncidentResult(*payload), nil
}

// ResolveIncident submits direct incident resolution without doing confirmation polling.
func (s *Service) ResolveIncident(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("resolving incident %s", key))
	resp, err := s.cc.ResolveIncidentWithResponse(ctx, key, camundav89.ResolveIncidentJSONRequestBody{})
	if err != nil {
		return d.IncidentResolutionResponse{Key: key}, err
	}
	result := d.IncidentResolutionResponse{
		Key:        key,
		Ok:         resp.StatusCode() >= 200 && resp.StatusCode() < 300,
		StatusCode: resp.StatusCode(),
		Status:     resp.Status(),
	}
	if err := httpc.HttpStatusErr(resp.HTTPResponse, resp.Body); err != nil {
		result.Ok = false
		return result, err
	}
	return result, nil
}

// SearchProcessInstanceIncidents uses the scoped process-instance incident endpoint for active incident enrichment.
func (s *Service) SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	callCfg := services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("searching pi %s incidents", key))
	tenantFilter, err := newStringEqFilterPtr(s.cfg.App.Tenant)
	if err != nil {
		return nil, fmt.Errorf("building tenant incident filter: %w", err)
	}
	stateFilter, err := newIncidentSearchStateFilter(callCfg.IncidentState)
	if err != nil {
		return nil, fmt.Errorf("building incident state filter: %w", err)
	}
	errorTypeFilter, err := newIncidentSearchErrorTypeFilter(callCfg.IncidentErrorType)
	if err != nil {
		return nil, fmt.Errorf("building incident error type filter: %w", err)
	}
	filter := &camundav89.IncidentFilter{
		State:     stateFilter,
		TenantId:  tenantFilter,
		ErrorType: errorTypeFilter,
	}
	return s.searchProcessInstanceIncidentsPages(ctx, key, filter, callCfg.IncidentErrorMessage)
}

// SearchIncidents returns up to size top-level incidents after server-safe and local filters.
func (s *Service) SearchIncidents(ctx context.Context, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	page, err := s.SearchIncidentsPage(ctx, filter, d.IncidentPageRequest{Size: size}, opts...)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

// SearchIncidentsPage uses safe server-side filters on v8.9 and applies filters
// whose semantics are not represented by the API locally.
func (s *Service) SearchIncidentsPage(ctx context.Context, filter d.IncidentFilter, pageReq d.IncidentPageRequest, opts ...services.CallOption) (d.IncidentPage, error) {
	_ = services.ApplyCallOptions(opts)
	bodyFilter, err := s.newIncidentFilter(filter)
	if err != nil {
		return d.IncidentPage{}, err
	}
	page := newIncidentSearchQueryPageRequest(pageReq)
	body := camundav89.SearchIncidentsJSONRequestBody{
		Filter: bodyFilter,
		Page:   &page,
	}
	resp, err := s.cc.SearchIncidentsWithResponse(ctx, body)
	if err != nil {
		return d.IncidentPage{}, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return d.IncidentPage{}, err
	}
	items := filterIncidentDetailsLocally(filter, toolx.MapSlice(payload.Items, fromIncidentResult))
	return d.IncidentPage{
		Items:         items,
		Request:       pageReq,
		OverflowState: incidentSearchOverflowState(payload.Page, pageReq, len(payload.Items)),
		ReportedTotal: incidentReportedTotal(payload.Page, len(payload.Items), incidentLocalFilteringRequired(filter)),
	}, nil
}

func (s *Service) newIncidentFilter(filter d.IncidentFilter) (*camundav89.IncidentFilter, error) {
	tenantFilter, err := newStringEqFilterPtr(s.cfg.App.Tenant)
	if err != nil {
		return nil, fmt.Errorf("building tenant incident filter: %w", err)
	}
	stateFilter, err := newIncidentSearchStateFilter(filter.State)
	if err != nil {
		return nil, fmt.Errorf("building incident state filter: %w", err)
	}
	errorTypeFilter, err := newIncidentSearchErrorTypeFilter(filter.ErrorType)
	if err != nil {
		return nil, fmt.Errorf("building incident error type filter: %w", err)
	}
	processInstanceKeyFilter, err := newProcessInstanceKeyEqFilterPtr(filter.ProcessInstanceKey)
	if err != nil {
		return nil, fmt.Errorf("building incident process-instance-key filter: %w", err)
	}
	processDefinitionKeyFilter, err := newProcessDefinitionKeyEqFilterPtr(filter.ProcessDefinitionKey)
	if err != nil {
		return nil, fmt.Errorf("building incident process-definition-key filter: %w", err)
	}
	processDefinitionIDFilter, err := newStringEqFilterPtr(filter.ProcessDefinitionId)
	if err != nil {
		return nil, fmt.Errorf("building incident process-definition-id filter: %w", err)
	}
	flowNodeIDFilter, err := newStringEqFilterPtr(filter.FlowNodeId)
	if err != nil {
		return nil, fmt.Errorf("building incident flow-node-id filter: %w", err)
	}
	flowNodeInstanceKeyFilter, err := newElementInstanceKeyEqFilterPtr(filter.FlowNodeInstanceKey)
	if err != nil {
		return nil, fmt.Errorf("building incident flow-node-instance-key filter: %w", err)
	}
	creationTimeAfter, err := parseIncidentTimeBound(filter.CreationTimeAfter)
	if err != nil {
		return nil, fmt.Errorf("building incident creation-time-after filter: %w", err)
	}
	creationTimeBefore, err := parseIncidentTimeBound(filter.CreationTimeBefore)
	if err != nil {
		return nil, fmt.Errorf("building incident creation-time-before filter: %w", err)
	}
	creationTimeFilter, err := newDateTimeRangeFilterPtr(creationTimeAfter, creationTimeBefore, nil)
	if err != nil {
		return nil, fmt.Errorf("building incident creation-time filter: %w", err)
	}
	bodyFilter := &camundav89.IncidentFilter{
		TenantId:             tenantFilter,
		State:                stateFilter,
		ErrorType:            errorTypeFilter,
		ProcessInstanceKey:   processInstanceKeyFilter,
		ProcessDefinitionKey: processDefinitionKeyFilter,
		ProcessDefinitionId:  processDefinitionIDFilter,
		ElementId:            flowNodeIDFilter,
		ElementInstanceKey:   flowNodeInstanceKeyFilter,
		CreationTime:         creationTimeFilter,
	}
	if bodyFilter.TenantId == nil &&
		bodyFilter.State == nil &&
		bodyFilter.ErrorType == nil &&
		bodyFilter.ProcessInstanceKey == nil &&
		bodyFilter.ProcessDefinitionKey == nil &&
		bodyFilter.ProcessDefinitionId == nil &&
		bodyFilter.ElementId == nil &&
		bodyFilter.ElementInstanceKey == nil &&
		bodyFilter.CreationTime == nil {
		return nil, nil
	}
	return bodyFilter, nil
}

func newIncidentSearchStateFilter(state string) (*camundav89.IncidentStateFilterProperty, error) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "", "active":
		return newIncidentStateEqFilterPtr(camundav89.IncidentStateEnumACTIVE)
	case "pending":
		return newIncidentStateEqFilterPtr(camundav89.IncidentStateEnumPENDING)
	case "resolved":
		return newIncidentStateEqFilterPtr(camundav89.IncidentStateEnumRESOLVED)
	case "migrated":
		return newIncidentStateEqFilterPtr(camundav89.IncidentStateEnumMIGRATED)
	case "unknown":
		return newIncidentStateEqFilterPtr(camundav89.IncidentStateEnumUNKNOWN)
	case "all":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported incident state %q", state)
	}
}

func filterIncidentDetailsLocally(filter d.IncidentFilter, items []d.ProcessInstanceIncidentDetail) []d.ProcessInstanceIncidentDetail {
	out := make([]d.ProcessInstanceIncidentDetail, 0, len(items))
	for _, item := range items {
		if filter.RootProcessInstanceKey != "" && item.RootProcessInstanceKey != filter.RootProcessInstanceKey {
			continue
		}
		if !incidentfilter.ErrorMessageContains(filter.ErrorMessage, item.ErrorMessage) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func incidentLocalFilteringRequired(filter d.IncidentFilter) bool {
	return filter.RootProcessInstanceKey != "" || filter.ErrorMessage != ""
}

func parseIncidentTimeBound(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return &t, nil
	}
	if t, err := time.Parse(time.DateOnly, raw); err == nil {
		return &t, nil
	}
	return nil, fmt.Errorf("parse %q as incident timestamp", raw)
}

func newIncidentSearchErrorTypeFilter(errorType string) (*camundav89.IncidentErrorTypeFilterProperty, error) {
	normalized, ok := incidentfilter.NormalizeErrorType(errorType)
	if !ok {
		return nil, fmt.Errorf("unsupported incident error type %q", errorType)
	}
	if normalized == "" {
		return nil, nil
	}
	return newIncidentErrorTypeEqFilterPtr(camundav89.IncidentErrorTypeEnum(normalized))
}

func (s *Service) searchProcessInstanceIncidentsPages(ctx context.Context, key string, filter *camundav89.IncidentFilter, errorMessage string) ([]d.ProcessInstanceIncidentDetail, error) {
	const pageSize int32 = 1000
	var out []d.ProcessInstanceIncidentDetail
	for from := int32(0); ; {
		page := newSearchQueryPageRequest(d.ProcessInstancePageRequest{From: from, Size: pageSize})
		body := camundav89.SearchProcessInstanceIncidentsJSONRequestBody{
			Page:   &page,
			Filter: filter,
		}
		resp, err := s.cc.SearchProcessInstanceIncidentsWithResponse(ctx, key, body)
		if err != nil {
			return nil, err
		}
		payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
		if err != nil {
			return nil, err
		}
		out = append(out, filterIncidentDetailsByMessage(errorMessage, toolx.MapSlice(payload.Items, fromIncidentResult))...)
		if !incidentSearchHasMore(payload.Page, from, len(payload.Items), pageSize) {
			return out, nil
		}
		from += int32(len(payload.Items))
	}
}

func filterIncidentDetailsByMessage(errorMessage string, items []d.ProcessInstanceIncidentDetail) []d.ProcessInstanceIncidentDetail {
	out := make([]d.ProcessInstanceIncidentDetail, 0, len(items))
	for _, item := range items {
		if !incidentfilter.ErrorMessageContains(errorMessage, item.ErrorMessage) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func incidentSearchHasMore(page camundav89.SearchQueryPageResponse, from int32, itemCount int, pageSize int32) bool {
	if itemCount == 0 {
		return false
	}
	visibleCount := int64(from) + int64(itemCount)
	if page.TotalItems > visibleCount {
		return true
	}
	return page.HasMoreTotalItems && itemCount >= int(pageSize)
}

func incidentSearchOverflowState(page camundav89.SearchQueryPageResponse, req d.IncidentPageRequest, itemCount int) d.ProcessInstanceOverflowState {
	if itemCount == 0 {
		return d.ProcessInstanceOverflowStateNoMore
	}
	visibleCount := int64(req.From) + int64(itemCount)
	if page.TotalItems > visibleCount {
		return d.ProcessInstanceOverflowStateHasMore
	}
	if page.HasMoreTotalItems {
		return d.ProcessInstanceOverflowStateIndeterminate
	}
	return d.ProcessInstanceOverflowStateNoMore
}

func incidentReportedTotal(page camundav89.SearchQueryPageResponse, itemCount int, localFiltering bool) *d.IncidentReportedTotal {
	if localFiltering || (page.TotalItems == 0 && itemCount > 0) {
		return nil
	}
	kind := d.IncidentReportedTotalKindExact
	if page.HasMoreTotalItems {
		kind = d.IncidentReportedTotalKindLowerBound
	}
	return &d.IncidentReportedTotal{
		Count: page.TotalItems,
		Kind:  kind,
	}
}

// WaitForIncidentResolved polls direct incident lookup until the selected incident is no longer active.
func (s *Service) WaitForIncidentResolved(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	return waiter.WaitForIncidentResolved(ctx, s, s.cfg, s.log, key, opts...)
}

// WaitForProcessInstanceIncidentsResolved polls process-instance incident lookup until the initial incident set is gone.
func (s *Service) WaitForProcessInstanceIncidentsResolved(ctx context.Context, processInstanceKey string, incidentKeys []string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	return waiter.WaitForProcessInstanceIncidentsResolved(ctx, s, s.cfg, s.log, processInstanceKey, incidentKeys, opts...)
}

// newSearchQueryPageRequest builds the v8.9 page request for incident lookups.
func newSearchQueryPageRequest(pageReq d.ProcessInstancePageRequest) camundav89.SearchQueryPageRequest {
	page := camundav89.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav89.OffsetPagination{
		From:  &pageReq.From,
		Limit: &pageReq.Size,
	})
	return page
}

func newIncidentSearchQueryPageRequest(pageReq d.IncidentPageRequest) camundav89.SearchQueryPageRequest {
	page := camundav89.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav89.OffsetPagination{
		From:  &pageReq.From,
		Limit: &pageReq.Size,
	})
	return page
}
