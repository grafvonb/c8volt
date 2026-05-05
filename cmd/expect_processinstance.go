// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

var (
	flagExpectPIKeys     []string
	flagExpectPIStates   []string
	flagExpectPIIncident string
)

var expectProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Wait for process instances to reach states",
	Long: "Wait for process instances to reach one of the requested states.\n\n" +
		"Use after `run`, `cancel`, or `delete` when a command returns before the final state is visible.\n\n" +
		"On Camunda 8.8/8.9, canceled waits also match terminated.",
	Example: `  ./c8volt expect pi --key <process-instance-key> --state active
  ./c8volt expect pi --key <process-instance-key> --state completed --state absent
  ./c8volt expect pi --key <process-instance-key> --state canceled
  ./c8volt get pi --key <process-instance-key> --keys-only | ./c8volt expect pi --state active -`,
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
		incident, incidentSet, err := parseExpectPIIncidentFlag(cmd)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		expectation := process.ProcessInstanceExpectationRequest{States: states}
		if incidentSet {
			expectation.Incident = &incident
		}
		if !expectation.HasExpectations() {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("at least one process instance expectation flag is required: --state or --incident")))
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
			if incidentSet {
				logging.InfoOrVerbose(
					fmt.Sprintf("waiting for %d process instance(s) to satisfy expectation(s)", len(keys)),
					fmt.Sprintf("waiting for %d process instance(s) [%s] to satisfy expectation(s)", len(keys), keys),
					log,
					flagVerbose,
				)
			} else {
				logging.InfoOrVerbose(
					fmt.Sprintf("waiting for %d process instance(s) to reach one of %d desired state(s)", len(keys), len(states)),
					fmt.Sprintf("waiting for %d process instance(s) [%s] to reach one of the states [%s]", len(keys), keys, states),
					log,
					flagVerbose,
				)
			}
		}
		if incidentSet {
			reports, err := cli.WaitForProcessInstancesExpectation(cmd.Context(), keys, expectation, flagWorkers, collectOptions()...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("expecting process instance: %w", err))
			}
			if err := renderCommandResult(cmd, reports); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render expect result: %w", err))
			}
			if !commandUsesSharedEnvelope(cmd, pickMode()) {
				logging.InfoOrVerbose(
					fmt.Sprintf("%d process instance(s) satisfied expectation(s)", len(keys)),
					fmt.Sprintf("%d process instance(s) [%s] satisfied expectation(s)", len(keys), keys),
					log,
					flagVerbose,
				)
			}
			return
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
	fs.StringSliceVarP(&flagExpectPIKeys, "key", "k", nil, "process instance key(s) to watch")
	fs.StringSliceVarP(&flagExpectPIStates, "state", "s", nil, "state of a process instance; valid values are: [active, completed, canceled, terminated, absent]. On Camunda 8.8/8.9, canceled waits also match terminated")
	fs.StringVar(&flagExpectPIIncident, "incident", "", "incident expectation; valid values are: [true, false]")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	setCommandMutation(expectProcessInstanceCmd, CommandMutationReadOnly)
	setContractSupport(expectProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(expectProcessInstanceCmd, AutomationSupportUnsupported, "automation mode is not supported for wait commands")
}

func parseExpectPIIncidentFlag(cmd *cobra.Command) (process.IncidentExpectation, bool, error) {
	if cmd == nil || !cmd.Flags().Changed("incident") {
		return false, false, nil
	}
	incident, err := process.ParseIncidentExpectation(flagExpectPIIncident)
	if err != nil {
		return false, true, invalidFlagValuef("invalid value for --incident: %q; valid values are: %v", flagExpectPIIncident, process.ValidIncidentExpectationStrings())
	}
	return incident, true, nil
}
