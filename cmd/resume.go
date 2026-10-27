// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

// resumeCmd groups exact-instance resumption and the existing bounded wait controls.
var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume suspended process instances",
	Long:  "Resume explicitly selected suspended process instances on Camunda 8.10, releasing suspended execution.",
	RunE:  func(cmd *cobra.Command, args []string) error { return cmd.Help() },
}

// init registers the verb without introducing a separate timing policy.
func init() {
	rootCmd.AddCommand(resumeCmd)
	addBackoffFlagsAndBindings(resumeCmd)
	setCommandMutation(resumeCmd, CommandMutationStateChanging)
}
