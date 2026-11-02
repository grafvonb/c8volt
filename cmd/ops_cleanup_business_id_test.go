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
	"regexp"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// businessIDCleanupFixture owns all discovery, parent checks, frozen families and mutation outcomes.
type businessIDCleanupFixture struct {
	searches   atomic.Int32
	approved   atomic.Bool
	reads      testx.SafeSlice[string]
	deleted    testx.SafeSlice[string]
	successful testx.SafeSlice[string]
	canceled   testx.SafeSlice[string]
	filters    testx.SafeSlice[map[string]any]
}

// newBusinessIDCleanupFixture introduces new children after approval and fails on any unowned endpoint.
func newBusinessIDCleanupFixture(t *testing.T, kind string, empty bool, failure string) (*businessIDCleanupFixture, *httptest.Server) {
	t.Helper()
	f := &businessIDCleanupFixture{}
	item := func(k string) map[string]any {
		parent := ""
		if k == "102" {
			parent = "101"
		}
		if k == "202" {
			parent = "201"
		}
		if k == "999" {
			parent = "101"
		}
		if kind == "orphan" && (k == "101" || k == "201") {
			parent = "9" + k
		}
		state := "COMPLETED"
		if (failure == "force" && k == "102") || (failure == "skip" && k == "201") {
			state = "ACTIVE"
		}
		if slices.Contains(f.canceled.Snapshot(), k) {
			state = "CANCELED"
		}
		p := map[string]any{"processInstanceKey": k, "state": state, "endDate": "2020-01-01T00:00:00Z", "tenantId": "tenant-a"}
		if parent != "" {
			p["parentProcessInstanceKey"] = parent
		}
		if k == "101" {
			p["businessId"] = "observed-other"
		}
		return p
	}
	s := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, "8.10") {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		encode := func(x any) { require.NoError(t, json.NewEncoder(w).Encode(x)) }
		fail := func() {
			w.Header().Set("Set-Cookie", "secret-cookie")
			w.WriteHeader(400)
			fmt.Fprint(w, `{"detail":"POST https://private.invalid:40401/path?token=secret-token body-only-secret resource-key-secret"}`)
		}
		path := r.URL.Path
		if path == "/v2/process-instances/search" {
			var b struct {
				Filter map[string]any `json:"filter"`
				Page   map[string]any `json:"page"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
			items := []map[string]any{}
			page := map[string]any{"totalItems": 0, "hasMoreTotalItems": false}
			if _, ok := b.Filter["businessId"]; ok {
				n := f.searches.Add(1)
				f.filters.Append(b.Filter)
				require.Equal(t, "tenant-a", b.Filter["tenantId"])
				require.False(t, f.approved.Load(), "BID rediscovery after approval")
				require.Empty(t, f.deleted.Snapshot())
				if kind == "retention" {
					require.Contains(t, b.Filter, "endDate")
				} else {
					require.Equal(t, map[string]any{"$exists": true}, b.Filter["parentProcessInstanceKey"])
				}
				if n == 3 && failure == "search" {
					fail()
					return
				}
				if !empty {
					if n == 1 {
						items = append(items, item("101"), item("101"))
					}
					if n == 3 {
						items = append(items, item("201"))
					}
				}
				if n < 3 {
					page = map[string]any{"totalItems": 10000, "hasMoreTotalItems": true, "endCursor": fmt.Sprint(n)}
				} else {
					page["totalItems"] = len(items)
				}
				if n == 2 && failure == "cycle" {
					page["endCursor"] = "1"
				}
			} else {
				require.NotContains(t, b.Filter, "tenantId")
				key, _ := b.Filter["parentProcessInstanceKey"].(string)
				f.reads.Append("children:" + key)
				if failure == "children" && key == "201" {
					fail()
					return
				}
				if key == "101" {
					items = append(items, item("102"))
					if f.approved.Load() {
						items = append(items, item("999"))
					}
				}
				if key == "201" {
					items = append(items, item("202"))
				}
				page["totalItems"] = len(items)
			}
			encode(map[string]any{"items": items, "page": page})
			return
		}
		if strings.HasSuffix(path, "/deletion") {
			require.Equal(t, int32(3), f.searches.Load(), "mutation before completed discovery")
			key := strings.Split(path, "/")[3]
			f.deleted.Append(key)
			if failure == "new-child" && key == "101" && f.approved.Load() {
				w.WriteHeader(409)
				fmt.Fprint(w, `{}`)
				return
			}
			if failure == "delete" || (failure == "partial" && key == "202") {
				fail()
				return
			}
			f.successful.Append(key)
			w.WriteHeader(204)
			return
		}
		if strings.HasSuffix(path, "/cancellation") {
			key := strings.Split(path, "/")[3]
			f.canceled.Append(key)
			w.WriteHeader(204)
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(path, "/v2/process-instances/") {
			key := strings.TrimPrefix(path, "/v2/process-instances/")
			f.reads.Append(key)
			if strings.HasPrefix(key, "9") {
				if failure == "parent" {
					fail()
					return
				}
				if failure == "non-orphan" || (failure == "skip" && key == "9201") {
					encode(item(key))
					return
				}
				w.WriteHeader(404)
				fmt.Fprint(w, `{}`)
				return
			}
			if slices.Contains(f.successful.Snapshot(), key) {
				if failure == "wait" {
					fail()
					return
				}
				w.WriteHeader(404)
				fmt.Fprint(w, `{}`)
				return
			}
			encode(item(key))
			return
		}
		t.Errorf("unexpected owned request %s %s", r.Method, path)
		w.WriteHeader(500)
	}))
	t.Cleanup(s.Close)
	return f, s
}

// businessIDCleanupArgs uses an isolated v810 profile and deterministic worker ordering.
func businessIDCleanupArgs(t *testing.T, s *httptest.Server, kind, format, level string, flags ...string) []string {
	t.Helper()
	args := []string{"--config", runBusinessIDConfig(t, s.URL, format, level), "--tenant", "tenant-a", "ops"}
	if kind == "retention" {
		args = append(args, "execute", "retention-policy", "--retention-days", "30")
	} else {
		args = append(args, "purge", "orphan-process-instances")
	}
	return append(append(args, "--bid", "ORDER-123", "--workers", "1"), flags...)
}

// TestBusinessIDOperationsCleanupScope exercises both real commands and preserves the selected/expanded distinction.
func TestBusinessIDOperationsCleanupScope(t *testing.T) {
	for _, kind := range []string{"retention", "orphan"} {
		for _, dry := range []bool{false, true} {
			for _, noWait := range []bool{false, true} {
				t.Run(fmt.Sprint(kind, dry, noWait), func(t *testing.T) {
					f, s := newBusinessIDCleanupFixture(t, kind, false, "")
					flags := []string{"--json", "--auto-confirm", "--state", "completed", "--bpmn-process-id", "order"}
					if dry {
						flags = append(flags, "--dry-run")
					}
					if noWait {
						flags = append(flags, "--no-wait")
					}
					out, stderr, err := runBusinessIDCommand(t, businessIDCleanupArgs(t, s, kind, "plain-time", "info", flags...), "configured", false)
					require.NoError(t, err, out+stderr)
					require.Empty(t, stderr)
					e := decodeBusinessIDCancelEnvelope(t, out)
					p := e["payload"].(map[string]any)
					discovery := p["discovery"].(map[string]any)
					selection := discovery["selection"].(map[string]any)
					require.Equal(t, []any{"101", "201"}, selection["keys"])
					require.Equal(t, "exhausted", selection["completion"])
					planName := "deletePlan"
					if kind == "orphan" {
						planName = "deletionPlan"
					}
					plan := p[planName].(map[string]any)
					require.Equal(t, []any{"101", "102", "201", "202"}, plan["affectedKeys"])
					require.Equal(t, int32(3), f.searches.Load())
					require.NotContains(t, out, `"FrozenPlan"`)
					require.Contains(t, out, "observed-other")
					for _, filter := range f.filters.Snapshot() {
						require.Equal(t, "ORDER-123", filter["businessId"])
						require.Equal(t, "COMPLETED", filter["state"])
						require.Equal(t, "order", filter["processDefinitionId"])
					}
					if dry {
						require.Empty(t, f.deleted.Snapshot())
					} else {
						require.Equal(t, []string{"102", "101", "202", "201"}, f.deleted.Snapshot())
					}
				})
			}
		}
	}
}

// TestBusinessIDOperationsCleanupEmpty proves sparse exhausted scopes do not prompt or submit, under real terminal stdin.
func TestBusinessIDOperationsCleanupEmpty(t *testing.T) {
	for _, kind := range []string{"retention", "orphan"} {
		for _, dry := range []bool{false, true} {
			for _, mode := range []string{"human", "quiet", "json", "quiet-json", "keys"} {
				for _, policy := range []string{"", "no-wait", "auto-confirm", "automation"} {
					if kind == "retention" && mode == "keys" {
						continue
					}
					t.Run(strings.Join([]string{kind, fmt.Sprint(dry), mode, policy}, "/"), func(t *testing.T) {
						f, s := newBusinessIDCleanupFixture(t, kind, true, "")
						var flags []string
						if dry {
							flags = append(flags, "--dry-run")
						}
						if policy != "" {
							flags = append(flags, "--"+policy)
						}
						switch mode {
						case "quiet":
							flags = append(flags, "--quiet")
						case "json":
							flags = append(flags, "--json")
						case "quiet-json":
							flags = append(flags, "--quiet", "--json")
						case "keys":
							flags = append(flags, "--keys-only")
						}
						out, stderr, err := runBusinessIDCommand(t, businessIDCleanupArgs(t, s, kind, "plain-time", "info", flags...), "inherited", false)
						require.NoError(t, err, out+stderr)
						require.Equal(t, int32(3), f.searches.Load())
						require.Empty(t, f.deleted.Snapshot())
						require.Empty(t, f.reads.Snapshot())
						require.NotContains(t, stderr, "proceed")
						if strings.Contains(mode, "json") {
							e := decodeBusinessIDCancelEnvelope(t, out)
							require.Equal(t, "succeeded", e["outcome"])
							p := e["payload"].(map[string]any)
							require.Equal(t, "planned", p["outcome"])
							require.Equal(t, "skipped", p["deletion"].(map[string]any)["status"])
							require.Empty(t, stderr)
						} else {
							require.Empty(t, out)
							records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
							if mode == "quiet" || mode == "keys" {
								require.Empty(t, records)
							} else {
								require.NotEmpty(t, records)
								for _, r := range records {
									require.Equal(t, "INFO", r[0])
								}
								requireBusinessIDCleanupEmptyHuman(t, records, kind, dry, policy, f.filters.Snapshot()[0])
							}
						}
					})
				}
			}
		}
	}
}

// TestBusinessIDOperationsCleanupLogging detects logger bypass, raw aggregate disclosure and duplicate envelopes on both command paths.
func TestBusinessIDOperationsCleanupLogging(t *testing.T) {
	for _, kind := range []string{"retention", "orphan"} {
		for _, format := range []string{"plain-time", "json"} {
			for _, destination := range []string{"configured", "inherited"} {
				for _, mode := range []string{"normal", "verbose", "quiet", "warn", "debug-flag", "debug-config", "json", "dry-run", "dry-verbose", "dry-warn", "dry-quiet"} {
					for _, failure := range []string{"", "delete", "partial", "wait"} {
						if strings.HasPrefix(mode, "dry") && failure != "" {
							continue
						}
						t.Run(strings.Join([]string{kind, format, destination, mode, failure}, "/"), func(t *testing.T) {
							f, s := newBusinessIDCleanupFixture(t, kind, false, failure)
							level := "info"
							if strings.Contains(mode, "warn") {
								level = "warn"
							}
							if mode == "debug-config" {
								level = "debug"
							}
							flags := []string{"--auto-confirm"}
							if failure != "wait" {
								flags = append(flags, "--no-wait")
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
							out, stderr, err := runBusinessIDCommand(t, businessIDCleanupArgs(t, s, kind, format, level, flags...), destination, false)
							if failure == "" {
								require.NoError(t, err, out+stderr)
							} else {
								assertRunBusinessIDExit(t, err, 2)
							}
							if strings.HasPrefix(mode, "dry") {
								require.Empty(t, f.deleted.Snapshot())
							}
							if mode == "json" {
								e := decodeBusinessIDCancelEnvelope(t, out)
								require.Empty(t, stderr)
								if failure != "" {
									require.Equal(t, "invalid", e["outcome"])
									require.Contains(t, out, "body-only-secret")
									deletion := e["payload"].(map[string]any)["deletion"].(map[string]any)
									require.Len(t, deletion["items"], 2)
									if failure == "partial" {
										require.True(t, deletion["items"].([]any)[0].(map[string]any)["ok"].(bool))
									}
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
									t.Fatalf("unexpected severity: %s", r[0])
								}
							}
							if failure == "" {
								require.Empty(t, errs)
								if strings.Contains(mode, "quiet") || strings.Contains(mode, "warn") {
									require.Empty(t, infos)
								} else {
									require.NotEmpty(t, infos)
								}
							} else {
								require.Len(t, errs, 1)
								require.Contains(t, errs[0], "selected 2, accepted ")
								require.Less(t, len(errs[0]), 250)
								if strings.Contains(mode, "quiet") || strings.Contains(mode, "warn") {
									require.Empty(t, infos)
								}
							}
							if strings.HasPrefix(mode, "debug-") {
								require.NotEmpty(t, debugs)
								require.Contains(t, stderr, "/v2/process-instances/search")
								require.NotContains(t, stderr, "secret-cookie")
							} else {
								require.Empty(t, debugs)
								for _, raw := range []string{"POST", "https://private.invalid", "secret-token", "body-only-secret", "resource-key-secret"} {
									require.NotContains(t, out, raw)
									require.NotContains(t, stderr, raw)
								}
							}
						})
					}
				}
			}
		}
	}
}

// TestBusinessIDOperationsCleanupConfirmation uses real terminal stdin and proves approval cannot admit a new child or BID root.
func TestBusinessIDOperationsCleanupConfirmation(t *testing.T) {
	for _, kind := range []string{"retention", "orphan"} {
		for _, answer := range []string{"y\n", "n\n", "\x04"} {
			for _, destination := range []string{"configured", "inherited"} {
				t.Run(fmt.Sprint(kind, answer, destination), func(t *testing.T) {
					f, s := newBusinessIDCleanupFixture(t, kind, false, "")
					prefix := "retention cleanup: 2 retention candidate(s), 4 affected process instance(s) across 2 root(s) will be deleted.\nDo you want to proceed? [y/N]: "
					if kind == "orphan" {
						prefix = "orphan purge: 2 orphan candidate(s), 4 affected process instance(s) across 2 root(s) will be deleted.\nDo you want to proceed? [y/N]: "
					}
					out, stderr, err := runBusinessIDCommand(t, businessIDCleanupArgs(t, s, kind, "plain-time", "info", "--json", "--no-wait"), destination, false, testx.CmdTerminalExchange{Prompt: prefix, Response: answer, EndOfInput: answer == "\x04", BeforeResponse: func(out, stderr string) {
						require.Equal(t, int32(3), f.searches.Load())
						require.Empty(t, f.deleted.Snapshot())
						f.approved.Store(true)
					}})
					e := decodeBusinessIDCancelEnvelope(t, out)
					require.Contains(t, stderr, prefix)
					require.NotContains(t, out, "Do you want")
					require.Equal(t, int32(3), f.searches.Load())
					if answer == "y\n" {
						require.NoError(t, err, out+stderr)
						require.Equal(t, []string{"102", "101", "202", "201"}, f.deleted.Snapshot())
						require.Equal(t, "succeeded", e["outcome"])
					} else {
						require.Error(t, err)
						require.Empty(t, f.deleted.Snapshot())
						require.NotEqual(t, "succeeded", e["outcome"])
					}
					require.NotContains(t, f.deleted.Snapshot(), "999")
					require.Equal(t, 1, countBusinessIDCancelKeys(f.reads.Snapshot(), "children:101"))
					require.Equal(t, 1, countBusinessIDCancelKeys(f.reads.Snapshot(), "children:201"))
				})
			}
		}
	}
}

// TestBusinessIDOperationsCleanupFailureAndBounds verifies sparse failures, unknown parents, force, skips and candidate bounds.
func TestBusinessIDOperationsCleanupFailureAndBounds(t *testing.T) {
	for _, kind := range []string{"retention", "orphan"} {
		for _, scenario := range []string{"search", "children", "cycle", "parent", "force", "skip", "non-orphan", "bound", "fail-fast"} {
			if kind == "retention" && (scenario == "parent" || scenario == "non-orphan") {
				continue
			}
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				failure := scenario
				if scenario == "bound" {
					failure = ""
				}
				if scenario == "fail-fast" {
					failure = "delete"
				}
				f, s := newBusinessIDCleanupFixture(t, kind, false, failure)
				flags := []string{"--json", "--auto-confirm", "--no-wait"}
				if scenario == "force" {
					flags = append(flags, "--force")
				}
				if scenario == "bound" {
					flags = append(flags, "--limit", "1", "--dry-run")
				}
				if scenario == "fail-fast" {
					flags = append(flags, "--fail-fast")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDCleanupArgs(t, s, kind, "plain-time", "info", flags...), "configured", false)
				require.Empty(t, stderr)
				e := decodeBusinessIDCancelEnvelope(t, out)
				if slices.Contains([]string{"search", "children", "cycle", "parent", "fail-fast"}, scenario) {
					require.Error(t, err)
					require.NotEqual(t, "succeeded", e["outcome"])
					if scenario == "fail-fast" {
						require.Equal(t, []string{"102"}, f.deleted.Snapshot())
					} else {
						require.Empty(t, f.deleted.Snapshot())
					}
					return
				}
				require.NoError(t, err, out)
				p := e["payload"].(map[string]any)
				switch scenario {
				case "bound":
					require.Equal(t, int32(1), f.searches.Load())
					require.Empty(t, f.deleted.Snapshot())
					require.Equal(t, float64(1), p["discovery"].(map[string]any)["count"])
				case "skip":
					require.Equal(t, []string{"102", "101"}, f.deleted.Snapshot())
				case "non-orphan":
					require.Empty(t, f.deleted.Snapshot())
					require.Equal(t, float64(0), p["discovery"].(map[string]any)["count"])
				case "force":
					require.Equal(t, []string{"102", "101", "202", "201"}, f.deleted.Snapshot())
				}
			})
		}
	}
}

// TestBusinessIDOperationsCleanupValidation rejects unsupported versions and invalid inputs without any operation requests.
func TestBusinessIDOperationsCleanupValidation(t *testing.T) {
	for _, kind := range []string{"retention", "orphan"} {
		for _, version := range []string{"8.8", "8.9", "8.10"} {
			cases := [][]string{{"--bid", "id"}}
			if version == "8.10" {
				cases = [][]string{{"--bid", ""}, {"--bid", "id", "--limit", "-1"}, {"--bid", "id", "-"}}
			}
			for _, flags := range cases {
				t.Run(fmt.Sprint(kind, version, flags), func(t *testing.T) {
					var calls atomic.Int32
					s := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if serveVersionCheckTopology(w, r, version) {
							return
						}
						calls.Add(1)
						w.WriteHeader(500)
					}))
					defer s.Close()
					args := []string{"--config", testx.WriteTestConfigForVersion(t, s.URL, version), "ops"}
					if kind == "retention" {
						args = append(args, "execute", "retention-policy", "--retention-days", "30")
					} else {
						args = append(args, "purge", "orphan-process-instances")
					}
					args = append(append(args, "--json"), flags...)
					out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
					require.Error(t, err)
					require.Zero(t, calls.Load())
					require.Empty(t, stderr)
					require.NotEqual(t, "succeeded", decodeBusinessIDCancelEnvelope(t, out)["outcome"])
				})
			}
		}
	}
}

// TestBusinessIDOperationsCleanupReports retains exact opaque criteria, independent observations and unchanged schema identifiers.
func TestBusinessIDOperationsCleanupReports(t *testing.T) {
	for _, kind := range []string{"retention", "orphan"} {
		for _, bid := range []string{" 01,Ω* ", "line\n[ref](x)|`*<tag>", strings.Repeat("x", 300)} {
			for _, format := range []string{"json", "markdown"} {
				t.Run(fmt.Sprint(kind, len(bid), format), func(t *testing.T) {
					f, s := newBusinessIDCleanupFixture(t, kind, false, "")
					extension := ".json"
					if format == "markdown" {
						extension = ".md"
					}
					path := filepath.Join(t.TempDir(), "report"+extension)
					out, stderr, err := runBusinessIDCommand(t, businessIDCleanupArgs(t, s, kind, "plain-time", "info", "--bid", bid, "--json", "--auto-confirm", "--no-wait", "--report-file", path), "configured", false)
					require.NoError(t, err, out+stderr)
					require.Empty(t, stderr)
					for _, filter := range f.filters.Snapshot() {
						require.Equal(t, bid, filter["businessId"])
					}
					require.Equal(t, int32(3), f.searches.Load())
					p := decodeBusinessIDCancelEnvelope(t, out)["payload"].(map[string]any)
					require.Equal(t, bid, p["request"].(map[string]any)["selection"].(map[string]any)["businessId"])
					raw, err := os.ReadFile(path)
					require.NoError(t, err)
					if format == "json" {
						var report map[string]any
						require.NoError(t, json.Unmarshal(raw, &report))
						schema := "ops.retention-policy.v1"
						if kind == "orphan" {
							schema = "ops.orphan-process-instances.v1"
						}
						require.Equal(t, schema, report["schemaVersion"])
						require.Equal(t, bid, report["selectionFilters"].(map[string]any)["businessId"])
						selection := report["discovery"].(map[string]any)["selection"].(map[string]any)
						items := selection["items"].([]any)
						require.Equal(t, "observed-other", items[0].(map[string]any)["businessId"])
						require.NotContains(t, items[1].(map[string]any), "businessId")
						require.NotContains(t, string(raw), "FrozenPlan")
					} else {
						require.Contains(t, string(raw), "Selected Observations")
						require.Contains(t, string(raw), "observed-other")
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
}

// TestBusinessIDOperationsCleanupWriteFailure preserves the operation exit class and makes one envelope write attempt.
func TestBusinessIDOperationsCleanupWriteFailure(t *testing.T) {
	for _, kind := range []string{"retention", "orphan"} {
		t.Run(kind, func(t *testing.T) {
			f, s := newBusinessIDCleanupFixture(t, kind, false, "delete")
			out, stderr, err := runBusinessIDCommand(t, businessIDCleanupArgs(t, s, kind, "plain-time", "info", "--json", "--auto-confirm", "--no-wait"), "configured", true)
			assertRunBusinessIDExit(t, err, 2)
			require.Equal(t, "attempt\n", out)
			require.Len(t, f.deleted.Snapshot(), 2)
			records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
			require.Len(t, records, 1)
			require.Equal(t, "ERROR", records[0][0])
			require.Contains(t, records[0][1], "could not write result")
		})
	}
}

// requireBusinessIDCleanupEmptyHuman checks exact established no-op wording after removing only elapsed duration.
func requireBusinessIDCleanupEmptyHuman(t *testing.T, records [][2]string, kind string, dry bool, policy string, filter map[string]any) {
	t.Helper()
	var expected, messages []string
	if policy != "automation" {
		expected = append(expected, "configured tenant: none", `--tenant "tenant-a" sets the tenant filter`, "selection scope: tenant-a only")
	}
	header := "execute retention policy"
	if kind == "orphan" {
		header = "purge orphan process-instances"
	}
	if dry {
		header = "dry run: " + header
	}
	expected = append(expected, header)
	if kind == "retention" {
		boundary := regexp.MustCompile(`[0-9]{4}-[0-9]{2}-[0-9]{2}`).FindString(fmt.Sprint(filter["endDate"]))
		expected = append(expected, `selection filters: {businessId="ORDER-123", endDateBefore="`+boundary+`"}`, "candidate retention process instances: 0", "no retention cleanup targets found")
		preview := "delete plan: skipped; 0 retention candidate(s), 0 affected process instance(s) across 0 root(s) will be deleted"
		if dry {
			preview = "delete preview: skipped (no retention cleanup targets)"
		}
		expected = append(expected, preview, "outcome: planned; no changes applied")
	} else {
		preview, outcome := "delete plan: skipped", "outcome: planned; no targets deleted"
		if dry {
			preview = "delete preview: skipped (no orphan process-instance targets)"
			outcome = "outcome: planned; no changes applied"
		}
		expected = append(expected, "candidate orphan process instances: 0", preview, outcome)
	}
	for _, record := range records {
		require.Equal(t, "INFO", record[0])
		messages = append(messages, record[1])
	}
	require.Contains(t, messages[len(messages)-1], "; elapsed: ")
	messages[len(messages)-1] = strings.SplitN(messages[len(messages)-1], "; elapsed: ", 2)[0]
	require.Equal(t, expected, messages)
}

// TestBusinessIDOperationsCleanupMachineCombinations keeps explicit machine results under quiet/automation and original error override precedence.
func TestBusinessIDOperationsCleanupMachineCombinations(t *testing.T) {
	for _, kind := range []string{"retention", "orphan"} {
		for _, dry := range []bool{false, true} {
			t.Run(fmt.Sprint(kind, dry), func(t *testing.T) {
				f, s := newBusinessIDCleanupFixture(t, kind, false, "")
				flags := []string{"--quiet", "--automation", "--json", "--no-wait"}
				if dry {
					flags = append(flags, "--dry-run")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDCleanupArgs(t, s, kind, "plain-time", "info", flags...), "configured", false)
				require.NoError(t, err, out+stderr)
				require.Empty(t, stderr)
				require.Equal(t, "succeeded", decodeBusinessIDCancelEnvelope(t, out)["outcome"])
				if dry {
					require.Empty(t, f.deleted.Snapshot())
				}
			})
		}
	}
	for _, dry := range []bool{false, true} {
		t.Run(fmt.Sprint("keys", dry), func(t *testing.T) {
			_, s := newBusinessIDCleanupFixture(t, "orphan", false, "")
			flags := []string{"--quiet", "--keys-only", "--auto-confirm", "--no-wait"}
			if dry {
				flags = append(flags, "--dry-run")
			}
			out, stderr, err := runBusinessIDCommand(t, businessIDCleanupArgs(t, s, "orphan", "plain-time", "info", flags...), "configured", false)
			require.NoError(t, err, out+stderr)
			require.Equal(t, "101\n201\n", out)
			require.Empty(t, stderr)
		})
	}
	for _, kind := range []string{"retention", "orphan"} {
		t.Run(kind+"override", func(t *testing.T) {
			_, s := newBusinessIDCleanupFixture(t, kind, false, "delete")
			out, stderr, err := runBusinessIDCommand(t, businessIDCleanupArgs(t, s, kind, "plain-time", "info", "--json", "--auto-confirm", "--no-wait", "--no-err-codes"), "configured", false)
			require.NoError(t, err, out+stderr)
			require.Empty(t, stderr)
			require.NotEqual(t, "succeeded", decodeBusinessIDCancelEnvelope(t, out)["outcome"])
		})
	}
}

// TestBusinessIDOperationsCleanupChangedChild preserves backend conflicts without admitting post-approval delete targets.
func TestBusinessIDOperationsCleanupChangedChild(t *testing.T) {
	for _, kind := range []string{"retention", "orphan"} {
		t.Run(kind, func(t *testing.T) {
			f, s := newBusinessIDCleanupFixture(t, kind, false, "new-child")
			prompt := "retention cleanup: 2 retention candidate(s), 4 affected process instance(s) across 2 root(s) will be deleted.\nDo you want to proceed? [y/N]: "
			if kind == "orphan" {
				prompt = "orphan purge: 2 orphan candidate(s), 4 affected process instance(s) across 2 root(s) will be deleted.\nDo you want to proceed? [y/N]: "
			}
			out, stderr, err := runBusinessIDCommand(t, businessIDCleanupArgs(t, s, kind, "plain-time", "info", "--json", "--no-wait"), "configured", false, testx.CmdTerminalExchange{Prompt: prompt, Response: "y", BeforeResponse: func(out, stderr string) { require.Empty(t, f.deleted.Snapshot()); f.approved.Store(true) }})
			require.Error(t, err)
			require.Contains(t, stderr, prompt)
			e := decodeBusinessIDCancelEnvelope(t, out)
			require.NotEqual(t, "succeeded", e["outcome"])
			p := e["payload"].(map[string]any)
			require.Equal(t, "partially_failed", p["outcome"])
			require.True(t, p["deletion"].(map[string]any)["submitted"].(bool))
			require.NotContains(t, f.deleted.Snapshot(), "999")
			require.Equal(t, int32(3), f.searches.Load())
			require.Equal(t, 1, countBusinessIDCancelKeys(f.reads.Snapshot(), "children:101"))
		})
	}
}

// TestBusinessIDOperationsCleanupForceGuard keeps actionable configured errors and partial plan evidence with zero submissions.
func TestBusinessIDOperationsCleanupForceGuard(t *testing.T) {
	for _, kind := range []string{"retention", "orphan"} {
		for _, machine := range []bool{false, true} {
			t.Run(fmt.Sprint(kind, machine), func(t *testing.T) {
				f, s := newBusinessIDCleanupFixture(t, kind, false, "force")
				var flags []string
				if machine {
					flags = append(flags, "--json")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDCleanupArgs(t, s, kind, "plain-time", "info", flags...), "configured", false)
				require.Error(t, err)
				require.Empty(t, f.deleted.Snapshot())
				require.NotContains(t, stderr, "proceed")
				if machine {
					require.Empty(t, stderr)
					e := decodeBusinessIDCancelEnvelope(t, out)
					require.Equal(t, "blocked", e["payload"].(map[string]any)["deletion"].(map[string]any)["status"])
				} else {
					require.Empty(t, out)
					records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
					var errors []string
					for _, r := range records {
						if r[0] == "ERROR" {
							errors = append(errors, r[1])
						}
					}
					require.Len(t, errors, 1)
					require.Contains(t, errors[0], "use --force")
					require.Contains(t, errors[0], "no delete request was submitted")
				}
			})
		}
	}
}

// TestBusinessIDOperationsCleanupBoundedNonOrphans proves a non-orphan consumes the candidate limit without prompting or mutation.
func TestBusinessIDOperationsCleanupBoundedNonOrphans(t *testing.T) {
	for _, dry := range []bool{false, true} {
		for _, mode := range []string{"json", "keys-only", "quiet"} {
			t.Run(fmt.Sprint(dry, mode), func(t *testing.T) {
				f, s := newBusinessIDCleanupFixture(t, "orphan", false, "non-orphan")
				flags := []string{"--limit", "1", "--no-wait", "--" + mode}
				if dry {
					flags = append(flags, "--dry-run")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDCleanupArgs(t, s, "orphan", "plain-time", "info", flags...), "configured", false)
				require.NoError(t, err, out+stderr)
				require.Empty(t, stderr)
				require.Empty(t, f.deleted.Snapshot())
				require.Equal(t, int32(1), f.searches.Load())
				require.Equal(t, []string{"9101"}, f.reads.Snapshot())
				if mode == "json" {
					e := decodeBusinessIDCancelEnvelope(t, out)
					require.Equal(t, "succeeded", e["outcome"])
					p := e["payload"].(map[string]any)
					discovery := p["discovery"].(map[string]any)
					require.Equal(t, float64(0), discovery["count"])
					require.NotContains(t, discovery, "keys")
					selection := discovery["selection"].(map[string]any)
					require.Equal(t, "limit_reached", selection["completion"])
					require.Equal(t, []any{"101"}, selection["keys"])
					require.Equal(t, "skipped", p["deletionPlan"].(map[string]any)["status"])
					require.NotContains(t, p["deletion"], "submitted")
				} else {
					require.Empty(t, out)
				}
			})
		}
	}
}
