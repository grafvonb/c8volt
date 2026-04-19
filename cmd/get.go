package cmd

import (
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Read cluster, process, and resource state without changing it",
	Long: `Read cluster, process, and resource state without changing it.

Use this command family when you need to inspect current Camunda state and choose a
resource-specific child command such as cluster topology, process definitions, process
instances, or resources. Prefer ` + "`get cluster`" + ` for cluster-wide inspection and
use the resource-specific leaf commands when you already know the object type you need.

Where a child command supports structured output, prefer ` + "`--json`" + ` for automation
and AI-assisted callers instead of scraping the default human-readable output.`,
	Example: `  ./c8volt get cluster --help
  ./c8volt get process-instance --json
  ./c8volt get process-definition --keys-only`,
	Aliases: []string{"g", "read"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"gett", "getr"},
}

func init() {
	rootCmd.AddCommand(getCmd)

	addBackoffFlagsAndBindings(getCmd)
	setCommandMutation(getCmd, CommandMutationReadOnly)
}
