// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/consts"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/internal/services/incident/waiter"
	"github.com/grafvonb/c8volt/internal/services/incidentfilter"
	"github.com/grafvonb/c8volt/internal/services/processinstance/searchpaging"
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
	resp, err := services.RetryCamundaMutation(ctx, s.log, "resolve incident", func(ctx context.Context) (*camundav810.ResolveIncidentResponse, *http.Response, []byte, error) {
		resp, err := s.cc.ResolveIncidentWithResponse(ctx, key, camundav810.ResolveIncidentJSONRequestBody{})
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
	filter := &camundav810.IncidentFilter{
		State:     stateFilter,
		TenantId:  tenantFilter,
		ErrorType: errorTypeFilter,
	}
	return s.searchProcessInstanceIncidentsPages(ctx, key, filter, callCfg.IncidentErrorMessage)
}

// SearchIncidents returns up to size top-level incidents after server-safe and local filters.
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

// SearchIncidentsPage uses safe server-side filters on v8.10 and applies filters
// whose semantics are not represented by the API locally.
func (s *Service) SearchIncidentsPage(ctx context.Context, filter d.IncidentFilter, pageReq d.IncidentPageRequest, opts ...services.CallOption) (d.IncidentPage, error) {
	_ = services.ApplyCallOptions(opts)
	bodyFilter, err := s.newIncidentFilter(filter)
	if err != nil {
		return d.IncidentPage{}, err
	}
	page := newIncidentSearchQueryPageRequest(pageReq)
	body := camundav810.SearchIncidentsJSONRequestBody{
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

func (s *Service) newIncidentFilter(filter d.IncidentFilter) (*camundav810.IncidentFilter, error) {
	tenantFilter, err := newStringEqFilterPtr(s.cfg.App.Tenant)
	if err != nil {
		return nil, fmt.Errorf("building tenant incident filter: %w", err)
	}
	incidentKeyFilter, err := newBasicStringInFilterPtr(filter.Keys)
	if err != nil {
		return nil, fmt.Errorf("building incident key filter: %w", err)
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
	elementIDFilter, err := newStringEqFilterPtr(filter.ElementId)
	if err != nil {
		return nil, fmt.Errorf("building incident element-id filter: %w", err)
	}
	elementInstanceKeyFilter, err := newElementInstanceKeyEqFilterPtr(filter.ElementInstanceKey)
	if err != nil {
		return nil, fmt.Errorf("building incident element-instance-key filter: %w", err)
	}
	creationTimeAfter, err := parseIncidentTimeLowerBound(filter.CreationTimeAfter)
	if err != nil {
		return nil, fmt.Errorf("building incident creation-time-after filter: %w", err)
	}
	creationTimeBefore, err := parseIncidentTimeUpperBound(filter.CreationTimeBefore)
	if err != nil {
		return nil, fmt.Errorf("building incident creation-time-before filter: %w", err)
	}
	creationTimeFilter, err := newDateTimeRangeFilterPtr(creationTimeAfter, creationTimeBefore, nil)
	if err != nil {
		return nil, fmt.Errorf("building incident creation-time filter: %w", err)
	}
	bodyFilter := &camundav810.IncidentFilter{
		TenantId:             tenantFilter,
		IncidentKey:          incidentKeyFilter,
		State:                stateFilter,
		ErrorType:            errorTypeFilter,
		ProcessInstanceKey:   processInstanceKeyFilter,
		ProcessDefinitionKey: processDefinitionKeyFilter,
		ProcessDefinitionId:  processDefinitionIDFilter,
		ElementId:            elementIDFilter,
		ElementInstanceKey:   elementInstanceKeyFilter,
		CreationTime:         creationTimeFilter,
	}
	if bodyFilter.TenantId == nil &&
		bodyFilter.IncidentKey == nil &&
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

func newIncidentSearchStateFilter(state string) (*camundav810.IncidentStateFilterProperty, error) {
	normalized, ok := incidentfilter.NormalizeState(state)
	if !ok {
		return nil, fmt.Errorf("unsupported incident state %q", state)
	}
	switch normalized {
	case "", "active":
		return newIncidentStateEqFilterPtr(camundav810.IncidentStateEnumACTIVE)
	case "pending":
		return newIncidentStateEqFilterPtr(camundav810.IncidentStateEnumPENDING)
	case "resolved":
		return newIncidentStateEqFilterPtr(camundav810.IncidentStateEnumRESOLVED)
	case "migrated":
		return newIncidentStateEqFilterPtr(camundav810.IncidentStateEnumMIGRATED)
	case "unknown":
		return newIncidentStateEqFilterPtr(camundav810.IncidentStateEnumUNKNOWN)
	case "all":
		return nil, nil
	default:
		return nil, fmt.Errorf("unsupported incident state %q", state)
	}
}

func filterIncidentDetailsLocally(filter d.IncidentFilter, items []d.ProcessInstanceIncidentDetail) []d.ProcessInstanceIncidentDetail {
	out := make([]d.ProcessInstanceIncidentDetail, 0, len(items))
	for _, item := range items {
		if !incidentKeyMatches(filter.Keys, item.IncidentKey) {
			continue
		}
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
	return len(filter.Keys) > 0 || filter.RootProcessInstanceKey != "" || filter.ErrorMessage != ""
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

func parseIncidentTimeLowerBound(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	if t, ok := parseIncidentTimestamp(raw); ok {
		return &t, nil
	}
	if t, err := time.Parse(time.DateOnly, raw); err == nil {
		return &t, nil
	}
	return nil, fmt.Errorf("parse %q as incident timestamp", raw)
}

func parseIncidentTimeUpperBound(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	if t, ok := parseIncidentTimestamp(raw); ok {
		return &t, nil
	}
	if t, err := time.Parse(time.DateOnly, raw); err == nil {
		t = t.AddDate(0, 0, 1).Add(-time.Nanosecond)
		return &t, nil
	}
	return nil, fmt.Errorf("parse %q as incident timestamp", raw)
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

func newIncidentSearchErrorTypeFilter(errorType string) (*camundav810.IncidentErrorTypeFilterProperty, error) {
	normalized, ok := incidentfilter.NormalizeErrorType(errorType)
	if !ok {
		return nil, fmt.Errorf("unsupported incident error type %q", errorType)
	}
	if normalized == "" {
		return nil, nil
	}
	return newIncidentErrorTypeEqFilterPtr(camundav810.IncidentErrorTypeEnum(normalized))
}

// searchProcessInstanceIncidentsPages exhausts scoped reads, rejecting malformed or cyclic continuation before planning can succeed.
func (s *Service) searchProcessInstanceIncidentsPages(ctx context.Context, key string, filter *camundav810.IncidentFilter, errorMessage string) ([]d.ProcessInstanceIncidentDetail, error) {
	req := d.ProcessInstancePageRequest{Size: 1000}
	var out []d.ProcessInstanceIncidentDetail
	seenCursors, seenPages, seenKeys := map[string]struct{}{}, map[string]bool{}, map[string]bool{}
	for {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		page := newIncidentSearchQueryPageRequest(d.IncidentPageRequest{From: req.From, Size: req.Size, After: req.After})
		resp, err := s.cc.SearchProcessInstanceIncidentsWithResponse(ctx, key, camundav810.SearchProcessInstanceIncidentsJSONRequestBody{Page: &page, Filter: filter})
		if err != nil {
			return out, err
		}
		payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
		if err != nil {
			return out, err
		}
		raw := toolx.MapSlice(payload.Items, fromIncidentResult)
		// Shared paging guards use only identities and continuation facts. This local
		// projection never becomes a process observation or escapes into the plan.
		metadata := d.ProcessInstancePage{
			Request:       req,
			EndCursor:     toolx.Deref(payload.Page.EndCursor, ""),
			OverflowState: d.ProcessInstanceOverflowStateNoMore,
			ReportedTotal: &d.ProcessInstanceReportedTotal{
				Count: payload.Page.TotalItems,
				Kind:  d.ProcessInstanceReportedTotalKindExact,
			},
		}
		if payload.Page.HasMoreTotalItems {
			metadata.OverflowState = d.ProcessInstanceOverflowStateIndeterminate
			metadata.ReportedTotal.Kind = d.ProcessInstanceReportedTotalKindLowerBound
		} else if payload.Page.TotalItems > int64(req.From)+int64(len(raw)) {
			metadata.OverflowState = d.ProcessInstanceOverflowStateHasMore
		}
		var signature strings.Builder
		for _, item := range raw {
			if item.IncidentKey == "" {
				return out, fmt.Errorf("%w: incident without key", d.ErrMalformedResponse)
			}
			metadata.Items = append(metadata.Items, d.ProcessInstance{Key: item.IncidentKey})
			fmt.Fprintf(&signature, "%d:%s;", len(item.IncidentKey), item.IncidentKey)
		}
		if err := searchpaging.ValidatePage(metadata); err != nil {
			return out, err
		}
		for _, item := range raw {
			if !seenKeys[item.IncidentKey] {
				seenKeys[item.IncidentKey] = true
				if incidentfilter.ErrorMessageContains(errorMessage, item.ErrorMessage) {
					out = append(out, item)
				}
			}
		}
		exhausted, err := searchpaging.Exhausted(metadata, int64(len(seenKeys)))
		if err != nil {
			return out, err
		}
		if exhausted {
			return out, ctx.Err()
		}
		if metadata.EndCursor == "" && len(raw) > 0 {
			if seenPages[signature.String()] {
				return out, fmt.Errorf("%w: incident page did not advance", d.ErrMalformedResponse)
			}
			seenPages[signature.String()] = true
		}
		req, err = searchpaging.Advance(req, metadata.EndCursor, len(raw), seenCursors)
		if err != nil {
			return out, err
		}
	}
}

func incidentSearchOverflowState(page camundav810.SearchQueryPageResponse, req d.IncidentPageRequest, itemCount int) d.ProcessInstanceOverflowState {
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

func incidentReportedTotal(page camundav810.SearchQueryPageResponse, itemCount int, localFiltering bool) *d.IncidentReportedTotal {
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

func nextIncidentSearchPageRequest(current d.IncidentPageRequest, page d.IncidentPage) d.IncidentPageRequest {
	nextFrom := current.From + current.Size
	if page.EndCursor != "" {
		return d.IncidentPageRequest{From: nextFrom, Size: current.Size, After: page.EndCursor}
	}
	return d.IncidentPageRequest{From: nextFrom, Size: current.Size}
}

// WaitForIncidentResolved polls direct incident lookup until the selected incident is no longer active.
func (s *Service) WaitForIncidentResolved(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	return waiter.WaitForIncidentResolved(ctx, s, s.cfg, s.log, key, opts...)
}

// WaitForProcessInstanceIncidentsResolved polls process-instance incident lookup until the initial incident set is gone.
func (s *Service) WaitForProcessInstanceIncidentsResolved(ctx context.Context, processInstanceKey string, incidentKeys []string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	return waiter.WaitForProcessInstanceIncidentsResolved(ctx, s, s.cfg, s.log, processInstanceKey, incidentKeys, opts...)
}

func newIncidentSearchQueryPageRequest(pageReq d.IncidentPageRequest) camundav810.SearchQueryPageRequest {
	page := camundav810.SearchQueryPageRequest{}
	if pageReq.After != "" {
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
