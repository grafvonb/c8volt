// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

var getClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Inspect cluster-wide topology, version, and license information",
	Long:  `Inspect cluster topology, gateway and broker versions, or license information.`,
	Example: `  c8volt get cluster topology
  c8volt get cluster version
  c8volt get cluster license`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	getCmd.AddCommand(getClusterCmd)

	setCommandMutation(getClusterCmd, CommandMutationReadOnly)
}
