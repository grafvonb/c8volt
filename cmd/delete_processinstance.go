package cmd

import (
	"context"
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
)

var (
	flagDeletePIKeys []string
)

var deleteProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Delete process instance(s), optionally cancelling first",
	Example: `  ./c8volt delete pi --key 2251799813711967 --force
  ./c8volt delete pi --state completed --end-date-after 2026-01-01 --end-date-before 2026-01-31 --auto-confirm
  ./c8volt delete pi --bpmn-process-id order-process --start-date-after 2026-01-01 --auto-confirm
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

		switch {
		case len(keys) > 0:
		default:
			if !hasPISearchFilterFlags() {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, missingDependentFlagsf("either at least one --key is required, or sufficient filtering options to search for process instances to delete"))
			}
			searchFilterOpts := populatePISearchFilterOpts()
			pisr, err := cli.SearchProcessInstances(cmd.Context(), searchFilterOpts, consts.MaxPISearchSize, collectOptions()...)
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error fetching process instances: %w", err))
			}
			keys = make([]string, 0, len(pisr.Items))
			for _, pi := range pisr.Items {
				keys = append(keys, pi.Key)
			}
		}
		if len(keys) == 0 {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no process instance keys provided or found to delete")))
		}
		roots, collected, err := cli.DryRunCancelOrDeleteGetPIKeys(context.Background(), keys, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("validating process instance keys for cancellation: %w", err))
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
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("deleting process instance(s): %w", err))
		}
	},
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
	fs.StringVarP(&flagGetPIBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter process instances")
	fs.Int32Var(&flagGetPIProcessVersion, "pd-version", 0, "process definition version")
	fs.StringVar(&flagGetPIProcessVersionTag, "pd-version-tag", "", "process definition version tag")
	fs.StringVar(&flagGetPIStartDateAfter, "start-date-after", "", "inclusive lower start-date bound in YYYY-MM-DD format")
	fs.StringVar(&flagGetPIStartDateBefore, "start-date-before", "", "inclusive upper start-date bound in YYYY-MM-DD format")
	fs.StringVar(&flagGetPIEndDateAfter, "end-date-after", "", "inclusive lower end-date bound in YYYY-MM-DD format")
	fs.StringVar(&flagGetPIEndDateBefore, "end-date-before", "", "inclusive upper end-date bound in YYYY-MM-DD format")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled")
}
