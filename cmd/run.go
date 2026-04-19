package cmd

import (
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start state-changing work such as process instances",
	Long: `Start state-changing work such as process instances.

Use this command family when you want c8volt to create new work in Camunda. Choose
` + "`run process-instance`" + ` to start one or more instances. Child commands document
whether they wait for confirmed creation by default, when ` + "`--no-wait`" + ` can return
accepted work earlier, and how to pair the result with follow-up inspection commands.`,
	Example: `  ./c8volt run process-instance --help
  ./c8volt run process-instance --bpmn-process-id order-process
  ./c8volt --automation --json run process-instance --bpmn-process-id order-process --no-wait`,
	Aliases: []string{"r"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"rum", "runn", "execute"},
}

func init() {
	rootCmd.AddCommand(runCmd)

	addBackoffFlagsAndBindings(runCmd)
	setCommandMutation(runCmd, CommandMutationStateChanging)
}
