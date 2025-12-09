package cmd

import (
	"fmt"
	"os"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/spf13/cobra"
)

var (
	flagExpectPIKeys   []string
	flagExpectPIStates []string
)

var expectProcessInstanceCmd = &cobra.Command{
	Use:     "process-instance",
	Short:   "Expect a process instance(s) to reach a certain state from list of states",
	Aliases: []string{"pi"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		if cmd.Flags().Changed("workers") && flagWorkers < 1 {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("--workers must be positive integer"))
		}
		states, err := process.ParseStates(flagExpectPIStates)
		if err != nil {
			log.Error(fmt.Sprintf("error parsing states: %v; valid values are: [active, completed, canceled, terminated or absent]", err))
			os.Exit(exitcode.NotFound)
		}
		keys := mergeAndValidateKeys(flagExpectPIKeys, log, cfg)
		if len(keys) == 0 {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("no process instance keys provided or found to watch"))
		}
		log.Info(fmt.Sprintf("waiting for %d process instance(s) [%s] to reach one of the states [%s]", len(keys), keys, states))
		_, err = cli.WaitForProcessInstancesState(cmd.Context(), keys, states, flagWorkers, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("expecting process instance: %w", err))
		}
		log.Info(fmt.Sprintf("%d process instance(s) [%s] reached desired state(s) [%s]", len(keys), keys, states))
	},
}

func init() {
	expectCmd.AddCommand(expectProcessInstanceCmd)

	fs := expectProcessInstanceCmd.Flags()
	fs.StringSliceVarP(&flagExpectPIKeys, "key", "k", nil, "process instance key(s) to expect a state for")
	_ = expectProcessInstanceCmd.MarkFlagRequired("key")
	fs.StringSliceVarP(&flagExpectPIStates, "state", "s", nil, "state of a process instance; valid values aer: [active, completed, canceled, terminated or absent]")
	_ = expectProcessInstanceCmd.MarkFlagRequired("state")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")
}
