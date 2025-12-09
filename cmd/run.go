package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run resources",
	Long: "Run resources such as process instances.\n" +
		"It is a root command and requires a subcommand to specify the resource type to run.",
	Aliases: []string{"r"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"rum", "runn", "execute"},
}

func init() {
	rootCmd.AddCommand(runCmd)

	addBackoffFlagsAndBindings(runCmd, viper.GetViper())
}
