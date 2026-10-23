// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

// runDefinitionDeleteTerminal reuses the repository terminal runner with separate
// configured/inherited diagnostic capture and real terminal stdin.
func runDefinitionDeleteTerminal(t *testing.T, args []string, destination string, exchanges []testx.CmdTerminalExchange) (string, string, error) {
	t.Helper()
	raw, err := json.Marshal(args)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "stderr")
	result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{ScopeTestName: "TestDefinitionDeleteTerminalProcess", Env: map[string]string{"C8VOLT_DEFINITION_ARGS": string(raw), "C8VOLT_DEFINITION_DESTINATION": destination, "C8VOLT_DEFINITION_STDERR": path}, Exchanges: exchanges})
	if !result.Supported {
		t.Skip(result.UnsupportedReason)
	}
	captured, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, string(captured), result.Stderr, "diagnostics must use configured or inherited command writer")
	return result.Stdout, result.Stderr, result.Err
}

// TestDefinitionDeleteTerminalProcess runs definition-delete or purge arguments
// only inside the test-owned terminal subprocess and asserts genuine terminal stdin.
func TestDefinitionDeleteTerminalProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	require.True(t, term.IsTerminal(int(os.Stdin.Fd())))
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_DEFINITION_ARGS")), &args))
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetOut(os.Stdout)
	root.SetErr(nil)
	commandPath := []string{"delete", "process-definition"}
	if slices.Contains(args, "all-process-definitions") {
		commandPath = []string{"ops", "purge", "all-process-definitions"}
		resetOpsPurgeAllProcessDefinitionsFlagState()
	}
	command, _, err := root.Find(commandPath)
	require.NoError(t, err)
	capture, err := os.Create(os.Getenv("C8VOLT_DEFINITION_STDERR"))
	require.NoError(t, err)
	writer := io.MultiWriter(os.Stderr, capture)
	if os.Getenv("C8VOLT_DEFINITION_DESTINATION") == "configured" {
		command.SetErr(writer)
	} else {
		root.SetErr(writer)
		command.SetErr(nil)
	}
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

// definitionDeleteLogConfig selects the real plain/structured handler, optionally at DEBUG.
func definitionDeleteLogConfig(t *testing.T, url, format string, debug bool) string {
	t.Helper()
	cfg := writeTestConfigForVersion(t, url, "8.10")
	level := "info"
	if debug {
		level = "debug"
	}
	f, err := os.OpenFile(cfg, os.O_WRONLY|os.O_APPEND, 0600)
	require.NoError(t, err)
	_, err = fmt.Fprintf(f, "log:\n  level: %s\n  format: %s\n", level, format)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return cfg
}

// TestDeleteProcessDefinitionUnfinishedCommand verifies blockers and previews
// through actual v810 union requests, retaining mode precedence and zero mutations.
func TestDeleteProcessDefinitionUnfinishedCommand(t *testing.T) {
	for _, population := range []struct {
		name                     string
		active, suspended, union int
	}{
		{"suspended", 0, 3, 3}, {"mixed", 2, 3, 5}, {"transition", 0, 0, 1},
	} {
		for _, flags := range [][]string{nil, {"--json"}, {"--quiet"}, {"--verbose"}, {"--dry-run"}, {"--dry-run", "--json"}, {"--dry-run", "--quiet", "--json"}} {
			t.Run(population.name+"/"+strings.Join(flags, " "), func(t *testing.T) {
				var reads, mutations atomic.Int32
				srv := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					switch {
					case r.URL.Path == "/v2/process-definitions/123":
						io.WriteString(w, opsAllProcessDefinitionsPurgeDefinitionJSON("123", "invoice", 1))
					case r.URL.Path == "/v2/process-instances/search":
						reads.Add(1)
						var body struct {
							Filter map[string]any `json:"filter"`
						}
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						require.Equal(t, "123", body.Filter["processDefinitionKey"])
						count := 0
						switch state := body.Filter["state"].(type) {
						case string:
							if state == "ACTIVE" {
								count = population.active
							}
							if state == "SUSPENDED" {
								count = population.suspended
							}
						case map[string]any:
							require.Equal(t, []any{"ACTIVE", "SUSPENDED"}, state["$in"])
							count = population.union
						}
						fmt.Fprintf(w, `{"items":[],"page":{"totalItems":%d,"hasMoreTotalItems":false}}`, count)
					default:
						mutations.Add(1)
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
						w.WriteHeader(500)
					}
				}))
				t.Cleanup(srv.Close)
				args := []string{"--config", definitionDeleteLogConfig(t, srv.URL, "plain-time", false), "delete", "process-definition", "--key", "123", "--no-wait"}
				args = append(args, flags...)
				stdout, stderr, err := runDefinitionDeleteTerminal(t, args, "configured", nil)
				dry := strings.Contains(strings.Join(flags, " "), "--dry-run")
				machine := strings.Contains(strings.Join(flags, " "), "--json")
				if dry {
					require.NoError(t, err, stderr)
				} else {
					var exitErr *exec.ExitError
					require.ErrorAs(t, err, &exitErr)
					require.Equal(t, 1, exitErr.ExitCode())
				}
				require.Zero(t, mutations.Load())
				require.Equal(t, int32(6), reads.Load())
				require.NotContains(t, stderr, "Proceed with")
				records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
				if machine {
					decoder := json.NewDecoder(strings.NewReader(stdout))
					var envelope map[string]any
					require.NoError(t, decoder.Decode(&envelope))
					require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
					if dry {
						require.Equal(t, "succeeded", envelope["outcome"])
						payload := requireJSONObject(t, envelope["payload"])
						totals := requireJSONObject(t, payload["totals"])
						require.Equal(t, float64(population.union), totals["unfinishedProcessInstances"])
						require.Equal(t, float64(population.active), totals["activeProcessInstances"])
						require.Equal(t, true, totals["suspendedCountSupported"])
						require.Equal(t, false, payload["mutationSubmitted"])
					} else {
						require.Equal(t, "local_precondition", envelope["class"])
					}
					require.Empty(t, records)
				} else {
					require.Empty(t, stdout)
					errors := []string{}
					for _, record := range records {
						if record[0] == "ERROR" {
							errors = append(errors, record[1])
						}
					}
					if dry {
						require.Empty(t, errors)
					} else {
						require.Len(t, errors, 1)
						require.Contains(t, errors[0], fmt.Sprintf("%d unfinished process instance(s) (active %d, suspended %d) block deletion", population.union, population.active, population.suspended))
					}
					if strings.Contains(strings.Join(flags, " "), "--quiet") {
						require.Len(t, records, 1)
					}
				}
			})
		}
	}
}

// TestDeleteProcessDefinitionUnfinishedDiagnosticLogging catches direct stderr
// bypasses, duplicate errors, raw HTTP disclosure and dropped partial results.
func TestDeleteProcessDefinitionUnfinishedDiagnosticLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for i, tc := range []struct {
			name                                                   string
			flags                                                  []string
			configDebug, debug, machine, quiet, partial, preflight bool
		}{
			{name: "normal"}, {name: "verbose", flags: []string{"--verbose"}}, {name: "quiet", flags: []string{"--quiet"}, quiet: true},
			{name: "debug", flags: []string{"--debug"}, debug: true}, {name: "configured debug", configDebug: true, debug: true},
			{name: "quiet overrides debug", flags: []string{"--quiet", "--debug"}, configDebug: true, quiet: true},
			{name: "JSON", flags: []string{"--json"}, machine: true}, {name: "partial JSON", flags: []string{"--json", "--quiet"}, machine: true, quiet: true, partial: true},
			{name: "partial verbose", flags: []string{"--verbose"}, partial: true}, {name: "preflight", preflight: true},
		} {
			t.Run(format+"/"+tc.name, func(t *testing.T) {
				var mutations atomic.Int32
				srv := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if r.URL.Path == "/v2/process-instances/search" && !tc.preflight {
						io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
						return
					}
					if strings.HasPrefix(r.URL.Path, "/v2/process-definitions/") {
						io.WriteString(w, opsAllProcessDefinitionsPurgeDefinitionJSON(strings.TrimPrefix(r.URL.Path, "/v2/process-definitions/"), "invoice", 1))
						return
					}
					if strings.HasPrefix(r.URL.Path, "/v2/resources/") && strings.HasSuffix(r.URL.Path, "/deletion") {
						mutations.Add(1)
						if tc.partial && strings.Contains(r.URL.Path, "/123/") {
							io.WriteString(w, `{"resourceKey":"123","batchOperation":null}`)
							return
						}
					} else if !tc.preflight {
						t.Errorf("unexpected request %s", r.URL.Path)
					}
					w.WriteHeader(400)
					io.WriteString(w, `{"detail":"POST https://private.invalid/secret body-only-secret resource-secret-456 token=password"}`)
				}))
				t.Cleanup(srv.Close)
				args := []string{"--config", definitionDeleteLogConfig(t, srv.URL, format, tc.configDebug), "delete", "process-definition", "--key", "123,456,789", "--no-wait", "--auto-confirm", "--workers", "1"}
				args = append(args, tc.flags...)
				destination := "inherited"
				if i%2 == 1 {
					destination = "configured"
				}
				stdout, stderr, err := runDefinitionDeleteTerminal(t, args, destination, nil)
				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr)
				require.Equal(t, 2, exitErr.ExitCode())
				records := bulkCompletionDiagnosticRecords(t, stderr, format)
				errors, debugs := []string{}, []string{}
				for _, record := range records {
					switch record[0] {
					case "ERROR":
						errors = append(errors, record[1])
					case "DEBUG":
						debugs = append(debugs, record[1])
					case "INFO":
						require.False(t, tc.quiet)
					case "WARN":
						require.False(t, tc.quiet)
					default:
						t.Errorf("unexpected severity %s", record[0])
					}
				}
				if tc.machine {
					require.Empty(t, errors)
					decoder := json.NewDecoder(strings.NewReader(stdout))
					var envelope map[string]any
					require.NoError(t, decoder.Decode(&envelope))
					require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
					require.Equal(t, "invalid_input", envelope["class"])
					require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "body-only-secret")
					payload := requireJSONObject(t, envelope["payload"])
					items := payload["items"].([]any)
					require.Len(t, items, 3)
					ok := 0
					for _, item := range items {
						if requireJSONObject(t, item)["ok"] == true {
							ok++
						}
					}
					want := 0
					if tc.partial {
						want = 1
					}
					require.Equal(t, want, ok)
				} else {
					require.Empty(t, stdout)
					require.Len(t, errors, 1)
					stage := "deleting process definitions"
					success := 0
					if tc.partial {
						success = 1
					}
					if tc.preflight {
						stage = "checking delete impact"
					}
					require.Equal(t, fmt.Sprintf("%s: selected 3, accepted %d; outcome unconfirmed; invalid input", stage, success), errors[0])
				}
				if tc.preflight {
					require.Zero(t, mutations.Load())
				} else {
					require.Equal(t, int32(3), mutations.Load())
				}
				for _, bad := range []string{"body-only-secret", "private.invalid", "resource-secret-456", "token=password"} {
					require.NotContains(t, stderr, bad)
				}
				if tc.debug {
					require.NotEmpty(t, debugs)
					require.Contains(t, strings.Join(debugs, "\n"), "POST /v2/resources/")
					require.Contains(t, strings.Join(debugs, "\n"), "status=400")
				} else {
					require.Empty(t, debugs)
					for _, bad := range []string{srv.URL, "POST", "/v2/resources/"} {
						require.NotContains(t, stderr, bad)
					}
				}
			})
		}
	}
}

// TestDeleteProcessDefinitionUnfinishedEmptyWorkloadControls preserves explicit-key
// deletion and confirmation when the instance workload is empty; previews and
// bypasses remain prompt-free/read-free where their existing controls require it.
func TestDeleteProcessDefinitionUnfinishedEmptyWorkloadControls(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		flags                      []string
		dry, bypass, abort, prompt bool
	}{
		{name: "preview", flags: []string{"--dry-run", "--force"}, dry: true},
		{name: "preview JSON", flags: []string{"--dry-run", "--force", "--json", "--no-wait"}, dry: true},
		{name: "preview keys", flags: []string{"--dry-run", "--keys-only"}, dry: true},
		{name: "preview quiet keys", flags: []string{"--dry-run", "--quiet", "--keys-only"}, dry: true},
		{name: "preview quiet", flags: []string{"--dry-run", "--quiet"}, dry: true},
		{name: "preview quiet JSON", flags: []string{"--dry-run", "--quiet", "--json"}, dry: true},
		{name: "skipped preview", flags: []string{"--dry-run", "--force", "--no-state-check", "--json"}, dry: true, bypass: true},
		{name: "accepted", flags: []string{"--auto-confirm", "--no-wait"}},
		{name: "quiet accepted", flags: []string{"--auto-confirm", "--no-wait", "--quiet"}},
		{name: "keys accepted", flags: []string{"--auto-confirm", "--no-wait", "--keys-only"}},
		{name: "quiet keys accepted", flags: []string{"--auto-confirm", "--no-wait", "--quiet", "--keys-only"}},
		{name: "confirmed JSON", flags: []string{"--auto-confirm", "--json"}},
		{name: "automation", flags: []string{"--automation", "--json", "--no-wait"}},
		{name: "quiet JSON", flags: []string{"--auto-confirm", "--quiet", "--json", "--no-wait"}},
		{name: "confirm", flags: []string{"--no-wait", "--json"}, prompt: true},
		{name: "abort", flags: []string{"--no-wait", "--json"}, prompt: true, abort: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var reads, mutations atomic.Int32
			srv := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/process-definitions/123":
					if mutations.Load() > 0 {
						w.WriteHeader(404)
						io.WriteString(w, `{"detail":"gone"}`)
						return
					}
					require.False(t, tc.bypass)
					io.WriteString(w, opsAllProcessDefinitionsPurgeDefinitionJSON("123", "invoice", 1))
				case "/v2/process-instances/search":
					reads.Add(1)
					var body struct {
						Page struct {
							Limit int `json:"limit"`
						} `json:"page"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.Equal(t, 1, body.Page.Limit, "zero workload must not start force discovery")
					io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
				case "/v2/batch-operations/search":
					io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
				case "/v2/resources/123/deletion":
					mutations.Add(1)
					io.WriteString(w, `{"resourceKey":"123","batchOperation":null}`)
				default:
					t.Errorf("unexpected request %s", r.URL.Path)
					w.WriteHeader(500)
				}
			}))
			t.Cleanup(srv.Close)
			args := []string{"--config", definitionDeleteLogConfig(t, srv.URL, "plain-time", false), "delete", "process-definition", "--key", "123"}
			args = append(args, tc.flags...)
			var exchanges []testx.CmdTerminalExchange
			if tc.prompt {
				answer := "y\n"
				if tc.abort {
					answer = "n\n"
				}
				exchanges = []testx.CmdTerminalExchange{{Prompt: confirmTerminalPrompt, Response: answer}}
			}
			stdout, stderr, err := runDefinitionDeleteTerminal(t, args, "inherited", exchanges)
			if tc.abort {
				require.Error(t, err)
			} else {
				require.NoError(t, err, stderr)
			}
			if tc.dry || tc.abort {
				require.Zero(t, mutations.Load())
			} else {
				require.Equal(t, int32(1), mutations.Load())
			}
			wantedReads := int32(6)
			if tc.bypass {
				wantedReads = 0
			} else if !tc.dry && !tc.abort {
				wantedReads = 12
			}
			require.Equal(t, wantedReads, reads.Load())
			if tc.prompt {
				require.Equal(t, 1, strings.Count(stderr, confirmTerminalPrompt))
			} else {
				require.NotContains(t, stderr, confirmTerminalPrompt)
			}
			joined := strings.Join(tc.flags, " ")
			if strings.Contains(joined, "--json") {
				decoder := json.NewDecoder(strings.NewReader(stdout))
				var envelope map[string]any
				require.NoError(t, decoder.Decode(&envelope))
				require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
				if tc.dry {
					require.Equal(t, "succeeded", envelope["outcome"])
					payload := requireJSONObject(t, envelope["payload"])
					require.Equal(t, false, payload["mutationSubmitted"])
					totals := requireJSONObject(t, payload["totals"])
					if tc.bypass {
						require.NotContains(t, totals, "suspendedCountSupported")
					} else {
						require.Equal(t, true, totals["suspendedCountSupported"])
					}
					require.NotContains(t, totals, "unfinishedProcessInstances")
				}
				require.NotContains(t, stdout, "Proceed with")
			} else {
				require.Empty(t, stdout)
				records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
				if strings.Contains(joined, "--quiet") || strings.Contains(joined, "--keys-only") {
					require.Empty(t, records)
				} else {
					messages := []string{}
					for _, r := range records {
						messages = append(messages, r[1])
					}
					want := []string{"selection scope: explicit resource keys; tenant filter not applied", "affected tenants: tenant"}
					if tc.dry {
						want = append(want, "dry run: delete process-definition")
					}
					want = append(want, "delete impact check: 1 process definition(s); no unfinished process instances found; no changes made yet", "deletion is irreversible: process-definition resources and associated history will be removed.")
					if !tc.dry {
						want = append(want, "pd delete accepted; requested 1, ok 1, failed 0")
					}
					require.Equal(t, want, messages, "exact empty-workload human transcript")
				}
			}
		})
	}
}

// TestDeleteProcessDefinitionSuspendedForceExecution exercises the full CLI,
// facade and v810 adapters against an isolated sparse union discovery fixture.
func TestDeleteProcessDefinitionSuspendedForceExecution(t *testing.T) {
	for _, scenario := range []string{"accepted", "preview", "cancel failure", "drain failure", "history failure"} {
		t.Run(scenario, func(t *testing.T) {
			var canceled, history, definitions atomic.Int32
			var events testx.SafeSlice[string]
			srv := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				state := "SUSPENDED"
				if canceled.Load() > 0 {
					state = "TERMINATED"
				}
				row := opsAllProcessDefinitionsPurgeProcessInstanceJSON("321", "321", "123", state)
				switch r.URL.Path {
				case "/v2/process-definitions/123":
					io.WriteString(w, opsAllProcessDefinitionsPurgeDefinitionJSON("123", "invoice", 1))
				case "/v2/process-instances/321":
					io.WriteString(w, row)
				case "/v2/process-instances/search":
					var body struct {
						Filter map[string]any `json:"filter"`
						Page   struct {
							Limit int    `json:"limit"`
							After string `json:"after"`
							From  int    `json:"from"`
						} `json:"page"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					if _, ok := body.Filter["parentProcessInstanceKey"]; ok {
						io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
						return
					}
					require.Equal(t, "123", body.Filter["processDefinitionKey"])
					if body.Page.Limit == 1 {
						count := 0
						if union, ok := body.Filter["state"].(map[string]any); ok {
							require.Equal(t, []any{"ACTIVE", "SUSPENDED"}, union["$in"])
							if canceled.Load() > 0 {
								events.Append("verify union")
								if scenario == "drain failure" {
									w.WriteHeader(400)
									io.WriteString(w, `{"detail":"POST https://private.invalid/secret body-only-secret"}`)
									return
								}
							} else {
								count = 1
							}
						} else if body.Filter["state"] == "SUSPENDED" && canceled.Load() == 0 {
							count = 1
						}
						fmt.Fprintf(w, `{"items":[],"page":{"totalItems":%d}}`, count)
						return
					}
					union, ok := body.Filter["state"].(map[string]any)
					require.True(t, ok)
					require.Equal(t, []any{"ACTIVE", "SUSPENDED"}, union["$in"])
					if body.Page.After == "" {
						io.WriteString(w, `{"items":[],"page":{"totalItems":1,"endCursor":"sparse"}}`)
						return
					}
					require.Equal(t, "sparse", body.Page.After)
					require.Zero(t, body.Page.From)
					fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":1}}`, row)
				case "/v2/process-instances/321/cancellation":
					events.Append("cancel")
					canceled.Add(1)
					if scenario == "cancel failure" {
						w.WriteHeader(400)
						io.WriteString(w, `{"detail":"POST https://private.invalid/secret body-only-secret"}`)
						return
					}
					w.WriteHeader(202)
					io.WriteString(w, `{}`)
				case "/v2/process-instances/321/deletion":
					events.Append("history")
					history.Add(1)
					if scenario == "history failure" {
						w.WriteHeader(400)
						io.WriteString(w, `{"detail":"POST https://private.invalid/secret body-only-secret"}`)
						return
					}
					w.WriteHeader(200)
					io.WriteString(w, `{}`)
				case "/v2/resources/123/deletion":
					events.Append("definition")
					definitions.Add(1)
					io.WriteString(w, `{"resourceKey":"123","batchOperation":null}`)
				default:
					t.Errorf("unexpected request (including resume or child cancellation): %s %s", r.Method, r.URL.Path)
					w.WriteHeader(500)
				}
			}))
			t.Cleanup(srv.Close)
			args := []string{"--config", definitionDeleteLogConfig(t, srv.URL, "plain-time", false), "delete", "process-definition", "--key", "123", "--force", "--no-wait", "--auto-confirm", "--json"}
			if scenario == "preview" {
				args = append(args, "--dry-run")
			}
			stdout, stderr, err := runDefinitionDeleteTerminal(t, args, "configured", nil)
			if strings.HasSuffix(scenario, "failure") {
				require.Error(t, err)
				require.Zero(t, definitions.Load())
			} else {
				require.NoError(t, err, stderr)
			}
			decoder := json.NewDecoder(strings.NewReader(stdout))
			var envelope map[string]any
			require.NoError(t, decoder.Decode(&envelope))
			require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
			require.NotContains(t, stderr, "body-only-secret")
			require.NotContains(t, stderr, "private.invalid")
			require.NotContains(t, stderr, "POST")
			if scenario == "preview" {
				require.Equal(t, "succeeded", envelope["outcome"])
				require.Zero(t, canceled.Load())
				require.Zero(t, history.Load())
				require.Zero(t, definitions.Load())
				payload := requireJSONObject(t, envelope["payload"])
				plan := requireJSONObject(t, payload["plan"])
				item := requireJSONObject(t, plan["items"].([]any)[0])
				require.Equal(t, []any{"321"}, item["unfinishedProcessInstanceKeys"])
				require.NotContains(t, item, "activeProcessInstanceKeys")
			} else {
				require.Equal(t, int32(1), canceled.Load())
				switch scenario {
				case "accepted":
					require.Equal(t, "accepted", envelope["outcome"])
					require.Equal(t, []string{"cancel", "verify union", "history", "definition"}, events.Snapshot())
				case "cancel failure", "drain failure":
					require.Zero(t, history.Load())
				case "history failure":
					require.Equal(t, int32(1), history.Load())
				}
			}
		})
	}
}
