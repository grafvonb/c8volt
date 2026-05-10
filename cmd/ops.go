// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

var opsCmd = &cobra.Command{
	Use:   "ops",
	Short: "Discover high-level operational workflows",
	Long: `Discover high-level operational workflows.

The ops command family groups operational playbooks for execution, repair, and
future maintenance workflows. This root command is intentionally discovery-only;
target-specific subcommands define concrete behavior.`,
	Example: `  ./c8volt ops --help
  ./c8volt ops execute --help
  ./c8volt ops repair --help`,
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
