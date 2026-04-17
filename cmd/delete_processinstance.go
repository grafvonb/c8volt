package cmd

import (
	"context"
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
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
	Short: "Delete process instance(s) by key or search filters, optionally cancelling first",
	Example: `  ./c8volt delete pi --key 2251799813711967 --force
  ./c8volt delete pi --state completed --count 250
  ./c8volt delete pi --state completed --end-date-after 2026-01-01 --end-date-before 2026-01-31 --auto-confirm
  ./c8volt delete pi --state completed --end-date-older-days 7 --end-date-newer-days 60 --auto-confirm
  ./c8volt delete pi --bpmn-process-id order-process --start-date-after 2026-01-01 --start-date-before 2026-01-31 --auto-confirm
  ./c8volt delete pi --bpmn-process-id order-process --state completed --count 200 --auto-confirm
  ./c8volt delete pi --state active --start-date-newer-days 30 --auto-confirm
  ./c8volt get pi --state completed --keys-only | ./c8volt delete pi - --auto-confirm`,
	Aliases: []string{"pi"},
	Args: func(cmd *cobra.Command, args []string) error {
		return validateOptionalDashArg(args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if err := validatePISearchFlags(); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}

		stdinKeys, err := readKeysIfDash(args) // only reads when args == []{"-"}
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(flagDeletePIKeys, stdinKeys, log, cfg)
		if err := validatePIKeyedModeDateFilters(len(keys)); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		searched := false

		switch {
		case len(keys) > 0:
		default:
			searched = true
			if !hasPISearchFilterFlags() {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, missingDependentFlagsf("either at least one --key is required, or sufficient filtering options to search for process instances to delete"))
			}
			searchFilterOpts := populatePISearchFilterOpts()
			err := processPISearchPagesWithAction(cmd, cli, cfg, searchFilterOpts, func(page process.ProcessInstancePage, firstPage bool) (processInstancePageImpact, error) {
				keys := make(types.Keys, 0, len(page.Items))
				for _, pi := range page.Items {
					keys = append(keys, pi.Key)
				}
				return deleteProcessInstancePage(cmd, cli, keys, firstPage)
			})
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("delete process instances: %w", err))
			}
			return
		}
		if len(keys) == 0 {
			if searched {
				cmd.Println("found:", 0)
				return
			}
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no process instance keys provided or found to delete")))
		}
		roots, collected, err := cli.DryRunCancelOrDeleteGetPIKeys(context.Background(), keys, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("delete validation: %w", err))
		}
		affectedCount, rootCount, requestedCount := len(collected), len(roots), len(keys)
		prompt := fmt.Sprintf("You are about to delete %d process instance(s). Do you want to proceed?", affectedCount)
		if affectedCount > requestedCount {
			prompt = fmt.Sprintf("You have requested to delete %d process instance(s), but due to dependencies, a total of %d instance(s) with %d root instance(s) will be deleted. Do you want to proceed?", requestedCount, affectedCount, rootCount)
		}
		if err := confirmCmdOrAbort(flagCmdAutoConfirm, prompt); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		_, err = cli.DeleteProcessInstances(cmd.Context(), roots, flagWorkers, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("delete process instances: %w", err))
		}
	},
}

func deleteProcessInstancePage(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool) (processInstancePageImpact, error) {
	roots, collected, err := cli.DryRunCancelOrDeleteGetPIKeys(context.Background(), keys, collectOptions()...)
	if err != nil {
		return processInstancePageImpact{}, fmt.Errorf("delete validation: %w", err)
	}
	impact := processInstancePageImpact{Requested: len(keys), Affected: len(collected), Roots: len(roots)}

	if firstPage {
		affectedCount, rootCount, requestedCount := impact.Affected, impact.Roots, impact.Requested
		prompt := fmt.Sprintf("You are about to delete %d process instance(s). Do you want to proceed?", affectedCount)
		if affectedCount > requestedCount {
			prompt = fmt.Sprintf("You have requested to delete %d process instance(s), but due to dependencies, a total of %d instance(s) with %d root instance(s) will be deleted. Do you want to proceed?", requestedCount, affectedCount, rootCount)
		}
		if err := confirmCmdOrAbortFn(flagCmdAutoConfirm, prompt); err != nil {
			return processInstancePageImpact{}, err
		}
	}

	_, err = cli.DeleteProcessInstances(cmd.Context(), roots, flagWorkers, collectOptions()...)
	if err != nil {
		return processInstancePageImpact{}, fmt.Errorf("delete process instances: %w", err)
	}
	return impact, nil
}

func init() {
	deleteCmd.AddCommand(deleteProcessInstanceCmd)

	fs := deleteProcessInstanceCmd.Flags()
	fs.BoolVar(&flagNoWait, "no-wait", false, "skip waiting for the deletion to be fully processed")
	fs.BoolVar(&flagNoStateCheck, "no-state-check", false, "skip checking the current state of the process instance before deleting it")
	fs.StringSliceVarP(&flagDeletePIKeys, "key", "k", nil, "process instance key(s) to delete")
	fs.BoolVar(&flagForce, "force", false, "force cancellation of the process instance(s), prior to deletion")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	// flags from get process instance for filtering
	registerPISharedProcessDefinitionFilterFlags(fs)
	registerPISharedDateRangeFlags(fs)
	registerPISharedRenderFlags(fs)
	fs.Int32VarP(&flagGetPISize, "count", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to process per page (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled")

	setCommandMutation(deleteProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(deleteProcessInstanceCmd, ContractSupportUnsupported)
	setOutputModes(deleteProcessInstanceCmd,
		OutputModeContract{
			Name:      RenderModeOneLine.String(),
			Supported: true,
		},
		OutputModeContract{
			Name:      RenderModeJSON.String(),
			Supported: false,
			Notes:     "shared result envelope not wired yet",
		},
	)
}
