package cmd

import (
	"github.com/spf13/cobra"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel running work with explicit confirmation semantics",
	Long: `Cancel running work with explicit confirmation semantics.

Use this command family when you need c8volt to stop active work in Camunda. Child
commands explain what gets validated before cancellation, when prompts appear, how
` + "`--auto-confirm`" + ` enables unattended destructive flows, and when ` + "`--no-wait`" + `
returns accepted cancellation before final completion is observed.`,
	Example: `  ./c8volt cancel process-instance --help
  ./c8volt cancel process-instance --key 2251799813711967
  ./c8volt cancel process-instance --state active --count 200 --auto-confirm --no-wait`,
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
