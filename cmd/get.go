// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Inspect cluster, process, job, element, incident, tenant, and resource state",
	Long: `Inspect cluster health, deployed definitions, process instances, tasks, jobs,
elements, and incidents without changing them.

Choose a resource command to look up a key or search with filters.`,
	Example: `  c8volt get cluster topology
  c8volt get process-instance --state active --limit 10
  c8volt get incident --state active --limit 10`,
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
