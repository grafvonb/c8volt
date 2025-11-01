package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

var getClusterTopologyCmd = &cobra.Command{
	Use:     "cluster-topology",
	Short:   "Get the cluster topology of the connected Camunda 8 cluster",
	Aliases: []string{"ct", "cluster-info", "ci"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, _, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, err)
		}
		log.Debug("fetching cluster topology")
		topology, err := cli.GetClusterTopology(cmd.Context())
		if err != nil {
			ferrors.HandleAndExit(log, fmt.Errorf("error fetching topology: %w", err))
		}
		cmd.Println(toolx.ToJSONString(topology))
	},
}

func init() {
	getCmd.AddCommand(getClusterTopologyCmd)
}
