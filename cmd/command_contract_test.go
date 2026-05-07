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

// Protects the discovery contract after removing the direct topology command and aliases.
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

func TestCapabilityDocumentForRoot_ConfigDiagnosticsContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	doc := capabilityDocumentForRoot(root)

	show, ok := findCommandCapability(doc.Commands, "config show")
	require.True(t, ok)
	require.Equal(t, CommandMutationReadOnly, show.Mutation)
	require.Contains(t, show.Flags, FlagContract{
		Name:        "validate",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "compatibility shortcut: validate the effective configuration and exit with an error code if invalid",
	})
	require.Contains(t, show.Flags, FlagContract{
		Name:        "template",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "compatibility shortcut: print a blank configuration template",
	})

	for _, path := range []string{
		"config validate",
		"config template",
		"config test-connection",
	} {
		capability, ok := findCommandCapability(doc.Commands, path)
		require.True(t, ok, "missing command capability for %s", path)
		require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	}
}

func TestCommandCapabilityForCommand_ProcessInstanceExpectIncidentFlag(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(expectProcessInstanceCmd)

	require.Equal(t, "expect process-instance", capability.Path)
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "state",
		Shorthand:   "s",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "state expectation; valid values are: [active, completed, canceled, terminated, absent]",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "incident",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "incident expectation; valid values are: [true, false]",
	})
}

func TestCommandCapabilityForCommand_ProcessInstanceVariableFlags(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(getProcessInstanceCmd)

	require.Equal(t, "get process-instance", capability.Path)
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-vars",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "include process-instance-scope variables for keyed or list/search process-instance output",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "var-value-limit",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum characters to show for human variable values when --with-vars is set; 0 disables truncation",
	})
}

func TestCommandCapabilityForCommand_UpdateProcessInstanceContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(updateProcessInstanceCmd)

	require.Equal(t, "update process-instance", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.Aliases, "pi")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "process instance key(s) to update; repeat or combine with stdin '-'",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "vars",
		Type:        "string",
		Required:    true,
		Repeated:    false,
		Description: "JSON object with variables to set on each process instance",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after the update request is accepted without variable confirmation",
	})
}

func TestCapabilityDocumentForRoot_UpdateCommandFamily(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	doc := capabilityDocumentForRoot(root)

	update, ok := findCommandCapability(doc.Commands, "update")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, update.Mutation)
	require.Equal(t, ContractSupportLimited, update.ContractSupport)
	require.Contains(t, update.Aliases, "u")

	updatePI, ok := findCommandCapability(doc.Commands, "update process-instance")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, updatePI.Mutation)
	require.Equal(t, ContractSupportFull, updatePI.ContractSupport)
	require.Equal(t, AutomationSupportFull, updatePI.AutomationSupport)
}

func TestUpdateProcessInstanceHelp_DocumentsVariableUpdateDiscovery(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"update"}, []string{
		"Update existing resources",
		"Camunda 8.8 and 8.9",
		"unsupported-version error before mutation",
		"./c8volt update process-instance --key 2251799813711967 --vars",
		"./c8volt --automation --json update pi --key 2251799813711967 --vars",
	}, nil)
	require.Contains(t, output, "process-instance")

	output = assertCommandHelpOutput(t, []string{"update", "process-instance"}, []string{
		"Update process-instance variables by key",
		"--vars flag must be a JSON object",
		"repeated --key values or newline-separated keys from stdin with '-'",
		"By default c8volt waits until requested process-instance-scope variables are visible",
		"add --no-wait to return after the update request is accepted",
		"Camunda 8.7 returns an unsupported-version error before mutation",
		"./c8volt update pi --key 2251799813711967 --key 2251799813711968 --vars",
		"printf '%s\\n' 2251799813711967 | ./c8volt update pi --key 2251799813711968 - --vars",
		"--workers",
		"--fail-fast",
	}, nil)
	require.Contains(t, output, "Aliases:")
	require.Contains(t, output, "pi")

	aliasOutput := assertCommandHelpOutput(t, []string{"update", "pi"}, []string{
		"Update process-instance variables by key",
		"--vars string",
		"--no-wait",
	}, nil)
	require.Contains(t, aliasOutput, "Aliases:")
	require.Contains(t, aliasOutput, "pi")
}

func TestProcessInstanceSelectorValidationHelpContract(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		wants []string
	}{
		{
			name: "get pi",
			args: []string{"get", "pi", "--help"},
			wants: []string{
				"When --bpmn-process-id is set, c8volt validates that the process definition is visible before searching process instances.",
				"A missing selector fails with a local diagnostic instead of looking like a valid empty result",
				"--json, --automation, --keys-only, and non-TTY runs never prompt for recovery output.",
			},
		},
		{
			name: "cancel pi",
			args: []string{"cancel", "pi", "--help"},
			wants: []string{
				"When --bpmn-process-id is set, c8volt validates that the process definition is visible before planning cancellation.",
				"A missing selector fails before mutation and automation-oriented modes never prompt for recovery output.",
			},
		},
		{
			name: "delete pi",
			args: []string{"delete", "pi", "--help"},
			wants: []string{
				"When --bpmn-process-id is set, c8volt validates that the process definition is visible before planning deletion.",
				"A missing selector fails before mutation and automation-oriented modes never prompt for recovery output.",
			},
		},
		{
			name: "run pi",
			args: []string{"run", "pi", "--help"},
			wants: []string{
				"When running by BPMN process ID, c8volt validates all requested process definitions before creating anything.",
				"Mixed visible and missing BPMN IDs fail as one request, so no partial process instances are started",
				"automation-oriented modes never prompt for recovery output.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := executeRootForTest(t, tt.args...)
			for _, want := range tt.wants {
				require.Contains(t, output, want)
			}
		})
	}
}

// commandCapabilityPaths flattens nested discovery output so removed aliases cannot hide under `get`.
func commandCapabilityPaths(commands []CommandCapability) []string {
	var paths []string
	for _, command := range commands {
		paths = append(paths, command.Path)
		paths = append(paths, commandCapabilityPaths(command.Children)...)
	}
	return paths
}

func findCommandCapability(commands []CommandCapability, path string) (CommandCapability, bool) {
	for _, command := range commands {
		if command.Path == path {
			return command, true
		}
		if child, ok := findCommandCapability(command.Children, path); ok {
			return child, true
		}
	}
	return CommandCapability{}, false
}
