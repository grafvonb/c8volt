// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestUpdateUserTaskAssignmentTerminal proves one plain approval on real terminal
// stdin with redirected results, both writer destinations and safe opaque identity.
func TestUpdateUserTaskAssignmentTerminal(t *testing.T) {
	for _, operation := range []string{"assign", "unassign"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, tc := range []struct {
				name, response        string
				flags                 []string
				eof, decline, invalid bool
			}{
				{name: "accept", response: " YeS "}, {name: "decline", response: "no", decline: true},
				{name: "default", decline: true}, {name: "EOF", eof: true, decline: true},
				{name: "preview", flags: []string{"--dry-run"}}, {name: "auto", flags: []string{"--auto-confirm"}},
				{name: "automation", flags: []string{"--automation"}},
				{name: "quiet machine", flags: []string{"--automation", "--quiet", "--keys-only"}},
				{name: "JSON preview", flags: []string{"--dry-run", "--json"}},
				{name: "invalid", flags: []string{"--workers", "0"}, invalid: true},
			} {
				t.Run(operation+"/"+destination+"/"+tc.name, func(t *testing.T) {
					server, calls := newUserTaskAssignmentServer(t, operation, nil)
					args := append([]string{"--config", completionLogConfig(t, server.URL, "plain-time", "info"), "update", "ut", "--key", completionKeyA + "," + completionKeyB}, assignmentFlags(operation)...)
					if operation == "assign" {
						args = append(args, "--allow-override")
					}
					args = append(args, tc.flags...)
					preview := slices.Contains(tc.flags, "--dry-run")
					prompted := !preview && !tc.invalid && !slices.Contains(tc.flags, "--auto-confirm") && !slices.Contains(tc.flags, "--automation")
					intent := "update user-task " + assignmentOperationLabel(operation) + ": 2 requested"
					prompt := "You are about to " + operation + " 2 user tasks"
					if operation == "assign" {
						intent += ` to " alice\nopaque " (reassignment: allowed)`
						prompt += ` to " alice\nopaque " (reassignment: allowed)`
					}
					prompt += ".\nDo you want to proceed? [y/N]: "
					var exchanges []testx.CmdTerminalExchange
					if prompted {
						exchanges = []testx.CmdTerminalExchange{{Prompt: prompt, Response: tc.response, EndOfInput: tc.eof}}
					}
					stdout, stderr, err := runUserTaskCompletionTerminal(t, args, destination, exchanges)
					code := 0
					if tc.decline {
						code = 1
					}
					if tc.invalid {
						code = 2
					}
					assertUserTaskCompletionExit(t, err, code)
					if tc.decline || preview || tc.invalid {
						require.Empty(t, calls.Snapshot())
					} else {
						require.Len(t, calls.Snapshot(), 2)
						if operation == "assign" {
							for _, call := range calls.Snapshot() {
								require.Equal(t, " alice\nopaque ", call.Body["assignee"])
								require.Equal(t, true, call.Body["allowOverride"])
							}
						}
					}
					require.NotContains(t, stdout, "Do you want to proceed?")
					if prompted {
						require.Equal(t, 1, strings.Count(stderr, prompt))
						stderr = strings.Replace(stderr, prompt, "", 1)
					} else {
						require.NotContains(t, stderr, "Do you want to proceed?")
					}
					records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
					if tc.invalid {
						require.Empty(t, stdout)
						require.Equal(t, [][2]string{{"ERROR", "invalid input: invalid flag value: --workers must be positive integer"}}, records)
						return
					}
					want := [][2]string{}
					if prompted {
						want = append(want, completionInfoRecords("selected user-task "+completionKeyA+"\nselected user-task "+completionKeyB+"\n")...)
					}
					if tc.decline {
						require.Empty(t, stdout)
						want = append(want, [2]string{"ERROR", "local precondition failed: aborted by user"})
					} else if slices.Contains(tc.flags, "--json") {
						require.Equal(t, "succeeded", decodeUserTaskCompletionEnvelope(t, stdout)["outcome"])
					} else if slices.Contains(tc.flags, "--keys-only") {
						require.Equal(t, completionKeyA+"\n"+completionKeyB+"\n", stdout)
					} else {
						require.Empty(t, stdout)
						message := "update user-task " + assignmentOperationLabel(operation) + ": 2 requested; 2 confirmed\n"
						if preview {
							message = "dry run: " + intent + "; no requests submitted\nselected user-task " + completionKeyA + "\nselected user-task " + completionKeyB + "\n"
						}
						want = append(want, completionInfoRecords(message)...)
					}
					require.Equal(t, want, records)
				})
			}
		}
	}
}

// TestUpdateUserTaskAssignmentSingleTerminal checks the key and protected default
// in approval, including automation/auto-confirm never enabling override.
func TestUpdateUserTaskAssignmentSingleTerminal(t *testing.T) {
	for _, operation := range []string{"assign", "unassign"} {
		for _, format := range []string{"plain-time", "json"} {
			for _, destination := range []string{"configured", "inherited"} {
				for _, mode := range []string{"normal", "verbose", "quiet", "warn"} {
					t.Run(operation+"/"+format+"/"+destination+"/"+mode, func(t *testing.T) {
						server, calls := newUserTaskAssignmentServer(t, operation, nil)
						prompt := "You are about to " + operation + " user-task " + completionKeyA
						intent := "update user-task " + completionKeyA + ": " + assignmentOperationLabel(operation)
						if operation == "assign" {
							prompt += ` to "alice" (reassignment: not allowed)`
							intent += ` to "alice" (reassignment: not allowed)`
						}
						prompt += ".\nDo you want to proceed? [y/N]: "
						level := "info"
						if mode == "warn" {
							level = "warn"
						}
						args := []string{"--config", completionLogConfig(t, server.URL, format, level), "update", "ut", assignmentModeFlag(operation), "--key", completionKeyA}
						if mode == "verbose" || mode == "quiet" {
							args = append(args, "--"+mode)
						}
						stdout, stderr, err := runUserTaskCompletionTerminal(t, args, destination, []testx.CmdTerminalExchange{{Prompt: prompt, Response: "y"}})
						require.NoError(t, err)
						require.Empty(t, stdout)
						require.Len(t, calls.Snapshot(), 1)
						require.Equal(t, 1, strings.Count(stderr, prompt))
						if operation == "assign" {
							require.Equal(t, false, calls.Snapshot()[0].Body["allowOverride"])
						}
						if mode != "verbose" {
							require.True(t, strings.HasPrefix(stderr, prompt), "prompt should be the first output: %s", stderr)
						}
						stderr = strings.Replace(stderr, prompt, "", 1)
						summary := fmt.Sprintf("updated user-task %s: %s confirmed", completionKeyA, assignmentOperationLabel(operation))
						want := [][2]string{}
						if mode == "verbose" {
							want = append(want, [2]string{"INFO", "selection scope: explicit resource keys; tenant filter not applied"}, [2]string{"INFO", "plan: " + intent})
						}
						if mode == "normal" || mode == "verbose" {
							want = append(want, [2]string{"INFO", summary})
						}
						if mode == "verbose" {
							want = append(want, [2]string{"INFO", summary + "; HTTP 204 No Content"})
						}
						require.Equal(t, want, bulkCompletionDiagnosticRecords(t, stderr, format))
					})
				}
			}
		}
	}
}
