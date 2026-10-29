// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update existing resources",
	Long: `Update element-local, process-instance or user-task variables, complete, assign or unassign user tasks, or update job retries, timeouts, and worker outcomes.

Element updates freeze exact local destinations before approval and confirm requested values unless --no-wait is selected. Dry-run and completed no-op plans never submit mutations.

User-task --complete accepts explicit keys and optional variables and previews locally with --dry-run. One lookup per key detects already completed tasks (conflict) and warns about missing assignees before confirmation. The backend's successful response confirms completion; c8volt adds no task-state polling or downstream waiting. For user tasks, --no-wait skips variable-update verification and is incompatible with --complete, which has no additional waiter to skip.

User-task --assign <assignee> checks ownership and asks before replacing existing assignees. Use --force to permit replacement without questions. Auto-confirm and automation do not grant replacement permission. --unassign removes explicit task assignments without preliminary reads. Neither mode polls. Dry-run previews intent locally; ordinary execution approves once and awaits the native response, including blocking listeners. Partial and uncertain outcomes remain visible. These modes reject variable payloads, each other, --complete and enabled --no-wait.

Requires Camunda 8.8 or newer. Use a resource subcommand to plan and confirm updates.`,
	Example: `  ./c8volt update element --key <element-key> --vars '{"approved":true}' --dry-run
  ./c8volt update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run
  ./c8volt update process-instance --key <process-instance-key> --vars-file ./vars.json --dry-run
  ./c8volt update user-task --key <user-task-key> --vars '{"approved":true}' --dry-run
  ./c8volt update user-task --key <user-task-key> --vars-file ./vars.json --auto-confirm
  ./c8volt update user-task --key 2251799813685249 --assign alice --dry-run
  ./c8volt update user-task --key 2251799813685249 --assign alice --force
  ./c8volt update user-task --key 2251799813685249 --unassign --auto-confirm
  ./c8volt update ut --key 2251799813685249 --complete --auto-confirm
  ./c8volt update job --key <job-key> --retries 3 --dry-run
  ./c8volt update job --key <job-key> --timeout 5m --auto-confirm
  printf '%s\n' "$PROCESS_INSTANCE_KEY_A" "$PROCESS_INSTANCE_KEY_B" | ./c8volt update process-instance - --vars '{"customerTier":"gold"}' --dry-run
  printf '%s\n' "$USER_TASK_KEY_A" "$USER_TASK_KEY_B" | ./c8volt update user-task - --vars '{"approved":true}' --dry-run
  ./c8volt --automation --json update process-instance --key <process-instance-key> --vars '{"customerTier":"gold"}' --dry-run`,
	Aliases: []string{"u"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"updte", "set"},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	addBackoffFlagsAndBindings(updateCmd)
	setCommandMutation(updateCmd, CommandMutationStateChanging)
}
