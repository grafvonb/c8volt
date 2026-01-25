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
	Args: func(cmd *cobra.Command, args []string) error {
		return validateOptionalDashArg(args)
	},
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

		stdinKeys, err := readKeysIfDash(args) // only reads when args == []{"-"}
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(flagDeletePDKeys, stdinKeys, log, cfg)

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

		fmt.Println("WARNING: Camunda's API v8.8+ deletion process removes process definition resources (from Zeebe) only; historic/Operate data remain.")
		fmt.Println("To avoid inconsistent data state c8volt cancels running instances (if any) to make definitions deletable, but final removal must be done manually in Operate.")
		prompt := fmt.Sprintf("You are about to delete %d process definition(s)?", len(keys))
		if !flagForce {
			fmt.Println("If you want to delete resource(s) from Zeebe without purging Operate data, please use --allow-inconsistent to confirm.")
			prompt = fmt.Sprintf("You are about to prepare %d process definition(s) for deletion in Operate?", len(keys))
		}
		if err := confirmCmdOrAbort(flagCmdAutoConfirm, prompt); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		_, err = cli.DeleteProcessDefinitions(cmd.Context(), keys, flagWorkers, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("deleting process definition(s): %w", err))
		}
	},
}

func init() {
	deleteCmd.AddCommand(deleteProcessDefinitionCmd)

	fs := deleteProcessDefinitionCmd.Flags()
	fs.BoolVar(&flagNoWait, "no-wait", false, "skip waiting for the deletion to be fully processed")
	fs.BoolVar(&flagNoStateCheck, "no-state-check", false, "skip checking the current state of the process instance(s) of the process definition before deleting it")
	fs.BoolVar(&flagAllowInconsistent, "allow-inconsistent", false, "allow deletion of process definitions even if their state will become inconsistent (not deleted from Operate's data)")
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
