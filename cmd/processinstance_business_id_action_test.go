// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const actionSeedA = "2251799813685101"
const actionSeedB = "2251799813685102"
const actionChild = "2251799813685103"

// businessIDActionFixture owns discovery, variable scopes, state observations and exact writes.
type businessIDActionFixture struct {
	verb     string
	empty    bool
	failure  string
	searches atomic.Int32
	approved atomic.Bool
	requests testx.SafeSlice[string]
	writes   testx.SafeSlice[string]
	children testx.SafeSlice[string]
}

// newBusinessIDActionFixture exposes changed post-approval discovery so accidental reselection fails.
func newBusinessIDActionFixture(t *testing.T, verb string, empty bool, failure string) (*businessIDActionFixture, *httptest.Server) {
	t.Helper()
	f := &businessIDActionFixture{verb: verb, empty: empty, failure: failure}
	item := func(key string) map[string]any {
		state := "ACTIVE"
		if verb == "resume" {
			state = "SUSPENDED"
		}
		if slices.Contains(f.writes.Snapshot(), key) && failure != "wait" {
			if verb == "suspend" {
				state = "SUSPENDED"
			}
			if verb == "resume" {
				state = "ACTIVE"
			}
		}
		if failure == "ineligible" {
			state = "COMPLETED"
		}
		row := map[string]any{"processInstanceKey": key, "processDefinitionKey": "9001", "processDefinitionId": "order", "state": state, "tenantId": "tenant-a", "businessId": "observed"}
		if key == actionChild {
			row["parentProcessInstanceKey"] = actionSeedA
		}
		return row
	}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, "8.10") {
			return
		}
		f.requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fail := func() {
			w.Header().Set("Set-Cookie", "secret-cookie")
			w.WriteHeader(400)
			fmt.Fprint(w, `{"detail":"POST https://private.invalid/path?token=secret-token body-only-secret resource-key-secret"}`)
		}
		switch {
		case r.URL.Path == "/v2/process-instances/search":
			var body struct {
				Filter map[string]any `json:"filter"`
				Page   map[string]any `json:"page"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			rows := []map[string]any{}
			page := map[string]any{"totalItems": 0, "hasMoreTotalItems": false}
			if bid, ok := body.Filter["businessId"]; ok {
				require.Equal(t, " ORDER,零* ", bid)
				require.Equal(t, "tenant-a", body.Filter["tenantId"])
				require.Empty(t, f.writes.Snapshot())
				require.False(t, f.approved.Load(), "BID rediscovered after approval")
				n := f.searches.Add(1)
				if !empty {
					switch n {
					case 1:
						rows = append(rows, item(actionSeedA), item(actionSeedA))
						page = map[string]any{"totalItems": 2, "hasMoreTotalItems": true, "endCursor": "second"}
					case 2:
						if failure == "search" {
							fail()
							return
						}
						page = map[string]any{"totalItems": 2, "hasMoreTotalItems": true, "endCursor": "third"}
					default:
						rows = append(rows, item(actionSeedB))
						page = map[string]any{"totalItems": 2, "hasMoreTotalItems": false}
					}
				}
			} else {
				require.NotContains(t, body.Filter, "tenantId")
				parent, _ := body.Filter["parentProcessInstanceKey"].(string)
				f.children.Append(parent)
				require.NotEmpty(t, parent)
				if failure == "children" {
					fail()
					return
				}
				if parent == actionSeedA {
					rows = append(rows, item(actionChild))
					page["totalItems"] = 1
				}
				if f.approved.Load() {
					t.Error("descendants rediscovered after approval")
				}
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": rows, "page": page}))
		case r.URL.Path == "/v2/variables/search":
			var body struct {
				Filter map[string]any `json:"filter"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.NotContains(t, body.Filter, "tenantId")
			key, _ := body.Filter["processInstanceKey"].(string)
			rows := []map[string]any{}
			if (slices.Contains(f.writes.Snapshot(), key) && failure != "wait") || failure == "unchanged" {
				rows = append(rows, map[string]any{"name": "foo", "value": `{"nested":["bar"]}`, "variableKey": "9001", "processInstanceKey": key, "scopeKey": key, "tenantId": "tenant-a"})
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": rows, "page": map[string]any{"totalItems": len(rows), "hasMoreTotalItems": false}}))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			require.NoError(t, json.NewEncoder(w).Encode(item(key)))
		case strings.HasSuffix(r.URL.Path, "/variables") || strings.HasSuffix(r.URL.Path, "/suspension") || strings.HasSuffix(r.URL.Path, "/resumption"):
			parts := strings.Split(r.URL.Path, "/")
			key := parts[len(parts)-2]
			if verb == "update" {
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, true, body["local"])
				require.Equal(t, map[string]any{"foo": map[string]any{"nested": []any{"bar"}}}, body["variables"])
			}
			f.writes.Append(key)
			if failure == "write" || (failure == "partial" && key == actionSeedB) {
				fail()
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	t.Cleanup(server.Close)
	return f, server
}

// businessIDActionArgs supplies a test-owned configuration and one opaque native criterion.
func businessIDActionArgs(t *testing.T, s *httptest.Server, verb, format, level string, flags ...string) []string {
	t.Helper()
	args := []string{"--config", runBusinessIDConfig(t, s.URL, format, level), "--tenant", "tenant-a", verb, "pi", "--bid", " ORDER,零* ", "--workers", "1"}
	if verb == "update" {
		args = append(args, "--set-vars", `{"foo":{"nested":["bar"]}}`)
	}
	return append(args, flags...)
}

// TestBusinessIDActionsScopes proves complete duplicate/sparse selection, exact versus descendant scope and selected result provenance.
func TestBusinessIDActionsScopes(t *testing.T) {
	for _, verb := range []string{"update", "suspend", "resume"} {
		for _, tree := range []bool{false, true} {
			if verb == "update" && tree {
				continue
			}
			for _, dry := range []bool{false, true} {
				t.Run(fmt.Sprint(verb, tree, dry), func(t *testing.T) {
					f, s := newBusinessIDActionFixture(t, verb, false, "")
					flags := []string{"--json", "--auto-confirm"}
					if tree {
						flags = append(flags, "--tree")
					}
					if dry {
						flags = append(flags, "--dry-run")
					}
					out, stderr, err := runBusinessIDCommand(t, businessIDActionArgs(t, s, verb, "plain-time", "info", flags...), "configured", false)
					require.NoError(t, err, out+stderr)
					require.Empty(t, stderr)
					envelope := decodeBusinessIDCancelEnvelope(t, out)
					require.Equal(t, "succeeded", envelope["outcome"])
					payload := envelope["payload"].(map[string]any)
					scope := payload
					if tree {
						scope = payload["plan"].(map[string]any)
					}
					selection := scope["selection"].(map[string]any)
					require.Equal(t, []string{actionSeedA, actionSeedB}, interfaceStrings(t, selection["keys"]))
					require.Contains(t, out, ` ORDER,零* `)
					require.Equal(t, int32(3), f.searches.Load())
					want := []string{actionSeedA, actionSeedB}
					if tree {
						want = []string{actionSeedA, actionChild, actionSeedB}
					}
					if dry {
						require.Empty(t, f.writes.Snapshot())
					} else {
						require.Equal(t, want, f.writes.Snapshot())
					}
					if !tree {
						if dry && verb != "update" {
							require.Len(t, f.requests.Snapshot(), 3)
						}
						require.Empty(t, f.children.Snapshot())
					}
				})
			}
		}
	}
}

// TestBusinessIDActionsEmpty verifies prompt-free real-terminal empty completion and exact machine payload conventions.
func TestBusinessIDActionsEmpty(t *testing.T) {
	for _, verb := range []string{"update", "suspend", "resume"} {
		for _, tree := range []bool{false, true} {
			if verb == "update" && tree {
				continue
			}
			for _, dry := range []bool{false, true} {
				for _, mode := range [][]string{nil, {"--quiet"}, {"--keys-only"}, {"--quiet", "--keys-only"}, {"--json", "--auto-confirm"}, {"--quiet", "--json", "--automation"}} {
					t.Run(fmt.Sprint(verb, tree, dry, mode), func(t *testing.T) {
						f, s := newBusinessIDActionFixture(t, verb, true, "")
						flags := append(slices.Clone(mode), "--no-wait")
						if tree {
							flags = append(flags, "--tree")
						}
						if dry {
							flags = append(flags, "--dry-run")
						}
						out, stderr, err := runBusinessIDCommand(t, businessIDActionArgs(t, s, verb, "plain-time", "info", flags...), "inherited", false)
						require.NoError(t, err, out+stderr)
						require.Equal(t, int32(1), f.searches.Load())
						require.Len(t, f.requests.Snapshot(), 1)
						require.Empty(t, f.writes.Snapshot())
						require.NotContains(t, stderr, "proceed")
						if containsArg(flags, "--json") {
							e := decodeBusinessIDCancelEnvelope(t, out)
							require.Equal(t, "succeeded", e["outcome"])
							require.Empty(t, stderr)
							if verb != "update" {
								require.Empty(t, e["payload"].(map[string]any)["items"])
							}
						} else {
							require.Empty(t, out)
							records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
							if containsArg(flags, "--quiet") || containsArg(flags, "--keys-only") {
								require.Empty(t, records)
							} else {
								require.Equal(t, businessIDActionEmptyRecords(verb, tree, dry), records)
							}
						}
					})
				}
			}
		}
	}
}

// TestBusinessIDActionsLogging rejects direct stream writes, raw joined failures and duplicate envelopes across each action.
func TestBusinessIDActionsLogging(t *testing.T) {
	for _, verb := range []string{"update", "suspend", "resume"} {
		for _, tree := range []bool{false, true} {
			if verb == "update" && tree {
				continue
			}
			for _, format := range []string{"plain-time", "json"} {
				for _, destination := range []string{"configured", "inherited"} {
					for _, mode := range []string{"normal", "verbose", "quiet", "warn", "debug-flag", "debug-config", "json", "dry-run"} {
						for _, failure := range []string{"", "write", "partial"} {
							if mode == "dry-run" && failure != "" {
								continue
							}
							t.Run(strings.Join([]string{verb, fmt.Sprint(tree), format, destination, mode, failure}, "/"), func(t *testing.T) {
								f, s := newBusinessIDActionFixture(t, verb, false, failure)
								level := "info"
								if mode == "warn" {
									level = "warn"
								}
								if mode == "debug-config" {
									level = "debug"
								}
								flags := []string{"--auto-confirm", "--no-wait"}
								if tree {
									flags = append(flags, "--tree")
								}
								switch mode {
								case "verbose", "quiet", "json", "dry-run":
									flags = append(flags, "--"+mode)
								case "debug-flag":
									flags = append(flags, "--debug")
								}
								out, stderr, err := runBusinessIDCommand(t, businessIDActionArgs(t, s, verb, format, level, flags...), destination, false)
								if failure == "" {
									require.NoError(t, err, out+stderr)
								} else {
									assertRunBusinessIDExit(t, err, 2)
								}
								want := []string{actionSeedA, actionSeedB}
								if tree {
									want = []string{actionSeedA, actionChild, actionSeedB}
								}
								if mode == "dry-run" {
									require.Empty(t, f.writes.Snapshot())
								} else {
									require.Equal(t, want, f.writes.Snapshot())
								}
								if mode == "json" {
									require.Empty(t, stderr)
									e := decodeBusinessIDCancelEnvelope(t, out)
									p := e["payload"].(map[string]any)
									require.Len(t, p["items"], len(want))
									if failure != "" {
										require.Equal(t, "invalid_input", e["class"])
										require.Contains(t, out, "body-only-secret")
									} else {
										require.Equal(t, "accepted", e["outcome"])
									}
									return
								}
								require.Empty(t, out)
								records := bulkCompletionDiagnosticRecords(t, stderr, format)
								var infos, errs, debugs []string
								for _, r := range records {
									switch r[0] {
									case "INFO":
										infos = append(infos, r[1])
									case "ERROR":
										errs = append(errs, r[1])
									case "DEBUG":
										debugs = append(debugs, r[1])
									default:
										t.Fatalf("unexpected severity %s", r[0])
									}
								}
								if failure == "" {
									require.Empty(t, errs)
								} else {
									require.Len(t, errs, 1)
									if verb == "update" {
										require.Contains(t, errs[0], "failed")
									} else {
										require.Contains(t, errs[0], "rejected:")
									}
								}
								if mode == "quiet" || mode == "warn" {
									require.Empty(t, infos)
								} else {
									require.NotEmpty(t, infos)
								}
								if strings.HasPrefix(mode, "debug-") {
									require.NotEmpty(t, debugs)
									require.Contains(t, stderr, "/v2/process-instances/search")
									require.NotContains(t, stderr, "secret-cookie")
								} else {
									require.Empty(t, debugs)
									for _, raw := range []string{"POST", "https://private.invalid", "secret-token", "body-only-secret", "resource-key-secret"} {
										require.NotContains(t, stderr, raw)
									}
								}
							})
						}
					}
				}
			}
		}
	}
}

// TestBusinessIDActionsConfirmation uses real terminal stdin and ensures approval cannot introduce new targets.
func TestBusinessIDActionsConfirmation(t *testing.T) {
	for _, verb := range []string{"update", "suspend", "resume"} {
		for _, tree := range []bool{false, true} {
			if verb == "update" && tree {
				continue
			}
			for _, destination := range []string{"configured", "inherited"} {
				for _, answer := range []string{"y", "n", "eof"} {
					t.Run(fmt.Sprint(verb, tree, destination, answer), func(t *testing.T) {
						f, s := newBusinessIDActionFixture(t, verb, false, "")
						count := 2
						if tree {
							count = 3
						}
						prompt := fmt.Sprintf("You are about to %s %d process instance(s).\nDo you want to proceed? [y/N]: ", verb, count)
						if verb == "update" {
							prompt = "You are about to update 2 requested variable value(s) on 2 process instance(s).\nDo you want to proceed? [y/N]: "
						}
						exchange := testx.CmdTerminalExchange{Prompt: prompt, Response: answer, EndOfInput: answer == "eof", BeforeResponse: func(out, stderr string) {
							require.Empty(t, out)
							require.Empty(t, f.writes.Snapshot())
							require.Equal(t, int32(3), f.searches.Load())
							f.approved.Store(true)
						}}
						flags := []string{"--no-wait"}
						if tree {
							flags = append(flags, "--tree")
						}
						out, stderr, err := runBusinessIDCommand(t, businessIDActionArgs(t, s, verb, "plain-time", "info", flags...), destination, false, exchange)
						require.Empty(t, out)
						require.Contains(t, stderr, prompt)
						require.NotContains(t, stderr, "INFO "+prompt)
						if answer == "y" {
							require.NoError(t, err, stderr)
							require.Len(t, f.writes.Snapshot(), count)
						} else {
							require.Error(t, err)
							require.Empty(t, f.writes.Snapshot())
						}
						require.Equal(t, int32(3), f.searches.Load())
					})
				}
			}
		}
	}
}

// TestBusinessIDActionsFailures retains incomplete selection and original failure exits without any mutation.
func TestBusinessIDActionsFailures(t *testing.T) {
	for _, verb := range []string{"update", "suspend", "resume"} {
		for _, failure := range []string{"search", "children", "partial", "wait"} {
			if verb == "update" && failure == "children" {
				continue
			}
			t.Run(verb+"/"+failure, func(t *testing.T) {
				f, s := newBusinessIDActionFixture(t, verb, false, failure)
				flags := []string{"--json", "--auto-confirm", "--backoff-timeout", "20ms"}
				if failure == "children" {
					flags = append(flags, "--tree")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDActionArgs(t, s, verb, "plain-time", "info", flags...), "configured", false)
				require.Error(t, err)
				require.Empty(t, stderr)
				e := decodeBusinessIDCancelEnvelope(t, out)
				require.NotEqual(t, "succeeded", e["outcome"])
				if failure == "search" || failure == "children" {
					require.Empty(t, f.writes.Snapshot())
				} else {
					require.Len(t, f.writes.Snapshot(), 2)
				}
				if failure == "search" {
					require.Contains(t, out, `"completion": "incomplete"`)
					require.Contains(t, out, actionSeedA)
				}
			})
		}
	}
}

// TestBusinessIDActionsValidation rejects old-version and exclusive selectors before backend calls.
func TestBusinessIDActionsValidation(t *testing.T) {
	for _, verb := range []string{"update", "suspend", "resume"} {
		for _, version := range []string{"8.8", "8.9", "8.10"} {
			for _, extra := range [][]string{{}, {"--key", actionSeedA}, {"-"}, {"--bid", ""}, {"--limit", "-1"}} {
				if version == "8.10" && len(extra) == 0 {
					continue
				}
				t.Run(fmt.Sprint(verb, version, extra), func(t *testing.T) {
					calls := atomic.Int32{}
					s := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if serveVersionCheckTopology(w, r, version) {
							return
						}
						calls.Add(1)
						w.WriteHeader(500)
					}))
					defer s.Close()
					args := []string{"--config", testx.WriteTestConfigForVersion(t, s.URL, version), verb, "pi", "--bid", "order", "--json", "--dry-run"}
					if verb == "update" {
						args = append(args, "--set-vars", `{"foo":1}`)
					}
					args = append(args, extra...)
					out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
					require.Error(t, err)
					require.Empty(t, stderr)
					decodeBusinessIDCancelEnvelope(t, out)
					require.Zero(t, calls.Load())
				})
			}
		}
	}
}

// TestBusinessIDActionsLimitAndWriteFailure checks seed bounds, descendant expansion and one failed envelope attempt.
func TestBusinessIDActionsLimitAndWriteFailure(t *testing.T) {
	for _, verb := range []string{"update", "suspend", "resume"} {
		for _, tree := range []bool{false, true} {
			if verb == "update" && tree {
				continue
			}
			for _, broken := range []bool{false, true} {
				t.Run(fmt.Sprint(verb, tree, broken), func(t *testing.T) {
					f, s := newBusinessIDActionFixture(t, verb, false, "")
					flags := []string{"--json", "--auto-confirm", "--no-wait", "--limit", "1"}
					if tree {
						flags = append(flags, "--tree")
					}
					out, stderr, err := runBusinessIDCommand(t, businessIDActionArgs(t, s, verb, "plain-time", "info", flags...), "configured", broken)
					if broken {
						require.Error(t, err)
						require.Equal(t, "attempt\n", out)
					} else {
						require.NoError(t, err, out+stderr)
						require.Empty(t, stderr)
						decodeBusinessIDCancelEnvelope(t, out)
						require.Contains(t, out, "limit_reached")
					}
					require.Equal(t, int32(1), f.searches.Load())
					want := []string{actionSeedA}
					if tree {
						want = append(want, actionChild)
					}
					require.Equal(t, want, f.writes.Snapshot())
				})
			}
		}
	}
}

// TestBusinessIDActionsNoChangeAndKeys retains unchanged/root and empty-eligible no-ops plus explicit machine results under automation.
func TestBusinessIDActionsNoChangeAndKeys(t *testing.T) {
	for _, verb := range []string{"update", "suspend", "resume"} {
		for _, noChange := range []bool{false, true} {
			for _, dry := range []bool{false, true} {
				t.Run(fmt.Sprint(verb, noChange, dry), func(t *testing.T) {
					failure := ""
					if noChange {
						failure = "ineligible"
						if verb == "update" {
							failure = "unchanged"
						}
					}
					f, s := newBusinessIDActionFixture(t, verb, false, failure)
					flags := []string{"--no-wait", "--keys-only", "--quiet", "--automation"}
					if verb != "update" {
						flags = append(flags, "--tree")
					}
					if dry {
						flags = append(flags, "--dry-run")
					}
					out, stderr, err := runBusinessIDCommand(t, businessIDActionArgs(t, s, verb, "plain-time", "info", flags...), "configured", false)
					require.NoError(t, err, out+stderr)
					require.Empty(t, stderr)
					want := []string{actionSeedA, actionSeedB}
					if verb != "update" {
						want = []string{actionSeedA, actionChild, actionSeedB}
					}
					if noChange && !(dry && verb == "update") {
						require.Empty(t, out)
					} else {
						require.Equal(t, strings.Join(want, "\n")+"\n", out)
					}
					if noChange || dry {
						require.Empty(t, f.writes.Snapshot())
					} else {
						require.Equal(t, want, f.writes.Snapshot())
					}
				})
			}
		}
	}
}

// TestBusinessIDActionsFailFast preserves skipped slots and one original-error envelope after the first failed target.
func TestBusinessIDActionsFailFast(t *testing.T) {
	for _, verb := range []string{"update", "suspend", "resume"} {
		t.Run(verb, func(t *testing.T) {
			f, s := newBusinessIDActionFixture(t, verb, false, "write")
			out, stderr, err := runBusinessIDCommand(t, businessIDActionArgs(t, s, verb, "json", "info", "--json", "--auto-confirm", "--no-wait", "--fail-fast"), "configured", false)
			assertRunBusinessIDExit(t, err, 2)
			require.Empty(t, stderr)
			e := decodeBusinessIDCancelEnvelope(t, out)
			require.Len(t, e["payload"].(map[string]any)["items"], 2)
			require.Equal(t, []string{actionSeedA}, f.writes.Snapshot())
		})
	}
}

// TestBusinessIDActionsKeysConfirmation keeps a variable-update approval plain while stdout contains only accepted keys.
func TestBusinessIDActionsKeysConfirmation(t *testing.T) {
	f, s := newBusinessIDActionFixture(t, "update", false, "")
	prompt := "You are about to update 2 requested variable value(s) on 2 process instance(s).\nDo you want to proceed? [y/N]: "
	exchange := testx.CmdTerminalExchange{Prompt: prompt, Response: "y", BeforeResponse: func(out, stderr string) { require.Empty(t, out); require.Empty(t, f.writes.Snapshot()) }}
	out, stderr, err := runBusinessIDCommand(t, businessIDActionArgs(t, s, "update", "plain-time", "info", "--keys-only", "--no-wait"), "configured", false, exchange)
	require.NoError(t, err, out+stderr)
	require.Equal(t, actionSeedA+"\n"+actionSeedB+"\n", out)
	require.Equal(t, prompt, stderr)
}

// businessIDActionEmptyRecords pins existing human no-op wording independently of production renderers.
func businessIDActionEmptyRecords(verb string, tree, dry bool) [][2]string {
	messages := []string{}
	if verb == "update" {
		label, status := "plan", "no confirmation required"
		if dry {
			label, status = "dry run", "no changes applied"
		}
		messages = append(messages, "configured tenant: none", `--tenant "tenant-a" sets the tenant filter`, "selection scope: tenant-a only", fmt.Sprintf("%s: update process-instance variables: nothing to update (0 requested value(s) already match visible variables); %s", label, status))
	} else if tree {
		if dry {
			messages = append(messages, "dry run: "+verb+" process-instance tree")
		}
		if verb == "suspend" {
			messages = append(messages, "selected process instances: 0; process instances in scope: 0; active targets: 0; already suspended: 0; excluded: 0")
		} else {
			messages = append(messages, "selected process instances: 0; process instances in scope: 0; suspended targets: 0; excluded: 0")
		}
		switch {
		case dry:
			messages = append(messages, "outcome: preview; no changes applied")
		case verb == "suspend":
			messages = append(messages, "suspension: no active process instances to suspend; no changes applied")
		default:
			messages = append(messages, "resumption: no suspended process instances to resume; no changes applied")
		}
	} else if dry {
		messages = append(messages, "dry run: "+verb+" process-instance; selected: 0; eligibility not checked; no changes applied")
	} else {
		name := "suspension"
		if verb == "resume" {
			name = "resumption"
		}
		messages = append(messages, name+": no process instances selected; no changes applied")
	}
	records := make([][2]string, len(messages))
	for i, message := range messages {
		records[i] = [2]string{"INFO", message}
	}
	return records
}
