// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRunCommand_CommandLocalBackoffTimeoutFlagOverridesEnvProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_BACKOFF_TIMEOUT", "22s")

	cfg := resolveCommandConfigForTest(t, runCmd, writeBackoffPrecedenceConfig(t), func(cmd *cobra.Command) {
		require.NoError(t, cmd.PersistentFlags().Set("backoff-timeout", "44s"))
	})

	require.Equal(t, 44*time.Second, cfg.App.Backoff.Timeout)
}

func TestRunHelp_DocumentsWaitAndVerificationRouting(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"run"}, []string{
		"Start process instances",
		"waits for active instances by default",
		"--no-wait",
		"./c8volt run pi -b C88_SimpleUserTask_Process",
	}, nil)

	require.Contains(t, output, "process-instance")

	output = assertCommandHelpOutput(t, []string{"run", "process-instance"}, []string{
		"Run by BPMN process ID",
		"waits for active instances",
		"Add --no-wait to verify later with `get pi`, `expect pi`, or `walk pi`",
		"./c8volt expect pi --key <process-instance-key> --state active",
	}, nil)
	require.Contains(t, output, "--no-wait")
}

// Verifies run commands consume the profile selected by the root flag for tenant and API URL resolution.
func TestRunProcessInstanceCommand_ProfileFlagSelectsProfileTenantAndBaseURL(t *testing.T) {
	baseSrv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("base profile server should not be used: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(baseSrv.Close)

	prodSrv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	output, err := testx.RunCmdSubprocess(t, "TestRunProcessInstanceCommand_RejectsMutuallyExclusiveDefinitionFlagsHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "flags --pd-key and --bpmn-process-id are mutually exclusive")
}

func TestRunProcessInstanceCommand_JSONInvalidInputUsesEnvelope(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestRunProcessInstanceCommand_JSONInvalidInputUsesEnvelopeHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())

	var got map[string]any
	require.NoError(t, json.Unmarshal(output, &got))
	require.Equal(t, string(OutcomeInvalid), got["outcome"])
	require.Equal(t, "run process-instance", got["command"])
}

// Verifies run process-instance maps HTTP 409 responses to the conflict exit code.
func TestRunProcessInstanceCommand_ConflictUsesConflictExitCode(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-instances", r.URL.Path)
		http.Error(w, "already exists", http.StatusConflict)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestRunProcessInstanceCommand_ConflictUsesConflictExitCodeHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Conflict, exitErr.ExitCode())
	require.Contains(t, string(output), "conflict")
	require.Contains(t, string(output), "running process instance(s)")
}

func TestRunProcessInstanceCommand_V89NoWait(t *testing.T) {
	var sawRun bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-instances", r.URL.Path)
		sawRun = true
		defer r.Body.Close()
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "order-process", body["processDefinitionId"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","tenantId":"<default>","variables":{}}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
		"--no-wait",
	)

	require.True(t, sawRun)
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	require.Equal(t, string(OutcomeAccepted), got["outcome"])
	require.Equal(t, "run process-instance", got["command"])
	payload, ok := got["payload"].(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 1, payload["total"])
	require.Contains(t, stderr, "INFO")
}

func TestRunProcessInstanceCommand_DefaultOutputDoesNotEmitMachineEnvelope(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-instances", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","state":"ACTIVE","tenantId":"<default>","variables":{}}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
		"--no-wait",
	)

	require.NotContains(t, output, `"outcome"`)
	require.NotContains(t, output, `"command"`)
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

func TestRunProcessInstanceCommand_JSONInvalidInputUsesEnvelopeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--json", "run", "process-instance", "--pd-key", "2251799813685255", "--bpmn-process-id", "order-process"})
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
