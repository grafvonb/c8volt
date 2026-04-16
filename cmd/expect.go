package cmd

import (
	"github.com/spf13/cobra"
)

var expectCmd = &cobra.Command{
	Use:   "expect",
	Short: "Expect resources to be in a certain state",
	Long: "Expect resources such as process instances to be in a certain state.\n" +
		"It is a root command and requires a subcommand to specify the resource type to expect.",
	Aliases: []string{"e", "exp"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"expecte", "exepct"},
}

func init() {
	rootCmd.AddCommand(expectCmd)

	addBackoffFlagsAndBindings(expectCmd)
}
