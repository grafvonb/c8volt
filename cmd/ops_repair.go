// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

var opsRepairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Discover repair and remediation workflows",
	Long: `Discover repair and remediation workflows.

The repair command group is reserved for future workflows that repair
operational issues through target-specific subcommands. This grouping command
does not define target keys or run remediation behavior by itself.
Target-specific subcommands will define their own target semantics as they are
added.`,
	Example: `  ./c8volt ops repair --help
  ./c8volt capabilities --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"fix", "remediate", "remediation"},
}

func init() {
	opsCmd.AddCommand(opsRepairCmd)

	setCommandMutation(opsRepairCmd, CommandMutationStateChanging)
}
