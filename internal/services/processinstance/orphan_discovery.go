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

type OrphanDiscoveryAPI interface {
	SearchForProcessInstancesPage(ctx context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error)
	FilterProcessInstanceWithOrphanParent(ctx context.Context, items []d.ProcessInstance, opts ...services.CallOption) ([]d.ProcessInstance, error)
}

type OrphanDiscoveryRequest struct {
	Filter    d.ProcessInstanceFilter
	BatchSize int32
	Limit     int32
	Progress  func(OrphanDiscoveryProgress)
}

type OrphanDiscoveryProgress struct {
	Page                  int
	Phase                 string
	CurrentPageCandidates int
	CurrentPageOrphans    int
	CandidatesChecked     int
	OrphansFound          int
	Limit                 int32
	OverflowState         d.ProcessInstanceOverflowState
}

// OrphanDiscovery distinguishes inspected candidates from confirmed orphan seeds.
type OrphanDiscovery struct {
	Selection d.ProcessInstanceSelection
	Filter    d.ProcessInstanceFilter
	Items     []d.ProcessInstance
	Keys      typex.Keys
}

// DiscoverOrphanProcessInstances counts distinct inspected children before classifying their parent, including non-orphans.
func DiscoverOrphanProcessInstances(ctx context.Context, api OrphanDiscoveryAPI, request OrphanDiscoveryRequest, opts ...services.CallOption) (OrphanDiscovery, error) {
	filter := request.Filter
	filter.HasParent = new(true)
	selectionRequest := copyProcessInstanceSelectionRequest(d.ProcessInstanceSelectionRequest{Cardinality: d.ProcessInstanceSelectionCollection, SearchRequest: d.ProcessInstanceSearchRequest{Filter: filter, Page: d.ProcessInstancePageRequest{Size: normalizeOrphanDiscoveryBatchSize(request.BatchSize)}, Limit: request.Limit}})
	out := OrphanDiscovery{Filter: selectionRequest.SearchRequest.Filter, Selection: d.ProcessInstanceSelection{Request: selectionRequest, Origin: "search", IgnoreTenant: services.ApplyCallOptions(opts).IgnoreTenant, Completion: d.ProcessInstanceSearchCompletionIncomplete}}
	if request.Limit < 0 {
		return out, fmt.Errorf("%w: orphan candidate limit must be nonnegative", d.ErrValidation)
	}
	seen := make(map[string]struct{})
	completion, err := searchpaging.Walk(ctx, api, selectionRequest.SearchRequest, false, func(page d.ProcessInstancePage, number int32) (d.ProcessInstanceSearchCompletion, error) {
		out.Selection.Pages = number
		items := searchpaging.DistinctItems(page.Items, seen)
		items = limitOrphanDiscoveryItems(items, request.Limit, len(out.Selection.Keys))
		for _, item := range items {
			if item.Key == "" {
				return "", fmt.Errorf("%w: orphan candidate has no key", d.ErrMalformedResponse)
			}
		}
		if len(items) > 0 {
			event := OrphanDiscoveryProgress{Page: int(number), Phase: "checking", CurrentPageCandidates: len(items), CandidatesChecked: len(out.Selection.Keys), OrphansFound: len(out.Keys), Limit: request.Limit, OverflowState: page.OverflowState}
			reportOrphanDiscoveryProgress(request.Progress, event)
			out.Selection.Items = append(out.Selection.Items, items...)
			for _, item := range items {
				out.Selection.Keys = append(out.Selection.Keys, item.Key)
			}
			orphans, err := api.FilterProcessInstanceWithOrphanParent(ctx, items, opts...)
			if err != nil {
				return "", err
			}
			candidates := make(map[string]d.ProcessInstance, len(items))
			for _, item := range items {
				candidates[item.Key] = item
			}
			orphanKeys := make(map[string]bool)
			for _, item := range orphans {
				candidate, ok := candidates[item.Key]
				if !ok || candidate.ParentKey == "" || orphanKeys[item.Key] {
					return "", fmt.Errorf("%w: invalid orphan classification", d.ErrMalformedResponse)
				}
				orphanKeys[item.Key] = true
				out.Items = append(out.Items, candidate)
				out.Keys = append(out.Keys, candidate.Key)
			}
			event.Phase, event.CurrentPageOrphans, event.CandidatesChecked, event.OrphansFound = "checked", len(orphans), len(out.Selection.Keys), len(out.Keys)
			reportOrphanDiscoveryProgress(request.Progress, event)
		}
		if request.Limit > 0 && len(out.Selection.Keys) >= int(request.Limit) {
			return d.ProcessInstanceSearchCompletionLimitReached, nil
		}
		return "", nil
	}, opts...)
	out.Selection.Completion = completion
	out.Selection.TenantEvidence = selectionTenantEvidence(out.Selection.Items)
	out.Selection = CopyProcessInstanceSelection(out.Selection)
	out.Items = CopyProcessInstanceSelection(d.ProcessInstanceSelection{Items: out.Items}).Items
	return out, err
}

func reportOrphanDiscoveryProgress(progress func(OrphanDiscoveryProgress), event OrphanDiscoveryProgress) {
	if progress == nil {
		return
	}
	progress(event)
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
