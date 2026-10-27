// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResumeProcessInstanceDiagnostics detects direct-stderr bypass, duplicate errors,
// lost partial payloads and raw joined-error disclosure at the real command boundary.
func TestResumeProcessInstanceDiagnostics(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, tc := range []struct {
				name                                           string
				flags                                          []string
				debug, configDebug, quiet, json, keys, verbose bool
			}{
				{name: "normal"}, {name: "verbose", flags: []string{"--verbose"}, verbose: true},
				{name: "quiet", flags: []string{"--quiet"}, quiet: true},
				{name: "debug flag", flags: []string{"--debug"}, debug: true},
				{name: "debug config", debug: true, configDebug: true},
				{name: "quiet beats debug", flags: []string{"--debug", "--quiet"}, configDebug: true, quiet: true},
				{name: "json", flags: []string{"--json"}, json: true},
				{name: "json debug", flags: []string{"--json", "--debug"}, json: true, debug: true},
				{name: "quiet json", flags: []string{"--json", "--quiet"}, json: true, quiet: true},
				{name: "keys", flags: []string{"--keys-only"}, keys: true},
				{name: "verbose keys", flags: []string{"--keys-only", "--verbose"}, keys: true, verbose: true},
				{name: "quiet keys", flags: []string{"--keys-only", "--quiet"}, keys: true, quiet: true},
			} {
				t.Run(format+"/"+destination+"/"+tc.name, func(t *testing.T) {
					f := newResumptionFixture(t)
					f.reject[resumeKeyB] = 400
					f.reject[resumeKeyC] = 400
					level := "info"
					if tc.configDebug {
						level = "debug"
					}
					workers := "1"
					if destination == "inherited" {
						workers = "2"
					}
					args := append([]string{"--config", completionLogConfig(t, f.server.URL, format, level), "--auto-confirm", "resume", "pi", "--key", resumeKeyA + "," + resumeKeyB + "," + resumeKeyC, "--workers", workers, "--no-wait"}, tc.flags...)
					out, stderr, err := runResumptionTerminal(t, args, destination, nil)
					assertUserTaskCompletionExit(t, err, 2)
					calls, writes, reads := f.snapshot()
					require.Len(t, calls, 3)
					require.Len(t, writes, 3)
					require.Empty(t, reads)
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					errors, debugs, infos := []string{}, []string{}, []string{}
					for _, record := range records {
						switch record[0] {
						case "ERROR":
							errors = append(errors, record[1])
						case "INFO":
							infos = append(infos, record[1])
						case "DEBUG":
							debugs = append(debugs, record[1])
						default:
							t.Errorf("unexpected diagnostic severity: %v", record)
						}
					}
					if tc.json {
						require.Empty(t, errors)
						env := requireSingleJSONObjectDocument(t, out)
						require.Equal(t, "invalid_input", env["class"])
						require.Equal(t, "invalid", env["outcome"])
						require.Contains(t, requireJSONObject(t, env["detail"])["message"], "body-only-secret")
						items := requireJSONItems(t, requireJSONObject(t, env["payload"])["items"], 3)
						require.Equal(t, "accepted", requireJSONObject(t, items[0])["submission"])
						for _, item := range items[1:] {
							r := requireJSONObject(t, item)
							require.Equal(t, "rejected", r["submission"])
							require.Equal(t, "failed", r["status"])
							require.Contains(t, r["error"], "body-only-secret")
						}
					} else {
						require.Equal(t, []string{"resume process instances: rejected: 2"}, errors)
						if tc.keys {
							require.Equal(t, resumeKeyA+"\n", out)
						} else {
							require.Empty(t, out)
						}
						for _, bad := range []string{"POST", f.server.URL, "/v2/", "private.invalid", "body-only-secret", "private-cursor", "resumption-token-secret", "ERROR", "DEBUG"} {
							require.NotContains(t, out, bad)
						}
					}

					if tc.json || tc.keys || tc.quiet {
						require.Empty(t, infos)
					} else {
						want := []string{"resumption: submitted 1/3 process instance(s); confirmation not requested; failed 2"}
						if tc.verbose {
							want = append(want,
								resumeKeyA+": submission: accepted; confirmation: not requested; last state: not observed",
								resumeKeyB+": submission: rejected; confirmation: not requested; last state: not observed",
								resumeKeyC+": submission: rejected; confirmation: not requested; last state: not observed")
						}
						require.Equal(t, want, infos)
					}
					for _, bad := range []string{"private.invalid", "body-only-secret", "private-cursor", "resumption-token-secret", "resumption-cookie-secret"} {
						require.NotContains(t, stderr, bad)
					}
					if tc.debug {
						mutations := 0
						for _, msg := range debugs {
							if strings.Contains(msg, "api #") && strings.Contains(msg, "POST /v2/process-instances/") {
								mutations++
							}
						}
						require.Equal(t, 3, mutations)
					} else {
						require.Empty(t, debugs)
						if !tc.verbose || tc.keys || tc.quiet {
							for _, key := range []string{resumeKeyA, resumeKeyB, resumeKeyC} {
								require.NotContains(t, stderr, key)
							}
						}
						for _, bad := range []string{"POST", f.server.URL, "/v2/"} {
							require.NotContains(t, stderr, bad)
						}
					}
				})
			}
		}
	}
}

// Success, preview and verbose output must obey the same logger destination,
// severity and level controls as failures, including with redirected stdout.
func TestResumeProcessInstanceSummaryLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, level := range []string{"info", "warn"} {
				for _, tc := range []struct {
					name                    string
					flags                   []string
					summary                 string
					verbose, machine, quiet bool
				}{
					{name: "confirmed", summary: "resumption: resumed 1/1 process instance(s)"},
					{name: "no-wait", flags: []string{"--no-wait"}, summary: "resumption: submitted 1/1 process instance(s); confirmation not requested"},
					{name: "preview", flags: []string{"--dry-run"}, summary: "dry run: resume process-instance; selected: 1; eligibility not checked; no changes applied"},
					{name: "verbose", flags: []string{"--verbose"}, summary: "resumption: resumed 1/1 process instance(s)", verbose: true},
					{name: "verbose preview", flags: []string{"--verbose", "--dry-run"}, summary: "dry run: resume process-instance; selected: 1; eligibility not checked; no changes applied", verbose: true},
					{name: "quiet", flags: []string{"--quiet"}, quiet: true},
					{name: "json", flags: []string{"--json"}, machine: true},
					{name: "quiet json", flags: []string{"--quiet", "--json"}, machine: true},
					{name: "keys", flags: []string{"--keys-only"}, machine: true},
				} {
					t.Run(format+"/"+destination+"/"+level+"/"+tc.name, func(t *testing.T) {
						f := newResumptionFixture(t)
						args := []string{"--config", completionLogConfig(t, f.server.URL, format, level), "resume", "pi", "--key", resumeKeyA, "--auto-confirm"}
						args = append(args, tc.flags...)
						stdout, stderr, err := runResumptionTerminal(t, args, destination, nil)
						require.NoError(t, err, stderr)
						records := bulkCompletionDiagnosticRecords(t, stderr, format)
						if tc.machine {
							require.Empty(t, records)
							if tc.name == "keys" {
								require.Equal(t, resumeKeyA+"\n", stdout)
							} else {
								requireSingleJSONObjectDocument(t, stdout)
							}
							return
						}
						require.Empty(t, stdout, "human summaries must use the configured logger")
						if level == "warn" || tc.quiet {
							require.Empty(t, records)
							return
						}
						want := [][2]string{{"INFO", tc.summary}}
						if tc.verbose {
							row := resumeKeyA + ": submission: accepted; confirmation: confirmed; last state: ACTIVE"
							if tc.name == "verbose preview" {
								row = resumeKeyA + ": submission: not submitted; confirmation: not requested; last state: not observed"
							}
							want = append(want, [2]string{"INFO", row})
						}
						require.Equal(t, want, records)
					})
				}
			}
		}
	}
}
