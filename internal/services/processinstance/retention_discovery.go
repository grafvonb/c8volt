// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/typex"
)

type RetentionDiscoveryRequest struct {
	Filter    d.ProcessInstanceFilter
	BatchSize int32
	Limit     int32
}

type RetentionDiscovery struct {
	Filter d.ProcessInstanceFilter
	Items  []d.ProcessInstance
	Keys   typex.Keys
}

func DiscoverRetentionProcessInstances(ctx context.Context, api RetentionDiscoveryAPI, request RetentionDiscoveryRequest, opts ...services.CallOption) (RetentionDiscovery, error) {
	filter := request.Filter
	pageReq := d.ProcessInstancePageRequest{Size: normalizeRetentionDiscoveryBatchSize(request.BatchSize)}
	out := RetentionDiscovery{Filter: filter}
	cumulative := 0

	for {
		page, err := api.SearchForProcessInstancesPage(ctx, filter, pageReq, opts...)
		if err != nil {
			return RetentionDiscovery{}, err
		}
		if len(page.Items) > 0 {
			items := limitRetentionDiscoveryItems(filterRetentionDiscoveryItems(page.Items), request.Limit, cumulative)
			out.Items = append(out.Items, items...)
			for _, item := range items {
				out.Keys = append(out.Keys, item.Key)
			}
			cumulative += len(items)
		}
		if shouldStopRetentionDiscovery(page, request.Limit, cumulative) {
			out.Keys = out.Keys.Unique()
			return out, nil
		}
		pageReq = nextRetentionDiscoveryPageRequest(pageReq, page)
	}
}

func normalizeRetentionDiscoveryBatchSize(size int32) int32 {
	if size <= 0 || size > consts.MaxPISearchSize {
		return consts.MaxPISearchSize
	}
	return size
}

func filterRetentionDiscoveryItems(items []d.ProcessInstance) []d.ProcessInstance {
	out := make([]d.ProcessInstance, 0, len(items))
	for _, item := range items {
		if item.EndDate == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func limitRetentionDiscoveryItems(items []d.ProcessInstance, limit int32, cumulative int) []d.ProcessInstance {
	if limit <= 0 {
		return items
	}
	remaining := int(limit) - cumulative
	if remaining <= 0 {
		return nil
	}
	if len(items) > remaining {
		return items[:remaining]
	}
	return items
}

func shouldStopRetentionDiscovery(page d.ProcessInstancePage, limit int32, cumulative int) bool {
	if limit > 0 && cumulative >= int(limit) {
		return true
	}
	if len(page.Items) == 0 {
		return true
	}
	return page.OverflowState != d.ProcessInstanceOverflowStateHasMore
}

func nextRetentionDiscoveryPageRequest(current d.ProcessInstancePageRequest, page d.ProcessInstancePage) d.ProcessInstancePageRequest {
	next := d.ProcessInstancePageRequest{Size: current.Size}
	if page.EndCursor != "" {
		next.After = page.EndCursor
		return next
	}
	next.From = current.From + int32(len(page.Items))
	return next
}
