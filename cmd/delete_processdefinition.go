package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

var (
	flagDeletePDKey               string
	flagDeletePDBpmnProcessId     string
	flagDeletePDProcessVersion    int32
	flagDeletePDProcessVersionTag string
	flagDeletePDLatest            bool
)

var deleteProcessDefinitionCmd = &cobra.Command{
	Use:     "process-definition",
	Short:   "Delete a process definition(s)",
	Aliases: []string{"pd"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		if flagDeletePDKey == "" && flagDeletePDBpmnProcessId == "" {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("either --key or --bpmn-process-id must be provided to delete process definition(s)"))
		}
		switch {
		case flagDeletePDKey != "":
		}
		filter := process.ProcessDefinitionFilter{
			Key:               flagDeletePDKey,
			BpmnProcessId:     flagDeletePDBpmnProcessId,
			ProcessVersion:    flagDeletePDProcessVersion,
			ProcessVersionTag: flagDeletePDProcessVersionTag,
		}
		var pds process.ProcessDefinitions
		if !flagDeletePDLatest {
			pds, err = cli.SearchProcessDefinitions(cmd.Context(), filter)
		} else {
			pds, err = cli.SearchProcessDefinitionsLatest(cmd.Context(), filter)
		}
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("searching for process definitions to delete: %w", err))
		}
		if len(pds.Items) == 0 {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("no process definitions found to delete"))
		}
		prompt := fmt.Sprintf("You are about to delete %d process definition(s)?", len(pds.Items))
		if err := confirmCmdOrAbort(flagCmdAutoConfirm, prompt); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		_, err = cli.DeleteProcessDefinitions(cmd.Context(), filter, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("cancelling process instance(s): %w", err))
		}
	},
}

func init() {
	deleteCmd.AddCommand(deleteProcessDefinitionCmd)

	fs := deleteProcessDefinitionCmd.Flags()
	fs.StringVarP(&flagDeletePDKey, "key", "k", "", "process definition key to delete")
	fs.StringVarP(&flagDeletePDBpmnProcessId, "bpmn-process-id", "b", "", "BPMN process ID of the process definition (all versions) to delete")
	fs.Int32Var(&flagDeletePDProcessVersion, "pd-version", 0, "process definition version")
	fs.StringVar(&flagDeletePDProcessVersionTag, "pd-version-tag", "", "process definition version tag")
	fs.BoolVar(&flagDeletePDLatest, "latest", false, "fetch the latest version(s) of the given BPMN process(s)")
}
