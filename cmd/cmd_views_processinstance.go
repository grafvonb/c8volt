package cmd

import (
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

func printDryRunExpansionWarning(cmd *cobra.Command, plan process.DryRunPIKeyExpansion) {
	if plan.Warning == "" && len(plan.MissingAncestors) == 0 {
		return
	}

	warning := plan.Warning
	if warning == "" {
		warning = "one or more parent process instances were not found"
	}
	cmd.PrintErrf("warning: %s\n", warning)

	if len(plan.MissingAncestors) == 0 {
		return
	}
	printMissingAncestorKeyWarning(cmd.PrintErrf, missingAncestorKeys(plan.MissingAncestors))
}

func missingAncestorKeys(items []process.MissingAncestor) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	return keys
}

func printMissingAncestorKeyWarning(print func(string, ...interface{}), keys []string) {
	if flagVerbose {
		print("missing ancestor keys: %s\n", strings.Join(keys, ", "))
		return
	}
	print("missing ancestor keys: %d (use --verbose to list keys)\n", len(keys))
}
