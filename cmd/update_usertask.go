// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

var (
	flagUpdateUserTaskComplete bool
	flagUpdateUserTaskKeys     []string
	flagUpdateUserTaskVars     string
	flagUpdateUserTaskVarsFile string
)

var updateUserTaskCmd = &cobra.Command{
	Use:   "user-task [-]",
	Short: "Update variables, complete, assign or unassign user tasks by key",
	Long: `Update variables, complete, assign, or unassign user tasks selected by key.

Use --set-vars or --set-vars-file for variables. Updates preserve the owning scope
of existing variables; new names are written to the task's element scope.
Use --complete with optional variables to complete a task; the backend response
confirms completion. Already completed tasks are skipped with a conflict.

Use --set-assignee to assign a task or --unassign to remove its assignee.
Replacing another assignee requires approval or --force; --auto-confirm alone
does not authorize replacement. Assignment modes cannot include variables.

Use --dry-run to preview. Completion and assignment await the backend response
and cannot be combined with --no-wait.`,
	Example: `  c8volt update user-task --key <user-task-key> --set-vars-file vars.json --dry-run
  c8volt update user-task --key <user-task-key> --complete --set-vars '{"approved":true}'
  c8volt update user-task --key <user-task-key> --set-assignee alice --dry-run`,
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
		mode, err := resolveUpdateUserTaskAssignmentMode(cmd)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			return
		}
		if mode != "" {
			runUpdateUserTaskAssignment(cmd, args, cli, log, cfg)
			return
		}
		if flagUpdateUserTaskComplete {
			runUpdateUserTaskCompletion(cmd, args, cli, log, cfg)
			return
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

// init registers user-task update flags and declares mutation, automation, and output
// capabilities.
func init() {
	updateCmd.AddCommand(updateUserTaskCmd)

	setFlagMigrations(updateUserTaskCmd, map[string]flagMigration{
		"vars":      {Replacement: "--set-vars", TakesValue: true},
		"vars-file": {Replacement: "--set-vars-file", TakesValue: true},
		"assign":    {Replacement: "--set-assignee", TakesValue: true},
	})
	fs := updateUserTaskCmd.Flags()
	fs.StringSliceVarP(&flagUpdateUserTaskKeys, "key", "k", nil, "user task key(s) to update; repeat or combine with stdin '-'")
	trackUserTaskKeyInput(fs.Lookup("key"))
	fs.String("set-assignee", "", "assign selected tasks to this backend identity; ask before replacement, or use --force")
	fs.Bool("unassign", false, "remove the assignee from selected tasks")
	fs.Bool("force", false, "replace existing assignees without confirmation; requires --set-assignee")
	fs.BoolVar(&flagUpdateUserTaskComplete, "complete", false, "complete selected tasks with optional variables; await backend confirmation")
	fs.StringVar(&flagUpdateUserTaskVars, "set-vars", "", "JSON object with variables to set or submit with each completion")
	fs.StringVar(&flagUpdateUserTaskVarsFile, "set-vars-file", "", "path to JSON object file with variables to set or submit with each completion")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview variable updates, completions or assignment changes without submitting mutations")
	fs.BoolVar(&flagNoWait, "no-wait", false, "skip variable-update verification; incompatible with --complete, --set-assignee or --unassign")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers; omit for an automatic limit")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued task mutations or scope updates as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new task mutations or scope updates after the first error")

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
