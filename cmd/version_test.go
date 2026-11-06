// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"testing"

	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestCurrentBuildInfoIncludesSupportedCamundaVersions verifies build metadata
// reports the shared supported-version list used by the CLI.
func TestCurrentBuildInfoIncludesSupportedCamundaVersions(t *testing.T) {
	info := CurrentBuildInfo()

	require.Equal(t, toolx.SupportedCamundaVersionsString(), info.SupportedCamundaVersions)
}

// TestVersionCommandJSONIncludesSupportedCamundaVersions verifies JSON version
// output reports supported release lines without obsolete source-pin hints.
func TestVersionCommandJSONIncludesSupportedCamundaVersions(t *testing.T) {
	output := executeRootForTest(t, "version", "--json")

	var envelope struct {
		Outcome string            `json:"outcome"`
		Command string            `json:"command"`
		Payload map[string]string `json:"payload"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope.Outcome)
	require.Equal(t, "version", envelope.Command)
	payload := envelope.Payload
	require.Equal(t, toolx.SupportedCamundaVersionsString(), payload["supportedCamundaVersions"])
	require.Len(t, payload, 4)
	for _, field := range []string{"version", "commit", "date", "supportedCamundaVersions"} {
		require.Contains(t, payload, field)
	}
	requireSingleJSONValue(t, output)
}

// TestVersionCommand_DefaultOutputRemainsCompactPlainText verifies human output
// remains a compact plain text block while supported versions expand.
func TestVersionCommand_DefaultOutputRemainsCompactPlainText(t *testing.T) {
	output := executeRootForTest(t, "version")

	require.Contains(t, output, "c8volt ")
	require.Contains(t, output, "Supported Camunda versions: "+toolx.SupportedCamundaVersionsString())
	require.NotContains(t, output, "baseline")
	require.NotContains(t, output, "8.10.3")
	require.NotContains(t, output, `"outcome"`)
	require.NotContains(t, output, `"command"`)
}

// TestVersionHelp_DocumentsReadOnlyAutomationGuidance verifies help still
// advertises the script-friendly metadata path.
func TestVersionHelp_DocumentsReadOnlyAutomationGuidance(t *testing.T) {
	output := executeRootForTest(t, "version", "--help")

	require.Contains(t, output, "Show the c8volt version, build metadata")
	require.Contains(t, output, "c8volt version --json")
}
