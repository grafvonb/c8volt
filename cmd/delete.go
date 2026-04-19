package cmd

import (
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete resources with explicit destructive confirmation",
	Long: `Delete resources with explicit destructive confirmation.

Use this command family when work should be removed rather than merely inspected.
Child commands explain whether c8volt prompts before deletion, whether cancellation or
preparation happens first, how ` + "`--auto-confirm`" + ` enables unattended destructive
flows, and when ` + "`--no-wait`" + ` returns accepted deletion instead of confirmed completion.`,
	Example: `  ./c8volt delete process-instance --help
  ./c8volt delete process-definition --help
  ./c8volt delete process-instance --state completed --count 200 --auto-confirm --no-wait`,
	Aliases: []string{"d", "del", "remove", "rm"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"deelte", "delet"},
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	addBackoffFlagsAndBindings(deleteCmd)
	setCommandMutation(deleteCmd, CommandMutationStateChanging)
}
