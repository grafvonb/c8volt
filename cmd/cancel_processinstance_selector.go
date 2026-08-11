// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	processOptions "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

// runCancelProcessInstanceSearch owns selector/search cancellation dispatch
// after the base command has established that no explicit keys were provided.
func runCancelProcessInstanceSearch(cmd *cobra.Command, cli process.API, cfg *config.Config, log *slog.Logger) {
	if !hasPISearchFilterFlags() {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, missingDependentFlagsf("either at least one --key is required, or sufficient filtering options to search for process instances to cancel"))
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

	results, err := cancelProcessInstanceSearchPages(cmd, cli, cfg, populatePISearchFilterOpts())
	if err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("cancel process instances: %w", err))
	}
	if flagDryRun {
		if len(results.DryRunPreviews) > 0 {
			summary := newProcessInstanceDryRunSummary("cancel", results.DryRunPreviews)
			if err := renderProcessInstanceDryRunSummary(cmd, summary); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render cancel dry-run result: %w", err))
			}
		}
		return
	}

	reports := results.Reports
	if len(reports) == 0 {
		return
	}
	payload := process.CancelReports{Items: make([]process.CancelReport, len(reports))}
	for i, report := range reports {
		payload.Items[i] = process.CancelReport(report)
	}
	if err := renderCommandResult(cmd, payload); err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render cancel result: %w", err))
	}
}

// cancelProcessInstanceSearchPages handles search-selected cancel rendering and
// mutation while facade/service code owns page traversal and impact planning.
func cancelProcessInstanceSearchPages(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (processInstancePageActionResults, error) {
	firstPage := true
	var results processInstancePageActionResults
	progress, progressSeen := newProcessInstanceMutationProgressReporterWithState(cmd, "cancel")

	planned, err := cli.PlanProcessInstanceMutationPages(cmd.Context(), process.ProcessInstanceMutationPlanRequest{
		SearchRequest: newProcessInstanceSearchRequest(cmd, cfg, filter),
		Workers:       flagWorkers,
	}, func(step process.ProcessInstanceMutationPlanStep) (process.ProcessInstanceSearchPageAction, error) {
		hasSelection := len(step.RequestedKeys) > 0
		if hasSelection {
			result := processInstancePageActionResultFromPlan("cancel", step)
			printProcessInstanceMutationPlanStepFallbackProgress(cmd, "cancel", step, progressSeen)
			if flagDryRun {
				if result.DryRunPreview != nil {
					results.DryRunPreviews = append(results.DryRunPreviews, *result.DryRunPreview)
				}
			} else {
				printDryRunExpansionWarning(cmd, step.Plan)
				impact := result.Impact
				if firstPage {
					affectedCount, rootCount, requestedCount := impact.Affected, impact.Roots, impact.Requested
					prompt := fmt.Sprintf("You are about to cancel %d process instance(s). Do you want to proceed?", affectedCount)
					if affectedCount > requestedCount {
						prompt = fmt.Sprintf("You have requested to cancel %d process instance(s), but due to dependencies, a total of %d instance(s) with %d root instance(s) will be canceled. Do you want to proceed?", requestedCount, affectedCount, rootCount)
					}
					if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
						return process.ProcessInstanceSearchPageActionStop, err
					}
				}

				mutationOpts := append(compactProcessInstanceMutationOptions(collectOptions()),
					processOptions.WithAffectedProcessInstanceCount(len(step.Plan.Collected)),
					processOptions.WithProgress(progress),
				)
				reports, err := cli.CancelProcessInstances(cmd.Context(), step.Plan.Roots, flagWorkers, mutationOpts...)
				if err != nil {
					return process.ProcessInstanceSearchPageActionStop, fmt.Errorf("cancel process instances: %w", err)
				}
				for _, report := range reports.Items {
					results.Reports = append(results.Reports, process.Reporter(report))
				}
				if result.DryRunPreview != nil {
					results.DryRunPreviews = append(results.DryRunPreviews, *result.DryRunPreview)
				}
			}
		}

		summary := newPIProgressSummary(step.Page, int(step.CumulativeCount), flagDryRun || shouldAutoContinuePISearchPages(cmd))
		printPISearchProgress(cmd, summary)

		switch summary.ContinuationState {
		case processInstanceContinuationCompleted, processInstanceContinuationWarningStop, processInstanceContinuationLimitReached:
			return process.ProcessInstanceSearchPageActionStop, nil
		case processInstanceContinuationAutoContinue:
			if hasSelection {
				firstPage = false
			}
			return process.ProcessInstanceSearchPageActionContinue, nil
		case processInstanceContinuationPrompt:
			prompt := fmt.Sprintf("Processed %d process instance(s) on this page (%s, %d including dependencies). More matching process instances remain. Continue?", summary.CurrentPageCount, formatProcessInstancePagingProgress(step.Page, summary.CumulativeCount, "requested"), step.CumulativeImpact)
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
			if hasSelection {
				firstPage = false
			}
			return process.ProcessInstanceSearchPageActionContinue, nil
		}
		return process.ProcessInstanceSearchPageActionStop, nil
	}, append(collectOptions(), processOptions.WithProgress(progress))...)
	if err != nil {
		return processInstancePageActionResults{}, err
	}
	if len(results.Reports) > 0 {
		renderProcessInstanceMutationResultSummary(cmd, "cancel", results.Reports, processInstancePageImpact{
			Requested: int(planned.RequestedCount),
			Affected:  int(planned.CumulativeImpact),
			Roots:     len(results.Reports),
		})
	}
	if planned.RequestedCount == 0 {
		renderOutputLine(cmd, "found: %d", 0)
	}
	return results, nil
}
