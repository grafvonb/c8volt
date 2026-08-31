// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	processOptions "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/config"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// runDeleteProcessInstanceSearch owns selector/search deletion dispatch after
// the base command has established that no explicit keys were provided.
func runDeleteProcessInstanceSearch(cmd *cobra.Command, cli process.API, cfg *config.Config, log *slog.Logger) {
	if !hasPISearchFilterFlags() {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, missingDependentFlagsf("either at least one --key is required, or sufficient filtering options to search for process instances to delete"))
	}
	if err := validatePISearchVersionSupport(cfg); err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
	}
	if flagGetPIBpmnProcessID != "" {
		result, err := validateProcessDefinitionSelectorsForCommand(cmd.Context(), cmd, cli, newPIProcessDefinitionSelectorValidationRequest(), collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if !result.Valid() {
			handleProcessDefinitionSelectorValidationError(cmd, log, cfg.App.NoErrCodes, cli, result)
		}
	}

	results, err := deleteProcessInstanceSearchPages(cmd, cli, cfg, populatePISearchFilterOpts())
	if err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("delete process instances: %w", err))
	}
	if flagDryRun {
		if len(results.DryRunPreviews) > 0 {
			summary := newProcessInstanceDryRunSummary("delete", results.DryRunPreviews)
			if err := renderProcessInstanceDryRunSummary(cmd, summary); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render delete dry-run result: %w", err))
			}
		}
		return
	}

	reports := results.Reports
	if len(reports) == 0 {
		return
	}
	payload := process.DeleteReports{Items: make([]process.DeleteReport, len(reports))}
	for i, report := range reports {
		payload.Items[i] = process.DeleteReport(report)
	}
	if err := renderCommandResult(cmd, payload); err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render delete result: %w", err))
	}
}

// deleteProcessInstanceSearchPages freezes search-selected delete scope before
// submitting one aggregate mutation, preserving confirmation semantics.
func deleteProcessInstanceSearchPages(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (processInstancePageActionResults, error) {
	if flagDryRun {
		return planDeleteProcessInstanceSearchPages(cmd, cli, cfg, filter)
	}

	results, aborted, err := planDeleteProcessInstanceSearchPagesForMutation(cmd, cli, cfg, filter)
	if err != nil {
		return processInstancePageActionResults{}, err
	}
	if aborted || len(results.DryRunPreviews) == 0 {
		return results, nil
	}
	tenantCtx := attachProcessInstanceDiscoveryTenantContext(cmd, attachDiscoveryTenantContext(cmd, cfg), results.TenantEvidence)
	plan := aggregateDeleteSearchPlan(results.DryRunPreviews, results.TenantEvidence)
	renderDeleteSearchTenantContext(cmd, tenantCtx)
	printDryRunExpansionWarning(cmd, plan)
	if err := rejectDeletePlanRequiringForce(plan); err != nil {
		return processInstancePageActionResults{}, err
	}
	impact := processInstancePageImpact{
		Requested: searchDryRunRequestedCount(results.DryRunPreviews),
		Affected:  len(plan.Collected),
		Roots:     len(plan.Roots),
	}
	prompt := fmt.Sprintf("You are about to delete %d process instance(s). Do you want to proceed?", impact.Affected)
	if impact.Affected > impact.Requested {
		prompt = fmt.Sprintf("You have requested to delete %d process instance(s), but due to dependencies, a total of %d instance(s) with %d root instance(s) will be deleted. Do you want to proceed?", impact.Requested, impact.Affected, impact.Roots)
	}
	if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
		return processInstancePageActionResults{}, err
	}

	semanticReporter := newProcessInstanceMutationSemanticReporter(cmd, "delete", impact)
	defer semanticReporter.Close()
	opts := append(compactProcessInstanceMutationOptions(collectOptions()),
		processOptions.WithAffectedProcessInstanceCount(len(plan.Collected)),
		processOptions.WithProgress(processInstanceMutationSemanticProgressCallback(semanticReporter)),
	)
	reports, err := cli.DeleteProcessInstances(cmd.Context(), plan.Roots, flagWorkers, opts...)
	if err != nil {
		return processInstancePageActionResults{}, fmt.Errorf("delete process instances: %w", err)
	}
	results.Reports = make([]process.Reporter, len(reports.Items))
	for i, report := range reports.Items {
		results.Reports[i] = process.Reporter(report)
	}
	renderProcessInstanceMutationResultSummary(cmd, "delete", results.Reports, impact)
	return results, nil
}

// planDeleteProcessInstanceSearchPages records delete previews without mutating.
func planDeleteProcessInstanceSearchPages(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (processInstancePageActionResults, error) {
	var results processInstancePageActionResults
	progress, progressSeen := newProcessInstanceMutationProgressReporterWithState(cmd, "delete")
	tenantCtx := attachDiscoveryTenantContext(cmd, cfg)
	tenantContextRendered := false
	renderDiscoveryTenantContext := func(evidence process.TenantEvidence) {
		if tenantContextRendered {
			return
		}
		renderDeleteSearchTenantContext(cmd, attachProcessInstanceDiscoveryTenantContext(cmd, tenantCtx, evidence))
		tenantContextRendered = true
	}

	planned, err := cli.PlanProcessInstanceMutationPages(cmd.Context(), process.ProcessInstanceMutationPlanRequest{
		SearchRequest: newProcessInstanceSearchRequest(cmd, cfg, filter),
		Workers:       flagWorkers,
	}, func(step process.ProcessInstanceMutationPlanStep) (process.ProcessInstanceSearchPageAction, error) {
		if len(step.RequestedKeys) > 0 {
			if !flagDryRun {
				renderDiscoveryTenantContext(step.Plan.TenantEvidence)
			}
			result := processInstancePageActionResultFromPlan("delete", step)
			printProcessInstanceMutationPlanStepFallbackProgress(cmd, "delete", step, progressSeen)
			if result.DryRunPreview != nil {
				results.DryRunPreviews = append(results.DryRunPreviews, *result.DryRunPreview)
			}
		}
		summary := newPIProgressSummary(step.Page, int(step.CumulativeCount), flagDryRun || shouldAutoContinuePISearchPages(cmd))
		printPISearchProgress(cmd, summary)

		switch summary.ContinuationState {
		case processInstanceContinuationCompleted, processInstanceContinuationWarningStop, processInstanceContinuationLimitReached:
			return process.ProcessInstanceSearchPageActionStop, nil
		case processInstanceContinuationAutoContinue:
			return process.ProcessInstanceSearchPageActionContinue, nil
		case processInstanceContinuationPrompt:
			prompt := fmt.Sprintf("Checked delete impact for %d process instance(s) on this page (%s, %d including dependencies); no changes made yet. More matching process instances remain. Continue checking?", summary.CurrentPageCount, formatProcessInstancePagingProgress(step.Page, summary.CumulativeCount, "requested"), step.CumulativeImpact)
			if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
				if isCmdAborted(err) {
					printPISearchProgress(cmd, processInstanceProgressSummary{
						PageSize:          summary.PageSize,
						CurrentPageCount:  summary.CurrentPageCount,
						CumulativeCount:   summary.CumulativeCount,
						OverflowState:     summary.OverflowState,
						ContinuationState: processInstanceContinuationPartialComplete,
					})
					return process.ProcessInstanceSearchPageActionStop, nil
				}
				return process.ProcessInstanceSearchPageActionStop, err
			}
			return process.ProcessInstanceSearchPageActionContinue, nil
		}
		return process.ProcessInstanceSearchPageActionStop, nil
	}, append(collectOptions(), processOptions.WithProgress(progress))...)
	if err != nil {
		return processInstancePageActionResults{}, err
	}
	if flagDryRun && planned.RequestedCount > 0 {
		attachTenantContext(cmd, withTenantContextEvidence(tenantCtx, planned.TenantEvidence.ResolvedTenantIDs, planned.TenantEvidence.UnknownTargetCount))
	}
	results.TenantEvidence = planned.TenantEvidence
	if planned.RequestedCount == 0 {
		renderOutputLine(cmd, "found: %d", 0)
	}
	return results, nil
}

// renderDeleteSearchTenantContext keeps preview scope visible while preserving
// the destructive search progress contract that reserves stdout for results.
func renderDeleteSearchTenantContext(cmd *cobra.Command, ctx tenant.Context) {
	if flagDryRun {
		renderTenantContext(cmd, ctx)
		return
	}
	renderProcessInstanceMutationTenantContextStderr(cmd, ctx)
}

// planDeleteProcessInstanceSearchPagesForMutation records every selected
// delete preview before the command performs confirmation and mutation.
func planDeleteProcessInstanceSearchPagesForMutation(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (processInstancePageActionResults, bool, error) {
	aborted := false
	results, err := planDeleteProcessInstanceSearchPagesWithPrompt(cmd, cli, cfg, filter, &aborted)
	return results, aborted, err
}

// planDeleteProcessInstanceSearchPagesWithPrompt freezes page-level delete
// previews and records when the operator stops before aggregate mutation.
func planDeleteProcessInstanceSearchPagesWithPrompt(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter, aborted *bool) (processInstancePageActionResults, error) {
	var results processInstancePageActionResults
	progress, progressSeen := newProcessInstanceMutationProgressReporterWithState(cmd, "delete")

	planned, err := cli.PlanProcessInstanceMutationPages(cmd.Context(), process.ProcessInstanceMutationPlanRequest{
		SearchRequest: newProcessInstanceSearchRequest(cmd, cfg, filter),
		Workers:       flagWorkers,
	}, func(step process.ProcessInstanceMutationPlanStep) (process.ProcessInstanceSearchPageAction, error) {
		if len(step.RequestedKeys) > 0 {
			result := processInstancePageActionResultFromPlan("delete", step)
			printProcessInstanceMutationPlanStepFallbackProgress(cmd, "delete", step, progressSeen)
			if result.DryRunPreview != nil {
				results.DryRunPreviews = append(results.DryRunPreviews, *result.DryRunPreview)
			}
		}
		summary := newPIProgressSummary(step.Page, int(step.CumulativeCount), shouldAutoContinuePISearchPages(cmd))
		printPISearchProgress(cmd, summary)

		switch summary.ContinuationState {
		case processInstanceContinuationCompleted, processInstanceContinuationWarningStop, processInstanceContinuationLimitReached:
			return process.ProcessInstanceSearchPageActionStop, nil
		case processInstanceContinuationAutoContinue:
			return process.ProcessInstanceSearchPageActionContinue, nil
		case processInstanceContinuationPrompt:
			prompt := fmt.Sprintf("Checked delete impact for %d process instance(s) on this page (%s, %d including dependencies); no changes made yet. More matching process instances remain. Continue checking?", summary.CurrentPageCount, formatProcessInstancePagingProgress(step.Page, summary.CumulativeCount, "requested"), step.CumulativeImpact)
			if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
				if isCmdAborted(err) {
					printPISearchProgress(cmd, processInstanceProgressSummary{
						PageSize:          summary.PageSize,
						CurrentPageCount:  summary.CurrentPageCount,
						CumulativeCount:   summary.CumulativeCount,
						OverflowState:     summary.OverflowState,
						ContinuationState: processInstanceContinuationPartialComplete,
					})
					if aborted != nil {
						*aborted = true
					}
					return process.ProcessInstanceSearchPageActionStop, nil
				}
				return process.ProcessInstanceSearchPageActionStop, err
			}
			return process.ProcessInstanceSearchPageActionContinue, nil
		}
		return process.ProcessInstanceSearchPageActionStop, nil
	}, append(collectOptions(), processOptions.WithProgress(progress))...)
	if err != nil {
		return processInstancePageActionResults{}, err
	}
	results.TenantEvidence = planned.TenantEvidence
	if planned.RequestedCount == 0 {
		renderOutputLine(cmd, "found: %d", 0)
	}
	return results, nil
}

// aggregateDeleteSearchPlan merges page-level delete previews into one frozen
// mutation plan while preserving the service-owned tenant evidence snapshot.
func aggregateDeleteSearchPlan(previews []processInstanceDryRunPreview, evidence process.TenantEvidence) process.DryRunPIKeyExpansion {
	var roots types.Keys
	var collected types.Keys
	var requiresCancel []process.ProcessInstance
	var missing []process.MissingAncestor
	outcome := process.TraversalOutcomeComplete
	var warning string

	for _, preview := range previews {
		roots = append(roots, preview.ResolvedRoots...)
		collected = append(collected, preview.AffectedFamilyKeys...)
		for _, item := range preview.RequiresCancelBeforeDelete {
			requiresCancel = append(requiresCancel, process.ProcessInstance{Key: item.Key, State: item.State})
		}
		for _, item := range preview.MissingAncestors {
			missing = append(missing, process.MissingAncestor{Key: item.Key, StartKey: item.StartKey})
		}
		if preview.Warning != "" && warning == "" {
			warning = preview.Warning
		}
		if preview.TraversalOutcome == process.TraversalOutcomePartial {
			outcome = process.TraversalOutcomePartial
		}
	}

	requiresCancel = uniqueProcessInstancesByKey(requiresCancel)
	missing = uniqueMissingAncestorsByKey(missing)
	if warning == "" && len(missing) > 0 {
		warning = "one or more parent process instances were not found"
	}
	return process.DryRunPIKeyExpansion{
		Roots:                      roots.Unique(),
		Collected:                  collected.Unique(),
		TenantEvidence:             evidence,
		RequiresCancelBeforeDelete: requiresCancel,
		MissingAncestors:           missing,
		Warning:                    warning,
		Outcome:                    outcome,
	}
}

// searchDryRunRequestedCount totals the selected process-instance count across search previews.
func searchDryRunRequestedCount(previews []processInstanceDryRunPreview) int {
	total := 0
	for _, preview := range previews {
		total += preview.RequestedCount
	}
	return total
}

// uniqueProcessInstancesByKey preserves the first process-instance entry for each key.
func uniqueProcessInstancesByKey(items []process.ProcessInstance) []process.ProcessInstance {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]process.ProcessInstance, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item.Key]; ok {
			continue
		}
		seen[item.Key] = struct{}{}
		out = append(out, item)
	}
	return out
}

// uniqueMissingAncestorsByKey preserves first-seen missing ancestor entries by start/key pair.
func uniqueMissingAncestorsByKey(items []process.MissingAncestor) []process.MissingAncestor {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]process.MissingAncestor, 0, len(items))
	for _, item := range items {
		key := item.StartKey + ":" + item.Key
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}
