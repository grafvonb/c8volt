// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapabilityDocumentForRoot_BuildsNestedDiscoveryMetadata(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	doc := capabilityDocumentForRoot(root)

	require.Equal(t, "capabilities", doc.Command)
	require.Equal(t, "v1", doc.Version)
	require.NotEmpty(t, doc.Commands)
	require.NotContains(t, doc.Commands, CommandCapability{Path: "completion"})

	var getCapability CommandCapability
	var capabilitiesCapability CommandCapability
	for _, command := range doc.Commands {
		if command.Path == "get" {
			getCapability = command
		}
		if command.Path == "capabilities" {
			capabilitiesCapability = command
		}
	}
	require.Equal(t, "capabilities", capabilitiesCapability.Path)
	require.Equal(t, ContractSupportLimited, capabilitiesCapability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capabilitiesCapability.AutomationSupport)
	require.Equal(t, "discovery command for automation support", capabilitiesCapability.AutomationNotes)
	require.Equal(t, []OutputModeContract{
		{Name: "json", Supported: true, MachinePreferred: true, Notes: "full discovery format"},
		{Name: "one-line", Supported: true, Notes: "summary"},
	}, capabilitiesCapability.OutputModes)

	require.Equal(t, "get", getCapability.Path)
	require.Equal(t, ContractSupportLimited, getCapability.ContractSupport)
	require.Equal(t, CommandMutationReadOnly, getCapability.Mutation)
	require.Equal(t, AutomationSupportUnsupported, getCapability.AutomationSupport)

	var processInstanceCapability CommandCapability
	var runCapability CommandCapability
	for _, child := range getCapability.Children {
		if child.Path == "get process-instance" {
			processInstanceCapability = child
		}
	}
	for _, command := range doc.Commands {
		if command.Path == "run" {
			runCapability = command
			break
		}
	}
	require.Equal(t, "get process-instance", processInstanceCapability.Path)
	require.Contains(t, processInstanceCapability.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, processInstanceCapability.OutputModes, OutputModeContract{
		Name:      "keys-only",
		Supported: true,
	})
	require.Equal(t, ContractSupportFull, processInstanceCapability.ContractSupport)
	require.Equal(t, AutomationSupportFull, processInstanceCapability.AutomationSupport)
	require.Equal(t, ContractSupportLimited, runCapability.ContractSupport)
	require.Equal(t, AutomationSupportUnsupported, runCapability.AutomationSupport)
	require.NotEmpty(t, runCapability.Children)
	require.Equal(t, ContractSupportFull, runCapability.Children[0].ContractSupport)
	require.Equal(t, AutomationSupportFull, runCapability.Children[0].AutomationSupport)
	require.Contains(t, runCapability.Children[0].OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, runCapability.Children[0].OutputModes, OutputModeContract{
		Name:      "keys-only",
		Supported: true,
	})
}

func TestCapabilitiesCommand_JSONOutput(t *testing.T) {
	output := executeRootForTest(t, "capabilities", "--json")

	var doc CapabilityDocument
	require.NoError(t, json.Unmarshal([]byte(output), &doc))
	require.Equal(t, "capabilities", doc.Command)
	require.NotEmpty(t, doc.Commands)

	var walkCapability CommandCapability
	for _, command := range doc.Commands {
		if command.Path == "walk" {
			walkCapability = command
			break
		}
	}
	require.Equal(t, "walk", walkCapability.Path)
	require.Equal(t, ContractSupportLimited, walkCapability.ContractSupport)
	require.Equal(t, CommandMutationReadOnly, walkCapability.Mutation)
	require.Equal(t, AutomationSupportUnsupported, walkCapability.AutomationSupport)
}

func TestCapabilitiesCommand_JSONIncludesResolveMetadata(t *testing.T) {
	output := executeRootForTest(t, "capabilities", "--json")

	var doc CapabilityDocument
	require.NoError(t, json.Unmarshal([]byte(output), &doc))

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
	require.Contains(t, incident.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Description: "preview incident resolutions without submitting mutation",
	})
	require.Contains(t, incident.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Description: "return after the resolution request is accepted without incident confirmation",
	})

	processInstance, ok := findCommandCapability(doc.Commands, "resolve process-instance")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, processInstance.Mutation)
	require.Equal(t, ContractSupportFull, processInstance.ContractSupport)
	require.Equal(t, AutomationSupportFull, processInstance.AutomationSupport)
	require.Contains(t, processInstance.Aliases, "pi")
	require.Contains(t, processInstance.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Description: "preview process-instance incident resolutions without submitting mutation",
	})
	require.Contains(t, processInstance.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Description: "return after resolution requests are accepted without incident confirmation",
	})
}

func TestCapabilitiesCommand_JSONIncludesOpsRootMetadata(t *testing.T) {
	output := executeRootForTest(t, "capabilities", "--json")

	var doc CapabilityDocument
	require.NoError(t, json.Unmarshal([]byte(output), &doc))

	ops, ok := findCommandCapability(doc.Commands, "ops")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, ops.Mutation)
	require.Equal(t, ContractSupportLimited, ops.ContractSupport)
	require.Equal(t, AutomationSupportUnsupported, ops.AutomationSupport)
	require.Contains(t, ops.Aliases, "operations")
	require.Contains(t, ops.Summary, "Discover high-level operational workflows")
	execute, ok := findCommandCapability(ops.Children, "ops execute")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, execute.Mutation)
	require.Equal(t, ContractSupportLimited, execute.ContractSupport)
	require.Equal(t, AutomationSupportUnsupported, execute.AutomationSupport)
	require.Contains(t, execute.Summary, "Discover predefined operational playbooks")
	retentionPolicy, ok := findCommandCapability(execute.Children, "ops execute retention-policy")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, retentionPolicy.Mutation)
	require.Equal(t, ContractSupportFull, retentionPolicy.ContractSupport)
	require.Equal(t, AutomationSupportFull, retentionPolicy.AutomationSupport)
	repair, ok := findCommandCapability(ops.Children, "ops repair")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, repair.Mutation)
	require.Equal(t, ContractSupportLimited, repair.ContractSupport)
	require.Equal(t, AutomationSupportUnsupported, repair.AutomationSupport)
	require.Contains(t, repair.Summary, "Discover repair and remediation workflows")
	repairIncident, ok := findCommandCapability(repair.Children, "ops repair incident")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, repairIncident.Mutation)
	require.Equal(t, ContractSupportFull, repairIncident.ContractSupport)
	require.Equal(t, AutomationSupportFull, repairIncident.AutomationSupport)
	for _, flag := range repair.Flags {
		require.NotEqual(t, "key", flag.Name)
	}
	purge, ok := findCommandCapability(ops.Children, "ops purge")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, purge.Mutation)
	require.Equal(t, ContractSupportLimited, purge.ContractSupport)
	require.Equal(t, AutomationSupportUnsupported, purge.AutomationSupport)
	require.Contains(t, purge.Summary, "Discover destructive operational cleanup workflows")
	orphanPurge, ok := findCommandCapability(purge.Children, "ops purge orphan-process-instances")
	require.True(t, ok)
	require.Equal(t, CommandMutationStateChanging, orphanPurge.Mutation)
	require.Equal(t, ContractSupportFull, orphanPurge.ContractSupport)
	require.Equal(t, AutomationSupportFull, orphanPurge.AutomationSupport)
	require.Contains(t, ops.OutputModes, OutputModeContract{Name: "one-line", Supported: true})
	require.Contains(t, ops.OutputModes, OutputModeContract{Name: "json", Supported: true})
	require.Contains(t, ops.OutputModes, OutputModeContract{Name: "keys-only", Supported: true})
}

// TestCapabilitiesCommand_JSONIncludesPagedWorkflowContracts verifies the
// serialized capability document carries the updated paging and automation text.
func TestCapabilitiesCommand_JSONIncludesPagedWorkflowContracts(t *testing.T) {
	output := executeRootForTest(t, "capabilities", "--json")

	var doc CapabilityDocument
	require.NoError(t, json.Unmarshal([]byte(output), &doc))

	tests := []struct {
		path       string
		batchDesc  string
		limitDesc  string
		aliases    []string
		mutation   CommandMutation
		outputMode OutputModeContract
	}{
		{
			path:       "get job",
			batchDesc:  "number of jobs to request per page; does not cap total returned rows (max limit 1000 enforced by server)",
			limitDesc:  "maximum number of matching jobs to return across all pages; omit to continue through all matches",
			mutation:   CommandMutationReadOnly,
			outputMode: OutputModeContract{Name: "keys-only", Supported: true},
		},
		{
			path:       "get element",
			batchDesc:  "number of elements to request per page; does not cap total returned rows (max limit 1000 enforced by server)",
			limitDesc:  "maximum number of matching elements to return across all pages; omit to continue through all matches",
			aliases:    []string{"ei"},
			mutation:   CommandMutationReadOnly,
			outputMode: OutputModeContract{Name: "keys-only", Supported: true},
		},
		{
			path:       "get incident",
			batchDesc:  "number of incidents to request per page; does not cap total returned rows (max limit 1000 enforced by server)",
			limitDesc:  "maximum number of matching incidents to return across all pages; omit to continue through all matches",
			aliases:    []string{"incidents", "inc"},
			mutation:   CommandMutationReadOnly,
			outputMode: OutputModeContract{Name: "keys-only", Supported: true},
		},
		{
			path:       "get process-instance",
			batchDesc:  "number of process instances to request per page; does not cap total returned rows (max limit 1000 enforced by server)",
			limitDesc:  "maximum number of matching process instances to return across all pages; omit to continue through all matches",
			aliases:    []string{"process-instances", "pi", "pis"},
			mutation:   CommandMutationReadOnly,
			outputMode: OutputModeContract{Name: "keys-only", Supported: true},
		},
		{
			path:       "cancel process-instance",
			batchDesc:  "number of process instances to inspect per discovery page; does not cap total selected scope (max limit 1000 enforced by server)",
			limitDesc:  "maximum number of matching process instances to select for cancellation across all pages; omit to continue through all matches",
			aliases:    []string{"pi"},
			mutation:   CommandMutationStateChanging,
			outputMode: OutputModeContract{Name: "json", Supported: true, MachinePreferred: true},
		},
		{
			path:       "delete process-instance",
			batchDesc:  "number of process instances to inspect per discovery page; does not cap total frozen scope (max limit 1000 enforced by server)",
			limitDesc:  "maximum number of matching process instances to freeze for deletion across all pages; omit to continue through all matches",
			aliases:    []string{"pi"},
			mutation:   CommandMutationStateChanging,
			outputMode: OutputModeContract{Name: "json", Supported: true, MachinePreferred: true},
		},
		{
			path:       "ops execute retention-policy",
			batchDesc:  "number of process instances to inspect per discovery page; does not cap total frozen scope (max limit 1000 enforced by server)",
			limitDesc:  "maximum number of matching process instances to freeze for retention cleanup; omit to discover all matches",
			aliases:    []string{"ret-pol", "rp"},
			mutation:   CommandMutationStateChanging,
			outputMode: OutputModeContract{Name: "json", Supported: true, MachinePreferred: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			capability, ok := findCommandCapability(doc.Commands, tt.path)
			require.True(t, ok, "expected %q to appear in capability discovery", tt.path)
			require.Equal(t, tt.mutation, capability.Mutation)
			require.Equal(t, ContractSupportFull, capability.ContractSupport)
			require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
			require.Contains(t, capability.OutputModes, tt.outputMode)
			for _, alias := range tt.aliases {
				require.Contains(t, capability.Aliases, alias)
			}
			require.Contains(t, capability.Flags, FlagContract{
				Name:        "batch-size",
				Shorthand:   "n",
				Type:        "int32",
				Description: tt.batchDesc,
			})
			require.Contains(t, capability.Flags, FlagContract{
				Name:        "limit",
				Shorthand:   "l",
				Type:        "int32",
				Description: tt.limitDesc,
			})
		})
	}
}

// TestCapabilitiesCommand_OpsGroupingCommandsDoNotClaimFullAutomation keeps grouping-only ops commands out of full automation contracts.
func TestCapabilitiesCommand_OpsGroupingCommandsDoNotClaimFullAutomation(t *testing.T) {
	output := executeRootForTest(t, "capabilities", "--json")

	var doc CapabilityDocument
	require.NoError(t, json.Unmarshal([]byte(output), &doc))

	for _, path := range []string{"ops", "ops execute", "ops purge", "ops repair"} {
		capability, ok := findCommandCapability(doc.Commands, path)
		require.True(t, ok, "expected %q to appear in capability discovery", path)
		require.Equal(t, ContractSupportLimited, capability.ContractSupport)
		require.Equal(t, AutomationSupportUnsupported, capability.AutomationSupport)
		require.Empty(t, capability.AutomationNotes)
	}
}

func TestCapabilitiesCommand_AutomationJSONUsesOnlyStdoutForDocument(t *testing.T) {
	stdout, stderr := executeRootWithSeparateOutputsForTest(t, "--automation", "capabilities", "--json")

	var doc CapabilityDocument
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	require.Equal(t, "capabilities", doc.Command)
	require.Empty(t, stderr)
}

func TestCapabilitiesCommand_HelpDocumentsCanonicalAutomationSurface(t *testing.T) {
	output := executeRootForTest(t, "capabilities", "--help")

	require.Contains(t, output, "public c8volt command contract for scripts")
	require.Contains(t, output, "command paths, flags, output modes")
	require.Contains(t, output, "automation support")
	require.Contains(t, output, "./c8volt capabilities")
	require.Contains(t, output, "./c8volt capabilities --json")
}

func TestCapabilitiesHelpAndGeneratedMarkdownShareDiscoveryAnchors(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})

	helpOutput := executeRootForTest(t, "capabilities", "--help")
	markdown := renderMarkdownForCommand(t, capabilitiesCmd)

	for _, anchor := range []string{
		"public c8volt command contract for scripts",
		"command paths, flags, output modes",
		"automation support",
	} {
		require.Contains(t, helpOutput, anchor)
		require.Contains(t, markdown, anchor)
	}
}

func TestCapabilitiesCommand_DefaultOutputUsesHumanSummary(t *testing.T) {
	output := executeRootForTest(t, "capabilities")

	require.Contains(t, output, "Machine-readable public CLI capabilities")
	require.Contains(t, output, "Use --json for the full discovery document. Inspect automationSupport before using --automation.")
	require.Contains(t, output, "Hidden and shell-internal commands are excluded.")
	require.Contains(t, output, "- capabilities [read_only, limited, automation:full] modes: json, one-line")
	require.Contains(t, output, "- get [read_only, limited, automation:unsupported] modes: one-line, json, keys-only")
	require.Contains(t, output, "- run process-instance [state_changing, full, automation:full] modes: one-line, json, keys-only")
	require.NotContains(t, output, "\"command\":\"capabilities\"")
}

func TestResultEnvelopeForError_UsesSharedOutcomeMapping(t *testing.T) {
	envelope := resultEnvelopeForError(runProcessInstanceCmd, invalidFlagValuef("provide either --pd-key or --bpmn-process-id"))

	require.Equal(t, OutcomeInvalid, envelope.Outcome)
	require.Equal(t, "invalid_input", envelope.Class)
	require.Equal(t, "run process-instance", envelope.Command)
	require.NotNil(t, envelope.Detail)
	require.Contains(t, envelope.Detail.Message, "provide either --pd-key or --bpmn-process-id")
}
