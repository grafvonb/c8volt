// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// newOpsBusinessIDOutcomeFixture owns three independent roots and two repeated
// backend failures; an empty run rejects every lookup and mutation.
func newOpsBusinessIDOutcomeFixture(t *testing.T, workflow string, empty, fail bool) (*httptest.Server, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	reads, writes := &atomic.Int32{}, &atomic.Int32{}
	keys := []string{"2251799813685101", "2251799813685102", "2251799813685103"}
	instance := func(key string) map[string]any {
		item := map[string]any{"processInstanceKey": key, "processDefinitionKey": "9001", "processDefinitionId": "order", "state": "COMPLETED", "tenantId": "<default>", "endDate": "2026-01-01T00:00:00Z"}
		if key == keys[0] {
			item["businessId"] = "new"
		}
		if key == keys[1] {
			item["businessId"] = opsBusinessIDOpaque
		}
		if workflow == "orphan" {
			item["parentProcessInstanceKey"] = "2251799813685999"
		}
		return item
	}
	srv := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, "8.10") {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/deletion") {
			require.False(t, empty)
			writes.Add(1)
			if !fail || strings.Contains(r.URL.Path, keys[0]) {
				fmt.Fprint(w, `{}`)
				return
			}
			w.WriteHeader(400)
			fmt.Fprint(w, `{"detail":"POST https://private.invalid/secret body-only-secret resource-secret-2251799813685102 token=password"}`)
			return
		}
		reads.Add(1)
		if r.Method == "GET" {
			require.False(t, empty)
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if key == "2251799813685999" {
				w.WriteHeader(404)
				fmt.Fprint(w, `{}`)
				return
			}
			require.Contains(t, keys, key)
			require.NoError(t, json.NewEncoder(w).Encode(instance(key)))
			return
		}
		require.Equal(t, "POST", r.Method)
		require.Contains(t, []string{"/v2/process-instances/search", "/v2/incidents/search"}, r.URL.Path)
		var body struct {
			Filter map[string]any `json:"filter"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		items := []map[string]any{}
		_, exactParent := body.Filter["parentProcessInstanceKey"].(string)
		if !empty && !exactParent {
			for i, key := range keys {
				if r.URL.Path == "/v2/incidents/search" {
					items = append(items, map[string]any{"incidentKey": fmt.Sprint(2251799813685201 + i), "processInstanceKey": key, "state": "ACTIVE", "errorType": "JOB_NO_RETRIES", "errorMessage": "failed"})
				} else {
					items = append(items, instance(key))
				}
			}
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "page": map[string]any{"totalItems": len(items)}}))
	}))
	t.Cleanup(srv.Close)
	return srv, reads, writes
}

// opsBusinessIDCommand selects only existing operations and their required flags.
func opsBusinessIDCommand(workflow string) []string {
	switch workflow {
	case "retention":
		return []string{"ops", "execute", "retention-policy", "--retention-days", "1"}
	case "incident":
		return []string{"ops", "purge", "process-instances-with-incidents"}
	default:
		return []string{"ops", "purge", "orphan-process-instances"}
	}
}

// opsBusinessIDWithoutPreflightWarning preserves the established destructive
// warning while requiring every new result/detail record to honor INFO policy.
func opsBusinessIDWithoutPreflightWarning(t *testing.T, records [][2]string, workflow string) [][2]string {
	t.Helper()
	warning := map[string]string{
		"retention": "retention cleanup is destructive: retention cleanup will discover eligible process instances, validate delete impact, and delete confirmed roots",
		"incident":  "incident purge is destructive: incident purge will discover matching incidents, validate delete impact, and delete confirmed roots",
		"orphan":    "orphan purge is destructive: orphan purge will discover child candidates, check parent existence, validate delete impact, and delete confirmed roots",
	}[workflow]
	var out [][2]string
	seen := false
	for _, record := range records {
		if record[0] == "WARN" {
			require.False(t, seen)
			require.Equal(t, warning, record[1])
			seen = true
		} else {
			out = append(out, record)
		}
	}
	return out
}

// TestOpsBusinessIDPartialFailure requires one configured aggregate ERROR and one
// machine envelope retaining observed plans and partial outcomes, plus saved files.
func TestOpsBusinessIDPartialFailure(t *testing.T) {
	for _, workflow := range []string{"retention", "incident", "orphan"} {
		for _, format := range []string{"plain-time", "json"} {
			for _, mode := range []string{"normal", "verbose", "quiet", "json", "debug", "config-debug", "markdown"} {
				t.Run(workflow+"/"+format+"/"+mode, func(t *testing.T) {
					srv, reads, writes := newOpsBusinessIDOutcomeFixture(t, workflow, false, true)
					level := "info"
					if mode == "config-debug" {
						level = "debug"
					}
					extension := "json"
					if mode == "markdown" {
						extension = "md"
					}
					path := filepath.Join(t.TempDir(), "audit."+extension)
					args := append([]string{"--config", runBusinessIDConfig(t, srv.URL, format, level)}, opsBusinessIDCommand(workflow)...)
					args = append(args, "--auto-confirm", "--no-wait", "--workers", "1", "--report-file", path)
					if mode != "normal" && mode != "config-debug" && mode != "markdown" {
						args = append(args, "--"+mode)
					}
					stdout, stderr, err := runBusinessIDCommand(t, args, "inherited", false)
					var exit *exec.ExitError
					require.ErrorAs(t, err, &exit)
					require.Equal(t, 2, exit.ExitCode())
					require.EqualValues(t, 3, writes.Load())
					wantReads := 16
					if workflow == "orphan" {
						wantReads = 22
					}
					require.EqualValues(t, wantReads, reads.Load())
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					var errors, debugs []string
					for _, record := range records {
						if record[0] == "ERROR" {
							errors = append(errors, record[1])
						}
						if record[0] == "DEBUG" {
							debugs = append(debugs, record[1])
						}
						if record[0] != "DEBUG" {
							for _, raw := range []string{"https://private.invalid", "POST ", "body-only-secret", "resource-secret-"} {
								require.NotContains(t, record[1], raw)
							}
						}
					}
					expected := map[string]*string{"2251799813685101": toolx.Ptr("new"), "2251799813685102": toolx.Ptr(opsBusinessIDOpaque), "2251799813685103": nil}
					if mode == "json" {
						var envelope map[string]any
						dec := json.NewDecoder(strings.NewReader(stdout))
						require.NoError(t, dec.Decode(&envelope))
						require.ErrorIs(t, dec.Decode(new(any)), io.EOF)
						require.Equal(t, "invalid", envelope["outcome"])
						require.Positive(t, assertOpsBusinessIDObservations(t, envelope["payload"], expected))
						require.Empty(t, errors)
					} else {
						require.Empty(t, stdout)
						require.Len(t, errors, 1)
						require.Contains(t, errors[0], "selected 3, accepted 1")
						if mode == "quiet" {
							require.Len(t, records, 1)
						}
					}
					if mode == "debug" || mode == "config-debug" {
						require.NotEmpty(t, debugs)
						require.Contains(t, strings.Join(debugs, "\n"), "POST")
						require.NotContains(t, stderr, "token=password")
					} else {
						require.Empty(t, debugs)
						for _, raw := range []string{"https://private.invalid", "POST ", "body-only-secret", "resource-secret-"} {
							require.NotContains(t, stderr, raw)
						}
					}
					data, err := os.ReadFile(path)
					require.NoError(t, err)
					if mode == "markdown" {
						require.Contains(t, string(data), `\|\<tag\>`)
						require.Contains(t, string(data), "- Run Errors:")
					} else {
						var report any
						require.NoError(t, json.Unmarshal(data, &report))
						require.Positive(t, assertOpsBusinessIDObservations(t, report, expected))
					}
				})
			}
		}
	}
}

// TestOpsBusinessIDEmptyScopes keeps authoritative empty discovery prompt-free
// with real terminal stdin, exact empty keys and one successful JSON envelope.
func TestOpsBusinessIDEmptyScopes(t *testing.T) {
	for _, workflow := range []string{"retention", "incident", "orphan"} {
		for _, dry := range []bool{false, true} {
			modes := [][]string{nil, {"--json"}, {"--quiet"}, {"--quiet", "--json"}, {"--auto-confirm", "--no-wait"}, {"--automation", "--json", "--no-wait"}}
			if workflow == "orphan" {
				modes = append(modes, []string{"--keys-only"}, []string{"--quiet", "--keys-only"})
			}
			for _, flags := range modes {
				t.Run(fmt.Sprint(workflow, dry, flags), func(t *testing.T) {
					srv, reads, writes := newOpsBusinessIDOutcomeFixture(t, workflow, true, false)
					path := filepath.Join(t.TempDir(), "audit.json")
					args := append([]string{"--config", runBusinessIDConfig(t, srv.URL, "plain-time", "info")}, opsBusinessIDCommand(workflow)...)
					args = append(args, "--report-file", path)
					if dry {
						args = append(args, "--dry-run")
					}
					args = append(args, flags...)
					stdout, stderr, err := runBusinessIDCommand(t, args, "configured", false)
					require.NoError(t, err, stdout+stderr)
					require.EqualValues(t, 1, reads.Load())
					require.Zero(t, writes.Load())
					joined := strings.Join(flags, " ")
					if strings.Contains(joined, "--json") {
						payload := decodeJobTaskBusinessIDEnvelope(t, stdout)
						require.Equal(t, "planned", payload["outcome"])
						require.NotContains(t, requireJSONObject(t, payload["deletion"]), "submitted")
					} else {
						require.Empty(t, stdout)
					}
					require.NotContains(t, stderr, "Do you want to proceed?")
					records := opsBusinessIDWithoutPreflightWarning(t, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"), workflow)
					if strings.Contains(joined, "--quiet") || strings.Contains(joined, "--json") || strings.Contains(joined, "--keys-only") {
						require.Empty(t, records)
					} else {
						var messages []string
						for _, r := range records {
							require.Equal(t, "INFO", r[0])
							messages = append(messages, r[1])
						}
						require.NotEmpty(t, messages)
						require.NotContains(t, strings.Join(messages, "\n"), "business-id:")
						require.Contains(t, strings.Join(messages, "\n"), "outcome: planned;")
					}
					data, err := os.ReadFile(path)
					require.NoError(t, err)
					require.NotContains(t, string(data), "businessId")
				})
			}
		}
	}
}

// TestOpsBusinessIDSuccessfulMutation verifies INFO routing and suppression on
// actual accepted mutation paths, retaining observed report data and no new reads.
func TestOpsBusinessIDSuccessfulMutation(t *testing.T) {
	for _, workflow := range []string{"retention", "incident", "orphan"} {
		for _, format := range []string{"plain-time", "json"} {
			for _, mode := range []string{"verbose", "warn", "quiet", "json"} {
				t.Run(workflow+format+mode, func(t *testing.T) {
					srv, reads, writes := newOpsBusinessIDOutcomeFixture(t, workflow, false, false)
					level := "info"
					if mode == "warn" {
						level = "warn"
					}
					path := filepath.Join(t.TempDir(), "audit.md")
					args := append([]string{"--config", runBusinessIDConfig(t, srv.URL, format, level)}, opsBusinessIDCommand(workflow)...)
					args = append(args, "--auto-confirm", "--no-wait", "--report-file", path)
					if mode != "warn" {
						args = append(args, "--"+mode)
					}
					stdout, stderr, err := runBusinessIDCommand(t, args, "inherited", false)
					require.NoError(t, err, stdout+stderr)
					require.EqualValues(t, 3, writes.Load())
					wantReads := 16
					if workflow == "orphan" {
						wantReads = 22
					}
					require.EqualValues(t, wantReads, reads.Load())
					if mode == "json" {
						decodeJobTaskBusinessIDEnvelope(t, stdout)
					} else {
						require.Empty(t, stdout)
					}
					records := opsBusinessIDWithoutPreflightWarning(t, bulkCompletionDiagnosticRecords(t, stderr, format), workflow)
					if mode != "verbose" {
						require.Empty(t, records)
					} else {
						var messages []string
						for _, r := range records {
							require.Equal(t, "INFO", r[0])
							messages = append(messages, r[1])
						}
						require.Contains(t, strings.Join(messages, "\n"), "business-id:"+strconv.Quote(opsBusinessIDOpaque))
					}
					data, err := os.ReadFile(path)
					require.NoError(t, err)
					require.Contains(t, string(data), `\|\<tag\>`)
				})
			}
		}
	}
}
