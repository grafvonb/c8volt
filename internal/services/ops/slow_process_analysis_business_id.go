// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
)

// validateSlowProcessAnalysisBusinessID gates supplied criteria before keyed or empty bypasses.
func (s *Service) validateSlowProcessAnalysisBusinessID(request d.SlowProcessAnalysisRequest) error {
	bid := request.ProcessInstanceFilters.BusinessId
	if bid == nil {
		if request.SelectionMode == d.SlowProcessAnalysisSelectionModeProcessInstanceSearch {
			return fmt.Errorf("%w: process-instance search requires a business ID", d.ErrValidation)
		}
		return nil
	}
	if err := common.ValidateBusinessIDSearch(bid, s.version); err != nil {
		return err
	}
	if err := s.piAPI.ValidateProcessInstanceSearchFilter(d.ProcessInstanceFilter{BusinessId: bid}); err != nil {
		return err
	}
	if len(request.InputKeys) > 0 || request.SelectionMode == d.SlowProcessAnalysisSelectionModeExplicitKeys {
		return fmt.Errorf("%w: explicit keys cannot be combined with business-ID search", d.ErrValidation)
	}
	selector := request.ProcessDefinitionSelector
	hasDefinition := selector.BpmnProcessID != "" || selector.ProcessDefinitionKey != ""
	if (request.SelectionMode == d.SlowProcessAnalysisSelectionModeProcessInstanceSearch && hasDefinition) ||
		(request.SelectionMode == d.SlowProcessAnalysisSelectionModeProcessDefinitionSearch && !hasDefinition) {
		return fmt.Errorf("%w: slow-analysis selection mode must match its selectors", d.ErrValidation)
	}
	return nil
}

// slowProcessAnalysisDiscoverBusinessID reuses completed distinct selection and confirms only real frozen work.
func slowProcessAnalysisDiscoverBusinessID(ctx context.Context, api pisvc.API, request d.SlowProcessAnalysisRequest, opts ...services.CallOption) (slowProcessAnalysisSearchDiscovery, *d.ProcessInstanceSelection, error) {
	filter, err := slowProcessAnalysisProcessInstanceFilter(request)
	if err != nil {
		return slowProcessAnalysisSearchDiscovery{}, nil, err
	}
	batch := slowProcessAnalysisDiscoveryBatchSize(request.BatchSize)
	selected, err := pisvc.ResolveProcessInstanceSelection(ctx, api, nil, d.ProcessInstanceSelectionRequest{
		Cardinality:   d.ProcessInstanceSelectionCollection,
		SearchRequest: d.ProcessInstanceSearchRequest{Filter: filter, Page: d.ProcessInstancePageRequest{Size: batch}, Limit: request.Limit},
	}, opts...)
	discovery := slowProcessAnalysisSearchDiscovery{items: selected.Items, scope: d.DiscoveryScopeStatus{
		Complete: selected.Completion == d.ProcessInstanceSearchCompletionExhausted,
		Limited:  selected.Completion == d.ProcessInstanceSearchCompletionLimitReached,
		Limit:    request.Limit, BatchSize: batch, Pages: int(selected.Pages), CandidatesSeen: len(selected.Keys),
	}}
	if err != nil {
		return discovery, &selected, err
	}
	discovery.scope.CandidatesFrozen = len(selected.Keys)
	total := int64(len(selected.Keys))
	kind := d.ProcessInstanceReportedTotalKindExact
	if discovery.scope.Limited {
		kind = d.ProcessInstanceReportedTotalKindLowerBound
	}
	page := d.ProcessInstancePage{ReportedTotal: &d.ProcessInstanceReportedTotal{Count: total, Kind: kind}}
	preflight := slowProcessAnalysisPreflightScope(request, page, batch)
	// This scope is already frozen; page count is observed, not estimated from a seed limit.
	if len(selected.Keys) > 0 {
		pages := int64(selected.Pages)
		preflight.PageCount = &pages
		preflight.PageCountKind = d.OpsPageCountKindExact
	}
	preflight.SelectorSummary = fmt.Sprintf("business ID %q", *request.ProcessInstanceFilters.BusinessId)
	if request.SelectionMode == d.SlowProcessAnalysisSelectionModeProcessDefinitionSearch {
		preflight.SelectorSummary = fmt.Sprintf("%s; %s", slowProcessAnalysisSelectorSummary(request.ProcessDefinitionSelector), preflight.SelectorSummary)
	}
	preflight.RequiresConfirmation = len(selected.Keys) > 0
	if preflight.RequiresConfirmation {
		preflight.ConsequenceSummary.WorkSummary = "load runtime element timelines for the selected instances"
	}
	discovery.preflight = &preflight
	slowProcessAnalysisEmitProgress(request, d.OpsProgressEvent{Kind: d.OpsProgressEventKindPreflight, Preflight: &preflight})
	if request.ConfirmPreflight != nil && preflight.RequiresConfirmation {
		if err := request.ConfirmPreflight(preflight); err != nil {
			return discovery, &selected, err
		}
	}
	return discovery, &selected, nil
}

// slowProcessAnalysisEmitEnrichmentFailure keeps raw backend errors out of BID progress diagnostics.
func slowProcessAnalysisEmitEnrichmentFailure(request d.SlowProcessAnalysisRequest, progress d.OpsFrozenScopeProgress, key string, err error) {
	detail := err.Error()
	if request.ProcessInstanceFilters.BusinessId != nil {
		detail = ""
	}
	slowProcessAnalysisEmitCompletion(request, progress.Phase, progress.Total, key, d.OpsCompletionDispositionFailed, detail)
}
