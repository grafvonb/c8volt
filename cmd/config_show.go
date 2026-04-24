package cmd

import (
	"fmt"

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
	Long: `Show the effective configuration with sensitive values sanitized.

Precedence follows one shared contract for all config-backed settings:
flag > env > profile > base config > default.

Use this command to inspect the values a command will actually use after
applying flags, environment variables, profile overlays, base config, and
defaults. Profile values overlay base config field by field and never override
an explicit flag or environment winner.`,
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
				err = cfg.Validate()
				if err != nil {
					ferrors.HandleAndExit(log, cfg.App.NoErrCodes, localPreconditionError(config.FormatValidationError("configuration is invalid", err)))
				}
				ferrors.HandleAndExitOK(log, "configuration is valid")
			}
		} else {
			cfg := config.New()
			_ = cfg.Normalize()
			yCfg, err := cfg.ToTemplateYAML()
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("marshaling configuration to YAML template: %w", err))
			}
			cmd.Println(yCfg)
		}
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)

	configShowCmd.Flags().BoolVar(&flagShowConfigValidate, "validate", false, "validate the effective configuration and exit with an error code if invalid")
	configShowCmd.Flags().BoolVar(&flagShowConfigTemplate, "template", false, "template configuration with values blanked out (copy-paste ready)")
	configShowCmd.MarkFlagsMutuallyExclusive("validate", "template")
	configShowCmd.Example += `

# Inspect how flags override env/profile/config for the current command invocation
./c8volt --config ./config.yaml --profile prod --tenant ops-tenant config show

# Validate the effective config after env/profile/config resolution
C8VOLT_AUTH_MODE=oauth2 ./c8volt --config ./config.yaml config show --validate`
}
