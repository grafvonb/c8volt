// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update existing resources",
	Long: `Change variables, user-task assignment or completion, and job retries or outcomes.

Use --set-* for values to write. Preview with --dry-run where supported, then
confirm the prepared plan. Choose a resource command for its supported actions.`,
	Example: `  c8volt update process-instance --key <process-instance-key> --set-vars-file vars.json --dry-run
  c8volt update job --key <job-key> --set-retries 3 --dry-run
  c8volt update user-task --key <user-task-key> --set-assignee alice --dry-run`,
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
