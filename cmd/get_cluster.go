package cmd

import "github.com/spf13/cobra"

var getClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Get cluster resources",
	Long: "Get cluster resources such as the topology or license of the connected Camunda 8 cluster.\n" +
		"It is a parent command and requires a subcommand to specify the cluster resource to get.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	getCmd.AddCommand(getClusterCmd)

	setCommandMutation(getClusterCmd, CommandMutationReadOnly)
}
