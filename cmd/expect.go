// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/spf13/cobra"
)

var expectCmd = &cobra.Command{
	Use:   "expect",
	Short: "Wait for process instances to satisfy expectations",
	Long: `Wait for process instances to satisfy state or incident expectations.

Use after run, cancel, or delete when success depends on an observed
process-instance state or incident marker.`,
	Example: `  ./c8volt expect pi --key <process-instance-key> --state active
  ./c8volt expect pi --key <process-instance-key> --incident true
  ./c8volt expect pi --key <process-instance-key> --state active --incident false
  ./c8volt expect pi --key <process-instance-key> --state absent
  ./c8volt get pi --key <process-instance-key> --keys-only | ./c8volt expect pi --incident true -`,
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
