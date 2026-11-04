// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect and validate c8volt configuration",
	Long: `Inspect configuration, print a starter template, or test the connection.

Settings resolve as: flag > environment > profile > base config > default.
Use --profile to select an environment.`,
	Example: `  c8volt config template
  c8volt config validate
  c8volt config test-connection`,
	Aliases: []string{"cfg"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"confige", "exepct"},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
