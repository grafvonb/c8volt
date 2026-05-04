// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

var (
	flagShowConfigValidate bool
	flagShowConfigTemplate bool
)

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show effective configuration",
	Long: `Show effective configuration with sensitive values sanitized.

Precedence: flag > env > profile > base config > default.
The --validate and --template flags remain supported as compatibility shortcuts
for validation and template rendering.`,
	Example: `  ./c8volt config show
  ./c8volt --config ./config.yaml --profile prod config show
  ./c8volt --config ./config.yaml config show --validate
  ./c8volt config show --template`,
	Run: func(cmd *cobra.Command, args []string) {
		log, _ := logging.FromContext(cmd.Context())
		cfg, err := config.FromContext(cmd.Context())
		if err != nil {
			_, noErrCodes := bootstrapFailureContext(cmd)
			ferrors.HandleAndExit(log, noErrCodes, normalizeBootstrapError(fmt.Errorf("loading configuration: %w", err)))
		}
		if !flagShowConfigTemplate {
			yCfg, err := cfg.ToSanitizedYAML()
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("marshaling configuration to YAML: %w", err))
			}
			cmd.Println(yCfg)
			for _, warning := range cfg.Warnings() {
				cmd.PrintErrf("warning: %s\n", warning)
			}
			if flagShowConfigValidate {
				validateConfigForCommand(log, cfg)
			}
		} else {
			templateCfg, yCfg, err := renderBlankConfigTemplateYAML()
			if err != nil {
				ferrors.HandleAndExit(log, templateCfg.App.NoErrCodes, err)
			}
			cmd.Println(yCfg)
		}
	},
}

func validateConfigForCommand(log *slog.Logger, cfg *config.Config) {
	if err := cfg.Validate(); err != nil {
		ferrors.HandleAndExit(log, cfg.App.NoErrCodes, localPreconditionError(config.FormatValidationError("configuration is invalid", err)))
	}
	ferrors.HandleAndExitOK(log, "configuration is valid")
}

func renderBlankConfigTemplateYAML() (*config.Config, string, error) {
	cfg := config.New()
	_ = cfg.Normalize()
	yCfg, err := cfg.ToTemplateYAML()
	if err != nil {
		return cfg, "", fmt.Errorf("marshaling configuration to YAML template: %w", err)
	}
	return cfg, yCfg, nil
}

func init() {
	configCmd.AddCommand(configShowCmd)

	configShowCmd.Flags().BoolVar(&flagShowConfigValidate, "validate", false, "compatibility shortcut: validate the effective configuration and exit with an error code if invalid")
	configShowCmd.Flags().BoolVar(&flagShowConfigTemplate, "template", false, "compatibility shortcut: print a blank configuration template")
	configShowCmd.MarkFlagsMutuallyExclusive("validate", "template")
}
