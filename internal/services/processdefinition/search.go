// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processdefinition

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// SearchProcessDefinitionsPages owns process-definition page traversal and
// limit trimming while callers retain rendering and prompt decisions.
func SearchProcessDefinitionsPages(ctx context.Context, api API, request d.ProcessDefinitionSearchRequest, visitor d.ProcessDefinitionSearchPageVisitor, opts ...services.CallOption) (d.ProcessDefinitionSearchPagesResult, error) {
	normalizeProcessDefinitionSearchRequest(&request)

	pageReq := request.Page
	batchSize := pageReq.Size
	traversalLimit := request.Limit
	if request.Latest {
		traversalLimit = 0
		request.Filter.IsLatestVersion = true
	}
	items := make([]d.ProcessDefinition, 0, minPositiveProcessDefinitionSearchSize(batchSize, traversalLimit))
	pages := int32(0)
	for {
		if processDefinitionSearchLimitReached(len(items), traversalLimit) {
			break
		}
		pageReq.Size = batchSize
		page, err := api.SearchProcessDefinitionsPage(ctx, request.Filter, pageReq, opts...)
		if err != nil {
			return d.ProcessDefinitionSearchPagesResult{}, err
		}
		rawCount := len(page.Items)
		page.Items = limitProcessDefinitionSearchItems(page.Items, traversalLimit, int32(len(items)))
		items = append(items, page.Items...)
		pages++

		limitReached := processDefinitionSearchLimitReached(len(items), traversalLimit)
		if visitor != nil {
			action, err := visitor(d.ProcessDefinitionSearchPageStep{
				Page:            page,
				CumulativeCount: int32(len(items)),
				LimitReached:    limitReached,
			})
			if err != nil {
				return d.ProcessDefinitionSearchPagesResult{}, err
			}
			if action == d.ProcessDefinitionSearchPageActionStop {
				break
			}
		}
		if limitReached || page.OverflowState != d.ProcessInstanceOverflowStateHasMore {
			break
		}
		if rawCount == 0 && page.EndCursor == "" {
			break
		}
		pageReq = nextProcessDefinitionSearchPageRequest(pageReq, page, rawCount)
	}
	return d.ProcessDefinitionSearchPagesResult{
		Items: finalizeProcessDefinitionSearchItems(items, request),
		Limit: request.Limit,
		Pages: pages,
	}, nil
}

// CollectProcessDefinitionWatchSnapshot collects one complete process-definition
// snapshot while keeping selector dispatch and page traversal in the service layer.
func CollectProcessDefinitionWatchSnapshot(ctx context.Context, api API, request d.ProcessDefinitionWatchSnapshotRequest, opts ...services.CallOption) (d.ProcessDefinitionWatchSnapshot, error) {
	switch {
	case request.Key != "":
		item, err := api.GetProcessDefinition(ctx, request.Key, opts...)
		if err != nil {
			return d.ProcessDefinitionWatchSnapshot{}, err
		}
		return newProcessDefinitionWatchSnapshot([]d.ProcessDefinition{item}, 1, nil), nil
	case request.Latest:
		return collectProcessDefinitionPagedWatchSnapshot(ctx, api, d.ProcessDefinitionSearchRequest{
			Filter: request.Filter,
			Page:   request.Page,
			Latest: true,
		}, opts...)
	default:
		return collectProcessDefinitionPagedWatchSnapshot(ctx, api, d.ProcessDefinitionSearchRequest{
			Filter: request.Filter,
			Page:   request.Page,
		}, opts...)
	}
}

func collectProcessDefinitionPagedWatchSnapshot(ctx context.Context, api API, request d.ProcessDefinitionSearchRequest, opts ...services.CallOption) (d.ProcessDefinitionWatchSnapshot, error) {
	var reportedTotal *d.ProcessDefinitionReportedTotal
	result, err := SearchProcessDefinitionsPages(ctx, api, request, func(step d.ProcessDefinitionSearchPageStep) (d.ProcessDefinitionSearchPageAction, error) {
		if reportedTotal == nil && step.Page.ReportedTotal != nil {
			copied := *step.Page.ReportedTotal
			reportedTotal = &copied
		}
		return d.ProcessDefinitionSearchPageActionContinue, nil
	}, opts...)
	if err != nil {
		return d.ProcessDefinitionWatchSnapshot{}, err
	}
	return newProcessDefinitionWatchSnapshot(result.Items, result.Pages, reportedTotal), nil
}

// newProcessDefinitionWatchSnapshot derives count and empty metadata from the
// service-selected process-definition items.
func newProcessDefinitionWatchSnapshot(items []d.ProcessDefinition, pages int32, reportedTotal *d.ProcessDefinitionReportedTotal) d.ProcessDefinitionWatchSnapshot {
	return d.ProcessDefinitionWatchSnapshot{
		Items:         items,
		Total:         int32(len(items)),
		Pages:         pages,
		ReportedTotal: reportedTotal,
		Empty:         len(items) == 0,
	}
}

func normalizeProcessDefinitionSearchRequest(request *d.ProcessDefinitionSearchRequest) {
	if request.Page.Size <= 0 {
		request.Page.Size = MaxResultSize
	}
}

func minPositiveProcessDefinitionSearchSize(batchSize int32, limit int32) int {
	size := int(batchSize)
	if size <= 0 {
		size = int(MaxResultSize)
	}
	if limit > 0 && int(limit) < size {
		return int(limit)
	}
	return size
}

func processDefinitionSearchLimitReached(count int, limit int32) bool {
	return limit > 0 && count >= int(limit)
}

func limitProcessDefinitionSearchItems(items []d.ProcessDefinition, limit int32, cumulative int32) []d.ProcessDefinition {
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

func nextProcessDefinitionSearchPageRequest(current d.ProcessDefinitionPageRequest, page d.ProcessDefinitionPage, rawCount int) d.ProcessDefinitionPageRequest {
	advance := rawCount
	if advance == 0 {
		advance = int(current.Size)
	}
	next := d.ProcessDefinitionPageRequest{From: current.From + int32(advance), Size: current.Size}
	if page.EndCursor != "" {
		next.After = page.EndCursor
		return next
	}
	return next
}

// sortedProcessDefinitionSearchItems applies final version-neutral ordering
// after traversal without changing page visitor progress or limit selection.
func sortedProcessDefinitionSearchItems(items []d.ProcessDefinition) []d.ProcessDefinition {
	d.SortProcessDefinitionsCanonical(items)
	return items
}

func finalizeProcessDefinitionSearchItems(items []d.ProcessDefinition, request d.ProcessDefinitionSearchRequest) []d.ProcessDefinition {
	if request.Latest {
		items = latestProcessDefinitionSearchItems(items)
		items = sortedProcessDefinitionSearchItems(items)
		return limitProcessDefinitionSearchItems(items, request.Limit, 0)
	}
	return sortedProcessDefinitionSearchItems(items)
}

type processDefinitionLatestGroupKey struct {
	tenantID      string
	bpmnProcessID string
}

func latestProcessDefinitionSearchItems(items []d.ProcessDefinition) []d.ProcessDefinition {
	latestByGroup := make(map[processDefinitionLatestGroupKey]d.ProcessDefinition, len(items))
	for _, item := range items {
		group := processDefinitionLatestGroupKey{
			tenantID:      item.TenantId,
			bpmnProcessID: item.BpmnProcessId,
		}
		current, ok := latestByGroup[group]
		if !ok || processDefinitionSearchLatestCandidateLess(item, current) {
			latestByGroup[group] = item
		}
	}
	out := make([]d.ProcessDefinition, 0, len(latestByGroup))
	for _, item := range latestByGroup {
		out = append(out, item)
	}
	return out
}

func processDefinitionSearchLatestCandidateLess(candidate, current d.ProcessDefinition) bool {
	if candidate.ProcessVersion > current.ProcessVersion {
		return true
	}
	if candidate.ProcessVersion < current.ProcessVersion {
		return false
	}
	return candidate.Key < current.Key
}
