// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSuspendProcessInstanceTreeOutput exercises preview and actual no-op through
// discovery, preserving exact empty keys, classified plans, and mode precedence.
func TestSuspendProcessInstanceTreeOutput(t *testing.T) {
	for _, empty := range []bool{false, true} {
		for _, dry := range []bool{false, true} {
			for _, noWait := range []bool{false, true} {
				for _, mode := range []string{"human", "json", "keys", "quiet", "quiet-json", "quiet-keys", "json-keys"} {
					t.Run(fmt.Sprintf("empty=%t/dry=%t/noWait=%t/%s", empty, dry, noWait, mode), func(t *testing.T) {
						f := newSuspensionTreeFixture(t)
						f.observed[suspendKeyA] = "SUSPENDED"
						f.observed[suspendKeyB] = "COMPLETED"
						if empty {
							f.observed[suspendKeyC] = "UNKNOWN"
						}
						args := []string{"suspend", "process-instance", "--tree", "--key", suspendKeyA, "--auto-confirm"}
						if dry {
							args = append(args, "--dry-run")
						}
						if noWait {
							args = append(args, "--no-wait")
						}
						if strings.Contains(mode, "json") {
							args = append(args, "--json")
						}
						if strings.Contains(mode, "keys") {
							args = append(args, "--keys-only")
						}
						if strings.Contains(mode, "quiet") {
							args = append(args, "--quiet")
						}
						out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), "", args...)
						require.NoError(t, err, stderr)
						active, excluded := 1, 1
						if empty {
							active, excluded = 0, 2
						}
						if mode != "human" {
							require.Empty(t, stderr)
						}
						switch {
						case strings.Contains(mode, "json"):
							env := requireSingleJSONObjectDocument(t, out)
							outcome, status, submission, confirmation := "succeeded", "confirmed", "accepted", "confirmed"
							if noWait {
								outcome, status, confirmation = "accepted", "accepted", "not_requested"
							}
							if dry {
								outcome, status, submission, confirmation = "succeeded", "preview", "not_submitted", "not_requested"
							}
							if empty {
								outcome = "succeeded"
							}
							require.Equal(t, outcome, env["outcome"])
							payload := requireJSONObject(t, env["payload"])
							items := requireJSONItems(t, payload["items"], active)
							plan := requireJSONObject(t, payload["plan"])
							require.Equal(t, []any{suspendKeyA}, plan["requestedKeys"])
							targets := []any{}
							if !empty {
								targets = append(targets, suspendKeyC)
							}
							require.Equal(t, targets, plan["targetKeys"])
							instances := requireJSONItems(t, plan["instances"], 3)
							require.Equal(t, "already_suspended", requireJSONObject(t, instances[0])["classification"])
							require.Equal(t, "excluded", requireJSONObject(t, instances[1])["classification"])
							if empty {
								require.Equal(t, "excluded", requireJSONObject(t, instances[2])["classification"])
							}
							for _, v := range items {
								item := requireJSONObject(t, v)
								require.Equal(t, suspendKeyC, item["key"])
								require.Equal(t, status, item["status"])
								require.Equal(t, submission, item["submission"])
								require.Equal(t, confirmation, item["confirmation"])
								require.Equal(t, !dry && !noWait, item["hasObservation"])
								if dry || noWait {
									require.NotContains(t, item, "lastObservedState")
									require.NotContains(t, item, "suspendedDate")
								}
								if dry {
									require.NotContains(t, item, "statusCode")
								}
							}
						case strings.Contains(mode, "keys"):
							want := ""
							if !empty {
								want = suspendKeyC + "\n"
							}
							require.Equal(t, want, out)
						default:
							require.Empty(t, out)
							if mode == "quiet" {
								break
							}
							scope := fmt.Sprintf("selected process instances: 1; process instances in scope: 3; active targets: %d; already suspended: 1; excluded: %d", active, excluded)
							records := [][2]string{}
							if !dry && !empty {
								records = append(records, [2]string{"INFO", "plan: suspend 1 process instance(s); 1 selected; 3 in scope; 1 already suspended; 1 excluded"})
							}
							if dry {
								records = append(records, [2]string{"INFO", "dry run: suspend process-instance tree"})
							}
							records = append(records, [2]string{"INFO", scope})
							summary := "suspension: suspended 1/1 process instance(s)"
							if noWait {
								summary = "suspension: submitted 1/1 process instance(s); confirmation not requested"
							}
							if empty {
								summary = "suspension: no active process instances to suspend; no changes applied"
							}
							if dry {
								summary = "outcome: preview; no changes applied"
							}
							records = append(records, [2]string{"INFO", summary})
							require.Equal(t, records, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
						}
						require.Equal(t, []string{"GET " + suspendKeyA, "POST " + suspendKeyA, "POST " + suspendKeyB, "POST " + suspendKeyC}, f.discoverySnapshot())
						calls, writes, reads := f.snapshot()
						if dry || empty {
							require.Empty(t, calls)
							require.Empty(t, writes)
							require.Empty(t, reads)
						} else {
							require.Equal(t, map[string]int{suspendKeyC: 1}, writes)
							if noWait {
								require.Empty(t, reads)
								require.Len(t, calls, 1)
							} else {
								require.Equal(t, writes, reads)
								require.Len(t, calls, 2)
							}
						}
					})
				}
			}
		}
	}
}

// TestSuspendProcessInstanceTreeNoopLogging verifies verbose preview/no-op facts
// obey configured/inherited destinations and filtering in both logger formats.
func TestSuspendProcessInstanceTreeNoopLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, level := range []string{"info", "warn"} {
				for _, dry := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%s/%s/dry=%t", format, destination, level, dry), func(t *testing.T) {
						f := newSuspensionTreeFixture(t)
						f.observed[suspendKeyA] = "SUSPENDED"
						f.observed[suspendKeyB] = "COMPLETED"
						f.observed[suspendKeyC] = "UNKNOWN"
						args := []string{"--config", completionLogConfig(t, f.server.URL, format, level), "suspend", "pi", "--tree", "--key", suspendKeyA, "--verbose", "--no-wait"}
						if dry {
							args = append(args, "--dry-run")
						}
						out, stderr, err := runSuspensionTerminal(t, args, destination, nil)
						require.NoError(t, err, stderr)
						require.Empty(t, out)
						records := bulkCompletionDiagnosticRecords(t, stderr, format)
						if level == "warn" {
							require.Empty(t, records)
							return
						}
						want := [][2]string{}
						if dry {
							want = append(want, [2]string{"INFO", "dry run: suspend process-instance tree"})
						}
						want = append(want, [2]string{"INFO", "selected process instances: 1; process instances in scope: 3; active targets: 0; already suspended: 1; excluded: 2"})
						summary := "suspension: no active process instances to suspend; no changes applied"
						if dry {
							summary = "outcome: preview; no changes applied"
						}
						want = append(want, [2]string{"INFO", summary},
							[2]string{"INFO", suspendKeyA + ": observed state: SUSPENDED; classification: already_suspended"},
							[2]string{"INFO", suspendKeyB + ": observed state: COMPLETED; classification: excluded"},
							[2]string{"INFO", suspendKeyC + ": observed state: UNKNOWN; classification: excluded"})
						require.Equal(t, want, records)
						calls, _, _ := f.snapshot()
						require.Empty(t, calls)
					})
				}
			}
		}
	}
}
