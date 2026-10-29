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
	Long: `Update variables, complete, assign or unassign explicitly selected user tasks on supported Camunda versions.

Use --assign <assignee> to assign every selected task to the same opaque backend identity, or --unassign to remove its assignee. Assignment explicitly prohibits replacement unless --allow-override is enabled; --auto-confirm and --automation never grant that permission. Assignment, unassignment and completion are mutually exclusive. Assignment and unassignment reject --vars, --vars-file and enabled --no-wait. Without an active mode selector, the variable-update workflow applies.

Assignment and unassignment accept repeated/comma-separated --key values and newline-separated stdin with positional '-'. Selection must be nonempty and contain valid 16-digit keys; duplicates retain first-input order. They perform no task reads, discovery, completion preparation or polling. --dry-run previews the frozen keys, action, identity and replacement permission locally with zero requests. Eligible interactive execution shows one confirmation followed by the result; bulk confirmation also lists the selected keys. Plan and tenant-scope details require --verbose. Auto-confirm and automation skip approval. --workers bounds native requests; --fail-fast stops new scheduling while preserving in-flight results.

Both operations await the documented native response, including blocking task listeners. Confirmed means that successful response was observed, without independent read-back or permanent ownership guarantees. Partial failures retain ordered per-key outcomes; timeouts or cancellation may leave acceptance unknown. Human plans, previews, summaries and verbose outcomes use the configured INFO logger. A single assignment blocked by a confirmed existing assignee reports one INFO hint to use --allow-override, while retaining the conflict exit code. Quiet or a WARN log level suppresses that hint. Other failures retain their error diagnostics. Quiet suppresses INFO but retains one compact ERROR for those failures. JSON emits one envelope, retaining partial evidence and original failure exits; keys-only prints preview keys or execution successes in selection order, omitting keys when none qualify. Quiet preserves machine output. JSON execution requires --auto-confirm or --automation and cannot combine with --verbose. HTTP diagnostics require DEBUG independently of verbose.

Use --complete to submit native completion for each eligible unique key, with optional --vars or --vars-file data. Omitted variables and {} do not suppress completion. Completion preserves JSON numbers exactly, including large integers and precise decimals. The backend's successful response confirms completion, including blocking task listeners. c8volt performs no additional task-state polling or downstream waiting; no assignment is performed. --no-wait skips variable-update verification and cannot bypass the completion request or its listener processing, so it is incompatible with --complete. A configured tenant does not filter explicit keys.

Before confirmation, c8volt reads each unique task once. Already completed tasks are skipped with a conflict result; any supplied variables are not applied. Human summaries omit zero counters; conflict errors explain that the task is already completed and no completion request was sent. Variables are mentioned only when supplied. Tasks with no assignee produce a warning and remain eligible. Lookup failures produce a warning and leave eligibility to the backend. There is no recheck after approval. Ordinary human approval shows selection scope, plan, warnings and confirmation in that order. A single-task prompt names the key; a bulk prompt names the eligible count. A single success says "updated user-task <key>: completion confirmed"; batches use "update user-task" with nonzero counts. Completion summaries and verbose details honor the configured log level; quiet suppresses them. Auto-confirm and automation skip the approval plan; quiet and machine output suppress its informational messages.

Completion accepts repeated or comma-separated --key values and newline-separated keys with positional '-'. Keys must contain exactly 16 decimal digits. --dry-run previews the frozen selection locally without requests or prompts. Ordinary execution confirms the eligible selection once, with no prompt when none can be submitted; --auto-confirm and --automation allow unattended execution. --workers bounds concurrency and --fail-fast stops new scheduling after a failure while retaining in-flight outcomes. With --fail-fast, finding an already completed task during preparation prevents all completion submissions. Without --fail-fast, completed tasks are skipped and eligible tasks continue.

Without an active --assign, --unassign or --complete selector, variable-update behavior is unchanged; false boolean selectors are inactive.

Provide repeated --key values or newline-separated keys from stdin with '-'. Supply exactly one payload source: --vars with a JSON object or --vars-file with its file path. The same variable map is applied to every unique key. Explicit keys use backend authorization without tenant filtering.

c8volt reads each task's complete effective variables, preserves existing local or inherited scopes, creates absent names at the task's element scope, freezes one deduplicated plan, and waits for the requested values at those scopes. Detailed previews label local writes as element:<key> and inherited writes as pi:<key> or element:<key> according to their owning scope.

Use --dry-run to inspect changes without mutation, or --auto-confirm for unattended updates. Use --no-wait to return after accepted scope writes without confirmation.`,
	Example: `  ./c8volt update user-task --key 2251799813685249 --assign alice --dry-run
  ./c8volt update user-task --key 2251799813685249,2251799813685250 --assign alice --allow-override --auto-confirm
  ./c8volt update user-task --key 2251799813685249 --unassign --auto-confirm
  printf '%s\n' "$USER_TASK_KEY_A" "$USER_TASK_KEY_B" | ./c8volt update user-task - --assign alice --workers 1 --automation --json
  ./c8volt update ut --key 2251799813685249 --complete
  ./c8volt update ut --key 2251799813685249,2251799813685250 --complete --vars '{"approved":true}' --dry-run
  ./c8volt update ut --key 2251799813685249 --complete --vars-file ./vars.json --auto-confirm
  printf '%s\n' "$USER_TASK_KEY_A" "$USER_TASK_KEY_B" | ./c8volt update ut - --complete --workers 1 --auto-confirm
  ./c8volt update user-task --key <user-task-key> --vars '{"approved":true}' --dry-run
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

	fs := updateUserTaskCmd.Flags()
	fs.StringSliceVarP(&flagUpdateUserTaskKeys, "key", "k", nil, "user task key(s) to update; repeat or combine with stdin '-'")
	trackUserTaskKeyInput(fs.Lookup("key"))
	fs.String("assign", "", "assign selected tasks to this backend identity; replacement is prohibited unless --allow-override is enabled")
	fs.Bool("unassign", false, "remove the assignee from selected tasks")
	fs.Bool("allow-override", false, "allow replacing existing assignees; requires --assign")
	fs.BoolVar(&flagUpdateUserTaskComplete, "complete", false, "complete selected tasks with optional variables; report backend-confirmed completion without additional polling")
	fs.StringVar(&flagUpdateUserTaskVars, "vars", "", "JSON object with variables to set or submit with each completion")
	fs.StringVar(&flagUpdateUserTaskVarsFile, "vars-file", "", "path to JSON object file with variables to set or submit with each completion")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview variable updates, completions or assignment changes without submitting mutations")
	fs.BoolVar(&flagNoWait, "no-wait", false, "skip variable-update verification after accepted scope writes; native task mutations have no additional polling, so incompatible with --complete, --assign or --unassign")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers for task mutations or variable scope updates (default: min(count, 2*GOMAXPROCS, 32))")
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
