// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strconv"
	"strings"

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
major/minor version.`,
	Example: `  ./c8volt --config ./config.yaml config test-connection
  ./c8volt --profile prod config test-connection`,
	Run: func(cmd *cobra.Command, args []string) {
		log, _ := logging.FromContext(cmd.Context())
		cfg, err := config.FromContext(cmd.Context())
		if err != nil {
			_, noErrCodes := bootstrapFailureContext(cmd)
			ferrors.HandleAndExit(log, noErrCodes, normalizeBootstrapError(fmt.Errorf("loading configuration: %w", err)))
		}

		log.Info(configSourceDescriptionFromContext(cmd.Context()).InfoMessage())
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

		log.Info("connection to configured Camunda cluster succeeded")
		warnOnCamundaMajorMinorMismatch(log, string(cfg.App.CamundaVersion), topology.GatewayVersion)
		if err := renderClusterTopologyTree(cmd, topology); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render cluster topology: %w", err))
		}
	},
}

func warnOnCamundaMajorMinorMismatch(log interface{ Warn(string, ...any) }, configuredVersion string, gatewayVersion string) {
	configuredMajorMinor, configuredOK := parseMajorMinorVersion(configuredVersion)
	gatewayMajorMinor, gatewayOK := parseMajorMinorVersion(gatewayVersion)
	if !configuredOK || !gatewayOK || configuredMajorMinor == gatewayMajorMinor {
		return
	}
	log.Warn(fmt.Sprintf("configured Camunda version %s differs from gateway version %s by major/minor version", configuredVersion, gatewayVersion))
}

func parseMajorMinorVersion(version string) (string, bool) {
	version = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(version), "v"))
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return "", false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", false
	}
	return fmt.Sprintf("%d.%d", major, minor), true
}

func init() {
	configCmd.AddCommand(configTestConnectionCmd)

	setCommandMutation(configTestConnectionCmd, CommandMutationReadOnly)
}
