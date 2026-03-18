package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	require.Contains(t, output, "license")
	require.Contains(t, output, "topology")
}

func TestGetClusterLicenseHelp(t *testing.T) {
	output := executeRootForTest(t, "get", "cluster", "license", "--help")

	require.Contains(t, output, "Get the cluster license of the connected Camunda 8 cluster")
	require.Contains(t, output, "Usage:")
	require.Contains(t, output, "c8volt get cluster license")
}

func TestGetClusterTopologyLegacyHelp(t *testing.T) {
	output := executeRootForTest(t, "get", "cluster-topology", "--help")

	require.Contains(t, output, "Get the cluster topology of the connected Camunda 8 cluster")
	require.Contains(t, output, "Deprecated but supported: use `c8volt get cluster topology`.")
}

func TestGetClusterTopologyNestedCommand_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/topology", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "brokers": [
    {
      "host": "camunda-platform-c88-zeebe-0.camunda-platform-c88-zeebe",
      "nodeId": 0,
      "partitions": [
        {
          "health": "healthy",
          "partitionId": 1,
          "role": "leader"
        }
      ],
      "port": 26501,
      "version": "8.8.0"
    }
  ],
  "clusterSize": 1,
  "gatewayVersion": "8.8.0",
  "partitionsCount": 1,
  "replicationFactor": 1,
  "lastCompletedChangeId": ""
}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	output := executeRootForTest(t, "--config", cfgPath, "get", "cluster", "topology")

	require.Contains(t, output, `"GatewayVersion": "8.8.0"`)
	require.Contains(t, output, `"ClusterSize": 1`)
}

func TestGetClusterLicenseCommand_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/license", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "licenseType": "SaaS",
  "validLicense": true
}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigWithVersion(t, srv.URL, "8.7")

	output := executeRootForTest(t, "--config", cfgPath, "get", "cluster", "license")

	require.Contains(t, output, `"licenseType": "SaaS"`)
	require.Contains(t, output, `"validLicense": true`)
	require.NotContains(t, output, `"expiresAt"`)
	require.NotContains(t, output, `"isCommercial"`)
}

func TestGetClusterLicenseCommand_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2/license", r.URL.Path)
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfig(t, srv.URL)

	cmd := exec.Command(os.Args[0], "-test.run=TestGetClusterLicenseCommand_FailureHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Unavailable, exitErr.ExitCode())
	require.Contains(t, string(output), "error fetching cluster license")
}

func TestGetClusterTopologyLegacyCommand_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/topology", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "brokers": [
    {
      "host": "camunda-platform-c88-zeebe-0.camunda-platform-c88-zeebe",
      "nodeId": 0,
      "partitions": [
        {
          "health": "healthy",
          "partitionId": 1,
          "role": "leader"
        }
      ],
      "port": 26501,
      "version": "8.8.0"
    }
  ],
  "clusterSize": 1,
  "gatewayVersion": "8.8.0",
  "partitionsCount": 1,
  "replicationFactor": 1,
  "lastCompletedChangeId": ""
}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	output := executeRootForTest(t, "--config", cfgPath, "get", "cluster-topology")

	require.Contains(t, output, `"GatewayVersion": "8.8.0"`)
	require.Contains(t, output, `"ClusterSize": 1`)
	require.NotContains(t, output, "Deprecated:")
}

func TestGetClusterTopologyLegacyAlias_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/topology", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "brokers": [],
  "clusterSize": 0,
  "gatewayVersion": "8.8.0",
  "partitionsCount": 0,
  "replicationFactor": 0,
  "lastCompletedChangeId": ""
}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	output := executeRootForTest(t, "--config", cfgPath, "get", "ct")

	require.Contains(t, output, `"GatewayVersion": "8.8.0"`)
	require.NotContains(t, output, "Deprecated:")
}

func TestGetClusterTopologyNestedCommand_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2/topology", r.URL.Path)
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfig(t, srv.URL)

	cmd := exec.Command(os.Args[0], "-test.run=TestGetClusterTopologyNestedCommand_FailureHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Unavailable, exitErr.ExitCode())
	require.Contains(t, string(output), "error fetching topology")
}

func TestGetClusterTopologyLegacyCommand_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2/topology", r.URL.Path)
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfig(t, srv.URL)

	cmd := exec.Command(os.Args[0], "-test.run=TestGetClusterTopologyLegacyCommand_FailureHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Unavailable, exitErr.ExitCode())
	require.Contains(t, string(output), "error fetching topology")
	require.NotContains(t, string(output), "Deprecated:")
}

func TestGetClusterTopologyNestedCommand_FailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "cluster", "topology"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestGetClusterLicenseCommand_FailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "cluster", "license"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestGetClusterTopologyLegacyCommand_FailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "cluster-topology"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func executeRootForTest(t *testing.T, args ...string) string {
	t.Helper()

	root := Root()
	resetCommandTreeFlags(root)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	_, err := root.ExecuteC()
	require.NoError(t, err)

	return buf.String()
}

func resetCommandTreeFlags(cmd *cobra.Command) {
	resetFlagSet := func(fs *pflag.FlagSet) {
		fs.VisitAll(func(flag *pflag.Flag) {
			_ = flag.Value.Set(flag.DefValue)
			flag.Changed = false
		})
	}

	resetFlagSet(cmd.Flags())
	resetFlagSet(cmd.PersistentFlags())
	resetFlagSet(cmd.InheritedFlags())

	for _, child := range cmd.Commands() {
		resetCommandTreeFlags(child)
	}
}

func writeTestConfig(t *testing.T, baseURL string) string {
	t.Helper()

	return writeTestConfigWithVersion(t, baseURL, "8.8")
}

func writeTestConfigWithVersion(t *testing.T, baseURL, camundaVersion string) string {
	t.Helper()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	content := fmt.Sprintf(`app:
  camunda_version: %q
auth:
  mode: none
apis:
  camunda_api:
    base_url: %q
`, camundaVersion, baseURL)
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o600))
	return cfgPath
}
