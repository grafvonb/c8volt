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

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate effective configuration",
	Long:  `Validate effective configuration without contacting Camunda.`,
	Example: `  c8volt config validate
  c8volt --profile production config validate`,
	Run: func(cmd *cobra.Command, args []string) {
		log, _ := logging.FromContext(cmd.Context())
		cfg, err := config.FromContext(cmd.Context())
		if err != nil {
			_, noErrCodes := bootstrapFailureContext(cmd)
			ferrors.HandleAndExit(log, noErrCodes, normalizeBootstrapError(fmt.Errorf("loading configuration: %w", err)))
		}
		tenantCtx := attachConfigurationTenantContext(cmd, cfg)
		renderTenantContext(cmd, tenantCtx)
		validateConfigForCommand(log, cfg)
	},
}

func init() {
	configCmd.AddCommand(configValidateCmd)

	setCommandMutation(configValidateCmd, CommandMutationReadOnly)
}
