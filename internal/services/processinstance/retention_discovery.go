// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/processinstance/searchpaging"
	"github.com/grafvonb/c8volt/typex"
)

type RetentionDiscoveryRequest struct {
	Filter    d.ProcessInstanceFilter
	BatchSize int32
	Limit     int32
	Progress  func(RetentionDiscoveryProgress)
}

type RetentionDiscoveryProgress struct {
	Page                d.ProcessInstancePage
	PageNumber          int
	CurrentPageCount    int
	CurrentPageSelected int
	CandidatesSeen      int
	CandidatesFrozen    int
	LimitReached        bool
}

// RetentionDiscovery retains completed selection evidence alongside eligible seeds.
type RetentionDiscovery struct {
	Selection d.ProcessInstanceSelection
	Filter    d.ProcessInstanceFilter
	Items     []d.ProcessInstance
	Keys      typex.Keys
}

// DiscoverRetentionProcessInstances shares authoritative paging and counts distinct ended seeds before the bound.
func DiscoverRetentionProcessInstances(ctx context.Context, api RetentionDiscoveryAPI, request RetentionDiscoveryRequest, opts ...services.CallOption) (RetentionDiscovery, error) {
	selectionRequest := d.ProcessInstanceSelectionRequest{Cardinality: d.ProcessInstanceSelectionCollection, SearchRequest: d.ProcessInstanceSearchRequest{Filter: request.Filter, Page: d.ProcessInstancePageRequest{Size: normalizeRetentionDiscoveryBatchSize(request.BatchSize)}, Limit: request.Limit}}
	selectionRequest = copyProcessInstanceSelectionRequest(selectionRequest)
	out := RetentionDiscovery{Filter: selectionRequest.SearchRequest.Filter}
	out.Selection = d.ProcessInstanceSelection{Request: selectionRequest, Origin: "search", IgnoreTenant: services.ApplyCallOptions(opts).IgnoreTenant, Completion: d.ProcessInstanceSearchCompletionIncomplete}
	if request.Limit < 0 {
		return out, fmt.Errorf("%w: retention limit must be nonnegative", d.ErrValidation)
	}
	seen := make(map[string]struct{})
	candidates := 0
	completion, err := searchpaging.Walk(ctx, api, selectionRequest.SearchRequest, false, func(page d.ProcessInstancePage, number int32) (d.ProcessInstanceSearchCompletion, error) {
		out.Selection.Pages = number
		candidates += len(page.Items)
		items := searchpaging.DistinctItems(filterRetentionDiscoveryItems(page.Items), seen)
		items = limitRetentionDiscoveryItems(items, request.Limit, len(out.Items))
		for _, item := range items {
			if item.Key == "" {
				return "", fmt.Errorf("%w: retention candidate has no key", d.ErrMalformedResponse)
			}
			out.Items = append(out.Items, item)
			out.Keys = append(out.Keys, item.Key)
		}
		limited := request.Limit > 0 && len(out.Items) >= int(request.Limit)
		reportRetentionDiscoveryProgress(request.Progress, RetentionDiscoveryProgress{Page: page, PageNumber: int(number), CurrentPageCount: len(page.Items), CurrentPageSelected: len(items), CandidatesSeen: candidates, CandidatesFrozen: len(out.Items), LimitReached: limited})
		if limited {
			return d.ProcessInstanceSearchCompletionLimitReached, nil
		}
		return "", nil
	}, opts...)
	out.Selection.Completion = completion
	out.Selection.Items, out.Selection.Keys = out.Items, out.Keys
	out.Selection.TenantEvidence = selectionTenantEvidence(out.Selection.Items)
	out.Selection = CopyProcessInstanceSelection(out.Selection)
	out.Items, out.Keys = out.Selection.Items, out.Selection.Keys
	return out, err
}

func reportRetentionDiscoveryProgress(progress func(RetentionDiscoveryProgress), event RetentionDiscoveryProgress) {
	if progress == nil {
		return
	}
	progress(event)
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
