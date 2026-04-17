package cmd

import (
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
	for _, command := range doc.Commands {
		if command.Path == "get" {
			getCapability = command
			break
		}
	}
	require.Equal(t, "get", getCapability.Path)
	require.Equal(t, ContractSupportLimited, getCapability.ContractSupport)
	require.Equal(t, CommandMutationReadOnly, getCapability.Mutation)

	var processInstanceCapability CommandCapability
	for _, child := range getCapability.Children {
		if child.Path == "get process-instance" {
			processInstanceCapability = child
			break
		}
	}
	require.Equal(t, "get process-instance", processInstanceCapability.Path)
	require.Contains(t, processInstanceCapability.OutputModes, OutputModeContract{
		Name:      "json",
		Supported: true,
	})
	require.Contains(t, processInstanceCapability.OutputModes, OutputModeContract{
		Name:      "keys-only",
		Supported: true,
	})
}

func TestResultEnvelopeForError_UsesSharedOutcomeMapping(t *testing.T) {
	envelope := resultEnvelopeForError(runProcessInstanceCmd, invalidFlagValuef("provide either --pd-key or --bpmn-process-id"))

	require.Equal(t, OutcomeInvalid, envelope.Outcome)
	require.Equal(t, "invalid_input", envelope.Class)
	require.Equal(t, "run process-instance", envelope.Command)
	require.NotNil(t, envelope.Detail)
	require.Contains(t, envelope.Detail.Message, "provide either --pd-key or --bpmn-process-id")
}
