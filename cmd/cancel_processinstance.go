// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	processOptions "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

var (
	flagCancelPIKeys []string
)

var cancelProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Cancel process instances by key or filters",
	Long: "Cancel process instances by key or search filters.\n\n" +
		"By default c8volt validates the affected root and descendant instances, asks for confirmation, and waits until cancellation is observed. Use --force when a selected child must be escalated to its root instance.\n\n" +
		"Tenant contract: --tenant scopes search-derived candidate discovery where supported. Explicit --key and stdin keys are backend-authorized admin input; existing dry-run, confirmation, force, and wait safety checks still apply.\n\n" +
		"When --bpmn-process-id is set, c8volt validates that the process definition is visible before searching process instances. A missing selector fails with a local diagnostic before paging, dry-run planning, confirmation, or cancellation; --json, --automation, and non-TTY runs never prompt for recovery output. If the selector is visible but no matching instances are found, no cancellation request is submitted.\n\n" +
		"Search mode pages through matching process instances by default. --batch-size controls each discovery page request, --limit caps the selected process-instance scope across all pages, and --workers, --fail-fast, and --no-worker-limit bound independent planning or cancellation work. Verbose paging progress is written away from stdout; JSON, quiet, and automation output remain free of prompts unless confirmation is explicitly supplied.\n\n" +
		"Use --dry-run to preview selected, in-scope, final-state, and partial-scope instances without cancelling.\n\n" +
		"Use --auto-confirm for unattended destructive runs.",
	Example: `  ./c8volt cancel process-instance --key <process-instance-key>
  ./c8volt cancel process-instance --key <process-instance-key> --dry-run
  ./c8volt cancel process-instance --key <process-instance-key> --force
  ./c8volt cancel process-instance --state active --batch-size 250 --limit 5 --dry-run
  ./c8volt cancel process-instance --state active --start-date-before 2026-05-31 --limit 5 --dry-run
  ./c8volt cancel process-instance --state active --start-date-newer-days 30 --limit 5 --dry-run
  ./c8volt cancel process-instance --bpmn-process-id <bpmn-process-id> --state active --limit 5 --auto-confirm
  ./c8volt expect process-instance --key <process-instance-key> --state canceled
  ./c8volt get process-instance --key <process-instance-key> --keys-only | ./c8volt cancel process-instance --auto-confirm -`,
	Aliases: []string{"pi"},
	Args: func(cmd *cobra.Command, args []string) error {
		return validateOptionalDashArg(args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if cmd.Flags().Changed("workers") && flagWorkers < 1 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("--workers must be positive integer"))
		}
		if err := validatePISearchFlags(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}

		stdinKeys, err := readKeysIfDash(args) // only reads when args == []{"-"}
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(flagCancelPIKeys, stdinKeys, log, cfg).Unique()
		if err := validatePIKeyedModeDateFilters(len(keys)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if err := validatePIKeyedModeLimit(len(keys)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		searched := false

		switch {
		case len(keys) > 0:
		default:
			searched = true
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
			searchFilterOpts := populatePISearchFilterOpts()
			results, err := cancelProcessInstanceSearchPages(cmd, cli, cfg, searchFilterOpts)
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
			if len(reports) > 0 {
				payload := process.CancelReports{Items: make([]process.CancelReport, len(reports))}
				for i, report := range reports {
					payload.Items[i] = process.CancelReport(report)
				}
				if err := renderCommandResult(cmd, payload); err != nil {
					handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render cancel result: %w", err))
				}
			}
			return
		}
		if len(keys) == 0 {
			if searched {
				renderOutputLine(cmd, "found: %d", 0)
				return
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no process instance keys provided or found to cancel")))
		}
		result, err := cancelProcessInstancesWithPlan(cmd, cli, keys, true)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if flagDryRun {
			return
		}
		payload := process.CancelReports{Items: make([]process.CancelReport, len(result.Reports))}
		for i, report := range result.Reports {
			payload.Items[i] = process.CancelReport(report)
		}
		if err := renderCommandResult(cmd, payload); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render cancel result: %w", err))
		}
		return
	},
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

				mutationOpts := append(collectOptions(),
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
	if planned.RequestedCount == 0 {
		renderOutputLine(cmd, "found: %d", 0)
	}
	return results, nil
}

// cancelProcessInstancesWithPlan validates the cancel scope, renders dry-run
// output when requested, and submits the mutation otherwise.
func cancelProcessInstancesWithPlan(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool) (processInstancePageActionResult, error) {
	return cancelProcessInstancesWithPlanAndRenderWithOptions(cmd, cli, keys, firstPage, true, collectExplicitPIAdminInputOptions())
}

// cancelProcessInstancesWithPlanAndRender shares cancel planning for keyed and
// paged flows while allowing callers to defer dry-run rendering.
func cancelProcessInstancesWithPlanAndRender(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool, renderDryRun bool) (processInstancePageActionResult, error) {
	return cancelProcessInstancesWithPlanAndRenderWithOptions(cmd, cli, keys, firstPage, renderDryRun, collectOptions())
}

// cancelProcessInstancesWithPlanAndRenderWithOptions keeps direct-key admin
// input separate from tenant-scoped search-derived candidates.
func cancelProcessInstancesWithPlanAndRenderWithOptions(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool, renderDryRun bool, opts []processOptions.FacadeOption) (processInstancePageActionResult, error) {
	planned, err := planProcessInstanceDryRunPreviewWithOptions(cmd, cli, "cancel", keys, opts)
	if err != nil {
		return processInstancePageActionResult{}, err
	}
	plan := planned.Plan
	if flagDryRun {
		if renderDryRun {
			if err := renderProcessInstanceDryRunPreview(cmd, planned.Preview); err != nil {
				return processInstancePageActionResult{}, fmt.Errorf("render cancel dry-run result: %w", err)
			}
		}
		return processInstancePageActionResult{
			Impact:        planned.Impact,
			DryRunPreview: &planned.Preview,
		}, nil
	}
	printDryRunExpansionWarning(cmd, plan)

	impact := planned.Impact
	if firstPage {
		affectedCount, rootCount, requestedCount := impact.Affected, impact.Roots, impact.Requested
		prompt := fmt.Sprintf("You are about to cancel %d process instance(s). Do you want to proceed?", affectedCount)
		if affectedCount > requestedCount {
			prompt = fmt.Sprintf("You have requested to cancel %d process instance(s), but due to dependencies, a total of %d instance(s) with %d root instance(s) will be canceled. Do you want to proceed?", requestedCount, affectedCount, rootCount)
		}
		if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
			return processInstancePageActionResult{}, err
		}
	}

	mutationOpts := append(opts, processOptions.WithAffectedProcessInstanceCount(len(plan.Collected)))
	reports, err := cli.CancelProcessInstances(cmd.Context(), plan.Roots, flagWorkers, mutationOpts...)
	if err != nil {
		return processInstancePageActionResult{}, fmt.Errorf("cancel process instances: %w", err)
	}
	result := processInstancePageActionResult{
		Impact:        impact,
		Reports:       make([]process.Reporter, len(reports.Items)),
		DryRunPreview: &planned.Preview,
	}
	for i, report := range reports.Items {
		result.Reports[i] = process.Reporter(report)
	}
	return result, nil
}

func cancelProcessInstancePage(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool) (processInstancePageActionResult, error) {
	return cancelProcessInstancesWithPlan(cmd, cli, keys, firstPage)
}

func init() {
	cancelCmd.AddCommand(cancelProcessInstanceCmd)
	useInvalidInputFlagErrors(cancelProcessInstanceCmd)

	fs := cancelProcessInstanceCmd.Flags()
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after cancellation is accepted")
	fs.BoolVar(&flagNoStateCheck, "no-state-check", false, "skip checking the current state of the process instance before cancelling it")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview cancel scope without submitting cancellation")

	fs.StringSliceVarP(&flagCancelPIKeys, "key", "k", nil, "process instance key(s) to cancel")
	fs.BoolVar(&flagForce, "force", false, "cancel the root instance when a selected instance is a child")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --batch-size > 1 (default: min(batch-size, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	// flags from get process instance for filtering
	registerPISharedProcessDefinitionFilterFlags(fs)
	registerPISharedDateRangeFlags(fs)
	fs.Int32VarP(&flagGetPISize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to inspect per discovery page; does not cap total selected scope (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetPILimit, "limit", "l", 0, "maximum number of matching process instances to select for cancellation across all pages; omit to continue through all matches")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled, terminated")

	setCommandMutation(cancelProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(cancelProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(cancelProcessInstanceCmd, AutomationSupportFull, "supports unattended destructive confirmation, non-mutating dry-run previews, and paged continuation")
}
