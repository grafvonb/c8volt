package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

var (
	flagConfigShow     bool
	flagConfigValidate bool
	flagConfigTemplate bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		log, _ := logging.FromContext(cmd.Context())
		cfg, err := config.FromContext(cmd.Context())
		if err != nil {
			return fmt.Errorf("config from context: %w", err)
		}

		if flagConfigValidate {
			err = cfg.Validate()
			if err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("configuration is invalid:\n%w", err))
			}
			ferrors.HandleAndExitOK(log, "configuration is valid")
		}
		if flagConfigShow {
			yCfg, err := cfg.ToSanitizedYAML()
			if err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("marshaling configuration to YAML: %w", err))
			}
			cmd.Println(yCfg)
			return nil
		}
		if flagConfigTemplate {
			yCfg, err := cfg.ToTemplateYAML()
			if err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("marshaling configuration to YAML template: %w", err))
			}
			cmd.Println(yCfg)
			return nil
		}
		return cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	configCmd.Flags().BoolVar(&flagConfigShow, "show", false, "Show effective configuration with sensitive values sanitized")
	configCmd.Flags().BoolVar(&flagConfigValidate, "validate", false, "Validate the effective configuration and exit with an error code if invalid")
	configCmd.Flags().BoolVar(&flagConfigTemplate, "template", false, "Template configuration with all values blanked out (copy-paste ready)")
}
