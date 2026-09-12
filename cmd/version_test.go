// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestCurrentBuildInfoIncludesSupportedCamundaVersions verifies build metadata
// reports the shared supported-version list used by the CLI.
func TestCurrentBuildInfoIncludesSupportedCamundaVersions(t *testing.T) {
	info := CurrentBuildInfo()

	require.Equal(t, toolx.SupportedCamundaVersionsString(), info.SupportedCamundaVersions)
	require.Equal(t, toolx.V810Baseline().Tag, info.Camunda810Baseline)
	require.Equal(t, toolx.V810Baseline().Status, info.Camunda810BaselineStatus)
}

// TestVersionCommandJSONIncludesSupportedCamundaVersions verifies JSON version
// output follows the additive supported-version discovery contract.
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
	require.Equal(t, "8.10.0-alpha4", payload["camunda810Baseline"])
	require.Equal(t, "prerelease", payload["camunda810BaselineStatus"])
}

// TestVersionCommandSeparatesBaselineMetadataFromVersionIdentity verifies
// baseline tag/status updates cannot leak into supported-version discovery.
func TestVersionCommandSeparatesBaselineMetadataFromVersionIdentity(t *testing.T) {
	jsonOutput := executeRootForTest(t, "version", "--json")

	var envelope struct {
		Payload map[string]string `json:"payload"`
	}
	require.NoError(t, json.Unmarshal([]byte(jsonOutput), &envelope))

	supportedVersions := envelope.Payload["supportedCamundaVersions"]
	baselineTag := envelope.Payload["camunda810Baseline"]
	baselineStatus := envelope.Payload["camunda810BaselineStatus"]
	require.Equal(t, "8.7, 8.8, 8.9, 8.10", supportedVersions)
	require.Equal(t, toolx.V810Baseline().Tag, baselineTag)
	require.Equal(t, toolx.V810Baseline().Status, baselineStatus)
	require.NotContains(t, supportedVersions, baselineTag)
	require.NotContains(t, supportedVersions, baselineStatus)
	require.NotContains(t, supportedVersions, "alpha")
	require.NotContains(t, supportedVersions, "rc")
	require.NotContains(t, supportedVersions, "8.10.0")

	humanOutput := executeRootForTest(t, "version")
	supportedLine := firstVersionOutputLineContaining(t, humanOutput, "Supported Camunda versions:")
	require.Contains(t, supportedLine, "Supported Camunda versions: "+supportedVersions)
	require.NotContains(t, supportedLine, baselineTag)
	require.Contains(t, humanOutput, "Camunda 8.10 baseline: "+baselineTag+" ("+baselineStatus+")")
}

// TestVersionCommand_DefaultOutputRemainsCompactPlainText verifies human output
// remains a compact plain text block while supported versions expand.
func TestVersionCommand_DefaultOutputRemainsCompactPlainText(t *testing.T) {
	output := executeRootForTest(t, "version")

	require.Contains(t, output, "c8volt ")
	require.Contains(t, output, "Supported Camunda versions: "+toolx.SupportedCamundaVersionsString())
	require.Contains(t, output, "Camunda 8.10 baseline: 8.10.0-alpha4 (prerelease)")
	require.NotContains(t, output, `"outcome"`)
	require.NotContains(t, output, `"command"`)
}

// TestVersionHelp_DocumentsReadOnlyAutomationGuidance verifies help still
// advertises the script-friendly metadata path.
func TestVersionHelp_DocumentsReadOnlyAutomationGuidance(t *testing.T) {
	output := executeRootForTest(t, "version", "--help")

	require.Contains(t, output, "Show the c8volt version, build metadata")
	require.Contains(t, output, "./c8volt version --json")
}

// firstVersionOutputLineContaining extracts a single line from version output
// while tolerating the logger prefix present in command test harness output.
func firstVersionOutputLineContaining(t *testing.T, output string, text string) string {
	t.Helper()

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, text) {
			return line
		}
	}
	require.Failf(t, "missing version output line", "text %q not found in %q", text, output)
	return ""
}
