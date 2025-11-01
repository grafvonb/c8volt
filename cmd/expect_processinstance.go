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
	flagExpectPIKey    string
	flagExpectPIStates []string
)

var expectProcessInstanceCmd = &cobra.Command{
	Use:     "process-instance",
	Short:   "Expect a process instance to reach a certain state",
	Aliases: []string{"pi"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, _, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, err)
		}
		states, err := process.ParseStates(flagExpectPIStates)
		if err != nil {
			log.Error(fmt.Sprintf("error parsing states: %v", err))
			os.Exit(exitcode.NotFound)
		}
		log.Info(fmt.Sprintf("waiting for process instance %s to reach one of the states [%s]", flagExpectPIKey, states))
		got, err := cli.WaitForProcessInstanceState(cmd.Context(), flagExpectPIKey, states, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, fmt.Errorf("cancelling process instance: %w", err))
		}
		log.Info(fmt.Sprintf("process instance %s reached desired state %s", flagExpectPIKey, got))
	},
}

func init() {
	expectCmd.AddCommand(expectProcessInstanceCmd)

	expectProcessInstanceCmd.Flags().StringVarP(&flagExpectPIKey, "key", "k", "", "process instance key to expect a state for")
	_ = expectProcessInstanceCmd.MarkFlagRequired("key")
	expectProcessInstanceCmd.Flags().StringSliceVarP(&flagExpectPIStates, "state", "s", nil, "state of a process instance: ACTIVE, COMPLETED, CANCELED, TERMINATED or ABSENT")
	_ = expectProcessInstanceCmd.MarkFlagRequired("state")
}
