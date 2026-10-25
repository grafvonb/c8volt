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

const elementScopePagePrompt = "Fetched 1 element(s) on this page (1 loaded). More matching elements remain.\nContinue? [y/N]: "

// runElementScopeTerminal uses a real terminal input and redirected result file, checking both command-owned and inherited stderr.
func runElementScopeTerminal(t *testing.T, args []string, destination string, exchanges []testx.CmdTerminalExchange) (string, string, error) {
	t.Helper()
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	dir := t.TempDir()
	outPath, errPath := filepath.Join(dir, "stdout"), filepath.Join(dir, "stderr")
	result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
		ScopeTestName: "TestElementScopeVariablesGetTerminalProcess",
		Env:           map[string]string{"C8VOLT_ELEMENT_ARGS": string(encoded), "C8VOLT_ELEMENT_DESTINATION": destination, "C8VOLT_ELEMENT_STDOUT": outPath, "C8VOLT_ELEMENT_STDERR": errPath}, Exchanges: exchanges,
	})
	if !result.Supported {
		t.Skip(result.UnsupportedReason)
	}
	require.Empty(t, result.Stdout)
	stdout, err := os.ReadFile(outPath)
	require.NoError(t, err)
	stderr, err := os.ReadFile(errPath)
	require.NoError(t, err)
	require.Equal(t, string(stderr), result.Stderr, "all prompts and diagnostics must use the effective configured writer")
	return string(stdout), result.Stderr, result.Err
}

// TestElementScopeVariablesGetTerminalProcess executes the selected element read or update with real terminal stdin and separate configured streams.
func TestElementScopeVariablesGetTerminalProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	require.True(t, term.IsTerminal(int(os.Stdin.Fd())))
	require.False(t, term.IsTerminal(int(os.Stdout.Fd())))
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_ELEMENT_ARGS")), &args))
	stdout, err := os.Create(os.Getenv("C8VOLT_ELEMENT_STDOUT"))
	require.NoError(t, err)
	defer stdout.Close()
	stderr, err := os.Create(os.Getenv("C8VOLT_ELEMENT_STDERR"))
	require.NoError(t, err)
	defer stderr.Close()
	root := Root()
	root.SetOut(stdout)
	root.SetErr(nil)
	target := getElementCmd
	for _, arg := range args {
		if arg == "update" {
			target = updateElementCmd
			break
		}
	}
	target.SetErr(nil)
	writer := io.MultiWriter(os.Stderr, stderr)
	if os.Getenv("C8VOLT_ELEMENT_DESTINATION") == "configured" {
		target.SetErr(writer)
	} else {
		root.SetErr(writer)
	}
	root.SetArgs(args)
	require.NoError(t, root.Execute())
	os.Exit(0)
}

// TestElementScopeVariablesGetTerminal preserves continuation, decline/default/EOF and unattended policies with uncontaminated stdout.
func TestElementScopeVariablesGetTerminal(t *testing.T) {
	for _, destination := range []string{"configured", "inherited"} {
		for _, tc := range []struct {
			name, response   string
			eof, empty, keys bool
			flags            []string
			pages            int
		}{
			{name: "continue", response: "yes", pages: 2},
			{name: "decline", response: "no", pages: 1},
			{name: "default", pages: 1},
			{name: "EOF", eof: true, pages: 1},
			{name: "keys", response: "yes", keys: true, pages: 2},
			{name: "quiet keys", response: "yes", keys: true, flags: []string{"--quiet"}, pages: 2},
			{name: "auto confirm", flags: []string{"--auto-confirm"}, pages: 2},
			{name: "automation", flags: []string{"--automation"}, pages: 2},
			{name: "JSON", flags: []string{"--json"}, pages: 2},
			{name: "empty", empty: true, pages: 1},
			{name: "empty keys", empty: true, keys: true, pages: 1},
			{name: "empty quiet", empty: true, flags: []string{"--quiet"}, pages: 1},
			{name: "empty JSON", empty: true, flags: []string{"--json"}, pages: 1},
		} {
			t.Run(destination+"/"+tc.name, func(t *testing.T) {
				elements := []string{scopePage([]string{scopeElementJSON(scopeElementA)}, 2, true), scopePage([]string{scopeElementJSON(scopeElementB)}, 2, false)}
				if tc.empty {
					elements = []string{scopePage(nil, 0, false)}
				}
				f := newElementScopeFixture(t, elements, []string{scopePage(nil, 0, false), scopePage(nil, 0, false)}, 0)
				args := append([]string{"--config", testx.WriteTestConfig(t, f.server.URL), "--no-indicator", "get", "element", "--batch-size", "1", "--with-vars"}, tc.flags...)
				if tc.keys {
					args = append(args, "--keys-only")
				}
				var exchanges []testx.CmdTerminalExchange
				prompt := !tc.empty && !strings.Contains(strings.Join(tc.flags, " "), "--auto") && !strings.Contains(strings.Join(tc.flags, " "), "--json")
				if prompt {
					exchanges = []testx.CmdTerminalExchange{{Prompt: elementScopePagePrompt, Response: tc.response, EndOfInput: tc.eof}}
				}
				stdout, stderr, err := runElementScopeTerminal(t, args, destination, exchanges)
				require.NoError(t, err, stderr)
				if prompt {
					require.Equal(t, elementScopePagePrompt, stderr)
				} else {
					require.Empty(t, stderr)
				}
				require.NotContains(t, stdout, "Continue?")
				require.NotContains(t, stdout, "proceed?")
				calls := strings.Join(f.requests.Snapshot(), "\n")
				require.Equal(t, tc.pages, strings.Count(calls, "/v2/element-instances/search"))
				wantVars := tc.pages
				if tc.keys || tc.empty {
					wantVars = 0
				}
				require.Equal(t, wantVars, strings.Count(calls, "/v2/variables/search"))
				switch {
				case tc.keys:
					expected := scopeElementA + "\n"
					if tc.pages == 2 {
						expected += scopeElementB + "\n"
					}
					if tc.empty {
						expected = ""
					}
					require.Equal(t, expected, stdout)
				case strings.Contains(strings.Join(tc.flags, " "), "--json"):
					env := requireSingleJSONObjectDocument(t, stdout)
					require.Equal(t, "succeeded", env["outcome"])
				case strings.Contains(strings.Join(tc.flags, " "), "--quiet"):
					require.Empty(t, stdout)
				case tc.empty:
					require.Equal(t, "found: 0\n", stdout)
				default:
					require.True(t, strings.HasSuffix(stdout, fmt.Sprintf("found: %d\n", tc.pages)), stdout)
				}
			})
		}
	}
}
