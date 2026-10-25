// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestSuspendProcessInstanceOutput exercises selected-format results through real dispatch,
// including preview/no-wait truthfulness and quiet/JSON/keys precedence.
func TestSuspendProcessInstanceOutput(t *testing.T) {
	for _, dry := range []bool{false, true} {
		for _, noWait := range []bool{false, true} {
			for _, mode := range []string{"human", "json", "keys", "quiet", "quiet-json", "quiet-keys", "json-keys"} {
				t.Run(fmt.Sprintf("dry=%t/noWait=%t/%s", dry, noWait, mode), func(t *testing.T) {
					f := newSuspensionFixture(t)
					args := []string{"suspend", "process-instance", "--key", suspendKeyA + "," + suspendKeyB, "--auto-confirm"}
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
					require.Empty(t, stderr)
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
						require.Equal(t, outcome, env["outcome"])
						items := requireJSONItems(t, requireJSONObject(t, env["payload"])["items"], 2)
						for _, item := range items {
							r := requireJSONObject(t, item)
							require.Equal(t, status, r["status"])
							require.Equal(t, submission, r["submission"])
							require.Equal(t, confirmation, r["confirmation"])
							if dry {
								require.NotContains(t, r, "statusCode")
								require.NotContains(t, r, "suspendedDate")
								require.Equal(t, false, r["hasObservation"])
							}
						}
					case strings.Contains(mode, "keys"):
						require.Equal(t, suspendKeyA+"\n"+suspendKeyB+"\n", out)
					case strings.Contains(mode, "quiet"):
						require.Empty(t, out)
					default:
						want := "suspend process instances: 2 selected; accepted: 2, confirmed: 2, failed: 0, skipped: 0\n"
						if noWait {
							want = "suspend process instances: 2 selected; accepted: 2, confirmed: 0, failed: 0, skipped: 0\n"
						}
						if dry {
							want = "dry run: suspend process instances: 2 selected; no changes applied; eligibility not checked\n"
						}
						require.Equal(t, want, out)
					}
					calls, writes, reads := f.snapshot()
					if dry {
						require.Empty(t, calls)
					} else {
						require.Equal(t, map[string]int{suspendKeyA: 1, suspendKeyB: 1}, writes)
						if noWait {
							require.Empty(t, reads)
							require.Len(t, calls, 2)
						} else {
							require.Equal(t, writes, reads)
							require.Len(t, calls, 4)
						}
					}
				})
			}
		}
	}
}

// TestSuspendProcessInstanceInvalidInputAndCapability rejects selection/mode errors and old
// adapters before requests; empty stdin remains an input error in every result mode.
func TestSuspendProcessInstanceInvalidInputAndCapability(t *testing.T) {
	for _, tc := range []struct {
		name, stdin, version string
		flags                []string
		code                 int
		message              string
	}{
		{name: "missing", code: 1, message: "no process instance keys"},
		{name: "invalid", flags: []string{"--key", suspendKeyA + ",bad"}, code: 2, message: "16 digits"},
		{name: "empty comma", flags: []string{"--key", suspendKeyA + ","}, code: 2, message: "16 digits"},
		{name: "workers", flags: []string{"--key", suspendKeyA, "--workers", "0"}, code: 2, message: "positive integer"},
		{name: "negative workers", flags: []string{"--key", suspendKeyA, "--workers", "-1"}, code: 2, message: "positive integer"},
		{name: "json interactive", flags: []string{"--key", suspendKeyA, "--json"}, code: 2, message: "requires"},
		{name: "json verbose", flags: []string{"--key", suspendKeyA, "--json", "--verbose", "--dry-run"}, code: 2, message: "cannot be combined"},
		{name: "88", version: "8.8", flags: []string{"--key", suspendKeyA, "--json", "--auto-confirm"}, code: 1, message: "unsupported"},
		{name: "89 preview", version: "8.9", flags: []string{"--key", suspendKeyA, "--json", "--dry-run", "--no-wait"}, code: 1, message: "unsupported"},
		{name: "88 preview", version: "8.8", flags: []string{"--key", suspendKeyA, "--dry-run"}, code: 1, message: "unsupported"},
		{name: "89", version: "8.9", flags: []string{"--key", suspendKeyA}, code: 1, message: "unsupported"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newSuspensionFixture(t)
			v := tc.version
			if v == "" {
				v = "8.10"
			}
			out, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, f.server.URL, v), tc.stdin, append([]string{"suspend", "pi"}, tc.flags...)...)
			assertUserTaskCompletionExit(t, err, tc.code)
			calls, _, _ := f.snapshot()
			require.Empty(t, calls)
			require.NotContains(t, out+stderr, "proceed?")
			if strings.Contains(strings.Join(tc.flags, " "), "--json") {
				env := requireSingleJSONObjectDocument(t, out)
				require.NotContains(t, env, "payload")
				require.NotEqual(t, "accepted", env["outcome"])
				require.Contains(t, requireJSONObject(t, env["detail"])["message"], tc.message)
			} else {
				require.Empty(t, out)
				require.Contains(t, stderr, tc.message)
			}
		})
	}
	for _, stdin := range []string{"", " \n\t\n"} {
		for _, mode := range [][]string{nil, {"--json"}, {"--keys-only"}, {"--quiet"}, {"--quiet", "--json"}, {"--quiet", "--keys-only"}} {
			t.Run(fmt.Sprintf("empty/%q/%v", stdin, mode), func(t *testing.T) {
				f := newSuspensionFixture(t)
				args := append([]string{"suspend", "pi", "-", "--auto-confirm", "--no-wait"}, mode...)
				out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), stdin, args...)
				assertUserTaskCompletionExit(t, err, 2)
				calls, _, _ := f.snapshot()
				require.Empty(t, calls)
				require.NotContains(t, stderr, "proceed?")
				if strings.Contains(strings.Join(mode, " "), "--json") {
					env := requireSingleJSONObjectDocument(t, out)
					require.Equal(t, "invalid", env["outcome"])
					require.Contains(t, requireJSONObject(t, env["detail"])["message"], "stdin contained no keys")
					require.Empty(t, stderr)
				} else {
					require.Empty(t, out)
					records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
					require.Len(t, records, 1)
					require.Equal(t, "ERROR", records[0][0])
					require.Contains(t, records[0][1], "stdin contained no keys")
				}
			})
		}
	}
}

// TestSuspendProcessInstancePartialEvidence retains accepted-but-unconfirmed, rejected and
// unknown submission evidence, original exits and keys through execution, never reconciling rejection.
func TestSuspendProcessInstancePartialEvidence(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		reject                     int
		state                      string
		readStatus, code           int
		submission, status, reason string
		noWait                     bool
	}{
		{name: "rejected", reject: 409, code: 6, submission: "rejected", status: "failed", reason: "rejected"},
		{name: "no-wait rejection", reject: 409, code: 6, submission: "rejected", status: "failed", reason: "rejected", noWait: true},
		{name: "unexpected success", reject: 200, code: 1, submission: "unknown", status: "failed", reason: "acceptance unknown"},
		{name: "completed race", state: "COMPLETED", code: 6, submission: "accepted", status: "failed", reason: "terminal observed"},
		{name: "canceled race", state: "CANCELED", code: 6, submission: "accepted", status: "failed", reason: "terminal observed"},
		{name: "terminated race", state: "TERMINATED", code: 6, submission: "accepted", status: "failed", reason: "terminal observed"},
		{name: "permission", readStatus: 403, code: 1, submission: "accepted", status: "failed", reason: "confirmation failed"},
		{name: "active timeout", state: "ACTIVE", code: 4, submission: "accepted", status: "timed_out", reason: "timed out"},
		{name: "unknown timeout", state: "UNKNOWN", code: 4, submission: "accepted", status: "timed_out", reason: "timed out"},
		{name: "absent timeout", readStatus: 404, code: 4, submission: "accepted", status: "timed_out", reason: "timed out"},
	} {
		for _, mode := range []string{"json", "keys", "human"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				f := newSuspensionFixture(t)
				f.reject[suspendKeyB] = tc.reject
				f.readStatus[suspendKeyB] = tc.readStatus
				if tc.state != "" {
					f.states[suspendKeyB] = []string{tc.state}
				}
				args := []string{"suspend", "pi", "--key", suspendKeyA + "," + suspendKeyB, "--auto-confirm", "--workers", "1", "--backoff-max-retries", "1", "--backoff-timeout", "1s"}
				if tc.noWait {
					args = append(args, "--no-wait")
				}
				if mode == "json" {
					args = append(args, "--json")
				}
				if mode == "keys" {
					args = append(args, "--keys-only")
				}
				out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), "", args...)
				assertUserTaskCompletionExit(t, err, tc.code)
				_, writes, reads := f.snapshot()
				require.Equal(t, map[string]int{suspendKeyA: 1, suspendKeyB: 1}, writes)
				if tc.reject != 0 || tc.noWait {
					require.Zero(t, reads[suspendKeyB])
				} else {
					require.Equal(t, 1, reads[suspendKeyB])
				}
				if mode == "json" {
					require.Empty(t, stderr)
					env := requireSingleJSONObjectDocument(t, out)
					require.NotEqual(t, "succeeded", env["outcome"])
					require.NotEqual(t, "accepted", env["outcome"])
					items := requireJSONItems(t, requireJSONObject(t, env["payload"])["items"], 2)
					first, last := requireJSONObject(t, items[0]), requireJSONObject(t, items[1])
					require.Equal(t, "accepted", first["submission"])
					require.Equal(t, tc.submission, last["submission"])
					require.Equal(t, tc.status, last["status"])
					require.NotEmpty(t, last["error"])
					if tc.submission == "accepted" {
						require.Equal(t, float64(204), last["statusCode"])
						require.Equal(t, "unconfirmed", last["confirmation"])
					} else {
						require.Equal(t, "not_requested", last["confirmation"])
						require.Equal(t, false, last["hasObservation"])
					}
					if tc.state != "" {
						require.Equal(t, tc.state, last["lastObservedState"])
						require.Equal(t, true, last["hasObservation"])
					}
					if tc.readStatus == 403 {
						require.Equal(t, false, last["hasObservation"])
						require.NotContains(t, last, "lastObservedState")
					}
					if tc.readStatus == 404 {
						require.Equal(t, "ABSENT", last["lastObservedState"])
						require.NotContains(t, last, "suspendedDate")
					}
				} else {
					require.Equal(t, [][2]string{{"ERROR", "suspend process instances: " + tc.reason + ": 1"}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
					accepted, confirmed := 1, 1
					if tc.submission == "accepted" {
						accepted = 2
					}
					if tc.noWait {
						confirmed = 0
					}
					if mode == "keys" {
						want := suspendKeyA + "\n"
						if accepted == 2 {
							want += suspendKeyB + "\n"
						}
						require.Equal(t, want, out)
					} else {
						require.Equal(t, fmt.Sprintf("suspend process instances: 2 selected; accepted: %d, confirmed: %d, failed: 1, skipped: 0\n", accepted, confirmed), out)
					}
				}
			})
		}
	}
}

// TestSuspendProcessInstanceEmptyView protects direct empty-result rendering without making
// missing CLI selection a successful no-op or attributing acceptance to --no-wait.
func TestSuspendProcessInstanceEmptyView(t *testing.T) {
	resetCommandTreeFlags(Root())
	t.Cleanup(func() { resetCommandTreeFlags(Root()) })
	flagNoWait = true
	for _, preview := range []bool{false, true} {
		for _, mode := range []string{"json", "keys", "human", "quiet"} {
			flagViewAsJson = mode == "json"
			flagViewKeysOnly = mode == "keys"
			flagQuiet = mode == "quiet"
			var out bytes.Buffer
			c := &cobra.Command{Use: "process-instance"}
			c.SetOut(&out)
			require.NoError(t, renderProcessInstanceSuspension(c, process.SuspensionResults{}, preview, nil))
			switch mode {
			case "json":
				env := requireSingleJSONObjectDocument(t, out.String())
				require.Equal(t, "succeeded", env["outcome"])
				require.Equal(t, []any{}, requireJSONObject(t, env["payload"])["items"])
			case "keys", "quiet":
				require.Zero(t, out.Len())
			default:
				if preview {
					require.Equal(t, "dry run: suspend process instances: 0 selected; no changes applied; eligibility not checked\n", out.String())
				} else {
					require.Equal(t, "suspend process instances: 0 selected; accepted: 0, confirmed: 0, failed: 0, skipped: 0\n", out.String())
				}
			}
		}
	}
}

// TestSuspendProcessInstanceWriterFailure proves one write attempt and preserved operation exit
// even when the configured result destination fails after partial output.
func TestSuspendProcessInstanceWriterFailure(t *testing.T) {
	for _, phase := range []string{"preview", "accepted", "rejected"} {
		for _, mode := range []string{"json", "keys", "human"} {
			t.Run(phase+"/"+mode, func(t *testing.T) {
				f := newSuspensionFixture(t)
				if phase == "rejected" {
					f.reject[suspendKeyB] = 409
				}
				args := []string{"--config", completionLogConfig(t, f.server.URL, "plain-time", "info"), "suspend", "pi", "--key", suspendKeyA + "," + suspendKeyB, "--auto-confirm", "--no-wait", "--workers", "1"}
				if phase == "preview" {
					args = append(args, "--dry-run")
				}
				if mode == "json" {
					args = append(args, "--json")
				}
				if mode == "keys" {
					args = append(args, "--keys-only")
				}
				encoded, err := json.Marshal(args)
				require.NoError(t, err)
				out, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, "TestSuspendProcessInstanceWriterProcess", "", map[string]string{"C8VOLT_SUSPEND_ARGS": string(encoded)}, "")
				code := 1
				if phase == "rejected" {
					code = 6
				}
				assertUserTaskCompletionExit(t, err, code)
				require.Equal(t, "partial-result\n", out)
				message := "suspend process instances: could not write result"
				if phase == "preview" {
					message = "suspend process instances: could not write preview"
				}
				require.Equal(t, [][2]string{{"ERROR", message}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
				calls, _, _ := f.snapshot()
				if phase == "preview" {
					require.Empty(t, calls)
				} else {
					require.Len(t, calls, 2)
				}
			})
		}
	}
}

// TestSuspendProcessInstanceWriterProcess injects only the result writer; normal dispatch and logging remain intact.
func TestSuspendProcessInstanceWriterProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_SUSPEND_ARGS")), &args))
	root := Root()
	root.SetOut(completionFailingWriter{})
	root.SetErr(os.Stderr)
	root.SetArgs(args)
	require.NoError(t, root.Execute())
	os.Exit(0)
}
