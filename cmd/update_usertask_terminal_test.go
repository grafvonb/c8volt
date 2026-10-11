// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const (
	updateUserTaskTerminalArgsEnv    = "C8VOLT_UPDATE_USER_TASK_TERMINAL_ARGS"
	updateUserTaskTerminalStderrEnv  = "C8VOLT_UPDATE_USER_TASK_TERMINAL_STDERR"
	updateUserTaskTerminalCaptureEnv = "C8VOLT_UPDATE_USER_TASK_TERMINAL_CAPTURE"
)

const updateUserTaskTerminalPrompt = "You are about to update 1 requested variable value(s) on 1 user task(s).\nDo you want to proceed? [y/N]: "

// TestUpdateUserTaskTerminal proves real-terminal confirmation, abort, and
// unattended policies preserve separate result and control streams.
func TestUpdateUserTaskTerminal(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		runUpdateUserTaskTerminalHelper(t)
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
		wantTaskReads int
	}{
		{name: "configured stderr accepts", args: []string{"update", "ut", "--key", "101", "--vars", `{"approved":true}`}, configured: true, prompt: true, preview: true, response: "yes", wantMutates: 1, wantReads: 2},
		{name: "inherited stderr declines", args: []string{"update", "ut", "--key", "101", "--vars", `{"approved":true}`}, prompt: true, preview: true, response: "no", wantExitCode: exitcode.Error, wantReads: 1},
		{name: "default answer declines", args: []string{"update", "ut", "--key", "101", "--vars", `{"approved":true}`}, prompt: true, preview: true, wantExitCode: exitcode.Error, wantReads: 1},
		{name: "terminal eof aborts", args: []string{"update", "ut", "--key", "101", "--vars", `{"approved":true}`}, prompt: true, preview: true, endOfInput: true, wantExitCode: exitcode.Error, wantReads: 1},
		{name: "dry run is prompt free", args: []string{"update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--dry-run"}, preview: true, wantReads: 1},
		{name: "no-op is prompt free", args: []string{"update", "ut", "--key", "101", "--vars", `{"approved":true}`}, initialValue: true, wantReads: 1},
		{name: "automation no-wait is prompt free", args: []string{"--automation", "update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--no-wait"}, wantMutates: 1, wantReads: 1},
		{name: "auto-confirm is prompt free", args: []string{"--auto-confirm", "update", "ut", "--key", "101", "--vars", `{"approved":true}`}, wantMutates: 1, wantReads: 2},
		{
			name:         "mixed destinations normal preview declines without writes",
			args:         []string{"update", "ut", "--key", "103", "--vars", `{"approved":true,"middle":2,"local":true,"new":"value"}`},
			prompt:       true,
			promptText:   "You are about to update 4 requested variable value(s) on 1 user task(s).\nDo you want to proceed? [y/N]: ",
			response:     "no",
			wantExitCode: exitcode.Error,
			wantReads:    1,
			wantPreview: []string{
				"~ approved (inherited, pi:301): false -> true",
				"~ middle (inherited, element:203): 1 -> 2",
				"~ local (local, element:501103): false -> true",
				`+ new (local, element:501103): "value"`,
			},
		},
		{
			name:          "compact bulk normal preview stays summarized",
			args:          []string{"update", "ut", "--key", "103,104", "--vars", `{"approved":true,"middle":2,"local":true,"new":"value"}`},
			prompt:        true,
			promptText:    "You are about to update 8 requested variable value(s) on 2 user task(s).\nDo you want to proceed? [y/N]: ",
			response:      "no",
			wantExitCode:  exitcode.Error,
			wantReads:     2,
			wantTaskReads: 2,
			wantPreview: []string{
				"plan: update user-task variables: 2 user task(s), 6 change(s), 2 addition(s), 0 unchanged, 0 untouched",
				"inherited target scopes: pi:301, element:203",
			},
			absentPreview: []string{"103: ~ approved", "104: ~ approved"},
		},
		{
			name:          "verbose bulk normal preview labels every write",
			args:          []string{"--verbose", "update", "ut", "--key", "103,104", "--vars", `{"approved":true,"middle":2,"local":true,"new":"value"}`},
			prompt:        true,
			promptText:    "You are about to update 8 requested variable value(s) on 2 user task(s).\nDo you want to proceed? [y/N]: ",
			response:      "no",
			wantExitCode:  exitcode.Error,
			wantReads:     2,
			wantTaskReads: 2,
			wantPreview: []string{
				"103: ~ approved (inherited, pi:301): false -> true; ~ local (local, element:501103): false -> true; ~ middle (inherited, element:203): 1 -> 2; + new (local, element:501103): \"value\"",
				"104: ~ approved (inherited, pi:301): false -> true; ~ local (local, element:501104): false -> true; ~ middle (inherited, element:203): 1 -> 2; + new (local, element:501104): \"value\"",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wantTaskReads := test.wantTaskReads
			if wantTaskReads == 0 {
				wantTaskReads = 1
			}
			server, capture := newUpdateUserTaskOutputServer(t, test.initialValue, "")
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
					promptText = updateUserTaskTerminalPrompt
				}
				exchanges = []testx.CmdTerminalExchange{{
					Prompt:     promptText,
					Response:   test.response,
					EndOfInput: test.endOfInput,
				}}
			}
			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
				ScopeTestName: "TestUpdateUserTaskTerminal",
				Env: map[string]string{
					"C8VOLT_TEST_CONFIG":             writeTestConfigForVersion(t, server.URL, "8.10"),
					updateUserTaskTerminalArgsEnv:    string(encodedArgs),
					updateUserTaskTerminalStderrEnv:  stderrMode,
					updateUserTaskTerminalCaptureEnv: capturePath,
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
					promptText = updateUserTaskTerminalPrompt
				}
				require.Equal(t, 1, strings.Count(result.Stderr, promptText))
			} else {
				require.NotContains(t, result.Stderr, "Do you want to proceed?")
			}
			if test.preview {
				require.Contains(t, result.Stderr, "~ approved (inherited, element:201): false -> true")
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
			require.Equal(t, test.wantMutates, capture.mutations)
			require.Equal(t, wantTaskReads, capture.taskReads)
			require.Equal(t, test.wantReads, capture.variableReads)
			require.Equal(t, wantTaskReads+test.wantReads+test.wantMutates, capture.totalRequests)
		})
	}
}

// runUpdateUserTaskTerminalHelper executes the real command with terminal stdin
// while selecting either a configured root stderr or inherited process stderr.
func runUpdateUserTaskTerminalHelper(t *testing.T) {
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv(updateUserTaskTerminalArgsEnv)), &args))
	root := Root()
	resetCommandTreeFlags(root)
	root.SetOut(os.Stdout)
	updateUserTaskCmd.SetErr(nil)
	if os.Getenv(updateUserTaskTerminalStderrEnv) == "configured" {
		capture, err := os.Create(os.Getenv(updateUserTaskTerminalCaptureEnv))
		require.NoError(t, err)
		defer capture.Close()
		root.SetErr(io.MultiWriter(os.Stderr, capture))
	} else {
		root.SetErr(nil)
	}
	root.SetArgs(append([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG")}, args...))
	_ = root.Execute()
}
