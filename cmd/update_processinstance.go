// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	flagUpdatePIKeys []string
	flagUpdatePIVars string
)

var updateProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Update process-instance variables by key",
	Long: "Update process-instance variables by key.\n\n" +
		"The command accepts repeated --key values or newline-separated keys from stdin with '-'. The --vars flag must be a JSON object and the same variable map is applied to every unique target key.\n\n" +
		"By default c8volt waits until requested process-instance-scope variables are visible through the same lookup path as `get pi --with-vars`; add --no-wait to return after the update request is accepted.\n\n" +
		"Variable updates are supported for Camunda 8.8 and 8.9. Camunda 8.7 returns an unsupported-version error before mutation.",
	Example: `  ./c8volt update pi --key 2251799813711967 --vars '{"customerTier":"gold"}'
  ./c8volt update process-instance --key 2251799813711967 --vars '{"customerTier":"gold"}'
  ./c8volt update pi --key 2251799813711967 --key 2251799813711968 --vars '{"customerTier":"gold"}'
  printf '%s\n' 2251799813711967 2251799813711968 | ./c8volt update pi - --vars '{"customerTier":"gold"}'
  printf '%s\n' 2251799813711967 | ./c8volt update pi --key 2251799813711968 - --vars '{"customerTier":"gold"}'
  ./c8volt --json update pi --key 2251799813711967 --vars '{"customerTier":"gold"}' --no-wait`,
	Aliases: []string{"pi"},
	Args: func(cmd *cobra.Command, args []string) error {
		return validateOptionalDashArg(args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if cmd.Flags().Changed("workers") && flagWorkers < 1 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("--workers must be positive integer"))
		}
		variables, err := parseUpdateProcessInstanceVariables(flagUpdatePIVars)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		stdinKeys, err := readKeysIfDash(args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(flagUpdatePIKeys, stdinKeys, log, cfg).Unique()
		if len(keys) == 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no process instance keys provided or found to update")))
		}
		results, err := cli.UpdateProcessInstancesVariables(cmd.Context(), keys, variables, flagWorkers, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("update process-instance variables: %w", err))
		}
		if err := renderUpdateProcessInstanceVariableResults(cmd, results); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render update result: %w", err))
		}
	},
}

func init() {
	updateCmd.AddCommand(updateProcessInstanceCmd)

	fs := updateProcessInstanceCmd.Flags()
	fs.StringSliceVar(&flagUpdatePIKeys, "key", nil, "process instance key(s) to update; repeat or combine with stdin '-'")
	fs.StringVar(&flagUpdatePIVars, "vars", "", "JSON object with variables to set on each process instance")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after the update request is accepted without variable confirmation")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when updating multiple process instances (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new updates after the first error")

	setFlagContractRequired(updateProcessInstanceCmd, "vars")
	setCommandMutation(updateProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(updateProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(updateProcessInstanceCmd, AutomationSupportFull, "supports shared machine output and accepted results with --no-wait")
}

// parseUpdateProcessInstanceVariables decodes the --vars JSON object used for process-instance updates.
func parseUpdateProcessInstanceVariables(raw string) (map[string]any, error) {
	if raw == "" {
		return nil, invalidFlagValuef("--vars is required and must be a JSON object")
	}
	var variables map[string]any
	if err := json.Unmarshal([]byte(raw), &variables); err != nil {
		return nil, invalidFlagValuef("--vars must be a valid JSON object: %v", err)
	}
	if variables == nil {
		return nil, invalidFlagValuef("--vars must be a JSON object")
	}
	return variables, nil
}
