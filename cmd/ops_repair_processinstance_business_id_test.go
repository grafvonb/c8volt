// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// businessIDRepairFixture owns native paging, reviewed actions and approval-time changes.
type businessIDRepairFixture struct {
	searches atomic.Int32
	approved atomic.Bool
	reads    testx.SafeSlice[string]
	writes   testx.SafeSlice[string]
	filters  testx.SafeSlice[map[string]any]
}

// newBusinessIDRepairFixture returns sparse/duplicate pages and introduces unreviewed work after approval.
func newBusinessIDRepairFixture(t *testing.T, empty bool, failure string) (*businessIDRepairFixture, *httptest.Server) {
	t.Helper()
	f := &businessIDRepairFixture{}
	s := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fail := func() {
			w.Header().Set("Set-Cookie", "secret-cookie")
			w.WriteHeader(400)
			fmt.Fprint(w, `{"detail":"POST https://private.invalid/path?token=secret-token body-only-secret resource-key-secret"}`)
		}
		encode := func(x any) { require.NoError(t, json.NewEncoder(w).Encode(x)) }
		incident := func(k, pi, state string) map[string]any {
			return map[string]any{"incidentKey": k, "processInstanceKey": pi, "rootProcessInstanceKey": pi, "jobKey": "3" + pi, "state": state, "tenantId": "tenant-a"}
		}
		path := r.URL.Path
		if path == "/v2/topology" {
			encode(map[string]any{"gatewayVersion": "8.10.3", "brokers": []any{}})
			return
		}
		if path == "/v2/process-instances/search" {
			n := f.searches.Add(1)
			var b struct {
				Filter map[string]any `json:"filter"`
				Page   map[string]any `json:"page"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
			f.filters.Append(b.Filter)
			require.Equal(t, "tenant-a", b.Filter["tenantId"])
			require.Equal(t, true, b.Filter["hasIncident"])
			require.Contains(t, b.Filter, "businessId")
			if failure == "search" && n == 3 {
				fail()
				return
			}
			items := []map[string]any{}
			pi := func(k string) map[string]any {
				return map[string]any{"processInstanceKey": k, "businessId": b.Filter["businessId"], "state": "ACTIVE", "hasIncident": true, "tenantId": "tenant-a"}
			}
			more := n < 3
			if !empty {
				if n == 1 {
					items = append(items, pi("101"), pi("101"))
				}
				if n == 3 {
					items = append(items, pi("102"))
				}
			}
			if f.approved.Load() {
				items = append(items, pi("999"))
			}
			page := map[string]any{"totalItems": len(items), "hasMoreTotalItems": more}
			if more {
				page["totalItems"] = 10000
			}
			if more {
				page["endCursor"] = fmt.Sprint(n)
			}
			encode(map[string]any{"items": items, "page": page})
			return
		}
		if strings.HasSuffix(path, "/incidents/search") {
			key := strings.Split(path, "/")[3]
			f.reads.Append(key)
			if failure == "planning" && key == "102" {
				fail()
				return
			}
			var b struct {
				Filter map[string]any `json:"filter"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
			require.Equal(t, "tenant-a", b.Filter["tenantId"])
			items := []map[string]any{}
			if failure != "no-incidents" && !(failure == "first-empty" && key == "101") {
				items = append(items, incident("2"+key, key, "ACTIVE"), incident("2"+key, key, "ACTIVE"), incident("old"+key, key, "RESOLVED"), incident("foreign", "other", "ACTIVE"))
			}
			if f.approved.Load() {
				items = append(items, incident("new"+key, key, "ACTIVE"))
			}
			encode(map[string]any{"items": items, "page": map[string]any{"totalItems": len(items), "hasMoreTotalItems": false}})
			return
		}
		if r.Method == http.MethodPut || r.Method == http.MethodPatch || strings.HasSuffix(path, "/resolution") {
			require.Equal(t, int32(3), f.searches.Load(), "mutation preceded complete selection")
			require.GreaterOrEqual(t, len(f.reads.Snapshot()), 2, "mutation preceded complete incident planning")
			f.writes.Append(r.Method + " " + path)
			if strings.Contains(path, "/variables") {
				var b map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
				require.Equal(t, map[string]any{"nested": map[string]any{"value": "frozen"}}, b["variables"])
			}
			if (failure == "write" && strings.HasSuffix(path, "/resolution")) || (failure == "partial" && strings.Contains(path, "/2102/")) || (failure == "variables" && r.Method == http.MethodPut) || (failure == "job" && r.Method == http.MethodPatch) {
				fail()
				return
			}
			w.WriteHeader(204)
			return
		}
		if strings.HasPrefix(path, "/v2/incidents/") {
			if failure == "wait" {
				fail()
				return
			}
			key := strings.Split(path, "/")[3]
			encode(incident(key, strings.TrimPrefix(key, "2"), "RESOLVED"))
			return
		}
		if path == "/v2/jobs/search" {
			var b struct {
				Filter map[string]any `json:"filter"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
			key := b.Filter["jobKey"]
			if m, ok := key.(map[string]any); ok {
				key = m["$eq"]
			}
			encode(map[string]any{"items": []map[string]any{{"jobKey": key, "retries": 1, "state": "CREATED", "tenantId": "tenant-a"}}, "page": map[string]any{"totalItems": 1, "hasMoreTotalItems": false}})
			return
		}
		if strings.HasPrefix(path, "/v2/jobs/") {
			encode(map[string]any{"jobKey": strings.Split(path, "/")[3], "retries": 1, "state": "CREATED"})
			return
		}
		if path == "/v2/variables/search" {
			var b struct {
				Filter map[string]any `json:"filter"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
			scope := b.Filter["scopeKey"]
			encode(map[string]any{"items": []map[string]any{{"tenantId": "tenant-a", "variableKey": "501", "name": "nested", "value": `{"value":"frozen"}`, "scopeKey": scope, "processInstanceKey": scope}}, "page": map[string]any{"totalItems": 1, "hasMoreTotalItems": false}})
			return
		}
		t.Errorf("unexpected owned request: %s %s", r.Method, path)
		w.WriteHeader(500)
	}))
	t.Cleanup(s.Close)
	return f, s
}

// businessIDRepairArgs uses isolated configuration and deterministic worker order.
func businessIDRepairArgs(t *testing.T, s *httptest.Server, format, level string, flags ...string) []string {
	t.Helper()
	return append([]string{"--config", runBusinessIDConfig(t, s.URL, format, level), "--tenant", "tenant-a", "ops", "repair", "pi", "--bid", "ORDER-123", "--workers", "1"}, flags...)
}

// TestBusinessIDOperationsRepairScope verifies complete distinct selection, exact actions, preserved criteria and no-op status.
func TestBusinessIDOperationsRepairScope(t *testing.T) {
	for _, dry := range []bool{false, true} {
		for _, noWait := range []bool{false, true} {
			t.Run(fmt.Sprint(dry, noWait), func(t *testing.T) {
				f, s := newBusinessIDRepairFixture(t, false, "")
				flags := []string{"--auto-confirm", "--json", "--set-vars", `{"nested":{"value":"frozen"}}`, "--job-timeout", "1m"}
				if dry {
					flags = append(flags, "--dry-run")
				}
				if noWait {
					flags = append(flags, "--no-wait")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDRepairArgs(t, s, "plain-time", "info", flags...), "configured", false)
				require.NoError(t, err, out+stderr)
				require.Empty(t, stderr)
				e := decodeBusinessIDCancelEnvelope(t, out)
				p := e["payload"].(map[string]any)
				require.Equal(t, "search", p["request"].(map[string]any)["discoveryMode"])
				frozen := p["frozenSet"].(map[string]any)
				require.Equal(t, []any{"101", "102"}, frozen["processInstanceKeys"])
				require.Equal(t, []any{"2101", "2102"}, frozen["incidentKeys"])
				require.Contains(t, out, `"businessId": "ORDER-123"`)
				require.NotContains(t, out, `"FrozenPlan"`)
				require.Equal(t, int32(3), f.searches.Load())
				require.Equal(t, []string{"101", "102"}, f.reads.Snapshot())
				if dry {
					require.Empty(t, f.writes.Snapshot())
					require.Equal(t, "succeeded", e["outcome"])
				} else {
					require.Equal(t, []string{"PUT /v2/element-instances/101/variables", "PUT /v2/element-instances/102/variables", "PATCH /v2/jobs/3101", "PATCH /v2/jobs/3101", "POST /v2/incidents/2101/resolution", "PATCH /v2/jobs/3102", "PATCH /v2/jobs/3102", "POST /v2/incidents/2102/resolution"}, f.writes.Snapshot())
				}
			})
		}
	}
}

// TestBusinessIDOperationsRepairEmpty uses real terminal stdin to prove authoritative empty scopes never prompt or mutate.
func TestBusinessIDOperationsRepairEmpty(t *testing.T) {
	for _, noIncidents := range []bool{false, true} {
		for _, dry := range []bool{false, true} {
			for _, mode := range []string{"human", "quiet", "json", "quiet-json"} {
				for _, control := range []string{"", "auto-confirm", "automation", "no-wait"} {
					t.Run(fmt.Sprint(noIncidents, dry, mode, control), func(t *testing.T) {
						failure := ""
						if noIncidents {
							failure = "no-incidents"
						}
						f, s := newBusinessIDRepairFixture(t, !noIncidents, failure)
						flags := []string{}
						if dry {
							flags = append(flags, "--dry-run")
						}
						if control != "" {
							flags = append(flags, "--"+control)
						}
						if strings.Contains(mode, "json") {
							flags = append(flags, "--json")
						}
						if strings.HasPrefix(mode, "quiet") {
							flags = append(flags, "--quiet")
						}
						out, stderr, err := runBusinessIDCommand(t, businessIDRepairArgs(t, s, "plain-time", "info", flags...), "inherited", false)
						require.NoError(t, err, out+stderr)
						require.Empty(t, f.writes.Snapshot())
						require.NotContains(t, stderr, "Do you want")
						require.Equal(t, int32(3), f.searches.Load())
						if noIncidents {
							require.Len(t, f.reads.Snapshot(), 2)
						} else {
							require.Empty(t, f.reads.Snapshot())
						}
						if strings.Contains(mode, "json") {
							e := decodeBusinessIDCancelEnvelope(t, out)
							require.Equal(t, "succeeded", e["outcome"])
							p := e["payload"].(map[string]any)
							require.Equal(t, "planned", p["outcome"])
							require.NotContains(t, p, "plan")
							require.NotContains(t, p["frozenSet"], "incidentKeys")
							require.Empty(t, stderr)
						} else {
							require.Empty(t, out)
							if mode == "quiet" {
								require.Empty(t, stderr)
							} else {
								records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
								require.NotEmpty(t, records)
								var messages []string
								for _, r := range records {
									require.Equal(t, "INFO", r[0])
									messages = append(messages, r[1])
								}
								header := "repair process-instance incidents"
								preview := "repair plan: skipped"
								if dry {
									header = "dry run: repair process-instance incidents"
									preview = "repair preview: skipped (no active incident targets)"
								}
								var expected []string
								if control != "automation" {
									expected = append(expected, "configured tenant: none", `--tenant "tenant-a" sets the tenant filter`, "selection scope: tenant-a only")
								}
								expected = append(expected, header)
								if noIncidents && control != "automation" {
									expected = append(expected, "affected tenants: tenant-a")
								}
								selected := 0
								if noIncidents {
									selected = 2
								}
								expected = append(expected, `selection filters: {businessId="ORDER-123", hasIncident=true}`, fmt.Sprintf("selected process instances: %d", selected), "repairable process instances: 0", "active incidents: 0", preview)
								outcome := "outcome: planned; no changes applied"
								if noIncidents {
									expected = append(expected, "skipped process instances: 2 without active incidents")
									outcome += "; use --verbose to list keys"
								}
								expected = append(expected, outcome)
								require.Contains(t, messages[len(messages)-1], "; elapsed: ")
								messages[len(messages)-1] = strings.SplitN(messages[len(messages)-1], "; elapsed: ", 2)[0]
								require.Equal(t, expected, messages)
							}
						}
					})
				}
			}
		}
	}
}

// TestBusinessIDOperationsRepairConfirmation preserves reviewed work and plain prompt routing on approve, decline and EOF.
func TestBusinessIDOperationsRepairConfirmation(t *testing.T) {
	for _, destination := range []string{"configured", "inherited"} {
		for _, answer := range []string{"y", "n", "eof"} {
			t.Run(destination+answer, func(t *testing.T) {
				f, s := newBusinessIDRepairFixture(t, false, "")
				prompt := "process-instance repair: 2 repairable process instance(s), 2 active incident(s), 2 related job(s), 2 variable scope(s) will be repaired.\nDo you want to proceed? [y/N]: "
				exchange := testx.CmdTerminalExchange{Prompt: prompt, Response: answer, EndOfInput: answer == "eof", BeforeResponse: func(out, stderr string) {
					require.Empty(t, out)
					require.Empty(t, f.writes.Snapshot())
					require.Equal(t, int32(3), f.searches.Load())
					require.Equal(t, []string{"101", "102"}, f.reads.Snapshot())
					f.approved.Store(true)
				}}
				out, stderr, err := runBusinessIDCommand(t, businessIDRepairArgs(t, s, "plain-time", "info", "--no-wait", "--set-vars", `{"nested":{"value":"frozen"}}`), destination, false, exchange)
				require.Empty(t, out)
				require.Contains(t, stderr, prompt)
				require.NotContains(t, stderr, "INFO "+prompt)
				require.Equal(t, int32(3), f.searches.Load())
				require.Equal(t, []string{"101", "102"}, f.reads.Snapshot())
				if answer == "y" {
					require.NoError(t, err, stderr)
					require.Len(t, f.writes.Snapshot(), 6)
				} else {
					require.Error(t, err)
					require.Empty(t, f.writes.Snapshot())
				}
				require.NotContains(t, strings.Join(f.writes.Snapshot(), "\n"), "new")
				require.NotContains(t, strings.Join(f.writes.Snapshot(), "\n"), "999")
			})
		}
	}
}

// TestBusinessIDOperationsRepairLogging detects direct-print routing, raw HTTP disclosure and duplicate error emission.
func TestBusinessIDOperationsRepairLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, mode := range []string{"normal", "verbose", "quiet", "warn", "debug-flag", "debug-config", "json", "dry-run", "dry-verbose", "dry-warn", "dry-quiet"} {
				for _, failure := range []string{"", "write", "partial", "wait", "variables", "job"} {
					if strings.HasPrefix(mode, "dry") && failure != "" {
						continue
					}
					t.Run(strings.Join([]string{format, destination, mode, failure}, "/"), func(t *testing.T) {
						f, s := newBusinessIDRepairFixture(t, false, failure)
						level := "info"
						if mode == "warn" || mode == "dry-warn" {
							level = "warn"
						}
						if mode == "debug-config" {
							level = "debug"
						}
						flags := []string{"--auto-confirm"}
						if failure != "wait" {
							flags = append(flags, "--no-wait")
						}
						if failure == "variables" {
							flags = append(flags, "--set-vars", `{"nested":{"value":"frozen"}}`)
						}
						switch mode {
						case "verbose", "quiet", "json", "dry-run":
							flags = append(flags, "--"+mode)
						case "debug-flag":
							flags = append(flags, "--debug")
						case "dry-verbose":
							flags = append(flags, "--dry-run", "--verbose")
						case "dry-warn":
							flags = append(flags, "--dry-run")
						case "dry-quiet":
							flags = append(flags, "--dry-run", "--quiet")
						}
						out, stderr, err := runBusinessIDCommand(t, businessIDRepairArgs(t, s, format, level, flags...), destination, false)
						if failure == "" {
							require.NoError(t, err, out+stderr)
						} else {
							assertRunBusinessIDExit(t, err, 2)
						}
						if strings.HasPrefix(mode, "dry") {
							require.Empty(t, f.writes.Snapshot())
						}
						if mode == "json" {
							e := decodeBusinessIDCancelEnvelope(t, out)
							require.Empty(t, stderr)
							if failure != "" {
								require.Equal(t, "invalid", e["outcome"])
								require.NotEmpty(t, e["payload"])
								require.Contains(t, out, "body-only-secret")
							}
							return
						}
						require.Empty(t, out)
						records := bulkCompletionDiagnosticRecords(t, stderr, format)
						var infos, errs, debugs []string
						for _, r := range records {
							switch r[0] {
							case "INFO":
								infos = append(infos, r[1])
							case "ERROR":
								errs = append(errs, r[1])
							case "DEBUG":
								debugs = append(debugs, r[1])
							default:
								t.Fatalf("unexpected severity %s", r[0])
							}
						}
						if failure == "" {
							require.Empty(t, errs)
						} else {
							require.Len(t, errs, 1)
							require.Contains(t, errs[0], "repair process instances: selected 2, incidents 2")
							if failure == "partial" {
								require.Contains(t, errs[0], "submitted 1, confirmed 0, failed 1")
							} else {
								require.Contains(t, errs[0], "failed 2")
							}
						}
						if mode == "quiet" || mode == "warn" || mode == "dry-warn" || mode == "dry-quiet" {
							require.Empty(t, infos)
						} else {
							require.NotEmpty(t, infos)
						}
						if strings.HasPrefix(mode, "debug-") {
							require.NotEmpty(t, debugs)
							require.Contains(t, stderr, "/v2/process-instances/search")
							require.NotContains(t, stderr, "secret-cookie")
						} else {
							require.Empty(t, debugs)
							for _, raw := range []string{"POST", "https://private.invalid", "secret-token", "body-only-secret", "resource-key-secret"} {
								require.NotContains(t, stderr, raw)
								require.NotContains(t, out, raw)
							}
						}
					})
				}
			}
		}
	}
}

// TestBusinessIDOperationsRepairFailures preserves original exits and partial planning in one envelope without writes.
func TestBusinessIDOperationsRepairFailures(t *testing.T) {
	for _, failure := range []string{"search", "planning"} {
		t.Run(failure, func(t *testing.T) {
			f, s := newBusinessIDRepairFixture(t, false, failure)
			out, stderr, err := runBusinessIDCommand(t, businessIDRepairArgs(t, s, "json", "info", "--json", "--auto-confirm"), "configured", false)
			assertRunBusinessIDExit(t, err, 2)
			require.Empty(t, stderr)
			require.Empty(t, f.writes.Snapshot())
			e := decodeBusinessIDCancelEnvelope(t, out)
			require.Equal(t, "invalid", e["outcome"])
			require.NotEmpty(t, e["payload"].(map[string]any)["frozenSet"])
		})
	}
}

// TestBusinessIDOperationsRepairValidation rejects old versions and invalid selectors before search or mutation.
func TestBusinessIDOperationsRepairValidation(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		cases := [][]string{{"--bid", "ORDER"}}
		if version == "8.10" {
			cases = [][]string{{"--bid", ""}, {"--bid", "ORDER", "--key", "101"}, {"--bid", "ORDER", "-"}, {"--bid", "ORDER", "--limit", "-1"}, {"--bid", "ORDER", "--batch-size", "0"}}
		}
		for _, flags := range cases {
			t.Run(version+strings.Join(flags, "/"), func(t *testing.T) {
				var requests testx.SafeSlice[string]
				s := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if serveVersionCheckTopology(w, r, version) {
						return
					}
					requests.Append(r.URL.Path)
					w.WriteHeader(500)
				}))
				defer s.Close()
				args := append([]string{"--config", testx.WriteTestConfigForVersion(t, s.URL, version), "ops", "repair", "pi", "--json"}, flags...)
				out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
				require.Error(t, err)
				require.Empty(t, requests.Snapshot())
				require.Empty(t, stderr)
				e := decodeBusinessIDCancelEnvelope(t, out)
				require.NotEqual(t, "succeeded", e["outcome"])
			})
		}
	}
}

// TestBusinessIDOperationsRepairBoundsAndFailFast keeps documented distinct limits and unattempted incident identities.
func TestBusinessIDOperationsRepairBoundsAndFailFast(t *testing.T) {
	for _, limit := range []string{"1", "2"} {
		t.Run(limit, func(t *testing.T) {
			f, s := newBusinessIDRepairFixture(t, false, "")
			out, stderr, err := runBusinessIDCommand(t, businessIDRepairArgs(t, s, "plain-time", "info", "--json", "--dry-run", "--limit", limit), "configured", false)
			require.NoError(t, err, out+stderr)
			require.Empty(t, stderr)
			require.Empty(t, f.writes.Snapshot())
			e := decodeBusinessIDCancelEnvelope(t, out)
			frozen := e["payload"].(map[string]any)["frozenSet"].(map[string]any)
			require.Equal(t, true, frozen["limited"])
			require.Contains(t, out, "limit_reached")
			if limit == "1" {
				require.Equal(t, int32(1), f.searches.Load())
				require.Equal(t, []any{"101"}, frozen["processInstanceKeys"])
			} else {
				require.Equal(t, int32(3), f.searches.Load())
				require.Equal(t, []any{"101", "102"}, frozen["processInstanceKeys"])
			}
		})
	}
	f, s := newBusinessIDRepairFixture(t, false, "write")
	out, stderr, err := runBusinessIDCommand(t, businessIDRepairArgs(t, s, "plain-time", "info", "--json", "--auto-confirm", "--no-wait", "--fail-fast"), "configured", false)
	assertRunBusinessIDExit(t, err, 2)
	require.Empty(t, stderr)
	e := decodeBusinessIDCancelEnvelope(t, out)
	plan := e["payload"].(map[string]any)["plan"].([]any)
	require.Len(t, plan, 2)
	require.Equal(t, "2102", plan[1].(map[string]any)["incidentKey"])
	require.Equal(t, "skipped", plan[1].(map[string]any)["resolutionStatus"])
	require.Equal(t, []string{"PATCH /v2/jobs/3101", "POST /v2/incidents/2101/resolution"}, f.writes.Snapshot())
}

// TestBusinessIDOperationsRepairWriteFailure attempts a failed envelope write only once and retains the original exit.
func TestBusinessIDOperationsRepairWriteFailure(t *testing.T) {
	f, s := newBusinessIDRepairFixture(t, false, "write")
	out, stderr, err := runBusinessIDCommand(t, businessIDRepairArgs(t, s, "plain-time", "info", "--json", "--auto-confirm", "--no-wait"), "configured", true)
	assertRunBusinessIDExit(t, err, 2)
	require.Equal(t, "attempt\n", out)
	require.Len(t, f.writes.Snapshot(), 4)
	require.Equal(t, [][2]string{{"ERROR", "repair process instances: could not write result"}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
}

// TestBusinessIDOperationsRepairReports preserves opaque criteria and observed identities through executed JSON and escaped Markdown reports.
func TestBusinessIDOperationsRepairReports(t *testing.T) {
	for _, bid := range []string{" 01,Ω* ", "line\n[ref](x)|`*<tag>", strings.Repeat("x", 300)} {
		for _, format := range []string{"json", "markdown"} {
			t.Run(fmt.Sprint(len(bid), format), func(t *testing.T) {
				f, s := newBusinessIDRepairFixture(t, false, "")
				extension := ".json"
				if format == "markdown" {
					extension = ".md"
				}
				path := filepath.Join(t.TempDir(), "report"+extension)
				args := businessIDRepairArgs(t, s, "plain-time", "info", "--bid", bid, "--auto-confirm", "--json", "--no-wait", "--bpmn-process-id", "orders", "--state", "active", "--report-file", path)
				out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
				require.NoError(t, err, out+stderr)
				require.Empty(t, stderr)
				e := decodeBusinessIDCancelEnvelope(t, out)
				p := e["payload"].(map[string]any)
				require.Equal(t, bid, p["request"].(map[string]any)["processInstanceSelection"].(map[string]any)["businessId"])
				for _, filter := range f.filters.Snapshot() {
					require.Equal(t, bid, filter["businessId"])
					require.Equal(t, "orders", filter["processDefinitionId"])
					require.Contains(t, fmt.Sprint(filter["state"]), "ACTIVE")
				}
				require.Equal(t, int32(3), f.searches.Load())
				require.Equal(t, []string{"101", "102"}, f.reads.Snapshot())
				raw, err := os.ReadFile(path)
				require.NoError(t, err)
				if format == "json" {
					var report ops.RepairAuditReport
					require.NoError(t, json.Unmarshal(raw, &report))
					require.Equal(t, "ops.repair.v1", report.SchemaVersion)
					require.Equal(t, bid, *report.Request.ProcessInstanceSelection.BusinessId)
					require.Equal(t, bid, *report.FrozenSet.ProcessFilters.BusinessId)
					require.Equal(t, bid, *report.FrozenSet.Selection.Items[0].BusinessId)
					require.Equal(t, ops.RepairDiscoveryModeSearch, report.Request.DiscoveryMode)
					require.Nil(t, report.Request.FrozenPlan)
				} else {
					require.Contains(t, string(raw), "- Process Filters: {businessId=")
					require.Contains(t, string(raw), "Selected Process Instances")
					require.NotContains(t, string(raw), "line\n[ref]")
					require.NotContains(t, string(raw), "<tag>")
					if strings.Contains(bid, "[ref]") {
						require.Contains(t, string(raw), `\[ref\]\(x\)\|\`+"`"+`\*\<tag\>`)
					}
				}
			})
		}
	}
}

// TestBusinessIDOperationsRepairDirectEligibility applies direct-incident eligibility before a distinct seed bound.
func TestBusinessIDOperationsRepairDirectEligibility(t *testing.T) {
	f, s := newBusinessIDRepairFixture(t, false, "first-empty")
	out, stderr, err := runBusinessIDCommand(t, businessIDRepairArgs(t, s, "plain-time", "info", "--json", "--dry-run", "--direct-incidents-only", "--limit", "1"), "configured", false)
	require.NoError(t, err, out+stderr)
	require.Empty(t, stderr)
	require.Empty(t, f.writes.Snapshot())
	require.Equal(t, int32(3), f.searches.Load())
	e := decodeBusinessIDCancelEnvelope(t, out)
	frozen := e["payload"].(map[string]any)["frozenSet"].(map[string]any)
	require.Equal(t, []any{"102"}, frozen["processInstanceKeys"])
	require.Equal(t, []any{"2102"}, frozen["incidentKeys"])
}

// TestBusinessIDOperationsRepairAutomation keeps unattended nonempty work, quiet JSON precedence and error-code overrides intact.
func TestBusinessIDOperationsRepairAutomation(t *testing.T) {
	for _, failure := range []string{"", "write"} {
		for _, quiet := range []bool{false, true} {
			t.Run(fmt.Sprint(failure, quiet), func(t *testing.T) {
				f, s := newBusinessIDRepairFixture(t, false, failure)
				flags := []string{"--automation", "--json", "--keys-only", "--no-wait"}
				if quiet {
					flags = append(flags, "--quiet")
				}
				if failure != "" {
					flags = append(flags, "--no-err-codes")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDRepairArgs(t, s, "plain-time", "info", flags...), "inherited", false)
				require.NoError(t, err, out+stderr)
				require.Empty(t, stderr)
				e := decodeBusinessIDCancelEnvelope(t, out)
				if failure != "" {
					require.Equal(t, "invalid", e["outcome"])
				} else {
					require.Equal(t, "accepted", e["outcome"])
				}
				require.Equal(t, int32(3), f.searches.Load())
				require.Len(t, f.reads.Snapshot(), 2)
				require.Len(t, f.writes.Snapshot(), 4)
			})
		}
	}
}
