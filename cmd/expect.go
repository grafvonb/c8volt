// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/spf13/cobra"
)

var expectCmd = &cobra.Command{
	Use:   "expect",
	Short: "Wait for process instances to reach a state",
	Long: `Wait for process instances to reach a state.

Use this command family after run, cancel, or delete when success depends on an
observed process-instance state rather than an accepted request.`,
	Example: `  ./c8volt expect pi --key <process-instance-key> --state active
  ./c8volt expect pi --key <process-instance-key> --state absent
  ./c8volt get pi --key <process-instance-key> --keys-only | ./c8volt expect pi --state active -`,
	Aliases: []string{"e", "exp"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"expecte", "exepct"},
}

func init() {
	rootCmd.AddCommand(expectCmd)

	addBackoffFlagsAndBindings(expectCmd)
	setCommandMutation(expectCmd, CommandMutationReadOnly)
}
