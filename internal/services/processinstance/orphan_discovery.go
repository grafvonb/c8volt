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

type OrphanDiscoveryAPI interface {
	SearchForProcessInstancesPage(ctx context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error)
	FilterProcessInstanceWithOrphanParent(ctx context.Context, items []d.ProcessInstance, opts ...services.CallOption) ([]d.ProcessInstance, error)
}

type OrphanDiscoveryRequest struct {
	Filter    d.ProcessInstanceFilter
	BatchSize int32
	Limit     int32
}

type OrphanDiscovery struct {
	Filter d.ProcessInstanceFilter
	Items  []d.ProcessInstance
	Keys   typex.Keys
}

func DiscoverOrphanProcessInstances(ctx context.Context, api OrphanDiscoveryAPI, request OrphanDiscoveryRequest, opts ...services.CallOption) (OrphanDiscovery, error) {
	filter := request.Filter
	filter.HasParent = new(bool)
	*filter.HasParent = true

	pageReq := d.ProcessInstancePageRequest{Size: normalizeOrphanDiscoveryBatchSize(request.BatchSize)}
	var out OrphanDiscovery
	out.Filter = filter
	cumulativeOrphans := 0

	for {
		page, err := api.SearchForProcessInstancesPage(ctx, filter, pageReq, opts...)
		if err != nil {
			return OrphanDiscovery{}, err
		}
		if len(page.Items) > 0 {
			orphans, err := api.FilterProcessInstanceWithOrphanParent(ctx, page.Items, opts...)
			if err != nil {
				return OrphanDiscovery{}, err
			}
			limitedOrphans := limitOrphanDiscoveryItems(orphans, request.Limit, cumulativeOrphans)
			out.Items = append(out.Items, limitedOrphans...)
			for _, item := range limitedOrphans {
				out.Keys = append(out.Keys, item.Key)
			}
			cumulativeOrphans += len(limitedOrphans)
		}
		if shouldStopOrphanDiscovery(page, request.Limit, cumulativeOrphans) {
			out.Keys = out.Keys.Unique()
			return out, nil
		}
		pageReq = nextOrphanDiscoveryPageRequest(pageReq, page)
	}
}

func normalizeOrphanDiscoveryBatchSize(size int32) int32 {
	if size <= 0 || size > consts.MaxPISearchSize {
		return consts.MaxPISearchSize
	}
	return size
}

func limitOrphanDiscoveryItems(items []d.ProcessInstance, limit int32, cumulative int) []d.ProcessInstance {
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

func shouldStopOrphanDiscovery(page d.ProcessInstancePage, limit int32, cumulative int) bool {
	if limit > 0 && cumulative >= int(limit) {
		return true
	}
	if len(page.Items) == 0 {
		return true
	}
	return page.OverflowState != d.ProcessInstanceOverflowStateHasMore
}

func nextOrphanDiscoveryPageRequest(current d.ProcessInstancePageRequest, page d.ProcessInstancePage) d.ProcessInstancePageRequest {
	next := d.ProcessInstancePageRequest{Size: current.Size}
	if page.EndCursor != "" {
		next.After = page.EndCursor
		return next
	}
	next.From = current.From + int32(len(page.Items))
	return next
}
