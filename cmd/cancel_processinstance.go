package cmd

import (
	"context"
	"fmt"

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
	Short: "Cancel process instance(s) by key or search filters and wait for completion",
	Long: "Cancel process instance(s) by key or search filters and wait for completion.\n\n" +
		"By default c8volt validates the affected root and descendant instances, asks for confirmation before " +
		"the destructive action, and waits until cancellation is observed. Use --auto-confirm for unattended " +
		"runs, and combine it with --no-wait when accepted cancellation should return immediately instead of " +
		"waiting for the final state.\n\n" +
		"After non-blocking cancellation, use `get process-instance` or `expect process-instance` to verify the " +
		"eventual state of the affected instances.",
	Example: `  ./c8volt cancel pi --key 2251799813711967
  ./c8volt cancel pi --key 2251799813711977 --force
  ./c8volt cancel pi --state active --count 250
  ./c8volt cancel pi --state active --start-date-before 2026-03-31
  ./c8volt cancel pi --state active --start-date-newer-days 30
  ./c8volt cancel pi --bpmn-process-id order-process --state active --count 200 --auto-confirm
  ./c8volt cancel pi --bpmn-process-id order-process --start-date-after 2026-01-01 --start-date-before 2026-01-31
  ./c8volt cancel pi --bpmn-process-id order-process --start-date-older-days 14 --state active
  ./c8volt cancel pi --end-date-after 2026-01-01 --end-date-before 2026-01-31 --state completed
  ./c8volt cancel pi --state active --count 200 --auto-confirm --no-wait
  ./c8volt expect pi --key 2251799813711967 --state canceled --state terminated
  ./c8volt get pi --state active --bpmn-process-id C88_SimpleUserTask_Process --keys-only | ./c8volt cancel pi -`,
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
		if err := validatePISearchFlags(); err != nil {
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
		searched := false

		switch {
		case len(keys) > 0:
		default:
			searched = true
			if !hasPISearchFilterFlags() {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, missingDependentFlagsf("either at least one --key is required, or sufficient filtering options to search for process instances to cancel"))
			}
			searchFilterOpts := populatePISearchFilterOpts()
			reports, err := processPISearchPagesWithAction(cmd, cli, cfg, searchFilterOpts, func(page process.ProcessInstancePage, firstPage bool) (processInstancePageActionResult, error) {
				keys := make(types.Keys, 0, len(page.Items))
				for _, pi := range page.Items {
					keys = append(keys, pi.Key)
				}
				return cancelProcessInstancePage(cmd, cli, keys, firstPage)
			})
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("cancel process instances: %w", err))
			}
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
				cmd.Println("found:", 0)
				return
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no process instance keys provided or found to cancel")))
		}
		roots, collected, err := cli.DryRunCancelOrDeleteGetPIKeys(context.Background(), keys, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("cancel validation: %w", err))
		}
		affectedCount, rootCount, requestedCount := len(collected), len(roots), len(keys)
		prompt := fmt.Sprintf("You are about to cancel %d process instance(s). Do you want to proceed?", affectedCount)
		if affectedCount > requestedCount {
			prompt = fmt.Sprintf("You have requested to cancel %d process instance(s), but due to dependencies, a total of %d instance(s) with %d root instance(s) will be canceled. Do you want to proceed?", requestedCount, affectedCount, rootCount)
		}
		if err := confirmCmdOrAbort(shouldImplicitlyConfirm(cmd), prompt); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		reports, err := cli.CancelProcessInstances(cmd.Context(), keys, flagWorkers, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("cancel process instances: %w", err))
		}
		if err := renderCommandResult(cmd, reports); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render cancel result: %w", err))
		}
	},
}

func cancelProcessInstancePage(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool) (processInstancePageActionResult, error) {
	roots, collected, err := cli.DryRunCancelOrDeleteGetPIKeys(context.Background(), keys, collectOptions()...)
	if err != nil {
		return processInstancePageActionResult{}, fmt.Errorf("cancel validation: %w", err)
	}
	impact := processInstancePageImpact{Requested: len(keys), Affected: len(collected), Roots: len(roots)}

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

	reports, err := cli.CancelProcessInstances(cmd.Context(), keys, flagWorkers, collectOptions()...)
	if err != nil {
		return processInstancePageActionResult{}, fmt.Errorf("cancel process instances: %w", err)
	}
	result := processInstancePageActionResult{
		Impact:  impact,
		Reports: make([]process.Reporter, len(reports.Items)),
	}
	for i, report := range reports.Items {
		result.Reports[i] = process.Reporter(report)
	}
	return result, nil
}

func init() {
	cancelCmd.AddCommand(cancelProcessInstanceCmd)

	fs := cancelProcessInstanceCmd.Flags()
	fs.BoolVar(&flagNoWait, "no-wait", false, "skip waiting for the cancellation to be fully processed")
	fs.BoolVar(&flagNoStateCheck, "no-state-check", false, "skip checking the current state of the process instance before cancelling it")
	// fs.BoolVar(&flagDryRun, "dry-run", false, "perform a dry-run; show which process instances would be canceled without actually cancelling them")

	fs.StringSliceVarP(&flagCancelPIKeys, "key", "k", nil, "process instance key(s) to cancel")
	fs.BoolVar(&flagForce, "force", false, "force cancellation of the root process instance if a process instance is a child, including all its child instances")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	// flags from get process instance for filtering
	registerPISharedProcessDefinitionFilterFlags(fs)
	registerPISharedDateRangeFlags(fs)
	registerPISharedRenderFlags(fs)
	fs.Int32VarP(&flagGetPISize, "count", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to process per page (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled")

	setCommandMutation(cancelProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(cancelProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(cancelProcessInstanceCmd, AutomationSupportFull, "supports unattended destructive confirmation and paged continuation")
}
