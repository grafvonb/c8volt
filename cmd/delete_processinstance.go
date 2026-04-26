// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
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
		"By default c8volt validates the affected tree, prompts before deletion, and waits until deletion is observed. Use --force when active instances should be cancelled before deletion.\n\n" +
		"Use --dry-run to preview the resolved scope without submitting deletion, cancel-before-delete requests, prompting for confirmation, or waiting for completion.\n\n" +
		"Use --auto-confirm for unattended destructive runs. Add --no-wait when accepted deletion is enough for the current step, then verify later with `get pi` or `expect pi --state absent`.",
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
			results, err := processPISearchPagesWithAction(cmd, cli, cfg, searchFilterOpts, func(page process.ProcessInstancePage, firstPage bool) (processInstancePageActionResult, error) {
				keys := make(types.Keys, 0, len(page.Items))
				for _, pi := range page.Items {
					keys = append(keys, pi.Key)
				}
				return deleteProcessInstancesWithPlanAndRender(cmd, cli, keys, firstPage, false)
			})
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

func deleteProcessInstancesWithPlan(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool) (processInstancePageActionResult, error) {
	return deleteProcessInstancesWithPlanAndRender(cmd, cli, keys, firstPage, true)
}

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

func deleteProcessInstancePage(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool) (processInstancePageActionResult, error) {
	return deleteProcessInstancesWithPlan(cmd, cli, keys, firstPage)
}

func init() {
	deleteCmd.AddCommand(deleteProcessInstanceCmd)
	useInvalidInputFlagErrors(deleteProcessInstanceCmd)

	fs := deleteProcessInstanceCmd.Flags()
	fs.BoolVar(&flagNoWait, "no-wait", false, "skip waiting for the deletion to be fully processed")
	fs.BoolVar(&flagNoStateCheck, "no-state-check", false, "skip checking the current state of the process instance before deleting it")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview which process instances would be deleted without submitting deletion")
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
