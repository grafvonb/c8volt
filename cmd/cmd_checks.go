// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

var completionCommands = map[string]struct{}{
	"__complete":       {},
	"__completeNoDesc": {},
}

var utilityCommandPaths = map[string]struct{}{
	"capabilities":           {},
	"help":                   {},
	"version":                {},
	"completion":             {},
	"config":                 {},
	"config show":            {},
	"config test-connection": {},
	"config template":        {},
	"config validate":        {},
}

func isUtilityCommand(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	_, ok := utilityCommandPaths[bootstrapCommandPath(cmd)]
	return ok
}

// bootstrapCommandPath keeps bootstrap bypass keyed by the user-facing path, so nested
// commands like `get cluster version` do not collide with top-level utilities.
func bootstrapCommandPath(cmd *cobra.Command) string {
	if cmd == nil {
		return ""
	}
	path := cmd.CommandPath()
	if root := cmd.Root(); root != nil && root != cmd {
		path = strings.TrimSpace(strings.TrimPrefix(path, root.Name()))
	}
	if path == "" {
		return cmd.Name()
	}
	return strings.TrimSpace(path)
}

func isCompletionCommand(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	_, ok := completionCommands[cmd.Name()]
	return ok
}

func bypassRootBootstrap(cmd *cobra.Command) bool {
	return isUtilityCommand(cmd) || isCompletionCommand(cmd)
}

func hasHelpFlag(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	return cmd.Flags().Changed("help")
}
