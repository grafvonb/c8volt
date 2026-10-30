// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// rejectElementUpdate returns documented rejection evidence containing deliberately unsafe diagnostics.
func rejectElementUpdate(w http.ResponseWriter, key string) bool {
	if key == scopeElementA {
		return false
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("Set-Cookie", "session=element-cookie-secret")
	w.WriteHeader(400)
	fmt.Fprintf(w, `{"type":"about:blank","title":"Bad Request","status":400,"detail":"PUT https://private.invalid/secret?token=element-token-secret body-only-secret resource=%s","instance":"/v2/element-instances/%s/variables"}`, key, key)
	return true
}

// TestElementScopeVariablesUpdateDiagnostics detects logger bypass, duplicate errors, raw aggregate
// disclosure, missing quiet errors and duplicate envelopes through real command execution.
func TestElementScopeVariablesUpdateDiagnostics(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, tc := range []struct {
				name                                  string
				flags                                 []string
				debug, configDebug, quiet, json, keys bool
			}{
				{name: "normal"}, {name: "verbose", flags: []string{"--verbose"}},
				{name: "quiet", flags: []string{"--quiet"}, quiet: true},
				{name: "debug flag", flags: []string{"--debug"}, debug: true},
				{name: "debug config", debug: true, configDebug: true},
				{name: "quiet beats debug", flags: []string{"--debug", "--quiet"}, configDebug: true, quiet: true},
				{name: "JSON", flags: []string{"--json"}, json: true},
				{name: "quiet JSON", flags: []string{"--json", "--quiet"}, json: true, quiet: true},
				{name: "keys", flags: []string{"--keys-only"}, keys: true},
				{name: "quiet keys", flags: []string{"--keys-only", "--quiet"}, keys: true, quiet: true},
			} {
				t.Run(format+"/"+destination+"/"+tc.name, func(t *testing.T) {
					f := newElementUpdateFixture(t, "8.10")
					f.mutationFailure = rejectElementUpdate
					level := "info"
					if tc.configDebug {
						level = "debug"
					}
					workers := "1"
					if destination == "inherited" {
						workers = "2"
					}
					args := append([]string{"--config", completionLogConfig(t, f.server.URL, format, level), "--no-indicator", "--auto-confirm", "update", "ei", "--key", scopeElementA + "," + scopeElementB + "," + scopeElementC, "--vars", `{"password":"element-payload-secret"}`, "--workers", workers, "--no-wait"}, tc.flags...)
					out, stderr, err := runElementScopeTerminal(t, args, destination, nil)
					assertUserTaskCompletionExit(t, err, 2)
					calls, writes, reads := f.snapshot()
					require.Len(t, calls, 9)
					require.Len(t, writes, 1)
					require.Equal(t, map[string]int{scopeElementA: 1, scopeElementB: 1, scopeElementC: 1}, reads)
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					errors, debugs, infos := []string{}, []string{}, []string{}
					for _, record := range records {
						switch record[0] {
						case "ERROR":
							errors = append(errors, record[1])
						case "DEBUG":
							debugs = append(debugs, record[1])
						case "INFO":
							infos = append(infos, record[1])
						default:
							t.Errorf("unexpected severity: %v", record)
						}
					}
					if tc.json {
						require.Empty(t, errors)
						require.Empty(t, infos)
						env := requireSingleJSONObjectDocument(t, out)
						require.Equal(t, "invalid_input", env["class"])
						require.Equal(t, "invalid", env["outcome"])
						require.Contains(t, requireJSONObject(t, env["detail"])["message"], "body-only-secret")
						items := requireJSONItems(t, requireJSONObject(t, env["payload"])["items"], 3)
						require.Equal(t, true, requireJSONObject(t, items[0])["mutationAccepted"])
						require.Equal(t, "skipped", requireJSONObject(t, items[0])["confirmationStatus"])
						for _, item := range items[1:] {
							r := requireJSONObject(t, item)
							require.Equal(t, "rejected", r["submissionStatus"])
							require.Equal(t, "mutation_failed", r["status"])
							require.Contains(t, r["error"], "body-only-secret")
						}
					} else {
						require.Equal(t, []string{"element variable update: mutation rejected: 2"}, errors)
						for _, key := range []string{scopeElementA, scopeElementB, scopeElementC} {
							require.NotContains(t, strings.Join(errors, "\n"), key)
						}
						if tc.keys {
							require.Equal(t, scopeElementA+"\n", out)
						} else {
							require.Empty(t, out)
						}
						if tc.quiet || tc.keys {
							require.Empty(t, infos)
						} else {
							expected := []string{"update element variables: 3 selected; accepted: 1, confirmed: 0, unchanged: 0, failed: 2, skipped: 0"}
							if tc.name == "verbose" {
								expected = append(expected, scopeElementA+": submitted; submission: accepted; confirmation: skipped", scopeElementB+": mutation_failed; submission: rejected; confirmation: not_required", scopeElementC+": mutation_failed; submission: rejected; confirmation: not_required")
							}
							require.Equal(t, expected, infos)
						}
						for _, bad := range []string{"PUT", f.server.URL, "/v2/", "private.invalid", "body-only-secret", "element-token-secret", "element-payload-secret", "ERROR", "DEBUG"} {
							require.NotContains(t, out, bad)
						}
					}
					for _, bad := range []string{"body-only-secret", "private.invalid", "element-token-secret", "element-cookie-secret", "element-payload-secret"} {
						require.NotContains(t, stderr, bad)
					}
					if tc.debug {
						gets, searches, mutations := 0, 0, 0
						for _, msg := range debugs {
							if !strings.Contains(msg, "api #") {
								continue
							}
							switch {
							case strings.Contains(msg, "GET /v2/element-instances/"):
								gets++
							case strings.Contains(msg, "POST /v2/variables/search"):
								searches++
							case strings.Contains(msg, "PUT /v2/element-instances/"):
								mutations++
							}
						}
						require.Equal(t, 3, gets)
						require.Equal(t, 3, searches)
						require.Equal(t, 3, mutations)
					} else {
						require.Empty(t, debugs)
						for _, bad := range []string{"PUT", f.server.URL, "/v2/element-instances/", "/v2/variables/search"} {
							require.NotContains(t, stderr, bad)
						}
					}
				})
			}
		}
	}
}
