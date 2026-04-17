package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestDeployCommand_CommandLocalBackoffTimeoutFlagOverridesEnvProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_BACKOFF_TIMEOUT", "18s")

	cfg := resolveCommandConfigForTest(t, deployCmd, writeBackoffPrecedenceConfig(t), func(cmd *cobra.Command) {
		require.NoError(t, cmd.PersistentFlags().Set("backoff-timeout", "41s"))
	})

	require.Equal(t, 41*time.Second, cfg.App.Backoff.Timeout)
}

func TestDeployProcessDefinitionCommand_TenantFlagOverridesEnvProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_TENANT", "env-tenant")

	var sawDeploy bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			sawDeploy = true
			require.NoError(t, r.ParseMultipartForm(1<<20))
			require.Equal(t, "flag-tenant", r.FormValue("tenantId"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tenantId":"flag-tenant"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeRawTestConfig(t, `active_profile: dev
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: `+srv.URL+`
profiles:
  dev:
    app:
      tenant: profile-tenant
`)
	bpmnPath := writeTempFile(t, "order-process.bpmn", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="order-process" isExecutable="true" />
</bpmn:definitions>`))

	output, err := testx.RunCmdSubprocess(t, "TestDeployProcessDefinitionCommand_TenantFlagOverridesEnvProfileAndConfigHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":    cfgPath,
		"C8VOLT_TEST_BPMN_PATH": bpmnPath,
	})
	require.NoError(t, err, string(output))
	require.True(t, sawDeploy)
}

func TestDeployProcessDefinitionCommand_RunFallsBackToBPMNIDForV87(t *testing.T) {
	var sawDeploy bool
	var sawRun bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			sawDeploy = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tenantId":"<default>"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances":
			sawRun = true
			defer r.Body.Close()
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "order-process", body["processDefinitionId"])
			require.Equal(t, "<default>", body["tenantId"])
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionVersion":1,"tenantId":"<default>","variables":{}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")
	bpmnPath := writeTempFile(t, "order-process.bpmn", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="order-process" isExecutable="true" />
</bpmn:definitions>`))

	output, err := testx.RunCmdSubprocess(t, "TestDeployProcessDefinitionCommand_RunFallsBackToBPMNIDForV87Helper", map[string]string{
		"C8VOLT_TEST_CONFIG":    cfgPath,
		"C8VOLT_TEST_BPMN_PATH": bpmnPath,
	})
	require.NoError(t, err, string(output))

	require.True(t, sawDeploy)
	require.True(t, sawRun)
}

func TestDeployProcessDefinitionCommand_V89NoWait(t *testing.T) {
	var sawDeploy bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			sawDeploy = true
			require.NoError(t, r.ParseMultipartForm(1<<20))
			require.Equal(t, "order-process.bpmn", r.MultipartForm.File["resources"][0].Filename)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"deploymentKey":"deployment-123","tenantId":"<default>","deployments":[{"processDefinition":{"processDefinitionId":"order-process","processDefinitionKey":"2251799813685255","processDefinitionVersion":3,"resourceName":"order-process.bpmn","tenantId":"<default>"}}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
	bpmnPath := writeTempFile(t, "order-process.bpmn", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="order-process" isExecutable="true" />
</bpmn:definitions>`))

	output, err := testx.RunCmdSubprocess(t, "TestDeployProcessDefinitionCommand_V89NoWaitHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":    cfgPath,
		"C8VOLT_TEST_BPMN_PATH": bpmnPath,
	})

	require.NoError(t, err, string(output))
	require.True(t, sawDeploy)
	require.Contains(t, string(output), "2251799813685255")
}

func TestDeployProcessDefinitionCommand_RunFallsBackToBPMNIDForV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"deploy", "process-definition",
		"--file", os.Getenv("C8VOLT_TEST_BPMN_PATH"),
		"--run",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestDeployProcessDefinitionCommand_TenantFlagOverridesEnvProfileAndConfigHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"--tenant", "flag-tenant",
		"deploy", "process-definition",
		"--file", os.Getenv("C8VOLT_TEST_BPMN_PATH"),
		"--no-wait",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestDeployProcessDefinitionCommand_V89NoWaitHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"deploy", "process-definition",
		"--file", os.Getenv("C8VOLT_TEST_BPMN_PATH"),
		"--no-wait",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// Verifies deploy process-definition rejects multiple stdin markers in --file arguments.
func TestDeployProcessDefinitionCommand_RejectsRepeatedStdinFile(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestDeployProcessDefinitionCommand_RejectsRepeatedStdinFileHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "only one '-' (stdin) allowed")
}

func writeTempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := t.TempDir() + string(os.PathSeparator) + name
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return path
}

// Helper-process entrypoint for repeated-stdin-file validation.
func TestDeployProcessDefinitionCommand_RejectsRepeatedStdinFileHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "deploy", "process-definition", "--file", "-", "--file", "-"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}
