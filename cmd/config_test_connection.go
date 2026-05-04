// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

var configTestConnectionCmd = &cobra.Command{
	Use:   "test-connection",
	Short: "Test configured Camunda connection",
	Long: `Test configured Camunda connection.

Loads the effective configuration and logs the config source. The command
validates local configuration before retrieving cluster topology, then warns
when the configured Camunda version differs from the gateway version by
major/minor version.

Use --json for a structured diagnostic payload on stdout; logs remain on stderr.`,
	Example: `  ./c8volt --config ./config.yaml config test-connection
  ./c8volt --config ./config.yaml config test-connection --json
  ./c8volt --profile prod config test-connection`,
	Run: func(cmd *cobra.Command, args []string) {
		log, _ := logging.FromContext(cmd.Context())
		cfg, err := config.FromContext(cmd.Context())
		if err != nil {
			_, noErrCodes := bootstrapFailureContext(cmd)
			ferrors.HandleAndExit(log, noErrCodes, normalizeBootstrapError(fmt.Errorf("loading configuration: %w", err)))
		}

		configSource := configSourceDescriptionFromContext(cmd.Context())
		log.Info(configSource.InfoMessage())
		logConfigProfile(log, cfg)
		if err := cfg.Validate(); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, localPreconditionError(config.FormatValidationError("configuration is invalid", err)))
		}

		ctx, err := installRemoteCommandServices(cmd.Context(), cfg, log)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		cmd.SetContext(ctx)

		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, err)
		}
		topology, err := cli.GetClusterTopology(cmd.Context())
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("config test-connection: %w", err))
		}

		log.Info(fmt.Sprintf("connection to configured Camunda cluster succeeded base_url=%s", cfg.APIs.Camunda.BaseURL))
		warnings := camundaMajorMinorMismatchWarnings(string(cfg.App.CamundaVersion), topology.GatewayVersion)
		for _, warning := range warnings {
			log.Warn(warning)
		}
		if pickMode() == RenderModeJSON {
			if err := renderJSONPayload(cmd, RenderModeJSON, newConfigTestConnectionView(cfg, configSource, topology, warnings)); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render config test-connection result: %w", err))
			}
			return
		}
		if err := renderClusterTopologyTree(cmd, topology); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render cluster topology: %w", err))
		}
	},
}

func init() {
	configCmd.AddCommand(configTestConnectionCmd)

	setCommandMutation(configTestConnectionCmd, CommandMutationReadOnly)
	setOutputModes(configTestConnectionCmd,
		OutputModeContract{
			Name:             RenderModeJSON.String(),
			Supported:        true,
			MachinePreferred: true,
		},
	)
}
