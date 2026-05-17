// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

var opsPurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Discover destructive operational cleanup workflows",
	Long: `Discover destructive operational cleanup workflows.

The purge command group is reserved for workflows that remove operational
targets through target-specific subcommands. This grouping command only shows
available purge workflows and never performs cleanup by itself.`,
	Example: `  ./c8volt ops purge --help
  ./c8volt ops purge orphan-process-instances --dry-run
  ./c8volt ops purge orphan-process-instances --state completed --limit 25 --auto-confirm --report-file orphan-purge.md`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"delete", "cleanup"},
}

func init() {
	opsCmd.AddCommand(opsPurgeCmd)

	setCommandMutation(opsPurgeCmd, CommandMutationStateChanging)
}
