// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestUpdateUserTaskAssignmentCommandOutput checks exact results and envelope EOF
// through real execution, including partial/empty keys and quiet precedence.
func TestUpdateUserTaskAssignmentCommandOutput(t *testing.T) {
	for _, operation := range []string{"assign", "unassign"} {
		t.Run(operation, func(t *testing.T) {
			label := assignmentOperationLabel(operation)

			for _, phase := range []string{"preview", "accepted", "partial", "rejected"} {
				for _, flags := range [][]string{nil, {"--verbose"}, {"--quiet"}, {"--json"}, {"--keys-only"}, {"--quiet", "--json"}, {"--quiet", "--keys-only"}, {"--json", "--keys-only"}} {
					t.Run(phase+"/"+strings.Join(flags, " "), func(t *testing.T) {
						server, calls := newUserTaskAssignmentServer(t, "8.10", operation, func(w http.ResponseWriter, r *http.Request, key string) {
							if phase == "rejected" || (phase == "partial" && key == completionKeyB) {
								w.Header().Set("Content-Type", "application/problem+json")
								w.WriteHeader(409)
								fmt.Fprint(w, `{"detail":"body-only-secret"}`)
								return
							}
							w.WriteHeader(204)
						})
						args := append([]string{"--log-format", "plain-time", "--auto-confirm", "update", "ut", assignmentModeFlag(operation), "--key", completionKeyA + "," + completionKeyB, "--workers", "1"}, flags...)
						if phase == "preview" {
							args = append(args, "--dry-run")
						}
						stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), "", args...)
						accepted, failed, code := 2, 0, 0
						switch phase {
						case "preview":
							accepted = 0
						case "partial":
							accepted, failed, code = 1, 1, 6
						case "rejected":
							accepted, failed, code = 0, 2, 6
						}
						assertUserTaskCompletionExit(t, err, code)
						if phase == "preview" {
							require.Empty(t, calls.Snapshot())
						} else {
							require.Len(t, calls.Snapshot(), 2)
						}
						jsonMode := slices.Contains(flags, "--json")
						if jsonMode {
							envelope := decodeUserTaskCompletionEnvelope(t, stdout)
							outcome := "accepted"
							if phase == "preview" {
								outcome = "succeeded"
							} else if failed > 0 {
								outcome = "failed"
							}
							require.Equal(t, outcome, envelope["outcome"])
							payload := requireJSONObject(t, envelope["payload"])
							require.Equal(t, operation, payload["operation"])
							wantPhase := "execution"
							preview := 0
							if phase == "preview" {
								wantPhase = "preview"
								preview = 2
							}
							require.Equal(t, wantPhase, payload["phase"])
							require.Equal(t, map[string]any{"preview": float64(preview), "accepted": float64(accepted), "failed": float64(failed), "timedOut": float64(0), "canceled": float64(0), "notAttempted": float64(0)}, payload["counts"])
							if operation == "assign" {
								require.Equal(t, "alice", payload["assignee"])
								require.Equal(t, false, payload["allowOverride"])
							} else {
								require.NotContains(t, payload, "assignee")
								require.NotContains(t, payload, "allowOverride")
							}
							delete(payload, "assignee")
							delete(payload, "allowOverride")
							if phase == "accepted" {
								// Human clarification must not change the established machine payload.
								require.Equal(t, map[string]any{
									"operation": operation, "phase": "execution",
									"requestedKeys": []any{completionKeyA, completionKeyB}, "requestedCount": float64(2),

									"items": []any{
										map[string]any{"key": completionKeyA, "status": "accepted", "submission": "accepted", "statusCode": float64(204)},
										map[string]any{"key": completionKeyB, "status": "accepted", "submission": "accepted", "statusCode": float64(204)},
									},
									"counts": map[string]any{"preview": float64(0), "accepted": float64(2), "failed": float64(0), "timedOut": float64(0), "canceled": float64(0), "notAttempted": float64(0)},
								}, payload)
							}
							items := requireJSONItems(t, payload["items"], 2)
							for i, key := range []string{completionKeyA, completionKeyB} {
								row := requireJSONObject(t, items[i])
								require.Equal(t, key, row["key"])
								status, submission := "accepted", "accepted"
								if phase == "preview" {
									status, submission = "preview", "not_submitted"
								} else if phase == "rejected" || (phase == "partial" && i == 1) {
									status, submission = "failed", "rejected"
									require.Contains(t, row["error"], "body-only-secret")
								}
								require.Equal(t, status, row["status"])
								require.Equal(t, submission, row["submission"])
							}
							if failed > 0 {
								require.Equal(t, "conflict", envelope["class"])
								require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "body-only-secret")
							}
							require.Empty(t, stderr)
						} else {
							want := ""
							if slices.Contains(flags, "--keys-only") {
								if phase == "preview" || accepted > 0 {
									want = completionKeyA + "\n"
								}
								if phase == "preview" || accepted == 2 {
									want += completionKeyB + "\n"
								}
							} else if !slices.Contains(flags, "--quiet") {
								if phase == "preview" {
									want = "dry run: update user-task " + label + ": 2 requested"
									if operation == "assign" {
										want += " to \"alice\" (reassignment: not allowed)"
									}
									want += "; no requests submitted\nselected user-task " + completionKeyA + "\nselected user-task " + completionKeyB + "\n"
								} else {
									counts := map[string]string{"accepted": "2 confirmed", "partial": "1 confirmed, 1 failed", "rejected": "2 failed"}
									want = "update user-task " + label + ": 2 requested; " + counts[phase]
									want += "\n"
								}
								if slices.Contains(flags, "--verbose") {
									for i, key := range []string{completionKeyA, completionKeyB} {
										status, submission, suffix := "accepted", "accepted", "; HTTP 204 No Content"
										if phase == "preview" {
											status, submission, suffix = "preview", "not_submitted", ""
										} else if phase == "rejected" || (phase == "partial" && i == 1) {
											status, submission, suffix = "failed", "rejected", "; HTTP 409 Conflict"
										}
										if status == "accepted" {
											want += "updated user-task " + key + ": " + label + " confirmed" + suffix + "\n"
										} else {
											want += fmt.Sprintf("%s: %s; submission: %s%s\n", key, status, submission, suffix)
										}
									}
								}
							}
							wantRecords := [][2]string{}
							if slices.Contains(flags, "--keys-only") {
								require.Equal(t, want, stdout)
							} else {
								require.Empty(t, stdout)
								if phase == "preview" && slices.Contains(flags, "--verbose") && !slices.Contains(flags, "--quiet") {
									wantRecords = append(wantRecords, [2]string{"INFO", "selection scope: explicit resource keys; tenant filter not applied"})
								}
								wantRecords = append(wantRecords, completionInfoRecords(want)...)
							}
							if failed > 0 {
								wantRecords = append(wantRecords, [2]string{"ERROR", fmt.Sprintf("user-task %s: conflict (HTTP 409): %d", label, failed)})
							}
							require.Equal(t, wantRecords, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
							require.NotContains(t, stdout+stderr, "body-only-secret")
						}
					})
				}
			}
		})
	}
}

// TestUpdateUserTaskAssignmentCommandFailureClasses preserves native rejection,
// unknown submission and no-error-code behavior without fabricating acceptance.
func TestUpdateUserTaskAssignmentCommandFailureClasses(t *testing.T) {
	for _, operation := range []string{"assign", "unassign"} {
		t.Run(operation, func(t *testing.T) {

			for _, tc := range []struct {
				status, code      int
				class, submission string
			}{
				{400, 2, "invalid_input", "rejected"}, {401, 1, "local_precondition", "rejected"}, {403, 1, "local_precondition", "rejected"}, {404, 3, "not_found", "rejected"}, {409, 6, "conflict", "rejected"}, {500, 1, "internal", "unknown"}, {504, 4, "timeout", "unknown"},
			} {
				t.Run(fmt.Sprint(tc.status), func(t *testing.T) {
					for _, override := range []bool{false, true} {
						server, calls := newUserTaskAssignmentServer(t, "8.10", operation, func(w http.ResponseWriter, r *http.Request, key string) {
							w.Header().Set("Content-Type", "application/problem+json")
							w.WriteHeader(tc.status)
							fmt.Fprint(w, `{"detail":"distinctive backend failure"}`)
						})
						args := []string{"--json", "--auto-confirm", "update", "ut", assignmentModeFlag(operation), "--key", completionKeyA}
						if override {
							args = append(args, "--no-err-codes")
						}
						stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), "", args...)
						code := tc.code
						if override {
							code = 0
						}
						assertUserTaskCompletionExit(t, err, code)
						require.Empty(t, stderr)
						envelope := decodeUserTaskCompletionEnvelope(t, stdout)
						require.Equal(t, tc.class, envelope["class"])
						row := requireJSONObject(t, requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 1)[0])
						require.Equal(t, tc.submission, row["submission"])
						status := "failed"
						if tc.status == 504 {
							status = "timed_out"
						}
						require.Equal(t, status, row["status"])
						require.Equal(t, float64(tc.status), row["statusCode"])
						require.Len(t, calls.Snapshot(), 1)
					}
				})
			}
		})
	}
}

// TestUpdateUserTaskAssignmentCommandRenderFailure keeps one rendering attempt,
// safe configured logging and the original mutation failure code on write errors.
func TestUpdateUserTaskAssignmentCommandRenderFailure(t *testing.T) {
	for _, operation := range []string{"assign", "unassign"} {
		t.Run(operation, func(t *testing.T) {
			label := assignmentOperationLabel(operation)

			for _, phase := range []string{"preview", "accepted", "rejected"} {
				t.Run(phase, func(t *testing.T) {
					server, calls := newUserTaskAssignmentServer(t, "8.10", operation, func(w http.ResponseWriter, r *http.Request, key string) {
						if phase == "rejected" {
							w.Header().Set("Content-Type", "application/problem+json")
							w.WriteHeader(409)
							fmt.Fprint(w, `{"detail":"body-only-secret"}`)
							return
						}
						w.WriteHeader(204)
					})
					args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.10"), "--log-format", "plain-time", "--auto-confirm", "--json", "update", "ut", assignmentModeFlag(operation), "--key", completionKeyA}
					if phase == "preview" {
						args = append(args, "--dry-run")
					}
					encoded, err := json.Marshal(args)
					require.NoError(t, err)
					stdout, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, "TestUpdateUserTaskCompletionRenderFailureProcess", "", map[string]string{"C8VOLT_COMPLETION_ARGS": string(encoded)}, "")
					code := 1
					if phase == "rejected" {
						code = 6
					}
					assertUserTaskCompletionExit(t, err, code)
					require.Equal(t, "partial-result\n", stdout)
					message := "user-task " + label + ": could not write result"
					if phase == "preview" {
						message = "user-task " + label + ": could not write preview"
						require.Empty(t, calls.Snapshot())
					} else {
						require.Len(t, calls.Snapshot(), 1)
					}
					require.Equal(t, [][2]string{{"ERROR", message}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
				})
			}
		})
	}
}
