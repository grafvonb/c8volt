// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start process instances",
	Long: `Start process instances and wait until they are visible.

Creation confirmation does not mean the process has completed. Use expect
process-instance when execution must reach a particular state.`,
	Example: `  c8volt run process-instance --bpmn-process-id order
  c8volt run process-instance --bpmn-process-id order --set-vars '{"customerId":"1234"}'`,
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
