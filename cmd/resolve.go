// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

var resolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Resolve operational incidents",
	Long: `Resolve operational incidents.

The incident command resolves known incident keys and reports each target
independently. Resolution is state-changing and waits for confirmation by
default unless a leaf command supports an explicit opt-out.`,
	Example: `  ./c8volt resolve incident --key 2251799813685249
  ./c8volt resolve inc --key 2251799813685249 --key 2251799813685250
  printf '%s\n' 2251799813685249 2251799813685250 | ./c8volt resolve inc -`,
	Aliases: []string{"res"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"reslove", "resovle"},
}

func init() {
	rootCmd.AddCommand(resolveCmd)

	addBackoffFlagsAndBindings(resolveCmd)
	setCommandMutation(resolveCmd, CommandMutationStateChanging)
}
