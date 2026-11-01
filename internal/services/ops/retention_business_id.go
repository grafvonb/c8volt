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

// executeBusinessIDRetention freezes the fixed boundary and eligible families before exact deletion.
func (s *Service) executeBusinessIDRetention(ctx context.Context, request d.RetentionPolicyRequest, opts ...services.CallOption) (d.RetentionPolicyResult, error) {
	request.DryRun = request.DryRun || services.ApplyCallOptions(opts).DryRun
	request.Selection = pisvc.CopyProcessInstanceSelection(d.ProcessInstanceSelection{Request: cleanupSelectionRequest(request.Selection, request.BatchSize, request.Limit)}).Request.SearchRequest.Filter
	result := newRetentionPolicyResult(request)
	result.Request.FrozenPlan = nil
	fail := func(err error) (d.RetentionPolicyResult, error) {
		result.Deletion.Status = d.OpsWorkflowStepStatusFailed
		return finishRetentionPolicyResult(result, d.RetentionPolicyOutcomeFailed, err)
	}
	if err := common.ValidateBusinessIDSearch(request.Selection.BusinessId, s.version); err != nil {
		return fail(err)
	}
	if request.FrozenPlan != nil {
		if err := common.ValidateBusinessIDSearch(request.FrozenPlan.Scope.Selection.Request.SearchRequest.Filter.BusinessId, s.version); err != nil {
			return fail(err)
		}
	}
	if request.Selection.BusinessId == nil || request.DiscoveredKeys != nil {
		return fail(fmt.Errorf("%w: BID retention requires a reviewed plan instead of bare keys", d.ErrValidation))
	}
	if err := validateRetentionPolicyRequest(request); err != nil {
		return fail(err)
	}
	filter := request.Selection
	filter.EndDateBefore = request.DerivedEndDateBoundary
	if err := s.piAPI.ValidateProcessInstanceSearchFilter(filter); err != nil {
		return fail(err)
	}
	plan := d.RetentionFrozenPlan{OriginalSelection: request.Selection, RetentionDays: request.RetentionDays, DerivedEndDateBoundary: request.DerivedEndDateBoundary}
	var err error
	if request.FrozenPlan == nil {
		discovery, discoveryErr := pisvc.DiscoverRetentionProcessInstances(ctx, s.piAPI, pisvc.RetentionDiscoveryRequest{Filter: filter, BatchSize: request.BatchSize, Limit: request.Limit, Progress: retentionPolicyDiscoveryProgress(request)}, opts...)
		plan.Scope = d.CleanupFrozenScope{Selection: discovery.Selection, Tenant: s.piAPI.ProcessInstanceSearchTenant(opts...), Seeds: slices.Clone(discovery.Keys)}
		result.Discovery = retentionBusinessIDDiscovery(request, plan.Scope)
		if discoveryErr != nil {
			result.Discovery.Status = d.OpsWorkflowStepStatusFailed
			result.Discovery.Errors = []string{discoveryErr.Error()}
			return fail(discoveryErr)
		}
		plan.Scope, err = s.planCleanupFamilies(ctx, plan.Scope, true, request.Workers, opts...)
	} else {
		plan = *request.FrozenPlan
		plan.Scope = copyCleanupScope(plan.Scope)
	}
	result.Discovery = retentionBusinessIDDiscovery(request, plan.Scope)
	result.DeletePlan = retentionBusinessIDDeletePlan(plan.Scope, !request.DryRun)
	if err != nil {
		return fail(err)
	}
	if !reflect.DeepEqual(plan.OriginalSelection, request.Selection) || plan.RetentionDays != request.RetentionDays || plan.DerivedEndDateBoundary != request.DerivedEndDateBoundary {
		return fail(fmt.Errorf("%w: retention plan boundary or original criteria changed", d.ErrPrecondition))
	}
	if err = s.validateCleanupScope(plan.Scope, cleanupSelectionRequest(filter, request.BatchSize, request.Limit), true, opts...); err != nil {
		return fail(err)
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	plan.Scope = copyCleanupScope(plan.Scope)
	plan.OriginalSelection = result.Request.Selection
	result.FrozenPlan = &plan
	emitOpsTenantScope(request.Progress, result.DeletePlan.TenantEvidence)
	if request.DryRun || len(plan.Scope.Families) == 0 {
		result.Deletion.Status = d.OpsWorkflowStepStatusSkipped
		return finishRetentionPolicyResult(result, d.RetentionPolicyOutcomePlanned, nil)
	}
	executionOpts := compactOpsExecutionOptions(opts...)
	if request.Force {
		executionOpts = append(executionOpts, services.WithForce())
	}
	if request.NoWait {
		executionOpts = append(executionOpts, services.WithNoWait())
	}
	if request.NoStateCheck {
		executionOpts = append(executionOpts, services.WithNoStateCheck())
	}
	if request.FailFast {
		executionOpts = append(executionOpts, services.WithFailFast())
	}
	if request.NoWorkerLimit {
		executionOpts = append(executionOpts, services.WithNoWorkerLimit())
	}
	executed, err := pisvc.ExecuteProcessInstanceFrozenDeletion(ctx, s.piAPI, plan.Scope.Families, request.Workers, executionOpts...)
	result.Deletion = d.RetentionDeletionResult{Status: deletionStatusForReports(executed.Roots, request.NoWait, err), Items: executed.Roots, Submitted: executed.MutationSubmitted, Confirmed: err == nil && !request.NoWait && allReportsOK(executed.Roots), NoWait: request.NoWait, Errors: deletionErrors(err)}
	result.Deletion.SubmittedRootKeys = slices.Clone(executed.SubmittedRootKeys)
	if err != nil && len(executed.Roots) == 0 {
		result.Deletion.Status = d.OpsWorkflowStepStatusBlocked
	}
	return finishRetentionPolicyResult(result, retentionDeletionOutcomeForReports(executed.Roots), err)
}

// retentionBusinessIDDiscovery preserves original matched observations and bounded completion in reports.
func retentionBusinessIDDiscovery(request d.RetentionPolicyRequest, scope d.CleanupFrozenScope) d.RetentionDiscoveryResult {
	selection := pisvc.CopyProcessInstanceSelection(scope.Selection)
	return d.RetentionDiscoveryResult{Status: d.OpsWorkflowStepStatusPlanned, RetentionDays: request.RetentionDays, DerivedEndDateBoundary: request.DerivedEndDateBoundary, Filters: selection.Request.SearchRequest.Filter, SeedKeys: slices.Clone(scope.Seeds), Count: len(scope.Seeds), Selection: &selection}
}

// retentionBusinessIDDeletePlan derives existing skip/blocker fields from immutable ancestry and membership.
func retentionBusinessIDDeletePlan(scope d.CleanupFrozenScope, confirmation bool) d.RetentionDeletePlan {
	expansion := cleanupExpansion(scope)
	plan := d.RetentionDeletePlan{Status: d.OpsWorkflowStepStatusPlanned, SeedKeys: slices.Clone(scope.Seeds), ResolvedRootKeys: expansion.Roots, AffectedKeys: expansion.Collected, TenantEvidence: expansion.TenantEvidence, FinalStateItems: expansion.SelectedFinalState, NonFinalAffectedItems: expansion.RequiresCancelBeforeDelete, RequiresConfirmation: confirmation && len(expansion.Roots) > 0}
	seen := map[string]bool{}
	for _, path := range scope.Ancestries {
		if len(path.Items) == 0 || len(path.Keys) == 0 {
			continue
		}
		root := path.Items[0]
		if !root.State.IsTerminal() {
			plan.SkippedSeedKeys = append(plan.SkippedSeedKeys, path.Keys[len(path.Keys)-1])
			plan.SkippedNonFinalRoots = appendUniqueProcessInstancesByKey(plan.SkippedNonFinalRoots, root)
		} else if seen[root.Key] {
			plan.DuplicateKeys = append(plan.DuplicateKeys, root.Key)
		}
		seen[root.Key] = true
	}
	if len(scope.Seeds) == 0 {
		plan.Status = d.OpsWorkflowStepStatusSkipped
	}
	plan.DuplicateKeys = plan.DuplicateKeys.Unique()
	return plan
}
