// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

var getClusterTopologyCmd = &cobra.Command{
	Use:   "cluster-topology",
	Short: "Show connected cluster topology",
	Long: "Show connected cluster topology.\n\n" +
		"This legacy command reports brokers, partitions, and gateway metadata. Prefer `c8volt get cluster topology` for new usage.",
	Example: `  ./c8volt get cluster-topology
  ./c8volt get cluster topology --json`,
	Aliases: []string{"ct", "cluster-info", "ci"},
	Run:     runGetClusterTopology,
}

var getClusterTopologyNestedCmd = &cobra.Command{
	Use:   "topology",
	Short: "Show connected cluster topology",
	Long: "Show connected cluster topology.\n\n" +
		"This command reports brokers, partitions, and gateway metadata for the configured Camunda cluster.",
	Example: `  ./c8volt get cluster topology
  ./c8volt get cluster topology --json`,
	Run: runGetClusterTopology,
}

func init() {
	getCmd.AddCommand(getClusterTopologyCmd)
	getClusterCmd.AddCommand(getClusterTopologyNestedCmd)

	setCommandMutation(getClusterTopologyCmd, CommandMutationReadOnly)
	setContractSupport(getClusterTopologyCmd, ContractSupportLimited)
	setOutputModes(getClusterTopologyCmd,
		OutputModeContract{
			Name:             RenderModeJSON.String(),
			Supported:        true,
			MachinePreferred: true,
		},
	)

	setCommandMutation(getClusterTopologyNestedCmd, CommandMutationReadOnly)
	setContractSupport(getClusterTopologyNestedCmd, ContractSupportLimited)
	setOutputModes(getClusterTopologyNestedCmd,
		OutputModeContract{
			Name:             RenderModeJSON.String(),
			Supported:        true,
			MachinePreferred: true,
		},
	)
}

func runGetClusterTopology(cmd *cobra.Command, args []string) {
	cli, log, cfg, err := NewCli(cmd)
	if err != nil {
		handleNewCliError(cmd, log, cfg, err)
	}
	log.Debug("fetching cluster topology")
	topology, err := cli.GetClusterTopology(cmd.Context())
	if err != nil {
		ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("get cluster topology: %w", err))
	}
	if err := renderJSONPayload(cmd, RenderModeJSON, topology); err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render cluster topology: %w", err))
	}
}
