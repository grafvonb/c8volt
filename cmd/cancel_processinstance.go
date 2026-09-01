// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	processOptions "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
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
		"Tenant contract: --tenant scopes search-derived candidate discovery where supported. Empty tenant configuration leaves discovery unfiltered and is reported as \"selection scope: unfiltered across accessible tenants\". Explicit --tenant changes are reported before scope, and --tenant \"\" warns when it clears a named configured filter. Explicit --key and stdin keys are backend-authorized admin input and report that the tenant filter is not applied; existing dry-run, confirmation, force, and wait safety checks still apply.\n\n" +
		"Resolved plans show one known resource tenant informationally, emit one warning-level \"affected tenants\" summary when the frozen scope spans multiple tenants, and warn separately for targets with unknown tenant metadata.\n\n" +
		"When --bpmn-process-id is set, c8volt validates that the process definition is visible before searching process instances. A missing selector fails with a local diagnostic before paging, dry-run planning, confirmation, or cancellation; --json, --automation, and non-TTY runs never prompt for recovery output. If the selector is visible but no matching instances are found, no cancellation request is submitted.\n\n" +
		"Search mode pages through matching process instances by default. --batch-size controls each discovery page request, --limit caps the selected process-instance scope across all pages, and --workers, --fail-fast, and --no-worker-limit bound independent planning or cancellation work. Verbose paging progress is written away from stdout; JSON, quiet, and automation output remain free of prompts unless confirmation is explicitly supplied.\n\n" +
		"After confirmation, default human output keeps one workflow activity updated from real cancellation completions and writes compact stderr milestones at most once per 10-second interval, plus immediate failure warnings. Verbose and debug output replace aggregate milestones with one per-root completion line. JSON, keys-only, and automation output remain free of human progress text; quiet mode suppresses successful progress and retains failure warnings.\n\n" +
		"Use --dry-run to preview selected, in-scope, final-state, and partial-scope instances without cancelling.\n\n" +
		"Use --auto-confirm for unattended destructive runs.",
	Example: `  ./c8volt cancel process-instance --key <process-instance-key>
  ./c8volt cancel process-instance --key <process-instance-key> --dry-run
  ./c8volt cancel process-instance --key <process-instance-key> --force
  ./c8volt --tenant tenant-a cancel process-instance --key <process-instance-key> --dry-run
  ./c8volt --tenant tenant-a cancel process-instance --state active --limit 5 --dry-run
  ./c8volt --tenant "" cancel process-instance --state active --limit 5 --dry-run
  ./c8volt cancel process-instance --state active --batch-size 250 --limit 5 --dry-run
  ./c8volt cancel process-instance --state active --start-date-before 2026-05-31 --limit 5 --dry-run
  ./c8volt cancel process-instance --state active --start-date-newer-days 30 --limit 5 --dry-run
  ./c8volt cancel process-instance --bpmn-process-id <bpmn-process-id> --state active --limit 5 --auto-confirm
  ./c8volt --verbose cancel process-instance --state active --limit 25 --auto-confirm
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

		if len(keys) == 0 {
			runCancelProcessInstanceSearch(cmd, cli, cfg, log)
			return
		}

		result, err := runCancelProcessInstanceDirect(cmd, cli, keys)
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

// runCancelProcessInstanceDirect keeps explicit key and stdin-key cancellation
// on the direct-key path, separate from selector/search execution.
func runCancelProcessInstanceDirect(cmd *cobra.Command, cli process.API, keys types.Keys) (processInstancePageActionResult, error) {
	return cancelProcessInstancesWithPlan(cmd, cli, keys, true)
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
	renderAttachedTenantContext(cmd)
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

	mutationOpts, closeSemanticProgress := appendProcessInstanceMutationSemanticProgressOptions(cmd, "cancel", impact, opts, len(plan.Collected))
	reports, err := cli.CancelProcessInstances(cmd.Context(), plan.Roots, flagWorkers, mutationOpts...)
	closeSemanticProgress()
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
	renderProcessInstanceMutationResultSummary(cmd, "cancel", result.Reports, impact)
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
