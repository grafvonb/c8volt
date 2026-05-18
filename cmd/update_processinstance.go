// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

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
		"By default c8volt loads current process-instance-scope variables, previews planned additions and changes, asks for confirmation, then waits until requested variables are visible through the same lookup path as `get pi --with-vars`. Use --dry-run to preview without mutating, or --auto-confirm for unattended mutation.\n\n" +
		"Variable updates are supported for Camunda 8.8 and 8.9. Camunda 8.7 returns an unsupported-version error before mutation.",
	Example: `  ./c8volt update pi --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run
  ./c8volt update pi --key <process-instance-key> --vars-file ./vars.json --dry-run
  ./c8volt update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run
  ./c8volt update pi --key <process-instance-key-a> --key <process-instance-key-b> --vars '{"customerTier":"gold"}' --dry-run
  printf '%s\n' "$PROCESS_INSTANCE_KEY_A" "$PROCESS_INSTANCE_KEY_B" | ./c8volt update pi - --vars '{"customerTier":"gold"}' --dry-run
  ./c8volt --json update pi --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run`,
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
		if err := validateUpdateProcessInstanceJSONConfirmation(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		preview, err := planUpdateProcessInstanceVariables(cmd.Context(), cmd, cli, keys, variables)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("plan process-instance variable update: %w", err))
		}
		if flagDryRun {
			if err := renderUpdateProcessInstanceVariablePreview(cmd, preview); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render update dry-run result: %w", err))
			}
			return
		}
		if !preview.HasPlannedChanges() {
			if err := renderUpdateProcessInstanceVariablePlan(cmd, preview); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render update plan: %w", err))
			}
			return
		}
		if !shouldImplicitlyConfirm(cmd) {
			if err := renderUpdateProcessInstanceVariablePlan(cmd, preview); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render update plan: %w", err))
			}
			requestedUpdates := preview.VariableAddCount + preview.VariableChangeCount
			prompt := fmt.Sprintf("You are about to update %d requested variable value(s) on %d process instance(s). Do you want to proceed?", requestedUpdates, preview.UpdateCount)
			if err := confirmCmdOrAbortFn(false, prompt); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
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
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview variable updates without submitting mutation")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after the update request is accepted without variable confirmation")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when updating multiple process instances (default: min(count, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new updates after the first error")

	useInvalidInputFlagErrors(updateProcessInstanceCmd)
	setCommandMutation(updateProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(updateProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(updateProcessInstanceCmd, AutomationSupportFull, "supports shared machine output, non-mutating dry-run previews, and accepted results")
}
