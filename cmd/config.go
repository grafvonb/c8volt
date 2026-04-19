package cmd

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage application configuration",
	Long: `Manage application configuration.

Use this command family to inspect the effective configuration, validate the values a
command would run with, or generate a copy-pasteable template config file. Choose
` + "`config show`" + ` when you need to understand how flags, environment variables,
profiles, and base config resolve into one effective command context.`,
	Example: `  ./c8volt config show
  ./c8volt --config ./config.yaml --profile prod config show --validate
  ./c8volt config show --template`,
	Aliases: []string{"cfg"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"confige", "exepct"},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
