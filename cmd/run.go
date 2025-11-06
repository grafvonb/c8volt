package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	flagRunNoWait bool
)

var runCmd = &cobra.Command{
	Use:     "run",
	Short:   "Run resources",
	Aliases: []string{"r"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"rum", "runn"},
}

func init() {
	rootCmd.AddCommand(runCmd)

	addBackoffFlagsAndBindings(runCmd, viper.GetViper())

	runCmd.PersistentFlags().BoolVar(&flagRunNoWait, "no-wait", false, "skip waiting for the creation to be fully processed (no status checks)")
}
