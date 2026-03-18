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
	Long:    "Get the cluster topology of the connected Camunda 8 cluster.\n\nDeprecated: use `c8volt get cluster topology`.",
	Aliases: []string{"ct", "cluster-info", "ci"},
	Run:     runGetClusterTopology,
}

var getClusterTopologyNestedCmd = &cobra.Command{
	Use:   "topology",
	Short: "Get the cluster topology of the connected Camunda 8 cluster",
	Run:   runGetClusterTopology,
}

func init() {
	getCmd.AddCommand(getClusterTopologyCmd)
	getClusterCmd.AddCommand(getClusterTopologyNestedCmd)
}

func runGetClusterTopology(cmd *cobra.Command, args []string) {
	cli, log, cfg, err := NewCli(cmd)
	if err != nil {
		ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
	}
	log.Debug("fetching cluster topology")
	topology, err := cli.GetClusterTopology(cmd.Context())
	if err != nil {
		ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error fetching topology: %w", err))
	}
	cmd.Println(toolx.ToJSONString(topology))
}
