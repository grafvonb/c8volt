// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

var getClusterLicenseCmd = &cobra.Command{
	Use:     "license",
	Aliases: []string{"licence"},
	Short:   "Show connected cluster license",
	Long: "Show connected cluster license.\n\n" +
		"This command prints flat human-readable fields returned by the configured Camunda cluster. Use --json for the structured license payload.",
	Example: `  ./c8volt get cluster license
  ./c8volt get cluster license --json
  ./c8volt get cluster licence`,
	Args: cobra.NoArgs,
	Run:  runGetClusterLicense,
}

func init() {
	getClusterCmd.AddCommand(getClusterLicenseCmd)

	setCommandMutation(getClusterLicenseCmd, CommandMutationReadOnly)
	setContractSupport(getClusterLicenseCmd, ContractSupportFull)
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
	if pickMode() == RenderModeJSON {
		if err := renderJSONPayload(cmd, RenderModeJSON, license); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render cluster license: %w", err))
		}
		return
	}
	if err := renderClusterLicenseFlat(cmd, license); err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render cluster license: %w", err))
	}
}
