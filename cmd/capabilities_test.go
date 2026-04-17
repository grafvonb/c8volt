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
	require.Equal(t, []OutputModeContract{
		{Name: "json", Supported: true, MachinePreferred: true, Notes: "canonical discovery format"},
		{Name: "one-line", Supported: false, Notes: "use JSON discovery output"},
	}, capabilitiesCapability.OutputModes)

	require.Equal(t, "get", getCapability.Path)
	require.Equal(t, ContractSupportLimited, getCapability.ContractSupport)
	require.Equal(t, CommandMutationReadOnly, getCapability.Mutation)

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
		Name:      "json",
		Supported: true,
		MachinePreferred: true,
	})
	require.Contains(t, processInstanceCapability.OutputModes, OutputModeContract{
		Name:      "keys-only",
		Supported: true,
	})
	require.Equal(t, ContractSupportFull, processInstanceCapability.ContractSupport)
	require.Equal(t, ContractSupportLimited, runCapability.ContractSupport)
	require.NotEmpty(t, runCapability.Children)
	require.Equal(t, ContractSupportFull, runCapability.Children[0].ContractSupport)
	require.Contains(t, runCapability.Children[0].OutputModes, OutputModeContract{
		Name:      "json",
		Supported: true,
		MachinePreferred: true,
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
}

func TestResultEnvelopeForError_UsesSharedOutcomeMapping(t *testing.T) {
	envelope := resultEnvelopeForError(runProcessInstanceCmd, invalidFlagValuef("provide either --pd-key or --bpmn-process-id"))

	require.Equal(t, OutcomeInvalid, envelope.Outcome)
	require.Equal(t, "invalid_input", envelope.Class)
	require.Equal(t, "run process-instance", envelope.Command)
	require.NotNil(t, envelope.Detail)
	require.Contains(t, envelope.Detail.Message, "provide either --pd-key or --bpmn-process-id")
}
