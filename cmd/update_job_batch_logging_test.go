// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestBulkCompletionFailureSummaryBounds keeps heterogeneous backlog failures
// bounded and excludes untrusted diagnostic/reason text from normal logs.
func TestBulkCompletionFailureSummaryBounds(t *testing.T) {
	raw := "POST https://private.invalid/secret body-only-secret\nforged log record"
	status := http.StatusBadRequest
	report := job.CompletionBatchReport{
		Status: job.CompletionBatchFinished,
		Counts: job.CompletionCounts{Unsatisfied: 6},
	}
	for _, reason := range []job.CompletionOutcomeReason{
		job.CompletionOutcomeRejected, job.CompletionOutcomeRejected,
		job.CompletionOutcomeScopeChanged, job.CompletionOutcomeOtherTerminal,
		job.CompletionOutcomeMissing, job.CompletionOutcomeReason(raw),
	} {
		report.Outcomes = append(report.Outcomes, job.CompletionOutcome{
			Disposition: job.CompletionDispositionUnsatisfied, Reason: reason,
			Submission: job.CompletionSubmission{HTTPStatus: &status, Diagnostic: raw},
		})
	}
	runErr := errors.Join(ferrors.ErrInvalidInput, errors.New(raw))
	require.Equal(t, "job completion: 6 unsatisfied job(s); bad request (HTTP 400): 2; selection changed: 1; already in another terminal state: 1; other failures: 2", formatJobCompletionFailure(report, runErr))
	// Unknown reasons must also stay safe when they are among the visible groups.
	report.Outcomes = report.Outcomes[5:]
	report.Counts.Unsatisfied = 1
	require.Equal(t, "job completion: 1 unsatisfied job(s); completion unresolved: 1", formatJobCompletionFailure(report, runErr))
	report.Status = job.CompletionBatchInterrupted
	report.Counts.NotStarted = 5
	require.Equal(t, "job completion: interrupted; 1 unsatisfied job(s); 5 not started; completion unresolved: 1", formatJobCompletionFailure(report, runErr))
}

// TestBulkCompletionDiagnosticLogging exercises real command failures and logger
// configuration against a test-owned server, including terminal writer routing.
func TestBulkCompletionDiagnosticLogging(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, format := range []string{"plain-time", "json"} {
			for i, tc := range []struct {
				name                                                                    string
				flags                                                                   []string
				configuredDebug, debug, json, quiet, partial, preparation, suppressExit bool
			}{
				{name: "normal"},
				{name: "verbose", flags: []string{"--verbose"}},
				{name: "quiet", flags: []string{"--quiet"}, quiet: true},
				{name: "debug flag", flags: []string{"--debug"}, debug: true},
				{name: "debug configuration", configuredDebug: true, debug: true},
				{name: "quiet overrides debug", flags: []string{"--quiet", "--debug"}, configuredDebug: true, quiet: true},
				{name: "JSON", flags: []string{"--json"}, json: true},
				{name: "quiet JSON partial", flags: []string{"--quiet", "--json"}, json: true, quiet: true, partial: true},
				{name: "debug JSON partial", flags: []string{"--json", "--debug"}, json: true, debug: true, partial: true},
				{name: "partial verbose", flags: []string{"--verbose"}, partial: true},
				{name: "preparation", preparation: true},
				{name: "suppressed exit", flags: []string{"--no-err-codes"}, suppressExit: true},
			} {
				t.Run(version+"/"+format+"/"+tc.name, func(t *testing.T) {
					var writes atomic.Int32
					keys := []string{completionAcceptanceKey(0), completionAcceptanceKey(1), completionAcceptanceKey(2), completionAcceptanceKey(3)}
					server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						if r.Method == http.MethodPost && r.URL.Path == "/v2/jobs/search" && !tc.preparation {
							var body struct {
								Filter map[string]any `json:"filter"`
							}
							require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
							if key, ok := body.Filter["jobKey"].(string); ok {
								fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":1}}`, bulkCompletionRow(key, "CREATED"))
								return
							}
							rows := []string{}
							for _, key := range keys {
								rows = append(rows, bulkCompletionRow(key, "CREATED"))
							}
							fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":4}}`, strings.Join(rows, ","))
							return
						}
						if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/completion") {
							writes.Add(1)
							if tc.partial && strings.Contains(r.URL.Path, keys[0]) {
								w.WriteHeader(204)
								return
							}
						} else if !(tc.preparation && r.URL.Path == "/v2/jobs/search") {
							t.Errorf("unexpected endpoint: %s %s", r.Method, r.URL.Path)
						}
						w.WriteHeader(400)
						io.WriteString(w, `{"title":"Bad Request","detail":"Request property [leaseToken] cannot be parsed; body-only-secret https://private.invalid/secret POST"}`)
					}))
					t.Cleanup(server.Close)
					cfg := writeTestConfigForVersion(t, server.URL, version)
					level := "info"
					if tc.configuredDebug {
						level = "debug"
					}
					file, err := os.OpenFile(cfg, os.O_APPEND|os.O_WRONLY, 0600)
					require.NoError(t, err)
					_, err = fmt.Fprintf(file, "log:\n  level: %s\n  format: %s\n", level, format)
					require.NoError(t, err)
					require.NoError(t, file.Close())
					args := []string{"--config", cfg, "--tenant", "tenant-a", "update", "job", "--complete", "--kind", "TASK_LISTENER", "--auto-confirm"}
					args = append(args, tc.flags...)
					destination := "inherited"
					if i%2 == 1 {
						destination = "configured"
					}
					stdout, stderr, err := runBulkCompletionTerminal(t, args, destination, false, false, nil)
					if tc.suppressExit {
						require.NoError(t, err)
					} else {
						var exitErr *exec.ExitError
						require.ErrorAs(t, err, &exitErr)
						require.Equal(t, 2, exitErr.ExitCode())
					}
					accepted, failed := 0, 4
					if tc.partial {
						accepted, failed = 1, 3
					}
					expected := fmt.Sprintf("job completion: %d unsatisfied job(s); bad request (HTTP 400): %d", failed, failed)
					if tc.preparation {
						expected = "job completion: preparation failed (invalid input); no changes applied"
						require.Zero(t, writes.Load())
					} else {
						require.Equal(t, int32(4), writes.Load())
					}
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					errors := []string{}
					debugs := []string{}
					for _, record := range records {
						switch record[0] {
						case "ERROR":
							errors = append(errors, record[1])
						case "DEBUG":
							debugs = append(debugs, record[1])
						default:
							t.Errorf("unexpected log level: %s", record[0])
						}
					}
					if tc.json {
						require.Empty(t, errors)
						envelope := decodeBulkCompletionEnvelope(t, stdout)
						require.Equal(t, "invalid_input", envelope["class"])
						require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "body-only-secret", "machine evidence must remain unchanged")
						if !tc.preparation {
							payload := requireJSONObject(t, envelope["payload"])
							require.Len(t, payload["outcomes"], 4)
							require.Equal(t, float64(accepted), requireJSONObject(t, payload["counts"])["accepted"])
						}
					} else {
						require.Equal(t, []string{expected}, errors)
						if tc.quiet {
							require.Empty(t, stdout)
						} else if !tc.preparation {
							require.Contains(t, stdout, fmt.Sprintf("outcome: failed; accepted %d; observed completed 0; unsatisfied %d; not started 0", accepted, failed))
						}
						for _, bad := range []string{server.URL, "/v2/jobs/", "POST", "body-only-secret", "leaseToken", "ERROR", "DEBUG"} {
							require.NotContains(t, stdout, bad)
						}
					}
					require.NotContains(t, stderr, "body-only-secret")
					require.NotContains(t, stderr, "private.invalid")
					require.NotContains(t, stderr, "leaseToken")
					if tc.debug {
						require.NotEmpty(t, debugs)
						require.Contains(t, strings.Join(debugs, "\n"), "POST /v2/jobs/")
						require.Contains(t, strings.Join(debugs, "\n"), "status=400")
					} else {
						require.Empty(t, debugs)
						for _, bad := range []string{server.URL, "/v2/jobs/", "POST", keys[0], keys[1]} {
							require.NotContains(t, stderr, bad)
						}
					}
				})
			}
		}
	}
}

// bulkCompletionDiagnosticRecords rejects plain writes masquerading as configured
// log records, checking each line rather than matching only an error substring.
func bulkCompletionDiagnosticRecords(t *testing.T, output, format string) [][2]string {
	t.Helper()
	records := [][2]string{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		if format == "json" {
			var record struct {
				Time    string `json:"time"`
				Level   string `json:"level"`
				Message string `json:"msg"`
			}
			require.NoError(t, json.Unmarshal([]byte(line), &record))
			require.NotEmpty(t, record.Time)
			records = append(records, [2]string{record.Level, record.Message})
		} else {
			match := regexp.MustCompile(`^\d{2}:\d{2}:\d{2}\.\d{3} (ERROR|DEBUG|INFO|WARN) (.*)$`).FindStringSubmatch(line)
			require.Len(t, match, 3, "diagnostic bypassed configured logger: %s", line)
			records = append(records, [2]string{match[1], match[2]})
		}
	}
	return records
}
