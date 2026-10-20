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
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

// Literal expectations deliberately do not reuse the production prompt builder.
const bulkCompletionTerminalPrompt = "You are about to complete 3 selected job(s); selection scope: tenant-a only; filters: type=cleanup. Completing a listener job acknowledges its work without executing its business logic.\nDo you want to proceed? [y/N]: "
const bulkCompletionTerminalReady = "test terminal input ready\n"

// TestBulkCompletionTerminalStreams proves real terminal decisions, inherited and
// command-specific error writers, file redirection, and prompt-free machine modes.
func TestBulkCompletionTerminalStreams(t *testing.T) {
	tests := []struct {
		name, destination, response string
		flags                       []string
		eof, decline, partial       bool
		redirect                    bool
	}{
		{name: "ordinary accepts inherited", destination: "inherited", response: "yes"},
		{name: "verbose accepts configured", destination: "configured", response: "y", flags: []string{"--verbose"}},
		{name: "redirected ordinary accepts configured", destination: "configured", redirect: true, response: "y"},
		{name: "redirected verbose accepts inherited", destination: "inherited", redirect: true, response: "y", flags: []string{"--verbose"}},
		{name: "decline configured", destination: "configured", response: "n", decline: true},
		{name: "blank defaults to no inherited", destination: "inherited", decline: true},
		{name: "EOF configured", destination: "configured", eof: true, decline: true},
		{name: "redirected EOF inherited", destination: "inherited", redirect: true, eof: true, decline: true},
		{name: "quiet accepts default stderr", response: "y", flags: []string{"--quiet"}},
		{name: "auto confirm human", destination: "configured", flags: []string{"--auto-confirm"}},
		{name: "automation human", destination: "inherited", flags: []string{"--automation"}},
		{name: "JSON auto confirm configured", destination: "configured", flags: []string{"--json", "--auto-confirm"}},
		{name: "JSON automation inherited", destination: "inherited", flags: []string{"--json", "--automation"}},
		{name: "quiet JSON redirected", destination: "inherited", redirect: true, flags: []string{"--quiet", "--json", "--auto-confirm"}},
		{name: "quiet JSON no wait", destination: "configured", flags: []string{"--quiet", "--json", "--automation", "--no-wait"}},
		{name: "ordinary preview inherited", destination: "inherited", flags: []string{"--dry-run"}},
		{name: "verbose preview configured", destination: "configured", flags: []string{"--dry-run", "--verbose"}},
		{name: "JSON preview inherited", destination: "inherited", flags: []string{"--json", "--dry-run"}},
		{name: "quiet JSON preview redirected", destination: "configured", redirect: true, flags: []string{"--quiet", "--json", "--dry-run"}},
		{name: "partial JSON configured", destination: "configured", partial: true, flags: []string{"--json", "--auto-confirm"}},
		{name: "partial quiet JSON redirected", destination: "inherited", partial: true, redirect: true, flags: []string{"--quiet", "--json", "--automation"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls, server := newBulkCompletionTerminalServer(t, false, tt.partial)
			args := bulkCompletionTerminalArgs(t, server.URL, false)
			args = append(args, tt.flags...)
			preview := slices.Contains(tt.flags, "--dry-run")
			jsonMode := slices.Contains(tt.flags, "--json")
			prompted := !preview && !slices.Contains(tt.flags, "--auto-confirm") && !slices.Contains(tt.flags, "--automation")
			var exchanges []testx.CmdTerminalExchange
			if prompted {
				exchanges = []testx.CmdTerminalExchange{{Prompt: bulkCompletionTerminalPrompt, Response: tt.response, EndOfInput: tt.eof}}
			}
			stdout, stderr, runErr := runBulkCompletionTerminal(t, args, tt.destination, tt.redirect, false, exchanges)
			if tt.decline || tt.partial {
				var exitErr *exec.ExitError
				require.ErrorAs(t, runErr, &exitErr)
				require.NotZero(t, exitErr.ExitCode())
			} else {
				require.NoError(t, runErr, stderr)
			}

			// Keep prompts plain and exact; a decline must then emit exactly
			// one configured ERROR record, not a direct stderr print.
			wantStderr := ""
			if prompted {
				wantStderr = bulkCompletionTerminalPrompt
			}
			if tt.decline {
				require.True(t, strings.HasPrefix(stderr, wantStderr), stderr)
				records := bulkCompletionDiagnosticRecords(t, strings.TrimPrefix(stderr, wantStderr), "plain-time")
				require.Equal(t, [][2]string{{"ERROR", "job completion: aborted by user; no changes applied"}}, records)
			} else {
				require.Equal(t, wantStderr, stderr)
			}
			require.NotContains(t, stdout, "Do you want to proceed?")
			require.NotContains(t, stdout, "Continue?")

			accepted, unsatisfied, notStarted := 3, 0, 0
			status, outcome := "finished", "succeeded"
			if preview {
				status, accepted = "preview", 0
			} else if tt.decline {
				status, accepted, notStarted = "aborted", 0, 3
			} else if tt.partial {
				accepted, unsatisfied, outcome = 2, 1, "failed"
			} else if slices.Contains(tt.flags, "--no-wait") {
				outcome = "accepted"
			}
			if jsonMode {
				envelope := decodeBulkCompletionEnvelope(t, stdout)
				require.Equal(t, outcome, envelope["outcome"])
				payload := requireJSONObject(t, envelope["payload"])
				require.Equal(t, status, payload["status"])
				require.Equal(t, map[string]any{"selected": float64(3), "accepted": float64(accepted), "completed": float64(0), "unsatisfied": float64(unsatisfied), "notStarted": float64(notStarted)}, payload["counts"])
				if preview {
					require.Len(t, payload["candidates"], 3)
					require.NotContains(t, payload, "outcomes")
				} else {
					require.NotContains(t, payload, "candidates")
					rows := payload["outcomes"].([]any)
					require.Len(t, rows, 3)
					for i, value := range rows {
						row := requireJSONObject(t, value)
						require.Equal(t, strconv.Itoa(i+1), requireJSONObject(t, row["candidate"])["key"])
						disposition := "accepted"
						if tt.partial && i == 1 {
							disposition = "unsatisfied"
							require.Equal(t, "other_terminal", row["reason"])
							require.Equal(t, "not_submitted", requireJSONObject(t, row["submission"])["status"])
						} else {
							require.Equal(t, "accepted", requireJSONObject(t, row["submission"])["status"])
							require.Equal(t, "skipped", requireJSONObject(t, row["observation"])["status"])
						}
						require.Equal(t, disposition, row["disposition"])
					}
				}
			} else if slices.Contains(tt.flags, "--quiet") {
				require.Empty(t, stdout)
			} else {
				require.Contains(t, stdout, "selected job(s): 3\n")
				if preview {
					require.Contains(t, stdout, "Completing a listener job acknowledges its work without executing its business logic.\noutcome: preview; no changes applied\n")
				} else {
					require.Contains(t, stdout, fmt.Sprintf("outcome: %s; accepted %d; observed completed 0; unsatisfied %d; not started %d\n", status, accepted, unsatisfied, notStarted))
				}
				if slices.Contains(tt.flags, "--verbose") {
					require.Contains(t, stdout, "discovery: pages 3; batch size 1\n")
					require.Contains(t, stdout, "pi:2251799813711000")
				} else {
					require.NotContains(t, stdout, "discovery:")
					require.NotContains(t, stdout, "pi:")
				}
			}

			wantCalls := []string{"search:0", "search:1", "search:2"}
			if !preview && !tt.decline {
				for _, key := range []string{"1", "2", "3"} {
					wantCalls = append(wantCalls, "read:"+key)
					if !tt.partial || key != "2" {
						wantCalls = append(wantCalls, "submit:"+key)
					}
				}
			}
			require.Equal(t, wantCalls, calls.Snapshot(), "complete discovery, then one attempt per eligible frozen key")
		})
	}
}

// TestBulkCompletionTerminalEmpty keeps terminal stdin open for empty discovery
// and supplies canonical EOF for empty stdin; neither may request confirmation.
func TestBulkCompletionTerminalEmpty(t *testing.T) {
	for _, stdin := range []bool{false, true} {
		for _, lifecycle := range []string{"normal", "dry-run", "no-wait"} {
			for _, mode := range []string{"human", "verbose", "quiet", "json", "quiet-json"} {
				t.Run(fmt.Sprintf("stdin=%t/%s/%s", stdin, lifecycle, mode), func(t *testing.T) {
					calls, server := newBulkCompletionTerminalServer(t, true, false)
					args := bulkCompletionTerminalArgs(t, server.URL, stdin)
					if lifecycle != "normal" {
						args = append(args, "--"+lifecycle)
					}
					switch mode {
					case "verbose":
						args = append(args, "--verbose")
					case "quiet":
						args = append(args, "--quiet")
					case "json":
						args = append(args, "--json")
					case "quiet-json":
						args = append(args, "--quiet", "--json")
					}
					jsonMode := strings.Contains(mode, "json")
					if jsonMode && lifecycle != "dry-run" {
						// Preserve JSON mutation eligibility even when discovery
						// will later establish that no work exists.
						args = append(args, "--auto-confirm")
					}
					var exchanges []testx.CmdTerminalExchange
					wantStderr := ""
					if stdin {
						exchanges = []testx.CmdTerminalExchange{{Prompt: bulkCompletionTerminalReady, EndOfInput: true}}
						wantStderr = bulkCompletionTerminalReady
					}
					destination := "configured"
					if stdin {
						destination = "inherited"
					}
					stdout, stderr, runErr := runBulkCompletionTerminal(t, args, destination, jsonMode, stdin, exchanges)
					require.NoError(t, runErr, stderr)
					require.Equal(t, wantStderr, stderr)
					if stdin {
						require.Empty(t, calls.Snapshot())
					} else {
						require.Equal(t, []string{"search:0"}, calls.Snapshot())
					}
					if jsonMode {
						envelope := decodeBulkCompletionEnvelope(t, stdout)
						require.Equal(t, "succeeded", envelope["outcome"], "empty no-wait never claims acceptance")
						payload := requireJSONObject(t, envelope["payload"])
						require.Equal(t, map[string]any{"selected": float64(0), "accepted": float64(0), "completed": float64(0), "unsatisfied": float64(0), "notStarted": float64(0)}, payload["counts"])
						field, omitted, status := "outcomes", "candidates", "finished"
						if lifecycle == "dry-run" {
							field, omitted, status = "candidates", "outcomes", "preview"
						}
						require.Equal(t, status, payload["status"])
						require.Equal(t, []any{}, payload[field])
						require.NotContains(t, payload, omitted)
					} else if mode == "quiet" {
						require.Empty(t, stdout)
					} else {
						header, selection, scope, result := "job completion\n", "filters: type=cleanup\n", "selection scope: tenant-a only\n", "outcome: no matching jobs; no changes applied\n"
						tenantProvenance := "configured tenant: none\n--tenant \"tenant-a\" sets the tenant filter\n"
						if stdin {
							selection, scope, result = "selection: explicit keys\n", "selection scope: explicit resource keys; tenant filter not applied\n", "outcome: no selected jobs; no changes applied\n"
							tenantProvenance = ""
						}
						if lifecycle == "dry-run" {
							header = "dry run: job completion\n"
							result = "Completing a listener job acknowledges its work without executing its business logic.\noutcome: preview; no changes applied\n"
						}
						verbose := ""
						if mode == "verbose" && !stdin {
							verbose = "discovery: pages 1; batch size 1\n"
						}
						require.Equal(t, header+selection+tenantProvenance+scope+"selected job(s): 0\n"+verbose+result, stdout)
					}
				})
			}
		}
	}
}

// bulkCompletionTerminalArgs keeps all terminal scenarios on the real command,
// with serial execution making the expected HTTP sequence deterministic.
func bulkCompletionTerminalArgs(t *testing.T, serverURL string, stdin bool) []string {
	t.Helper()
	args := []string{"--config", writeTestConfigForVersion(t, serverURL, "8.8"), "--tenant", "tenant-a", "--no-indicator", "update", "job", "--complete", "--workers", "1"}
	if stdin {
		return append(args, "-")
	}
	return append(args, "--type", "cleanup", "--batch-size", "1")
}

// newBulkCompletionTerminalServer supplies three separate pages so a prompt
// between pages or before preparation completes cannot pass terminal acceptance.
func newBulkCompletionTerminalServer(t *testing.T, empty, partial bool) (*testx.SafeSlice[string], *httptest.Server) {
	t.Helper()
	calls := &testx.SafeSlice[string]{}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == "/v2/jobs/search" {
			var body struct {
				Filter map[string]any `json:"filter"`
				Page   struct {
					From int `json:"from"`
				} `json:"page"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode terminal search request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if key, ok := body.Filter["jobKey"].(string); ok {
				calls.Append("read:" + key)
				state := "CREATED"
				if partial && key == "2" {
					state = "CANCELED"
				}
				fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":1}}`, bulkCompletionRow(key, state))
				return
			}
			calls.Append("search:" + strconv.Itoa(body.Page.From))
			if empty {
				io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
				return
			}
			if body.Page.From < 0 || body.Page.From >= 3 {
				t.Errorf("unexpected terminal discovery offset: %d", body.Page.From)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":3}}`, bulkCompletionRow(strconv.Itoa(body.Page.From+1), "CREATED"))
			return
		}
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v2/jobs/") && strings.HasSuffix(r.URL.Path, "/completion") {
			calls.Append("submit:" + strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/jobs/"), "/completion"))
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Errorf("unexpected terminal workflow request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)
	return calls, server
}

// runBulkCompletionTerminal checks the configured writer's independent file
// capture and returns redirected results without mixing them with control text.
func runBulkCompletionTerminal(t *testing.T, args []string, destination string, redirect, stdin bool, exchanges []testx.CmdTerminalExchange) (string, string, error) {
	t.Helper()
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	dir := t.TempDir()
	env := map[string]string{
		"C8VOLT_TERMINAL_BATCH_ARGS":        string(encoded),
		"C8VOLT_TERMINAL_BATCH_DESTINATION": destination,
		"C8VOLT_TERMINAL_BATCH_STDERR":      filepath.Join(dir, "stderr"),
	}
	if redirect {
		env["C8VOLT_TERMINAL_BATCH_STDOUT"] = filepath.Join(dir, "stdout")
	}
	if stdin {
		env["C8VOLT_TERMINAL_BATCH_STDIN"] = "1"
	}
	result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
		ScopeTestName: "TestBulkCompletionTerminalProcess",
		Env:           env,
		Exchanges:     exchanges,
	})
	if !result.Supported {
		t.Skip(result.UnsupportedReason)
	}
	if destination != "" {
		captured, err := os.ReadFile(env["C8VOLT_TERMINAL_BATCH_STDERR"])
		require.NoError(t, err)
		require.Equal(t, result.Stderr, string(captured), "all control text must use the effective command error writer")
	}
	stdout := result.Stdout
	if redirect {
		require.Empty(t, stdout, "results must go to the redirected file")
		captured, err := os.ReadFile(env["C8VOLT_TERMINAL_BATCH_STDOUT"])
		require.NoError(t, err)
		stdout = string(captured)
	}
	return stdout, result.Stderr, result.Err
}

// TestBulkCompletionTerminalProcess runs only as an isolated helper; real
// terminal stdin and nonterminal stdout are asserted, never mocked.
func TestBulkCompletionTerminalProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	require.True(t, term.IsTerminal(int(os.Stdin.Fd())))
	require.False(t, term.IsTerminal(int(os.Stdout.Fd())))
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TERMINAL_BATCH_ARGS")), &args))
	root := Root()
	root.SetOut(os.Stdout)
	root.SetErr(nil)
	if path := os.Getenv("C8VOLT_TERMINAL_BATCH_STDOUT"); path != "" {
		output, err := os.Create(path)
		require.NoError(t, err)
		defer output.Close()
		root.SetOut(output)
	}
	command, _, err := root.Find([]string{"update", "job"})
	require.NoError(t, err)
	if destination := os.Getenv("C8VOLT_TERMINAL_BATCH_DESTINATION"); destination != "" {
		capture, err := os.Create(os.Getenv("C8VOLT_TERMINAL_BATCH_STDERR"))
		require.NoError(t, err)
		defer capture.Close()
		writer := io.MultiWriter(os.Stderr, capture)
		if destination == "configured" {
			command.SetErr(writer)
		} else {
			root.SetErr(writer)
			command.SetErr(nil)
		}
	}
	// A test-only readiness marker allows canonical EOF without introducing
	// product prompts or timing assumptions into empty stdin scenarios.
	if os.Getenv("C8VOLT_TERMINAL_BATCH_STDIN") == "1" {
		fmt.Fprint(command.ErrOrStderr(), bulkCompletionTerminalReady)
	}
	root.SetArgs(args)
	if _, err := root.ExecuteC(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
