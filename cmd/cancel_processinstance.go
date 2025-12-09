package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
)

var (
	flagCancelPIKeys []string
)

var cancelProcessInstanceCmd = &cobra.Command{
	Use:     "process-instance",
	Short:   "Cancel process instance(s) by key(s) and wait for the cancellation to complete",
	Aliases: []string{"pi"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("initializing client: %w", err))
		}
		if cmd.Flags().Changed("workers") && flagWorkers < 1 {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("--workers must be positive integer"))
		}
		keys := mergeAndValidateKeys(flagCancelPIKeys, log, cfg)

		switch {
		case len(keys) > 0:
		default:
			searchFilterOpts, ok := populatePISearchFilterOpts()
			if !ok {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("either at least one --key is required, or sufficient filtering options to search for process instances to cancel"))
			}
			if searchFilterOpts.State.In(process.StateCanceled, process.StateCompleted, process.StateTerminated) {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("it does not make sense to cancel process instances already in state %q", searchFilterOpts.State.String()))
			}
			pisr, err := cli.SearchProcessInstances(cmd.Context(), searchFilterOpts, consts.MaxPISearchSize)
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error fetching process instances: %w", err))
			}
			keys = make([]string, 0, len(pisr.Items))
			for _, pi := range pisr.Items {
				keys = append(keys, pi.Key)
			}
		}
		if len(keys) == 0 {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("no process instance keys provided or found to cancel"))
		}
		prompt := fmt.Sprintf("You are about to cancel %d process instance(s)?", len(keys))
		if err := confirmCmdOrAbort(flagCmdAutoConfirm, prompt); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		_, err = cli.CancelProcessInstances(cmd.Context(), keys, flagWorkers, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("cancelling process instance(s): %w", err))
		}
	},
}

func init() {
	cancelCmd.AddCommand(cancelProcessInstanceCmd)

	fs := cancelProcessInstanceCmd.Flags()
	fs.BoolVar(&flagNoWait, "no-wait", false, "skip waiting for the cancellation to be fully processed (no status checks)")
	fs.BoolVar(&flagNoStateCheck, "no-state-check", false, "skip checking the current state of the process instance before cancelling it")
	fs.BoolVar(&flagDryRun, "dry-run", false, "perform a dry-run; show which process instances would be cancelled without actually cancelling them")

	fs.StringSliceVarP(&flagCancelPIKeys, "key", "k", nil, "process instance key(s) to cancel")
	fs.BoolVar(&flagForce, "force", false, "force cancellation of the root process instance if a process instance is a child, including all its child instances")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	// flags from get process instance for filtering
	fs.StringVarP(&flagGetPIBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter process instances")
	fs.Int32Var(&flagGetPIProcessVersion, "pd-version", 0, "process definition version")
	fs.StringVar(&flagGetPIProcessVersionTag, "pd-version-tag", "", "process definition version tag")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled")
}
