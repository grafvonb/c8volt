package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

//nolint:unused
var getVariableCmd = &cobra.Command{
	Use:     "variable",
	Short:   "Get a variable by its name from a process instance",
	Aliases: []string{"var"},
	Run: func(cmd *cobra.Command, args []string) {
		_, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}

		log.Info("Not implemented yet: get variable by name from process instance")
	},
}

func init() {
	// getCmd.AddCommand(getVariableCmd)
}
