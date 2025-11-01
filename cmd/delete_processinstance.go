package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

var (
	flagDeletePIKey        string
	flagDeletePIWithCancel bool
)

var deleteProcessInstanceCmd = &cobra.Command{
	Use:     "process-instance",
	Short:   "Delete a process instance by its key",
	Aliases: []string{"pi"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, _, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, err)
		}
		_, err = cli.DeleteProcessInstance(cmd.Context(), flagDeletePIKey, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, fmt.Errorf("deleting process instance with key %s: %w", flagDeletePIKey, err))
		}

	},
}

func init() {
	deleteCmd.AddCommand(deleteProcessInstanceCmd)

	deleteProcessInstanceCmd.Flags().StringVarP(&flagDeletePIKey, "key", "k", "", "process instance key to delete")
	_ = deleteProcessInstanceCmd.MarkFlagRequired("key")
	deleteProcessInstanceCmd.Flags().BoolVar(&flagDeletePIWithCancel, "with-cancel", false, "cancel the process instance before deleting it")
}
