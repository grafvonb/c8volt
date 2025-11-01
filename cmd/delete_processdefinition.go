package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

var (
	flagDeletePDKey           string
	flagDeletePDBpmnProcessId string
)

var deleteProcessDefinitionCmd = &cobra.Command{
	Use:     "process-definition",
	Short:   "Delete a process definition(s)",
	Aliases: []string{"pd"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, _, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, err)
		}
		if flagDeletePDKey == "" && flagDeletePDBpmnProcessId == "" {
			ferrors.HandleAndExit(log, fmt.Errorf("either --key or --bpmn-process-id must be provided to delete process definition(s)"))
		}
		if flagDeletePDKey != "" {
			if err = cli.DeleteProcessDefinitionByKey(cmd.Context(), flagDeletePDKey, collectOptions()...); err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("deleting process definition with key %s: %w", flagDeletePDKey, err))
			}
		}
		if flagDeletePDBpmnProcessId != "" {
			if err = cli.DeleteProcessDefinitionVersionsByBpmnProcessId(cmd.Context(), flagDeletePDBpmnProcessId, collectOptions()...); err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("deleting process definition(s) with BPMN process ID %s: %w", flagDeletePDBpmnProcessId, err))
			}
		}
	},
}

func init() {
	deleteCmd.AddCommand(deleteProcessDefinitionCmd)

	deleteProcessDefinitionCmd.Flags().StringVarP(&flagDeletePDKey, "key", "k", "", "process definition key to delete")
	deleteProcessDefinitionCmd.Flags().StringVarP(&flagDeletePDBpmnProcessId, "bpmn-process-id", "b", "", "BPMN process ID of the process definition (all versions) to delete")
}
