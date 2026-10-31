// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestUpdateUserTaskAssignmentPreflight verifies consent with real terminal stdin,
// configured/inherited output writers, and fake native reads and mutations.
func TestUpdateUserTaskAssignmentPreflight(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, mode := range []string{"replace", "opaque", "decline", "default", "EOF", "quiet", "warn", "auto", "automation", "json", "keys", "force", "force JSON", "dry-run", "unassigned", "same", "read failure", "race", "mixed", "all assigned", "fail-fast", "invalid"} {
				t.Run(format+"/"+destination+"/"+mode, func(t *testing.T) {
					calls := &testx.SafeSlice[string]{}
					currentAssignee := "bob"
					if mode == "opaque" {
						currentAssignee = "bob\n\x1b\t\""
					}
					occupied := mode != "unassigned" && mode != "same" && mode != "race"
					automatic := mode == "auto" || mode == "automation" || mode == "json"
					forced := mode == "force" || mode == "force JSON"
					bulk := mode == "mixed" || mode == "all assigned" || mode == "fail-fast"
					rejected := mode == "decline" || mode == "default" || mode == "EOF"
					prompted := !automatic && !forced && mode != "dry-run" && mode != "read failure" && mode != "invalid" && mode != "same"
					replacement := prompted && occupied
					server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if serveVersionCheckTopology(w, r, "8.10") {
							return
						}

						key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/user-tasks/"), "/assignment")
						calls.Append(r.Method + " " + key)
						w.Header().Set("Content-Type", "application/json")
						switch r.Method {
						case http.MethodGet:
							require.Equal(t, "/v2/user-tasks/"+key, r.URL.Path)
							if mode == "read failure" {
								w.WriteHeader(403)
								fmt.Fprint(w, `{"detail":"POST https://private.invalid body-secret"}`)
								return
							}
							assignee := currentAssignee
							if !occupied || ((mode == "mixed" || mode == "fail-fast") && key == completionKeyB) {
								assignee = ""
							}
							if mode == "same" {
								assignee = "alice"
							}
							require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"userTaskKey": key, "processInstanceKey": completionKeyB, "state": "CREATED", "assignee": assignee}))
						case http.MethodPost:
							require.Equal(t, "/v2/user-tasks/"+key+"/assignment", r.URL.Path)
							var body map[string]any
							require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
							require.Equal(t, "alice", body["assignee"])
							approvedKey := replacement && (key == completionKeyA || mode == "all assigned")
							require.Equal(t, forced || approvedKey, body["allowOverride"])
							if mode == "race" {
								w.WriteHeader(409)
								require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"detail": "Expected to claim user task with key '" + key + "', but it has already been assigned"}))
								return
							}
							w.WriteHeader(204)
						default:
							t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
							w.WriteHeader(400)
						}
					}))
					defer server.Close()
					level := "info"
					if mode == "warn" {
						level = "warn"
					}
					keys := completionKeyA
					if bulk {
						keys += "," + completionKeyB
					}
					args := []string{"--config", completionLogConfig(t, server.URL, format, level), "update", "ut", "--assign", "alice", "--key", keys, "--workers", "1"}
					switch mode {
					case "auto":
						args = append(args, "--auto-confirm")
					case "automation":
						args = append(args, "--automation")
					case "quiet":
						args = append(args, "--quiet")
					case "json":
						args = append(args, "--json", "--auto-confirm")
					case "keys":
						args = append(args, "--keys-only")
					case "force":
						args = append(args, "--force")
					case "force JSON":
						args = append(args, "--force", "--json")
					case "dry-run":
						args = append(args, "--dry-run")
					case "fail-fast":
						args = append(args, "--fail-fast")
					case "invalid":
						args = append(args, "--workers", "0")
					}
					prompt := "You are about to assign user-task " + completionKeyA + " to \"alice\".\nDo you want to proceed? [y/N]: "
					if replacement {
						prompt = fmt.Sprintf("Replace assignee %q with \"alice\"? [y/N]: ", currentAssignee)
					}
					if bulk {
						count := 1
						if mode == "all assigned" {
							count = 2
						}
						prompt = fmt.Sprintf("Assign 2 user tasks to \"alice\", replacing %d existing assignee(s)? [y/N]: ", count)
					}
					var exchanges []testx.CmdTerminalExchange
					if prompted {
						response := "y"
						if mode == "decline" {
							response = "n"
						}
						if mode == "default" {
							response = ""
						}
						exchanges = []testx.CmdTerminalExchange{{Prompt: prompt, Response: response, EndOfInput: mode == "EOF"}}
					}
					stdout, stderr, err := runUserTaskCompletionTerminal(t, args, destination, exchanges)
					code := 0
					if automatic || mode == "race" {
						code = 6
					}
					if rejected || mode == "read failure" {
						code = 1
					}
					if mode == "invalid" {
						code = 2
					}
					assertUserTaskCompletionExit(t, err, code)
					wantCalls := []string{"GET " + completionKeyA}
					if bulk {
						wantCalls = append(wantCalls, "GET "+completionKeyB)
					}
					if forced || mode == "dry-run" || mode == "invalid" {
						wantCalls = nil
					}
					if (prompted && !rejected) || forced {
						wantCalls = append(wantCalls, "POST "+completionKeyA)
						if bulk {
							wantCalls = append(wantCalls, "POST "+completionKeyB)
						}
					}
					require.Equal(t, wantCalls, calls.Snapshot())
					if prompted {
						require.Equal(t, 1, strings.Count(stderr, prompt))
						stderr = strings.Replace(stderr, prompt, "", 1)
					} else {
						require.NotContains(t, stderr, "[y/N]")
					}
					if mode == "json" || mode == "force JSON" {
						envelope := decodeUserTaskCompletionEnvelope(t, stdout)
						item := requireJSONObject(t, requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 1)[0])
						if forced {
							require.Equal(t, "accepted", envelope["outcome"])
							require.Equal(t, "accepted", item["submission"])
						} else {
							require.Equal(t, "conflict", envelope["class"])
							require.Equal(t, "not_submitted", item["submission"])
							require.NotContains(t, item, "statusCode")
						}
						require.Empty(t, stderr)
						return
					}
					if mode == "keys" {
						require.Equal(t, completionKeyA+"\n", stdout)
					} else {
						require.Empty(t, stdout)
					}
					expected := [][2]string{}
					if replacement && mode != "quiet" {
						expected = append(expected, [2]string{"WARN", fmt.Sprintf("user-task %s is already assigned to %q.", completionKeyA, currentAssignee)})
						if mode == "all assigned" {
							expected = append(expected, [2]string{"WARN", "user-task " + completionKeyB + " is already assigned to \"bob\"."})
						}
					}
					switch {
					case rejected:
						expected = append(expected, [2]string{"ERROR", "local precondition failed: aborted by user"})
					case mode == "quiet" || mode == "warn" || mode == "keys":
					case automatic:
						expected = append(expected, completionInfoRecords("user-task "+completionKeyA+" is already assigned; use --force to replace the current assignee.\n")...)
					case mode == "dry-run":
						expected = completionInfoRecords("dry run: update user-task " + completionKeyA + ": assignment to \"alice\" (reassignment: not allowed); no requests submitted\n")
					case mode == "race":
						expected = completionInfoRecords("user-task " + completionKeyA + " was not assigned: reassignment is not allowed; use --force to replace the current assignee.\n")
					case mode == "read failure":
						expected = [][2]string{{"ERROR", "user-task assignment: could not check current assignees (local_precondition)"}}
					case mode == "invalid":
						expected = [][2]string{{"ERROR", "invalid input: invalid flag value: --workers must be positive integer"}}
					case mode == "same":
						expected = completionInfoRecords("user-task " + completionKeyA + " is already assigned to \"alice\"; no change needed.\n")
					case bulk:
						expected = append(expected, [2]string{"INFO", "update user-task assignment: 2 requested; 2 confirmed"})
					default:
						expected = append(expected, [2]string{"INFO", "updated user-task " + completionKeyA + ": assignment confirmed"})
					}
					require.Equal(t, expected, bulkCompletionDiagnosticRecords(t, stderr, format))
					require.NotContains(t, stderr, "private.invalid")
					require.NotContains(t, stderr, "body-secret")
				})
			}
		}
	}
}

// TestUpdateUserTaskAssignmentNonterminalRequiresForce preserves the existing command contract with advisory-aware fixtures.
func TestUpdateUserTaskAssignmentNonterminalRequiresForce(t *testing.T) {
	for _, flag := range []string{"", "--force=false", "--allow-override"} {
		t.Run(flag, func(t *testing.T) {
			calls := &testx.SafeSlice[string]{}
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveVersionCheckTopology(w, r, "8.10") {
					return
				}

				calls.Append(r.Method)
				require.Equal(t, http.MethodGet, r.Method, "nonterminal input must never imply replacement approval")
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"userTaskKey": completionKeyA, "processInstanceKey": completionKeyB, "state": "CREATED", "assignee": "bob"}))
			}))
			defer server.Close()
			args := []string{"update", "ut", "-", "--assign", "alice"}
			if flag != "" {
				args = append(args, flag)
			}
			stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), completionKeyA+"\n", args...)
			require.Error(t, err)
			require.Empty(t, stdout)
			require.NotContains(t, stderr, "[y/N]")
			if flag == "--allow-override" {
				require.Empty(t, calls.Snapshot())
				require.Contains(t, stderr, "unknown flag")
				return
			}
			assertUserTaskCompletionExit(t, err, 6)
			require.Equal(t, []string{http.MethodGet}, calls.Snapshot())
			require.Contains(t, stderr, "use --force to replace the current assignee")
		})
	}
}

// A completed ownership lookup is enough to finish a same-assignee request.
// Real terminal stdin proves that no confirmation is attempted, including when
// all keys are unchanged. Machine results must never imply a submitted mutation.
func TestUpdateUserTaskAssignmentUnchanged(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, bulk := range []bool{false, true} {
				for _, mode := range []string{"human", "verbose", "warn", "quiet", "auto-confirm", "automation", "json", "quiet json", "keys-only", "quiet keys-only", "dry-run", "force"} {
					t.Run(fmt.Sprintf("%s/%s/bulk=%t/%s", format, destination, bulk, mode), func(t *testing.T) {
						keys := []string{completionKeyA}
						if bulk {
							keys = append(keys, completionKeyB)
						}
						calls := &testx.SafeSlice[string]{}
						server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							if serveVersionCheckTopology(w, r, "8.10") {
								return
							}

							key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/user-tasks/"), "/assignment")
							calls.Append(r.Method + " " + key)
							w.Header().Set("Content-Type", "application/json")
							if mode == "force" {
								require.Equal(t, http.MethodPost, r.Method)
								require.Equal(t, "/v2/user-tasks/"+key+"/assignment", r.URL.Path)
								w.WriteHeader(http.StatusNoContent)
								return
							}
							require.Equal(t, http.MethodGet, r.Method, "same-assignee requests must not mutate")
							require.Equal(t, "/v2/user-tasks/"+key, r.URL.Path)
							require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"userTaskKey": key, "processInstanceKey": completionKeyB, "state": "CREATED", "assignee": "alice"}))
						}))
						defer server.Close()
						level := "info"
						if mode == "warn" {
							level = "warn"
						}
						args := []string{"--config", completionLogConfig(t, server.URL, format, level), "update", "ut", "--assign", "alice", "-k", strings.Join(keys, ",") + "," + keys[0], "--workers", "1"}
						for _, flag := range strings.Fields(mode) {
							if flag != "human" && flag != "warn" {
								args = append(args, "--"+flag)
							}
						}
						if strings.Contains(mode, "json") {
							args = append(args, "--auto-confirm")
						}
						stdout, stderr, err := runUserTaskCompletionTerminal(t, args, destination, nil)
						assertUserTaskCompletionExit(t, err, 0)
						require.NotContains(t, stderr, "[y/N]")
						var wantCalls []string
						for _, key := range keys {
							if mode == "force" {
								wantCalls = append(wantCalls, "POST "+key)
							} else if mode != "dry-run" {
								wantCalls = append(wantCalls, "GET "+key)
							}
						}
						require.Equal(t, wantCalls, calls.Snapshot())
						if strings.Contains(mode, "json") {
							envelope := decodeUserTaskCompletionEnvelope(t, stdout)
							require.Equal(t, "succeeded", envelope["outcome"])
							payload := requireJSONObject(t, envelope["payload"])
							require.Equal(t, float64(len(keys)), requireJSONObject(t, payload["counts"])["unchanged"])
							for i, raw := range requireJSONItems(t, payload["items"], len(keys)) {
								require.Equal(t, map[string]any{"key": keys[i], "status": "unchanged", "submission": "not_submitted"}, requireJSONObject(t, raw))
							}
							require.Empty(t, stderr)
							return
						}
						require.Empty(t, stdout)
						if mode == "warn" || strings.Contains(mode, "quiet") || strings.Contains(mode, "keys-only") {
							require.Empty(t, stderr)
							return
						}
						summary := "user-task " + completionKeyA + " is already assigned to \"alice\"; no change needed."
						if bulk {
							summary = "update user-task assignment: 2 requested; 2 unchanged"
						}
						expected := [][2]string{{"INFO", summary}}
						if mode == "verbose" && bulk {
							for _, key := range keys {
								expected = append(expected, [2]string{"INFO", key + ": unchanged; submission: not_submitted"})
							}
						}
						if mode == "force" {
							summary = "updated user-task " + completionKeyA + ": assignment confirmed"
							if bulk {
								summary = "update user-task assignment: 2 requested; 2 confirmed"
							}
							expected = [][2]string{{"INFO", summary}}
						}
						if mode == "dry-run" {
							scope := "update user-task " + completionKeyA + ": assignment"
							if bulk {
								scope = "update user-task assignment: 2 requested"
							}
							expected = [][2]string{{"INFO", "dry run: " + scope + " to \"alice\" (reassignment: not allowed); no requests submitted"}}
							if bulk {
								for _, key := range keys {
									expected = append(expected, [2]string{"INFO", "selected user-task " + key})
								}
							}
						}
						require.Equal(t, expected, bulkCompletionDiagnosticRecords(t, stderr, format))
					})
				}
			}
		}
	}
}

// TestUpdateUserTaskAssignmentMixedUnchanged preserves the existing command contract with advisory-aware fixtures.
func TestUpdateUserTaskAssignmentMixedUnchanged(t *testing.T) {
	for _, mode := range []string{"unassigned", "replace", "keys", "json", "blocked", "fail-fast"} {
		t.Run(mode, func(t *testing.T) {
			calls := &testx.SafeSlice[string]{}
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveVersionCheckTopology(w, r, "8.10") {
					return
				}

				key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/user-tasks/"), "/assignment")
				calls.Append(r.Method + " " + key)
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					assignee := "alice"
					if key == completionKeyB {
						assignee = ""
						if mode == "replace" || mode == "blocked" || mode == "fail-fast" {
							assignee = "bob"
						}
					}
					require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"userTaskKey": key, "processInstanceKey": completionKeyB, "state": "CREATED", "assignee": assignee}))
					return
				}
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/v2/user-tasks/"+completionKeyB+"/assignment", r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, mode == "replace", body["allowOverride"])
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			args := []string{"--config", completionLogConfig(t, server.URL, "json", "info"), "update", "ut", "--assign", "alice", "-k", completionKeyA + "," + completionKeyB, "--workers", "1"}
			blocked := mode == "blocked" || mode == "fail-fast"
			machine := mode == "json" || blocked
			prompt := "You are about to assign user-task " + completionKeyB + " to \"alice\".\nDo you want to proceed? [y/N]: "
			if mode == "replace" {
				prompt = "Replace assignee \"bob\" with \"alice\"? [y/N]: "
			}
			if mode == "keys" {
				args = append(args, "--keys-only")
			}
			if mode == "fail-fast" {
				args = append(args, "--fail-fast")
			}
			exchanges := []testx.CmdTerminalExchange{{Prompt: prompt, Response: "y"}}
			if machine {
				args = append(args, "--json", "--auto-confirm")
				exchanges = nil
			}
			stdout, stderr, err := runUserTaskCompletionTerminal(t, args, "inherited", exchanges)
			code := 0
			if blocked {
				code = 6
			}
			assertUserTaskCompletionExit(t, err, code)
			wantCalls := []string{"GET " + completionKeyA, "GET " + completionKeyB}
			if !blocked {
				wantCalls = append(wantCalls, "POST "+completionKeyB)
			}
			require.Equal(t, wantCalls, calls.Snapshot())
			if machine {
				envelope := decodeUserTaskCompletionEnvelope(t, stdout)
				payload := requireJSONObject(t, envelope["payload"])
				items := requireJSONItems(t, payload["items"], 2)
				require.Equal(t, map[string]any{"key": completionKeyA, "status": "unchanged", "submission": "not_submitted"}, requireJSONObject(t, items[0]))
				require.Equal(t, float64(1), requireJSONObject(t, payload["counts"])["unchanged"])
				if blocked {
					require.Equal(t, "conflict", envelope["class"])
					require.Equal(t, "not_submitted", requireJSONObject(t, items[1])["submission"])
				} else {
					require.Equal(t, "accepted", envelope["outcome"])
					require.Equal(t, "accepted", requireJSONObject(t, items[1])["submission"])
				}
				require.Empty(t, stderr)
				return
			}
			require.Equal(t, 1, strings.Count(stderr, prompt))
			stderr = strings.Replace(stderr, prompt, "", 1)
			if mode == "keys" {
				require.Equal(t, completionKeyB+"\n", stdout)
				require.Empty(t, stderr)
				return
			}
			require.Empty(t, stdout)
			expected := [][2]string{}
			if mode == "replace" {
				expected = append(expected, [2]string{"WARN", "user-task " + completionKeyB + " is already assigned to \"bob\"."})
			}
			expected = append(expected, [2]string{"INFO", "update user-task assignment: 2 requested; 1 unchanged, 1 confirmed"})
			require.Equal(t, expected, bulkCompletionDiagnosticRecords(t, stderr, "json"))
		})
	}
}
