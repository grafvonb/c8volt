// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

var opsCmd = &cobra.Command{
	Use:   "ops",
	Short: "Run operational playbooks",
	Long: `Run operational playbooks for analysis, retention, purge, repair, and cluster smoke testing.

Choose a subcommand for a specific workflow.`,
	Example: `  ./c8volt ops --help
  ./c8volt capabilities --json`,
	Aliases: []string{"operations"},
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"op", "operation"},
}

func init() {
	rootCmd.AddCommand(opsCmd)

	addBackoffFlagsAndBindings(opsCmd)
	setCommandMutation(opsCmd, CommandMutationStateChanging)
}
