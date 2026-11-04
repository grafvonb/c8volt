// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

// suspendCmd groups exact-instance suspension and the existing bounded wait controls.
var suspendCmd = &cobra.Command{
	Use:   "suspend",
	Short: "Suspend active process instances",
	Long:  `Pause process execution while retaining runtime state (only c8.10+).`,
	RunE:  func(cmd *cobra.Command, args []string) error { return cmd.Help() },
}

// init registers the verb without introducing a separate timing policy.
func init() {
	rootCmd.AddCommand(suspendCmd)
	addBackoffFlagsAndBindings(suspendCmd)
	setCommandMutation(suspendCmd, CommandMutationStateChanging)
}
