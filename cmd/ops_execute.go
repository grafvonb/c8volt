// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

var opsExecuteCmd = &cobra.Command{
	Use:   "execute",
	Short: "Discover predefined operational playbooks",
	Long: `Discover predefined operational playbooks.

The execute command group lists playbooks that discover target sets and execute
existing c8volt resource actions. This grouping command does not run concrete
operational workflows by itself.`,
	Example: `  ./c8volt ops execute --help
  ./c8volt ops execute retention-policy --retention-days 90 --dry-run
  ./c8volt capabilities --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"exec", "execution"},
}

func init() {
	opsCmd.AddCommand(opsExecuteCmd)

	setCommandMutation(opsExecuteCmd, CommandMutationStateChanging)
}
