package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

var getIncidentCmd = &cobra.Command{
	Use:     "incident",
	Short:   "Get a incidents",
	Aliases: []string{"i"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		log.Debug("fetching process incidents")

	},
}

func init() {
	getCmd.AddCommand(getIncidentCmd)
}
