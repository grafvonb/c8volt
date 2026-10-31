// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestAllCommandHelpFocusesOnActions checks every command, including grouping
// and hidden commands, without coupling help to runtime result formatting.
func TestAllCommandHelpFocusesOnActions(t *testing.T) {
	var visit func(*cobra.Command)
	visit = func(command *cobra.Command) {
		t.Run(command.CommandPath(), func(t *testing.T) {
			prose := strings.ToLower(command.Short + "\n" + command.Long)
			for _, presentation := range []string{
				"default human", "stdout", "stderr", "milestone", "repaint",
				"shared error envelope", "one key per line", "zero bytes",
				"warning-level", "selection scope:", "creation target:",
				"compact human", "under matching element rows",
			} {
				require.NotContains(t, prose, presentation)
			}
		})
		for _, child := range command.Commands() {
			visit(child)
		}
	}
	visit(Root())
}

// TestProcessInstanceMutationStateHelp keeps suspended selection discoverable
// with its version requirement while preserving the existing selector default.
func TestProcessInstanceMutationStateHelp(t *testing.T) {
	for _, verb := range []string{"cancel", "delete"} {
		t.Run(verb, func(t *testing.T) {
			command, _, err := Root().Find([]string{verb, "process-instance"})
			require.NoError(t, err)
			state := command.Flags().Lookup("state")
			require.NotNil(t, state)
			require.Contains(t, state.Usage, "all, active, suspended (Camunda 8.10), completed, canceled, terminated")
			require.Equal(t, "all", state.DefValue)
			require.Equal(t, "s", state.Shorthand)
		})
	}
}

// TestVersionCheckHelp explains advisory scope and the independent explicit read.
func TestVersionCheckHelp(t *testing.T) {
	for _, fragment := range []string{"major/minor", "two seconds", "shared", "--no-version-check", "configured adapter"} {
		require.Contains(t, Root().Long, fragment)
	}
	require.Contains(t, configTestConnectionCmd.Long, "disables only that advisory")
}
