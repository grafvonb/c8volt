package cmd

import (
	"fmt"

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
	Use:   "process-definition",
	Short: "Delete process definition resources",
	Long: "Delete process definition resources from Zeebe.\n\n" +
		"By default c8volt prompts before the destructive step. Without --allow-inconsistent, it prepares definitions for later manual cleanup instead of forcing inconsistent Operate state.\n\n" +
		"Use --auto-confirm for unattended destructive runs. Add --no-wait when accepted deletion work is enough for the current step, then verify later with `get pd`.",
	Example: `  ./c8volt delete pd --key <process-definition-key> --auto-confirm
  ./c8volt delete pd --bpmn-process-id C88_SimpleUserTask_Process --latest --force
  ./c8volt delete pd --bpmn-process-id C88_SimpleUserTask_Process --latest --allow-inconsistent --auto-confirm --no-wait
  ./c8volt get pd --bpmn-process-id C88_SimpleUserTask_Process --latest --json
  ./c8volt get pd --bpmn-process-id C88_SimpleUserTask_Process --latest --keys-only | ./c8volt delete pd --allow-inconsistent --auto-confirm --no-wait -`,
	Aliases: []string{"pd"},
	Args: func(cmd *cobra.Command, args []string) error {
		return validateOptionalDashArg(args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, err)
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if cmd.Flags().Changed("workers") && flagWorkers < 1 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("--workers must be positive integer"))
		}
		stdinKeys, err := readKeysIfDash(args) // only reads when args == []{"-"}
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(flagDeletePDKeys, stdinKeys, log, cfg)
		if len(keys) == 0 && flagDeletePDBpmnProcessId == "" {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, missingDependentFlagsf("either --key, stdin keys, or --bpmn-process-id must be provided to delete process definition(s)"))
		}

		switch {
		case len(keys) > 0:
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
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("searching for process definitions to delete: %w", err))
			}
			keys = make([]string, 0, len(pds.Items))
			for _, pd := range pds.Items {
				keys = append(keys, pd.Key)
			}
		}
		if len(keys) == 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no process definitions found to delete")))
		}

		fmt.Println("WARNING: This removes process-definition resources from Zeebe only. Operate history remains and must be cleaned up manually.")
		prompt := fmt.Sprintf("Delete %d process definition(s) from Zeebe?", len(keys))
		if !flagAllowInconsistent {
			fmt.Println("Without --allow-inconsistent, c8volt prepares deletion only (for example, cancels active instances).")
			prompt = fmt.Sprintf("Prepare %d process definition(s) for later manual deletion?", len(keys))
		}
		if err := confirmCmdOrAbort(shouldImplicitlyConfirm(cmd), prompt); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		reports, err := cli.DeleteProcessDefinitions(cmd.Context(), keys, flagWorkers, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("deleting process definition(s): %w", err))
		}
		if err := renderCommandResult(cmd, reports); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render delete result: %w", err))
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

	setCommandMutation(deleteProcessDefinitionCmd, CommandMutationStateChanging)
	setContractSupport(deleteProcessDefinitionCmd, ContractSupportFull)
	setAutomationSupport(deleteProcessDefinitionCmd, AutomationSupportFull, "supports unattended destructive confirmation")
}
