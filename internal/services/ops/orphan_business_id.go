// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
)

// purgeBusinessIDOrphans freezes inspected candidates, classified orphan seeds and complete descendants once.
func (s *Service) purgeBusinessIDOrphans(ctx context.Context, request d.OrphanPurgeRequest, opts ...services.CallOption) (d.OrphanPurgeResult, error) {
	request.DryRun = request.DryRun || services.ApplyCallOptions(opts).DryRun
	request.Selection = pisvc.CopyProcessInstanceSelection(d.ProcessInstanceSelection{Request: cleanupSelectionRequest(request.Selection, request.BatchSize, request.Limit)}).Request.SearchRequest.Filter
	result := newOrphanPurgeResult(request)
	result.Request.FrozenPlan = nil
	fail := func(err error) (d.OrphanPurgeResult, error) {
		result.Deletion.Status = d.OpsWorkflowStepStatusFailed
		return finishOrphanPurgeResult(result, d.OrphanPurgeOutcomeFailed, err)
	}
	if err := common.ValidateBusinessIDSearch(request.Selection.BusinessId, s.version); err != nil {
		return fail(err)
	}
	if request.FrozenPlan != nil {
		if err := common.ValidateBusinessIDSearch(request.FrozenPlan.Scope.Selection.Request.SearchRequest.Filter.BusinessId, s.version); err != nil {
			return fail(err)
		}
	}
	if request.Selection.BusinessId == nil || request.DiscoveredKeys != nil || request.Selection.Key != "" {
		return fail(fmt.Errorf("%w: BID orphan purge requires search criteria and a reviewed plan instead of bare keys", d.ErrValidation))
	}
	filter := request.Selection
	filter.HasParent = new(true)
	if err := s.piAPI.ValidateProcessInstanceSearchFilter(filter); err != nil {
		return fail(err)
	}
	plan := d.OrphanPurgeFrozenPlan{OriginalSelection: request.Selection}
	var err error
	if request.FrozenPlan == nil {
		discovery, discoveryErr := pisvc.DiscoverOrphanProcessInstances(ctx, s.piAPI, pisvc.OrphanDiscoveryRequest{Filter: filter, BatchSize: request.BatchSize, Limit: request.Limit, Progress: orphanPurgeDiscoveryProgress(request)}, opts...)
		plan.Scope = d.CleanupFrozenScope{Selection: discovery.Selection, Tenant: s.piAPI.ProcessInstanceSearchTenant(opts...), Seeds: slices.Clone(discovery.Keys)}
		result.Discovery = orphanBusinessIDDiscovery(plan.Scope)
		if discoveryErr != nil {
			result.Discovery.Status = d.OpsWorkflowStepStatusFailed
			result.Discovery.Errors = []string{discoveryErr.Error()}
			return fail(discoveryErr)
		}
		plan.Scope, err = s.planCleanupFamilies(ctx, plan.Scope, false, request.Workers, opts...)
	} else {
		plan = *request.FrozenPlan
		plan.Scope = copyCleanupScope(plan.Scope)
	}
	result.Discovery = orphanBusinessIDDiscovery(plan.Scope)
	expansion := cleanupExpansion(plan.Scope)
	result.DeletionPlan = d.DeletionPlan{Status: d.OpsWorkflowStepStatusPlanned, RequestedKeys: slices.Clone(plan.Scope.Seeds), AffectedKeys: expansion.Collected, RootKeys: expansion.Roots, TenantEvidence: expansion.TenantEvidence, RequiresConfirmation: !request.DryRun && len(expansion.Roots) > 0, DryRunPreview: expansion}
	if len(plan.Scope.Seeds) == 0 {
		result.DeletionPlan.Status = d.OpsWorkflowStepStatusSkipped
	}
	if err != nil {
		return fail(err)
	}
	if !reflect.DeepEqual(plan.OriginalSelection, request.Selection) {
		return fail(fmt.Errorf("%w: orphan plan original criteria changed", d.ErrPrecondition))
	}
	if err = s.validateCleanupScope(plan.Scope, cleanupSelectionRequest(filter, request.BatchSize, request.Limit), false, opts...); err != nil {
		return fail(err)
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	plan.Scope = copyCleanupScope(plan.Scope)
	plan.OriginalSelection = result.Request.Selection
	result.FrozenPlan = &plan
	emitOpsTenantScope(request.Progress, result.DeletionPlan.TenantEvidence)
	if request.DryRun || len(plan.Scope.Families) == 0 {
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishOrphanPurgeResult(result, d.OrphanPurgeOutcomePlanned, nil)
	}
	executionOpts := compactOpsExecutionOptions(opts...)
	if request.NoWait {
		executionOpts = append(executionOpts, services.WithNoWait())
	}
	executed, err := pisvc.ExecuteProcessInstanceFrozenDeletion(ctx, s.piAPI, plan.Scope.Families, request.Workers, executionOpts...)
	result.DeleteRequested = executed.MutationSubmitted
	result.Deletion = d.DeletionResult{Status: deletionStatusForReports(executed.Roots, request.NoWait, err), Items: executed.Roots, Submitted: executed.MutationSubmitted, Confirmed: err == nil && !request.NoWait && allReportsOK(executed.Roots), NoWait: request.NoWait, Errors: deletionErrors(err)}
	if err != nil && len(executed.Roots) == 0 {
		result.Deletion.Status = d.OpsWorkflowStepStatusBlocked
	}
	return finishOrphanPurgeResult(result, deletionOutcomeForReports(executed.Roots), err)
}

// orphanBusinessIDDiscovery retains every inspected child independently of the orphan subset.
func orphanBusinessIDDiscovery(scope d.CleanupFrozenScope) d.OrphanDiscoveryResult {
	selection := pisvc.CopyProcessInstanceSelection(scope.Selection)
	return d.OrphanDiscoveryResult{Status: d.OpsWorkflowStepStatusPlanned, Filters: selection.Request.SearchRequest.Filter, Keys: slices.Clone(scope.Seeds), Count: len(scope.Seeds), Selection: &selection}
}
