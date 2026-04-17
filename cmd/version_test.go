package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCurrentBuildInfoIncludesSupportedCamundaVersions(t *testing.T) {
	t.Parallel()

	info := CurrentBuildInfo()

	require.Equal(t, "8.7, 8.8, 8.9", info.SupportedCamundaVersions)
}
