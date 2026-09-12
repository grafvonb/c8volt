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
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const apiDiagnosticsProcessDefinitionKey = "2251799813685255"
const apiDiagnosticsInvocationHelper = "TestAPIDiagnosticsInvocationHelper"

// TestAPIDiagnosticsCommandReadPreservesResultsAndRequests verifies inherited verbose placement enables one diagnostic without changing command results or traffic.
func TestAPIDiagnosticsCommandReadPreservesResultsAndRequests(t *testing.T) {
	var requests atomic.Int32
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		require.Equal(t, http.MethodGet, request.Method)
		require.Equal(t, "/v2/process-definitions/"+apiDiagnosticsProcessDefinitionKey, request.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		time.Sleep(5 * time.Millisecond)
		_, _ = w.Write([]byte(apiDiagnosticsProcessDefinitionFixture))
	}))
	t.Cleanup(server.Close)
	configPath := writeAPIDiagnosticsCommandConfig(t, server.URL, "none")

	baselineStdout, baselineStderr := executeAPIDiagnosticsRoot(t,
		"--config", configPath, "--log-format", "plain",
		"get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey,
	)
	require.NotContains(t, baselineStderr, "api #")

	for _, args := range [][]string{
		{"--verbose", "--config", configPath, "--log-format", "plain", "get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey},
		{"--config", configPath, "--log-format", "plain", "get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey, "--verbose"},
	} {
		stdout, stderr := executeAPIDiagnosticsRoot(t, args...)
		require.Equal(t, baselineStdout, stdout)
		require.Equal(t, 1, strings.Count(stderr, "api #"))
		require.Contains(t, stderr, "api #1 GET /v2/process-definitions/"+apiDiagnosticsProcessDefinitionKey+": status=200")
		require.Contains(t, stderr, " headers=")
		require.Contains(t, stderr, " body=")
	}

	require.Equal(t, int32(3), requests.Load(), "diagnostics must not add requests")
}

// TestAPIDiagnosticsCommandAuthenticationBootstrap verifies token and cookie initialization traffic shares command diagnostics while auth-none adds no bootstrap exchange.
func TestAPIDiagnosticsCommandAuthenticationBootstrap(t *testing.T) {
	for _, test := range []struct {
		name            string
		authMode        string
		wantRequests    int32
		wantFirstTarget string
	}{
		{name: "none", authMode: "none", wantRequests: 1, wantFirstTarget: "GET /v2/process-definitions/"},
		{name: "oauth2", authMode: "oauth2", wantRequests: 2, wantFirstTarget: "POST /oauth/token"},
		{name: "cookie", authMode: "cookie", wantRequests: 2, wantFirstTarget: "POST /api/login"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				requests.Add(1)
				switch request.URL.Path {
				case "/oauth/token":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"access_token":"command-token","expires_in":120,"token_type":"Bearer"}`))
				case "/api/login":
					http.SetCookie(w, &http.Cookie{Name: "SESSION", Value: "command-cookie", Path: "/"})
					_, _ = w.Write([]byte(`{}`))
				case "/v2/process-definitions/" + apiDiagnosticsProcessDefinitionKey:
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(apiDiagnosticsProcessDefinitionFixture))
				default:
					http.NotFound(w, request)
				}
			}))
			t.Cleanup(server.Close)
			configPath := writeAPIDiagnosticsCommandConfig(t, server.URL, test.authMode)

			_, stderr := executeAPIDiagnosticsRoot(t,
				"--config", configPath, "--log-format", "plain", "--verbose",
				"get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey,
			)

			require.Equal(t, test.wantRequests, requests.Load())
			require.Equal(t, int(test.wantRequests), strings.Count(stderr, "api #"))
			require.Contains(t, stderr, "api #1 "+test.wantFirstTarget)
			require.NotContains(t, stderr, "command-client-secret")
			require.NotContains(t, stderr, "command-token")
			require.NotContains(t, stderr, "command-cookie")
		})
	}
}

// TestAPIDiagnosticsCommandFiltering verifies verbose-off, debug-only, quiet and restrictive INFO settings install no admitted command diagnostic output.
func TestAPIDiagnosticsCommandFiltering(t *testing.T) {
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/v2/process-definitions/"+apiDiagnosticsProcessDefinitionKey, request.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(apiDiagnosticsProcessDefinitionFixture))
	}))
	t.Cleanup(server.Close)
	configPath := writeAPIDiagnosticsCommandConfig(t, server.URL, "none")

	for _, test := range []struct {
		name string
		args []string
	}{
		{name: "verbose off", args: nil},
		{name: "debug only", args: []string{"--debug"}},
		{name: "quiet verbose", args: []string{"--quiet", "--verbose"}},
		{name: "info filtered", args: []string{"--verbose", "--log-level", "warn"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := []string{"--config", configPath, "--log-format", "plain"}
			args = append(args, test.args...)
			args = append(args, "get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey)
			_, stderr := executeAPIDiagnosticsRoot(t, args...)
			require.NotContains(t, stderr, "api #")
		})
	}
}

// TestAPIDiagnosticsCommandHelpPerformsNoExchange verifies help accepts verbose without bootstrapping remote services.
func TestAPIDiagnosticsCommandHelpPerformsNoExchange(t *testing.T) {
	stdout, stderr := executeAPIDiagnosticsRoot(t, "get", "process-definition", "--verbose", "--help")

	require.Contains(t, stdout, "--verbose")
	require.NotContains(t, stderr, "api #")
}

// TestAPIDiagnosticsCommandEffectiveStderrRouting verifies diagnostics follow
// the executing child's configured writer, fall back to inherited root stderr,
// and never reuse a prior invocation's destination.
func TestAPIDiagnosticsCommandEffectiveStderrRouting(t *testing.T) {
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/v2/process-definitions/"+apiDiagnosticsProcessDefinitionKey, request.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(apiDiagnosticsProcessDefinitionFixture))
	}))
	t.Cleanup(server.Close)
	configPath := writeAPIDiagnosticsCommandConfig(t, server.URL, "none")
	args := []string{
		"--config", configPath, "--log-format", "plain", "--verbose",
		"get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey,
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(func() {
		getProcessDefinitionCmd.SetErr(nil)
		resetCommandTreeFlags(root)
		resetGetProcessDefinitionCommandGlobals()
	})

	firstRootStderr := &bytes.Buffer{}
	firstChildStderr := &bytes.Buffer{}
	root.SetOut(&bytes.Buffer{})
	root.SetErr(firstRootStderr)
	getProcessDefinitionCmd.SetErr(firstChildStderr)
	root.SetArgs(args)
	_, err := root.ExecuteC()
	require.NoError(t, err)
	require.Empty(t, firstRootStderr.String())
	require.Contains(t, firstChildStderr.String(), "api #1 GET /v2/process-definitions/"+apiDiagnosticsProcessDefinitionKey)

	getProcessDefinitionCmd.SetErr(nil)
	resetCommandTreeFlags(root)
	resetGetProcessDefinitionCommandGlobals()
	secondRootStderr := &bytes.Buffer{}
	root.SetOut(&bytes.Buffer{})
	root.SetErr(secondRootStderr)
	root.SetArgs(args)
	_, err = root.ExecuteC()
	require.NoError(t, err)
	require.Contains(t, secondRootStderr.String(), "api #1 GET /v2/process-definitions/"+apiDiagnosticsProcessDefinitionKey)
	require.Equal(t, 1, strings.Count(firstChildStderr.String(), "api #"), "a later invocation must not reuse child stderr")
}

// TestAPIDiagnosticsCommandSubprocessInvocationsStayIsolated verifies real
// process invocations each own their stderr destination and sequence space.
func TestAPIDiagnosticsCommandSubprocessInvocationsStayIsolated(t *testing.T) {
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/v2/process-definitions/"+apiDiagnosticsProcessDefinitionKey, request.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(apiDiagnosticsProcessDefinitionFixture))
	}))
	t.Cleanup(server.Close)
	configPath := writeAPIDiagnosticsCommandConfig(t, server.URL, "none")

	var previousStdout string
	for invocation := range 2 {
		stdout, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, apiDiagnosticsInvocationHelper, "", map[string]string{
			"C8VOLT_API_DIAGNOSTICS_CONFIG": configPath,
		}, "")
		require.NoError(t, err, "invocation %d: stdout=%q stderr=%q", invocation+1, stdout, stderr)
		require.Equal(t, 1, strings.Count(stderr, "api #"))
		require.Contains(t, stderr, "api #1 GET /v2/process-definitions/"+apiDiagnosticsProcessDefinitionKey)
		require.NotContains(t, stderr, "api #2")
		if invocation > 0 {
			require.Equal(t, previousStdout, stdout)
		}
		previousStdout = stdout
	}
}

// TestAPIDiagnosticsInvocationHelper executes one fresh process-level command invocation.
func TestAPIDiagnosticsInvocationHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, apiDiagnosticsInvocationHelper, os.Getenv(testx.CmdSubprocessNameEnv))
	os.Args = []string{
		"c8volt",
		"--config", os.Getenv("C8VOLT_API_DIAGNOSTICS_CONFIG"),
		"--log-format", "plain", "--verbose",
		"get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey,
	}
	Execute()
	os.Exit(0)
}

// TestAPIDiagnosticsCommandReadOutputAndLogFormatMatrix verifies read results
// remain byte-identical in every machine mode while each supported logger
// framing retains the complete safe diagnostic message.
func TestAPIDiagnosticsCommandReadOutputAndLogFormatMatrix(t *testing.T) {
	var requests atomic.Int32
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		require.Equal(t, http.MethodGet, request.Method)
		require.Equal(t, "/v2/process-definitions/"+apiDiagnosticsProcessDefinitionKey, request.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(apiDiagnosticsProcessDefinitionFixture))
	}))
	t.Cleanup(server.Close)
	configPath := writeAPIDiagnosticsCommandConfig(t, server.URL, "none")

	for _, test := range []struct {
		name           string
		mode           []string
		wantDiagnostic bool
		jsonResult     bool
		keysResult     bool
	}{
		{name: "normal", wantDiagnostic: true},
		{name: "json", mode: []string{"--json"}, wantDiagnostic: true, jsonResult: true},
		{name: "keys only", mode: []string{"--keys-only"}, wantDiagnostic: true, keysResult: true},
		{name: "quiet json", mode: []string{"--quiet", "--json"}, jsonResult: true},
		{name: "quiet keys only", mode: []string{"--quiet", "--keys-only"}, keysResult: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := append([]string{"--config", configPath, "--log-format", "plain-time"}, test.mode...)
			command := []string{"get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey}
			baselineStdout, baselineStderr := executeAPIDiagnosticsRoot(t, append(base, command...)...)
			enabledArgs := append(append([]string{}, base...), "--verbose")
			enabledStdout, enabledStderr := executeAPIDiagnosticsRoot(t, append(enabledArgs, command...)...)

			require.Equal(t, baselineStdout, enabledStdout)
			require.NotContains(t, baselineStderr, "api #")
			if test.wantDiagnostic {
				require.Equal(t, 1, strings.Count(enabledStderr, "api #"))
			} else {
				require.NotContains(t, enabledStderr, "api #")
			}
			if test.jsonResult {
				requireSingleJSONValue(t, enabledStdout)
			}
			if test.keysResult {
				require.Equal(t, apiDiagnosticsProcessDefinitionKey+"\n", enabledStdout)
			}
		})
	}
	require.Equal(t, int32(10), requests.Load(), "diagnostics and result modes must not add requests")

	for _, format := range []string{"plain", "plain-time", "text", "json"} {
		t.Run("log format "+format, func(t *testing.T) {
			_, stderr := executeAPIDiagnosticsRoot(t,
				"--config", configPath, "--log-format", format, "--log-with-source", "--verbose",
				"get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey,
			)
			line := apiDiagnosticLine(t, stderr)
			require.Contains(t, line, "api #1 GET /v2/process-definitions/"+apiDiagnosticsProcessDefinitionKey+": status=200")
			require.Contains(t, line, " total=")
			require.Contains(t, line, " headers=")
			require.Contains(t, line, " body=")
			if format == "json" {
				var record map[string]any
				require.NoError(t, json.Unmarshal([]byte(line), &record))
				require.Equal(t, "INFO", record["level"])
				require.Contains(t, record["msg"], "api #1 GET")
				require.NotNil(t, record["source"], "source-enabled JSON framing must retain source metadata")
			} else {
				require.Contains(t, line, ".go:", "source-enabled framing must retain source metadata")
			}
		})
	}
}

// TestAPIDiagnosticsCommandEmptyCancellationMatrix verifies selector no-ops
// preserve request bodies, prompts and result contracts across mutation modes.
func TestAPIDiagnosticsCommandEmptyCancellationMatrix(t *testing.T) {
	for _, test := range []struct {
		name           string
		rootFlags      []string
		commandFlags   []string
		wantDiagnostic bool
		jsonResult     bool
		keysResult     bool
	}{
		{name: "human auto confirm", rootFlags: []string{"--auto-confirm"}, wantDiagnostic: true},
		{name: "json automation", rootFlags: []string{"--json", "--automation"}, wantDiagnostic: true, jsonResult: true},
		{name: "keys no wait", rootFlags: []string{"--keys-only"}, commandFlags: []string{"--no-wait"}, wantDiagnostic: true, keysResult: true},
		{name: "dry run json", rootFlags: []string{"--json"}, commandFlags: []string{"--dry-run"}, wantDiagnostic: true, jsonResult: true},
		{name: "quiet json", rootFlags: []string{"--quiet", "--json"}, jsonResult: true},
		{name: "quiet keys", rootFlags: []string{"--quiet", "--keys-only"}, keysResult: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requestBodies []string
			server := newProcessInstanceSearchCaptureServerWithResponses(t, &requestBodies, `{"items":[]}`, `{"items":[]}`)
			t.Cleanup(server.Close)
			configPath := writeTestConfigForVersion(t, server.URL, "8.8")
			previousConfirm := confirmCmdOrAbortFn
			confirmCmdOrAbortFn = func(_ io.Writer, _ bool, _ string) error {
				t.Fatal("empty cancellation scope must not prompt")
				return nil
			}
			t.Cleanup(func() { confirmCmdOrAbortFn = previousConfirm })

			prefix := append([]string{"--config", configPath, "--log-format", "plain-time"}, test.rootFlags...)
			command := append([]string{"cancel", "process-instance", "--state", "active"}, test.commandFlags...)
			baselineStdout, baselineStderr := executeRootForProcessInstanceWithSeparateOutputs(t, append(prefix, command...)...)
			enabledArgs := append(append([]string{}, prefix...), "--verbose")
			enabledStdout, enabledStderr := executeRootForProcessInstanceWithSeparateOutputs(t, append(enabledArgs, command...)...)

			require.Equal(t, baselineStdout, enabledStdout)
			require.NotContains(t, baselineStderr, "api #")
			if test.wantDiagnostic {
				require.Equal(t, 1, strings.Count(enabledStderr, "api #"))
			} else {
				require.NotContains(t, enabledStderr, "api #")
			}
			require.Len(t, requestBodies, 2)
			require.JSONEq(t, requestBodies[0], requestBodies[1])
			if test.jsonResult {
				requireSingleJSONValue(t, enabledStdout)
			}
			if test.keysResult {
				require.Empty(t, enabledStdout, "empty keys-only scope must emit zero bytes")
			}
		})
	}
}

// TestAPIDiagnosticsCommandExplicitCancellationPreservesSubmission verifies an
// explicit-key no-wait mutation keeps its JSON outcome and exact HTTP traffic.
func TestAPIDiagnosticsCommandExplicitCancellationPreservesSubmission(t *testing.T) {
	baselineStdout, baselineStderr, baselineRequests := runAPIDiagnosticsNoWaitCancellation(t, false)
	enabledStdout, enabledStderr, enabledRequests := runAPIDiagnosticsNoWaitCancellation(t, true)

	require.Equal(t, baselineStdout, enabledStdout)
	require.NotContains(t, baselineStderr, "api #")
	require.Equal(t, baselineRequests, enabledRequests)
	require.Equal(t, 1, strings.Count(strings.Join(enabledRequests, "\n"), "POST /v2/process-instances/301/cancellation"))
	require.Equal(t, len(enabledRequests), strings.Count(enabledStderr, "api #"))
	requireSingleJSONValue(t, enabledStdout)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(enabledStdout), &envelope))
	require.Equal(t, "accepted", envelope["outcome"])
	require.Equal(t, "cancel process-instance", envelope["command"])
}

// runAPIDiagnosticsNoWaitCancellation executes an isolated explicit-key
// cancellation fixture and captures each request method, path and body.
func runAPIDiagnosticsNoWaitCancellation(t *testing.T, verbose bool) (string, string, []string) {
	t.Helper()
	var requests testx.SafeSlice[string]
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		requests.Append(request.Method + " " + request.URL.Path + " " + string(body))
		w.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/v2/process-instances/search":
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"301","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`))
		case request.Method == http.MethodGet && request.URL.Path == "/v2/process-instances/301":
			_, _ = w.Write([]byte(`{"processInstanceKey":"301","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case request.Method == http.MethodPost && request.URL.Path == "/v2/process-instances/301/cancellation":
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	configPath := writeTestConfigForVersion(t, server.URL, "8.8")
	args := []string{
		"--config", configPath, "--log-format", "plain-time", "--automation", "--auto-confirm", "--json",
	}
	if verbose {
		args = append(args, "--verbose")
	}
	args = append(args, "cancel", "process-instance", "--key", "301", "--no-wait")
	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, args...)
	return stdout, stderr, requests.Snapshot()
}

// requireSingleJSONValue proves command stdout contains exactly one JSON value.
func requireSingleJSONValue(t *testing.T, output string) {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(output))
	var value any
	require.NoError(t, decoder.Decode(&value))
	require.ErrorIs(t, decoder.Decode(&value), io.EOF)
}

// apiDiagnosticLine returns the one physical log line containing an API record.
func apiDiagnosticLine(t *testing.T, output string) string {
	t.Helper()
	var found string
	for line := range strings.SplitSeq(strings.TrimSuffix(output, "\n"), "\n") {
		if strings.Contains(line, "api #") {
			require.Empty(t, found, "expected exactly one API diagnostic line")
			found = line
		}
	}
	require.NotEmpty(t, found)
	return found
}

// executeAPIDiagnosticsRoot resets shared Cobra state and captures result and diagnostic streams separately.
func executeAPIDiagnosticsRoot(t *testing.T, args ...string) (string, string) {
	t.Helper()

	root := Root()
	resetCommandTreeFlags(root)
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
		resetGetProcessDefinitionCommandGlobals()
	})
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)

	_, err := root.ExecuteC()
	require.NoError(t, err)
	return stdout.String(), stderr.String()
}

// withoutAPIDiagnosticLines lets existing progress assertions distinguish the new transport records from command-owned lifecycle text.
func withoutAPIDiagnosticLines(output string) string {
	var retained strings.Builder
	for line := range strings.SplitSeq(output, "\n") {
		if strings.Contains(line, " api #") || strings.HasPrefix(line, "api #") {
			continue
		}
		if line != "" {
			retained.WriteString(line)
			retained.WriteByte('\n')
		}
	}
	return retained.String()
}

// writeAPIDiagnosticsCommandConfig creates auth variants that all use one deterministic local API fixture.
func writeAPIDiagnosticsCommandConfig(t *testing.T, baseURL, authMode string) string {
	t.Helper()

	auth := "  mode: none\n"
	switch authMode {
	case "oauth2":
		auth = fmt.Sprintf("  mode: oauth2\n  oauth2:\n    token_url: %q\n    client_id: command-client\n    client_secret: command-client-secret\n", baseURL+"/oauth")
	case "cookie":
		auth = fmt.Sprintf("  mode: cookie\n  cookie:\n    base_url: %q\n    username: command-user\n    password: command-password\n", baseURL)
	}
	content := fmt.Sprintf("app:\n  camunda_version: %q\nauth:\n%sapis:\n  camunda_api:\n    base_url: %q\n", "8.8", auth, baseURL)
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o600))
	return configPath
}

const apiDiagnosticsProcessDefinitionFixture = `{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"}`
