// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

var flagGetClusterVersionWithBrokers bool

var getClusterVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show connected cluster version",
	Long: "Show connected cluster version.\n\n" +
		"This command prints the gateway version by default. Use --with-brokers to include broker versions sorted by broker node id.",
	Example: `  ./c8volt get cluster version
  ./c8volt get cluster version --with-brokers`,
	Args: cobra.NoArgs,
	Run:  runGetClusterVersion,
}

func init() {
	getClusterCmd.AddCommand(getClusterVersionCmd)

	fs := getClusterVersionCmd.Flags()
	fs.BoolVar(&flagGetClusterVersionWithBrokers, "with-brokers", false, "include broker versions")

	setCommandMutation(getClusterVersionCmd, CommandMutationReadOnly)
	setContractSupport(getClusterVersionCmd, ContractSupportLimited)
	setOutputModes(getClusterVersionCmd,
		OutputModeContract{
			Name:      RenderModeOneLine.String(),
			Supported: true,
		},
	)
}

func runGetClusterVersion(cmd *cobra.Command, args []string) {
	cli, log, cfg, err := NewCli(cmd)
	if err != nil {
		handleNewCliError(cmd, log, cfg, err)
	}
	log.Debug("fetching cluster topology for version")
	topology, err := cli.GetClusterTopology(cmd.Context())
	if err != nil {
		ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("get cluster version: %w", err))
	}
	if err := renderClusterVersion(cmd, topology, flagGetClusterVersionWithBrokers); err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render cluster version: %w", err))
	}
}
