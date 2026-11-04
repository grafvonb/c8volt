// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/spf13/cobra"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel running process instances",
	Long: `Cancel running process instances.

Validates the affected tree, prompts before destructive changes, and waits for
observed cancellation.`,
	Example: `  c8volt cancel process-instance --key <process-instance-key> --dry-run
  c8volt cancel process-instance --bpmn-process-id order --state active --dry-run`,
	Aliases: []string{"c", "cn", "stop", "abort"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"cancle", "cancl"},
}

func init() {
	rootCmd.AddCommand(cancelCmd)

	addBackoffFlagsAndBindings(cancelCmd)
	setCommandMutation(cancelCmd, CommandMutationStateChanging)
}
