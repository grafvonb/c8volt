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
	Long:  `Show the effective configuration with sensitive values sanitized.`,
	Run: func(cmd *cobra.Command, args []string) {
		log, _ := logging.FromContext(cmd.Context())
		cfg, err := config.FromContext(cmd.Context())
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("loading configuration: %w", err))
		}
		if !flagShowConfigTemplate {
			yCfg, err := cfg.ToSanitizedYAML()
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("marshaling configuration to YAML: %w", err))
			}
			cmd.Println(yCfg)
			if flagShowConfigValidate {
				err = cfg.Validate()
				if err != nil {
					ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("configuration is invalid:\n%w", err))
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
}
