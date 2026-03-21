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

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

	output := executeRootForTest(t, "--config", cfgPath, "get", "cluster", "topology")

	require.Contains(t, output, `"GatewayVersion": "8.8.0"`)
	require.Contains(t, output, `"ClusterSize": 1`)
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

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

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

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

	output := executeRootForTest(t, "--config", cfgPath, "get", "ct")

	require.Contains(t, output, `"GatewayVersion": "8.8.0"`)
	require.NotContains(t, output, "Deprecated:")
}

func TestGetClusterLicenseNestedCommand_Success(t *testing.T) {
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

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

	output := executeRootForTest(t, "--config", cfgPath, "get", "cluster", "license")

	require.Contains(t, output, `"LicenseType": "SaaS"`)
	require.Contains(t, output, `"ValidLicense": true`)
	require.NotContains(t, output, `"ExpiresAt": null`)
	require.NotContains(t, output, `"IsCommercial": null`)
	require.NotContains(t, output, `"ExpiresAt"`)
	require.NotContains(t, output, `"IsCommercial"`)
}

func TestGetClusterLicenseNestedCommand_SuccessWithOptionalFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/license", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "expiresAt": "2030-01-02T03:04:05Z",
  "isCommercial": true,
  "licenseType": "Enterprise",
  "validLicense": true
}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForTest(t, "--config", cfgPath, "get", "cluster", "license")

	require.Contains(t, output, `"ExpiresAt": "2030-01-02T03:04:05Z"`)
	require.Contains(t, output, `"IsCommercial": true`)
	require.Contains(t, output, `"LicenseType": "Enterprise"`)
	require.Contains(t, output, `"ValidLicense": true`)
}

func TestGetClusterLicenseNestedCommand_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/license", r.URL.Path)
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfig(t, srv.URL)

	cmd := exec.Command(os.Args[0], "-test.run=TestGetClusterLicenseNestedCommand_FailureHelper")
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

func TestGetClusterLicenseNestedCommand_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2/license", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfig(t, srv.URL)

	cmd := exec.Command(os.Args[0], "-test.run=TestGetClusterLicenseNestedCommand_MalformedResponseHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "error fetching cluster license")
	require.Contains(t, string(output), "malformed response")
}

func TestGetProcessDefinitionXMLCommand_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-definitions/2251799813685255/xml", r.URL.Path)
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte("<definitions id=\"order-process\"/>"))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForTest(t, "--config", cfgPath, "get", "process-definition", "--key", "2251799813685255", "--xml")

	require.Equal(t, "<definitions id=\"order-process\"/>", output)
}

func TestGetProcessDefinitionXMLCommand_PreservesFormattedXMLBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-definitions/2251799813685255/xml", r.URL.Path)
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte("<definitions id=\"order-process\">\n  <process id=\"order\" />\n</definitions>\n"))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForTest(t, "--config", cfgPath, "get", "process-definition", "--key", "2251799813685255", "--xml")

	require.Equal(t, "<definitions id=\"order-process\">\n  <process id=\"order\" />\n</definitions>\n", output)
}

func TestGetProcessDefinitionByKeyCommand_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-definitions/2251799813685255", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "processDefinitionId": "order-process",
  "processDefinitionKey": "2251799813685255",
  "tenantId": "<default>",
  "version": 7,
  "versionTag": "stable"
}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForTest(t, "--config", cfgPath, "get", "process-definition", "--key", "2251799813685255")

	require.Contains(t, output, "2251799813685255")
	require.Contains(t, output, "<default> order-process v7/stable")
	require.NotContains(t, output, "<definitions")
}

func TestGetResourceCommand_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/resources/resource-id-123", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "resourceId": "resource-id-123",
  "resourceKey": "resource-key-123",
  "resourceName": "order-process.bpmn",
  "tenantId": "<default>",
  "version": 7,
  "versionTag": "stable"
}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForTest(t, "--config", cfgPath, "get", "resource", "--id", "resource-id-123")

	require.Contains(t, output, "resource-id-123")
	require.Contains(t, output, "k:resource-key-123")
	require.Contains(t, output, "<default>")
	require.Contains(t, output, "order-process.bpmn")
	require.Contains(t, output, "v7/stable")
}

func TestGetResourceCommand_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/resources/missing-resource", r.URL.Path)
		http.Error(w, "missing", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	cmd := exec.Command(os.Args[0], "-test.run=TestGetResourceCommand_FailureHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.NotFound, exitErr.ExitCode())
	require.Contains(t, string(output), "error fetching resource by id missing-resource")
}

func TestGetResourceCommand_RequiresID(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := executeRootExpectErrorForTest(t, "--config", cfgPath, "get", "resource")

	require.Error(t, err)
	require.Contains(t, err.Error(), "resource lookup requires a non-empty --id")
	require.Contains(t, output, "resource lookup requires a non-empty --id")
}

func TestGetResourceCommand_RejectsWhitespaceID(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := executeRootExpectErrorForTest(t, "--config", cfgPath, "get", "resource", "--id", "   ")

	require.Error(t, err)
	require.Contains(t, err.Error(), "resource lookup requires a non-empty --id")
	require.Contains(t, output, "resource lookup requires a non-empty --id")
}

func TestGetResourceCommand_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/resources/resource-id-123", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"detail":"missing payload"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	cmd := exec.Command(os.Args[0], "-test.run=TestGetResourceCommand_MalformedResponseHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "error fetching resource by id resource-id-123")
	require.Contains(t, string(output), "malformed response")
}

func TestGetProcessDefinitionXMLCommand_RequiresKey(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	cmd := exec.Command(os.Args[0], "-test.run=TestGetProcessDefinitionXMLCommand_RequiresKeyHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "xml output requires --key")
}

func TestGetProcessDefinitionXMLCommand_RejectsIncompatibleFlags(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	cmd := exec.Command(os.Args[0], "-test.run=TestGetProcessDefinitionXMLCommand_RejectsIncompatibleFlagsHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "xml output only supports --key")
	require.Contains(t, string(output), "--json")
	require.Contains(t, string(output), "--latest")
}

func TestGetProcessDefinitionXMLCommand_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-definitions/2251799813685255/xml", r.URL.Path)
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	cmd := exec.Command(os.Args[0], "-test.run=TestGetProcessDefinitionXMLCommand_FailureHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Unavailable, exitErr.ExitCode())
	require.Contains(t, string(output), "error fetching process definition xml by key 2251799813685255")
	require.NotContains(t, string(output), "<definitions")
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

func TestGetClusterLicenseNestedCommand_MalformedResponseHelper(t *testing.T) {
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

func TestGetClusterLicenseNestedCommand_FailureHelper(t *testing.T) {
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

func TestGetProcessDefinitionXMLCommand_RequiresKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-definition", "--xml"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestGetResourceCommand_FailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "resource", "--id", "missing-resource"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestGetResourceCommand_MalformedResponseHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "resource", "--id", "resource-id-123"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestGetProcessDefinitionXMLCommand_RejectsIncompatibleFlagsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--json", "get", "process-definition", "--key", "2251799813685255", "--xml", "--latest"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestGetProcessDefinitionXMLCommand_FailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-definition", "--key", "2251799813685255", "--xml"})
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

func executeRootExpectErrorForTest(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := Root()
	resetCommandTreeFlags(root)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	_, err := root.ExecuteC()
	return buf.String(), err
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
	return writeTestConfigForVersion(t, baseURL, "8.8")
}

func writeTestConfigForVersion(t *testing.T, baseURL string, camundaVersion string) string {
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
