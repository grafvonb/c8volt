// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
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

// TestCommandCapabilityForCommand_DocumentsTenantContract keeps the machine
// contract aligned with operator-facing tenant help.
func TestCommandCapabilityForCommand_DocumentsTenantContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	const tenantDescription = "tenant ID for discovery/search, selection, create, deploy, and run flows; explicit keys/IDs remain backend-authorized"
	capability := commandCapabilityForCommand(getProcessInstanceCmd)

	require.Contains(t, capability.Flags, FlagContract{
		Name:        "tenant",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: tenantDescription,
	})

	for _, cmd := range []*cobra.Command{
		getProcessInstanceCmd,
		cancelProcessInstanceCmd,
		deleteProcessInstanceCmd,
		getProcessDefinitionCmd,
		deleteProcessDefinitionCmd,
		getResourceCmd,
	} {
		require.Contains(t, cmd.Long, "Tenant contract:")
		require.Contains(t, cmd.Long, "backend-authorized admin input")
	}
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

func TestCommandContractOpsRepairIncident(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(opsRepairIncidentCmd)

	require.Equal(t, "ops repair incident", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Equal(t, []OutputModeContract{
		{Name: "one-line", Supported: true},
		{Name: "json", Supported: true, MachinePreferred: true},
	}, capability.OutputModes)
	require.Contains(t, capability.Aliases, "inc")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "incident key(s) to repair; repeat or combine with stdin '-'",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "retries",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "retry count to set on related jobs; 0 skips retry restoration",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "element-id",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN element ID to filter incidents",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "element-instance-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "element instance key to filter incidents",
	})
	require.False(t, hasFlagContractNamed(capability.Flags, "flow-node-id"))
	require.False(t, hasFlagContractNamed(capability.Flags, "fni-key"))
}

// TestCommandContractOpsAnalyseSlowProcessInstances captures the read-only analysis machine contract.
func TestCommandContractOpsAnalyseSlowProcessInstances(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(opsAnalyseSlowProcessInstancesCmd)

	require.Equal(t, "ops analyse slow-process-instances", capability.Path)
	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.Aliases, "slow-pi")
	require.Contains(t, capability.Aliases, "spi")
	require.Contains(t, opsAnalyseCmd.Aliases, "analyze")
	aliasCmd, remaining, err := root.Find([]string{"ops", "analyze", "slow-process-instances"})
	require.NoError(t, err)
	require.Empty(t, remaining)
	require.Same(t, opsAnalyseSlowProcessInstancesCmd, aliasCmd)
	aliasCmd, remaining, err = root.Find([]string{"ops", "analyse", "spi"})
	require.NoError(t, err)
	require.Empty(t, remaining)
	require.Same(t, opsAnalyseSlowProcessInstancesCmd, aliasCmd)
	require.Contains(t, opsAnalyseSlowProcessInstancesCmd.Example, "ops analyse slow-process-instances --key")
	require.Contains(t, opsAnalyseSlowProcessInstancesCmd.Example, "ops analyze slow-process-instances --bpmn-process-id")
	require.Contains(t, opsAnalyseSlowProcessInstancesCmd.Example, "ops analyse spi --bpmn-process-id")
	require.Contains(t, opsAnalyseSlowProcessInstancesCmd.Example, "--dur-longer 1h30m --dur-element-longer 30s")
	require.Contains(t, opsAnalyseSlowProcessInstancesCmd.Example, "get pi --state active --keys-only")
	require.Contains(t, opsAnalyseSlowProcessInstancesCmd.Long, "Use --dur-longer to keep only process-instance roots")
	require.Contains(t, opsAnalyseSlowProcessInstancesCmd.Long, "Duration thresholds use Go duration syntax")
	require.Contains(t, opsAnalyseSlowProcessInstancesCmd.Long, "Calendar units such as 1d are not accepted")
	require.Equal(t, []OutputModeContract{
		{Name: "one-line", Supported: true},
		{Name: "json", Supported: true, MachinePreferred: true},
		{Name: "keys-only", Supported: true},
	}, capability.OutputModes)
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "process-instance key(s) to analyze; repeat or combine with stdin '-'",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "bpmn-process-id",
		Shorthand:   "b",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN process ID to discover process instances",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "pd-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "process definition key to discover process instances",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "state",
		Shorthand:   "s",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "state to filter discovered process instances: all, active, completed, canceled, terminated",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-incidents-only",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "only include process instances without incidents during discovery",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "batch-size",
		Shorthand:   "n",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "number of process instances to inspect per discovery page; does not cap explicit keys or timeline details (max limit 1000 enforced by server)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "limit",
		Shorthand:   "l",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "maximum number of matching process instances to freeze during discovery; omit to discover all matches",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dur-longer",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "only include process instances whose whole duration is longer than this duration, for example 5m or 1h30m",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dur-element-longer",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "only show element or transition detail rows longer than this duration, for example 30s or 2m",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "duration-after",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "deprecated alias for --dur-element-longer",
	})
	require.False(t, hasFlagContractNamed(capability.Flags, "incidents-only"))
}

// TestCommandCapabilityForCommand_OpsPagedDiscoveryFlagContracts verifies discovery flags describe page size and explicit caps distinctly.
func TestCommandCapabilityForCommand_OpsPagedDiscoveryFlagContracts(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	resetOpsRepairIncidentFlagState()
	resetProcessInstanceCommandGlobals()
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)
	t.Cleanup(resetOpsRepairIncidentFlagState)
	t.Cleanup(resetProcessInstanceCommandGlobals)
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	tests := []struct {
		name          string
		cmd           *cobra.Command
		batchDesc     string
		limitDesc     string
		longFragments []string
	}{
		{
			name:      "incident purge",
			cmd:       opsPurgeProcessInstancesWithIncidentsCmd,
			batchDesc: "number of incidents to inspect per discovery page; does not cap total frozen scope (max limit 1000 enforced by server)",
			limitDesc: "maximum number of matching incidents to freeze before candidate process-instance dedupe; omit to discover all matches",
			longFragments: []string{
				"Discovery pages through all matching incidents by default.",
				"--batch-size tunes per-page discovery requests only",
				"--limit intentionally caps the frozen scope",
			},
		},
		{
			name:      "repair incident",
			cmd:       opsRepairIncidentCmd,
			batchDesc: "number of incidents to inspect per discovery page; does not cap total frozen scope (max limit 1000 enforced by server)",
			limitDesc: "maximum number of matching incidents to freeze for repair; omit to discover all matches",
			longFragments: []string{
				"Search mode pages through all matching incidents by default.",
				"--batch-size tunes per-page discovery requests only",
				"--limit intentionally caps the frozen scope",
			},
		},
		{
			name:      "repair process-instance",
			cmd:       opsRepairProcessInstanceCmd,
			batchDesc: "number of process instances to inspect per discovery page; does not cap total frozen scope (max limit 1000 enforced by server)",
			limitDesc: "maximum number of matching process instances to freeze for repair; omit to discover all matches",
			longFragments: []string{
				"Search mode pages through all matching incident-bearing process instances by default.",
				"--batch-size tunes per-page discovery requests only",
				"--limit intentionally caps the frozen scope",
			},
		},
		{
			name:      "all process definitions purge",
			cmd:       opsPurgeAllProcessDefinitionsCmd,
			batchDesc: "number of process definitions to inspect per discovery page; does not cap total frozen scope (max limit 1000 enforced by server)",
			limitDesc: "maximum number of matching process definitions to freeze for purge; omit to discover all matches",
			longFragments: []string{
				"Discovery pages through all matching process definitions by default.",
				"--batch-size tunes per-page discovery requests only",
				"--limit intentionally caps the frozen scope",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capability := commandCapabilityForCommand(tt.cmd)
			require.Contains(t, capability.Flags, FlagContract{
				Name:        "batch-size",
				Shorthand:   "n",
				Type:        "int32",
				Required:    false,
				Repeated:    false,
				Description: tt.batchDesc,
			})
			require.Contains(t, capability.Flags, FlagContract{
				Name:        "limit",
				Shorthand:   "l",
				Type:        "int32",
				Required:    false,
				Repeated:    false,
				Description: tt.limitDesc,
			})
			for _, want := range tt.longFragments {
				require.Contains(t, tt.cmd.Long, want)
			}
		})
	}
}

// TestCommandContractOpsRepairProcessInstance verifies the process-instance repair target exposes automation metadata.
func TestCommandContractOpsRepairProcessInstance(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(opsRepairProcessInstanceCmd)

	require.Equal(t, "ops repair process-instance", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Equal(t, []OutputModeContract{
		{Name: "one-line", Supported: true},
		{Name: "json", Supported: true, MachinePreferred: true},
	}, capability.OutputModes)
	require.Contains(t, capability.Aliases, "pi")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "process-instance key(s) whose active incidents should be repaired; repeat or combine with stdin '-'",
	})
	for _, flag := range capability.Flags {
		require.NotEqual(t, "incidents-only", flag.Name)
	}
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "direct-incidents-only",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "select only process instances with direct active incidents",
	})
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
		Description: "maximum characters to show for variable values when --with-vars is set; 0 disables truncation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "var-exists",
		Type:        "stringArray",
		Required:    false,
		Repeated:    true,
		Description: "require variable name(s) to exist; repeat or separate names with commas",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "var",
		Type:        "stringArray",
		Required:    false,
		Repeated:    true,
		Description: "require variable equality or advanced clause(s); repeat or separate clauses with commas",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "var-like",
		Type:        "stringArray",
		Required:    false,
		Repeated:    true,
		Description: "require variable value pattern clause(s); repeat or separate clauses with commas",
	})
}

// TestCommandCapabilityForCommand_ProcessInstanceElementFlagAndContract verifies element enrichment is discoverable as read-only command metadata.
func TestCommandCapabilityForCommand_ProcessInstanceElementFlagAndContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(getProcessInstanceCmd)

	require.Equal(t, "get process-instance", capability.Path)
	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.Aliases, "pi")
	require.Contains(t, capability.Aliases, "pis")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-elements",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "include runtime element instances for keyed or list/search process-instance output",
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "one-line", Supported: true})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "json", Supported: true, MachinePreferred: true})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "keys-only", Supported: true})
	require.Contains(t, getProcessInstanceCmd.Long, "Use --with-elements to include runtime element instances under matching process-instance rows.")
	require.Contains(t, getProcessInstanceCmd.Example, "./c8volt get pi --key <process-instance-key> --with-elements")
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
		Required:    false,
		Repeated:    false,
		Description: "JSON object with variables to set on each process instance",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "vars-file",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "path to JSON object file with variables to set on each process instance",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "preview variable updates without submitting mutation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after the update request is accepted without variable confirmation",
	})
}

func TestCommandCapabilityForCommand_GetAndUpdateJobContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	getCapability := commandCapabilityForCommand(getJobCmd)
	require.Equal(t, "get job", getCapability.Path)
	require.Equal(t, CommandMutationReadOnly, getCapability.Mutation)
	require.Equal(t, ContractSupportFull, getCapability.ContractSupport)
	require.Equal(t, AutomationSupportFull, getCapability.AutomationSupport)
	require.Contains(t, getCapability.AutomationNotes, "unattended job reads")
	require.Contains(t, getCapability.OutputModes, OutputModeContract{Name: "json", Supported: true, MachinePreferred: true})
	require.Contains(t, getCapability.OutputModes, OutputModeContract{Name: "keys-only", Supported: true})
	require.Contains(t, getJobCmd.Long, "Search mode will use list filters")
	require.Contains(t, getJobCmd.Long, "Camunda 8.7 returns an unsupported-version error")
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "job key for exact lookup; omit to list or search jobs",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "state",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "Camunda job state to filter in search mode; case-insensitive",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "type",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "job type to filter in search mode",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "pi-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "process instance key to filter in search mode",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "element-instance-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "element instance key to filter in search mode",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "element-id",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN element ID to filter in search mode",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "worker",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "worker name to filter in search mode",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "retries",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "exact retry count to filter in search mode",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "kind",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "Camunda job kind to filter in search mode; case-insensitive",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "listener-event-type",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "listener event type to filter in search mode; case-insensitive",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "limit",
		Shorthand:   "l",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "maximum number of jobs to return in search mode",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "batch-size",
		Shorthand:   "n",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "number of jobs to fetch per page (max limit 1000 enforced by server)",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "total",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return only the numeric total of matching jobs",
	})
	require.Contains(t, getCapability.Flags, FlagContract{
		Name:        "error-message-limit",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum characters to show for error messages; 0 keeps full messages",
	})

	updateCapability := commandCapabilityForCommand(updateJobCmd)
	require.Equal(t, "update job", updateCapability.Path)
	require.Equal(t, CommandMutationStateChanging, updateCapability.Mutation)
	require.Equal(t, ContractSupportFull, updateCapability.ContractSupport)
	require.Equal(t, AutomationSupportFull, updateCapability.AutomationSupport)
	require.Contains(t, updateCapability.AutomationNotes, "non-mutating dry-run previews")
	require.Contains(t, updateCapability.OutputModes, OutputModeContract{Name: "json", Supported: true, MachinePreferred: true})
	require.NotContains(t, updateCapability.OutputModes, OutputModeContract{Name: "keys-only", Supported: true})
	require.Contains(t, updateJobCmd.Long, "worker outcome modes")
	require.Contains(t, updateJobCmd.Long, "Camunda 8.7 returns an unsupported-version error before mutation")
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "key",
		Type:        "string",
		Required:    true,
		Repeated:    false,
		Description: "job key to update",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "retries",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "retry count to set, or remaining retries for --fail",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "timeout",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "timeout duration to submit for the job, for example 60s, 5m, or 1h",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "fail",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "report a technical job failure",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "retry-backoff",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "duration before a failed job becomes retryable, for example 60s, 5m, or 1h",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "message",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "operator message for worker outcome modes",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "throw-bpmn-error",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN error code to throw for the job",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "complete",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "complete the job through the worker outcome API",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "vars",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "JSON object with variables for BPMN error or completion outcomes",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "preview job updates without submitting mutation",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after the update request is accepted without retry confirmation",
	})
	require.Contains(t, updateCapability.Flags, FlagContract{
		Name:        "auto-confirm",
		Shorthand:   "y",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "auto-confirm prompts for non-interactive use",
	})
}

// TestCommandCapabilityForCommand_GetElementContract verifies discovery
// metadata for the runtime element read command added to the get family.
func TestCommandCapabilityForCommand_GetElementContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(getElementCmd)

	require.Equal(t, "get element", capability.Path)
	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "unattended element reads")
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "json", Supported: true, MachinePreferred: true})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "keys-only", Supported: true})
	require.Equal(t, []string{"ei"}, capability.Aliases)
	require.Contains(t, getElementCmd.Long, "Use --key when you know an element instance key.")
	require.Contains(t, getElementCmd.Long, "Search mode follows the shared get paging and limit conventions.")
	require.Contains(t, getElementCmd.Long, "Use --json for the stable element payload and --keys-only when piping element instance keys.")
	require.Contains(t, getElementCmd.Example, "./c8volt get ei --pi-key <process-instance-key> --limit 10")
	require.Contains(t, getElementCmd.Example, "./c8volt get element --pi-key <process-instance-key> --total")
	require.Contains(t, getElementCmd.Example, "./c8volt --json get ei --pi-key <process-instance-key> --limit 5")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "element instance key for exact lookup; omit to list or search runtime elements",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "pi-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "process instance key to filter in search mode",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "element-id",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN element ID to filter in search mode",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "state",
		Shorthand:   "s",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "runtime element state to filter in search mode; case-insensitive",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "type",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "runtime element type to filter in search mode; case-insensitive",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "pd-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "process definition key to filter in search mode",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "bpmn-process-id",
		Shorthand:   "b",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN process ID to filter in search mode",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "batch-size",
		Shorthand:   "n",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "number of elements to fetch per page (max limit 1000 enforced by server)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "limit",
		Shorthand:   "l",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "maximum number of elements to return in search mode",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "total",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return only the numeric total of matching elements",
	})
}

func TestCommandCapabilityForCommand_GetIncidentContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(resetGetIncidentFlagState)

	capability := commandCapabilityForCommand(getIncidentCmd)
	require.Equal(t, "get incident", capability.Path)
	require.Contains(t, capability.Aliases, "incidents")
	require.Contains(t, capability.Aliases, "inc")
	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "incident key(s) to fetch; repeat or combine with stdin '-'",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "error-message-limit",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum characters to show for incident messages; 0 keeps full messages",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-no-error-message",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "omit error messages from incident output",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "pi-keys-only",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return only process instance keys for matching incidents",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "bpmn-process-id",
		Shorthand:   "b",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN process ID to validate and filter incidents",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "element-id",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN element ID to filter incidents",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "element-instance-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "element instance key to filter incidents",
	})
	require.False(t, hasFlagContractNamed(capability.Flags, "flow-node-id"))
	require.False(t, hasFlagContractNamed(capability.Flags, "fni-key"))
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:      "keys-only",
		Supported: true,
	})
}

func TestCommandCapabilityForCommand_BpmnSelectorAlignedCommandContracts(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	t.Cleanup(resetGetIncidentFlagState)

	tests := []struct {
		name              string
		cmd               *cobra.Command
		path              string
		mutation          CommandMutation
		automation        AutomationSupport
		wantAutomation    string
		wantOutputMode    OutputModeContract
		wantBpmnFlag      string
		wantDryRun        bool
		wantAutoConfirm   bool
		wantProcessKeysIn bool
	}{
		{
			name:              "get pi",
			cmd:               getProcessInstanceCmd,
			path:              "get process-instance",
			mutation:          CommandMutationReadOnly,
			automation:        AutomationSupportFull,
			wantOutputMode:    OutputModeContract{Name: "keys-only", Supported: true},
			wantBpmnFlag:      "BPMN process ID to filter process instances",
			wantProcessKeysIn: true,
		},
		{
			name:            "cancel pi",
			cmd:             cancelProcessInstanceCmd,
			path:            "cancel process-instance",
			mutation:        CommandMutationStateChanging,
			automation:      AutomationSupportFull,
			wantAutomation:  "unattended destructive confirmation",
			wantOutputMode:  OutputModeContract{Name: "json", Supported: true, MachinePreferred: true},
			wantBpmnFlag:    "BPMN process ID to filter process instances",
			wantDryRun:      true,
			wantAutoConfirm: true,
		},
		{
			name:            "delete pi",
			cmd:             deleteProcessInstanceCmd,
			path:            "delete process-instance",
			mutation:        CommandMutationStateChanging,
			automation:      AutomationSupportFull,
			wantAutomation:  "unattended destructive confirmation",
			wantOutputMode:  OutputModeContract{Name: "json", Supported: true, MachinePreferred: true},
			wantBpmnFlag:    "BPMN process ID to filter process instances",
			wantDryRun:      true,
			wantAutoConfirm: true,
		},
		{
			name:           "get incident",
			cmd:            getIncidentCmd,
			path:           "get incident",
			mutation:       CommandMutationReadOnly,
			automation:     AutomationSupportFull,
			wantOutputMode: OutputModeContract{Name: "keys-only", Supported: true},
			wantBpmnFlag:   "BPMN process ID to validate and filter incidents",
		},
		{
			name:           "get pd",
			cmd:            getProcessDefinitionCmd,
			path:           "get process-definition",
			mutation:       CommandMutationReadOnly,
			automation:     AutomationSupportUnsupported,
			wantOutputMode: OutputModeContract{Name: "json", Supported: true, MachinePreferred: true, Notes: "preferred for automation when not using --xml"},
			wantBpmnFlag:   "BPMN process ID to filter process instances",
		},
		{
			name:            "delete pd",
			cmd:             deleteProcessDefinitionCmd,
			path:            "delete process-definition",
			mutation:        CommandMutationStateChanging,
			automation:      AutomationSupportFull,
			wantAutomation:  "unattended destructive confirmation",
			wantOutputMode:  OutputModeContract{Name: "json", Supported: true, MachinePreferred: true},
			wantBpmnFlag:    "BPMN process ID of the process definition (all versions) to delete",
			wantDryRun:      true,
			wantAutoConfirm: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capability := commandCapabilityForCommand(tt.cmd)
			require.Equal(t, tt.path, capability.Path)
			require.Equal(t, tt.mutation, capability.Mutation)
			require.Equal(t, ContractSupportFull, capability.ContractSupport)
			require.Equal(t, tt.automation, capability.AutomationSupport)
			if tt.wantAutomation != "" {
				require.Contains(t, capability.AutomationNotes, tt.wantAutomation)
			}
			require.Contains(t, capability.OutputModes, tt.wantOutputMode)
			require.Contains(t, capability.Flags, FlagContract{
				Name:        "bpmn-process-id",
				Shorthand:   "b",
				Type:        "string",
				Required:    false,
				Repeated:    false,
				Description: tt.wantBpmnFlag,
			})
			if tt.wantDryRun {
				require.True(t, hasFlagContractNamed(capability.Flags, "dry-run"), "missing dry-run flag")
			}
			if tt.wantAutoConfirm {
				require.Contains(t, capability.Flags, FlagContract{
					Name:        "auto-confirm",
					Shorthand:   "y",
					Type:        "bool",
					Required:    false,
					Repeated:    false,
					Description: "auto-confirm prompts for non-interactive use",
				})
			}
			if tt.wantProcessKeysIn {
				require.Contains(t, capability.OutputModes, OutputModeContract{Name: "keys-only", Supported: true})
			}
		})
	}
}

// Verifies run process-instance advertises keys-only output for command composition.
func TestCommandCapabilityForCommand_RunProcessInstanceKeysOnlyContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(runProcessInstanceCmd)

	require.Equal(t, "run process-instance", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:      "keys-only",
		Supported: true,
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "bpmn-process-id",
		Shorthand:   "b",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "BPMN process ID(s) to run process instance for (mutually exclusive with --pd-key). Runs latest version unless --pd-version is specified",
	})
}

func TestCommandCapabilityForCommand_OpsPurgeOrphanProcessInstancesContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(opsPurgeOrphanProcessInstancesCmd)

	require.Equal(t, "ops purge orphan-process-instances", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "implicitly confirmed purges")
	require.Contains(t, capability.Aliases, "orphan-pi")
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "discover and validate orphan process-instance cleanup without submitting deletion requests",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "auto-confirm",
		Shorthand:   "y",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "auto-confirm prompts for non-interactive use",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "automation",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "enable non-interactive mode for commands that explicitly support it",
	})
}

// TestCommandCapabilityForCommand_OpsPurgeAllProcessDefinitionsContract verifies discovery metadata for the all-process-definitions purge command.
func TestCommandCapabilityForCommand_OpsPurgeAllProcessDefinitionsContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	capability := commandCapabilityForCommand(opsPurgeAllProcessDefinitionsCmd)

	require.Equal(t, "ops purge all-process-definitions", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "implicitly confirmed all-process-definitions purges")
	require.Contains(t, capability.Aliases, "all-pds")
	require.NotContains(t, capability.Aliases, "purge-definitions")
	require.NotContains(t, capability.Aliases, "delete-all")
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:      "one-line",
		Supported: true,
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "process definition key to select for candidate discovery",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "bpmn-process-id",
		Shorthand:   "b",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN process ID to filter candidate process definitions",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "latest",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "only include the latest matching process-definition version(s)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "pd-version",
		Type:        "int32",
		Required:    false,
		Repeated:    false,
		Description: "process definition version to filter candidate discovery",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "pd-version-tag",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "process definition version tag to filter candidate discovery",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "discover and validate process-definition cleanup without submitting deletion requests",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "report-file",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "write an audit report to the given path",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "report-format",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "audit report format: markdown, json (default inferred from report-file extension)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "automation",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "enable non-interactive mode for commands that explicitly support it",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "auto-confirm",
		Shorthand:   "y",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "auto-confirm prompts for non-interactive use",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "force",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "force cancellation of affected active process instances before deleting process definitions",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after deletion requests are accepted without deletion confirmation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "fail-fast",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "stop scheduling validation or deletion work after the first error",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-worker-limit",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "use all queued jobs as workers when --workers is unset",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "workers",
		Shorthand:   "w",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum concurrent workers when validating the delete plan and deleting process definitions (default: min(targets, 2*GOMAXPROCS, 32))",
	})
	require.NotContains(t, capability.Flags, FlagContract{Name: "xml"})
	require.NotContains(t, capability.Flags, FlagContract{Name: "stat"})

	doc := capabilityDocumentForRoot(root)
	paths := commandCapabilityPaths(doc.Commands)
	require.Contains(t, paths, "ops purge all-process-definitions")
	require.NotContains(t, paths, "ops purge purge-definitions")
	require.NotContains(t, paths, "ops purge delete-all")
}

// TestCommandCapabilityForCommand_OpsPurgeProcessInstancesWithIncidentsContract verifies discovery metadata for the incident purge command.
func TestCommandCapabilityForCommand_OpsPurgeProcessInstancesWithIncidentsContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)

	capability := commandCapabilityForCommand(opsPurgeProcessInstancesWithIncidentsCmd)

	require.Equal(t, "ops purge process-instances-with-incidents", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "implicitly confirmed incident-based purges")
	require.Contains(t, capability.Aliases, "pi-with-incidents")
	require.Contains(t, capability.Aliases, "piwi")
	require.NotContains(t, capability.Aliases, "incident-pis")
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:      "one-line",
		Supported: true,
	})
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "incident key(s) to select for candidate discovery",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "element-id",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN element ID to filter incidents",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "element-instance-key",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "element instance key to filter incidents",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "discover and validate incident-based process-instance cleanup without submitting deletion requests",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "report-format",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "audit report format: markdown, json (default inferred from report-file extension)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "automation",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "enable non-interactive mode for commands that explicitly support it",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "auto-confirm",
		Shorthand:   "y",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "auto-confirm prompts for non-interactive use",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "force",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "force cancellation of the process instance(s), prior to deletion",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after deletion requests are accepted without deletion confirmation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "workers",
		Shorthand:   "w",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum concurrent workers when validating the delete plan and deleting roots (default: min(targets, 2*GOMAXPROCS, 32))",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "fail-fast",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "stop scheduling validation or deletion work after the first error",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-worker-limit",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "use all queued jobs as workers when --workers is unset",
	})
	require.NotContains(t, capability.Flags, FlagContract{Name: "pi-keys-only"})
	require.NotContains(t, capability.Flags, FlagContract{Name: "total"})
	require.False(t, hasFlagContractNamed(capability.Flags, "flow-node-id"))
	require.False(t, hasFlagContractNamed(capability.Flags, "fni-key"))
}

func TestCommandCapabilityForCommand_OpsExecuteRetentionPolicyContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(opsExecuteRetentionPolicyCmd)

	require.Equal(t, "ops execute retention-policy", capability.Path)
	require.Contains(t, capability.Aliases, "ret-pol")
	require.Contains(t, capability.Aliases, "rp")
	require.NotContains(t, capability.Aliases, "rt")
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "implicitly confirmed retention cleanup")
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "retention-days",
		Type:        "int",
		Required:    true,
		Repeated:    false,
		Description: "required non-negative age in days for process-instance retention eligibility",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "discover and validate retention cleanup without submitting deletion requests",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "report-file",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "write an audit report to the given path",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "report-format",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "audit report format: markdown, json (default inferred from report-file extension)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "bpmn-process-id",
		Shorthand:   "b",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN process ID to filter process instances",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "workers",
		Shorthand:   "w",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum concurrent workers when validating the delete plan and deleting roots (default: min(targets, 2*GOMAXPROCS, 32))",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after deletion requests are accepted without deletion confirmation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "force",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "force cancellation of the process instance(s), prior to deletion",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "auto-confirm",
		Shorthand:   "y",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "auto-confirm prompts for non-interactive use",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "automation",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "enable non-interactive mode for commands that explicitly support it",
	})
}

func TestCommandCapabilityForCommand_OpsExecuteSmokeTestContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	capability := commandCapabilityForCommand(opsExecuteSmokeTestCmd)

	require.Equal(t, "ops execute smoke-test", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "implicitly confirmed smoke-test cleanup")
	require.Contains(t, capability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "count",
		Shorthand:   "n",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "number of process instances to create from the deployed smoke-test definition",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "workers",
		Shorthand:   "w",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum concurrent workers when creating, walking, or cleaning smoke-test resources (default: min(count, 2*GOMAXPROCS, 32))",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-worker-limit",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "use all queued smoke-test jobs as workers when --workers is unset",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "fail-fast",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "stop scheduling smoke-test work after the first error",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-cleanup",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "retain created process instances and the deployed process definition",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "validate the smoke-test plan without submitting mutation requests",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after cleanup requests are accepted without deletion confirmation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "report-file",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "write an audit report to the given path",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "report-format",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "audit report format: markdown, json (default inferred from report-file extension)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "auto-confirm",
		Shorthand:   "y",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "auto-confirm prompts for non-interactive use",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "automation",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "enable non-interactive mode for commands that explicitly support it",
	})
}

func TestCapabilityDocumentForRoot_OpsExecuteSmokeTestDiscovery(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	doc := capabilityDocumentForRoot(root)

	opsCapability, ok := findCommandCapability(doc.Commands, "ops")
	require.True(t, ok)
	executeCapability, ok := findCommandCapability(opsCapability.Children, "ops execute")
	require.True(t, ok)
	smokeTestCapability, ok := findCommandCapability(executeCapability.Children, "ops execute smoke-test")
	require.True(t, ok)
	require.Equal(t, "Execute a cluster smoke test workflow", smokeTestCapability.Summary)
	require.Equal(t, CommandMutationStateChanging, smokeTestCapability.Mutation)
	require.Equal(t, ContractSupportFull, smokeTestCapability.ContractSupport)
	require.Equal(t, AutomationSupportFull, smokeTestCapability.AutomationSupport)
	require.Contains(t, smokeTestCapability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, smokeTestCapability.Flags, FlagContract{
		Name:        "report-file",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "write an audit report to the given path",
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

	updateJob, ok := findCommandCapability(doc.Commands, "update job")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, updateJob.Mutation)
	require.Equal(t, ContractSupportFull, updateJob.ContractSupport)
	require.Equal(t, AutomationSupportFull, updateJob.AutomationSupport)
}

func TestCommandCapabilityForCommand_ResolveIncidentContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(resolveIncidentCmd)

	require.Equal(t, "resolve incident", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.Aliases, "inc")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "incident key(s) to resolve; repeat or combine with stdin '-'",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "workers",
		Shorthand:   "w",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum concurrent workers when resolving multiple incidents (default: min(count, 2*GOMAXPROCS, 32))",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "preview incident resolutions without submitting mutation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after the resolution request is accepted without incident confirmation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "fail-fast",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "stop scheduling new incident resolutions after the first error",
	})
}

func TestCommandCapabilityForCommand_ResolveProcessInstanceContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(resolveProcessInstanceCmd)

	require.Equal(t, "resolve process-instance", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.Aliases, "pi")
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "process instance key(s) to resolve; repeat or combine with stdin '-'",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "workers",
		Shorthand:   "w",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "maximum concurrent workers when resolving multiple process instances (default: min(count, 2*GOMAXPROCS, 32))",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "preview process-instance incident resolutions without submitting mutation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after resolution requests are accepted without incident confirmation",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "fail-fast",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "stop scheduling new process-instance resolutions after the first error",
	})
}

func TestCapabilityDocumentForRoot_ResolveCommandFamily(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	doc := capabilityDocumentForRoot(root)

	resolve, ok := findCommandCapability(doc.Commands, "resolve")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, resolve.Mutation)
	require.Equal(t, ContractSupportLimited, resolve.ContractSupport)
	require.Contains(t, resolve.Aliases, "res")

	incident, ok := findCommandCapability(doc.Commands, "resolve incident")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, incident.Mutation)
	require.Equal(t, ContractSupportFull, incident.ContractSupport)
	require.Equal(t, AutomationSupportFull, incident.AutomationSupport)
	require.Contains(t, incident.Aliases, "inc")

	processInstance, ok := findCommandCapability(doc.Commands, "resolve process-instance")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, processInstance.Mutation)
	require.Equal(t, ContractSupportFull, processInstance.ContractSupport)
	require.Equal(t, AutomationSupportFull, processInstance.AutomationSupport)
	require.Contains(t, processInstance.Aliases, "pi")
}

func TestGetJobAndUpdateJobHelp_DocumentsDiscoveryAndMutationGuards(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"get"}, []string{
		"Inspect cluster, process, job, element, incident, tenant, and resource state",
		"./c8volt get job --key <job-key>",
	}, nil)
	require.Contains(t, output, "job")

	output = assertCommandHelpOutput(t, []string{"get", "job"}, []string{
		"Inspect or search Camunda jobs",
		"Use --key with the jobKey exposed by incident-aware process-instance output",
		"Search mode will use list filters",
		"Search mode pages through matching jobs by default",
		"--batch-size tunes per-page discovery requests only",
		"--total returns only the matching count",
		"Use --json for the stable job payload",
		"--error-message-limit",
		"Camunda 8.8 and 8.9",
		"./c8volt get job --key <job-key>",
		"./c8volt get job --state failed --batch-size 10 --limit 50",
		"./c8volt get job --state failed --total",
		"./c8volt --json get job --key <job-key>",
		"--key string",
		"--state string",
		"--element-instance-key string",
		"--element-id string",
		"--listener-event-type string",
		"-n, --batch-size int32",
		"-l, --limit int32",
		"--total",
		"--error-message-limit int",
	}, nil)

	output = assertCommandHelpOutput(t, []string{"update"}, []string{
		"Update existing resources",
		"job retries and timeout by key",
		"dry-run planning",
		"submitted output",
		"./c8volt update job --key <job-key> --retries 3 --dry-run",
		"./c8volt update job --key <job-key> --timeout 5m --auto-confirm",
	}, nil)
	require.Contains(t, output, "job")

	output = assertCommandHelpOutput(t, []string{"update", "job"}, []string{
		"Update a Camunda job by key",
		"supports retries, timeout updates, and worker outcome modes",
		"pre-mutation plan",
		"--dry-run previews",
		"Retry updates are confirmed by reading the job by key by default",
		"timeout updates and worker outcomes report accepted submission",
		"JSON mutations require --dry-run, --auto-confirm, or --automation",
		"--json cannot be combined with --verbose",
		"Camunda 8.7 returns an unsupported-version error before mutation",
		"./c8volt update job --key <job-key> --retries 3 --dry-run",
		"./c8volt update job --key <job-key> --fail --retries 0",
		"./c8volt update job --key <job-key> --throw-bpmn-error PAYMENT_DECLINED",
		`./c8volt update job --key <job-key> --complete --vars '{"approved":true}' --dry-run`,
		"./c8volt --json update job --key <job-key> --retries 3 --dry-run",
		"--key string",
		"--retries int32",
		"--timeout string",
		"--fail",
		"--retry-backoff string",
		"--message string",
		"--throw-bpmn-error string",
		"--complete",
		"--vars string",
		"--dry-run",
		"--auto-confirm",
	}, nil)
}

// TestGetElementHelp_DocumentsSearchAndOutputModes keeps user-facing help in
// sync with the element CLI contract and generated documentation source.
func TestGetElementHelp_DocumentsSearchAndOutputModes(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"get", "element"}, []string{
		"List or fetch Camunda runtime element instances",
		"Use --key when you know an element instance key",
		"Omit --key to list or search element instances by process instance, BPMN element ID, state, type, process definition, or BPMN process ID",
		"Search mode follows the shared get paging and limit conventions",
		"--batch-size controls per-page discovery requests",
		"--total prints only the matching count",
		"Use --json for the stable element payload and --keys-only when piping element instance keys",
		"Element lookup and search require Camunda 8.8 or 8.9",
		"Aliases:",
		"ei",
		"./c8volt get ei -k <element-instance-key>",
		"./c8volt get ei --pi-key <process-instance-key> --limit 10",
		"./c8volt get element --pi-key <process-instance-key> --total",
		"./c8volt --json get ei --pi-key <process-instance-key> --limit 5",
		"-k, --key string",
		"--pi-key string",
		"--element-id string",
		"-s, --state string",
		"--type string",
		"--pd-key string",
		"-b, --bpmn-process-id string",
		"-n, --batch-size int32",
		"-l, --limit int32",
		"--total",
		"--json",
		"--keys-only",
	}, nil)
	require.NotContains(t, output, "--all")
}

func TestGetIncidentHelp_DocumentsAliasesPipelinesAndInheritedOutputModes(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"get", "incident"}, []string{
		"Get Camunda incidents by key or by search criteria",
		"repeated --key values or newline-separated keys from stdin with '-'",
		"Search mode defaults to active incidents",
		"When --bpmn-process-id is supplied in search mode, the BPMN process definition selector is validated before incident totals, key-only output, process-instance-key output, or paging.",
		"./c8volt get incident --key <incident-key>",
		"./c8volt get inc --key <incident-key> --key <another-incident-key>",
		"./c8volt get incident --state resolved --error-type io_mapping_error --limit 5",
		"./c8volt get incident --state active --keys-only | ./c8volt get inc -",
		"./c8volt get incident --state active --error-type io_mapping_error --pi-keys-only",
		"./c8volt get incident --state active --error-type io_mapping_error --pi-keys-only | ./c8volt cancel pi --dry-run -",
		"./c8volt --json get incident --key <incident-key>",
		"./c8volt --keys-only get incident --key <incident-key>",
		"--key strings",
		"--pi-keys-only",
		"return only process instance keys for matching incidents",
		"--state string",
		"--error-type string",
		"--bpmn-process-id string",
		"--pd-key string",
		"--pi-key string",
		"--root-key string",
		"--element-id string",
		"--element-instance-key string",
		"--batch-size int32",
		"--limit int32",
		"--error-message-limit int",
		"--json",
		"--keys-only",
	}, nil)
	require.Contains(t, output, "Aliases:")
	require.Contains(t, output, "incidents")
	require.Contains(t, output, "inc")
	require.NotContains(t, output, "AD_HOC_SUB_PROCESS_NO_RETRIES")
	require.NotContains(t, output, "--flow-node-id")
	require.NotContains(t, output, "--fni-key")
}

func TestIncidentCommandHelpOmitsLegacyElementTerminology(t *testing.T) {
	tests := [][]string{
		{"get", "incident"},
		{"ops", "repair", "incident"},
		{"ops", "purge", "process-instances-with-incidents"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			output := assertCommandHelpOutput(t, args, []string{
				"--element-id string",
				"--element-instance-key string",
			}, []string{
				"--flow-node-id",
				"--fni-key",
			})
			require.NotContains(t, output, "flow node")
		})
	}
}

func TestUpdateProcessInstanceHelp_DocumentsVariableUpdateDiscovery(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"update"}, []string{
		"Update existing resources",
		"Camunda 8.8 and 8.9",
		"unsupported-version error before these mutations",
		"./c8volt update process-instance --key <process-instance-key> --vars",
		"./c8volt update pi --key <process-instance-key> --vars-file",
		"./c8volt --automation --json update pi --key <process-instance-key> --vars",
	}, nil)
	require.Contains(t, output, "process-instance")

	output = assertCommandHelpOutput(t, []string{"update", "process-instance"}, []string{
		"Update process-instance variables by key",
		"Provide exactly one variable payload source",
		"--vars with a JSON object or --vars-file with a path",
		"repeated --key values or newline-separated keys from stdin with '-'",
		"loads current process-instance-scope variables",
		"Use --dry-run to preview without mutating",
		"--auto-confirm for unattended mutation",
		"Camunda 8.7 returns an unsupported-version error before mutation",
		"./c8volt update pi --key <process-instance-key> --vars '{\"customerTier\":\"gold\"}' --dry-run",
		"./c8volt update pi --key <process-instance-key-a> --key <process-instance-key-b> --vars",
		"printf '%s\\n' \"$PROCESS_INSTANCE_KEY_A\" \"$PROCESS_INSTANCE_KEY_B\" | ./c8volt update pi - --vars",
		"--workers",
		"--dry-run",
		"--fail-fast",
	}, nil)
	require.Contains(t, output, "Aliases:")
	require.Contains(t, output, "pi")

	aliasOutput := assertCommandHelpOutput(t, []string{"update", "pi"}, []string{
		"Update process-instance variables by key",
		"--vars string",
		"--vars-file string",
		"--dry-run",
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
				"When --bpmn-process-id is set, c8volt validates that the process definition is visible before searching process instances.",
				"A missing selector fails with a local diagnostic before paging, dry-run planning, confirmation, or cancellation",
				"If the selector is visible but no matching instances are found, no cancellation request is submitted.",
			},
		},
		{
			name: "delete pi",
			args: []string{"delete", "pi", "--help"},
			wants: []string{
				"When --bpmn-process-id is set, c8volt validates that the process definition is visible before searching process instances.",
				"A missing selector fails with a local diagnostic before paging, dry-run planning, confirmation, cancellation, or deletion",
				"If the selector is visible but no matching instances are found, no deletion request is submitted.",
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

func TestProcessDefinitionSelectorValidationHelpContract(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		wants []string
	}{
		{
			name: "get pd",
			args: []string{"get", "pd", "--help"},
			wants: []string{
				"When `--bpmn-process-id` is set, c8volt validates that at least one visible",
				"process definition matches the selector before rendering output.",
				"A missing selector",
				"fails with the shared local diagnostic instead of rendering an ambiguous empty list.",
			},
		},
		{
			name: "delete pd",
			args: []string{"delete", "pd", "--help"},
			wants: []string{
				"When --bpmn-process-id is set, c8volt validates visible process-definition matches before delete impact planning, confirmation, cancellation, or deletion.",
				"A missing selector fails with the shared local diagnostic.",
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

func hasFlagContractNamed(flags []FlagContract, name string) bool {
	for _, flag := range flags {
		if flag.Name == name {
			return true
		}
	}
	return false
}
