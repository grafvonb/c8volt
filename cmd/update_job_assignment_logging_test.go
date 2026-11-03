// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUpdateJobAssignmentScalarLogging checks logger records, destinations,
// suppression, redaction and one-envelope failures through command execution.
func TestUpdateJobAssignmentScalarLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, mode := range []string{"success", "preview", "verbose", "warn", "quiet", "failure", "failure verbose", "failure quiet", "failure debug", "failure configured-debug", "failure json", "failure quiet json"} {
				t.Run(format+"/"+destination+"/"+mode, func(t *testing.T) {
					failed := strings.HasPrefix(mode, "failure")
					status := 204
					if failed {
						status = 400
					}
					url, calls := scalarJobOutcomeFixture(t, "8.10", "CREATED", "completion", status, false)
					cfg := writeTestConfigForVersion(t, url, "8.10")
					level := "info"
					if mode == "warn" {
						level = "warn"
					}
					if strings.Contains(mode, "configured-debug") {
						level = "debug"
					}
					file, err := os.OpenFile(cfg, os.O_APPEND|os.O_WRONLY, 0600)
					require.NoError(t, err)
					_, err = fmt.Fprintf(file, "log:\n  level: %s\n  format: %s\n", level, format)
					require.NoError(t, err)
					require.NoError(t, file.Close())
					args := []string{"--config", cfg, "update", "job", "--key", scalarJobKey, "--complete", "--set-vars", `{"approved":true}`, "--set-bid", "ORDER-123", "--auto-confirm"}
					for _, flag := range []string{"verbose", "quiet", "json"} {
						if strings.Contains(mode, flag) {
							args = append(args, "--"+flag)
						}
					}
					if mode == "failure debug" {
						args = append(args, "--debug")
					}
					if mode == "preview" {
						args = append(args, "--dry-run")
					}
					stdout, stderr, err := runBulkCompletionTerminal(t, args, destination, true, false, nil)
					if failed {
						var e *exec.ExitError
						require.ErrorAs(t, err, &e)
						require.Equal(t, 2, e.ExitCode())
					} else {
						require.NoError(t, err, stderr)
					}
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					var infos, errors, debugs []string
					for _, r := range records {
						switch r[0] {
						case "INFO":
							infos = append(infos, r[1])
						case "ERROR":
							errors = append(errors, r[1])
						case "DEBUG":
							debugs = append(debugs, r[1])
						default:
							t.Errorf("unexpected record %v", r)
						}
					}
					if strings.Contains(mode, "json") {
						envelope := decodeBulkCompletionEnvelope(t, stdout)
						require.Equal(t, "invalid_input", envelope["class"])
						payload := requireJSONObject(t, envelope["payload"])
						require.Equal(t, "rejected", requireJSONObject(t, payload["assignment"])["status"])
						require.Empty(t, infos)
						require.Empty(t, errors)
					} else {
						require.Empty(t, stdout)
						if failed {
							require.Equal(t, []string{"update job: completion request failed: invalid input"}, errors)
						} else {
							require.Empty(t, errors)
						}
						if mode == "warn" || strings.Contains(mode, "quiet") {
							require.Empty(t, infos)
						} else {
							require.NotEmpty(t, infos)
						}
					}
					for _, secret := range []string{"private.invalid", "body-only-secret", "leaseToken"} {
						require.NotContains(t, stderr, secret)
					}
					if strings.Contains(mode, "debug") {
						require.NotEmpty(t, debugs)
						require.Contains(t, strings.Join(debugs, "\n"), "POST /v2/jobs/")
					} else {
						require.Empty(t, debugs)
						require.NotContains(t, stderr, "POST")
						require.NotContains(t, stderr, "/v2/jobs/")
					}
					submissions := 0
					for _, call := range calls.Snapshot() {
						if strings.HasSuffix(call, "/completion") {
							submissions++
						}
					}
					if mode == "preview" {
						require.Zero(t, submissions)
					} else {
						require.Equal(t, 1, submissions)
					}
				})
			}
		}
	}
}
