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
	Long: `Validate effective configuration.

Loads the effective configuration through the normal config resolver and uses
the same validation behavior as ` + "`config show --validate`" + `.

Tenant context describes configuration scope only: a named tenant is a discovery
filter, while an empty tenant means no configured tenant filter and is not
reported as <default>. Human diagnostics report explicit --tenant changes before
the resulting scope; --tenant "" warns when it clears a named configured filter.`,
	Example: `  ./c8volt --config ./config.yaml config validate
  ./c8volt --profile prod config validate
  ./c8volt --tenant tenant-a config validate
  ./c8volt --tenant "" config validate`,
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
