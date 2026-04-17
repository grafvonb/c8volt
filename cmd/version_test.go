package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCurrentBuildInfoIncludesSupportedCamundaVersions(t *testing.T) {
	info := CurrentBuildInfo()

	require.Equal(t, "8.7, 8.8, 8.9", info.SupportedCamundaVersions)
}

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
	require.Equal(t, "8.7, 8.8, 8.9", payload["supportedCamundaVersions"])
}

func TestVersionCommand_DefaultOutputRemainsCompactPlainText(t *testing.T) {
	output := executeRootForTest(t, "version")

	require.Contains(t, output, "c8volt ")
	require.Contains(t, output, "camunda: 8.7, 8.8, 8.9")
	require.NotContains(t, output, `"outcome"`)
	require.NotContains(t, output, `"command"`)
}
