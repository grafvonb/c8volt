// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete process instances or definitions",
	Long: `Delete process instances or process definitions.

Choose a resource command to validate its scope and confirm deletion.`,
	Example: `  c8volt delete process-instance --key <process-instance-key> --dry-run
  c8volt delete process-definition --bpmn-process-id order --dry-run`,
	Aliases: []string{"d", "del", "remove", "rm"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"deelte", "delet"},
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	addBackoffFlagsAndBindings(deleteCmd)
	setCommandMutation(deleteCmd, CommandMutationStateChanging)
}
