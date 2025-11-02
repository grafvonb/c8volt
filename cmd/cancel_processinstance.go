package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

var (
	flagCancelPIKey       string
	flagCancelPIWithForce bool
)

var cancelProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Cancel a process instance by its key and wait for the cancellation to complete",
	Long: `Cancel a process instance by its key and wait for the cancellation to complete.
If the process instance is a child instance, use the --force flag to cancel
the root process instance including all its child instances.
You can use the --no-wait flag to skip waiting for the cancellation to complete.`,
	Aliases: []string{"pi"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("initializing client: %w", err))
		}

		_, err = cli.CancelProcessInstance(cmd.Context(), flagCancelPIKey, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("cancelling process instance: %w", err))
		}
	},
}

func init() {
	cancelCmd.AddCommand(cancelProcessInstanceCmd)

	cancelProcessInstanceCmd.Flags().BoolVar(&flagCancelNoWait, "no-wait", false, "skip waiting for the cancellation to be fully processed (no status checks)")
	cancelProcessInstanceCmd.Flags().BoolVar(&flagCancelNoStateCheck, "no-state-check", false, "skip checking the current state of the process instance before cancelling it")
	cancelProcessInstanceCmd.Flags().StringVarP(&flagCancelPIKey, "key", "k", "", "process instance key to cancel")
	_ = cancelProcessInstanceCmd.MarkFlagRequired("key")
	cancelProcessInstanceCmd.Flags().BoolVar(&flagCancelPIWithForce, "force", false, "force cancellation of the root process instance if a process instance is a child, including all its child instances")
}
