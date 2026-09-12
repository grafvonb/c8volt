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
