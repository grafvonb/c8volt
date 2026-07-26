// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/typex"
)

// SearchProcessInstancesPages owns process-instance page traversal, limit
// trimming, local compatibility filtering, and optional direct incident-index
// lookup while callers retain rendering and prompt decisions.
func SearchProcessInstancesPages(ctx context.Context, piAPI API, incAPI SearchProcessInstanceIncidentAPI, request d.ProcessInstanceSearchRequest, visitor d.ProcessInstanceSearchPageVisitor, opts ...services.CallOption) (d.ProcessInstanceSearchPagesResult, error) {
	_ = services.ApplyCallOptions(opts)
	normalizeProcessInstanceSearchRequest(&request)
	if request.DirectIncidentIndex {
		return searchProcessInstancesViaDirectIncidentIndex(ctx, piAPI, incAPI, request, visitor, opts...)
	}

	pageReq := request.Page
	batchSize := pageReq.Size
	items := make([]d.ProcessInstance, 0, minPositiveProcessInstanceSearchSize(batchSize, request.Limit))
	pages := int32(0)
	for {
		if processInstanceSearchLimitReached(len(items), request.Limit) {
			break
		}
		pageReq.Size = batchSize
		page, err := piAPI.SearchForProcessInstancesPage(ctx, request.Filter, pageReq, opts...)
		if err != nil {
			return d.ProcessInstanceSearchPagesResult{}, err
		}
		rawCount := len(page.Items)
		filteredItems, err := filterProcessInstanceSearchItems(ctx, piAPI, incAPI, page.Items, request.LocalFilters, opts...)
		if err != nil {
			return d.ProcessInstanceSearchPagesResult{}, err
		}
		page.Items = limitProcessInstanceSearchItems(filteredItems, request.Limit, int32(len(items)))
		items = append(items, page.Items...)
		pages++

		limitReached := processInstanceSearchLimitReached(len(items), request.Limit)
		if visitor != nil {
			action, err := visitor(d.ProcessInstanceSearchPageStep{
				Page:            page,
				CumulativeCount: int32(len(items)),
				LimitReached:    limitReached,
			})
			if err != nil {
				return d.ProcessInstanceSearchPagesResult{}, err
			}
			if action == d.ProcessInstanceSearchPageActionStop {
				break
			}
		}
		if limitReached || page.OverflowState != d.ProcessInstanceOverflowStateHasMore {
			break
		}
		pageReq = nextProcessInstanceSearchPageRequest(pageReq, page, rawCount)
	}
	return d.ProcessInstanceSearchPagesResult{
		Items: items,
		Limit: request.Limit,
		Pages: pages,
	}, nil
}

// SearchProcessInstancesTotal uses exact backend totals when compatible and
// otherwise counts filtered pages through service-owned traversal.
func SearchProcessInstancesTotal(ctx context.Context, piAPI API, incAPI SearchProcessInstanceIncidentAPI, request d.ProcessInstanceSearchRequest, visitor d.ProcessInstanceSearchTotalVisitor, opts ...services.CallOption) (int64, error) {
	_ = services.ApplyCallOptions(opts)
	normalizeProcessInstanceSearchRequest(&request)
	if request.DirectIncidentIndex {
		result, err := SearchProcessInstancesPages(ctx, piAPI, incAPI, request, nil, opts...)
		if err != nil {
			return 0, err
		}
		return int64(len(result.Items)), nil
	}

	pageReq := request.Page
	batchSize := pageReq.Size
	total := int64(0)
	countingByPaging := false
	for {
		pageReq.Size = batchSize
		page, err := piAPI.SearchForProcessInstancesPage(ctx, request.Filter, pageReq, opts...)
		if err != nil {
			return 0, err
		}
		if request.ReportedTotalAllowed && !request.LocalFilters.Active() && page.ReportedTotal != nil && page.ReportedTotal.Kind == d.ProcessInstanceReportedTotalKindExact {
			if visitor != nil {
				if err := visitor(d.ProcessInstanceSearchTotalStep{
					Page:           page,
					FilteredCount:  int32(len(page.Items)),
					TotalBefore:    total,
					TotalAfter:     page.ReportedTotal.Count,
					ExactTotalUsed: true,
				}); err != nil {
					return 0, err
				}
			}
			return page.ReportedTotal.Count, nil
		}

		countingByPaging = true
		filteredItems, err := filterProcessInstanceSearchItems(ctx, piAPI, incAPI, page.Items, request.LocalFilters, opts...)
		if err != nil {
			return 0, err
		}
		before := total
		total += int64(len(filteredItems))
		if visitor != nil {
			if err := visitor(d.ProcessInstanceSearchTotalStep{
				Page:             page,
				FilteredCount:    int32(len(filteredItems)),
				TotalBefore:      before,
				TotalAfter:       total,
				CountingByPaging: countingByPaging,
			}); err != nil {
				return 0, err
			}
		}
		if len(page.Items) == 0 || page.OverflowState == d.ProcessInstanceOverflowStateNoMore {
			return total, nil
		}
		pageReq = nextProcessInstanceSearchPageRequest(pageReq, page, len(page.Items))
	}
}

// normalizeProcessInstanceSearchRequest applies default page sizing once before traversal.
func normalizeProcessInstanceSearchRequest(request *d.ProcessInstanceSearchRequest) {
	if request.Page.Size <= 0 {
		request.Page.Size = consts.MaxPISearchSize
	}
}

// searchProcessInstancesViaDirectIncidentIndex collects unique process-instance
// keys from incidents before loading those process instances in one bounded bulk call.
func searchProcessInstancesViaDirectIncidentIndex(ctx context.Context, piAPI API, incAPI SearchProcessInstanceIncidentAPI, request d.ProcessInstanceSearchRequest, visitor d.ProcessInstanceSearchPageVisitor, opts ...services.CallOption) (d.ProcessInstanceSearchPagesResult, error) {
	if incAPI == nil {
		return d.ProcessInstanceSearchPagesResult{}, fmt.Errorf("%w: direct incident process-instance search requires incident lookup support", d.ErrUnsupported)
	}
	keys, pages, err := collectDirectIncidentProcessInstanceKeys(ctx, incAPI, request, opts...)
	if err != nil {
		return d.ProcessInstanceSearchPagesResult{}, err
	}
	if len(keys) == 0 {
		return d.ProcessInstanceSearchPagesResult{Limit: request.Limit, Pages: pages}, nil
	}
	items, err := GetProcessInstances(ctx, piAPI, keys, len(keys), opts...)
	if err != nil {
		return d.ProcessInstanceSearchPagesResult{}, fmt.Errorf("get process instances for direct incident matches: %w", err)
	}
	items = filterDirectIncidentIndexedProcessInstances(items, request.Filter)
	items = limitProcessInstanceSearchItems(items, request.Limit, 0)
	page := d.ProcessInstancePage{
		Items:         items,
		Request:       request.Page,
		OverflowState: d.ProcessInstanceOverflowStateNoMore,
	}
	if visitor != nil {
		if _, err := visitor(d.ProcessInstanceSearchPageStep{
			Page:            page,
			CumulativeCount: int32(len(items)),
			LimitReached:    processInstanceSearchLimitReached(len(items), request.Limit),
		}); err != nil {
			return d.ProcessInstanceSearchPagesResult{}, err
		}
	}
	return d.ProcessInstanceSearchPagesResult{Items: items, Limit: request.Limit, Pages: pages}, nil
}

// collectDirectIncidentProcessInstanceKeys keeps walking incident pages until
// the requested number of unique process-instance keys is found or incidents end.
func collectDirectIncidentProcessInstanceKeys(ctx context.Context, incAPI SearchProcessInstanceIncidentAPI, request d.ProcessInstanceSearchRequest, opts ...services.CallOption) (typex.Keys, int32, error) {
	pageReq := d.IncidentPageRequest{Size: request.Page.Size}
	limit := int(request.Limit)
	keys := make(typex.Keys, 0, limit)
	seen := make(map[string]struct{}, limit)
	pages := int32(0)
	for {
		page, err := incAPI.SearchIncidentsPage(ctx, request.DirectIncidentFilter, pageReq, incidentSearchOptions(request.LocalFilters, opts...)...)
		if err != nil {
			return nil, pages, fmt.Errorf("search direct incidents: %w", err)
		}
		pages++
		for _, item := range page.Items {
			if item.ProcessInstanceKey == "" {
				continue
			}
			if _, ok := seen[item.ProcessInstanceKey]; ok {
				continue
			}
			seen[item.ProcessInstanceKey] = struct{}{}
			keys = append(keys, item.ProcessInstanceKey)
			if limit > 0 && len(keys) >= limit {
				return keys, pages, nil
			}
		}
		if page.OverflowState == d.ProcessInstanceOverflowStateNoMore {
			return keys, pages, nil
		}
		pageReq = nextDirectIncidentSearchPageRequest(pageReq, page)
	}
}

// filterProcessInstanceSearchItems applies version-neutral local filters in the
// same order the command-owned loop historically used.
func filterProcessInstanceSearchItems(ctx context.Context, piAPI API, incAPI SearchProcessInstanceIncidentAPI, items []d.ProcessInstance, filters d.ProcessInstanceSearchLocalFilters, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	out := append([]d.ProcessInstance(nil), items...)
	if filters.ChildrenOnly {
		out = filterProcessInstanceItems(out, func(pi d.ProcessInstance) bool { return pi.ParentKey != "" })
	}
	if filters.RootsOnly {
		out = filterProcessInstanceItems(out, func(pi d.ProcessInstance) bool { return pi.ParentKey == "" })
	}
	if filters.OrphanChildrenOnly {
		orphans, err := piAPI.FilterProcessInstanceWithOrphanParent(ctx, out, opts...)
		if err != nil {
			return nil, fmt.Errorf("error filtering orphan children: %w", err)
		}
		out = orphans
	}
	if filters.IncidentsOnly {
		out = filterProcessInstanceItems(out, func(pi d.ProcessInstance) bool { return pi.Incident })
	}
	if filters.DirectIncidentsOnly {
		direct, err := filterProcessInstancesWithDirectIncidents(ctx, incAPI, out, filters, opts...)
		if err != nil {
			return nil, err
		}
		out = direct
	}
	if filters.NoIncidentsOnly {
		out = filterProcessInstanceItems(out, func(pi d.ProcessInstance) bool { return !pi.Incident })
	}
	return out, nil
}

// filterProcessInstancesWithDirectIncidents keeps only process instances that
// have at least one incident detail matching the request's incident filters.
func filterProcessInstancesWithDirectIncidents(ctx context.Context, incAPI SearchProcessInstanceIncidentAPI, items []d.ProcessInstance, filters d.ProcessInstanceSearchLocalFilters, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	if len(items) == 0 {
		return nil, nil
	}
	if incAPI == nil {
		return nil, fmt.Errorf("%w: direct incident process-instance filtering requires incident lookup support", d.ErrUnsupported)
	}
	enriched, err := EnrichProcessInstancesWithIncidents(ctx, incAPI, items, incidentSearchOptions(filters, opts...)...)
	if err != nil {
		return nil, fmt.Errorf("error filtering direct incidents: %w", err)
	}
	out := make([]d.ProcessInstance, 0, len(enriched.Items))
	for _, item := range enriched.Items {
		if len(item.Incidents) > 0 {
			out = append(out, item.Item)
		}
	}
	return out, nil
}

// incidentSearchOptions appends normalized incident filters for service-owned incident lookups.
func incidentSearchOptions(filters d.ProcessInstanceSearchLocalFilters, opts ...services.CallOption) []services.CallOption {
	out := append([]services.CallOption(nil), opts...)
	if filters.IncidentState != "" {
		out = append(out, services.WithIncidentState(filters.IncidentState))
	}
	if filters.IncidentErrorType != "" {
		out = append(out, services.WithIncidentErrorType(filters.IncidentErrorType))
	}
	if filters.IncidentErrorMessage != "" {
		out = append(out, services.WithIncidentErrorMessage(filters.IncidentErrorMessage))
	}
	return out
}

// filterProcessInstanceItems preserves input order while keeping matching items.
func filterProcessInstanceItems(items []d.ProcessInstance, pred func(d.ProcessInstance) bool) []d.ProcessInstance {
	if len(items) == 0 {
		return items
	}
	out := make([]d.ProcessInstance, 0, len(items))
	for _, item := range items {
		if pred(item) {
			out = append(out, item)
		}
	}
	return out
}

// filterDirectIncidentIndexedProcessInstances rechecks process-instance fields
// after keys are loaded from the incident index.
func filterDirectIncidentIndexedProcessInstances(items []d.ProcessInstance, filter d.ProcessInstanceFilter) []d.ProcessInstance {
	out := make([]d.ProcessInstance, 0, len(items))
	for _, item := range items {
		if filter.State != "" && item.State != filter.State {
			continue
		}
		if filter.BpmnProcessId != "" && item.BpmnProcessId != filter.BpmnProcessId {
			continue
		}
		if filter.ProcessDefinitionKey != "" && item.ProcessDefinitionKey != filter.ProcessDefinitionKey {
			continue
		}
		out = append(out, item)
	}
	return out
}

// limitProcessInstanceSearchItems trims a page to the remaining caller limit.
func limitProcessInstanceSearchItems(items []d.ProcessInstance, limit int32, cumulative int32) []d.ProcessInstance {
	if limit <= 0 {
		return items
	}
	remaining := int(limit - cumulative)
	if remaining <= 0 {
		return nil
	}
	if len(items) > remaining {
		return items[:remaining]
	}
	return items
}

// minPositiveProcessInstanceSearchSize picks a useful result slice capacity.
func minPositiveProcessInstanceSearchSize(batchSize int32, limit int32) int {
	if limit > 0 && limit < batchSize {
		return int(limit)
	}
	return int(batchSize)
}

// processInstanceSearchLimitReached checks a positive caller cap after local filtering.
func processInstanceSearchLimitReached(count int, limit int32) bool {
	return limit > 0 && int32(count) >= limit
}

// nextProcessInstanceSearchPageRequest advances by cursor when available and
// otherwise by the raw backend page size to avoid local-filter skips.
func nextProcessInstanceSearchPageRequest(current d.ProcessInstancePageRequest, page d.ProcessInstancePage, rawCount int) d.ProcessInstancePageRequest {
	if page.EndCursor != "" {
		return d.ProcessInstancePageRequest{Size: current.Size, After: page.EndCursor}
	}
	advance := rawCount
	if advance == 0 {
		advance = int(current.Size)
	}
	return d.ProcessInstancePageRequest{From: current.From + int32(advance), Size: current.Size}
}

// nextDirectIncidentSearchPageRequest advances incident pages by cursor when available.
func nextDirectIncidentSearchPageRequest(current d.IncidentPageRequest, page d.IncidentPage) d.IncidentPageRequest {
	if page.EndCursor != "" {
		return d.IncidentPageRequest{Size: current.Size, After: page.EndCursor}
	}
	advance := len(page.Items)
	if advance == 0 {
		advance = int(current.Size)
	}
	return d.IncidentPageRequest{From: current.From + int32(advance), Size: current.Size}
}
