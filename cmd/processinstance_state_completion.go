// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// processInstanceStateCompletion offers local vocabulary without loading configuration
// or contacting the backend. Search offers ALL; expectations offer ABSENT.
func processInstanceStateCompletion(expectation bool) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, _ []string, prefix string) ([]string, cobra.ShellCompDirective) {
		states := process.States{process.StateActive, process.StateSuspended, process.StateCompleted, process.StateCanceled, process.StateTerminated}
		if expectation {
			states = append(states, process.StateAbsent)
		} else {
			states = append(process.States{process.StateAll}, states...)
		}
		var candidates []string
		for _, state := range states {
			value := strings.ToLower(state.String())
			if !strings.HasPrefix(value, strings.ToLower(prefix)) {
				continue
			}
			// Preserve the typed prefix so shells can match case variants too.
			value = prefix + value[len(prefix):]
			if state == process.StateSuspended {
				value += "\tRequires Camunda 8.10"
			}
			candidates = append(candidates, value)
		}
		return candidates, cobra.ShellCompDirectiveNoFileComp
	}
}
