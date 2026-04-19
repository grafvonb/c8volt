package cmd

import (
	"github.com/spf13/cobra"
)

var expectCmd = &cobra.Command{
	Use:   "expect",
	Short: "Wait for verification targets to reach the expected state",
	Long: `Wait for verification targets to reach the expected state.

Use this read-only command family after a state-changing operation when success depends
on a later observed state. Child commands document the wait contract, the acceptable
target states, and which output modes are safe for follow-up verification.`,
	Example: `  ./c8volt expect process-instance --help
  ./c8volt expect process-instance --key 2251799813711967 --state active
  ./c8volt expect process-instance --key 2251799813711967 --state absent`,
	Aliases: []string{"e", "exp"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"expecte", "exepct"},
}

func init() {
	rootCmd.AddCommand(expectCmd)

	addBackoffFlagsAndBindings(expectCmd)
	setCommandMutation(expectCmd, CommandMutationReadOnly)
}
