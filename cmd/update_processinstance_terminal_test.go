// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const (
	updateProcessInstanceTerminalArgsEnv    = "C8VOLT_UPDATE_PROCESS_INSTANCE_TERMINAL_ARGS"
	updateProcessInstanceTerminalStderrEnv  = "C8VOLT_UPDATE_PROCESS_INSTANCE_TERMINAL_STDERR"
	updateProcessInstanceTerminalCaptureEnv = "C8VOLT_UPDATE_PROCESS_INSTANCE_TERMINAL_CAPTURE"
)

const updateProcessInstanceTerminalPrompt = "You are about to update 2 requested variable value(s) on 1 process instance(s).\nDo you want to proceed? [y/N]: "

// updateProcessInstanceOutputCapture records planning, mutation, and
// confirmation requests independently of terminal rendering.
type updateProcessInstanceOutputCapture struct {
	mu            sync.Mutex
	initialValue  bool
	mutatedScopes map[string]bool
	mutations     int
	instanceReads int
	variableReads int
	totalRequests int
}

// TestUpdateProcessInstanceTerminal proves actual-terminal confirmation and
// unattended paths preserve annotated previews and separate output streams.
func TestUpdateProcessInstanceTerminal(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		runUpdateProcessInstanceTerminalHelper(t)
		os.Exit(0)
	}

	tests := []struct {
		name          string
		args          []string
		initialValue  bool
		configured    bool
		prompt        bool
		preview       bool
		promptText    string
		wantPreview   []string
		absentPreview []string
		response      string
		endOfInput    bool
		wantExitCode  int
		wantMutates   int
		wantReads     int
	}{
		{name: "configured stderr accepts", args: []string{"update", "pi", "--key", "101", "--vars", `{"approved":true,"new":"value"}`}, configured: true, prompt: true, preview: true, response: "yes", wantMutates: 1, wantReads: 2},
		{name: "inherited stderr declines", args: []string{"update", "pi", "--key", "101", "--vars", `{"approved":true,"new":"value"}`}, prompt: true, preview: true, response: "no", wantExitCode: exitcode.Error, wantReads: 1},
		{name: "default answer declines", args: []string{"update", "pi", "--key", "101", "--vars", `{"approved":true,"new":"value"}`}, prompt: true, preview: true, wantExitCode: exitcode.Error, wantReads: 1},
		{name: "terminal eof aborts", args: []string{"update", "pi", "--key", "101", "--vars", `{"approved":true,"new":"value"}`}, prompt: true, preview: true, endOfInput: true, wantExitCode: exitcode.Error, wantReads: 1},
		{name: "dry run is prompt free", args: []string{"update", "pi", "--key", "101", "--vars", `{"approved":true,"new":"value"}`, "--dry-run"}, preview: true, wantReads: 1},
		{name: "configured stderr no-op is prompt free", args: []string{"update", "pi", "--key", "101", "--vars", `{"approved":true}`}, initialValue: true, configured: true, wantReads: 1},
		{name: "automation no-wait is prompt free", args: []string{"--automation", "update", "pi", "--key", "101", "--vars", `{"approved":true,"new":"value"}`, "--no-wait"}, wantMutates: 1, wantReads: 1},
		{name: "auto-confirm is prompt free", args: []string{"--auto-confirm", "update", "pi", "--key", "101", "--vars", `{"approved":true,"new":"value"}`}, wantMutates: 1, wantReads: 2},
		{
			name:         "compact bulk normal preview stays summarized",
			args:         []string{"update", "pi", "--key", "101", "--key", "202", "--vars", `{"approved":true,"new":"value"}`},
			prompt:       true,
			promptText:   "You are about to update 4 requested variable value(s) on 2 process instance(s).\nDo you want to proceed? [y/N]: ",
			response:     "no",
			wantExitCode: exitcode.Error,
			wantReads:    2,
			wantPreview: []string{
				"plan: update process-instance variables: 2 process instance(s), 2 change(s), 2 addition(s), 0 unchanged, 0 untouched",
			},
			absentPreview: []string{"101: ~ approved", "202: ~ approved"},
		},
		{
			name:         "verbose bulk normal preview labels both selected roots",
			args:         []string{"--verbose", "update", "pi", "--key", "101", "--key", "202", "--vars", `{"approved":true,"new":"value"}`},
			prompt:       true,
			promptText:   "You are about to update 4 requested variable value(s) on 2 process instance(s).\nDo you want to proceed? [y/N]: ",
			response:     "no",
			wantExitCode: exitcode.Error,
			wantReads:    2,
			wantPreview: []string{
				`101: ~ approved (pi:101): false -> true; + new (pi:101): "value"`,
				`202: ~ approved (pi:202): false -> true; + new (pi:202): "value"`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, capture := newUpdateProcessInstanceOutputServer(t, "8.10", test.initialValue)
			encodedArgs, err := json.Marshal(test.args)
			require.NoError(t, err)
			capturePath := filepath.Join(t.TempDir(), "configured-stderr.txt")
			stderrMode := "inherited"
			if test.configured {
				stderrMode = "configured"
			}
			exchanges := []testx.CmdTerminalExchange(nil)
			if test.prompt {
				promptText := test.promptText
				if promptText == "" {
					promptText = updateProcessInstanceTerminalPrompt
				}
				exchanges = []testx.CmdTerminalExchange{{
					Prompt:     promptText,
					Response:   test.response,
					EndOfInput: test.endOfInput,
				}}
			}
			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
				ScopeTestName: "TestUpdateProcessInstanceTerminal",
				Env: map[string]string{
					"C8VOLT_TEST_CONFIG":                    writeTestConfigForVersion(t, server.URL, "8.10"),
					updateProcessInstanceTerminalArgsEnv:    string(encodedArgs),
					updateProcessInstanceTerminalStderrEnv:  stderrMode,
					updateProcessInstanceTerminalCaptureEnv: capturePath,
				},
				Exchanges: exchanges,
				Timeout:   5 * time.Second,
			})

			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			if test.wantExitCode == 0 {
				require.NoError(t, result.Err, result.Stderr)
			} else {
				var exitErr *exec.ExitError
				require.ErrorAs(t, result.Err, &exitErr)
				require.Equal(t, test.wantExitCode, exitErr.ExitCode())
			}
			require.Empty(t, result.Stdout, "terminal control and human results must stay off redirected stdout")
			if test.prompt {
				promptText := test.promptText
				if promptText == "" {
					promptText = updateProcessInstanceTerminalPrompt
				}
				require.Equal(t, 1, strings.Count(result.Stderr, promptText))
			} else {
				require.NotContains(t, result.Stderr, "Do you want to proceed?")
			}
			if test.preview {
				require.Contains(t, result.Stderr, "~ approved (pi:101): false -> true")
				if !test.initialValue {
					require.Contains(t, result.Stderr, "+ new (pi:101): \"value\"")
				}
			}
			for _, preview := range test.wantPreview {
				require.Contains(t, result.Stderr, preview)
			}
			for _, preview := range test.absentPreview {
				require.NotContains(t, result.Stderr, preview)
			}
			if test.configured {
				configuredOutput, readErr := os.ReadFile(capturePath)
				require.NoError(t, readErr)
				require.Equal(t, result.Stderr, string(configuredOutput))
			}

			capture.mu.Lock()
			defer capture.mu.Unlock()
			selectedKeys := 0
			for _, arg := range test.args {
				if arg == "--key" {
					selectedKeys++
				}
			}
			require.Equal(t, test.wantMutates, capture.mutations)
			require.Equal(t, selectedKeys, capture.instanceReads)
			require.Equal(t, test.wantReads, capture.variableReads)
			require.Equal(t, selectedKeys+test.wantReads+test.wantMutates, capture.totalRequests)
		})
	}
}

// newUpdateProcessInstanceOutputServer serves one selected root and fails
// every unexpected endpoint so preview labels cannot hide added requests.
func newUpdateProcessInstanceOutputServer(t *testing.T, version string, initialValue bool) (*httptest.Server, *updateProcessInstanceOutputCapture) {
	t.Helper()
	capture := &updateProcessInstanceOutputCapture{
		initialValue:  initialValue,
		mutatedScopes: make(map[string]bool),
	}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if serveVersionCheckTopology(writer, request, version) {
			return
		}

		capture.mu.Lock()
		capture.totalRequests++
		capture.mu.Unlock()
		switch {
		case request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(request.URL.Path, "/v2/process-instances/")
			require.NotEmpty(t, key)
			capture.mu.Lock()
			capture.instanceReads++
			capture.mu.Unlock()
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"processInstanceKey":%q,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionVersion":1,"state":"ACTIVE"}`, key)
		case request.Method == http.MethodPost && request.URL.Path == "/v2/variables/search":
			var body struct {
				Filter struct {
					ProcessInstanceKey string `json:"processInstanceKey"`
				} `json:"filter"`
			}
			require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
			key := body.Filter.ProcessInstanceKey
			require.NotEmpty(t, key)
			capture.mu.Lock()
			capture.variableReads++
			mutated := capture.mutatedScopes[key]
			value := capture.initialValue || mutated
			capture.mu.Unlock()
			writer.Header().Set("Content-Type", "application/json")
			if mutated {
				_, _ = fmt.Fprintf(writer, `{"items":[{"name":"approved","value":%q,"variableKey":%q,"processInstanceKey":%q,"scopeKey":%q,"tenantId":"tenant-a"},{"name":"new","value":"\"value\"","variableKey":%q,"processInstanceKey":%q,"scopeKey":%q,"tenantId":"tenant-a"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`, fmt.Sprint(value), "901"+key, key, key, "902"+key, key, key)
				return
			}
			_, _ = fmt.Fprintf(writer, `{"items":[{"name":"approved","value":%q,"variableKey":%q,"processInstanceKey":%q,"scopeKey":%q,"tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, fmt.Sprint(value), "901"+key, key, key)
		case request.Method == http.MethodPut && strings.HasPrefix(request.URL.Path, "/v2/element-instances/") && strings.HasSuffix(request.URL.Path, "/variables"):
			key := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, "/v2/element-instances/"), "/variables")
			capture.mu.Lock()
			capture.mutations++
			capture.mutatedScopes[key] = true
			capture.mu.Unlock()
			writer.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server, capture
}

// runUpdateProcessInstanceTerminalHelper executes the command with terminal
// stdin while selecting either configured root stderr or inherited stderr.
func runUpdateProcessInstanceTerminalHelper(t *testing.T) {
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv(updateProcessInstanceTerminalArgsEnv)), &args))
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetOut(os.Stdout)
	updateProcessInstanceCmd.SetErr(nil)
	if os.Getenv(updateProcessInstanceTerminalStderrEnv) == "configured" {
		capture, err := os.Create(os.Getenv(updateProcessInstanceTerminalCaptureEnv))
		require.NoError(t, err)
		defer capture.Close()
		root.SetErr(io.MultiWriter(os.Stderr, capture))
	} else {
		root.SetErr(nil)
	}
	root.SetArgs(append([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG")}, args...))
	_ = root.Execute()
}
