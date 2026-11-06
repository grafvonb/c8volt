// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

var configTestConnectionCmd = &cobra.Command{
	Use:   "test-connection",
	Short: "Test configured Camunda connection",
	Long: `Validate configuration and test the connection to Camunda.

Report the gateway version and warn if it differs from the configured release
line. A version warning does not fail an otherwise successful connection test.
On failure, report a concise reason and a suggested check.`,
	Example: `  c8volt config test-connection
  c8volt --profile production config test-connection`,
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
		tenantCtx := attachConfigurationTenantContext(cmd, cfg)
		renderTenantContext(cmd, tenantCtx)

		ctx, err := installRemoteCommandServices(cmd.Context(), cfg, log)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err, configConnectionFailureMessage(err))
		}
		cmd.SetContext(ctx)

		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, err)
		}
		// The explicit topology operation supplies version evidence as well as the
		// connection result. Own its compact failure diagnostic while retaining
		// normal timeout/retries and redacted DEBUG exchanges, without a second probe.
		connectionCtx := httpc.WithoutReadRetryLogs(httpc.BypassBeforeRequest(cmd.Context()))
		topology, err := cli.GetClusterTopology(connectionCtx)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("config test-connection: %w", err), configConnectionFailureMessage(err))
		}

		log.Info("camunda connection ok")
		warnings := camundaReleaseLineCompatibilityWarnings(string(cfg.App.CamundaVersion), topology.GatewayVersion)
		if !versionCheckWarned(cmd.Context()) {
			for _, warning := range camundaHumanCompatibilityWarnings(string(cfg.App.CamundaVersion), topology.GatewayVersion) {
				log.Warn(warning)
			}
		}
		if pickMode() == RenderModeJSON {
			if err := renderJSONPayload(cmd, RenderModeJSON, newConfigTestConnectionView(cfg, configSource, topology, warnings, tenantCtx)); err != nil {
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
