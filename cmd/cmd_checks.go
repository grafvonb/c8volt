// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

var completionCommands = map[string]struct{}{
	"__complete":       {},
	"__completeNoDesc": {},
}

var utilityCommands = map[string]struct{}{
	"capabilities": {},
	"help":         {},
	"version":      {},
	"completion":   {},
	"config":       {},
	"show":         {},
}

func isUtilityCommand(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	_, ok := utilityCommands[cmd.Name()]
	return ok
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
