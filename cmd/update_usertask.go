// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

var (
	flagUpdateUserTaskKeys     []string
	flagUpdateUserTaskVars     string
	flagUpdateUserTaskVarsFile string
)

var updateUserTaskCmd = &cobra.Command{
	Use:   "user-task [-]",
	Short: "Update user-task variables by key",
	Long: `Update effective user-task variables on Camunda 8.8 or newer.

Provide repeated --key values or newline-separated keys from stdin with '-'. Supply exactly one payload source: --vars with a JSON object or --vars-file with its file path. The same variable map is applied to every unique key. Explicit keys use backend authorization without tenant filtering.

c8volt reads each task's complete effective variables, preserves existing local or inherited scopes, creates absent names at the task's element scope, freezes one deduplicated plan, and waits for the requested values at those scopes.

Use --dry-run to inspect changes without mutation, or --auto-confirm for unattended updates. Use --no-wait to return after accepted scope writes without confirmation.`,
	Example: `  ./c8volt update user-task --key <user-task-key> --vars '{"approved":true}' --dry-run
  ./c8volt update ut --key <user-task-key> --vars-file ./vars.json --dry-run
  ./c8volt update uts --key <user-task-key-a> --key <user-task-key-b> --vars '{"approved":true}' --auto-confirm
  printf '%s\n' "$USER_TASK_KEY_A" "$USER_TASK_KEY_B" | ./c8volt update user-tasks - --vars '{"approved":true}' --dry-run
  ./c8volt --automation --json update ut --key <user-task-key> --vars '{"approved":true}'`,
	Aliases: []string{"user-tasks", "ut", "uts"},
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
		variables, err := parseUpdateVariablesFromFlags(cmd, flagUpdateUserTaskVars, flagUpdateUserTaskVarsFile)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		stdinKeys, err := readKeysIfDash(args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(cmd, flagUpdateUserTaskKeys, stdinKeys, log, cfg).Unique()
		if len(keys) == 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no user task keys provided or found to update")))
		}
		if err := validateUpdateUserTaskJSONConfirmation(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}

		plan, err := planUpdateUserTaskVariables(cmd, cli, keys, variables)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("plan user-task variable update: %w", err))
		}
		if flagDryRun {
			if err := renderUpdateUserTaskVariablePreview(cmd, plan); err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("render update dry-run result: %w", err))
			}
			return
		}
		if !userTaskVariablePlanHasChanges(plan) {
			if err := renderUpdateUserTaskVariablePlan(cmd, plan); err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("render update plan: %w", err))
			}
			return
		}
		if !shouldImplicitlyConfirm(cmd) {
			if err := renderUpdateUserTaskVariablePlan(cmd, plan); err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("render update plan: %w", err))
			}
			requestedUpdates := plan.VariableAddCount + plan.VariableChangeCount
			prompt := fmt.Sprintf("You are about to update %d requested variable value(s) on %d user task(s). Do you want to proceed?", requestedUpdates, plan.UpdateCount)
			if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), false, prompt); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
		}

		results, executeErr := cli.ExecuteUserTaskVariableUpdates(cmd.Context(), plan, flagWorkers, collectExplicitAdminInputOptions()...)
		if executeErr != nil {
			if len(results.Items) > 0 {
				if err := renderUpdateUserTaskVariableFailure(cmd, results, executeErr); err != nil {
					ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("render partial update result: %w", err))
				}
				exitAfterRenderedResult(cfg.App.NoErrCodes, executeErr)
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("update user-task variables: %w", executeErr))
		}
		if err := renderUpdateUserTaskVariableResults(cmd, results); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("render update result: %w", err))
		}
	},
}

func init() {
	updateCmd.AddCommand(updateUserTaskCmd)

	fs := updateUserTaskCmd.Flags()
	fs.StringSliceVarP(&flagUpdateUserTaskKeys, "key", "k", nil, "user task key(s) to update; repeat or combine with stdin '-'")
	fs.StringVar(&flagUpdateUserTaskVars, "vars", "", "JSON object with variables to set for each user task")
	fs.StringVar(&flagUpdateUserTaskVarsFile, "vars-file", "", "path to JSON object file with variables to set for each user task")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview variable updates without submitting mutation")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after scope update requests are accepted without variable confirmation")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when updating multiple user-task scopes (default: min(count, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued scope updates as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new scope updates after the first error")

	useInvalidInputFlagErrors(updateUserTaskCmd)
	setCommandMutation(updateUserTaskCmd, CommandMutationStateChanging)
	setContractSupport(updateUserTaskCmd, ContractSupportFull)
	setAutomationSupport(updateUserTaskCmd, AutomationSupportFull, "supports shared machine output, non-mutating dry-run previews, and accepted results")
	setOutputModes(updateUserTaskCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
		OutputModeContract{Name: RenderModeKeysOnly.String(), Supported: true},
	)
}
