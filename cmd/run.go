// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start process instances",
	Long: `Start process instances.

The process-instance command waits for active instances by default.`,
	Example: `  ./c8volt run pi -b C88_SimpleUserTask_Process
  ./c8volt run pi -b C88_SimpleUserTask_Process --vars '{"customerId":"1234"}'
  ./c8volt run pi -b C88_SimpleUserTask_Process --no-wait`,
	Aliases: []string{"r"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"rum", "runn", "execute"},
}

func init() {
	rootCmd.AddCommand(runCmd)

	addBackoffFlagsAndBindings(runCmd)
	setCommandMutation(runCmd, CommandMutationStateChanging)
}
