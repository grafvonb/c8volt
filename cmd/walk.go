// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/spf13/cobra"
)

var walkCmd = &cobra.Command{
	Use:   "walk",
	Short: "Inspect process-instance relationships",
	Long: `Inspect process-instance relationships.

Inspect ancestry, descendants, or a process-instance family around a key.`,
	Example: `  ./c8volt walk pi --key 2251799813711967
  ./c8volt walk pi --key 2251799813711967 --with-incidents
  ./c8volt walk pi --key 2251799813711967 --children`,
	Aliases: []string{"w", "traverse"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"walkk", "travers"},
}

func init() {
	rootCmd.AddCommand(walkCmd)

	setCommandMutation(walkCmd, CommandMutationReadOnly)
}
