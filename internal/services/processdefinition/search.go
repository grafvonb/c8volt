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
	items := make([]d.ProcessDefinition, 0, minPositiveProcessDefinitionSearchSize(batchSize, request.Limit))
	pages := int32(0)
	for {
		if processDefinitionSearchLimitReached(len(items), request.Limit) {
			break
		}
		pageReq.Size = batchSize
		page, err := api.SearchProcessDefinitionsPage(ctx, request.Filter, pageReq, opts...)
		if err != nil {
			return d.ProcessDefinitionSearchPagesResult{}, err
		}
		rawCount := len(page.Items)
		page.Items = limitProcessDefinitionSearchItems(page.Items, request.Limit, int32(len(items)))
		items = append(items, page.Items...)
		pages++

		limitReached := processDefinitionSearchLimitReached(len(items), request.Limit)
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
		Items: items,
		Limit: request.Limit,
		Pages: pages,
	}, nil
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
	next := d.ProcessDefinitionPageRequest{Size: current.Size}
	if page.EndCursor != "" {
		next.After = page.EndCursor
		return next
	}
	next.From = current.From + int32(rawCount)
	return next
}
