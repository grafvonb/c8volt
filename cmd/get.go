// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Inspect cluster, process, tenant, and resource state",
	Long: `Inspect cluster, process, tenant, and resource state without changing it.

Check cluster health, list deployed process definitions, inspect process
instances, list visible tenants, or fetch a known resource.`,
	Example: `  ./c8volt get cluster topology
  ./c8volt get pd --latest
  ./c8volt get pi --state active
  ./c8volt get tenant
  ./c8volt get resource --id <resource-key>`,
	Aliases: []string{"g", "read"},
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"gett", "getr"},
}

func init() {
	rootCmd.AddCommand(getCmd)

	addBackoffFlagsAndBindings(getCmd)
	setCommandMutation(getCmd, CommandMutationReadOnly)
}
