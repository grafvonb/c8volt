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
	keys := make([]string, 0, len(plan.MissingAncestors))
	for _, item := range plan.MissingAncestors {
		keys = append(keys, item.Key)
	}
	cmd.PrintErrf("missing ancestor keys: %s\n", strings.Join(keys, ", "))
}
