// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/consts"
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
	resp, err := services.RetryCamundaMutation(ctx, s.log, "resolve incident", func(ctx context.Context) (*camundav88.ResolveIncidentResponse, *http.Response, []byte, error) {
		resp, err := s.cc.ResolveIncidentWithResponse(ctx, key, camundav88.ResolveIncidentJSONRequestBody{})
		if resp == nil {
			return resp, nil, nil, err
		}
		return resp, resp.HTTPResponse, resp.Body, err
	})
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
	s.log.Debug(fmt.Sprintf("searching pi %s incidents", key))
	return s.searchProcessInstanceIncidentsPages(ctx, key, common.EffectiveTenant(s.cfg), callCfg)
}

// SearchIncidents returns up to size top-level incidents after version-compatible filtering.
func (s *Service) SearchIncidents(ctx context.Context, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	result, err := s.SearchIncidentsPages(ctx, filter, d.IncidentPageRequest{Size: size}, size, nil, opts...)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// SearchIncidentsPages owns page advancement, per-page size capping, local
// compatibility filtering, and caller-limit trimming for top-level incident search.
func (s *Service) SearchIncidentsPages(ctx context.Context, filter d.IncidentFilter, page d.IncidentPageRequest, limit int32, visitor d.IncidentSearchPageVisitor, opts ...services.CallOption) (d.IncidentSearchPagesResult, error) {
	_ = services.ApplyCallOptions(opts)
	if page.Size <= 0 {
		page.Size = consts.MaxPISearchSize
	}
	batchSize := page.Size
	items := make([]d.ProcessInstanceIncidentDetail, 0, minPositiveIncidentSearchSize(batchSize, limit))
	pages := int32(0)
	for {
		if limit > 0 && int32(len(items)) >= limit {
			break
		}
		page.Size = batchSize
		resultPage, err := s.SearchIncidentsPage(ctx, filter, page, opts...)
		if err != nil {
			return d.IncidentSearchPagesResult{}, err
		}
		if limit > 0 {
			remaining := int(limit) - len(items)
			if remaining <= 0 {
				break
			}
			if len(resultPage.Items) > remaining {
				resultPage.Items = resultPage.Items[:remaining]
			}
		}
		items = append(items, resultPage.Items...)
		pages++
		limitReached := limit > 0 && int32(len(items)) >= limit
		if visitor != nil {
			action, err := visitor(d.IncidentSearchPageStep{
				Page:            resultPage,
				CumulativeCount: int32(len(items)),
				LimitReached:    limitReached,
			})
			if err != nil {
				return d.IncidentSearchPagesResult{}, err
			}
			if action == d.IncidentSearchPageActionStop {
				break
			}
		}
		if limitReached {
			break
		}
		if resultPage.OverflowState != d.ProcessInstanceOverflowStateHasMore {
			break
		}
		page = nextIncidentSearchPageRequest(page, resultPage)
	}
	return d.IncidentSearchPagesResult{
		Items: items,
		Limit: limit,
		Pages: pages,
	}, nil
}

func nextIncidentSearchPageRequest(current d.IncidentPageRequest, page d.IncidentPage) d.IncidentPageRequest {
	nextFrom := current.From + current.Size
	if page.EndCursor != "" {
		return d.IncidentPageRequest{From: nextFrom, Size: current.Size, After: page.EndCursor}
	}
	return d.IncidentPageRequest{From: nextFrom, Size: current.Size}
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
		EndCursor:     toolx.Deref(payload.Page.EndCursor, ""),
	}, nil
}

// SearchIncidentsTotal returns exact backend totals when compatible and
// otherwise falls back to service-owned page counting.
func (s *Service) SearchIncidentsTotal(ctx context.Context, filter d.IncidentFilter, page d.IncidentPageRequest, opts ...services.CallOption) (int64, error) {
	var total int64
	_, err := s.SearchIncidentsPages(ctx, filter, page, 0, func(step d.IncidentSearchPageStep) (d.IncidentSearchPageAction, error) {
		if step.Page.ReportedTotal != nil && step.Page.ReportedTotal.Kind == d.IncidentReportedTotalKindExact {
			total = step.Page.ReportedTotal.Count
			return d.IncidentSearchPageActionStop, nil
		}
		total += int64(len(step.Page.Items))
		return d.IncidentSearchPageActionContinue, nil
	}, opts...)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func minPositiveIncidentSearchSize(batchSize int32, limit int32) int {
	if limit > 0 && limit < batchSize {
		return int(limit)
	}
	return int(batchSize)
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
		if !incidentKeyMatches(filter.Keys, item.IncidentKey) {
			continue
		}
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
		if filter.ElementId != "" && item.ElementId != filter.ElementId {
			continue
		}
		if filter.ElementInstanceKey != "" && item.ElementInstanceKey != filter.ElementInstanceKey {
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
	state, ok := incidentfilter.NormalizeState(want)
	if !ok {
		return false
	}
	switch state {
	case "", "active":
		return got == camundav88.IncidentStateEnumACTIVE
	case "all":
		return true
	default:
		return strings.EqualFold(string(got), state)
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
		bound, err := parseIncidentTimeLowerBound(after)
		if err != nil || got.Before(bound) {
			return false
		}
	}
	if before != "" {
		bound, err := parseIncidentTimeUpperBound(before)
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

func parseIncidentTimeLowerBound(raw string) (time.Time, error) {
	if t, ok := parseIncidentTimestamp(raw); ok {
		return t, nil
	}
	if t, err := time.Parse(time.DateOnly, raw); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("parse %q as incident timestamp", raw)
}

func parseIncidentTimeUpperBound(raw string) (time.Time, error) {
	if t, ok := parseIncidentTimestamp(raw); ok {
		return t, nil
	}
	if t, err := time.Parse(time.DateOnly, raw); err == nil {
		return t.AddDate(0, 0, 1).Add(-time.Nanosecond), nil
	}
	return time.Time{}, fmt.Errorf("parse %q as incident timestamp", raw)
}

func parseIncidentTimestamp(raw string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05.999999999", raw, time.UTC); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", raw, time.UTC); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func incidentLocalFilteringRequired(filter d.IncidentFilter) bool {
	state, _ := incidentfilter.NormalizeState(filter.State)
	return len(filter.Keys) > 0 ||
		state != "all" ||
		filter.ErrorType != "" ||
		filter.ErrorMessage != "" ||
		filter.ProcessInstanceKey != "" ||
		filter.RootProcessInstanceKey != "" ||
		filter.ProcessDefinitionKey != "" ||
		filter.ProcessDefinitionId != "" ||
		filter.ElementId != "" ||
		filter.ElementInstanceKey != "" ||
		filter.CreationTimeAfter != "" ||
		filter.CreationTimeBefore != ""
}

func incidentKeyMatches(keys []string, got string) bool {
	if len(keys) == 0 {
		return true
	}
	for _, key := range keys {
		if key == got {
			return true
		}
	}
	return false
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
	if itemCount == 0 && page.TotalItems > int64(from)+int64(pageSize) {
		return true
	}
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
	if itemCount == 0 && page.TotalItems > int64(req.From)+int64(req.Size) {
		return d.ProcessInstanceOverflowStateHasMore
	}
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
	normalized, ok := incidentfilter.NormalizeState(state)
	if !ok {
		return nil, fmt.Errorf("unsupported incident state %q", state)
	}
	switch normalized {
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
	if pageReq.After != "" {
		_ = page.FromCursorForwardPagination(camundav88.CursorForwardPagination{
			After: pageReq.After,
			Limit: &pageReq.Size,
		})
		return page
	}
	_ = page.FromOffsetPagination(camundav88.OffsetPagination{
		From:  &pageReq.From,
		Limit: &pageReq.Size,
	})
	return page
}
