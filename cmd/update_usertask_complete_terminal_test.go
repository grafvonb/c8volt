// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

const userTaskCompletionPrompt = "You are about to complete 2 user task(s).\nDo you want to proceed? [y/N]: "

// TestUpdateUserTaskCompletionTerminal proves whole-selection confirmation with
// real terminal stdin, redirected stdout and both effective stderr destinations.
func TestUpdateUserTaskCompletionTerminal(t *testing.T) {
	for _, destination := range []string{"configured", "inherited"} {
		for _, tc := range []struct {
			name, response string
			flags          []string
			eof, decline   bool
		}{
			{name: "accept", response: "yes"}, {name: "decline", response: "no", decline: true},
			{name: "default", decline: true}, {name: "EOF", eof: true, decline: true},
			{name: "preview", flags: []string{"--dry-run"}},
			{name: "auto confirm", flags: []string{"--auto-confirm"}},
			{name: "automation", flags: []string{"--automation"}},
			{name: "quiet machine", flags: []string{"--automation", "--quiet", "--keys-only"}},
			{name: "JSON preview", flags: []string{"--dry-run", "--json"}},
		} {
			t.Run(destination+"/"+tc.name, func(t *testing.T) {
				server, calls := newUserTaskCompletionServer(t, nil)
				args := append([]string{"--config", writeTestConfigForVersion(t, server.URL, "8.10"), "--log-format", "plain-time", "update", "ut", "--complete", "--key", completionKeyA + "," + completionKeyB}, tc.flags...)
				preview := slices.Contains(tc.flags, "--dry-run")
				prompted := !preview && !slices.Contains(tc.flags, "--auto-confirm") && !slices.Contains(tc.flags, "--automation")
				var exchanges []testx.CmdTerminalExchange
				if prompted {
					exchanges = []testx.CmdTerminalExchange{{Prompt: userTaskCompletionPrompt, Response: tc.response, EndOfInput: tc.eof}}
				}
				stdout, stderr, err := runUserTaskCompletionTerminal(t, args, destination, exchanges)
				code := 0
				if tc.decline {
					code = 1
				}
				assertUserTaskCompletionExit(t, err, code)
				if tc.decline || preview {
					require.Empty(t, calls.Snapshot())
				} else {
					require.Len(t, calls.Snapshot(), 2)
				}
				if prompted {
					require.True(t, strings.HasPrefix(stderr, userTaskCompletionPrompt), stderr)
					require.Equal(t, 1, strings.Count(stderr, userTaskCompletionPrompt))
					stderr = strings.TrimPrefix(stderr, userTaskCompletionPrompt)
				}
				if tc.decline {
					require.Empty(t, stdout)
					require.Equal(t, [][2]string{{"ERROR", "local precondition failed: aborted by user"}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
				} else {
					require.Empty(t, stderr)
					if slices.Contains(tc.flags, "--json") {
						require.Equal(t, "succeeded", decodeUserTaskCompletionEnvelope(t, stdout)["outcome"])
					} else if slices.Contains(tc.flags, "--keys-only") {
						require.Equal(t, completionKeyA+"\n"+completionKeyB+"\n", stdout)
					} else if preview {
						require.Equal(t, "dry run: complete user-task: 2 user task(s), 0 supplied variable(s); no requests submitted\n", stdout)
					} else {
						require.Equal(t, "complete user-task: 2 requested; 2 accepted, 0 failed, 0 timed out, 0 canceled, 0 not attempted; backend confirmed completion of accepted tasks; no additional state polling\n", stdout)
					}
				}
				require.NotContains(t, stdout, "Do you want to proceed?")
			})
		}
	}
}

// runUserTaskCompletionTerminal captures actual configured/inherited error writers
// and redirected results independently; no terminal eligibility is mocked.
func runUserTaskCompletionTerminal(t *testing.T, args []string, destination string, exchanges []testx.CmdTerminalExchange) (string, string, error) {
	t.Helper()
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	dir := t.TempDir()
	outPath, errPath := filepath.Join(dir, "stdout"), filepath.Join(dir, "stderr")
	result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
		ScopeTestName: "TestUpdateUserTaskCompletionTerminalProcess",
		Env:           map[string]string{"C8VOLT_COMPLETION_ARGS": string(encoded), "C8VOLT_COMPLETION_DESTINATION": destination, "C8VOLT_COMPLETION_STDOUT": outPath, "C8VOLT_COMPLETION_STDERR": errPath}, Exchanges: exchanges,
	})
	if !result.Supported {
		t.Skip(result.UnsupportedReason)
	}
	require.Empty(t, result.Stdout)
	stdout, readErr := os.ReadFile(outPath)
	require.NoError(t, readErr)
	stderr, readErr := os.ReadFile(errPath)
	require.NoError(t, readErr)
	require.Equal(t, string(stderr), result.Stderr, "diagnostics/prompts must honor the effective command writer")
	return string(stdout), result.Stderr, result.Err
}

// TestUpdateUserTaskCompletionTerminalProcess is an isolated real-terminal entry
// point with an actual output file and a command-specific or inherited writer.
func TestUpdateUserTaskCompletionTerminalProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	require.True(t, term.IsTerminal(int(os.Stdin.Fd())))
	require.False(t, term.IsTerminal(int(os.Stdout.Fd())))
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_COMPLETION_ARGS")), &args))
	stdout, err := os.Create(os.Getenv("C8VOLT_COMPLETION_STDOUT"))
	require.NoError(t, err)
	defer stdout.Close()
	stderr, err := os.Create(os.Getenv("C8VOLT_COMPLETION_STDERR"))
	require.NoError(t, err)
	defer stderr.Close()
	root := Root()
	root.SetOut(stdout)
	root.SetErr(nil)
	updateUserTaskCmd.SetErr(nil)
	writer := io.MultiWriter(os.Stderr, stderr)
	if os.Getenv("C8VOLT_COMPLETION_DESTINATION") == "configured" {
		updateUserTaskCmd.SetErr(writer)
	} else {
		root.SetErr(writer)
	}
	root.SetArgs(args)
	require.NoError(t, root.Execute())
	os.Exit(0)
}
