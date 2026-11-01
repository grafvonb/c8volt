// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// businessIDResolveFixture owns all incident discovery and mutation alongside the overlapping family fixture.
type businessIDResolveFixture struct {
	family *businessIDCancelFixture
	reads  testx.SafeSlice[string]
	writes testx.SafeSlice[string]
}

// newBusinessIDResolveFixture introduces new incidents after approval or submission to detect rediscovery into writes.
func newBusinessIDResolveFixture(t *testing.T, empty bool, failure string) (*businessIDResolveFixture, *httptest.Server) {
	t.Helper()
	family, base := newBusinessIDCancelFixture(t, empty, failure == "children", failure)
	t.Cleanup(base.Close)
	f := &businessIDResolveFixture{family: family}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if !strings.HasSuffix(path, "/incidents/search") && !strings.HasPrefix(path, "/v2/incidents/") {
			base.Config.Handler.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fail := func() {
			w.Header().Set("Set-Cookie", "secret-cookie")
			w.WriteHeader(400)
			fmt.Fprint(w, `{"detail":"POST https://private.invalid/path?token=secret-token body-only-secret resource-key-secret"}`)
		}
		if strings.HasSuffix(path, "/resolution") {
			key := strings.Split(path, "/")[3]
			f.writes.Append(key)
			require.GreaterOrEqual(t, len(f.reads.Snapshot()), 4, "mutation preceded complete planning")
			if failure == "write" || (failure == "partial" && key == "i202") {
				fail()
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		key := strings.Split(path, "/")[3]
		f.reads.Append(key)
		if (failure == "planning" && key == "203") || (failure == "wait" && len(f.writes.Snapshot()) > 0) {
			fail()
			return
		}
		var body struct {
			Filter map[string]any `json:"filter"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		// Incident scope preserves the existing tenant policy even after PI-family expansion.
		require.Equal(t, "tenant-a", body.Filter["tenantId"])
		items := []map[string]any{}
		incident := func(k, pi, state string) map[string]any {
			return map[string]any{"incidentKey": k, "processInstanceKey": pi, "state": state, "tenantId": "tenant-a", "jobKey": "job-" + k, "elementInstanceKey": "scope-" + k}
		}
		if failure != "no-incidents" && (key == "102" || key == "202") && !slices.Contains(f.writes.Snapshot(), "i"+key) {
			items = append(items, incident("i"+key, key, "ACTIVE"), incident("i"+key, key, "ACTIVE"))
		}
		if key == "101" {
			items = append(items, incident("old", key, "RESOLVED"), incident("foreign", "outside", "ACTIVE"))
		}
		if family.afterApproval.Load() || len(f.writes.Snapshot()) > 0 {
			items = append(items, incident("new-"+key, key, "ACTIVE"))
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "page": map[string]any{"totalItems": len(items), "hasMoreTotalItems": false}}))
	}))
	t.Cleanup(server.Close)
	return f, server
}

// businessIDResolveArgs uses owned configuration and deterministic worker order.
func businessIDResolveArgs(t *testing.T, s *httptest.Server, format, level string, flags ...string) []string {
	t.Helper()
	args := []string{"--config", runBusinessIDConfig(t, s.URL, format, level), "--tenant", "tenant-a", "resolve", "pi", "--bid", "ORDER-123", "--workers", "1"}
	return append(args, flags...)
}

// TestBusinessIDResolveScope proves duplicate/sparse selection, global expansion, frozen incident work and result provenance.
func TestBusinessIDResolveScope(t *testing.T) {
	for _, dry := range []bool{false, true} {
		for _, noWait := range []bool{false, true} {
			t.Run(fmt.Sprint(dry, noWait), func(t *testing.T) {
				f, s := newBusinessIDResolveFixture(t, false, "")
				flags := []string{"--auto-confirm", "--json"}
				if dry {
					flags = append(flags, "--dry-run")
				}
				if noWait {
					flags = append(flags, "--no-wait")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDResolveArgs(t, s, "plain-time", "info", flags...), "configured", false)
				require.NoError(t, err, out+stderr)
				require.Empty(t, stderr)
				e := decodeBusinessIDCancelEnvelope(t, out)
				p := e["payload"].(map[string]any)
				require.Contains(t, out, `"businessId": "ORDER-123"`)
				require.Len(t, p["items"], 7)
				if dry {
					require.Empty(t, f.writes.Snapshot())
					require.Equal(t, "succeeded", e["outcome"])
				} else {
					require.Equal(t, []string{"i102", "i202"}, f.writes.Snapshot())
				}
				require.Equal(t, int32(3), f.family.searches.Load())
				reads := 7
				if !dry && !noWait {
					reads += 2
				}
				require.Len(t, f.reads.Snapshot(), reads)
			})
		}
	}
}

// TestBusinessIDResolveEmpty exercises real-terminal empty/no-incident scopes and preserves exact machine and logger contracts.
func TestBusinessIDResolveEmpty(t *testing.T) {
	for _, empty := range []bool{false, true} {
		for _, dry := range []bool{false, true} {
			for _, mode := range []string{"human", "quiet", "json", "quiet-json", "keys", "quiet-keys"} {
				for _, control := range []string{"", "auto-confirm", "automation", "no-wait"} {
					t.Run(fmt.Sprint(empty, dry, mode, control), func(t *testing.T) {
						f, s := newBusinessIDResolveFixture(t, empty, "no-incidents")
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
						if strings.Contains(mode, "keys") {
							flags = append(flags, "--keys-only")
						}
						if strings.HasPrefix(mode, "quiet") {
							flags = append(flags, "--quiet")
						}
						out, stderr, err := runBusinessIDCommand(t, businessIDResolveArgs(t, s, "plain-time", "info", flags...), "configured", false)
						require.NoError(t, err, out+stderr)
						require.Empty(t, f.writes.Snapshot())
						require.NotContains(t, stderr, "proceed")
						if strings.Contains(mode, "json") {
							require.Empty(t, stderr)
							e := decodeBusinessIDCancelEnvelope(t, out)
							require.Equal(t, "succeeded", e["outcome"])
							p := e["payload"].(map[string]any)
							require.Equal(t, false, p["mutationSubmitted"])
							require.Equal(t, dry, p["dryRun"])
							if empty {
								require.Equal(t, map[string]any{"operation": "resolveProcessInstance", "total": float64(0), "submitted": float64(0), "confirmed": float64(0), "skipped": float64(0), "failed": float64(0), "dryRun": dry, "mutationSubmitted": false}, p)
							}
						} else {
							require.Empty(t, out)
							if strings.Contains(mode, "keys") || strings.HasPrefix(mode, "quiet") {
								require.Empty(t, stderr)
							} else {
								records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
								require.NotEmpty(t, records)
								for _, r := range records {
									require.Equal(t, "INFO", r[0])
								}
								if empty {
									want := "resolved process-instances: 0 (confirmed/submitted/skipped: 0, failed: 0)"
									if dry {
										want = "dry run: resolve process-instances: 0 target(s) checked, 0 ready or already skipped, 0 failed; no changes applied"
									}
									require.Equal(t, [][2]string{{"INFO", want}}, records)
								}
							}
						}
						if empty {
							require.Empty(t, f.reads.Snapshot())
							require.Empty(t, f.family.children.Snapshot())
						} else {
							require.Len(t, f.reads.Snapshot(), 7)
						}
					})
				}
			}
		}
	}
}

// TestBusinessIDResolveConfirmation freezes incident work before plain configured/inherited prompts, including decline and EOF.
func TestBusinessIDResolveConfirmation(t *testing.T) {
	for _, destination := range []string{"configured", "inherited"} {
		for _, answer := range []string{"y", "n", "eof"} {
			for _, keys := range []bool{false, true} {
				t.Run(fmt.Sprint(destination, answer, keys), func(t *testing.T) {
					f, s := newBusinessIDResolveFixture(t, false, "")
					prompt := "You have requested to resolve incidents for 3 process instance(s), but due to the process-instance family scope, 7 instance(s) with 2 root instance(s) will be inspected and active incidents found in that family will be resolved.\nDo you want to proceed? [y/N]: "
					flags := []string{"--no-wait"}
					if keys {
						flags = append(flags, "--keys-only")
					}
					exchange := testx.CmdTerminalExchange{Prompt: prompt, Response: answer, EndOfInput: answer == "eof", BeforeResponse: func(out, stderr string) {
						require.Empty(t, out)
						require.Len(t, f.reads.Snapshot(), 7)
						require.Empty(t, f.writes.Snapshot())
						f.family.afterApproval.Store(true)
					}}
					out, stderr, err := runBusinessIDCommand(t, businessIDResolveArgs(t, s, "plain-time", "info", flags...), destination, false, exchange)
					require.Contains(t, stderr, prompt)
					require.NotContains(t, stderr, "INFO "+prompt)
					if answer == "y" {
						require.NoError(t, err, stderr)
						require.Equal(t, []string{"i102", "i202"}, f.writes.Snapshot())
						if keys {
							require.Equal(t, "102\n202\n", out)
							require.Equal(t, prompt, stderr)
						} else {
							require.Empty(t, out)
						}
					} else {
						require.Error(t, err)
						require.Empty(t, out)
						require.Empty(t, f.writes.Snapshot())
					}
					require.Len(t, f.reads.Snapshot(), 7)
				})
			}
		}
	}
}

// TestBusinessIDResolveLogging detects logger bypasses, raw aggregate disclosure, duplicate envelopes and independent DEBUG routing.
func TestBusinessIDResolveLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, mode := range []string{"normal", "verbose", "quiet", "warn", "debug-flag", "debug-config", "json", "dry-run", "dry-verbose", "dry-warn", "dry-quiet"} {
				for _, failure := range []string{"", "write", "partial", "wait"} {
					if strings.HasPrefix(mode, "dry") && failure != "" {
						continue
					}
					t.Run(strings.Join([]string{format, destination, mode, failure}, "/"), func(t *testing.T) {
						f, s := newBusinessIDResolveFixture(t, false, failure)
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
						out, stderr, err := runBusinessIDCommand(t, businessIDResolveArgs(t, s, format, level, flags...), destination, false)
						if failure == "" {
							require.NoError(t, err, out+stderr)
						} else {
							assertRunBusinessIDExit(t, err, 2)
						}
						if strings.HasPrefix(mode, "dry") {
							require.Empty(t, f.writes.Snapshot())
						} else {
							require.Equal(t, []string{"i102", "i202"}, f.writes.Snapshot())
						}
						if mode == "json" {
							require.Empty(t, stderr)
							e := decodeBusinessIDCancelEnvelope(t, out)
							p := e["payload"].(map[string]any)
							require.Len(t, p["items"], 7)
							if failure != "" {
								require.Equal(t, "invalid_input", e["class"])
								require.Contains(t, out, "body-only-secret")
								if failure == "partial" || failure == "wait" {
									require.Equal(t, true, p["mutationSubmitted"])
								}
							} else {
								require.Equal(t, "accepted", e["outcome"])
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
							require.Contains(t, errs[0], "resolve process instances: selected 3, affected 7")
							if failure == "partial" {
								require.Contains(t, errs[0], "failed 1")
							} else {
								require.Contains(t, errs[0], "failed 2")
							}
							if failure == "partial" {
								require.Contains(t, errs[0], "submitted 1")
								require.Contains(t, errs[0], "mutation submitted; outcome incomplete")
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

// TestBusinessIDResolveFailures keeps planning failures non-executable and preserves the original exit in a sole envelope.
func TestBusinessIDResolveFailures(t *testing.T) {
	for _, failure := range []string{"planning", "search", "children", "cycle", "unknown"} {
		t.Run(failure, func(t *testing.T) {
			f, s := newBusinessIDResolveFixture(t, false, failure)
			out, stderr, err := runBusinessIDCommand(t, businessIDResolveArgs(t, s, "plain-time", "info", "--json"), "configured", false)
			require.Error(t, err)
			require.Empty(t, stderr)
			require.Empty(t, f.writes.Snapshot())
			e := decodeBusinessIDCancelEnvelope(t, out)
			require.NotEqual(t, "succeeded", e["outcome"])
			if failure == "planning" {
				require.Contains(t, out, "completedKeys")
				assertRunBusinessIDExit(t, err, 2)
			}
		})
	}
}

// TestBusinessIDResolveBoundAndWriteFailure preserves family expansion beyond a seed limit and makes only one envelope write attempt.
func TestBusinessIDResolveBoundAndWriteFailure(t *testing.T) {
	for _, broken := range []bool{false, true} {
		t.Run(fmt.Sprint(broken), func(t *testing.T) {
			f, s := newBusinessIDResolveFixture(t, false, "")
			out, stderr, err := runBusinessIDCommand(t, businessIDResolveArgs(t, s, "plain-time", "info", "--json", "--auto-confirm", "--no-wait", "--limit", "1"), "configured", broken)
			require.Equal(t, []string{"i102"}, f.writes.Snapshot())
			require.Len(t, f.reads.Snapshot(), 4)
			require.Equal(t, int32(1), f.family.searches.Load())
			if broken {
				require.Error(t, err)
				require.Equal(t, "attempt\n", out)
			} else {
				require.NoError(t, err, out+stderr)
				require.Empty(t, stderr)
				e := decodeBusinessIDCancelEnvelope(t, out)
				require.Len(t, e["payload"].(map[string]any)["items"], 4)
				require.Contains(t, out, "limit_reached")
			}
		})
	}
}

// TestBusinessIDResolveValidation rejects unsupported versions and selector conflicts before any operation request.
func TestBusinessIDResolveValidation(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		cases := [][]string{{"--bid", "ORDER-123"}}
		if version == "8.10" {
			cases = [][]string{{"--bid", ""}, {"--bid", "ORDER", "--key", "101"}, {"--bid", "ORDER", "-"}, {"--bid", "ORDER", "--limit", "-1"}, {"--bid", "ORDER", "--batch-size", "0"}, {"--limit", "1"}}
		}
		for _, flags := range cases {
			t.Run(version+strings.Join(flags, "/"), func(t *testing.T) {
				requests := &testx.SafeSlice[string]{}
				s := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if serveVersionCheckTopology(w, r, version) {
						return
					}
					requests.Append(r.URL.Path)
					w.WriteHeader(500)
				}))
				defer s.Close()
				args := []string{"--config", testx.WriteTestConfigForVersion(t, s.URL, version), "resolve", "pi", "--json"}
				args = append(args, flags...)
				out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
				require.Error(t, err)
				require.Empty(t, stderr)
				require.Empty(t, requests.Snapshot())
				e := decodeBusinessIDCancelEnvelope(t, out)
				require.NotEqual(t, "succeeded", e["outcome"])
			})
		}
	}
}

// TestBusinessIDResolveFailFast retains unattempted instance identities in partial machine evidence.
func TestBusinessIDResolveFailFast(t *testing.T) {
	f, s := newBusinessIDResolveFixture(t, false, "write")
	out, stderr, err := runBusinessIDCommand(t, businessIDResolveArgs(t, s, "plain-time", "info", "--json", "--auto-confirm", "--no-wait", "--fail-fast"), "configured", false)
	assertRunBusinessIDExit(t, err, 2)
	require.Empty(t, stderr)
	require.Equal(t, []string{"i102"}, f.writes.Snapshot())
	e := decodeBusinessIDCancelEnvelope(t, out)
	p := e["payload"].(map[string]any)
	require.Len(t, p["items"], 7)
	require.Contains(t, out, "not_attempted")
	require.Equal(t, false, p["mutationSubmitted"])
}

// TestBusinessIDResolveMachineModes preserves requested key/JSON results under quiet, automation and previews.
func TestBusinessIDResolveMachineModes(t *testing.T) {
	for _, dry := range []bool{false, true} {
		for _, mode := range []string{"keys", "quiet-keys", "quiet-json", "automation-keys"} {
			t.Run(fmt.Sprint(dry, mode), func(t *testing.T) {
				f, s := newBusinessIDResolveFixture(t, false, "")
				flags := []string{"--auto-confirm", "--no-wait"}
				if dry {
					flags = append(flags, "--dry-run")
				}
				if strings.Contains(mode, "keys") {
					flags = append(flags, "--keys-only")
				} else {
					flags = append(flags, "--json")
				}
				if strings.HasPrefix(mode, "quiet") {
					flags = append(flags, "--quiet")
				}
				if strings.HasPrefix(mode, "automation") {
					flags = append(flags, "--automation")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDResolveArgs(t, s, "plain-time", "info", flags...), "inherited", false)
				require.NoError(t, err, out+stderr)
				require.Empty(t, stderr)
				if strings.Contains(mode, "keys") {
					require.Equal(t, "102\n202\n", out)
				} else {
					e := decodeBusinessIDCancelEnvelope(t, out)
					want := "accepted"
					if dry {
						want = "succeeded"
					}
					require.Equal(t, want, e["outcome"])
				}
				if dry {
					require.Empty(t, f.writes.Snapshot())
				} else {
					require.Equal(t, []string{"i102", "i202"}, f.writes.Snapshot())
				}
			})
		}
	}
}

// TestBusinessIDResolveOpaqueCriterion preserves exact input through discovery and safely quotes it in one human record.
func TestBusinessIDResolveOpaqueCriterion(t *testing.T) {
	for _, mode := range []string{"human", "json"} {
		t.Run(mode, func(t *testing.T) {
			f, s := newBusinessIDResolveFixture(t, false, "")
			bid := " 000,零*?\nreference "
			f.family.bid = bid
			args := businessIDResolveArgs(t, s, "plain-time", "info", "--dry-run")
			for i, v := range args {
				if v == "ORDER-123" {
					args[i] = bid
				}
			}
			if mode == "json" {
				args = append(args, "--json")
			}
			out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
			require.NoError(t, err, out+stderr)
			require.Empty(t, f.writes.Snapshot())
			if mode == "json" {
				require.Empty(t, stderr)
				e := decodeBusinessIDCancelEnvelope(t, out)
				p := e["payload"].(map[string]any)["plan"].(map[string]any)
				scope := p["scope"].(map[string]any)
				request := scope["request"].(map[string]any)["selection"].(map[string]any)["searchRequest"].(map[string]any)
				require.Equal(t, bid, request["filter"].(map[string]any)["businessId"])
			} else {
				require.Empty(t, out)
				records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
				require.Equal(t, [2]string{"INFO", fmt.Sprintf("selection business ID: %q", bid)}, records[0])
			}
		})
	}
}
