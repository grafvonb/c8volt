// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestResumeProcessInstanceOutput exercises selected-format results through real dispatch,
// including preview/no-wait truthfulness and quiet/JSON/keys precedence.
func TestResumeProcessInstanceOutput(t *testing.T) {
	for _, dry := range []bool{false, true} {
		for _, noWait := range []bool{false, true} {
			for _, mode := range []string{"human", "json", "keys", "quiet", "quiet-json", "quiet-keys", "json-keys"} {
				t.Run(fmt.Sprintf("dry=%t/noWait=%t/%s", dry, noWait, mode), func(t *testing.T) {
					f := newResumptionFixture(t)
					args := []string{"resume", "process-instance", "--key", resumeKeyA + "," + resumeKeyB, "--auto-confirm"}
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
						require.Equal(t, resumeKeyA+"\n"+resumeKeyB+"\n", out)
					case strings.Contains(mode, "quiet"):
						require.Empty(t, out)
					default:
						want := "resumption: resumed 2/2 process instance(s)"
						if noWait {
							want = "resumption: submitted 2/2 process instance(s); confirmation not requested"
						}
						if dry {
							want = "dry run: resume process-instance; selected: 2; eligibility not checked; no changes applied"
						}
						require.Empty(t, out)
						require.Equal(t, [][2]string{{"INFO", want}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
					}
					calls, writes, reads := f.snapshot()
					if dry {
						require.Empty(t, calls)
					} else {
						require.Equal(t, map[string]int{resumeKeyA: 1, resumeKeyB: 1}, writes)
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

// TestResumeProcessInstanceInvalidInputAndCapability rejects selection/mode errors and old
// adapters before requests; empty stdin remains an input error in every result mode.
func TestResumeProcessInstanceInvalidInputAndCapability(t *testing.T) {
	for _, tree := range []bool{false, true} {
		t.Run(fmt.Sprintf("tree=%t", tree), func(t *testing.T) {

			for _, tc := range []struct {
				name, stdin, version string
				flags                []string
				code                 int
				message              string
			}{
				{name: "missing", code: 1, message: "no process instance keys"},
				{name: "invalid stdin", stdin: "bad\n", flags: []string{"-", "--key", resumeKeyA, "--auto-confirm"}, code: 2, message: "16 digits"},
				{name: "invalid stdin json", stdin: "bad\n", flags: []string{"-", "--key", resumeKeyA, "--auto-confirm", "--json"}, code: 2, message: "16 digits"},
				{name: "invalid", flags: []string{"--key", resumeKeyA + ",bad"}, code: 2, message: "16 digits"},
				{name: "empty comma", flags: []string{"--key", resumeKeyA + ","}, code: 2, message: "16 digits"},
				{name: "workers", flags: []string{"--key", resumeKeyA, "--workers", "0"}, code: 2, message: "positive integer"},
				{name: "negative workers", flags: []string{"--key", resumeKeyA, "--workers", "-1"}, code: 2, message: "positive integer"},
				{name: "json interactive", flags: []string{"--key", resumeKeyA, "--json"}, code: 2, message: "requires"},
				{name: "json verbose", flags: []string{"--key", resumeKeyA, "--json", "--verbose", "--dry-run"}, code: 2, message: "cannot be combined"},
				{name: "88", version: "8.8", flags: []string{"--key", resumeKeyA, "--json", "--auto-confirm"}, code: 1, message: "unsupported"},
				{name: "89 preview", version: "8.9", flags: []string{"--key", resumeKeyA, "--json", "--dry-run", "--no-wait"}, code: 1, message: "unsupported"},
				{name: "88 preview", version: "8.8", flags: []string{"--key", resumeKeyA, "--dry-run"}, code: 1, message: "unsupported"},
				{name: "89", version: "8.9", flags: []string{"--key", resumeKeyA}, code: 1, message: "unsupported"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					f := newResumptionFixture(t)
					v := tc.version
					if v == "" {
						v = "8.10"
					}
					out, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, f.server.URL, v), tc.stdin, resumptionModeArgs(append([]string{"resume", "pi"}, tc.flags...), tree)...)
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
						f := newResumptionFixture(t)
						args := append([]string{"resume", "pi", "-", "--auto-confirm", "--no-wait"}, mode...)
						out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), stdin, resumptionModeArgs(args, tree)...)
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
		})
	}
}

// TestResumeProcessInstancePartialEvidence retains accepted-but-unconfirmed, rejected and
// unknown submission evidence, original exits and keys through execution, never reconciling rejection.
func TestResumeProcessInstancePartialEvidence(t *testing.T) {
	for _, tree := range []bool{false, true} {
		t.Run(fmt.Sprintf("tree=%t", tree), func(t *testing.T) {

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
				{name: "canceled race", state: "CANCELED", code: 6, submission: "accepted", status: "failed", reason: "cancellation observed"},
				{name: "terminated race", state: "TERMINATED", code: 6, submission: "accepted", status: "failed", reason: "cancellation observed"},
				{name: "permission", readStatus: 403, code: 1, submission: "accepted", status: "failed", reason: "confirmation failed"},
				{name: "suspended timeout", state: "SUSPENDED", code: 4, submission: "accepted", status: "timed_out", reason: "timed out"},
				{name: "unknown timeout", state: "UNKNOWN", code: 4, submission: "accepted", status: "timed_out", reason: "timed out"},
				{name: "absent timeout", readStatus: 404, code: 4, submission: "accepted", status: "timed_out", reason: "timed out"},
			} {
				for _, mode := range []string{"json", "keys", "human"} {
					t.Run(tc.name+"/"+mode, func(t *testing.T) {
						f := newResumptionModeFixture(t, tree)
						f.reject[resumeKeyB] = tc.reject
						f.readStatus[resumeKeyB] = tc.readStatus
						if tc.state != "" {
							f.states[resumeKeyB] = []string{tc.state}
						}
						args := []string{"resume", "pi", "--key", resumeKeyA + "," + resumeKeyB, "--auto-confirm", "--workers", "1", "--backoff-max-retries", "1", "--backoff-timeout", "1s"}
						if tc.noWait {
							args = append(args, "--no-wait")
						}
						if mode == "json" {
							args = append(args, "--json")
						}
						if mode == "keys" {
							args = append(args, "--keys-only")
						}
						out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), "", resumptionModeArgs(args, tree)...)
						assertUserTaskCompletionExit(t, err, tc.code)
						_, writes, reads := f.snapshot()
						require.Equal(t, map[string]int{resumeKeyA: 1, resumeKeyB: 1}, writes)
						if tc.reject != 0 || tc.noWait {
							require.Zero(t, reads[resumeKeyB])
						} else {
							require.Equal(t, 1, reads[resumeKeyB])
						}
						if mode == "json" {
							require.Empty(t, stderr)
							env := requireSingleJSONObjectDocument(t, out)
							require.NotEqual(t, "succeeded", env["outcome"])
							require.NotEqual(t, "accepted", env["outcome"])
							requireResumptionModePlan(t, requireJSONObject(t, env["payload"]), tree, []string{resumeKeyA, resumeKeyB})
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
							wantRecords := [][2]string{{"ERROR", "resume process instances: " + tc.reason + ": 1"}}
							accepted := 1
							if tc.submission == "accepted" {
								accepted = 2
							}
							if mode == "keys" {
								want := resumeKeyA + "\n"
								if accepted == 2 {
									want += resumeKeyB + "\n"
								}
								require.Equal(t, want, out)
							} else {
								require.Empty(t, out)
								summary := "resumption: resumed 1/2 process instance(s)"
								if tc.noWait {
									summary = "resumption: submitted 1/2 process instance(s); confirmation not requested"
								} else if accepted == 2 {
									summary += "; accepted but unconfirmed 1"
								}
								summary += "; failed 1"
								wantRecords = append([][2]string{{"INFO", summary}}, wantRecords...)
								if tree {
									wantRecords = append([][2]string{{"INFO", resumptionTreeTestApproval(2)}, {"INFO", resumptionTreeTestScope(2)}}, wantRecords...)
								}
							}
							require.Equal(t, wantRecords, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
						}
					})
				}
			}
		})
	}
}

// TestResumeProcessInstanceEmptyView protects direct empty-result rendering without making
// missing CLI selection a successful no-op or attributing acceptance to --no-wait.
func TestResumeProcessInstanceEmptyView(t *testing.T) {
	resetCommandTreeFlags(Root())
	t.Cleanup(func() { resetCommandTreeFlags(Root()) })
	flagNoWait = true
	for _, preview := range []bool{false, true} {
		for _, mode := range []string{"json", "keys", "human", "quiet"} {
			flagViewAsJson = mode == "json"
			flagViewKeysOnly = mode == "keys"
			flagQuiet = mode == "quiet"
			var out, diagnostics bytes.Buffer
			c := &cobra.Command{Use: "process-instance"}
			c.SetOut(&out)
			c.SetContext(logging.ToContext(context.Background(), logging.New(logging.LoggerConfig{Level: "info", Format: "json", Writer: &diagnostics})))
			require.NoError(t, renderProcessInstanceResumption(c, process.ResumptionResults{}, preview, nil))
			switch mode {
			case "json":
				require.Empty(t, diagnostics.String())
				env := requireSingleJSONObjectDocument(t, out.String())
				require.Equal(t, "succeeded", env["outcome"])
				require.Equal(t, []any{}, requireJSONObject(t, env["payload"])["items"])
			case "keys", "quiet":
				require.Zero(t, out.Len())
				require.Empty(t, diagnostics.String())
			default:
				require.Empty(t, out.String())
				if preview {
					require.Equal(t, [][2]string{{"INFO", "dry run: resume process-instance; selected: 0; eligibility not checked; no changes applied"}}, bulkCompletionDiagnosticRecords(t, diagnostics.String(), "json"))
				} else {
					require.Equal(t, [][2]string{{"INFO", "resumption: no process instances selected; no changes applied"}}, bulkCompletionDiagnosticRecords(t, diagnostics.String(), "json"))
				}
			}
		}
	}
}

// A fail-fast human summary must retain unsubmitted scope as well as completed work.
func TestResumeProcessInstanceFailFastSummary(t *testing.T) {
	for _, noWait := range []bool{false, true} {
		t.Run(fmt.Sprint(noWait), func(t *testing.T) {
			f := newResumptionFixture(t)
			f.reject[resumeKeyB] = 409
			args := []string{"resume", "pi", "--key", resumeKeyA + "," + resumeKeyB + "," + resumeKeyC, "--auto-confirm", "--workers", "1", "--fail-fast"}
			summary := "resumption: resumed 1/3 process instance(s); failed 1; not attempted 1"
			if noWait {
				args = append(args, "--no-wait")
				summary = "resumption: submitted 1/3 process instance(s); confirmation not requested; failed 1; not attempted 1"
			}
			out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), "", args...)
			assertUserTaskCompletionExit(t, err, 6)
			require.Empty(t, out)
			require.Equal(t, [][2]string{{"INFO", summary}, {"ERROR", "resume process instances: rejected: 1; not attempted: 1"}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
			_, writes, _ := f.snapshot()
			require.Equal(t, map[string]int{resumeKeyA: 1, resumeKeyB: 1}, writes)
		})
	}
}

// TestResumeProcessInstanceWriterFailure proves one write attempt and preserved operation exit
// even when the configured result destination fails after partial output.
func TestResumeProcessInstanceWriterFailure(t *testing.T) {
	for _, tree := range []bool{false, true} {
		t.Run(fmt.Sprintf("tree=%t", tree), func(t *testing.T) {

			for _, phase := range []string{"preview", "accepted", "rejected"} {
				for _, mode := range []string{"json", "keys"} {
					t.Run(phase+"/"+mode, func(t *testing.T) {
						f := newResumptionModeFixture(t, tree)
						if phase == "rejected" {
							f.reject[resumeKeyB] = 409
						}
						args := []string{"--config", completionLogConfig(t, f.server.URL, "plain-time", "info"), "resume", "pi", "--key", resumeKeyA + "," + resumeKeyB, "--auto-confirm", "--no-wait", "--workers", "1"}
						if phase == "preview" {
							args = append(args, "--dry-run")
						}
						if mode == "json" {
							args = append(args, "--json")
						}
						if mode == "keys" {
							args = append(args, "--keys-only")
						}
						encoded, err := json.Marshal(resumptionModeArgs(args, tree))
						require.NoError(t, err)
						out, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, "TestResumeProcessInstanceWriterProcess", "", map[string]string{"C8VOLT_RESUME_ARGS": string(encoded)}, "")
						code := 1
						if phase == "rejected" {
							code = 6
						}
						assertUserTaskCompletionExit(t, err, code)
						require.Equal(t, "partial-result\n", out)
						message := "resume process instances: could not write result"
						if phase == "preview" {
							message = "resume process instances: could not write preview"
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
		})
	}
}

// TestResumeProcessInstanceWriterProcess injects only the result writer; normal dispatch and logging remain intact.
func TestResumeProcessInstanceWriterProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_RESUME_ARGS")), &args))
	root := Root()
	root.SetOut(completionFailingWriter{})
	root.SetErr(os.Stderr)
	root.SetArgs(args)
	require.NoError(t, root.Execute())
	os.Exit(0)
}
