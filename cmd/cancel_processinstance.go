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
	Long: `Cancel process instances selected by key or search filters.

The plan includes affected instance families. Selecting a child requires --force
to cancel its root. By default, wait until every affected instance is terminal
or absent. Use --dry-run to review the family before cancellation.

Use --limit to bound matching seeds; family members are included in full.
Business-ID selection with --bid is available only c8.10+.
Empty searches finish without confirmation or mutation.`,
	Example: `  c8volt cancel process-instance --key <process-instance-key> --dry-run
  c8volt cancel process-instance --bpmn-process-id order --state active --dry-run
  c8volt cancel process-instance --key <child-instance-key> --force`,
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

		if searchBusinessIDFlag(cmd) != nil {
			if cmd.Flags().Changed("key") || len(args) > 0 {
				handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, mutuallyExclusiveFlagsf("--bid cannot be combined with --key or stdin keys"))
				return
			}
			runCancelProcessInstanceBusinessID(cmd, cli, cfg, log)
			return
		}
		stdinKeys, err := readKeysIfDash(args) // only reads when args == []{"-"}
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(cmd, flagCancelPIKeys, stdinKeys, log, cfg).Unique()
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
			handleProcessInstanceMutationError(cmd, log, cfg.App.NoErrCodes, err)
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
		if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), prompt); err != nil {
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

// init registers keyed, paged and frozen BID cancellation controls on the shared command.
func init() {
	cancelCmd.AddCommand(cancelProcessInstanceCmd)
	useInvalidInputFlagErrors(cancelProcessInstanceCmd)

	fs := cancelProcessInstanceCmd.Flags()
	fs.String("bid", "", "exact business ID to select (only c8.10+)")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after cancellation is accepted")
	fs.BoolVar(&flagNoStateCheck, "no-state-check", false, "skip checking the current state of the process instance before cancelling it")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview cancel scope without submitting cancellation")

	fs.StringSliceVarP(&flagCancelPIKeys, "key", "k", nil, "process instance key(s) to cancel")
	fs.BoolVar(&flagForce, "force", false, "cancel the root instance when a selected instance is a child")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers; omit for an automatic limit")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	// flags from get process instance for filtering
	registerPISharedProcessDefinitionFilterFlags(fs)
	registerPISharedDateRangeFlags(fs)
	fs.Int32VarP(&flagGetPISize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("process instances per discovery page (server maximum %d)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetPILimit, "limit", "l", 0, "maximum matching instances to select; omit for all matches")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, suspended (only c8.10+), completed, canceled, terminated")

	setCommandMutation(cancelProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(cancelProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(cancelProcessInstanceCmd, AutomationSupportFull, "supports unattended destructive confirmation, non-mutating dry-run previews, and paged continuation")
}
