package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCurrentBuildInfoIncludesSupportedCamundaVersions(t *testing.T) {
	t.Parallel()

	info := CurrentBuildInfo()

	require.Equal(t, "8.7, 8.8, 8.9", info.SupportedCamundaVersions)
}

func TestVersionCommandJSONIncludesSupportedCamundaVersions(t *testing.T) {
	t.Parallel()

	output := executeRootForTest(t, "version", "--json")

	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(output), &payload))
	require.Equal(t, "8.7, 8.8, 8.9", payload["supportedCamundaVersions"])
}
