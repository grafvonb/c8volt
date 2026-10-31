// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"net/http"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestUpdateUserTaskCompletionRhythm exercises the full operator sequence with real terminal stdin.
func TestUpdateUserTaskCompletionRhythm(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, scenario := range []string{"assigned", "unassigned", "lookup failed", "declined", "variables", "empty variables", "fail fast", "preview", "preview variables", "quiet", "keys", "auto confirm", "automation", "json", "warn level", "error level"} {
				t.Run(format+"/"+destination+"/"+scenario, func(t *testing.T) {
					server, calls, reads := newUserTaskCompletionPreflightServer(t, "8.10", func(w http.ResponseWriter, r *http.Request, key string) {
						if scenario == "lookup failed" {
							w.WriteHeader(503)
							return
						}
						state, assignee := "CREATED", "alice"
						if scenario == "unassigned" {
							assignee = ""
						}
						if scenario == "fail fast" && key == completionKeyA {
							state = "COMPLETED"
						}
						writeCompletionTask(w, key, state, assignee)
					}, nil)
					keys := completionKeyA
					if scenario == "fail fast" {
						keys += "," + completionKeyB
					}
					level := "info"
					if scenario == "warn level" {
						level = "warn"
					}
					if scenario == "error level" {
						level = "error"
					}
					infoSuppressed := level != "info"
					args := []string{"--config", completionLogConfig(t, server.URL, format, level), "--log-format", format, "update", "ut", "--complete", "-k", keys}
					preview := strings.HasPrefix(scenario, "preview")
					if preview {
						args = append(args, "--dry-run")
					}
					withVars := strings.Contains(scenario, "variables")
					if withVars {
						vars := `{"password":"rhythm-secret"}`
						if scenario == "empty variables" {
							vars = `{}`
						}
						args = append(args, "--vars", vars)
					}
					switch scenario {
					case "fail fast":
						args = append(args, "--fail-fast")
					case "quiet":
						args = append(args, "--quiet")
					case "keys":
						args = append(args, "--keys-only")
					case "auto confirm":
						args = append(args, "--auto-confirm")
					case "automation":
						args = append(args, "--automation")
					case "json":
						args = append(args, "--json", "--auto-confirm")
					}
					prompted := !preview && scenario != "fail fast" && scenario != "auto confirm" && scenario != "automation" && scenario != "json"
					prompt := "You are about to update user-task " + completionKeyA + ".\nDo you want to proceed? [y/N]: "
					var exchanges []testx.CmdTerminalExchange
					if prompted {
						response := "yes"
						if scenario == "declined" {
							response = "no"
						}
						exchanges = []testx.CmdTerminalExchange{{Prompt: prompt, Response: response}}
					}
					stdout, stderr, err := runUserTaskCompletionTerminal(t, args, destination, exchanges)
					code := 0
					if scenario == "declined" {
						code = 1
					}
					if scenario == "fail fast" {
						code = 6
					}
					assertUserTaskCompletionExit(t, err, code)
					wantReads, wantPosts := 1, 1
					if preview {
						wantReads, wantPosts = 0, 0
					}
					if scenario == "declined" {
						wantPosts = 0
					}
					if scenario == "fail fast" {
						wantReads, wantPosts = 2, 0
					}
					require.Len(t, reads.Snapshot(), wantReads)
					require.Len(t, calls.Snapshot(), wantPosts)
					require.NotContains(t, stdout, "Do you want to proceed?")
					if prompted {
						require.Equal(t, 1, strings.Count(stderr, prompt))
					} else {
						require.NotContains(t, stderr, "Do you want to proceed?")
					}
					diagnostics := strings.Replace(stderr, prompt, "", 1)
					records := bulkCompletionDiagnosticRecords(t, diagnostics, format)
					wantRecords := [][2]string{}
					showPlan := prompted && scenario != "quiet" && scenario != "keys" && !infoSuppressed
					if showPlan || preview || scenario == "fail fast" {
						wantRecords = append(wantRecords, [2]string{"INFO", "selection scope: explicit resource keys; tenant filter not applied"})
					}
					plan := "plan: update user-task " + completionKeyA + ": completion: submit"
					if withVars {
						plan += "; variables: submit"
					}
					if scenario == "fail fast" {
						plan = "plan: update user-task completion: 2 requested; 1 already completed; fail-fast: no completion requests will be sent"
					}
					if showPlan || scenario == "fail fast" {
						wantRecords = append(wantRecords, [2]string{"INFO", plan})
					}
					warning := ""
					if scenario == "unassigned" {
						warning = "user-task completion: task has no assignee"
					}
					if scenario == "lookup failed" {
						warning = "user-task completion: could not check state or assignee for 1 user task; backend validation will determine eligibility"
					}
					if warning != "" {
						wantRecords = append(wantRecords, [2]string{"WARN", warning})
					}
					if scenario == "declined" {
						wantRecords = append(wantRecords, [2]string{"ERROR", "local precondition failed: aborted by user"})
					}
					result := ""
					if scenario != "quiet" && scenario != "declined" && scenario != "keys" && scenario != "json" && !infoSuppressed {
						result = "updated user-task " + completionKeyA + ": completion confirmed"
						if scenario == "fail fast" {
							result = "update user-task: 2 requested; 1 failed, 1 not attempted"
						}
						if preview {
							result = "dry run: update user-task " + completionKeyA + ": completion: submit"
							if withVars {
								result += "; variables: submit"
							}
							result += "; no requests submitted"
						}
						wantRecords = append(wantRecords, [2]string{"INFO", result})
					}
					if scenario == "fail fast" {
						wantRecords = append(wantRecords, [2]string{"ERROR", "user-task completion: task is already completed; completion request not sent"})
					}
					require.Equal(t, wantRecords, records)
					if showPlan {
						require.Less(t, strings.Index(stderr, "selection scope:"), strings.Index(stderr, plan))
						require.Less(t, strings.Index(stderr, plan), strings.Index(stderr, prompt))
					}
					if warning != "" {
						require.Less(t, strings.Index(stderr, plan), strings.Index(stderr, warning))
						require.Less(t, strings.Index(stderr, warning), strings.Index(stderr, prompt))
					}
					switch {
					case scenario == "json":
						envelope := decodeUserTaskCompletionEnvelope(t, stdout)
						require.Equal(t, "accepted", envelope["outcome"])
						payload := requireJSONObject(t, envelope["payload"])
						require.Len(t, payload, 8)
						require.Equal(t, "accepted", requireJSONObject(t, requireJSONItems(t, payload["items"], 1)[0])["status"])
						require.Len(t, requireJSONObject(t, payload["counts"]), 6)
					case scenario == "quiet" || scenario == "declined":
						require.Empty(t, stdout)
					case scenario == "keys":
						require.Equal(t, completionKeyA+"\n", stdout)
					default:
						require.Empty(t, stdout)
						if prompted && result != "" {
							require.Less(t, strings.Index(stderr, prompt), strings.Index(stderr, result))
						}
					}
					require.NotContains(t, stdout+stderr, "rhythm-secret")
					require.NotContains(t, stdout+stderr, "affected tenants:", "no tenant evidence may be invented")
					require.NotContains(t, stdout, "selection scope:")
					require.NotContains(t, stdout, "plan:")
				})
			}
		}
	}
}
