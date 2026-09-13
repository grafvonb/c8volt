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
	Long: `Delete process instances by key or search filters.

c8volt validates the affected tree before any deletion, asks for confirmation, and waits until deletion is observed. Nonterminal instances block deletion unless --force is set.

--force allows cancellation when deletion encounters a nonterminal instance. Deletion traverses children first and retries after cancellation; the entire scope is not canceled before any deletion.

--tenant limits search-derived selection. An empty tenant or --all-tenants leaves discovery unfiltered across accessible tenants. Explicit --key and stdin keys use backend authorization without tenant filtering.

A --bpmn-process-id selector must match a visible process definition before instance discovery. An empty selection completes without confirmation or mutation.

Search mode plans all selected pages before one confirmation and deletion. --batch-size controls each discovery request; --limit caps the selected scope across all pages. --workers, --fail-fast, and --no-worker-limit control planning and deletion work.

Use --verbose to explain cancellation prerequisites, root escalation, accepted cancellation, confirmation waits, and deletion resumption. Use --debug for per-check state observations and HTTP diagnostics; it remains independent of --verbose. A confirmation timeout reports accepted cancellation as submitted but unconfirmed and does not claim that deletion resumed.

Use --dry-run to preview the affected family without deleting or cancelling. Use --auto-confirm for unattended deletion.`,
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
  ./c8volt --verbose --debug delete process-instance --key <process-instance-key> --force
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
	fs.BoolVar(&flagForce, "force", false, "allow cancellation when deletion encounters nonterminal process instances")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers for queued work; mutation work uses root trees (default: min(queued work, 2*GOMAXPROCS, 32)); independent of discovery page size")
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
