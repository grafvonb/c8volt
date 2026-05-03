// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestOutputModesForCommand_UsesConfiguredContractSupport(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "demo", Short: "Demo"}
	cmd.Flags().Bool("json", false, "output as JSON")
	cmd.Flags().Bool("keys-only", false, "keys only")
	setContractSupport(cmd, ContractSupportFull)

	modes := outputModesForCommand(cmd)

	require.Equal(t, []OutputModeContract{
		{Name: "one-line", Supported: true},
		{Name: "json", Supported: true, MachinePreferred: true},
		{Name: "keys-only", Supported: true},
	}, modes)
}

func TestCommandCapabilityForCommand_IncludesInheritedAndRequiredFlags(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(getResourceCmd)

	require.Equal(t, "get resource", capability.Path)
	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportUnsupported, capability.AutomationSupport)
	require.Contains(t, capability.Aliases, "r")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "id",
		Shorthand:   "i",
		Type:        "string",
		Required:    true,
		Repeated:    false,
		Description: "resource id to fetch",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "automation",
		Shorthand:   "",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "enable non-interactive mode for commands that explicitly support it",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "json",
		Shorthand:   "j",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "output as JSON (where applicable)",
	})
}

func TestCommandCapabilityForCommand_UsesExplicitAutomationOutputModes(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(runProcessInstanceCmd)

	require.Equal(t, "run process-instance", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Equal(t, []OutputModeContract{
		{Name: "one-line", Supported: true},
		{Name: "json", Supported: true, MachinePreferred: true},
		{Name: "keys-only", Supported: true},
	}, capability.OutputModes)
}

func TestCommandPath_TrimsRootName(t *testing.T) {
	require.Equal(t, "", commandPath(Root()))
	require.Equal(t, "version", commandPath(versionCmd))
	require.Equal(t, "walk process-instance", commandPath(walkProcessInstanceCmd))
}

func TestCommandCapabilityForCommand_IncludesExplicitAutomationMetadata(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "demo", Short: "Demo"}
	setAutomationSupport(cmd, AutomationSupportFull, "safe for unattended execution")

	capability := commandCapabilityForCommand(cmd)

	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Equal(t, "safe for unattended execution", capability.AutomationNotes)
}

func TestIsDiscoverableCommand_FiltersHiddenAndInternalCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  *cobra.Command
		want bool
	}{
		{
			name: "nil",
			cmd:  nil,
			want: false,
		},
		{
			name: "visible public command",
			cmd:  &cobra.Command{Use: "get", Short: "Get resources"},
			want: true,
		},
		{
			name: "hidden command",
			cmd: &cobra.Command{
				Use:    "completion",
				Short:  "Shell completion",
				Hidden: true,
			},
			want: false,
		},
		{
			name: "shell completion command",
			cmd:  &cobra.Command{Use: "completion", Short: "Shell completion"},
			want: false,
		},
		{
			name: "help command",
			cmd:  &cobra.Command{Use: "help", Short: "Help"},
			want: false,
		},
		{
			name: "shell completion plumbing",
			cmd:  &cobra.Command{Use: "__complete", Short: "internal"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isDiscoverableCommand(tt.cmd))
		})
	}
}

func TestContractSupportForCommand_IgnoresHiddenChildren(t *testing.T) {
	t.Parallel()

	parent := &cobra.Command{Use: "demo", Short: "Demo"}
	hiddenChild := &cobra.Command{Use: "completion", Short: "Hidden helper", Hidden: true}
	setContractSupport(hiddenChild, ContractSupportFull)
	parent.AddCommand(hiddenChild)

	require.Equal(t, ContractSupportUnsupported, contractSupportForCommand(parent))
}

func TestCapabilityDocumentForRoot_ExcludesHiddenAndShellInternalCommands(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	publicChild := &cobra.Command{Use: "discovery-fixture", Short: "Fixture"}
	hiddenChild := &cobra.Command{Use: "completion", Short: "Shell completion", Hidden: true}
	helpChild := &cobra.Command{Use: "help", Short: "Help"}
	internalChild := &cobra.Command{Use: "__complete", Short: "internal"}
	root.AddCommand(publicChild, hiddenChild, helpChild, internalChild)
	t.Cleanup(func() {
		root.RemoveCommand(publicChild, hiddenChild, helpChild, internalChild)
	})

	doc := capabilityDocumentForRoot(root)

	var paths []string
	for _, command := range doc.Commands {
		paths = append(paths, command.Path)
	}

	require.Contains(t, paths, "discovery-fixture")
	require.NotContains(t, paths, "completion")
	require.NotContains(t, paths, "help")
	require.NotContains(t, paths, "__complete")
}

func TestCapabilityDocumentForRoot_ExcludesRemovedClusterTopologyCommand(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	doc := capabilityDocumentForRoot(root)

	paths := commandCapabilityPaths(doc.Commands)
	require.NotContains(t, paths, "get cluster-topology")
	require.NotContains(t, paths, "get ct")
	require.NotContains(t, paths, "get cluster-info")
	require.NotContains(t, paths, "get ci")
	require.Contains(t, paths, "get cluster topology")
	require.Contains(t, paths, "get cluster version")
}

func commandCapabilityPaths(commands []CommandCapability) []string {
	var paths []string
	for _, command := range commands {
		paths = append(paths, command.Path)
		paths = append(paths, commandCapabilityPaths(command.Children)...)
	}
	return paths
}
