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

Choose ` + "`embed list`" + ` to see what ships in the binary, ` + "`embed deploy`" + ` to
create a runnable test environment quickly, or ` + "`embed export`" + ` when you want to
inspect or edit the files locally.`,
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
