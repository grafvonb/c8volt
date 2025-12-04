package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
)

var (
	flagCancelPIKeys      []string
	flagCancelPIWithForce bool

	flagCancelPIWorkers  int
	flagCancelPIFailFast bool
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
		if cmd.Flags().Changed("workers") && flagCancelPIWorkers < 1 {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("--workers must be positive integer"))
		}

		keys := append([]string{}, flagCancelPIKeys...)
		if inKeys, err := readKeysFromStdin(); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("reading stdin: %w", err))
		} else if len(inKeys) > 0 {
			if ok, firstBadKey, firstBadIndex := validateKeys(inKeys); !ok {
				if strings.HasPrefix(firstBadKey, "filter: ") {
					ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("validating keys from stdin failed: use --keys-only flag to get only keys as input"))
				}
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("validating keys from stdin failed: line %q at index %d is not a valid key; have you forgotten to use --keys-only flag in case of c8volt commands?", firstBadKey, firstBadIndex))
			}
			keys = append(keys, inKeys...)
		}

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
		_, err = cli.CancelProcessInstances(cmd.Context(), keys, flagCancelPIWorkers, flagCancelPIFailFast, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("cancelling process instance(s): %w", err))
		}
	},
}

func init() {
	cancelCmd.AddCommand(cancelProcessInstanceCmd)

	fs := cancelProcessInstanceCmd.Flags()
	fs.BoolVar(&flagCancelNoWait, "no-wait", false, "skip waiting for the cancellation to be fully processed (no status checks)")
	fs.BoolVar(&flagCancelNoStateCheck, "no-state-check", false, "skip checking the current state of the process instance before cancelling it")
	fs.StringSliceVarP(&flagCancelPIKeys, "key", "k", nil, "process instance key(s) to cancel")
	fs.BoolVar(&flagCancelPIWithForce, "force", false, "force cancellation of the root process instance if a process instance is a child, including all its child instances")

	fs.IntVarP(&flagCancelPIWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagCancelPIFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	// flags from get process instance for filtering
	fs.StringVarP(&flagGetPIBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter process instances")
	fs.Int32Var(&flagGetPIProcessVersion, "pd-version", 0, "process definition version")
	fs.StringVar(&flagGetPIProcessVersionTag, "pd-version-tag", "", "process definition version tag")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled")
}
