// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

const (
	confirmTerminalScenarioEnv = "C8VOLT_CONFIRM_TERMINAL_SCENARIO"
	confirmTerminalPrompt      = "Proceed with this deletion? [y/N]: "
	confirmDefaultYesPrompt    = "List visible process definitions? [Y/n]: "
	emptySelectorConfigEnv     = "C8VOLT_EMPTY_SELECTOR_CONFIG"
	emptySelectorOperationEnv  = "C8VOLT_EMPTY_SELECTOR_OPERATION"
	emptySelectorDryRunEnv     = "C8VOLT_EMPTY_SELECTOR_DRY_RUN"
	emptySelectorModeEnv       = "C8VOLT_EMPTY_SELECTOR_MODE"
)

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
