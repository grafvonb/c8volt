package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetHelp(t *testing.T) {
	output := executeRootForTest(t, "get", "--help")

	require.Contains(t, output, "Get resources")
	require.Contains(t, output, "cluster")
	require.Contains(t, output, "cluster-topology")
}

func TestGetClusterHelp(t *testing.T) {
	output := executeRootForTest(t, "get", "cluster", "--help")

	require.Contains(t, output, "Get cluster resources")
	require.Contains(t, output, "Usage:")
	require.Contains(t, output, "c8volt get cluster")
}

func executeRootForTest(t *testing.T, args ...string) string {
	t.Helper()

	root := Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	_, err := root.ExecuteC()
	require.NoError(t, err)

	return buf.String()
}
