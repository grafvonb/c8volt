// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestUpdateUserTaskCompletionPreflightModes preserves schema, ordering and conflict exits across output modes.
func TestUpdateUserTaskCompletionPreflightModes(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, scenario := range []string{"completed", "mixed", "unassigned", "lookup failed", "stale", "fail fast"} {
			for _, mode := range []string{"human", "verbose", "quiet", "json", "keys", "quiet json", "quiet keys"} {
				if version != "8.10" && (mode != "json" || (scenario != "completed" && scenario != "unassigned")) {
					continue
				}
				t.Run(version+"/"+scenario+"/"+mode, func(t *testing.T) {
					server, calls, reads := newUserTaskCompletionPreflightServer(t, version, func(w http.ResponseWriter, r *http.Request, key string) {
						if scenario == "lookup failed" {
							w.Header().Set("Content-Type", "application/problem+json")
							w.WriteHeader(403)
							fmt.Fprintf(w, `{"detail":"GET https://private.invalid body=private-body resource=%s"}`, key)
							return
						}
						state, assignee := "CREATED", "alice"
						if scenario == "completed" || ((scenario == "mixed" || scenario == "fail fast") && key == completionKeyA) {
							state = "COMPLETED"
						}
						if scenario == "unassigned" {
							assignee = ""
						}
						writeCompletionTask(w, key, state, assignee)
					}, func(w http.ResponseWriter, r *http.Request, key string) {
						if scenario == "stale" {
							w.WriteHeader(409)
						} else {
							w.WriteHeader(204)
						}
					})
					args := []string{"--auto-confirm", "--log-format", "plain-time", "update", "ut", "--complete", "-k", completionKeyA + "," + completionKeyB + "," + completionKeyA, "--vars", `{"approved":true}`}
					if scenario == "fail fast" {
						args = append(args, "--fail-fast")
					}
					if strings.Contains(mode, "quiet") {
						args = append(args, "--quiet")
					}
					jsonMode, keysMode := strings.Contains(mode, "json"), strings.Contains(mode, "keys")
					if jsonMode {
						args = append(args, "--json")
					}
					if keysMode {
						args = append(args, "--keys-only")
					}
					if mode == "verbose" {
						args = append(args, "--verbose")
					}
					stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, version), "", args...)
					conflict := scenario == "completed" || scenario == "mixed" || scenario == "stale" || scenario == "fail fast"
					code := 0
					if conflict {
						code = 6
					}
					assertUserTaskCompletionExit(t, err, code)
					require.ElementsMatch(t, []string{completionKeyA, completionKeyB}, reads.Snapshot())
					wantCalls := 2
					if scenario == "completed" || scenario == "fail fast" {
						wantCalls = 0
					}
					if scenario == "mixed" {
						wantCalls = 1
					}
					require.Len(t, calls.Snapshot(), wantCalls)
					if scenario == "mixed" {
						require.Equal(t, completionKeyB, calls.Snapshot()[0].Key)
					}
					records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
					warnings, errors := []string{}, []string{}
					info := completionInfoText(records)
					for _, record := range records {
						if record[0] == "INFO" {
							continue
						} else if record[0] == "WARN" {
							warnings = append(warnings, record[1])
						} else {
							require.Equal(t, "ERROR", record[0])
							errors = append(errors, record[1])
						}
					}
					if strings.Contains(mode, "quiet") || (scenario != "unassigned" && scenario != "lookup failed") {
						require.Empty(t, warnings)
					} else if scenario == "unassigned" {
						require.Equal(t, []string{"user-task completion: 2 eligible tasks have no assignee"}, warnings)
					} else {
						require.Equal(t, []string{"user-task completion: could not check state or assignee for 2 user tasks; backend validation will determine eligibility"}, warnings)
					}
					if conflict && !jsonMode {
						require.Len(t, errors, 1)
					} else {
						require.Empty(t, errors)
					}
					if jsonMode {
						require.Empty(t, info)
						envelope := decodeUserTaskCompletionEnvelope(t, stdout)
						outcome := "accepted"
						if conflict {
							outcome = "failed"
							require.Equal(t, "conflict", envelope["class"])
						}
						require.Equal(t, outcome, envelope["outcome"])
						payload := requireJSONObject(t, envelope["payload"])
						require.Len(t, payload, 8)
						require.Equal(t, []any{completionKeyA, completionKeyB}, payload["requestedKeys"])
						rows := requireJSONItems(t, payload["items"], 2)
						for i, row := range rows {
							item := requireJSONObject(t, row)
							skipped := scenario == "completed" || ((scenario == "mixed" || scenario == "fail fast") && i == 0)
							if skipped {
								require.Equal(t, "failed", item["status"])
								require.Equal(t, "not_submitted", item["submission"])
								require.NotContains(t, item, "statusCode")
								require.Contains(t, item["error"], "already completed")
								require.Contains(t, item["error"], "variables not applied")
							} else if scenario == "fail fast" {
								require.Equal(t, "not_attempted", item["status"])
							} else if scenario == "stale" {
								require.Equal(t, "rejected", item["submission"])
								require.Equal(t, float64(409), item["statusCode"])
							} else {
								require.Equal(t, "accepted", item["status"])
							}
						}
					} else if keysMode {
						require.Empty(t, info)
						want := ""
						if !conflict {
							want = completionKeyA + "\n" + completionKeyB + "\n"
						} else if scenario == "mixed" {
							want = completionKeyB + "\n"
						}
						require.Equal(t, want, stdout)
					} else if mode == "quiet" {
						require.Empty(t, info)
						require.Empty(t, stdout)
					} else {
						require.Empty(t, stdout)
						require.Contains(t, info, "update user-task: 2 requested;")
						if scenario == "completed" {
							require.Contains(t, info, "2 failed")
							require.Contains(t, stderr, "2 tasks are already completed; their completion requests were not sent; supplied variables were not applied")
						}
					}
					if !jsonMode {
						require.NotContains(t, stdout, server.URL)
					}
					require.NotContains(t, stderr, server.URL)
					for _, secret := range []string{"private.invalid", "private-body", "GET "} {
						require.NotContains(t, stdout+stderr, secret)
					}
				})
			}
		}
	}
}

// TestUpdateUserTaskCompletionPreflightTerminal proves warning-before-prompt ordering and prompt-free conflicts.
func TestUpdateUserTaskCompletionPreflightTerminal(t *testing.T) {
	for _, destination := range []string{"configured", "inherited"} {
		for _, format := range []string{"plain-time", "json"} {
			for _, scenario := range []string{"completed", "mixed accept", "mixed decline", "unassigned accept", "unassigned decline", "preview", "invalid"} {
				t.Run(destination+"/"+format+"/"+scenario, func(t *testing.T) {
					server, calls, reads := newUserTaskCompletionPreflightServer(t, "8.10", func(w http.ResponseWriter, r *http.Request, key string) {
						state, assignee := "CREATED", "alice"
						if scenario == "completed" || (strings.HasPrefix(scenario, "mixed") && key == completionKeyA) {
							state = "COMPLETED"
						}
						if strings.HasPrefix(scenario, "unassigned") {
							assignee = ""
						}
						writeCompletionTask(w, key, state, assignee)
					}, nil)
					args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.10"), "--log-format", format, "update", "ut", "--complete", "-k", completionKeyA + "," + completionKeyB}
					if scenario == "preview" {
						args = append(args, "--dry-run")
					}
					if scenario == "invalid" {
						args = append(args, "--vars", "{} {}")
					}
					prompt := ""
					var exchanges []testx.CmdTerminalExchange
					if strings.Contains(scenario, "accept") || strings.Contains(scenario, "decline") {
						n := 2
						if strings.HasPrefix(scenario, "mixed") {
							n = 1
						}
						prompt = "You are about to complete 2 user tasks.\nDo you want to proceed? [y/N]: "
						if n == 1 {
							prompt = "You are about to complete 1 user task.\nDo you want to proceed? [y/N]: "
						}
						response := "yes"
						if strings.HasSuffix(scenario, "decline") {
							response = "no"
						}
						exchanges = []testx.CmdTerminalExchange{{Prompt: prompt, Response: response}}
					}
					stdout, stderr, err := runUserTaskCompletionTerminal(t, args, destination, exchanges)
					code := 0
					if scenario == "completed" || scenario == "mixed accept" {
						code = 6
					}
					if strings.HasSuffix(scenario, "decline") {
						code = 1
					}
					if scenario == "invalid" {
						code = 2
					}
					assertUserTaskCompletionExit(t, err, code)
					if scenario == "preview" || scenario == "invalid" {
						require.Empty(t, reads.Snapshot())
					} else {
						require.ElementsMatch(t, []string{completionKeyA, completionKeyB}, reads.Snapshot())
					}
					wantCalls := 0
					if scenario == "mixed accept" {
						wantCalls = 1
					}
					if scenario == "unassigned accept" {
						wantCalls = 2
					}
					require.Len(t, calls.Snapshot(), wantCalls)
					diagnostics := stderr
					if prompt != "" {
						require.Equal(t, 1, strings.Count(stderr, prompt))
						diagnostics = strings.Replace(stderr, prompt, "", 1)
					} else {
						require.NotContains(t, stderr, "Do you want to proceed?")
					}
					require.Empty(t, stdout)
					records := bulkCompletionDiagnosticRecords(t, diagnostics, format)
					if prompt != "" || scenario == "preview" {
						require.Equal(t, [2]string{"INFO", "selection scope: explicit resource keys; tenant filter not applied"}, records[0])
						records = records[1:]
					}
					if prompt != "" {
						want := "plan: update user-task completion: 2 requested; 2 eligible"
						if strings.HasPrefix(scenario, "mixed") {
							want = "plan: update user-task completion: 2 requested; 1 eligible; 1 already completed"
						}
						require.Equal(t, [2]string{"INFO", want}, records[0])
						require.Less(t, strings.Index(stderr, want), strings.Index(stderr, prompt))
						records = records[1:]
					}
					if strings.HasPrefix(scenario, "unassigned") {
						require.NotEmpty(t, records)
						require.Equal(t, [2]string{"WARN", "user-task completion: 2 eligible tasks have no assignee"}, records[0])
						require.Less(t, strings.Index(stderr, "have no assignee"), strings.Index(stderr, prompt))
					}
				})
			}
		}
	}
}

// TestUpdateUserTaskCompletionPreflightDiagnostics proves warning/error routing and safe failed-lookup diagnostics.
func TestUpdateUserTaskCompletionPreflightDiagnostics(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, mode := range []string{"normal", "verbose", "quiet", "json", "quiet json", "debug flag", "debug config"} {
				t.Run(format+"/"+destination+"/"+mode, func(t *testing.T) {
					server, calls, reads := newUserTaskCompletionPreflightServer(t, "8.10", func(w http.ResponseWriter, r *http.Request, key string) {
						if key == completionKeyA {
							writeCompletionTask(w, key, "COMPLETED", "")
							return
						}
						w.Header().Set("Content-Type", "application/problem+json")
						w.Header().Set("Set-Cookie", "session=lookup-cookie-secret")
						status := 503
						if key == completionKeyB {
							status = 404
						}
						w.WriteHeader(status)
						fmt.Fprintf(w, `{"detail":"GET https://private.invalid?token=lookup-token-secret body=lookup-body-secret resource=%s"}`, key)
					}, func(w http.ResponseWriter, r *http.Request, key string) {
						if key == completionKeyB {
							w.WriteHeader(204)
						} else {
							w.WriteHeader(409)
						}
					})
					level := "info"
					if mode == "debug config" {
						level = "debug"
					}
					args := []string{"--config", completionLogConfig(t, server.URL, format, level), "--auto-confirm", "update", "ut", "--complete", "-k", completionKeyA + "," + completionKeyB + ",2251799813685251"}
					if strings.Contains(mode, "quiet") {
						args = append(args, "--quiet")
					}
					jsonMode := strings.Contains(mode, "json")
					if jsonMode {
						args = append(args, "--json")
					}
					if mode == "verbose" {
						args = append(args, "--verbose")
					}
					if mode == "debug flag" {
						args = append(args, "--debug")
					}
					stdout, stderr, err := runUserTaskCompletionTerminal(t, args, destination, nil)
					assertUserTaskCompletionExit(t, err, 6)
					require.Len(t, reads.Snapshot(), 3)
					require.Len(t, calls.Snapshot(), 2)
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					warnings, errors, gets, posts := []string{}, []string{}, 0, 0
					info := completionInfoText(records)
					for _, record := range records {
						switch record[0] {
						case "INFO":
						case "WARN":
							warnings = append(warnings, record[1])
						case "ERROR":
							errors = append(errors, record[1])
						case "DEBUG":
							require.Contains(t, mode, "debug")
							if strings.Contains(record[1], "api #") {
								if strings.Contains(record[1], "GET /v2/topology") {
									require.Contains(t, record[1], "status=200")
									continue
								}
								if strings.Contains(record[1], "GET /v2/user-tasks/") {
									gets++
								} else {
									require.Contains(t, record[1], "POST /v2/user-tasks/")
									posts++
								}
							}
						default:
							t.Errorf("unexpected severity %s", record[0])
						}
					}
					if strings.Contains(mode, "quiet") {
						require.Empty(t, info)
						require.Empty(t, warnings)
					} else {
						require.Equal(t, []string{"user-task completion: could not check state or assignee for 2 user tasks; backend validation will determine eligibility"}, warnings)
					}
					if jsonMode {
						require.Empty(t, info)
						require.Empty(t, errors)
						envelope := decodeUserTaskCompletionEnvelope(t, stdout)
						require.Equal(t, "conflict", envelope["class"])
						require.Len(t, requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 3), 3)
					} else {
						require.Empty(t, stdout)
						if !strings.Contains(mode, "quiet") {
							require.Contains(t, info, "update user-task: 3 requested; 1 completed, 2 failed\n")
						}
						require.Equal(t, []string{"user-task completion: task is already completed; completion request not sent; conflict (HTTP 409): 1"}, errors)
					}
					if strings.Contains(mode, "debug") {
						require.Equal(t, 3, gets)
						require.Equal(t, 2, posts)
					} else {
						require.Zero(t, gets+posts)
						require.NotContains(t, stderr, "/v2/user-tasks/")
						if !jsonMode {
							require.NotContains(t, stdout, "/v2/user-tasks/")
						}
					}
					if mode == "quiet" {
						require.Empty(t, stdout)
					}
					for _, secret := range []string{"lookup-body-secret", "lookup-token-secret", "lookup-cookie-secret", "private.invalid"} {
						require.NotContains(t, stdout+stderr, secret)
					}
					require.NotContains(t, stdout, "level=")
				})
			}
		}
	}
}

// TestUpdateUserTaskCompletionConflictWording keeps human explanations concise without changing machine evidence.
func TestUpdateUserTaskCompletionConflictWording(t *testing.T) {
	for _, count := range []int{1, 2} {
		for _, source := range []string{"omitted", "{}", `{"approved":true}`} {
			for _, mode := range []string{"human", "verbose", "json"} {
				t.Run(fmt.Sprintf("%d/%s/%s", count, source, mode), func(t *testing.T) {
					server, calls, reads := newUserTaskCompletionPreflightServer(t, "8.10", func(w http.ResponseWriter, r *http.Request, key string) {
						writeCompletionTask(w, key, "COMPLETED", "alice")
					}, func(http.ResponseWriter, *http.Request, string) { t.Error("completed task must not be submitted") })
					keys := []string{completionKeyA}
					if count == 2 {
						keys = append(keys, completionKeyB)
					}
					args := []string{"--auto-confirm", "--log-format", "plain-time", "update", "ut", "--complete", "-k", strings.Join(keys, ",")}
					if source != "omitted" {
						args = append(args, "--vars", source)
					}
					if mode != "human" {
						args = append(args, "--"+mode)
					}
					stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), "", args...)
					assertUserTaskCompletionExit(t, err, 6)
					require.Len(t, reads.Snapshot(), count)
					require.Empty(t, calls.Snapshot())
					if mode == "json" {
						envelope := decodeUserTaskCompletionEnvelope(t, stdout)
						require.Equal(t, "conflict", envelope["class"])
						require.Equal(t, "failed", envelope["outcome"])
						payload := requireJSONObject(t, envelope["payload"])
						require.Len(t, payload, 8)
						require.Equal(t, source != "omitted", payload["variablesProvided"])
						require.Equal(t, map[string]any{"preview": float64(0), "accepted": float64(0), "failed": float64(count), "timedOut": float64(0), "canceled": float64(0), "notAttempted": float64(0)}, payload["counts"])
						for i, row := range requireJSONItems(t, payload["items"], count) {
							require.Equal(t, map[string]any{"key": keys[i], "status": "failed", "submission": "not_submitted", "error": "user task " + keys[i] + " is already completed; completion skipped; supplied variables not applied"}, row)
						}
						require.Empty(t, stderr)
						return
					}
					want := fmt.Sprintf("update user-task: %d requested; %d failed\n", count, count)
					individual := "task is already completed; completion request not sent"
					reason := individual
					if count == 2 {
						reason = "2 tasks are already completed; their completion requests were not sent"
					}
					if source != "omitted" {
						reason += "; supplied variables were not applied"
						individual += "; supplied variables were not applied"
					}
					if mode == "verbose" {
						for _, key := range keys {
							want += key + ": failed; submission: not_submitted; " + individual + "\n"
						}
					}
					require.Empty(t, stdout)
					wantRecords := append(completionInfoRecords(want), [2]string{"ERROR", "user-task completion: " + reason})
					require.Equal(t, wantRecords, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
					require.NotContains(t, stdout+stderr, "no successful completion response")
					if source == "omitted" {
						require.NotContains(t, stdout+stderr, "variables")
					}
				})
			}
		}
	}
}
