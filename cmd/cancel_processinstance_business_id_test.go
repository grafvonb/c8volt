// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// businessIDCancelFixture maps the documented RA/A1/A1a/A2/RB/B1/B2 graph to native numeric keys.
type businessIDCancelFixture struct {
	filters             testx.SafeSlice[map[string]any]
	searches            atomic.Int32
	writes              testx.SafeSlice[string]
	deleted             testx.SafeSlice[string]
	deletedSuccessfully testx.SafeSlice[string]
	deletion            bool
	children            testx.SafeSlice[string]
	requests            testx.SafeSlice[string]
	afterApproval       atomic.Bool
	empty               bool
	large               bool
	failure             string
	bid                 string
}

// newBusinessIDCancelFixture owns all native requests, including wait-time family re-observation.
func newBusinessIDCancelFixture(t *testing.T, empty, large bool, failure string) (*businessIDCancelFixture, *httptest.Server) {
	t.Helper()
	f := &businessIDCancelFixture{empty: empty, large: large, failure: failure, bid: "ORDER-123"}
	parent := map[string]string{"101": "", "102": "101", "103": "102", "104": "101", "201": "", "202": "201", "203": "201", "301": "", "302": "301", "999": "101"}
	children := map[string][]string{"101": {"102", "104"}, "102": {"103"}, "201": {"202", "203"}, "301": {"302"}}
	if large {
		children["101"] = nil
		for i := 1000; i <= 2000; i++ {
			key := strconv.Itoa(i)
			parent[key] = "101"
			children["101"] = append(children["101"], key)
		}
		parent["2001"] = "2000"
		children["2000"] = []string{"2001"}
	}
	item := func(key string) map[string]any {
		state := "ACTIVE"
		if f.failure == "terminal" || (f.deletion && f.failure != "force" && f.failure != "cancel-wait" && f.failure != "retry-fail") {
			state = "COMPLETED"
		}
		root := key
		for parent[root] != "" {
			root = parent[root]
		}
		if slices.Contains(f.writes.Snapshot(), root) && f.failure != "cancel" && !(f.failure == "partial" && root == "201") {
			state = "CANCELED"
		}
		out := map[string]any{"processInstanceKey": key, "processDefinitionKey": "9001", "processDefinitionId": "order", "state": state, "tenantId": "tenant-a"}
		if p := parent[key]; p != "" {
			out["parentProcessInstanceKey"] = p
		}
		switch key {
		case "102", "104", "202":
			out["businessId"] = f.bid
		case "101", "203":
			out["businessId"] = "other"
		}
		if f.afterApproval.Load() {
			out["businessId"] = "changed after approval"
		}
		return out
	}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, "8.10") {
			return
		}
		f.requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fail := func() {
			w.Header().Set("Set-Cookie", "secret-cookie")
			w.WriteHeader(400)
			fmt.Fprint(w, `{"detail":"POST https://private.invalid/path?token=secret-token body-only-secret resource-key-secret; POST https://private.invalid/path?token=secret-token body-only-secret resource-key-secret"}`)
		}
		switch {
		case r.URL.Path == "/v2/process-definitions/search":
			var body struct {
				Filter map[string]any `json:"filter"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "order", body.Filter["processDefinitionId"])
			require.Equal(t, "tenant-a", body.Filter["tenantId"])
			fmt.Fprint(w, `{"items":[{"processDefinitionKey":"9001","processDefinitionId":"order","version":1,"tenantId":"tenant-a"}],"page":{"totalItems":1}}`)
		case r.URL.Path == "/v2/process-instances/search":
			var body struct {
				Filter map[string]any `json:"filter"`
				Page   map[string]any `json:"page"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			items := []map[string]any{}
			page := map[string]any{"totalItems": 0, "hasMoreTotalItems": false}
			if bid, ok := body.Filter["businessId"]; ok {
				f.filters.Append(body.Filter)
				n := f.searches.Add(1)
				require.Equal(t, f.bid, bid)
				require.Equal(t, "tenant-a", body.Filter["tenantId"])
				require.Empty(t, f.writes.Snapshot(), "mutation preceded discovery")
				require.False(t, f.afterApproval.Load(), "BID rediscovered after confirmation")
				if f.failure == "search" && n > 1 {
					fail()
					return
				}
				if !f.empty {
					if f.large {
						items = append(items, item("101"))
						page["totalItems"] = 1
					} else {
						switch n {
						case 1:
							items = append(items, item("102"), item("102"))
							page = map[string]any{"totalItems": 3, "hasMoreTotalItems": true, "endCursor": "second"}
						case 2:
							page = map[string]any{"totalItems": 3, "hasMoreTotalItems": true, "endCursor": "third"}
							if f.failure == "cycle" {
								page["endCursor"] = "second"
							}
							if f.failure == "unknown" {
								page = map[string]any{"totalItems": 3}
							}
						default:
							items = append(items, item("104"))
							page["totalItems"] = 3
							if body.Page["limit"].(float64) > 1 {
								items = append(items, item("202"))
							} else {
								page["hasMoreTotalItems"] = true
								page["endCursor"] = "fourth"
							}
						}
					}
				}
			} else {
				require.NotContains(t, body.Filter, "tenantId")
				key, _ := body.Filter["parentProcessInstanceKey"].(string)
				f.children.Append(key)
				keys := children[key]
				if f.deletion && f.afterApproval.Load() && key == "101" {
					keys = append(slices.Clone(keys), "999")
				}
				if f.large && key == "101" {
					if body.Page["after"] == nil {
						keys = keys[:1000]
						page = map[string]any{"totalItems": 1001, "hasMoreTotalItems": true, "endCursor": "child-next"}
					} else {
						if f.failure == "children" {
							fail()
							return
						}
						keys = keys[1000:]
						page["totalItems"] = 1001
					}
				}
				if !f.large || key != "101" {
					page["totalItems"] = len(keys)
				}
				for _, key := range keys {
					items = append(items, item(key))
				}
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "page": page}))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			if f.failure == "wait" && len(f.writes.Snapshot()) > 0 {
				fail()
				return
			}
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if f.deletion && slices.Contains(f.deletedSuccessfully.Snapshot(), key) {
				if f.failure == "delete-wait" {
					fail()
					return
				}
				w.WriteHeader(404)
				fmt.Fprint(w, `{}`)
				return
			}
			if f.failure == "cancel-wait" && len(f.writes.Snapshot()) > 0 {
				fail()
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(item(key)))
		case f.deletion && strings.HasSuffix(r.URL.Path, "/deletion"):
			key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/process-instances/"), "/deletion")
			require.Equal(t, int32(3), f.searches.Load(), "deletion before complete selection")
			f.deleted.Append(key)
			if (f.failure == "delete" && (key == "101" || key == "201")) || (f.failure == "partial" && key == "201") {
				fail()
				return
			}
			if f.failure == "new-child" && key == "101" {
				w.WriteHeader(409)
				fmt.Fprint(w, `{}`)
				return
			}
			if (f.failure == "force" || f.failure == "cancel-wait" || f.failure == "retry-fail") && len(f.writes.Snapshot()) == 0 {
				w.WriteHeader(409)
				fmt.Fprint(w, `{}`)
				return
			}
			if f.failure == "retry-fail" {
				fail()
				return
			}
			f.deletedSuccessfully.Append(key)
			w.WriteHeader(204)
		case strings.HasSuffix(r.URL.Path, "/cancellation"):
			key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/process-instances/"), "/cancellation")
			require.NotEmpty(t, f.children.Snapshot(), "mutation preceded expansion")
			f.writes.Append(key)
			if f.failure == "cancel" || (f.failure == "partial" && key == "201") {
				fail()
				return
			}
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected owned request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	t.Cleanup(server.Close)
	return f, server
}

// businessIDCancelArgs supplies scoped, deterministic single-worker execution against only the owned server.
func businessIDCancelArgs(t *testing.T, server *httptest.Server, format, level string, extra ...string) []string {
	t.Helper()
	return append([]string{"--config", runBusinessIDConfig(t, server.URL, format, level), "--tenant", "tenant-a", "cancel", "pi", "--bid", "ORDER-123", "--workers", "1"}, extra...)
}

// decodeBusinessIDCancelEnvelope requires exactly one result with no diagnostic or second-envelope suffix.
func decodeBusinessIDCancelEnvelope(t *testing.T, out string) map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(out))
	var envelope map[string]any
	require.NoError(t, dec.Decode(&envelope))
	require.ErrorIs(t, dec.Decode(new(any)), io.EOF)
	return envelope
}

// TestBusinessIDCancelScope proves global union/counts, seed limits, full membership and prompt-free dry-run.
func TestBusinessIDCancelScope(t *testing.T) {
	for _, limit := range []bool{false, true} {
		for _, mode := range []string{"json", "keys", "verbose"} {
			t.Run(fmt.Sprintf("%t/%s", limit, mode), func(t *testing.T) {
				f, server := newBusinessIDCancelFixture(t, false, false, "")
				extra := []string{"--dry-run", "--" + mode}
				if mode == "keys" {
					extra[1] = "--keys-only"
				}
				if limit {
					extra = append(extra, "--limit", "2")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDCancelArgs(t, server, "plain-time", "info", extra...), "configured", false)
				require.NoError(t, err, stderr+out)
				members := []string{"101", "102", "103", "104", "201", "202", "203"}
				seeds := []string{"102", "104", "202"}
				roots := []string{"101", "201"}
				if limit {
					members = members[:4]
					seeds = seeds[:2]
					roots = roots[:1]
				}
				require.Empty(t, f.writes.Snapshot())
				require.Equal(t, int32(3), f.searches.Load())
				for _, root := range roots {
					require.Equal(t, 1, countBusinessIDCancelKeys(f.children.Snapshot(), root))
				}
				switch mode {
				case "json":
					require.Empty(t, stderr)
					p := requireJSONObject(t, decodeBusinessIDCancelEnvelope(t, out)["payload"])
					require.Equal(t, len(seeds), int(p["requestedCount"].(float64)))
					require.Equal(t, len(roots), int(p["resolvedRootCount"].(float64)))
					require.Equal(t, len(members), int(p["affectedCount"].(float64)))
					require.EqualValues(t, members, interfaceStrings(t, p["affectedFamilyKeys"]))
					require.Equal(t, seeds, interfaceStrings(t, p["requestedKeys"]))
					require.Equal(t, roots, interfaceStrings(t, p["resolvedRoots"]))
					require.Equal(t, true, p["scopeComplete"])
					require.Equal(t, false, p["mutationSubmitted"])
					require.Contains(t, out, `"ORDER-123"`)
				case "keys":
					require.Equal(t, strings.Join(members, "\n")+"\n", out)
					require.Empty(t, stderr)
				case "verbose":
					require.Empty(t, out)
					records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
					require.NotEmpty(t, records)
					for _, r := range records {
						require.Equal(t, "INFO", r[0])
					}
					require.Contains(t, stderr, "in-scope process-instance keys: "+strings.Join(members, ", "))
					require.Contains(t, stderr, fmt.Sprintf("dependency expansion: %d additional", len(members)-len(seeds)))
				}
			})
		}
	}
}

// interfaceStrings converts decoded arrays without obscuring exact membership assertions.
func interfaceStrings(t *testing.T, value any) []string {
	t.Helper()
	var out []string
	for _, v := range value.([]any) {
		out = append(out, v.(string))
	}
	return out
}

// TestBusinessIDCancelEmpty preserves every supported empty payload with real terminal stdin and zero prompts/writes.
func TestBusinessIDCancelEmpty(t *testing.T) {
	for _, dry := range []bool{false, true} {
		for _, mode := range [][]string{nil, {"--quiet"}, {"--keys-only"}, {"--quiet", "--keys-only"}, {"--json"}, {"--quiet", "--json"}} {
			for _, policy := range [][]string{nil, {"--no-wait"}, {"--auto-confirm"}, {"--automation"}} {
				t.Run(fmt.Sprint(dry, mode, policy), func(t *testing.T) {
					f, server := newBusinessIDCancelFixture(t, true, false, "")
					flags := append(slices.Clone(mode), policy...)
					if dry {
						flags = append(flags, "--dry-run")
					}
					out, stderr, err := runBusinessIDCommand(t, businessIDCancelArgs(t, server, "plain-time", "info", flags...), "inherited", false)
					require.NoError(t, err, stderr+out)
					require.Equal(t, int32(1), f.searches.Load())
					require.Empty(t, f.writes.Snapshot())
					require.Empty(t, f.children.Snapshot())
					if containsArg(flags, "--json") {
						require.Empty(t, stderr)
						envelope := decodeBusinessIDCancelEnvelope(t, out)
						require.Equal(t, "succeeded", envelope["outcome"])
						requireEmptyProcessInstanceSelectorOutput(t, out, stderr, "cancel process-instance", "cancel", dry, RenderModeJSON)
					} else {
						require.Empty(t, out)
						records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
						if containsArg(flags, "--quiet") || containsArg(flags, "--keys-only") {
							require.Empty(t, records)
						} else {
							require.Equal(t, [][2]string{{"INFO", "found: 0"}}, records)
						}
					}
				})
			}
		}
	}
}

// TestBusinessIDCancelLargeAndIncomplete proves exhaustive child paging and no submission on late or malformed discovery.
func TestBusinessIDCancelLargeAndIncomplete(t *testing.T) {
	for _, failure := range []string{"", "children", "search", "cycle", "unknown"} {
		t.Run(failure, func(t *testing.T) {
			large := failure == "" || failure == "children"
			f, server := newBusinessIDCancelFixture(t, false, large, failure)
			flags := []string{"--json", "--auto-confirm", "--no-wait"}
			if failure == "" {
				flags = append(flags, "--dry-run")
			}
			out, stderr, err := runBusinessIDCommand(t, businessIDCancelArgs(t, server, "plain-time", "info", flags...), "configured", false)
			require.Empty(t, stderr)
			envelope := decodeBusinessIDCancelEnvelope(t, out)
			p := requireJSONObject(t, envelope["payload"])
			require.Empty(t, f.writes.Snapshot())
			if failure != "" {
				require.Error(t, err)
				require.NotEqual(t, "succeeded", envelope["outcome"])
				require.Equal(t, false, p["scopeComplete"])
			} else {
				require.NoError(t, err)
				require.Len(t, p["affectedFamilyKeys"], 1003)
				require.Contains(t, p["affectedFamilyKeys"], "2000")
				require.Contains(t, p["affectedFamilyKeys"], "2001")
				require.Equal(t, 2, countBusinessIDCancelKeys(f.children.Snapshot(), "101"))
			}
		})
	}
}

// countBusinessIDCancelKeys checks deduplicated root expansion without assuming request interleaving.
func countBusinessIDCancelKeys(keys []string, want string) int {
	n := 0
	for _, key := range keys {
		if key == want {
			n++
		}
	}
	return n
}

// TestBusinessIDCancelExecution preserves verified/no-wait results, complete scope and selected-format routing.
func TestBusinessIDCancelExecution(t *testing.T) {
	for _, noWait := range []bool{false, true} {
		for _, mode := range []string{"json", "keys", "verbose", "automation"} {
			t.Run(fmt.Sprint(noWait, mode), func(t *testing.T) {
				f, server := newBusinessIDCancelFixture(t, false, false, "")
				flags := []string{"--auto-confirm"}
				if noWait {
					flags = append(flags, "--no-wait")
				}
				if mode == "keys" {
					flags = append(flags, "--keys-only")
				} else {
					flags = append(flags, "--"+mode)
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDCancelArgs(t, server, "plain-time", "info", flags...), "inherited", false)
				require.NoError(t, err, stderr+out)
				require.Equal(t, []string{"101", "201"}, f.writes.Snapshot())
				require.Equal(t, int32(3), f.searches.Load())
				switch mode {
				case "json":
					envelope := decodeBusinessIDCancelEnvelope(t, out)
					want := "succeeded"
					if noWait {
						want = "accepted"
					}
					require.Equal(t, want, envelope["outcome"])
					p := requireJSONObject(t, envelope["payload"])
					require.Len(t, p["items"], 2)
					scope := requireJSONObject(t, p["scope"])
					require.Len(t, scope["affectedFamilyKeys"], 7)
					require.Equal(t, true, scope["mutationSubmitted"])
					require.Empty(t, stderr)
				case "keys":
					require.Equal(t, "101\n201\n", out)
					require.Empty(t, stderr)
				default:
					require.Empty(t, out)
					records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
					for _, r := range records {
						require.Equal(t, "INFO", r[0])
					}
					if mode == "verbose" {
						require.Contains(t, stderr, "in-scope process-instance keys: 101, 102, 103, 104, 201, 202, 203")
					}
				}
			})
		}
	}
}

// TestBusinessIDCancelConfirmation freezes scope before plain configured-stderr approval and preserves decline/EOF.
func TestBusinessIDCancelConfirmation(t *testing.T) {
	for _, destination := range []string{"configured", "inherited"} {
		for _, answer := range []string{"y", "n", "eof"} {
			t.Run(destination+"/"+answer, func(t *testing.T) {
				f, server := newBusinessIDCancelFixture(t, false, false, "")
				prompt := "You have requested to cancel 3 process instance(s), but due to dependencies, a total of 7 instance(s) with 2 root instance(s) will be canceled.\nDo you want to proceed? [y/N]: "
				exchange := testx.CmdTerminalExchange{Prompt: prompt, Response: answer, EndOfInput: answer == "eof", BeforeResponse: func(out, stderr string) {
					require.Empty(t, out)
					require.Empty(t, f.writes.Snapshot())
					require.Equal(t, int32(3), f.searches.Load())
					require.Equal(t, 7, len(f.children.Snapshot()))
					f.afterApproval.Store(true)
				}}
				out, stderr, err := runBusinessIDCommand(t, businessIDCancelArgs(t, server, "plain-time", "info", "--no-wait"), destination, false, exchange)
				require.Empty(t, out)
				require.Contains(t, stderr, prompt)
				require.NotContains(t, stderr, "INFO "+prompt)
				if answer == "y" {
					require.NoError(t, err, stderr)
					require.Equal(t, []string{"101", "201"}, f.writes.Snapshot())
				} else {
					require.Empty(t, f.writes.Snapshot())
					require.Error(t, err)
				}
				require.Equal(t, int32(3), f.searches.Load())
			})
		}
	}
}

// TestBusinessIDCancelLogging requires configured INFO/ERROR routing, threshold suppression and independent redacted DEBUG.
func TestBusinessIDCancelLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, mode := range []string{"normal", "verbose", "quiet", "warn", "debug-flag", "debug-config", "json"} {
				for _, failure := range []string{"", "cancel", "partial"} {
					t.Run(format+"/"+destination+"/"+mode+"/"+failure, func(t *testing.T) {
						f, server := newBusinessIDCancelFixture(t, false, false, failure)
						level := "info"
						if mode == "warn" {
							level = "warn"
						}
						if mode == "debug-config" {
							level = "debug"
						}
						flags := []string{"--auto-confirm", "--no-wait"}
						switch mode {
						case "verbose", "quiet", "json":
							flags = append(flags, "--"+mode)
						case "debug-flag":
							flags = append(flags, "--debug")
						}
						out, stderr, err := runBusinessIDCommand(t, businessIDCancelArgs(t, server, format, level, flags...), destination, false)
						if failure == "" {
							require.NoError(t, err, stderr+out)
						} else {
							assertRunBusinessIDExit(t, err, 2)
						}
						require.Equal(t, []string{"101", "201"}, f.writes.Snapshot())
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
								t.Fatalf("unexpected severity %s", r[0])
							}
						}
						if mode == "json" {
							require.Empty(t, stderr)
							envelope := decodeBusinessIDCancelEnvelope(t, out)
							p := requireJSONObject(t, envelope["payload"])
							require.Len(t, p["items"], 2)
							if failure != "" {
								require.Equal(t, "invalid_input", envelope["class"])
								require.Contains(t, out, "body-only-secret")
								if failure == "partial" {
									require.Equal(t, true, requireJSONObject(t, p["items"].([]any)[0])["ok"])
								}
							}
						} else {
							require.Empty(t, out)
							if mode == "quiet" || mode == "warn" {
								require.Empty(t, infos)
							} else {
								require.NotEmpty(t, infos)
							}
							if failure == "" {
								require.Empty(t, errors)
							} else {
								success := 0
								if failure == "partial" {
									success = 1
								}
								require.Equal(t, []string{fmt.Sprintf("cancel process instances: selected 3, roots 2, successful %d; invalid input", success)}, errors)
							}
						}
						for _, secret := range []string{"private.invalid", "secret-token", "secret-cookie", "body-only-secret", "resource-key-secret"} {
							require.NotContains(t, stderr, secret)
						}
						if strings.HasPrefix(mode, "debug-") {
							require.NotEmpty(t, debugs)
							require.Contains(t, stderr, "POST /v2/")
						} else {
							require.Empty(t, debugs)
							require.NotContains(t, stderr, "POST ")
							require.NotContains(t, stderr, server.URL)
						}
					})
				}
			}
		}
	}
}

// TestBusinessIDCancelDryRunLogging rejects direct prints and ensures verbose preview suppression at configured thresholds.
func TestBusinessIDCancelDryRunLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, mode := range []string{"normal", "verbose", "quiet", "warn"} {
				t.Run(format+destination+mode, func(t *testing.T) {
					f, server := newBusinessIDCancelFixture(t, false, false, "")
					level := "info"
					flags := []string{"--dry-run"}
					if mode == "warn" {
						level = "warn"
					} else if mode != "normal" {
						flags = append(flags, "--"+mode)
					}
					out, stderr, err := runBusinessIDCommand(t, businessIDCancelArgs(t, server, format, level, flags...), destination, false)
					require.NoError(t, err, stderr)
					require.Empty(t, out)
					require.Empty(t, f.writes.Snapshot())
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					if mode == "quiet" || mode == "warn" {
						require.Empty(t, records)
					} else {
						require.NotEmpty(t, records)
						for _, r := range records {
							require.Equal(t, "INFO", r[0])
						}
					}
				})
			}
		}
	}
}

// TestBusinessIDCancelRejectedInputs proves unsupported/empty/key/stdin inputs cannot trigger discovery or mutation.
func TestBusinessIDCancelRejectedInputs(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, mode := range []string{"bid", "empty", "key", "stdin", "definition"} {
			if version == "8.10" && (mode == "bid" || mode == "definition") {
				continue
			}
			t.Run(version+mode, func(t *testing.T) {
				calls := &testx.SafeSlice[string]{}
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if serveVersionCheckTopology(w, r, version) {
						return
					}
					calls.Append(r.URL.Path)
					http.Error(w, "unexpected", 500)
				}))
				defer server.Close()
				args := []string{"cancel", "pi", "--bid", "ORDER-123", "--json"}
				input := ""
				switch mode {
				case "empty":
					args[3] = ""
				case "key":
					args = append(args, "--key", "102")
				case "stdin":
					args = append(args, "-")
					input = "102\n"
				case "definition":
					args = append(args, "--bpmn-process-id", "order")
				}
				out, stderr, err := runGetUserTaskCommand(t, testx.WriteTestConfigForVersion(t, server.URL, version), input, args...)
				require.Error(t, err)
				require.Empty(t, stderr)
				decodeBusinessIDCancelEnvelope(t, out)
				require.Empty(t, calls.Snapshot())
			})
		}
	}
}

// TestBusinessIDCancelOutputFailure makes exactly one JSON write attempt and retains the original operation exit.
func TestBusinessIDCancelOutputFailure(t *testing.T) {
	for _, failure := range []string{"", "partial", "search"} {
		t.Run(failure, func(t *testing.T) {
			_, server := newBusinessIDCancelFixture(t, false, false, failure)
			out, stderr, err := runBusinessIDCommand(t, businessIDCancelArgs(t, server, "plain-time", "info", "--json", "--auto-confirm", "--no-wait"), "configured", true)
			require.Error(t, err)
			require.Equal(t, "attempt\n", out)
			require.NotContains(t, stderr, "body-only-secret")
			if failure != "" {
				assertRunBusinessIDExit(t, err, 2)
			}
		})
	}
}

// TestBusinessIDCancelCombinedFilters retains every AND predicate and quotes opaque values in human previews.
func TestBusinessIDCancelCombinedFilters(t *testing.T) {
	f, server := newBusinessIDCancelFixture(t, false, false, "")
	f.bid = " ORDER,零*?\n "
	args := businessIDCancelArgs(t, server, "plain-time", "info", "--dry-run", "--state", "active", "--bpmn-process-id", "order", "--start-date-before", "2026-05-31")
	for i, arg := range args {
		if arg == "--bid" {
			args[i+1] = f.bid
		}
	}
	out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
	require.NoError(t, err, stderr+out)
	require.Empty(t, out)
	require.Empty(t, f.writes.Snapshot())
	for _, filter := range f.filters.Snapshot() {
		require.Equal(t, f.bid, filter["businessId"])
		require.Equal(t, "order", filter["processDefinitionId"])
		require.Equal(t, "ACTIVE", filter["state"])
		require.Contains(t, filter, "startDate")
		require.Equal(t, "tenant-a", filter["tenantId"])
	}
	require.Contains(t, stderr, fmt.Sprintf("selection business ID: %q", f.bid))
	require.NotContains(t, stderr, "零*?\n ")
}

// TestBusinessIDCancelSubmittedUnconfirmed preserves acceptance facts when post-submission discovery fails.
func TestBusinessIDCancelSubmittedUnconfirmed(t *testing.T) {
	for _, machine := range []bool{false, true} {
		t.Run(fmt.Sprint(machine), func(t *testing.T) {
			f, server := newBusinessIDCancelFixture(t, false, false, "wait")
			flags := []string{"--auto-confirm"}
			if machine {
				flags = append(flags, "--json")
			}
			out, stderr, err := runBusinessIDCommand(t, businessIDCancelArgs(t, server, "plain-time", "info", flags...), "configured", false)
			assertRunBusinessIDExit(t, err, 2)
			require.Equal(t, []string{"101"}, f.writes.Snapshot())
			require.Equal(t, int32(3), f.searches.Load())
			if machine {
				require.Empty(t, stderr)
				envelope := decodeBusinessIDCancelEnvelope(t, out)
				scope := requireJSONObject(t, requireJSONObject(t, envelope["payload"])["scope"])
				require.Equal(t, true, scope["mutationSubmitted"])
				require.NotEqual(t, "succeeded", envelope["outcome"])
			} else {
				require.Empty(t, out)
				records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
				var failures []string
				for _, r := range records {
					if r[0] == "ERROR" {
						failures = append(failures, r[1])
					}
				}
				require.Len(t, failures, 1)
				require.Contains(t, failures[0], "cancellation submitted for 1 root(s), outcome unconfirmed")
				require.NotContains(t, stderr, "body-only-secret")
				require.NotContains(t, stderr, "POST ")
			}
		})
	}
}

// TestBusinessIDCancelTerminalNoOp retains state checks and never claims acceptance when all roots were already terminal.
func TestBusinessIDCancelTerminalNoOp(t *testing.T) {
	for _, machine := range []bool{false, true} {
		t.Run(fmt.Sprint(machine), func(t *testing.T) {
			f, server := newBusinessIDCancelFixture(t, false, false, "terminal")
			flags := []string{"--auto-confirm", "--no-wait"}
			if machine {
				flags = append(flags, "--json")
			}
			out, stderr, err := runBusinessIDCommand(t, businessIDCancelArgs(t, server, "plain-time", "info", flags...), "configured", false)
			require.NoError(t, err, stderr+out)
			require.Empty(t, f.writes.Snapshot())
			if machine {
				require.Empty(t, stderr)
				envelope := decodeBusinessIDCancelEnvelope(t, out)
				require.Equal(t, "succeeded", envelope["outcome"])
				scope := requireJSONObject(t, requireJSONObject(t, envelope["payload"])["scope"])
				require.Equal(t, false, scope["mutationSubmitted"])
			} else {
				require.Empty(t, out)
				require.Contains(t, stderr, "already terminal; no mutation submitted")
				require.NotContains(t, stderr, "cancellation: submitted")
			}
		})
	}
}

// TestBusinessIDCancelLargeVerbose retains every member beyond one backend page in configured INFO output.
func TestBusinessIDCancelLargeVerbose(t *testing.T) {
	f, server := newBusinessIDCancelFixture(t, false, true, "")
	out, stderr, err := runBusinessIDCommand(t, businessIDCancelArgs(t, server, "plain-time", "info", "--dry-run", "--verbose"), "configured", false)
	require.NoError(t, err, stderr)
	require.Empty(t, out)
	require.Empty(t, f.writes.Snapshot())
	members := []string{"101"}
	for i := 1000; i <= 2001; i++ {
		members = append(members, strconv.Itoa(i))
	}
	records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
	var scope []string
	for _, r := range records {
		require.Equal(t, "INFO", r[0])
		if strings.HasPrefix(r[1], "in-scope process-instance keys:") {
			scope = append(scope, r[1])
		}
	}
	require.Equal(t, []string{"in-scope process-instance keys: " + strings.Join(members, ", ")}, scope)
	require.Equal(t, 2, countBusinessIDCancelKeys(f.children.Snapshot(), "101"))
}
