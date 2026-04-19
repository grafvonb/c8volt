package cmd

import (
	"github.com/spf13/cobra"
)

//nolint:unused
var getVariableCmd = &cobra.Command{
	Use:     "variable",
	Short:   "Get a variable by its name from a process instance",
	Long:    "Get a variable by its name from a process instance.\n\nThis command is reserved for variable-specific read guidance when the public command is surfaced.",
	Example: "  ./c8volt get variable --help",
	Aliases: []string{"var"},
	Run: func(cmd *cobra.Command, args []string) {
		_, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, err)
		}

		log.Info("Not implemented yet: get variable by name from process instance")
	},
}

func init() {
	// getCmd.AddCommand(getVariableCmd)
}
