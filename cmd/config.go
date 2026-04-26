// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect and validate c8volt configuration",
	Long: `Inspect and validate c8volt configuration.

Use ` + "`config show`" + ` to see the effective settings c8volt will use after flags,
environment variables, profiles, config files, and defaults are resolved. Generate a
template first when setting up a new environment.`,
	Example: `  ./c8volt config show
  ./c8volt config show --template
  ./c8volt --config ./config.yaml config show --validate
  ./c8volt --profile prod config show`,
	Aliases: []string{"cfg"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"confige", "exepct"},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
