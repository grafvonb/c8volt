// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestElementScopeVariablesUpdateModes exercises authoritative no-ops and previews
// through execution, checking exact human text, JSON EOF and request counts.
func TestElementScopeVariablesUpdateModes(t *testing.T) {
	for _, kind := range []string{"empty object", "unchanged", "addition"} {
		for _, dryRun := range []bool{false, true} {
			for _, policy := range []string{"--auto-confirm", "--automation"} {
				for _, noWait := range []bool{false, true} {
					for _, mode := range [][]string{nil, {"--json"}, {"--keys-only"}, {"--quiet"}, {"--quiet", "--json"}, {"--quiet", "--keys-only"}} {
						t.Run(fmt.Sprintf("%s/dry=%t/%s/noWait=%t/%v", kind, dryRun, policy, noWait, mode), func(t *testing.T) {
							f := newElementUpdateFixture(t)
							payload := `{"x":true}`
							if kind == "empty object" {
								payload = "{}"
							}
							if kind == "unchanged" {
								f.local[scopeElementA] = []string{scopeVariableJSON("x", "true", scopeElementA, false)}
							}
							args := []string{policy, "update", "element", "--key", scopeElementA, "--vars", payload}
							if dryRun {
								args = append(args, "--dry-run")
							}
							if noWait {
								args = append(args, "--no-wait")
							}
							args = append(args, mode...)
							out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), "", args...)
							require.NoError(t, err, stderr)
							preview := dryRun || kind != "addition"
							modes := strings.Join(mode, " ")
							switch {
							case strings.Contains(modes, "--json"):
								require.Empty(t, stderr)
								env := requireSingleJSONObjectDocument(t, out)
								outcome := "succeeded"
								if !preview && noWait {
									outcome = "accepted"
								}
								require.Equal(t, outcome, env["outcome"])
								p := requireJSONObject(t, env["payload"])
								require.NotEmpty(t, env["tenantContext"])
								if preview {
									require.Equal(t, "update", p["operation"])
									require.Equal(t, false, p["mutationSubmitted"])
									require.Equal(t, []any{scopeElementA}, p["requestedKeys"])
									require.Equal(t, float64(1), p["requestedCount"])
									updateCount, addCount, sameCount := float64(0), float64(0), float64(0)
									if kind == "addition" {
										updateCount = 1
										addCount = 1
									}
									if kind == "unchanged" {
										sameCount = 1
									}
									require.Equal(t, updateCount, p["updateCount"])
									require.Equal(t, addCount, p["variableAddCount"])
									require.Equal(t, sameCount, p["variableUnchangedCount"])
									require.Equal(t, float64(0), p["variableChangeCount"])
									require.Equal(t, float64(0), p["variableUntouchedCount"])
									item := requireJSONObject(t, requireJSONItems(t, p["elements"], 1)[0])
									require.Equal(t, scopeElementA, item["elementKey"])
									require.Equal(t, scopeElementA, item["scopeKey"])
									require.Equal(t, scopeOwner, item["processInstanceKey"])
									require.Equal(t, "tenant-a", item["tenantId"])
									for _, key := range []string{"additions", "changes", "unchangedRequested", "untouched"} {
										require.IsType(t, []any{}, item[key])
										if key == "changes" || key == "untouched" || kind == "empty object" {
											require.Equal(t, []any{}, item[key])
										}
									}
									require.NotContains(t, item, "mutationVariables")
									require.NotContains(t, item, "verificationValues")
								} else {
									require.Len(t, requireJSONItems(t, p["items"], 1), 1)
								}
							case strings.Contains(modes, "--keys-only"):
								want := ""
								if kind == "addition" {
									want = scopeElementA + "\n"
								}
								require.Equal(t, want, out)
								require.Empty(t, stderr)
							case strings.Contains(modes, "--quiet"):
								require.Empty(t, out)
								require.Empty(t, stderr)
							default:
								info, diagnostics := assertBulkCompletionHumanLogs(t, out, stderr)
								require.Empty(t, diagnostics)
								prefix := ""
								if preview && policy != "--automation" {
									prefix = "selection scope: explicit resource keys; tenant filter not applied\naffected tenants: tenant-a\n"
								}
								label := "plan"
								if dryRun {
									label = "dry run"
								}
								expected := ""
								if kind != "addition" {
									same := 0
									if kind == "unchanged" {
										same = 1
									}
									expected = fmt.Sprintf("%s: update element variables: nothing to update (%d requested value(s) already match local variables); no changes applied; no confirmation required\n", label, same)
								} else if dryRun {
									expected = "dry run: update element variables: 1 element(s), 0 change(s), 1 addition(s), 0 unchanged, 0 untouched; no changes applied\n" + scopeElementA + ": + x (element:" + scopeElementA + "): true\n"
								} else {
									confirmed := 1
									if noWait {
										confirmed = 0
									}
									expected = fmt.Sprintf("update element variables: 1 selected; accepted: 1, confirmed: %d, unchanged: 0, failed: 0, skipped: 0\n", confirmed)
								}
								require.Equal(t, prefix+expected, info)
							}
							calls, writes, reads := f.snapshot()
							wantReads, wantWrites := 1, 0
							if !preview {
								wantWrites = 1
								if !noWait {
									wantReads = 2
								}
							}
							require.Len(t, writes, wantWrites)
							require.Equal(t, map[string]int{scopeElementA: wantReads}, reads)
							require.Len(t, calls, 1+wantReads+wantWrites)
						})
					}
				}
			}
		}
	}
}

// TestElementScopeVariablesUpdatePreviewKeys excludes unchanged targets and preserves unique input order.
func TestElementScopeVariablesUpdatePreviewKeys(t *testing.T) {
	f := newElementUpdateFixture(t)
	f.local[scopeElementB] = []string{scopeVariableJSON("x", "true", scopeElementB, false)}
	out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), "", "--keys-only", "update", "element", "--key", scopeElementA+","+scopeElementB+","+scopeElementC+","+scopeElementA, "--vars", `{"x":true}`, "--dry-run")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	require.Equal(t, scopeElementA+"\n"+scopeElementC+"\n", out)
	calls, writes, reads := f.snapshot()
	require.Len(t, calls, 6)
	require.Empty(t, writes)
	require.Len(t, reads, 3)
}

// TestElementScopeVariablesUpdateConfirmationFailure retains accepted keys and one failing envelope, including fail-fast unstarted confirmation.
func TestElementScopeVariablesUpdateConfirmationFailure(t *testing.T) {
	for _, failFast := range []bool{false, true} {
		for _, mode := range []string{"--json", "--keys-only", "--quiet", "--verbose"} {
			t.Run(fmt.Sprintf("fast=%t/%s", failFast, mode), func(t *testing.T) {
				f := newElementUpdateFixture(t)
				f.confirmationFailure = true
				args := []string{"--auto-confirm", mode, "update", "element", "--key", scopeElementA + "," + scopeElementB, "--vars", `{"x":true}`, "--workers", "1"}
				if failFast {
					args = append(args, "--fail-fast")
				}
				out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), "", args...)
				assertUserTaskCompletionExit(t, err, 2)
				calls, writes, reads := f.snapshot()
				require.Len(t, writes, 2)
				require.Equal(t, 2, reads[scopeElementA])
				secondReads := 2
				if failFast {
					secondReads = 1
				}
				require.Equal(t, secondReads, reads[scopeElementB])
				require.Len(t, calls, 6+secondReads)
				if mode == "--json" {
					require.Empty(t, stderr)
					env := requireSingleJSONObjectDocument(t, out)
					require.Equal(t, "invalid_input", env["class"])
					items := requireJSONItems(t, requireJSONObject(t, env["payload"])["items"], 2)
					for i, item := range items {
						r := requireJSONObject(t, item)
						require.Equal(t, true, r["mutationAccepted"])
						require.Equal(t, "accepted", r["submissionStatus"])
						status, confirmation := "confirmation_failed", "failed"
						if failFast && i == 1 {
							status = "submitted"
							confirmation = "not_started"
						}
						require.Equal(t, status, r["status"])
						require.Equal(t, confirmation, r["confirmationStatus"])
					}
				} else {
					if mode == "--keys-only" {
						require.Equal(t, scopeElementA+"\n"+scopeElementB+"\n", out)
					} else {
						require.Empty(t, out)
					}
					records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
					errors := []string{}
					for _, record := range records {
						if record[0] == "ERROR" {
							errors = append(errors, record[1])
						}
					}
					message := "element variable update: confirmation failed: 2"
					if failFast {
						message = "element variable update: confirmation failed: 1; confirmation not started: 1"
					}
					require.Equal(t, []string{message}, errors)
					for _, bad := range []string{"POST", "private.invalid", "body-only-secret", f.server.URL} {
						require.NotContains(t, out+stderr, bad)
					}
				}
			})
		}
	}
}

// TestElementScopeVariablesUpdateClassifications preserves before/truncation evidence and all
// four preview classes without mistaking parent values for local equality.
func TestElementScopeVariablesUpdateClassifications(t *testing.T) {
	f := newElementUpdateFixture(t)
	f.local[scopeElementA] = []string{
		scopeVariableJSON("added", "true", scopeOwner, false),
		scopeVariableJSON("change", "false", scopeElementA, false),
		scopeVariableJSON("truncated", "true", scopeElementA, true),
		scopeVariableJSON("same", "true", scopeElementA, false),
		scopeVariableJSON("untouched", `{"nested":[1,2]}`, scopeElementA, false),
	}
	out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), "", "--json", "update", "element", "--key", scopeElementA, "--vars", `{"added":true,"change":true,"truncated":true,"same":true}`, "--dry-run", "--no-wait")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	payload := requireJSONObject(t, requireSingleJSONObjectDocument(t, out)["payload"])
	for key, count := range map[string]float64{"variableAddCount": 1, "variableChangeCount": 2, "variableUnchangedCount": 1, "variableUntouchedCount": 1, "updateCount": 1} {
		require.Equal(t, count, payload[key])
	}
	p := requireJSONObject(t, requireJSONItems(t, payload["elements"], 1)[0])
	require.Equal(t, "added", requireJSONObject(t, requireJSONItems(t, p["additions"], 1)[0])["name"])
	changes := requireJSONItems(t, p["changes"], 2)
	require.Equal(t, false, requireJSONObject(t, changes[0])["before"])
	require.Equal(t, true, requireJSONObject(t, changes[1])["apiTruncated"])
	calls, writes, reads := f.snapshot()
	require.Len(t, calls, 2)
	require.Empty(t, writes)
	require.Equal(t, 1, reads[scopeElementA])
}

// TestElementScopeVariablesUpdateMutationFailFast preserves accepted keys while distinguishing
// failed and unscheduled destinations, without starting confirmation after the mutation failure.
func TestElementScopeVariablesUpdateMutationFailFast(t *testing.T) {
	f := newElementUpdateFixture(t)
	f.mutationFailure = rejectElementUpdate
	out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), "", "--json", "--auto-confirm", "update", "element", "--key", scopeElementA+","+scopeElementB+","+scopeElementC, "--vars", `{"x":1}`, "--workers", "1", "--no-worker-limit", "--fail-fast")
	assertUserTaskCompletionExit(t, err, 2)
	require.Empty(t, stderr)
	env := requireSingleJSONObjectDocument(t, out)
	require.Equal(t, "invalid_input", env["class"])
	items := requireJSONItems(t, requireJSONObject(t, env["payload"])["items"], 3)
	a, b, c := requireJSONObject(t, items[0]), requireJSONObject(t, items[1]), requireJSONObject(t, items[2])
	require.Equal(t, true, a["mutationAccepted"])
	require.Equal(t, "not_started", a["confirmationStatus"])
	require.Equal(t, "mutation_failed", b["status"])
	require.Equal(t, "rejected", b["submissionStatus"])
	require.Equal(t, "skipped", c["status"])
	require.Equal(t, "not_sent", c["submissionStatus"])
	calls, writes, reads := f.snapshot()
	require.Len(t, calls, 8)
	require.Len(t, writes, 1)
	require.Equal(t, map[string]int{scopeElementA: 1, scopeElementB: 1, scopeElementC: 1}, reads)
}
