// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"io"
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
