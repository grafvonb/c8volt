// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSuspendProcessInstanceDiagnostics detects direct-stderr bypass, duplicate errors,
// lost partial payloads and raw joined-error disclosure at the real command boundary.
func TestSuspendProcessInstanceDiagnostics(t *testing.T) {
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
					f := newSuspensionFixture(t)
					f.reject[suspendKeyB] = 400
					f.reject[suspendKeyC] = 400
					level := "info"
					if tc.configDebug {
						level = "debug"
					}
					workers := "1"
					if destination == "inherited" {
						workers = "2"
					}
					args := append([]string{"--config", completionLogConfig(t, f.server.URL, format, level), "--auto-confirm", "suspend", "pi", "--key", suspendKeyA + "," + suspendKeyB + "," + suspendKeyC, "--workers", workers, "--no-wait"}, tc.flags...)
					out, stderr, err := runSuspensionTerminal(t, args, destination, nil)
					assertUserTaskCompletionExit(t, err, 2)
					calls, writes, reads := f.snapshot()
					require.Len(t, calls, 3)
					require.Len(t, writes, 3)
					require.Empty(t, reads)
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					errors, debugs := []string{}, []string{}
					for _, record := range records {
						switch record[0] {
						case "ERROR":
							errors = append(errors, record[1])
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
						require.Equal(t, []string{"suspend process instances: rejected: 2"}, errors)
						if tc.keys {
							require.Equal(t, suspendKeyA+"\n", out)
						} else if tc.quiet {
							require.Empty(t, out)
						} else {
							want := "suspend process instances: 3 selected; accepted: 1, confirmed: 0, failed: 2, skipped: 0\n"
							if tc.verbose {
								want += suspendKeyA + ": submission: accepted; confirmation: not_requested; last state: not observed\n" + suspendKeyB + ": submission: rejected; confirmation: not_requested; last state: not observed\n" + suspendKeyC + ": submission: rejected; confirmation: not_requested; last state: not observed\n"
							}
							require.Equal(t, want, out)
						}
						for _, bad := range []string{"POST", f.server.URL, "/v2/", "private.invalid", "body-only-secret", "private-cursor", "suspension-token-secret", "ERROR", "DEBUG"} {
							require.NotContains(t, out, bad)
						}
					}
					for _, bad := range []string{"private.invalid", "body-only-secret", "private-cursor", "suspension-token-secret", "suspension-cookie-secret"} {
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
						for _, bad := range []string{"POST", f.server.URL, "/v2/", suspendKeyA, suspendKeyB, suspendKeyC} {
							require.NotContains(t, stderr, bad)
						}
					}
				})
			}
		}
	}
}
