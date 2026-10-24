// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// completionLogConfig configures the real command logger without inheriting a
// user profile; all backend traffic remains on the supplied test-owned server.
func completionLogConfig(t *testing.T, url, format, level string) string {
	t.Helper()
	path := writeTestConfigForVersion(t, url, "8.10")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	require.NoError(t, err)
	_, err = fmt.Fprintf(file, "log:\n  level: %s\n  format: %s\n", level, format)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	return path
}

// TestUpdateUserTaskCompletionDiagnosticLogging checks logger framing, severity,
// effective destination and absence of raw multi-resource error chains on both
// streams. Direct prints and raw-error logging must fail these assertions.
func TestUpdateUserTaskCompletionDiagnosticLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, tc := range []struct {
				name                            string
				flags                           []string
				debug, configDebug, quiet, json bool
			}{
				{name: "normal"}, {name: "verbose", flags: []string{"--verbose"}},
				{name: "quiet", flags: []string{"--quiet"}, quiet: true},
				{name: "debug flag", flags: []string{"--debug"}, debug: true},
				{name: "debug config", debug: true, configDebug: true},
				{name: "quiet beats debug", flags: []string{"--debug", "--quiet"}, configDebug: true, quiet: true},
				{name: "JSON partial", flags: []string{"--json"}, json: true},
				{name: "quiet JSON", flags: []string{"--quiet", "--json"}, json: true, quiet: true},
			} {
				t.Run(format+"/"+destination+"/"+tc.name, func(t *testing.T) {
					server, calls := newUserTaskCompletionServer(t, func(w http.ResponseWriter, r *http.Request, key string) {
						if key == completionKeyA {
							w.WriteHeader(204)
							return
						}
						w.Header().Set("Content-Type", "application/problem+json")
						w.Header().Set("Set-Cookie", "session=completion-cookie-secret")
						w.WriteHeader(400)
						fmt.Fprintf(w, `{"detail":"POST https://private.invalid/secret?leaseToken=completion-token-secret body-only-secret resource=%s"}`, key)
					})
					level := "info"
					if tc.configDebug {
						level = "debug"
					}
					workers := "1"
					if destination == "inherited" {
						workers = "2"
					}
					args := append([]string{"--config", completionLogConfig(t, server.URL, format, level), "--auto-confirm", "update", "ut", "--complete", "--key", completionKeyA + "," + completionKeyB + ",2251799813685251", "--workers", workers, "--vars", `{"password":"completion-variable-secret"}`}, tc.flags...)
					stdout, stderr, err := runUserTaskCompletionTerminal(t, args, destination, nil)
					assertUserTaskCompletionExit(t, err, 2)
					require.Len(t, calls.Snapshot(), 3)
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
							t.Errorf("unexpected diagnostic severity %s: %s", record[0], record[1])
						}
					}
					if tc.json {
						require.Empty(t, infos)
						require.Empty(t, errors)
						envelope := decodeUserTaskCompletionEnvelope(t, stdout)
						require.Equal(t, "invalid_input", envelope["class"])
						require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "body-only-secret")
						payload := requireJSONObject(t, envelope["payload"])
						require.Len(t, requireJSONItems(t, payload["items"], 3), 3)
						require.Equal(t, float64(1), requireJSONObject(t, payload["counts"])["accepted"])
					} else {
						require.Equal(t, []string{"user-task completion: bad request (HTTP 400): 2"}, errors)
						for _, key := range []string{completionKeyA, completionKeyB, "2251799813685251"} {
							require.NotContains(t, strings.Join(errors, "\n"), key)
						}
						require.Empty(t, stdout)
						if tc.quiet {
							require.Empty(t, infos)
						} else {
							expected := "update user-task: 3 requested; 1 completed, 2 failed\n"
							if tc.name == "verbose" {
								expected += "updated user-task " + completionKeyA + ": completion confirmed; HTTP 204 No Content\n" + completionKeyB + ": failed; submission: rejected; HTTP 400 Bad Request\n2251799813685251: failed; submission: rejected; HTTP 400 Bad Request\n"
							}
							require.Equal(t, strings.TrimSuffix(expected, "\n"), strings.Join(infos, "\n"))
						}
						for _, bad := range []string{server.URL, "/v2/user-tasks/", "POST", "body-only-secret", "private.invalid", "completion-token-secret", "completion-variable-secret", "ERROR", "DEBUG"} {
							require.NotContains(t, stdout, bad)
						}
					}
					for _, bad := range []string{"body-only-secret", "private.invalid", "leaseToken", "completion-token-secret", "completion-cookie-secret", "completion-variable-secret"} {
						require.NotContains(t, stderr, bad)
					}
					if tc.debug {
						exchanges, lookups := 0, 0
						for _, message := range debugs {
							if strings.Contains(message, "api #") {
								if strings.Contains(message, "GET /v2/user-tasks/") {
									lookups++
									continue
								}
								exchanges++
								require.Contains(t, message, "POST /v2/user-tasks/")
								require.Contains(t, message, "/completion: status=")
							}
						}
						require.Equal(t, 3, exchanges)
						require.Equal(t, 3, lookups)
					} else {
						require.Empty(t, debugs)
						for _, bad := range []string{server.URL, "/v2/user-tasks/", "POST"} {
							require.NotContains(t, stderr, bad)
						}
					}
				})
			}
		}
	}
}

// TestUpdateUserTaskCompletionDiagnosticRetries verifies real retry exchanges
// for sequential and bounded workers, both DEBUG entry points and redaction,
// while comparing INFO messages with the identical non-debug execution.
func TestUpdateUserTaskCompletionDiagnosticRetries(t *testing.T) {
	for _, workers := range []string{"1", "2"} {
		for _, format := range []string{"plain-time", "json"} {
			for _, status := range []int{429, 503} {
				t.Run(workers+"/"+format+"/"+fmt.Sprint(status), func(t *testing.T) {
					baseline := ""
					for _, mode := range []string{"normal", "flag", "configured"} {
						var mu sync.Mutex
						attempts := map[string]int{}
						server, calls := newUserTaskCompletionServer(t, func(w http.ResponseWriter, r *http.Request, key string) {
							mu.Lock()
							attempts[key]++
							attempt := attempts[key]
							mu.Unlock()
							if attempt == 1 {
								w.Header().Set("Retry-After", "0")
								w.Header().Set("Content-Type", "application/problem+json")
								w.Header().Set("Set-Cookie", "token=retry-cookie-secret")
								w.WriteHeader(status)
								fmt.Fprint(w, `{"title":"RESOURCE_EXHAUSTED","detail":"RESOURCE_EXHAUSTED body-only-secret"}`)
								return
							}
							w.WriteHeader(204)
						})
						level := "info"
						if mode == "configured" {
							level = "debug"
						}
						args := []string{"--auto-confirm", "update", "ut", "--complete", "--key", completionKeyA + "," + completionKeyB, "--workers", workers, "--vars", `{"password":"retry-payload-secret"}`}
						if mode == "flag" {
							args = append(args, "--debug")
						}
						stdout, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, server.URL, format, level), "", args...)
						require.NoError(t, err, stderr)
						require.Len(t, calls.Snapshot(), 4)
						mu.Lock()
						require.Equal(t, map[string]int{completionKeyA: 2, completionKeyB: 2}, attempts)
						mu.Unlock()
						require.Empty(t, stdout)
						records := bulkCompletionDiagnosticRecords(t, stderr, format)
						info := completionInfoText(records)
						require.Equal(t, "update user-task: 2 requested; 2 completed\n", info)
						if mode == "normal" {
							baseline = info
							require.Equal(t, completionInfoRecords(info), records)
						} else {
							require.Equal(t, baseline, info)
							exchanges, lookups := 0, 0
							for _, record := range records {
								if record[0] == "INFO" {
									continue
								}
								require.Equal(t, "DEBUG", record[0])
								if strings.Contains(record[1], "api #") {
									if strings.Contains(record[1], "GET /v2/user-tasks/") {
										lookups++
										continue
									}
									exchanges++
									require.Contains(t, record[1], "POST /v2/user-tasks/")
								}
							}
							require.Equal(t, 4, exchanges)
							require.Equal(t, 2, lookups)
							require.Contains(t, stderr, fmt.Sprintf("status=%d", status))
							require.Contains(t, stderr, "status=204")
						}
						for _, bad := range []string{"body-only-secret", "retry-cookie-secret", "retry-payload-secret"} {
							require.NotContains(t, stdout+stderr, bad)
						}
					}
				})
			}
		}
	}
}
