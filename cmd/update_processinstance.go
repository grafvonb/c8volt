// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagUpdatePIKeys     []string
	flagUpdatePIVars     string
	flagUpdatePIVarsFile string
)

var updateProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Update process-instance variables by key",
	Long: "Update process-instance variables by key.\n\n" +
		"The command accepts repeated --key values or newline-separated keys from stdin with '-'. Provide exactly one variable payload source: --vars with a JSON object or --vars-file with a path to a JSON object file. The same variable map is applied to every unique target key.\n\n" +
		"By default c8volt waits until requested process-instance-scope variables are visible through the same lookup path as `get pi --with-vars`; add --no-wait to return after the update request is accepted.\n\n" +
		"Variable updates are supported for Camunda 8.8 and 8.9. Camunda 8.7 returns an unsupported-version error before mutation.",
	Example: `  ./c8volt update pi --key 2251799813711967 --vars '{"customerTier":"gold"}'
  ./c8volt update pi --key 2251799813711967 --vars-file ./vars.json
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
		variables, err := parseUpdateProcessInstanceVariablesFromFlags(cmd, flagUpdatePIVars, flagUpdatePIVarsFile)
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
	fs.StringVar(&flagUpdatePIVarsFile, "vars-file", "", "path to JSON object file with variables to set on each process instance")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after the update request is accepted without variable confirmation")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when updating multiple process instances (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new updates after the first error")

	useInvalidInputFlagErrors(updateProcessInstanceCmd)
	setCommandMutation(updateProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(updateProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(updateProcessInstanceCmd, AutomationSupportFull, "supports shared machine output and accepted results with --no-wait")
}

// parseUpdateProcessInstanceVariablesFromFlags selects exactly one variable payload source and decodes it.
func parseUpdateProcessInstanceVariablesFromFlags(cmd *cobra.Command, raw string, filePath string) (map[string]any, error) {
	varsChanged := cmd.Flags().Changed("vars")
	varsFileChanged := cmd.Flags().Changed("vars-file")
	if varsChanged && varsFileChanged {
		return nil, mutuallyExclusiveFlagsf("--vars cannot be combined with --vars-file")
	}
	if varsFileChanged {
		if filePath == "" {
			return nil, invalidFlagValuef("--vars-file requires a file path")
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, invalidFlagValuef("--vars-file could not be read: %v", err)
		}
		return parseUpdateProcessInstanceVariables(string(data), "--vars-file")
	}
	return parseUpdateProcessInstanceVariables(raw, "--vars")
}

// parseUpdateProcessInstanceVariables decodes the --vars JSON object used for process-instance updates.
func parseUpdateProcessInstanceVariables(raw string, source string) (map[string]any, error) {
	if raw == "" {
		return nil, invalidFlagValuef("--vars or --vars-file is required and must be a JSON object")
	}
	var variables map[string]any
	if err := json.Unmarshal([]byte(raw), &variables); err != nil {
		return nil, invalidFlagValuef("%s must be a valid JSON object: %v", source, err)
	}
	if variables == nil {
		return nil, invalidFlagValuef("%s must be a JSON object", source)
	}
	return variables, nil
}
