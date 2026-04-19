package cmd

import "github.com/spf13/cobra"

var getClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Inspect cluster-wide topology and license information",
	Long: `Inspect cluster-wide topology and license information.

Use this parent command when you need cluster-level state rather than
process-specific resources. Choose ` + "`get cluster topology`" + ` to inspect
brokers, partitions, and gateway details, or ` + "`get cluster license`" + ` to
confirm the connected cluster's license payload.

These subcommands are read-only. Prefer ` + "`--json`" + ` on the leaf commands for
automation and AI-assisted callers.`,
	Example: `  ./c8volt get cluster topology
  ./c8volt get cluster license --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	getCmd.AddCommand(getClusterCmd)

	setCommandMutation(getClusterCmd, CommandMutationReadOnly)
}
