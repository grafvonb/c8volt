package cmd

import "github.com/spf13/cobra"

var getClusterLicenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Get the cluster license of the connected Camunda 8 cluster",
	Long: "Get the cluster license of the connected Camunda 8 cluster.\n\n" +
		"This command requires a configured Camunda 8 connection.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	getClusterCmd.AddCommand(getClusterLicenseCmd)
}
