// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	processOptions "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

var (
	flagDeletePIKeys []string
)

var deleteProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Delete process instances by key or filters",
	Long: "Delete process instances by key or search filters, optionally cancelling first.\n\n" +
		"By default c8volt validates the complete affected tree before submitting any delete request, prompts before deletion, and waits until deletion is observed. If any affected process instance is not in a final state, the whole delete batch is refused before mutation. Use --force to cancel the affected scope first, then delete it.\n\n" +
		"Use --dry-run to preview selected, in-scope, final-state, non-final, and partial-scope instances without deleting or cancelling.\n\n" +
		"Use --auto-confirm for unattended destructive runs. Add --no-wait to verify later with `get pi` or `expect pi --state absent`.",
	Example: `  ./c8volt delete pi --key 2251799813711967 --force
  ./c8volt delete pi --key 2251799813711967 --dry-run
  ./c8volt delete pi --state completed --batch-size 250
  ./c8volt delete pi --state completed --batch-size 250 --limit 25
  ./c8volt delete pi --state completed --batch-size 250 --limit 25 --dry-run
  ./c8volt delete pi --state completed --end-date-after 2026-01-01 --end-date-before 2026-01-31 --auto-confirm
  ./c8volt delete pi --state completed --end-date-older-days 7 --end-date-newer-days 60 --auto-confirm
  ./c8volt delete pi --bpmn-process-id C88_SimpleUserTask_Process --state completed --batch-size 200 --auto-confirm
  ./c8volt delete pi --state completed --batch-size 200 --auto-confirm --no-wait
  ./c8volt expect pi --key <process-instance-key> --state absent`,
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
		if err := validatePISearchFlags(cmd); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}

		stdinKeys, err := readKeysIfDash(args) // only reads when args == []{"-"}
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(flagDeletePIKeys, stdinKeys, log, cfg)
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
				handleCommandError(cmd, log, cfg.App.NoErrCodes, missingDependentFlagsf("either at least one --key is required, or sufficient filtering options to search for process instances to delete"))
			}
			searchFilterOpts := populatePISearchFilterOpts()
			results, err := deleteProcessInstanceSearchPages(cmd, cli, cfg, searchFilterOpts)
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
			if len(reports) > 0 {
				payload := process.DeleteReports{Items: make([]process.DeleteReport, len(reports))}
				for i, report := range reports {
					payload.Items[i] = process.DeleteReport(report)
				}
				if err := renderCommandResult(cmd, payload); err != nil {
					handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render delete result: %w", err))
				}
			}
			return
		}
		if len(keys) == 0 {
			if searched {
				renderOutputLine(cmd, "found: %d", 0)
				return
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no process instance keys provided or found to delete")))
		}
		result, err := deleteProcessInstancesWithPlan(cmd, cli, keys, true)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if flagDryRun {
			return
		}
		payload := process.DeleteReports{Items: make([]process.DeleteReport, len(result.Reports))}
		for i, report := range result.Reports {
			payload.Items[i] = process.DeleteReport(report)
		}
		if err := renderCommandResult(cmd, payload); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render delete result: %w", err))
		}
		return
	},
}

// deleteProcessInstancesWithPlan validates the delete scope, renders dry-run
// output when requested, and submits the mutation otherwise.
func deleteProcessInstancesWithPlan(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool) (processInstancePageActionResult, error) {
	return deleteProcessInstancesWithPlanAndRender(cmd, cli, keys, firstPage, true)
}

// deleteProcessInstancesWithPlanAndRender shares delete planning for keyed and
// paged flows while allowing callers to defer dry-run rendering.
func deleteProcessInstancesWithPlanAndRender(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool, renderDryRun bool) (processInstancePageActionResult, error) {
	planned, err := planProcessInstanceDryRunPreview(cmd, cli, "delete", keys)
	if err != nil {
		return processInstancePageActionResult{}, err
	}
	plan := planned.Plan
	if flagDryRun {
		if renderDryRun {
			if err := renderProcessInstanceDryRunPreview(cmd, planned.Preview); err != nil {
				return processInstancePageActionResult{}, fmt.Errorf("render delete dry-run result: %w", err)
			}
		}
		return processInstancePageActionResult{
			Impact:        planned.Impact,
			DryRunPreview: &planned.Preview,
		}, nil
	}
	printDryRunExpansionWarning(cmd, plan)
	if err := rejectDeletePlanRequiringForce(plan); err != nil {
		return processInstancePageActionResult{}, err
	}

	impact := planned.Impact

	if firstPage {
		affectedCount, rootCount, requestedCount := impact.Affected, impact.Roots, impact.Requested
		prompt := fmt.Sprintf("You are about to delete %d process instance(s). Do you want to proceed?", affectedCount)
		if affectedCount > requestedCount {
			prompt = fmt.Sprintf("You have requested to delete %d process instance(s), but due to dependencies, a total of %d instance(s) with %d root instance(s) will be deleted. Do you want to proceed?", requestedCount, affectedCount, rootCount)
		}
		if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
			return processInstancePageActionResult{}, err
		}
	}

	opts := append(collectOptions(), processOptions.WithAffectedProcessInstanceCount(len(plan.Collected)))
	reports, err := cli.DeleteProcessInstances(cmd.Context(), plan.Roots, flagWorkers, opts...)
	if err != nil {
		return processInstancePageActionResult{}, fmt.Errorf("delete process instances: %w", err)
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

// deleteProcessInstanceSearchPages preflights search-selected delete scope before submitting mutations.
func deleteProcessInstanceSearchPages(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (processInstancePageActionResults, error) {
	if flagDryRun {
		return processPISearchPagesWithAction(cmd, cli, cfg, filter, func(page process.ProcessInstancePage, firstPage bool) (processInstancePageActionResult, error) {
			keys := make(types.Keys, 0, len(page.Items))
			for _, pi := range page.Items {
				keys = append(keys, pi.Key)
			}
			return deleteProcessInstancesWithPlanAndRender(cmd, cli, keys, firstPage, false)
		})
	}

	results, aborted, err := planDeleteProcessInstanceSearchPages(cmd, cli, cfg, filter)
	if err != nil {
		return processInstancePageActionResults{}, err
	}
	if aborted || len(results.DryRunPreviews) == 0 {
		return results, nil
	}
	plan := aggregateDeleteSearchPlan(results.DryRunPreviews)
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

	opts := append(collectOptions(), processOptions.WithAffectedProcessInstanceCount(len(plan.Collected)))
	reports, err := cli.DeleteProcessInstances(cmd.Context(), plan.Roots, flagWorkers, opts...)
	if err != nil {
		return processInstancePageActionResults{}, fmt.Errorf("delete process instances: %w", err)
	}
	results.Reports = make([]process.Reporter, len(reports.Items))
	for i, report := range reports.Items {
		results.Reports[i] = process.Reporter(report)
	}
	return results, nil
}

// planDeleteProcessInstanceSearchPages walks search pages and records delete previews without mutating.
func planDeleteProcessInstanceSearchPages(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (processInstancePageActionResults, bool, error) {
	pageReq := newPISearchPageRequest(cmd, cfg, 0)
	cumulative := 0
	cumulativeAffected := 0
	var results processInstancePageActionResults

	for {
		page, err := cli.SearchProcessInstancesPage(cmd.Context(), filter, pageReq, collectOptions()...)
		if err != nil {
			return processInstancePageActionResults{}, false, err
		}
		if len(page.Items) == 0 {
			if cumulative == 0 {
				renderOutputLine(cmd, "found: %d", 0)
			}
			return results, false, nil
		}

		limitedPage := limitPIPageItems(page, cumulative)
		keys := make(types.Keys, 0, len(limitedPage.Items))
		for _, pi := range limitedPage.Items {
			keys = append(keys, pi.Key)
		}
		planned, err := planProcessInstanceDryRunPreview(cmd, cli, "delete", keys)
		if err != nil {
			return processInstancePageActionResults{}, false, err
		}
		results.DryRunPreviews = append(results.DryRunPreviews, planned.Preview)

		cumulative += len(limitedPage.Items)
		if planned.Impact.Affected > 0 {
			cumulativeAffected += planned.Impact.Affected
		} else {
			cumulativeAffected += len(limitedPage.Items)
		}
		summary := newPIProgressSummary(limitedPage, cumulative, shouldAutoContinuePISearchPages(cmd))
		printPISearchProgress(cmd, summary)

		switch summary.ContinuationState {
		case processInstanceContinuationCompleted, processInstanceContinuationWarningStop, processInstanceContinuationLimitReached:
			return results, false, nil
		case processInstanceContinuationAutoContinue:
			pageReq = newPISearchPageRequest(cmd, cfg, pageReq.From+int32(len(page.Items)))
			continue
		case processInstanceContinuationPrompt:
			prompt := fmt.Sprintf("Checked %d process instance(s) on this page (%d requested so far, %d including dependencies). More matching process instances remain. Continue preflight?", summary.CurrentPageCount, summary.CumulativeCount, cumulativeAffected)
			if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
				if isCmdAborted(err) {
					printPISearchProgress(cmd, processInstanceProgressSummary{
						PageSize:          summary.PageSize,
						CurrentPageCount:  summary.CurrentPageCount,
						CumulativeCount:   summary.CumulativeCount,
						OverflowState:     summary.OverflowState,
						ContinuationState: processInstanceContinuationPartialComplete,
					})
					return results, true, nil
				}
				return processInstancePageActionResults{}, false, err
			}
			pageReq = newPISearchPageRequest(cmd, cfg, pageReq.From+int32(len(page.Items)))
		}
	}
}

// rejectDeletePlanRequiringForce rejects non-final delete scope unless force mode can cancel first.
func rejectDeletePlanRequiringForce(plan process.DryRunPIKeyExpansion) error {
	if flagForce || len(plan.RequiresCancelBeforeDelete) == 0 {
		return nil
	}
	return localPreconditionError(fmt.Errorf("refusing to delete process-instance scope: %d affected process instance(s) are not in a final state; no delete request was submitted; use --force to cancel the entire affected scope before delete", len(plan.RequiresCancelBeforeDelete)))
}

// aggregateDeleteSearchPlan merges page-level delete previews into one mutation plan.
func aggregateDeleteSearchPlan(previews []processInstanceDryRunPreview) process.DryRunPIKeyExpansion {
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

func deleteProcessInstancePage(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool) (processInstancePageActionResult, error) {
	return deleteProcessInstancesWithPlan(cmd, cli, keys, firstPage)
}

func init() {
	deleteCmd.AddCommand(deleteProcessInstanceCmd)
	useInvalidInputFlagErrors(deleteProcessInstanceCmd)

	fs := deleteProcessInstanceCmd.Flags()
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after deletion is accepted")
	fs.BoolVar(&flagNoStateCheck, "no-state-check", false, "skip checking the current state of the process instance before deleting it")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview delete scope without submitting deletion or cancel-before-delete requests")
	fs.StringSliceVarP(&flagDeletePIKeys, "key", "k", nil, "process instance key(s) to delete")
	fs.BoolVar(&flagForce, "force", false, "force cancellation of the process instance(s), prior to deletion")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --batch-size > 1 (default: min(batch-size, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	// flags from get process instance for filtering
	registerPISharedProcessDefinitionFilterFlags(fs)
	registerPISharedDateRangeFlags(fs)
	registerPISharedRenderFlags(fs)
	fs.Int32VarP(&flagGetPISize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to process per page (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetPILimit, "limit", "l", 0, "maximum number of matching process instances to process across all pages")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled, terminated")

	setCommandMutation(deleteProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(deleteProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(deleteProcessInstanceCmd, AutomationSupportFull, "supports unattended destructive confirmation, non-mutating dry-run previews, and paged continuation")
}
