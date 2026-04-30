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

Leaf commands validate scope, require confirmation for destructive steps, and
show verification examples.`,
	Example: `  ./c8volt delete pi --key 2251799813711967 --force
  ./c8volt delete pi --state completed --batch-size 200 --auto-confirm
  ./c8volt delete pd --bpmn-process-id C88_SimpleUserTask_Process --latest --auto-confirm`,
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
