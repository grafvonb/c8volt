package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

var (
	flagDeletePDKeys              []string
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
		if cmd.Flags().Changed("workers") && flagWorkers < 1 {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("--workers must be positive integer"))
		}
		if len(flagDeletePDKeys) == 0 && flagDeletePDBpmnProcessId == "" {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("either --key or --bpmn-process-id must be provided to delete process definition(s)"))
		}
		keys := mergeAndValidateKeys(flagDeletePDKeys, log, cfg)

		switch {
		case len(flagDeletePDKeys) > 0:
		default:
			filter := process.ProcessDefinitionFilter{
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
			keys = make([]string, 0, len(pds.Items))
			for _, pd := range pds.Items {
				keys = append(keys, pd.Key)
			}
		}
		if len(keys) == 0 {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("no process definitions found to delete"))
		}
		prompt := fmt.Sprintf("You are about to delete %d process definition(s)?", len(keys))
		if err := confirmCmdOrAbort(flagCmdAutoConfirm, prompt); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		_, err = cli.DeleteProcessDefinitions(cmd.Context(), keys, flagWorkers, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("deleting process definiton(s): %w", err))
		}
	},
}

func init() {
	deleteCmd.AddCommand(deleteProcessDefinitionCmd)

	fs := deleteProcessDefinitionCmd.Flags()
	fs.StringSliceVarP(&flagDeletePDKeys, "key", "k", nil, "process definition key(s) to delete")
	fs.StringVarP(&flagDeletePDBpmnProcessId, "bpmn-process-id", "b", "", "BPMN process ID of the process definition (all versions) to delete")
	fs.Int32Var(&flagDeletePDProcessVersion, "pd-version", 0, "process definition version")
	fs.StringVar(&flagDeletePDProcessVersionTag, "pd-version-tag", "", "process definition version tag")
	fs.BoolVar(&flagDeletePDLatest, "latest", false, "fetch the latest version(s) of the given BPMN process(s)")

	fs.BoolVar(&flagForce, "force", false, "force cancellation of the process instance(s), prior to deletion")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")
}
