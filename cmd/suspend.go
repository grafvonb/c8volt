// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

// suspendCmd groups exact-instance suspension and the existing bounded wait controls.
var suspendCmd = &cobra.Command{
	Use:   "suspend",
	Short: "Suspend active process instances",
	Long:  "Suspend explicitly selected active process instances on Camunda 8.10, preserving runtime state and resources.",
	RunE:  func(cmd *cobra.Command, args []string) error { return cmd.Help() },
}

// init registers the verb without introducing a separate timing policy.
func init() {
	rootCmd.AddCommand(suspendCmd)
	addBackoffFlagsAndBindings(suspendCmd)
	setCommandMutation(suspendCmd, CommandMutationStateChanging)
}
