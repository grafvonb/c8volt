package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootHelp_PreservesHumanTaxonomyAndDiscoveryCommand(t *testing.T) {
	output := executeRootForTest(t, "--help")

	require.Contains(t, output, "get")
	require.Contains(t, output, "run")
	require.Contains(t, output, "expect")
	require.Contains(t, output, "walk")
	require.Contains(t, output, "deploy")
	require.Contains(t, output, "delete")
	require.Contains(t, output, "cancel")
	require.Contains(t, output, "config")
	require.Contains(t, output, "embed")
	require.Contains(t, output, "version")
	require.Contains(t, output, "capabilities")
	require.Contains(t, output, "For machine discovery, use \"c8volt capabilities --json\"")
}
