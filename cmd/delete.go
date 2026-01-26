package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete resources",
	Long: "Delete resources such as process instances.\n" +
		"It is a root command and requires a subcommand to specify the resource type to delete.",
	Aliases: []string{"d", "del", "remove", "rm"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"deelte", "delet"},
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	addBackoffFlagsAndBindings(deleteCmd, viper.GetViper())
}
