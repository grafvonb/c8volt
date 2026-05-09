// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"fmt"
	"strings"
	"time"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
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
	s.log.Debug(fmt.Sprintf("getting incident with key %s using generated camunda client", key))
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
	s.log.Debug(fmt.Sprintf("resolving incident with key %s using generated camunda client", key))
	resp, err := s.cc.ResolveIncidentWithResponse(ctx, key, camundav88.ResolveIncidentJSONRequestBody{})
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

// SearchProcessInstanceIncidents uses the scoped process-instance incident
// endpoint for v8.8 but does not send an incident filter. Some v8.8 clusters
// expose the endpoint but reject the filter object at runtime, so c8volt keeps
// the request tenant-safe through the path and applies direct/state/tenant
// filtering locally.
func (s *Service) SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	callCfg := services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("searching incidents for process instance with key %s using generated camunda client", key))
	return s.searchProcessInstanceIncidentsPages(ctx, key, common.EffectiveTenant(s.cfg), callCfg)
}

// SearchIncidents returns up to size top-level incidents after version-compatible filtering.
func (s *Service) SearchIncidents(ctx context.Context, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	if incidentSearchNeedsPagedLocalFiltering(filter) {
		return s.searchIncidentPagesUntilLimit(ctx, filter, size, opts...)
	}
	page, err := s.SearchIncidentsPage(ctx, filter, d.IncidentPageRequest{Size: size}, opts...)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

func (s *Service) searchIncidentPagesUntilLimit(ctx context.Context, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	if size <= 0 {
		return nil, nil
	}
	req := d.IncidentPageRequest{Size: size}
	out := make([]d.ProcessInstanceIncidentDetail, 0, size)
	for {
		page, err := s.SearchIncidentsPage(ctx, filter, req, opts...)
		if err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			if int32(len(out)) >= size {
				return out, nil
			}
			out = append(out, item)
		}
		if page.OverflowState == d.ProcessInstanceOverflowStateNoMore {
			return out, nil
		}
		req = nextIncidentSearchPageRequest(req, page)
	}
}

func incidentSearchNeedsPagedLocalFiltering(filter d.IncidentFilter) bool {
	return filter.ErrorMessage != "" ||
		filter.CreationTimeAfter != "" ||
		filter.CreationTimeBefore != ""
}

func nextIncidentSearchPageRequest(current d.IncidentPageRequest, page d.IncidentPage) d.IncidentPageRequest {
	if page.EndCursor != "" {
		return d.IncidentPageRequest{Size: current.Size, After: page.EndCursor}
	}
	return d.IncidentPageRequest{From: current.From + current.Size, Size: current.Size}
}

// SearchIncidentsPage uses the top-level v8.8 incident endpoint with a tenant
// filter and applies other incident filters locally to avoid compatibility
// issues with richer filter request shapes.
func (s *Service) SearchIncidentsPage(ctx context.Context, filter d.IncidentFilter, pageReq d.IncidentPageRequest, opts ...services.CallOption) (d.IncidentPage, error) {
	_ = services.ApplyCallOptions(opts)
	tenantFilter, err := common.NewStringEqFilterPtr(common.EffectiveTenant(s.cfg))
	if err != nil {
		return d.IncidentPage{}, fmt.Errorf("building tenant incident filter: %w", err)
	}
	bodyFilter := &camundav88.IncidentFilter{TenantId: tenantFilter}
	if bodyFilter.TenantId == nil {
		bodyFilter = nil
	}
	page := newIncidentSearchQueryPageRequest(pageReq)
	body := camundav88.SearchIncidentsJSONRequestBody{
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
	items := filterIncidentSearchResults(filter, common.EffectiveTenant(s.cfg), payload.Items)
	return d.IncidentPage{
		Items:         items,
		Request:       pageReq,
		OverflowState: incidentSearchOverflowState(payload.Page, pageReq, len(payload.Items)),
		ReportedTotal: incidentReportedTotal(payload.Page, len(payload.Items), incidentLocalFilteringRequired(filter)),
	}, nil
}

func filterIncidentResults(key string, tenant string, state string, errorType string, errorMessage string, items []camundav88.IncidentResult) []d.ProcessInstanceIncidentDetail {
	out := make([]d.ProcessInstanceIncidentDetail, 0, len(items))
	for _, item := range items {
		if item.ProcessInstanceKey != key {
			continue
		}
		if tenant != "" && item.TenantId != tenant {
			continue
		}
		if !incidentStateMatches(state, item.State) {
			continue
		}
		if !incidentfilter.ErrorTypeMatches(errorType, string(item.ErrorType)) {
			continue
		}
		if !incidentfilter.ErrorMessageContains(errorMessage, item.ErrorMessage) {
			continue
		}
		out = append(out, fromIncidentResult(item))
	}
	return out
}

func filterIncidentSearchResults(filter d.IncidentFilter, tenant string, items []camundav88.IncidentResult) []d.ProcessInstanceIncidentDetail {
	out := make([]d.ProcessInstanceIncidentDetail, 0, len(items))
	for _, item := range items {
		if tenant != "" && item.TenantId != tenant {
			continue
		}
		if !incidentStateMatches(filter.State, item.State) {
			continue
		}
		if !incidentfilter.ErrorTypeMatches(filter.ErrorType, string(item.ErrorType)) {
			continue
		}
		if !incidentfilter.ErrorMessageContains(filter.ErrorMessage, item.ErrorMessage) {
			continue
		}
		if filter.ProcessInstanceKey != "" && item.ProcessInstanceKey != filter.ProcessInstanceKey {
			continue
		}
		if filter.RootProcessInstanceKey != "" && toolx.Deref(item.RootProcessInstanceKey, "") != filter.RootProcessInstanceKey {
			continue
		}
		if filter.ProcessDefinitionKey != "" && item.ProcessDefinitionKey != filter.ProcessDefinitionKey {
			continue
		}
		if filter.ProcessDefinitionId != "" && item.ProcessDefinitionId != filter.ProcessDefinitionId {
			continue
		}
		if filter.FlowNodeId != "" && item.ElementId != filter.FlowNodeId {
			continue
		}
		if filter.FlowNodeInstanceKey != "" && item.ElementInstanceKey != filter.FlowNodeInstanceKey {
			continue
		}
		if !incidentCreationTimeMatches(incidentCreationTime(item.CreationTime), filter.CreationTimeAfter, filter.CreationTimeBefore) {
			continue
		}
		out = append(out, fromIncidentResult(item))
	}
	return out
}

func incidentStateMatches(want string, got camundav88.IncidentStateEnum) bool {
	switch strings.ToLower(strings.TrimSpace(want)) {
	case "", "active":
		return got == camundav88.IncidentStateEnumACTIVE
	case "all":
		return true
	default:
		return string(got) == strings.ToUpper(strings.TrimSpace(want))
	}
}

func incidentCreationTimeMatches(raw string, after string, before string) bool {
	if after == "" && before == "" {
		return true
	}
	got, err := parseIncidentTime(raw)
	if err != nil {
		return false
	}
	if after != "" {
		bound, err := parseIncidentTime(after)
		if err != nil || got.Before(bound) {
			return false
		}
	}
	if before != "" {
		bound, err := parseIncidentTime(before)
		if err != nil || got.After(bound) {
			return false
		}
	}
	return true
}

func parseIncidentTime(raw string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.DateOnly, raw); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("parse %q as incident timestamp", raw)
}

func incidentLocalFilteringRequired(filter d.IncidentFilter) bool {
	state := strings.ToLower(strings.TrimSpace(filter.State))
	return state != "all" ||
		filter.ErrorType != "" ||
		filter.ErrorMessage != "" ||
		filter.ProcessInstanceKey != "" ||
		filter.RootProcessInstanceKey != "" ||
		filter.ProcessDefinitionKey != "" ||
		filter.ProcessDefinitionId != "" ||
		filter.FlowNodeId != "" ||
		filter.FlowNodeInstanceKey != "" ||
		filter.CreationTimeAfter != "" ||
		filter.CreationTimeBefore != ""
}

func (s *Service) searchProcessInstanceIncidentsPages(ctx context.Context, key string, tenant string, callCfg *services.CallCfg) ([]d.ProcessInstanceIncidentDetail, error) {
	const pageSize int32 = 1000
	var out []d.ProcessInstanceIncidentDetail
	for from := int32(0); ; {
		page := newSearchQueryPageRequest(d.ProcessInstancePageRequest{From: from, Size: pageSize})
		body := camundav88.SearchProcessInstanceIncidentsJSONRequestBody{
			Page: &page,
		}
		resp, err := s.cc.SearchProcessInstanceIncidentsWithResponse(ctx, key, body)
		if err != nil {
			return nil, err
		}
		payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
		if err != nil {
			return nil, err
		}
		out = append(out, filterIncidentResults(key, tenant, callCfg.IncidentState, callCfg.IncidentErrorType, callCfg.IncidentErrorMessage, payload.Items)...)
		if !incidentSearchHasMore(payload.Page, from, len(payload.Items), pageSize) {
			return out, nil
		}
		from += int32(len(payload.Items))
	}
}

func incidentSearchHasMore(page camundav88.SearchQueryPageResponse, from int32, itemCount int, pageSize int32) bool {
	if itemCount == 0 {
		return false
	}
	visibleCount := int64(from) + int64(itemCount)
	if page.TotalItems > visibleCount {
		return true
	}
	return page.HasMoreTotalItems && itemCount >= int(pageSize)
}

func incidentSearchOverflowState(page camundav88.SearchQueryPageResponse, req d.IncidentPageRequest, itemCount int) d.ProcessInstanceOverflowState {
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

func incidentReportedTotal(page camundav88.SearchQueryPageResponse, itemCount int, localFiltering bool) *d.IncidentReportedTotal {
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

func newIncidentSearchStateFilter(state string) (*camundav88.IncidentStateFilterProperty, error) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "", "active":
		return newIncidentStateEqFilterPtr(camundav88.IncidentStateEnumACTIVE)
	case "pending":
		return newIncidentStateEqFilterPtr(camundav88.IncidentStateEnumPENDING)
	case "resolved":
		return newIncidentStateEqFilterPtr(camundav88.IncidentStateEnumRESOLVED)
	case "migrated":
		return newIncidentStateEqFilterPtr(camundav88.IncidentStateEnumMIGRATED)
	case "unknown":
		return newIncidentStateEqFilterPtr(camundav88.IncidentStateEnumUNKNOWN)
	case "all":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported incident state %q", state)
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

// newSearchQueryPageRequest builds the v8.8 page request for incident lookups.
func newSearchQueryPageRequest(pageReq d.ProcessInstancePageRequest) camundav88.SearchQueryPageRequest {
	page := camundav88.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav88.OffsetPagination{
		From:  &pageReq.From,
		Limit: &pageReq.Size,
	})
	return page
}

func newIncidentSearchQueryPageRequest(pageReq d.IncidentPageRequest) camundav88.SearchQueryPageRequest {
	page := camundav88.SearchQueryPageRequest{}
	_ = page.FromOffsetPagination(camundav88.OffsetPagination{
		From:  &pageReq.From,
		Limit: &pageReq.Size,
	})
	return page
}
