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
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

const (
	confirmTerminalScenarioEnv         = "C8VOLT_CONFIRM_TERMINAL_SCENARIO"
	confirmTerminalPrompt              = "Proceed with this deletion? [y/N]: "
	confirmDefaultYesPrompt            = "List visible process definitions? [Y/n]: "
	emptySelectorConfigEnv             = "C8VOLT_EMPTY_SELECTOR_CONFIG"
	emptySelectorOperationEnv          = "C8VOLT_EMPTY_SELECTOR_OPERATION"
	emptySelectorDryRunEnv             = "C8VOLT_EMPTY_SELECTOR_DRY_RUN"
	emptySelectorModeEnv               = "C8VOLT_EMPTY_SELECTOR_MODE"
	processInstanceConfirmConfigEnv    = "C8VOLT_PI_CONFIRM_CONFIG"
	processInstanceConfirmOperationEnv = "C8VOLT_PI_CONFIRM_OPERATION"
	processInstanceConfirmFormatEnv    = "C8VOLT_PI_CONFIRM_FORMAT"
	processInstanceConfirmStderrEnv    = "C8VOLT_PI_CONFIRM_STDERR"
	processInstanceConfirmCaptureEnv   = "C8VOLT_PI_CONFIRM_CAPTURE"
)

// TestProcessInstanceConfirmationTerminal verifies tenant logs and destructive
// prompts share stderr without contaminating command results on real terminals.
func TestProcessInstanceConfirmationTerminal(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		runProcessInstanceConfirmationTerminalHelper(t)
		os.Exit(0)
	}

	tests := []struct {
		name       string
		operation  string
		format     string
		configured bool
		accept     bool
	}{
		{name: "cancel configured plain accepts", operation: "cancel", format: "plain", configured: true, accept: true},
		{name: "cancel inherited json accepts", operation: "cancel", format: "json", accept: true},
		{name: "cancel configured json aborts", operation: "cancel", format: "json", configured: true},
		{name: "cancel inherited plain aborts", operation: "cancel", format: "plain"},
		{name: "delete configured plain accepts", operation: "delete", format: "plain", configured: true, accept: true},
		{name: "delete inherited json accepts", operation: "delete", format: "json", accept: true},
		{name: "delete configured json aborts", operation: "delete", format: "json", configured: true},
		{name: "delete inherited plain aborts", operation: "delete", format: "plain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturePath := filepath.Join(t.TempDir(), "configured-stderr.txt")
			var requests testx.SafeSlice[string]
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Append(r.Method + " " + r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					if strings.Contains(string(body), "parentProcessInstanceKey") {
						_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
						return
					}
					state := "ACTIVE"
					endDate := ""
					if tt.operation == "delete" {
						state = "COMPLETED"
						endDate = `,"endDate":"2026-09-10T12:00:00Z"`
					}
					_, _ = fmt.Fprintf(w, `{"items":[{"processInstanceKey":"301","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-09-10T11:00:00Z"%s,"state":"%s","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, endDate, state)
				case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/301":
					state := "ACTIVE"
					endDate := ""
					if tt.operation == "delete" {
						state = "COMPLETED"
						endDate = `,"endDate":"2026-09-10T12:00:00Z"`
					}
					_, _ = fmt.Fprintf(w, `{"processInstanceKey":"301","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-09-10T11:00:00Z"%s,"state":"%s","tenantId":"tenant-a"}`, endDate, state)
				case tt.operation == "cancel" && r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/301/cancellation":
					w.WriteHeader(http.StatusAccepted)
				case tt.operation == "delete" && r.Method == http.MethodDelete && r.URL.Path == "/v1/process-instances/301":
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
			stderrMode := "inherited"
			if tt.configured {
				stderrMode = "configured"
			}
			prompt := fmt.Sprintf("You are about to %s 1 process instance(s).\nDo you want to proceed? [y/N]: ", tt.operation)
			response := "n"
			if tt.accept {
				response = "y"
			}
			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
				ScopeTestName: "TestProcessInstanceConfirmationTerminal",
				Env: map[string]string{
					processInstanceConfirmConfigEnv:    cfgPath,
					processInstanceConfirmOperationEnv: tt.operation,
					processInstanceConfirmFormatEnv:    tt.format,
					processInstanceConfirmStderrEnv:    stderrMode,
					processInstanceConfirmCaptureEnv:   capturePath,
				},
				Exchanges: []testx.CmdTerminalExchange{{Prompt: prompt, Response: response}},
				Timeout:   4 * time.Second,
			})

			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			require.True(t, result.Supported, result.UnsupportedReason)
			if tt.accept {
				require.NoError(t, result.Err, result.Stderr)
			} else {
				var exitErr *exec.ExitError
				require.ErrorAs(t, result.Err, &exitErr)
				require.Equal(t, exitcode.Error, exitErr.ExitCode())
			}
			require.Equal(t, 1, strings.Count(result.Stderr, prompt))
			require.NotContains(t, result.Stdout, prompt)
			require.NotContains(t, result.Stdout, "selection scope:")
			requireProcessInstanceTerminalTenantLog(t, result.Stderr, tt.format)

			mutationPath := "/v2/process-instances/301/cancellation"
			if tt.operation == "delete" {
				mutationPath = "/v1/process-instances/301"
			}
			mutationCalls := 0
			for _, request := range requests.Snapshot() {
				if strings.HasSuffix(request, mutationPath) {
					mutationCalls++
				}
			}
			if tt.accept {
				require.Equal(t, 1, mutationCalls)
			} else {
				require.Zero(t, mutationCalls)
				require.Empty(t, result.Stdout)
			}

			if tt.configured {
				captured, err := os.ReadFile(capturePath)
				require.NoError(t, err)
				require.Equal(t, result.Stderr, string(captured))
			}
		})
	}
}

// requireProcessInstanceTerminalTenantLog validates the standard tenant record
// while allowing unrelated command diagnostics before and after the prompt.
func requireProcessInstanceTerminalTenantLog(t *testing.T, stderr string, format string) {
	t.Helper()
	const message = "selection scope: unfiltered across accessible tenants"
	if format == "plain" {
		require.Contains(t, stderr, " INFO "+message+"\n")
		return
	}
	for _, line := range strings.Split(stderr, "\n") {
		var record struct {
			Level string `json:"level"`
			Msg   string `json:"msg"`
		}
		if json.Unmarshal([]byte(line), &record) == nil && record.Msg == message {
			require.Equal(t, "INFO", record.Level)
			return
		}
	}
	t.Fatalf("missing JSON tenant log record in stderr: %q", stderr)
}

// TestConfirmOrAbortTerminal verifies default-no decisions and exact prompt routing with real terminal stdin.
func TestConfirmOrAbortTerminal(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		runConfirmOrAbortTerminalHelper(t)
		os.Exit(0)
	}

	tests := []struct {
		name       string
		scenario   string
		response   string
		endOfInput bool
		wantStdout string
	}{
		{
			name:       "yes with nil writer fallback",
			scenario:   "accept-nil-writer",
			response:   "yes",
			wantStdout: "terminal=true\ndecision=accepted\n",
		},
		{
			name:       "short yes with custom writer",
			scenario:   "accept-custom-writer",
			response:   "y",
			wantStdout: "terminal=true\ndecision=accepted\n",
		},
		{
			name:       "mixed case",
			scenario:   "accept",
			response:   "YeS",
			wantStdout: "terminal=true\ndecision=accepted\n",
		},
		{
			name:       "surrounding whitespace",
			scenario:   "accept",
			response:   "  yes  ",
			wantStdout: "terminal=true\ndecision=accepted\n",
		},
		{
			name:       "other answer aborts",
			scenario:   "decline",
			response:   "later",
			wantStdout: "terminal=true\ndecision=aborted\n",
		},
		{
			name:       "empty answer",
			scenario:   "empty",
			wantStdout: "terminal=true\ndecision=aborted\n",
		},
		{
			name:       "canonical EOF",
			scenario:   "eof",
			endOfInput: true,
			wantStdout: "terminal=true\ndecision=aborted\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
				ScopeTestName: "TestConfirmOrAbortTerminal",
				Env: map[string]string{
					confirmTerminalScenarioEnv: tt.scenario,
				},
				Exchanges: []testx.CmdTerminalExchange{{
					Prompt:     confirmTerminalPrompt,
					Response:   tt.response,
					EndOfInput: tt.endOfInput,
				}},
				Timeout: 2 * time.Second,
			})

			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			require.True(t, result.Supported, result.UnsupportedReason)
			require.NoError(t, result.Err)
			require.Equal(t, tt.wantStdout, result.Stdout)
			require.Equal(t, confirmTerminalPrompt, result.Stderr)
			require.NotContains(t, result.Stdout, confirmTerminalPrompt)
		})
	}
}

// TestConfirmOrAbortDefaultYesTerminal verifies default-yes decisions and exact prompt routing with real terminal stdin.
func TestConfirmOrAbortDefaultYesTerminal(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		runConfirmOrAbortDefaultYesTerminalHelper(t)
		os.Exit(0)
	}

	tests := []struct {
		name       string
		scenario   string
		response   string
		endOfInput bool
		wantStdout string
	}{
		{
			name:       "yes with nil writer fallback",
			scenario:   "accept-nil-writer",
			response:   "yes",
			wantStdout: "terminal=true\ndecision=accepted\n",
		},
		{
			name:       "short yes with custom writer",
			scenario:   "accept-custom-writer",
			response:   "y",
			wantStdout: "terminal=true\ndecision=accepted\n",
		},
		{
			name:       "mixed case",
			scenario:   "accept",
			response:   "YeS",
			wantStdout: "terminal=true\ndecision=accepted\n",
		},
		{
			name:       "surrounding whitespace",
			scenario:   "accept",
			response:   "  yes  ",
			wantStdout: "terminal=true\ndecision=accepted\n",
		},
		{
			name:       "empty answer accepts default",
			scenario:   "accept",
			wantStdout: "terminal=true\ndecision=accepted\n",
		},
		{
			name:       "other answer aborts",
			scenario:   "decline",
			response:   "later",
			wantStdout: "terminal=true\ndecision=aborted\n",
		},
		{
			name:       "canonical EOF aborts",
			scenario:   "eof",
			endOfInput: true,
			wantStdout: "terminal=true\ndecision=aborted\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
				ScopeTestName: "TestConfirmOrAbortDefaultYesTerminal",
				Env: map[string]string{
					confirmTerminalScenarioEnv: tt.scenario,
				},
				Exchanges: []testx.CmdTerminalExchange{{
					Prompt:     confirmDefaultYesPrompt,
					Response:   tt.response,
					EndOfInput: tt.endOfInput,
				}},
				Timeout: 2 * time.Second,
			})

			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			require.True(t, result.Supported, result.UnsupportedReason)
			require.NoError(t, result.Err)
			require.Equal(t, tt.wantStdout, result.Stdout)
			require.Equal(t, confirmDefaultYesPrompt, result.Stderr)
			require.NotContains(t, result.Stdout, confirmDefaultYesPrompt)
		})
	}
}

// TestConfirmationEmptySelectorResults verifies empty selector commands finish
// without reading real terminal stdin while stdout and stderr stay independent.
func TestConfirmationEmptySelectorResults(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		runConfirmationEmptySelectorResultsHelper(t)
		os.Exit(0)
	}

	tests := []struct {
		name      string
		operation string
		dryRun    bool
		mode      RenderMode
		quiet     bool
	}{
		{name: "delete human", operation: "delete", mode: RenderModeOneLine},
		{name: "delete human dry run", operation: "delete", dryRun: true, mode: RenderModeOneLine},
		{name: "delete json", operation: "delete", mode: RenderModeJSON},
		{name: "delete json dry run", operation: "delete", dryRun: true, mode: RenderModeJSON},
		{name: "delete keys only", operation: "delete", mode: RenderModeKeysOnly},
		{name: "delete keys only dry run", operation: "delete", dryRun: true, mode: RenderModeKeysOnly},
		{name: "delete quiet", operation: "delete", mode: RenderModeOneLine, quiet: true},
		{name: "delete quiet dry run", operation: "delete", dryRun: true, mode: RenderModeOneLine, quiet: true},
		{name: "cancel human", operation: "cancel", mode: RenderModeOneLine},
		{name: "cancel human dry run", operation: "cancel", dryRun: true, mode: RenderModeOneLine},
		{name: "cancel json", operation: "cancel", mode: RenderModeJSON},
		{name: "cancel json dry run", operation: "cancel", dryRun: true, mode: RenderModeJSON},
		{name: "cancel keys only", operation: "cancel", mode: RenderModeKeysOnly},
		{name: "cancel keys only dry run", operation: "cancel", dryRun: true, mode: RenderModeKeysOnly},
		{name: "cancel quiet", operation: "cancel", mode: RenderModeOneLine, quiet: true},
		{name: "cancel quiet dry run", operation: "cancel", dryRun: true, mode: RenderModeOneLine, quiet: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Append(r.Method + " " + r.URL.Path)
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/v2/process-instances/search", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
			}))
			t.Cleanup(srv.Close)
			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
				ScopeTestName: "TestConfirmationEmptySelectorResults",
				Env: map[string]string{
					emptySelectorConfigEnv:    cfgPath,
					emptySelectorOperationEnv: tt.operation,
					emptySelectorDryRunEnv:    fmt.Sprintf("%t", tt.dryRun),
					emptySelectorModeEnv:      emptySelectorTerminalMode(tt.mode, tt.quiet),
				},
				Timeout: 2 * time.Second,
			})

			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			require.NoError(t, result.Err)
			require.Equal(t, []string{"POST /v2/process-instances/search"}, requests.Snapshot())
			if tt.quiet {
				require.Empty(t, result.Stdout)
				require.Empty(t, result.Stderr)
				return
			}
			if tt.mode == RenderModeOneLine {
				require.Equal(t, "found: 0\n", result.Stdout)
				require.Empty(t, result.Stderr)
				return
			}
			requireEmptyProcessInstanceSelectorOutput(t, result.Stdout, result.Stderr, tt.operation+" process-instance", tt.operation, tt.dryRun, tt.mode)
		})
	}
}

func emptySelectorTerminalMode(mode RenderMode, quiet bool) string {
	if quiet {
		return "quiet"
	}
	if mode == RenderModeJSON {
		return "json"
	}
	if mode == RenderModeKeysOnly {
		return "keys-only"
	}
	return "human"
}

func runConfirmationEmptySelectorResultsHelper(t *testing.T) {
	require.Equal(t, "TestConfirmationEmptySelectorResults", os.Getenv(testx.CmdSubprocessNameEnv))
	require.True(t, term.IsTerminal(int(os.Stdin.Fd())))

	operation := os.Getenv(emptySelectorOperationEnv)
	state := "active"
	if operation == "delete" {
		state = "completed"
	}
	args := []string{"--config", os.Getenv(emptySelectorConfigEnv), operation, "process-instance", "--state", state}
	if os.Getenv(emptySelectorDryRunEnv) == "true" {
		args = append(args, "--dry-run")
	}
	switch os.Getenv(emptySelectorModeEnv) {
	case "json":
		args = append(args, "--json")
	case "keys-only":
		args = append(args, "--keys-only")
	case "quiet":
		args = append(args, "--quiet")
	}

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, args...)
	_, _ = fmt.Fprint(os.Stdout, stdout)
	_, _ = fmt.Fprint(os.Stderr, stderr)
}

// runProcessInstanceConfirmationTerminalHelper executes the real selector and
// mutation path with terminal stdin and either configured or inherited stderr.
func runProcessInstanceConfirmationTerminalHelper(t *testing.T) {
	require.Equal(t, "TestProcessInstanceConfirmationTerminal", os.Getenv(testx.CmdSubprocessNameEnv))
	require.True(t, term.IsTerminal(int(os.Stdin.Fd())))

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetOut(os.Stdout)
	if os.Getenv(processInstanceConfirmStderrEnv) == "configured" {
		capture, err := os.Create(os.Getenv(processInstanceConfirmCaptureEnv))
		require.NoError(t, err)
		defer capture.Close()
		root.SetErr(io.MultiWriter(os.Stderr, capture))
	} else {
		root.SetErr(nil)
	}

	operation := os.Getenv(processInstanceConfirmOperationEnv)
	state := "active"
	if operation == "delete" {
		state = "completed"
	}
	root.SetArgs([]string{
		"--config", os.Getenv(processInstanceConfirmConfigEnv),
		"--log-format", os.Getenv(processInstanceConfirmFormatEnv),
		"--no-indicator",
		operation, "process-instance",
		"--state", state,
		"--no-wait",
	})
	_, _ = root.ExecuteC()
}

// runConfirmOrAbortTerminalHelper exercises the real helper and reports only terminal and decision evidence on stdout.
func runConfirmOrAbortTerminalHelper(t *testing.T) {
	require.Equal(t, "TestConfirmOrAbortTerminal", os.Getenv(testx.CmdSubprocessNameEnv))
	require.True(t, term.IsTerminal(int(os.Stdin.Fd())))
	fmt.Fprintf(os.Stdout, "terminal=%t\n", term.IsTerminal(int(os.Stdin.Fd())))

	var customPrompt bytes.Buffer
	var promptWriter io.Writer = os.Stderr
	switch os.Getenv(confirmTerminalScenarioEnv) {
	case "accept-nil-writer":
		promptWriter = nil
	case "accept-custom-writer":
		promptWriter = io.MultiWriter(os.Stderr, &customPrompt)
	}
	err := confirmCmdOrAbort(promptWriter, false, "Proceed with this deletion?")
	if os.Getenv(confirmTerminalScenarioEnv) == "accept-custom-writer" {
		require.Equal(t, confirmTerminalPrompt, customPrompt.String())
	}
	if err == nil {
		fmt.Fprintln(os.Stdout, "decision=accepted")
		return
	}
	require.ErrorContains(t, err, ErrCmdAborted.Error())
	fmt.Fprintln(os.Stdout, "decision=aborted")
}

// runConfirmOrAbortDefaultYesTerminalHelper exercises the real default-yes helper with independently captured results.
func runConfirmOrAbortDefaultYesTerminalHelper(t *testing.T) {
	require.Equal(t, "TestConfirmOrAbortDefaultYesTerminal", os.Getenv(testx.CmdSubprocessNameEnv))
	require.True(t, term.IsTerminal(int(os.Stdin.Fd())))
	fmt.Fprintf(os.Stdout, "terminal=%t\n", term.IsTerminal(int(os.Stdin.Fd())))

	var customPrompt bytes.Buffer
	var promptWriter io.Writer = os.Stderr
	switch os.Getenv(confirmTerminalScenarioEnv) {
	case "accept-nil-writer":
		promptWriter = nil
	case "accept-custom-writer":
		promptWriter = io.MultiWriter(os.Stderr, &customPrompt)
	}
	err := confirmCmdOrAbortDefaultYes(promptWriter, false, "List visible process definitions?")
	if os.Getenv(confirmTerminalScenarioEnv) == "accept-custom-writer" {
		require.Equal(t, confirmDefaultYesPrompt, customPrompt.String())
	}
	if err == nil {
		fmt.Fprintln(os.Stdout, "decision=accepted")
		return
	}
	require.ErrorContains(t, err, ErrCmdAborted.Error())
	fmt.Fprintln(os.Stdout, "decision=aborted")
}
