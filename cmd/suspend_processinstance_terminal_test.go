// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

const suspensionPrompt = "You are about to suspend 1 process instance(s).\nDo you want to proceed? [y/N]: "

// runSuspensionTerminal verifies both configured and inherited stderr with redirected stdout.
func runSuspensionTerminal(t *testing.T, args []string, destination string, exchanges []testx.CmdTerminalExchange) (string, string, error) {
	t.Helper()
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	dir := t.TempDir()
	outPath, errPath := filepath.Join(dir, "stdout"), filepath.Join(dir, "stderr")
	result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{ScopeTestName: "TestSuspendProcessInstanceTerminalProcess", Env: map[string]string{"C8VOLT_SUSPEND_ARGS": string(encoded), "C8VOLT_SUSPEND_DESTINATION": destination, "C8VOLT_SUSPEND_STDOUT": outPath, "C8VOLT_SUSPEND_STDERR": errPath}, Exchanges: exchanges})
	if !result.Supported {
		t.Skip(result.UnsupportedReason)
	}
	require.Empty(t, result.Stdout)
	out, err := os.ReadFile(outPath)
	require.NoError(t, err)
	stderr, err := os.ReadFile(errPath)
	require.NoError(t, err)
	require.Equal(t, string(stderr), result.Stderr, "all diagnostics and prompts must use the configured destination")
	return string(out), result.Stderr, result.Err
}

// TestSuspendProcessInstanceTerminalProcess keeps terminal stdin while capturing results and
// the effective command logger destination independently; no terminal detection is mocked.
func TestSuspendProcessInstanceTerminalProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	require.True(t, term.IsTerminal(int(os.Stdin.Fd())))
	require.False(t, term.IsTerminal(int(os.Stdout.Fd())))
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_SUSPEND_ARGS")), &args))
	stdout, err := os.Create(os.Getenv("C8VOLT_SUSPEND_STDOUT"))
	require.NoError(t, err)
	defer stdout.Close()
	stderr, err := os.Create(os.Getenv("C8VOLT_SUSPEND_STDERR"))
	require.NoError(t, err)
	defer stderr.Close()
	root := Root()
	root.SetOut(stdout)
	root.SetErr(nil)
	suspendProcessInstanceCmd.SetErr(nil)
	writer := io.MultiWriter(os.Stderr, stderr)
	if os.Getenv("C8VOLT_SUSPEND_DESTINATION") == "configured" {
		suspendProcessInstanceCmd.SetErr(writer)
	} else {
		root.SetErr(writer)
	}
	root.SetArgs(args)
	require.NoError(t, root.Execute())
	os.Exit(0)
}

// TestSuspendProcessInstanceTerminal proves no I/O/output before approval, plain stderr
// prompts, default-no/EOF/abort behavior and prompt suppression on previews and invalid scopes.
func TestSuspendProcessInstanceTerminal(t *testing.T) {
	for _, destination := range []string{"configured", "inherited"} {
		for _, tc := range []struct {
			name, response, version                    string
			flags                                      []string
			eof, approve, dry, noPrompt, missing, dash bool
			code                                       int
		}{
			{name: "approve", response: "yes", approve: true}, {name: "decline", response: "no", code: 1}, {name: "default", code: 1}, {name: "EOF", eof: true, code: 1},
			{name: "keys", response: "y", approve: true, flags: []string{"--keys-only"}},
			{name: "quiet keys", response: "y", approve: true, flags: []string{"--keys-only", "--quiet"}},
			{name: "no-wait", response: "y", approve: true, flags: []string{"--no-wait"}},
			{name: "auto-confirm", approve: true, noPrompt: true, flags: []string{"--auto-confirm"}},
			{name: "automation", approve: true, noPrompt: true, flags: []string{"--automation"}},
			{name: "json", approve: true, noPrompt: true, flags: []string{"--auto-confirm", "--json"}},
			{name: "preview", dry: true, noPrompt: true, flags: []string{"--dry-run"}},
			{name: "preview keys no-wait", dry: true, noPrompt: true, flags: []string{"--dry-run", "--no-wait", "--keys-only"}},
			{name: "preview json", dry: true, noPrompt: true, flags: []string{"--dry-run", "--json"}},
			{name: "missing", missing: true, noPrompt: true, code: 1},
			{name: "missing no-wait", missing: true, noPrompt: true, code: 1, flags: []string{"--no-wait"}},
			{name: "terminal dash", dash: true, noPrompt: true, code: 2},
			{name: "unsupported", version: "8.9", noPrompt: true, code: 1},
		} {
			t.Run(destination+"/"+tc.name, func(t *testing.T) {
				f := newSuspensionFixture(t)
				cfg := completionLogConfig(t, f.server.URL, "plain-time", "info")
				if tc.version != "" {
					cfg = writeTestConfigForVersion(t, f.server.URL, tc.version)
				}
				args := []string{"--config", cfg, "--log-format", "plain-time", "suspend", "pi"}
				if !tc.missing {
					args = append(args, "--key", suspendKeyA)
				}
				if tc.dash {
					args = append(args, "-")
				}
				args = append(args, tc.flags...)
				var exchanges []testx.CmdTerminalExchange
				if !tc.noPrompt {
					exchanges = []testx.CmdTerminalExchange{{Prompt: suspensionPrompt, Response: tc.response, EndOfInput: tc.eof, BeforeResponse: func(out, stderr string) {
						calls, _, _ := f.snapshot()
						require.Empty(t, calls)
						require.Empty(t, out)
						require.Equal(t, suspensionPrompt, stderr)
					}}}
				}
				out, stderr, err := runSuspensionTerminal(t, args, destination, exchanges)
				assertUserTaskCompletionExit(t, err, tc.code)
				if !tc.noPrompt {
					require.Equal(t, 1, strings.Count(stderr, suspensionPrompt))
					stderr = strings.Replace(stderr, suspensionPrompt, "", 1)
				} else {
					require.NotContains(t, stderr, "proceed?")
				}
				require.NotContains(t, out, "proceed?")
				require.NotContains(t, out, "[y/N]")
				require.NotContains(t, out, "\r")
				flags := strings.Join(tc.flags, " ")
				calls, writes, reads := f.snapshot()
				if tc.approve {
					require.Equal(t, map[string]int{suspendKeyA: 1}, writes)
					if strings.Contains(flags, "--no-wait") {
						require.Empty(t, reads)
						require.Len(t, calls, 1)
					} else {
						require.Equal(t, writes, reads)
						require.Len(t, calls, 2)
					}
				} else {
					require.Empty(t, calls)
				}
				if tc.code != 0 {
					require.Empty(t, out)
					records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
					require.Len(t, records, 1)
					require.Equal(t, "ERROR", records[0][0])
					if !tc.noPrompt {
						require.Equal(t, "suspend process instances: aborted by user", records[0][1])
					}
					return
				}
				require.Empty(t, stderr)
				switch {
				case strings.Contains(flags, "--json"):
					require.Equal(t, "succeeded", requireSingleJSONObjectDocument(t, out)["outcome"])
				case strings.Contains(flags, "--keys-only"):
					require.Equal(t, suspendKeyA+"\n", out)
				case tc.dry:
					require.Equal(t, "dry run: suspend process instances: 1 selected; no changes applied; eligibility not checked\n", out)
				default:
					confirmed := 1
					if strings.Contains(flags, "--no-wait") {
						confirmed = 0
					}
					require.Equal(t, "suspend process instances: 1 selected; accepted: 1, confirmed: "+fmt.Sprint(confirmed)+", failed: 0, skipped: 0\n", out)
				}
			})
		}
	}
}
