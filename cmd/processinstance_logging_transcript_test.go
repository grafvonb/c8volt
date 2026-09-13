// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const (
	processInstanceLoggingTimeoutHelper = "TestProcessInstanceDeleteCancellationTimeoutTranscriptHelper"
	processInstanceLoggingSuccessHelper = "TestProcessInstanceDeleteCancellationSuccessTranscriptHelper"
	processInstancePollingBudgetHelper  = "TestProcessInstancePollingRecordBudgetHelper"
)

// TestProcessInstanceDeleteCancellationTimeoutTranscript reproduces child
// deletion conflict, accepted root cancellation, continued ACTIVE observations,
// and confirmation timeout through the real command and HTTP adapter path.
func TestProcessInstanceDeleteCancellationTimeoutTranscript(t *testing.T) {
	requests := &testx.SafeSlice[string]{}
	server := testx.NewIPv4Server(t, processInstanceLoggingTimeoutServer(t, requests))
	t.Cleanup(server.Close)
	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.9
  backoff:
    strategy: fixed
    initial_delay: 1ms
    max_retries: 0
    timeout: 30ms
auth:
  mode: none
apis:
  camunda_api:
    base_url: `+server.URL+`
`)

	stdout, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, processInstanceLoggingTimeoutHelper, "", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	}, "")
	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Timeout, exitErr.ExitCode())
	require.Empty(t, stdout)

	require.Contains(t, stderr, "root failed: cancellation confirmation timed out after 30ms")
	require.Contains(t, stderr, "root root; scope child,root; last observed child=ACTIVE, root=ACTIVE")
	require.Contains(t, stderr, "child child deletion conflicted")
	require.Contains(t, stderr, "root cancellation submitted, outcome unconfirmed")
	require.Contains(t, stderr, "resumed deletion not reached")
	require.Contains(t, stderr, "ERROR delete process instances: cancellation confirmation timed out for root root (1 tree); cancellation submitted, outcome unconfirmed")
	require.NotContains(t, strings.ToLower(stderr), "rollback")
	require.NotContains(t, strings.ToLower(stderr), "cancellation failed")

	gotRequests := requests.Snapshot()
	require.NotEmpty(t, gotRequests)
	require.Contains(t, gotRequests, "POST /v2/process-instances/child/deletion")
	require.Contains(t, gotRequests, "POST /v2/process-instances/root/cancellation")
	require.NotContains(t, gotRequests, "POST /v2/process-instances/root/deletion")
	require.Less(t, indexOfString(gotRequests, "POST /v2/process-instances/child/deletion"), indexOfString(gotRequests, "POST /v2/process-instances/root/cancellation"))

	jsonStdout, jsonStderr, jsonErr := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, processInstanceLoggingTimeoutHelper, "", map[string]string{
		"C8VOLT_TEST_CONFIG":      cfgPath,
		"C8VOLT_TEST_OUTPUT_MODE": "json",
	}, "")
	require.Error(t, jsonErr)
	jsonExitErr, ok := jsonErr.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Timeout, jsonExitErr.ExitCode())
	require.NotContains(t, jsonStderr, `"outcome"`, "diagnostics must remain separate from the machine result")
	decoder := json.NewDecoder(bytes.NewBufferString(jsonStdout))
	var envelope decodedCommandErrorEnvelope
	require.NoError(t, decoder.Decode(&envelope))
	require.ErrorIs(t, decoder.Decode(&struct{}{}), io.EOF)
	require.Equal(t, "failed", envelope.Outcome)
	require.Equal(t, "timeout", envelope.Class)
	require.Equal(t, "delete process-instance", envelope.Command)
	require.NotNil(t, envelope.Detail)
	require.Equal(t, "timeout", envelope.Detail.Class)
	require.Contains(t, envelope.Detail.Message, "delete process instances: operation timed out:")
	require.Contains(t, envelope.Detail.Message, "deleting child process instance with key child of process instance with key root:")
	require.Contains(t, envelope.Detail.Message, "delete cancel: cancel wait:")

	for _, tc := range []struct {
		name        string
		mode        string
		wantVerbose bool
		wantDebug   bool
	}{
		{name: "VerboseOnly", mode: "verbose", wantVerbose: true},
		{name: "DebugOnly", mode: "debug", wantDebug: true},
		{name: "VerboseAndDebug", mode: "both", wantVerbose: true, wantDebug: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, modeStderr, modeErr := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, processInstanceLoggingTimeoutHelper, "", map[string]string{
				"C8VOLT_TEST_CONFIG":       cfgPath,
				"C8VOLT_TEST_LOGGING_MODE": tc.mode,
			}, "")
			require.Error(t, modeErr)
			verboseRecords := []string{
				"pi deletion conflicted: key=child cancellation required before deletion can proceed",
				"pi cancellation escalated: requested=child root=root reason=cancel affected tree",
				"pi cancellation submitted: root=root accepted=true",
				"pi wait: phase=cancellation confirmation root=root scope=[root child] states=[COMPLETED CANCELED TERMINATED ABSENT] timeout=30ms backoff=fixed initial_delay=1ms max_retries=0",
			}
			for _, record := range verboseRecords {
				if tc.wantVerbose {
					require.Equal(t, 1, strings.Count(modeStderr, record), "%s\nstderr:\n%s", record, modeStderr)
				} else {
					require.NotContains(t, modeStderr, record)
				}
			}
			require.NotContains(t, modeStderr, "resuming deletion")
			if tc.wantDebug {
				require.Contains(t, modeStderr, "pi state observation:")
				require.Contains(t, modeStderr, "api #")
				require.Equal(t, 1, strings.Count(modeStderr, "process-instance mutation failure"), "full chain must be logged once at DEBUG\nstderr:\n%s", modeStderr)
			} else {
				require.NotContains(t, modeStderr, "pi state observation:")
				require.NotContains(t, modeStderr, "api #")
			}
		})
	}

	for _, tc := range []struct {
		name       string
		guard      string
		outputMode string
	}{
		{name: "QuietVerbose", guard: "quiet"},
		{name: "AutomationVerbose", guard: "automation"},
		{name: "JSONVerbose", outputMode: "json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			guardStdout, guardStderr, guardErr := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, processInstanceLoggingTimeoutHelper, "", map[string]string{
				"C8VOLT_TEST_CONFIG":       cfgPath,
				"C8VOLT_TEST_LOGGING_MODE": "verbose",
				"C8VOLT_TEST_GUARD_MODE":   tc.guard,
				"C8VOLT_TEST_OUTPUT_MODE":  tc.outputMode,
			}, "")
			require.Error(t, guardErr)
			require.NotContains(t, guardStderr, "pi deletion conflicted:")
			require.NotContains(t, guardStderr, "pi cancellation escalated:")
			require.NotContains(t, guardStderr, "pi cancellation submitted:")
			require.NotContains(t, guardStderr, "pi wait: phase=")
			if tc.outputMode == "json" {
				decoder := json.NewDecoder(strings.NewReader(guardStdout))
				require.NoError(t, decoder.Decode(&decodedCommandErrorEnvelope{}))
				require.ErrorIs(t, decoder.Decode(&struct{}{}), io.EOF)
			}
		})
	}
}

// TestFormatProcessInstanceMutationCommandFailures verifies joined tree
// failures produce a deterministic compact aggregate without state detail.
func TestFormatProcessInstanceMutationCommandFailures(t *testing.T) {
	t.Parallel()

	failures := []*ferrors.ProcessInstanceMutationFailure{
		{Operation: "delete", Phase: "cancellation confirmation", RootKey: "root-b", Timeout: 30 * time.Millisecond, LastStates: map[string]string{"root-b": "ACTIVE"}, CancellationSubmitted: true},
		{Operation: "delete", Phase: "cancellation confirmation", RootKey: "root-a", Timeout: 30 * time.Millisecond, CancellationSubmitted: true},
	}

	require.Equal(t,
		"delete process instances: cancellation confirmation timed out for roots root-a,root-b (2 trees); cancellations submitted, outcomes unconfirmed",
		formatProcessInstanceMutationCommandFailures(failures),
	)
}

func TestProcessInstanceDeleteCancellationTimeoutTranscriptHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, processInstanceLoggingTimeoutHelper, os.Getenv(testx.CmdSubprocessNameEnv))
	os.Args = processInstanceLoggingHelperArgs()
	if os.Getenv("C8VOLT_TEST_OUTPUT_MODE") == "json" {
		os.Args = append(os.Args[:3], append([]string{"--json"}, os.Args[3:]...)...)
	}
	Execute()
}

// TestProcessInstanceDeleteCancellationSuccessTranscript proves verbose narration follows
// accepted cancellation through both confirmation waits and resumed deletion.
func TestProcessInstanceDeleteCancellationSuccessTranscript(t *testing.T) {
	requests := &testx.SafeSlice[string]{}
	server := testx.NewIPv4Server(t, processInstanceLoggingSuccessServer(t, requests))
	t.Cleanup(server.Close)
	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.9
  backoff:
    strategy: fixed
    initial_delay: 1ms
    max_retries: 4
    timeout: 200ms
auth:
  mode: none
apis:
  camunda_api:
    base_url: `+server.URL+`
`)

	_, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, processInstanceLoggingSuccessHelper, "", map[string]string{
		"C8VOLT_TEST_CONFIG":       cfgPath,
		"C8VOLT_TEST_LOGGING_MODE": "verbose",
	}, "")
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(stderr, "pi deletion conflicted: key=child cancellation required before deletion can proceed"), "stderr:\n%s", stderr)
	require.Equal(t, 1, strings.Count(stderr, "pi cancellation escalated: requested=child root=root reason=cancel affected tree"))
	require.Equal(t, 1, strings.Count(stderr, "pi cancellation submitted: root=root accepted=true"))
	require.Equal(t, 1, strings.Count(stderr, "pi wait: phase=cancellation confirmation root=root"))
	require.Equal(t, 1, strings.Count(stderr, "pi wait: phase=deletion prerequisite confirmation root=child scope=[child]"))
	require.Equal(t, 1, strings.Count(stderr, "pi cancellation confirmed: key=child resuming deletion"))
	require.NotContains(t, stderr, "pi state observation:")
	require.NotContains(t, stderr, "api #")

	gotRequests := requests.Snapshot()
	require.Contains(t, gotRequests, "POST /v2/process-instances/root/cancellation")
	require.Contains(t, gotRequests, "POST /v2/process-instances/root/deletion")
	require.Less(t, indexOfString(gotRequests, "POST /v2/process-instances/root/cancellation"), indexOfString(gotRequests, "POST /v2/process-instances/root/deletion"))
}

func TestProcessInstanceDeleteCancellationSuccessTranscriptHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, processInstanceLoggingSuccessHelper, os.Getenv(testx.CmdSubprocessNameEnv))
	os.Args = processInstanceLoggingHelperArgs()
	Execute()
}

func processInstanceLoggingHelperArgs() []string {
	args := []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--no-indicator"}
	switch os.Getenv("C8VOLT_TEST_LOGGING_MODE") {
	case "verbose":
		args = append(args, "--verbose")
	case "debug":
		args = append(args, "--debug", "--log-format", "plain")
	case "both":
		args = append(args, "--verbose", "--debug", "--log-format", "plain")
	}
	switch os.Getenv("C8VOLT_TEST_GUARD_MODE") {
	case "quiet":
		args = append(args, "--quiet")
	case "automation":
		args = append(args, "--automation")
	}
	return append(args, "--auto-confirm", "delete", "process-instance", "--key", "root", "--force")
}

// TestProcessInstancePollingRecordBudget proves 36 cached-auth state checks
// emit only one observation and one HTTP diagnostic apiece.
func TestProcessInstancePollingRecordBudget(t *testing.T) {
	const key = "record-budget"
	const accessToken = "polling-access-token-never-log"
	var tokenRequests atomic.Int32
	var pollingRequests atomic.Int32
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/oauth/token":
			tokenRequests.Add(1)
			_, _ = io.WriteString(w, `{"access_token":"`+accessToken+`","expires_in":120,"token_type":"Bearer"}`)
		case "/v2/process-instances/" + key:
			require.Equal(t, "Bearer "+accessToken, r.Header.Get("Authorization"))
			attempt := pollingRequests.Add(1)
			require.LessOrEqual(t, attempt, int32(36))
			state := "ACTIVE"
			if attempt == 36 {
				state = "CANCELED"
			}
			_, _ = fmt.Fprintf(w, `{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":1,"processInstanceKey":%q,"startDate":"2026-09-13T12:00:00Z","state":%q,"tenantId":"tenant"}`, key, state)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.9
  backoff:
    strategy: fixed
    initial_delay: 1ms
    max_retries: 36
    timeout: 2s
auth:
  mode: oauth2
  oauth2:
    token_url: `+server.URL+`/oauth
    client_id: polling-client
    client_secret: polling-client-secret-never-log
    scopes:
      camunda_api: polling-scope
apis:
  camunda_api:
    base_url: `+server.URL+`
    require_scope: true
`)

	_, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, processInstancePollingBudgetHelper, "", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	}, "")
	require.NoError(t, err)
	require.Equal(t, int32(1), tokenRequests.Load())
	require.Equal(t, int32(36), pollingRequests.Load())
	require.Equal(t, 1, strings.Count(stderr, "api #1 POST /oauth/token: status=200"))
	require.NotContains(t, stderr, "auth token cache lookup")
	require.NotContains(t, stderr, "auth token cache hit")
	require.NotContains(t, stderr, "polling-client-secret-never-log")
	require.NotContains(t, stderr, accessToken)

	pollingRecords := 0
	for attempt := 1; attempt <= 36; attempt++ {
		observation := fmt.Sprintf("pi state observation: key=%s attempt=%d state=", key, attempt)
		require.Equal(t, 1, strings.Count(stderr, observation))
	}
	for line := range strings.SplitSeq(stderr, "\n") {
		if strings.Contains(line, "pi state observation: key="+key+" ") {
			pollingRecords++
		}
		if strings.Contains(line, " GET /v2/process-instances/"+key+": status=200") {
			pollingRecords++
		}
	}
	require.Equal(t, 72, pollingRecords, "polling scope must contain 36 observations and 36 HTTP records")
	require.Equal(t, 36, strings.Count(stderr, " GET /v2/process-instances/"+key+": status=200"))
	require.Contains(t, stderr, "pi state observation: key="+key+" attempt=35 state=ACTIVE")
	finalObservation := "pi state observation: key=" + key + " attempt=36 state=CANCELED"
	require.Contains(t, stderr, finalObservation)
	require.NotContains(t, stderr[strings.Index(stderr, finalObservation):], "next_delay=")
}

// TestProcessInstancePollingRecordBudgetHelper runs the real command path for
// the deterministic cached-auth polling budget fixture.
func TestProcessInstancePollingRecordBudgetHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, processInstancePollingBudgetHelper, os.Getenv(testx.CmdSubprocessNameEnv))
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--log-format", "plain", "--debug", "--no-indicator", "expect", "process-instance", "--key", "record-budget", "--state", "canceled"}
	Execute()
}

func processInstanceLoggingTimeoutServer(t *testing.T, requests *testx.SafeSlice[string]) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			parent := ""
			if key == "child" {
				parent = "root"
			}
			_, _ = fmt.Fprintf(w, `{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":1,"processInstanceKey":%q,"parentProcessInstanceKey":%q,"rootProcessInstanceKey":"root","startDate":"2026-09-13T12:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key, parent)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			items := ""
			if strings.Contains(string(body), `"parentProcessInstanceKey":"root"`) {
				items = `{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":1,"processInstanceKey":"child","parentProcessInstanceKey":"root","rootProcessInstanceKey":"root","startDate":"2026-09-13T12:00:00Z","state":"ACTIVE","tenantId":"tenant"}`
			}
			if items == "" {
				_, _ = io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
			} else {
				_, _ = io.WriteString(w, `{"items":[`+items+`],"page":{"totalItems":1}}`)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/child/deletion":
			w.WriteHeader(http.StatusConflict)
			_, _ = io.WriteString(w, `{"title":"Conflict","status":409,"detail":"active instance"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/root/cancellation":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s body=%s", r.Method, r.URL.Path, body)
		}
	})
}

func processInstanceLoggingSuccessServer(t *testing.T, requests *testx.SafeSlice[string]) http.Handler {
	t.Helper()
	var mu sync.Mutex
	cancellationSubmitted := false
	deleted := map[string]bool{}
	deleteAttempts := map[string]int{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if deleted[key] {
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `{"title":"Not Found","status":404,"detail":"missing"}`)
				return
			}
			parent := ""
			if key == "child" {
				parent = "root"
			}
			state := "ACTIVE"
			if cancellationSubmitted {
				state = "CANCELED"
			}
			_, _ = fmt.Fprintf(w, `{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":1,"processInstanceKey":%q,"parentProcessInstanceKey":%q,"rootProcessInstanceKey":"root","startDate":"2026-09-13T12:00:00Z","state":%q,"tenantId":"tenant"}`, key, parent, state)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			if strings.Contains(string(body), `"parentProcessInstanceKey":"root"`) && !deleted["child"] {
				state := "ACTIVE"
				if cancellationSubmitted {
					state = "CANCELED"
				}
				_, _ = fmt.Fprintf(w, `{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":1,"processInstanceKey":"child","parentProcessInstanceKey":"root","rootProcessInstanceKey":"root","startDate":"2026-09-13T12:00:00Z","state":%q,"tenantId":"tenant"}],"page":{"totalItems":1}}`, state)
			} else {
				_, _ = io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
			}
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/deletion"):
			key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/process-instances/"), "/deletion")
			deleteAttempts[key]++
			if key == "child" && deleteAttempts[key] == 1 {
				w.WriteHeader(http.StatusConflict)
				_, _ = io.WriteString(w, `{"title":"Conflict","status":409,"detail":"active instance"}`)
				return
			}
			deleted[key] = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/root/cancellation":
			cancellationSubmitted = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request %s %s body=%s", r.Method, r.URL.Path, body)
		}
	})
}

func indexOfString(values []string, wanted string) int {
	for i, value := range values {
		if value == wanted {
			return i
		}
	}
	return -1
}
