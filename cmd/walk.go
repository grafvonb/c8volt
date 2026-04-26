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

Use this command family before cancellation or deletion when parent/child structure
matters. It is also useful after a run to see which instances were created around a key.`,
	Example: `  ./c8volt walk pi --key 2251799813711967 --family
  ./c8volt walk pi --key 2251799813711967 --family --tree
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
