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
	flagDeletePIKeys []string
)

var deleteProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Delete process instances by key or filters",
	Long: "Delete process instances by key or search filters, optionally cancelling first.\n\n" +
		"By default c8volt validates the complete affected tree before submitting any delete request, prompts before deletion, and waits until deletion is observed. If any affected process instance is not in a final state, the whole delete batch is refused before mutation. Use --force to cancel the affected scope first, then delete it.\n\n" +
		"Tenant contract: --tenant scopes search-derived candidate discovery where supported. Empty tenant configuration leaves discovery unfiltered and is reported as \"selection scope: unfiltered across accessible tenants\". Explicit --tenant changes are reported before scope, and --tenant \"\" warns when it clears a named configured filter. Explicit --key and stdin keys are backend-authorized admin input and report that the tenant filter is not applied; existing dry-run, confirmation, force, and wait safety checks still apply.\n\n" +
		"Resolved delete plans show one known resource tenant informationally, emit one warning-level \"affected tenants\" summary when the frozen scope spans multiple tenants, and warn separately for targets with unknown tenant metadata.\n\n" +
		"When --bpmn-process-id is set, c8volt validates that the process definition is visible before searching process instances. A missing selector fails with a local diagnostic before paging, dry-run planning, confirmation, cancellation, or deletion; --json, --automation, and non-TTY runs never prompt for recovery output. If the selector is visible but no matching instances are found, no deletion request is submitted.\n\n" +
		"When a selector search succeeds with no matching instances, deletion completes as a successful no-op without confirmation or mutation. Human output is exactly \"found: 0\"; --quiet suppresses that summary, --keys-only writes zero bytes, and --json writes one succeeded result envelope with an empty deletion payload. The same output rules apply to --dry-run, whose JSON preview reports mutationSubmitted: false.\n\n" +
		"Search mode pages through matching process instances by default and freezes every selected page-level delete plan before one confirmation and mutation. --batch-size controls each discovery page request, --limit caps the frozen delete scope across all pages, and --workers, --fail-fast, and --no-worker-limit bound independent planning, cancellation, or deletion work. Verbose paging progress is written away from stdout; JSON, quiet, and automation output remain free of prompts unless confirmation is explicitly supplied.\n\n" +
		"After confirmation, default human output keeps one workflow activity updated from real deletion completions and writes compact stderr milestones at most once per 10-second interval, plus immediate failure warnings. Verbose and debug output replace aggregate milestones with one per-root completion line. JSON, keys-only, and automation output remain free of human progress text; quiet mode suppresses successful progress and retains failure warnings.\n\n" +
		"With --json, validation and runtime failures during command execution use one shared error envelope. Without --json, the diagnostic is written to stderr. --no-err-codes changes only the process exit status; the reported failure and immediate termination are unchanged. Bootstrap failures and argument or flag parsing errors before command execution retain their established diagnostics.\n\n" +
		"Use --dry-run to preview selected, in-scope, final-state, non-final, and partial-scope instances without deleting or cancelling.\n\n" +
		"Use --auto-confirm for unattended destructive runs.",
	Example: `  ./c8volt delete process-instance --key <process-instance-key> --force
  ./c8volt delete process-instance --key <process-instance-key> --dry-run
  ./c8volt --tenant tenant-a delete process-instance --key <process-instance-key> --dry-run
  ./c8volt --tenant tenant-a delete process-instance --state terminated --limit 5 --dry-run
  ./c8volt --tenant "" delete process-instance --state terminated --limit 5 --dry-run
  ./c8volt delete process-instance --state terminated --batch-size 250 --limit 5 --dry-run
  ./c8volt delete process-instance --state terminated --json --dry-run
  ./c8volt delete process-instance --state terminated --keys-only
  ./c8volt delete process-instance --state terminated --end-date-after 2026-05-01 --end-date-before 2026-05-31 --limit 5 --dry-run
  ./c8volt delete process-instance --bpmn-process-id <bpmn-process-id> --state terminated --batch-size 250 --limit 5 --dry-run
  ./c8volt --verbose delete process-instance --state terminated --limit 25 --auto-confirm
  ./c8volt expect process-instance --key <process-instance-key> --state absent`,
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
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}

		stdinKeys, err := readKeysIfDash(args) // only reads when args == []{"-"}
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(cmd, flagDeletePIKeys, stdinKeys, log, cfg).Unique()
		if err := validatePIKeyedModeDateFilters(len(keys)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if err := validatePIKeyedModeLimit(len(keys)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}

		if len(keys) == 0 {
			runDeleteProcessInstanceSearch(cmd, cli, cfg, log)
			return
		}

		result, err := runDeleteProcessInstanceDirect(cmd, cli, keys)
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

// runDeleteProcessInstanceDirect keeps explicit key and stdin-key deletion on
// the direct-key path, separate from selector/search execution.
func runDeleteProcessInstanceDirect(cmd *cobra.Command, cli process.API, keys types.Keys) (processInstancePageActionResult, error) {
	return deleteProcessInstancesWithPlan(cmd, cli, keys, true)
}

// deleteProcessInstancesWithPlan validates the delete scope, renders dry-run
// output when requested, and submits the mutation otherwise.
func deleteProcessInstancesWithPlan(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool) (processInstancePageActionResult, error) {
	return deleteProcessInstancesWithPlanAndRenderWithOptions(cmd, cli, keys, firstPage, true, collectExplicitPIAdminInputOptions())
}

// deleteProcessInstancesWithPlanAndRender shares delete planning for keyed and
// paged flows while allowing callers to defer dry-run rendering.
func deleteProcessInstancesWithPlanAndRender(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool, renderDryRun bool) (processInstancePageActionResult, error) {
	return deleteProcessInstancesWithPlanAndRenderWithOptions(cmd, cli, keys, firstPage, renderDryRun, collectOptions())
}

// deleteProcessInstancesWithPlanAndRenderWithOptions keeps direct-key admin
// input separate from tenant-scoped search-derived candidates.
func deleteProcessInstancesWithPlanAndRenderWithOptions(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool, renderDryRun bool, opts []processOptions.FacadeOption) (processInstancePageActionResult, error) {
	planned, err := planProcessInstanceDryRunPreviewWithOptions(cmd, cli, "delete", keys, opts)
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
	renderAttachedTenantContext(cmd)
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
		if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), prompt); err != nil {
			return processInstancePageActionResult{}, err
		}
	}

	mutationOpts, closeSemanticProgress := appendProcessInstanceMutationSemanticProgressOptions(cmd, "delete", impact, opts, len(plan.Collected))
	reports, err := cli.DeleteProcessInstances(cmd.Context(), plan.Roots, flagWorkers, mutationOpts...)
	closeSemanticProgress()
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
	renderProcessInstanceMutationResultSummary(cmd, "delete", result.Reports, impact)
	return result, nil
}

// rejectDeletePlanRequiringForce rejects non-final delete scope unless force mode can cancel first.
func rejectDeletePlanRequiringForce(plan process.DryRunPIKeyExpansion) error {
	if flagForce || len(plan.RequiresCancelBeforeDelete) == 0 {
		return nil
	}
	return localPreconditionError(fmt.Errorf("refusing to delete process-instance scope: %d affected process instance(s) are not in a final state; no delete request was submitted; use --force to cancel the entire affected scope before delete", len(plan.RequiresCancelBeforeDelete)))
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
	fs.StringSliceVarP(&flagDeletePIKeys, "key", "k", nil, "process instance key(s) to delete; repeat or combine with stdin '-'")
	fs.BoolVar(&flagForce, "force", false, "force cancellation of the process instance(s), prior to deletion")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --batch-size > 1 (default: min(batch-size, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	// flags from get process instance for filtering
	registerPISharedProcessDefinitionFilterFlags(fs)
	registerPISharedDateRangeFlags(fs)
	fs.Int32VarP(&flagGetPISize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to inspect per discovery page; does not cap total frozen scope (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetPILimit, "limit", "l", 0, "maximum number of matching process instances to freeze for deletion across all pages; omit to continue through all matches")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled, terminated")

	setCommandMutation(deleteProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(deleteProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(deleteProcessInstanceCmd, AutomationSupportFull, "supports unattended destructive confirmation, non-mutating dry-run previews, and paged continuation")
}
