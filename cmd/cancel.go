package cmd

import (
	"github.com/spf13/cobra"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel running process instances",
	Long: `Cancel running process instances.

Use this command family when active workflow work should be stopped. The
process-instance command validates the affected tree, prompts before destructive
changes, and waits for the observed cancellation unless you opt out.`,
	Example: `  ./c8volt cancel pi --key <process-instance-key>
  ./c8volt cancel pi --key <process-instance-key> --force
  ./c8volt cancel pi --state active --batch-size 200 --auto-confirm`,
	Aliases: []string{"c", "cn", "stop", "abort"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"cancle", "cancl"},
}

func init() {
	rootCmd.AddCommand(cancelCmd)

	addBackoffFlagsAndBindings(cancelCmd)
	setCommandMutation(cancelCmd, CommandMutationStateChanging)
}
