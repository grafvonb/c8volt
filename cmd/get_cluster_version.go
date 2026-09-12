// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var flagGetClusterVersionWithBrokers bool

var getClusterVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show connected cluster version",
	Long: `Get the connected Camunda gateway version.

Use --with-brokers to include broker versions.`,
	Example: `  ./c8volt get cluster version
  ./c8volt get cluster version --with-brokers
  ./c8volt get cluster version --json`,
	Args: cobra.NoArgs,
	Run:  runGetClusterVersion,
}

func init() {
	getClusterCmd.AddCommand(getClusterVersionCmd)

	fs := getClusterVersionCmd.Flags()
	fs.BoolVar(&flagGetClusterVersionWithBrokers, "with-brokers", false, "include broker versions")

	setCommandMutation(getClusterVersionCmd, CommandMutationReadOnly)
	setContractSupport(getClusterVersionCmd, ContractSupportFull)
	setOutputModes(getClusterVersionCmd,
		OutputModeContract{
			Name:      RenderModeOneLine.String(),
			Supported: true,
		},
		OutputModeContract{
			Name:             RenderModeJSON.String(),
			Supported:        true,
			MachinePreferred: true,
		},
	)
}

// runGetClusterVersion uses topology as the source of truth because Camunda exposes
// gateway and broker versions through the topology API.
func runGetClusterVersion(cmd *cobra.Command, args []string) {
	cli, log, cfg, err := NewCli(cmd)
	if err != nil {
		handleNewCliError(cmd, log, cfg, err)
	}
	log.Debug("getting cluster topology for version")
	topology, err := cli.GetClusterTopology(cmd.Context())
	if err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get cluster version: %w", err))
	}
	if pickMode() == RenderModeJSON {
		if err := renderJSONPayload(cmd, RenderModeJSON, newClusterVersionView(topology, flagGetClusterVersionWithBrokers)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render cluster version: %w", err))
		}
		return
	}
	if err := renderClusterVersion(cmd, topology, flagGetClusterVersionWithBrokers); err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render cluster version: %w", err))
	}
}
