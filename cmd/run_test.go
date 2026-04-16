package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// Verifies run commands consume the profile selected by the root flag for tenant and API URL resolution.
func TestRunProcessInstanceCommand_ProfileFlagSelectsProfileTenantAndBaseURL(t *testing.T) {
	baseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("base profile server should not be used: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(baseSrv.Close)

	prodSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-instances", r.URL.Path)
		defer r.Body.Close()
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "profile-tenant", body["tenantId"])
		require.Equal(t, "order-process", body["processDefinitionId"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionVersion":1,"tenantId":"profile-tenant","variables":{}}`))
	}))
	t.Cleanup(prodSrv.Close)

	cfgPath := writeRawTestConfig(t, `active_profile: base
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: `+baseSrv.URL+`
profiles:
  prod:
    app:
      tenant: profile-tenant
    apis:
      camunda_api:
        base_url: `+prodSrv.URL+`
`)

	output, err := testx.RunCmdSubprocess(t, "TestRunProcessInstanceCommand_ProfileFlagSelectsProfileTenantAndBaseURLHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.NoError(t, err, string(output))
}

// Verifies run process-instance rejects mutually exclusive definition selectors.
func TestRunProcessInstanceCommand_RejectsMutuallyExclusiveDefinitionFlags(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	cmd := exec.Command(os.Args[0], "-test.run=TestRunProcessInstanceCommand_RejectsMutuallyExclusiveDefinitionFlagsHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "flags --pd-key and --bpmn-process-id are mutually exclusive")
}

// Verifies run process-instance maps HTTP 409 responses to the conflict exit code.
func TestRunProcessInstanceCommand_ConflictUsesConflictExitCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-instances", r.URL.Path)
		http.Error(w, "already exists", http.StatusConflict)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	cmd := exec.Command(os.Args[0], "-test.run=TestRunProcessInstanceCommand_ConflictUsesConflictExitCodeHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Conflict, exitErr.ExitCode())
	require.Contains(t, string(output), "conflict")
	require.Contains(t, string(output), "running process instance(s)")
}

// Helper-process entrypoint for mutually-exclusive definition-flag validation.
func TestRunProcessInstanceCommand_RejectsMutuallyExclusiveDefinitionFlagsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "run", "process-instance", "--pd-key", "2251799813685255", "--bpmn-process-id", "order-process"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// Helper-process entrypoint for conflict exit-code mapping validation.
func TestRunProcessInstanceCommand_ConflictUsesConflictExitCodeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "run", "process-instance", "--bpmn-process-id", "order-process", "--no-wait"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestRunProcessInstanceCommand_ProfileFlagSelectsProfileTenantAndBaseURLHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	flagRunPIProcessDefinitionKey = nil
	flagRunPIProcessDefinitionBpmnProcessIds = nil

	root := Root()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"--profile", "prod",
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
		"--no-wait",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}
