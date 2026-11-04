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
	Short: "Wait for process instances to satisfy expectations",
	Long: `Wait for selected process instances to satisfy state or incident expectations.

Use after a mutation when an observed outcome matters. Repeat --state to accept
any listed state. Suspended-state expectations are available only c8.10+.`,
	Example: `  c8volt expect process-instance --key <process-instance-key> --state completed
  c8volt expect process-instance --key <process-instance-key> --incident false`,
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
		var states process.States
		if cmd.Flags().Changed("state") {
			states, err = process.ParseStates(flagExpectPIStates)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("error parsing states: %v; valid values are: %s; suspended requires Camunda 8.10", err, process.ValidStateStrings()))
			}
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
		keys := mergeAndValidateKeys(cmd, flagExpectPIKeys, stdinKeys, log, cfg)
		if len(keys) == 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no process instance keys provided or found to watch")))
		}
		if !commandUsesSharedEnvelope(cmd, pickMode()) {
			if incidentSet {
				expectationLabel := processInstanceExpectationLogLabel(len(states) > 0)
				logging.InfoOrVerbose(
					fmt.Sprintf("waiting for %d pi; expectations %s", len(keys), expectationLabel),
					fmt.Sprintf("waiting for %d pi [%s]; expectations %s", len(keys), keys, expectationLabel),
					log,
					flagVerbose,
				)
			} else {
				logging.InfoOrVerbose(
					fmt.Sprintf("waiting for %d pi; states %d", len(keys), len(states)),
					fmt.Sprintf("waiting for %d pi [%s]; states [%s]", len(keys), keys, states),
					log,
					flagVerbose,
				)
			}
		}
		expectOpts := collectExplicitPIAdminInputOptions()
		expectProgress := newExpectProcessInstanceSemanticProgress(cmd, len(keys))
		expectOpts = appendExpectProcessInstanceProgressOption(expectOpts, expectProgress)
		if incidentSet {
			reports, err := cli.WaitForProcessInstancesExpectation(cmd.Context(), keys, expectation, flagWorkers, expectOpts...)
			expectProgress.Close()
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("expecting process instance: %w", err))
			}
			if err := renderCommandResult(cmd, reports); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render expect result: %w", err))
			}
			if !commandUsesSharedEnvelope(cmd, pickMode()) {
				expectationLabel := processInstanceExpectationLogLabel(len(states) > 0)
				logging.InfoOrVerbose(
					fmt.Sprintf("%d pi satisfied; expectations %s", len(keys), expectationLabel),
					fmt.Sprintf("%d pi [%s] satisfied; expectations %s", len(keys), keys, expectationLabel),
					log,
					flagVerbose,
				)
			}
			return
		}
		reports, err := cli.WaitForProcessInstancesState(cmd.Context(), keys, states, flagWorkers, expectOpts...)
		expectProgress.Close()
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("expecting process instance: %w", err))
		}
		if err := renderCommandResult(cmd, reports); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render expect result: %w", err))
		}
		if !commandUsesSharedEnvelope(cmd, pickMode()) {
			logging.InfoOrVerbose(
				fmt.Sprintf("%d pi reached states", len(keys)),
				fmt.Sprintf("%d pi [%s] reached states [%s]", len(keys), keys, states),
				log,
				flagVerbose,
			)
		}
	},
}

// init registers expectation flags and bootstrap-free state completion.
func init() {
	expectCmd.AddCommand(expectProcessInstanceCmd)

	fs := expectProcessInstanceCmd.Flags()
	fs.StringSliceVarP(&flagExpectPIKeys, "key", "k", nil, "process instance key(s) to watch")
	fs.StringSliceVarP(&flagExpectPIStates, "state", "s", nil, "state expectation; valid values are: [active, suspended, completed, canceled, terminated, absent]; suspended: only c8.10+")
	_ = expectProcessInstanceCmd.RegisterFlagCompletionFunc("state", processInstanceStateCompletion(true))
	fs.StringVar(&flagExpectPIIncident, "incident", "", "incident expectation; valid values are: [true, false]")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers; omit for an automatic limit")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
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

func processInstanceExpectationLogLabel(hasState bool) string {
	if hasState {
		return "state and incident expectations"
	}
	return "incident expectation"
}
