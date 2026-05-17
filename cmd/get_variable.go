// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/spf13/cobra"
)

//nolint:unused
var getVariableCmd = &cobra.Command{
	Use:     "variable",
	Short:   "Get a variable by its name from a process instance",
	Long:    "Get a variable by name from a process instance.",
	Example: "  ./c8volt get variable --help",
	Aliases: []string{"var"},
	Run: func(cmd *cobra.Command, args []string) {
		_, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, err)
		}

		log.Info("get variable by name not implemented")
	},
}

func init() {
	// getCmd.AddCommand(getVariableCmd)
}
