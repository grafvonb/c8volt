package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel resources",
	Long: "Cancel resources such as process instances.\n" +
		"It is a root command and requires a subcommand to specify the resource type to cancel.",
	Aliases: []string{"c", "cn", "stop", "abort"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"cancle", "cancl"},
}

func init() {
	rootCmd.AddCommand(cancelCmd)

	addBackoffFlagsAndBindings(cancelCmd, viper.GetViper())
}
