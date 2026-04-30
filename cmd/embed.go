// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/spf13/cobra"
)

var embedCmd = &cobra.Command{
	Use:   "embed",
	Short: "Use bundled BPMN fixtures",
	Long: `Use bundled BPMN fixtures.

Use ` + "`embed list`" + ` to see bundled files, ` + "`embed deploy`" + ` to deploy fixtures, or
` + "`embed export`" + ` to inspect or edit files locally.`,
	Example: `  ./c8volt embed list
  ./c8volt embed deploy --all --run
  ./c8volt embed export --all --out ./fixtures`,
	Aliases: []string{"em", "emb"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"embedd", "embd", "embedded", "embeded"},
}

func init() {
	rootCmd.AddCommand(embedCmd)
}
