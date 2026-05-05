// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestExpectCommand_CommandLocalBackoffTimeoutEnvOverridesProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_BACKOFF_TIMEOUT", "19s")

	cfg := resolveCommandConfigForTest(t, expectCmd, writeBackoffPrecedenceConfig(t), nil)

	require.Equal(t, 19*time.Second, cfg.App.Backoff.Timeout)
}

func TestExpectHelp_DocumentsWaitVerificationUsage(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"expect"}, []string{
		"Wait for process instances to satisfy expectations",
		"success depends on an",
		"./c8volt expect pi --key <process-instance-key> --state absent",
		"./c8volt expect pi --key <process-instance-key> --incident true",
	}, nil)
	require.Contains(t, output, "process-instance")

	output = assertCommandHelpOutput(t, []string{"expect", "process-instance"}, []string{
		"Use after `run`, `cancel`, or `delete`",
		"final state or incident marker is visible",
		"state expectation; valid values are: [active, completed, canceled, terminated, absent]",
		"incident expectation; valid values are: [true, false]",
		"./c8volt get pi --key <process-instance-key> --keys-only | ./c8volt expect pi --incident true -",
	}, nil)
	require.Contains(t, output, "--state")
	require.Contains(t, output, "--incident")
}

// Verifies expect process-instance rejects unsupported state values through invalid-input handling.
func TestExpectProcessInstanceCommand_RejectsInvalidStates(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestExpectProcessInstanceCommand_RejectsInvalidStatesHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "error parsing states")
}

func TestExpectProcessInstanceCommand_JSONInvalidStateUsesEnvelope(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestExpectProcessInstanceCommand_JSONInvalidStateUsesEnvelopeHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())

	var got map[string]any
	require.NoError(t, json.Unmarshal(output, &got))
	require.Equal(t, string(OutcomeInvalid), got["outcome"])
	require.Equal(t, "expect process-instance", got["command"])
}

func TestExpectProcessInstanceCommand_RejectsInvalidIncident(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestExpectProcessInstanceCommand_RejectsInvalidIncidentHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), `invalid value for --incident: "maybe"`)
	require.Contains(t, string(output), "valid values")
}

func TestExpectProcessInstanceCommand_RequiresAtLeastOneExpectation(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestExpectProcessInstanceCommand_RequiresAtLeastOneExpectationHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "at least one process instance expectation flag is required: --state or --incident")
	require.NotContains(t, string(output), `required flag(s) "state" not set`)
}

func TestExpectProcessInstanceCommand_RejectsAutomationMode(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestExpectProcessInstanceCommand_RejectsAutomationModeHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "expect process-instance does not support --automation")
}

func TestExpectProcessInstanceCommand_DashDoesNotRequireKeyFlag(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestHelperExpectProcessInstanceCommand_DashDoesNotRequireKeyFlag", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.NotContains(t, string(output), `required flag(s) "key" not set`)
}

func TestExpectProcessInstanceCommand_IncidentDashReadsKeysFromStdin(t *testing.T) {
	var attempts atomic.Int32
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/2251799813685255", r.URL.Path)

		attempts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813685255","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
  backoff:
    strategy: fixed
    initial_delay: 1ms
    max_retries: 3
    timeout: 100ms
auth:
  mode: none
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

	output, err := testx.RunCmdSubprocessWithStdin(t, "TestHelperExpectProcessInstanceCommand_IncidentDashReadsKeysFromStdin", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	}, "2251799813685255\n")
	require.NoError(t, err)
	require.Equal(t, int32(1), attempts.Load())
	require.Contains(t, string(output), `"key": "2251799813685255"`)
	require.Contains(t, string(output), `"incident": true`)
	require.Contains(t, string(output), `"ok": true`)
}

func TestExpectProcessInstanceCommand_StateDashReadsKeysFromStdin(t *testing.T) {
	var attempts atomic.Int32
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/2251799813685255", r.URL.Path)

		attempts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813685255","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
  backoff:
    strategy: fixed
    initial_delay: 1ms
    max_retries: 3
    timeout: 100ms
auth:
  mode: none
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

	output, err := testx.RunCmdSubprocessWithStdin(t, "TestHelperExpectProcessInstanceCommand_StateDashReadsKeysFromStdin", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	}, "2251799813685255\n")
	require.NoError(t, err)
	require.Equal(t, int32(1), attempts.Load())
	require.Contains(t, string(output), `"state": "ACTIVE"`)
	require.Contains(t, string(output), "2251799813685255")
}

func TestExpectProcessInstanceCommand_IncidentTrueWaitsUntilMatched(t *testing.T) {
	var attempts atomic.Int32
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/123", r.URL.Path)

		incident := attempts.Add(1) >= 2
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fmt.Sprintf(`{"hasIncident":%t,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, incident)))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
  backoff:
    strategy: fixed
    initial_delay: 1ms
    max_retries: 3
    timeout: 100ms
auth:
  mode: none
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"expect", "pi",
		"--key", "123",
		"--incident", "true",
	)

	require.Equal(t, int32(2), attempts.Load())
	require.Contains(t, output, `"incident": true`)
	require.Contains(t, output, `"ok": true`)
}

func TestExpectProcessInstanceCommand_IncidentFalseSucceedsForPresentIncidentFreeInstance(t *testing.T) {
	var attempts atomic.Int32
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/123", r.URL.Path)

		attempts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
  backoff:
    strategy: fixed
    initial_delay: 1ms
    max_retries: 3
    timeout: 100ms
auth:
  mode: none
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"expect", "pi",
		"--key", "123",
		"--incident", "false",
	)

	require.Equal(t, int32(1), attempts.Load())
	require.Contains(t, output, `"incident": false`)
	require.Contains(t, output, `"ok": true`)
}

func TestExpectProcessInstanceCommand_StateAndIncidentWaitUntilBothMatch(t *testing.T) {
	var attempts atomic.Int32
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/123", r.URL.Path)

		attempt := attempts.Add(1)
		state := "ACTIVE"
		incident := false
		if attempt == 2 {
			state = "COMPLETED"
			incident = true
		}
		if attempt >= 3 {
			state = "ACTIVE"
			incident = true
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fmt.Sprintf(`{"hasIncident":%t,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":%q,"tenantId":"tenant"}`, incident, state)))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
  backoff:
    strategy: fixed
    initial_delay: 1ms
    max_retries: 4
    timeout: 100ms
auth:
  mode: none
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"expect", "pi",
		"--key", "123",
		"--state", "active",
		"--incident", "true",
	)

	require.Equal(t, int32(3), attempts.Load())
	require.Contains(t, output, `"state": "ACTIVE"`)
	require.Contains(t, output, `"incident": true`)
	require.Contains(t, output, `"ok": true`)
}

// Helper-process entrypoint for invalid expect-state validation.
func TestExpectProcessInstanceCommand_RejectsInvalidStatesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "expect", "process-instance", "--key", "2251799813685255", "--state", "broken"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestExpectProcessInstanceCommand_JSONInvalidStateUsesEnvelopeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--json", "expect", "process-instance", "--key", "2251799813685255", "--state", "broken"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestExpectProcessInstanceCommand_RejectsInvalidIncidentHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "expect", "process-instance", "--key", "2251799813685255", "--incident", "maybe"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestExpectProcessInstanceCommand_RequiresAtLeastOneExpectationHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "expect", "pi", "--key", "2251799813685255"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestExpectProcessInstanceCommand_RejectsAutomationModeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--automation", "expect", "process-instance", "--key", "2251799813685255", "--state", "completed"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestHelperExpectProcessInstanceCommand_DashDoesNotRequireKeyFlag(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "expect", "process-instance", "--state", "active", "-"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestHelperExpectProcessInstanceCommand_IncidentDashReadsKeysFromStdin(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--json", "expect", "process-instance", "--incident", "true", "-"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestHelperExpectProcessInstanceCommand_StateDashReadsKeysFromStdin(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--json", "expect", "process-instance", "--state", "active", "-"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}
