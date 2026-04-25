package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

var (
	flagExpectPIKeys   []string
	flagExpectPIStates []string
)

var expectProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Expect a process instance(s) to reach a certain state from list of states",
	Long: "Wait for process instance(s) to reach one of the requested states.\n\n" +
		"Use this read-only command after `run`, `cancel`, or `delete` when the operation returned before the " +
		"final state was visible, or when you need an explicit post-action assertion. The command waits until " +
		"each keyed process instance reaches one of the requested states or fails with a shared error model. " +
		"For cancellation waits, `canceled` is the user-facing intent state; on Camunda `8.8` and `8.9`, " +
		"that same outcome may be surfaced by the backend as `terminated`, and `c8volt` treats them as equivalent.\n\n" +
		"Default output stays human-oriented. Use --json when another tool needs the final wait report. " +
		"`--automation` remains unsupported because the broader waiting contract is not yet defined there.",
	Example: `  ./c8volt expect pi --key 2251799813685255 --state active
  ./c8volt expect pi --key 2251799813685255 --state completed --state absent
  ./c8volt expect pi --key 2251799813711967 --state canceled
  ./c8volt run pi --bpmn-process-id order-process --no-wait --json
  ./c8volt expect pi --key 2251799813711967 --state active
  ./c8volt get pi --bpmn-process-id order-process --keys-only | ./c8volt expect pi - --state terminated`,
	Aliases: []string{"pi"},
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
		states, err := process.ParseStates(flagExpectPIStates)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("error parsing states: %v; valid values are: %s", err, process.ValidStateStrings()))
		}

		stdinKeys, err := readKeysIfDash(args) // only reads when args == []{"-"}
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(flagExpectPIKeys, stdinKeys, log, cfg)
		if len(keys) == 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no process instance keys provided or found to watch")))
		}
		if !commandUsesSharedEnvelope(cmd, pickMode()) {
			logging.InfoOrVerbose(
				fmt.Sprintf("waiting for %d process instance(s) to reach one of %d desired state(s)", len(keys), len(states)),
				fmt.Sprintf("waiting for %d process instance(s) [%s] to reach one of the states [%s]", len(keys), keys, states),
				log,
				flagVerbose,
			)
		}
		reports, err := cli.WaitForProcessInstancesState(cmd.Context(), keys, states, flagWorkers, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("expecting process instance: %w", err))
		}
		if err := renderCommandResult(cmd, reports); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render expect result: %w", err))
		}
		if !commandUsesSharedEnvelope(cmd, pickMode()) {
			logging.InfoOrVerbose(
				fmt.Sprintf("%d process instance(s) reached desired state(s)", len(keys)),
				fmt.Sprintf("%d process instance(s) [%s] reached desired state(s) [%s]", len(keys), keys, states),
				log,
				flagVerbose,
			)
		}
	},
}

func init() {
	expectCmd.AddCommand(expectProcessInstanceCmd)

	fs := expectProcessInstanceCmd.Flags()
	fs.StringSliceVarP(&flagExpectPIKeys, "key", "k", nil, "process instance key(s) to expect a state for")
	_ = expectProcessInstanceCmd.MarkFlagRequired("key")
	fs.StringSliceVarP(&flagExpectPIStates, "state", "s", nil, "state of a process instance; valid values are: [active, completed, canceled, terminated, absent]. On Camunda 8.8/8.9, canceled waits also match terminated")
	_ = expectProcessInstanceCmd.MarkFlagRequired("state")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	setCommandMutation(expectProcessInstanceCmd, CommandMutationReadOnly)
	setContractSupport(expectProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(expectProcessInstanceCmd, AutomationSupportUnsupported, "waiting semantics are not yet defined for automation mode")
}
