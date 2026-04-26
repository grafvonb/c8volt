// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

var getClusterLicenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Show connected cluster license",
	Long: "Show connected cluster license.\n\n" +
		"This command prints the license payload returned by the configured Camunda cluster.",
	Example: `  ./c8volt get cluster license
  ./c8volt get cluster license --json`,
	Run: runGetClusterLicense,
}

func init() {
	getClusterCmd.AddCommand(getClusterLicenseCmd)

	setCommandMutation(getClusterLicenseCmd, CommandMutationReadOnly)
	setContractSupport(getClusterLicenseCmd, ContractSupportLimited)
	setOutputModes(getClusterLicenseCmd,
		OutputModeContract{
			Name:             RenderModeJSON.String(),
			Supported:        true,
			MachinePreferred: true,
		},
	)
}

func runGetClusterLicense(cmd *cobra.Command, args []string) {
	cli, log, cfg, err := NewCli(cmd)
	if err != nil {
		handleNewCliError(cmd, log, cfg, err)
	}
	log.Debug("fetching cluster license")
	license, err := cli.GetClusterLicense(cmd.Context())
	if err != nil {
		ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("get cluster license: %w", err))
	}
	if err := renderJSONPayload(cmd, RenderModeJSON, license); err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render cluster license: %w", err))
	}
}
