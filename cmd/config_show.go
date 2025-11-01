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
	Long:  "Show effective configuration with sensitive values sanitized. Use --validate to validate the configuration or --template to show a blanked-out template.",
	Run: func(cmd *cobra.Command, args []string) {
		log, _ := logging.FromContext(cmd.Context())
		cfg, err := config.FromContext(cmd.Context())
		if err != nil {
			ferrors.HandleAndExit(log, fmt.Errorf("loading configuration: %w", err))
		}
		if !flagShowConfigTemplate {
			yCfg, err := cfg.ToSanitizedYAML()
			if err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("marshaling configuration to YAML: %w", err))
			}
			cmd.Println(yCfg)
			if flagShowConfigValidate {
				err = cfg.Validate()
				if err != nil {
					ferrors.HandleAndExit(log, fmt.Errorf("configuration is invalid:\n%w", err))
				}
				ferrors.HandleAndExitOK(log, "configuration is valid")
			}
		} else {
			yCfg, err := cfg.ToTemplateYAML()
			if err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("marshaling configuration to YAML template: %w", err))
			}
			cmd.Println(yCfg)
		}
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)

	configShowCmd.Flags().BoolVar(&flagShowConfigValidate, "validate", false, "Validate the effective configuration and exit with an error code if invalid")
	configShowCmd.Flags().BoolVar(&flagShowConfigTemplate, "template", false, "Template configuration with all values blanked out (copy-paste ready)")
}
