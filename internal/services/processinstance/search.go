// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/typex"
)

// SearchProcessInstancesPages owns process-instance page traversal, limit
// trimming, local compatibility filtering, and optional direct incident-index
// lookup while callers retain rendering and prompt decisions.
func SearchProcessInstancesPages(ctx context.Context, piAPI API, incAPI SearchProcessInstanceIncidentAPI, request d.ProcessInstanceSearchRequest, visitor d.ProcessInstanceSearchPageVisitor, opts ...services.CallOption) (d.ProcessInstanceSearchPagesResult, error) {
	result := d.ProcessInstanceSearchPagesResult{Limit: request.Limit, Completion: d.ProcessInstanceSearchCompletionIncomplete}
	filter, err := common.NormalizeProcessInstanceSearchFilter(request.Filter, piAPI.ValidateProcessInstanceStates)
	if err != nil {
		return result, err
	}
	request.Filter = filter
	_ = services.ApplyCallOptions(opts)
	normalizeProcessInstanceSearchRequest(&request)
	if request.DirectIncidentIndex {
		return searchProcessInstancesViaDirectIncidentIndex(ctx, piAPI, incAPI, request, visitor, opts...)
	}
	result.Items = make([]d.ProcessInstance, 0, minPositiveProcessInstanceSearchSize(request.Page.Size, request.Limit))
	seen := make(map[string]struct{})
	completion, err := walkProcessInstanceSearchPages(ctx, piAPI, request, false, func(page d.ProcessInstancePage, pages int32) (d.ProcessInstanceSearchCompletion, error) {
		result.Pages = pages
		filtered, err := filterProcessInstanceSearchItems(ctx, piAPI, incAPI, page.Items, request.LocalFilters, opts...)
		if err != nil {
			return "", err
		}
		page.Items = limitProcessInstanceSearchItems(distinctProcessInstanceSearchItems(filtered, seen), request.Limit, int32(len(result.Items)))
		result.Items = append(result.Items, page.Items...)
		limited := processInstanceSearchLimitReached(len(result.Items), request.Limit)
		action := d.ProcessInstanceSearchPageActionContinue
		if visitor != nil {
			action, err = visitor(d.ProcessInstanceSearchPageStep{Page: page, CumulativeCount: int32(len(result.Items)), LimitReached: limited})
			if err != nil {
				return "", err
			}
			if action != d.ProcessInstanceSearchPageActionContinue && action != d.ProcessInstanceSearchPageActionStop {
				return "", malformedProcessInstanceSearchMetadata("visitor action")
			}
		}
		// A command may return Stop for a limit or terminal page; retain the stronger service evidence.
		if limited {
			return d.ProcessInstanceSearchCompletionLimitReached, nil
		}
		if page.OverflowState == d.ProcessInstanceOverflowStateNoMore {
			return d.ProcessInstanceSearchCompletionExhausted, nil
		}
		if action == d.ProcessInstanceSearchPageActionStop {
			return d.ProcessInstanceSearchCompletionVisitorStopped, nil
		}
		return "", nil
	}, opts...)
	result.Completion = completion
	return result, err
}

// SearchProcessInstancesTotal uses exact backend totals when compatible and
// otherwise counts distinct filtered keys through the same validated traversal as collection.
func SearchProcessInstancesTotal(ctx context.Context, piAPI API, incAPI SearchProcessInstanceIncidentAPI, request d.ProcessInstanceSearchRequest, visitor d.ProcessInstanceSearchTotalVisitor, opts ...services.CallOption) (int64, error) {
	filter, err := common.NormalizeProcessInstanceSearchFilter(request.Filter, piAPI.ValidateProcessInstanceStates)
	if err != nil {
		return 0, err
	}
	request.Filter = filter
	_ = services.ApplyCallOptions(opts)
	normalizeProcessInstanceSearchRequest(&request)
	if request.DirectIncidentIndex {
		result, err := SearchProcessInstancesPages(ctx, piAPI, incAPI, request, nil, opts...)
		return int64(len(result.Items)), err
	}
	var total int64
	seen := make(map[string]struct{})
	_, err = walkProcessInstanceSearchPages(ctx, piAPI, request, request.ReportedTotalAllowed && !request.LocalFilters.Active(), func(page d.ProcessInstancePage, _ int32) (d.ProcessInstanceSearchCompletion, error) {
		step := d.ProcessInstanceSearchTotalStep{Page: page, TotalBefore: total}
		if request.ReportedTotalAllowed && !request.LocalFilters.Active() && page.ReportedTotal != nil && page.ReportedTotal.Kind == d.ProcessInstanceReportedTotalKindExact {
			total = page.ReportedTotal.Count
			step.ExactTotalUsed = true
			step.FilteredCount = int32(len(page.Items))
		} else {
			filtered, err := filterProcessInstanceSearchItems(ctx, piAPI, incAPI, page.Items, request.LocalFilters, opts...)
			if err != nil {
				return "", err
			}
			step.FilteredCount = int32(len(distinctProcessInstanceSearchItems(filtered, seen)))
			total += int64(step.FilteredCount)
			step.CountingByPaging = true
		}
		step.TotalAfter = total
		if visitor != nil {
			if err := visitor(step); err != nil {
				return "", err
			}
		}
		if step.ExactTotalUsed {
			return d.ProcessInstanceSearchCompletionExhausted, nil
		}
		return "", nil
	}, opts...)
	return total, err
}

// normalizeProcessInstanceSearchRequest applies default page sizing once before traversal.
func normalizeProcessInstanceSearchRequest(request *d.ProcessInstanceSearchRequest) {
	if request.Page.Size <= 0 {
		request.Page.Size = consts.MaxPISearchSize
	}
}

// searchProcessInstancesViaDirectIncidentIndex applies the selected bound after loading and filtering distinct incident-owned keys.
// The visitor still receives one aggregate page, preserving the optimized query's prompt policy.
func searchProcessInstancesViaDirectIncidentIndex(ctx context.Context, piAPI API, incAPI SearchProcessInstanceIncidentAPI, request d.ProcessInstanceSearchRequest, visitor d.ProcessInstanceSearchPageVisitor, opts ...services.CallOption) (d.ProcessInstanceSearchPagesResult, error) {
	result := d.ProcessInstanceSearchPagesResult{Limit: request.Limit, Completion: d.ProcessInstanceSearchCompletionIncomplete}
	if incAPI == nil {
		return result, fmt.Errorf("%w: direct incident process-instance search requires incident lookup support", d.ErrUnsupported)
	}
	pageReq := d.IncidentPageRequest{Size: request.Page.Size}
	seen := make(map[string]struct{})
	cursors := make(map[string]struct{})
	var rawObserved int64
	completion := d.ProcessInstanceSearchCompletionExhausted
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		page, err := incAPI.SearchIncidentsPage(ctx, request.DirectIncidentFilter, pageReq, incidentSearchOptions(request.LocalFilters, opts...)...)
		if err != nil {
			return result, fmt.Errorf("search direct incidents: %w", err)
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}
		result.Pages++
		metadata := d.ProcessInstancePage{Request: d.ProcessInstancePageRequest{From: pageReq.From, Size: pageReq.Size, After: pageReq.After}, Items: make([]d.ProcessInstance, len(page.Items)), OverflowState: page.OverflowState, EndCursor: page.EndCursor}
		if page.ReportedTotal != nil {
			metadata.ReportedTotal = &d.ProcessInstanceReportedTotal{Count: page.ReportedTotal.Count, Kind: d.ProcessInstanceReportedTotalKind(page.ReportedTotal.Kind)}
		}
		if err := validateProcessInstanceSearchPage(metadata); err != nil {
			return result, err
		}
		rawObserved += int64(len(page.Items))
		exhausted, err := processInstanceSearchExhausted(metadata, rawObserved)
		if err != nil {
			return result, err
		}
		if !exhausted && page.EndCursor != "" {
			if _, exists := cursors[page.EndCursor]; exists {
				return result, malformedProcessInstanceSearchMetadata("incident cursor did not advance or forms a cycle")
			}
		}
		keys := make(typex.Keys, 0, len(page.Items))
		for _, item := range page.Items {
			if item.ProcessInstanceKey == "" {
				continue
			}
			if _, exists := seen[item.ProcessInstanceKey]; exists {
				continue
			}
			seen[item.ProcessInstanceKey] = struct{}{}
			keys = append(keys, item.ProcessInstanceKey)
		}
		if len(keys) > 0 {
			items, err := GetProcessInstances(ctx, piAPI, keys, len(keys), opts...)
			selected := filterDirectIncidentIndexedProcessInstances(items, request.Filter)
			result.Items = append(result.Items, limitProcessInstanceSearchItems(selected, request.Limit, int32(len(result.Items)))...)
			if err != nil {
				return result, fmt.Errorf("get process instances for direct incident matches: %w", err)
			}
		}
		if processInstanceSearchLimitReached(len(result.Items), request.Limit) {
			completion = d.ProcessInstanceSearchCompletionLimitReached
			break
		}
		if exhausted {
			break
		}
		next, err := advanceProcessInstanceSearchPage(metadata.Request, page.EndCursor, len(page.Items), cursors)
		if err != nil {
			return result, err
		}
		pageReq = d.IncidentPageRequest{From: next.From, Size: next.Size, After: next.After}
	}
	if visitor != nil && len(result.Items) > 0 {
		action, err := visitor(d.ProcessInstanceSearchPageStep{Page: d.ProcessInstancePage{Items: result.Items, Request: request.Page, OverflowState: d.ProcessInstanceOverflowStateNoMore}, CumulativeCount: int32(len(result.Items)), LimitReached: processInstanceSearchLimitReached(len(result.Items), request.Limit)})
		if err != nil {
			return result, err
		}
		if action != d.ProcessInstanceSearchPageActionStop && action != d.ProcessInstanceSearchPageActionContinue {
			return result, malformedProcessInstanceSearchMetadata("visitor action")
		}
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	result.Completion = completion
	return result, nil
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
		if len(filter.States) > 0 && !filter.States.Contains(item.State) {
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
