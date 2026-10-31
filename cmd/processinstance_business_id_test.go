// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later
package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// observedBusinessIDFixture owns a small family and records every read; any mutation is rejected.
type observedBusinessIDFixture struct {
	server    *httptest.Server
	requests  testx.SafeSlice[string]
	empty     bool
	instances map[string]map[string]any
}

// newObservedBusinessIDFixture serves current, historical, absent and empty resource identities independently.
func newObservedBusinessIDFixture(t *testing.T) *observedBusinessIDFixture {
	t.Helper()
	f := &observedBusinessIDFixture{}
	instances := map[string]map[string]any{}
	for i, key := range []string{"2251799813685101", "2251799813685102", "2251799813685103", "2251799813685104"} {
		item := map[string]any{"processInstanceKey": key, "processDefinitionKey": "9001", "processDefinitionId": "order", "processDefinitionVersion": 1, "state": "ACTIVE", "tenantId": "<default>", "startDate": "2026-10-01T00:00:00Z"}
		if i > 0 {
			item["parentProcessInstanceKey"] = "2251799813685101"
		}
		switch i {
		case 0:
			item["businessId"] = "new,\"ID\"\n"
		case 1:
			item["businessId"] = "old"
		case 3:
			item["businessId"] = ""
		}
		instances[key] = item
	}
	f.instances = instances
	f.server = testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, "8.10") {
			return
		}
		f.requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		encode := func(v any) { require.NoError(t, json.NewEncoder(w).Encode(v)) }
		page := func(items []map[string]any) {
			encode(map[string]any{"items": items, "page": map[string]any{"totalItems": len(items), "hasMoreTotalItems": false}})
		}
		switch {
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			item, ok := instances[key]
			require.True(t, ok, "unexpected instance read %s", key)
			encode(item)
		case r.Method == "GET" && r.URL.Path == "/v2/process-definitions/9001":
			encode(map[string]any{"processDefinitionKey": "9001", "processDefinitionId": "order", "version": 1, "tenantId": "<default>"})
		case r.Method == "POST" && r.URL.Path == "/v2/process-instances/search":
			var body struct {
				Filter map[string]any `json:"filter"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			items := []map[string]any{}
			if !f.empty {
				parent, hasParent := body.Filter["parentProcessInstanceKey"]
				if hasParent {
					if parent == "2251799813685101" {
						for _, k := range []string{"2251799813685102", "2251799813685103", "2251799813685104"} {
							items = append(items, instances[k])
						}
					}
				} else {
					for _, k := range []string{"2251799813685101", "2251799813685102", "2251799813685103", "2251799813685104"} {
						items = append(items, instances[k])
					}
				}
			}
			page(items)
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/incidents/search"):
			page([]map[string]any{})
		case r.Method == "POST" && slices.Contains([]string{"/v2/variables/search", "/v2/incidents/search", "/v2/jobs/search"}, r.URL.Path):
			page([]map[string]any{})
		case r.Method == "POST" && r.URL.Path == "/v2/element-instances/search":
			var body struct {
				Filter map[string]any `json:"filter"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			key, _ := body.Filter["processInstanceKey"].(string)
			page([]map[string]any{{"elementInstanceKey": "el-" + key, "processInstanceKey": key, "elementId": "task", "type": "SERVICE_TASK", "state": "ACTIVE", "hasIncident": false}})
		default:
			t.Errorf("unexpected request (including mutation): %s %s", r.Method, r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}

// assertObservedBusinessID checks each resource's own snapshot, never a parent fallback.
func assertObservedBusinessID(t *testing.T, item map[string]any, key string) {
	t.Helper()
	switch key {
	case "2251799813685101":
		require.Equal(t, "new,\"ID\"\n", item["businessId"])
	case "2251799813685102":
		require.Equal(t, "old", item["businessId"])
	case "2251799813685103":
		require.NotContains(t, item, "businessId")
	case "2251799813685104":
		require.Equal(t, "", item["businessId"])
	default:
		t.Fatalf("unexpected key %s", key)
	}
}

// TestGetProcessInstanceBusinessID observes identities through actual single/list and enriched commands without additional reads.
func TestGetProcessInstanceBusinessID(t *testing.T) {
	for _, keyed := range []bool{true, false} {
		for _, enrich := range [][]string{nil, {"--with-vars"}, {"--with-incidents"}, {"--with-elements"}, {"--with-vars", "--with-incidents", "--with-elements", "--with-listeners"}} {
			for _, mode := range []string{"human", "json", "keys", "quiet-json"} {
				if mode == "keys" && slices.Contains(enrich, "--with-elements") {
					continue
				}
				t.Run(fmt.Sprintf("keyed=%t/%v/%s", keyed, enrich, mode), func(t *testing.T) {
					f := newObservedBusinessIDFixture(t)
					args := []string{"--config", runBusinessIDConfig(t, f.server.URL, "plain-time", "info"), "get", "pi"}
					n := 4
					if keyed {
						args = append(args, "--key", "2251799813685101")
						n = 1
					}
					args = append(args, enrich...)
					if strings.Contains(mode, "json") {
						args = append(args, "--json")
					}
					if mode == "keys" {
						args = append(args, "--keys-only")
					}
					if strings.HasPrefix(mode, "quiet") {
						args = append(args, "--quiet")
					}
					out, stderr, err := runBusinessIDCommand(t, args, "inherited", false)
					require.NoError(t, err, stderr)
					require.Empty(t, stderr)
					switch mode {
					case "keys":
						want := "2251799813685101\n"
						if !keyed {
							want += "2251799813685102\n2251799813685103\n2251799813685104\n"
						}
						require.Equal(t, want, out)
					case "human":
						require.Contains(t, out, "business-id:"+strconv.Quote("new,\"ID\"\n"))
						if !keyed {
							require.Contains(t, out, `business-id:"old"`)
							require.Contains(t, out, `business-id:""`)
							require.Equal(t, 3, strings.Count(out, "business-id:"))
						}
					default:
						env := requireSingleJSONObjectDocument(t, out)
						require.Equal(t, "succeeded", env["outcome"])
						payload := requireJSONObject(t, env["payload"])
						items := requireJSONItems(t, payload["items"], n)
						for _, raw := range items {
							item := requireJSONObject(t, raw)
							if len(enrich) > 0 {
								wrapper := item
								item = requireJSONObject(t, wrapper["item"])
								if slices.Contains(enrich, "--with-listeners") {
									el := requireJSONObject(t, requireJSONItems(t, wrapper["elements"], 1)[0])
									require.Equal(t, []any{}, el["listeners"])
								}
							}
							assertObservedBusinessID(t, item, item["key"].(string))
						}
						require.Contains(t, payload, "meta")
					}
					requests := f.requests.Snapshot()
					getCount, searchCount := 0, 0
					for _, r := range requests {
						if strings.HasPrefix(r, "GET /v2/process-instances/") {
							getCount++
						}
						if r == "POST /v2/process-instances/search" {
							searchCount++
						}
					}
					if keyed {
						require.Equal(t, 1, getCount)
						require.Zero(t, searchCount)
					} else {
						require.Zero(t, getCount)
						require.Equal(t, 1, searchCount)
					}
				})
			}
		}
	}
}

// TestWalkProcessInstanceBusinessID preserves divergent family values in flat/tree and enriched traversal output.
func TestWalkProcessInstanceBusinessID(t *testing.T) {
	for _, scope := range []string{"family", "flat", "parent", "children"} {
		for _, enriched := range []bool{false, true} {
			for _, mode := range []string{"human", "json", "keys"} {
				if mode == "keys" && enriched {
					continue
				}
				t.Run(fmt.Sprintf("%s/enriched=%t/%s", scope, enriched, mode), func(t *testing.T) {
					f := newObservedBusinessIDFixture(t)
					key := "2251799813685101"
					if scope == "parent" {
						key = "2251799813685102"
					}
					args := []string{"--config", runBusinessIDConfig(t, f.server.URL, "plain-time", "info"), "walk", "pi", "--key", key}
					if scope != "family" {
						args = append(args, "--"+scope)
					}
					if enriched {
						args = append(args, "--with-vars", "--with-incidents", "--with-elements", "--with-listeners")
					}
					if mode == "json" {
						args = append(args, "--json")
					}
					if mode == "keys" {
						args = append(args, "--keys-only")
					}
					out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
					require.NoError(t, err, stderr)
					require.Empty(t, stderr)
					if mode == "json" {
						env := requireSingleJSONObjectDocument(t, out)
						var items []any
						switch p := env["payload"].(type) {
						case []any:
							items = p
						case map[string]any:
							items = p["items"].([]any)
						}
						wantCount := 4
						if scope == "parent" {
							wantCount = 2
						}
						require.Len(t, items, wantCount)
						for _, raw := range items {
							item := requireJSONObject(t, raw)
							if enriched {
								item = requireJSONObject(t, item["item"])
							}
							assertObservedBusinessID(t, item, item["key"].(string))
						}
					} else if mode == "keys" {
						want := "2251799813685101\n2251799813685102\n2251799813685103\n2251799813685104\n"
						if scope == "parent" {
							want = "2251799813685102\n2251799813685101\n"
						}
						require.Equal(t, want, out)
					} else {
						require.Contains(t, out, "business-id:"+strconv.Quote("new,\"ID\"\n"))
						require.Contains(t, out, `business-id:"old"`)
					}
					// Family/flat resolve ancestry then descendants; children and parent use only one traversal.
					gets, searches := 0, 0
					for _, r := range f.requests.Snapshot() {
						if strings.HasPrefix(r, "GET /v2/process-instances/") {
							gets++
						}
						if r == "POST /v2/process-instances/search" {
							searches++
						}
					}
					if scope == "parent" {
						require.Equal(t, 2, gets)
						require.Zero(t, searches)
					} else {
						require.Equal(t, 4, searches)
						if scope == "children" {
							require.Equal(t, 1, gets)
						} else {
							require.Equal(t, 2, gets)
						}
					}
				})
			}
		}
	}
}

// TestProcessInstanceMutationBusinessID executes observed previews under configured loggers and stable machine contracts.
func TestProcessInstanceMutationBusinessID(t *testing.T) {
	for _, command := range []string{"cancel", "delete", "update", "suspend", "resume", "definition"} {
		for _, format := range []string{"plain-time", "json"} {
			for i, mode := range []string{"verbose", "warn", "quiet", "json", "quiet-json", "keys"} {
				t.Run(command+"/"+format+"/"+mode, func(t *testing.T) {
					f := newObservedBusinessIDFixture(t)
					if command == "cancel" {
						for _, item := range f.instances {
							item["state"] = "COMPLETED"
						}
					}
					level := "info"
					if mode == "warn" {
						level = "warn"
					}
					args := []string{"--config", runBusinessIDConfig(t, f.server.URL, format, level)}
					switch command {
					case "definition":
						args = append(args, "delete", "pd", "--key", "9001", "--force")
					case "cancel":
						args = append(args, "cancel", "pi", "--key", "2251799813685101,2251799813685102,2251799813685103,2251799813685104")
					case "update":
						args = append(args, "update", "pi", "--key", "2251799813685101,2251799813685102,2251799813685103,2251799813685104", "--vars", `{"x":true}`)
					default:
						args = append(args, command, "pi", "--key", "2251799813685101")
						if command == "suspend" || command == "resume" {
							args = append(args, "--tree")
						}
					}
					args = append(args, "--dry-run")
					if mode == "verbose" || mode == "warn" {
						args = append(args, "--verbose")
					}
					if strings.Contains(mode, "json") {
						args = append(args, "--json")
					}
					if mode == "keys" {
						args = append(args, "--keys-only")
					}
					if strings.HasPrefix(mode, "quiet") {
						args = append(args, "--quiet")
					}
					dest := "inherited"
					if i%2 == 0 {
						dest = "configured"
					}
					out, stderr, err := runBusinessIDCommand(t, args, dest, false)
					require.NoError(t, err, stderr)
					if strings.Contains(mode, "json") {
						env := requireSingleJSONObjectDocument(t, out)
						require.Equal(t, "succeeded", env["outcome"])
						p := requireJSONObject(t, env["payload"])
						var items []any
						switch command {
						case "cancel":
							items = p["selectedFinalState"].([]any)
						case "delete":
							items = p["requiresCancelBeforeDelete"].([]any)
						case "update":
							items = p["processInstances"].([]any)
						case "suspend", "resume":
							items = requireJSONObject(t, p["plan"])["instances"].([]any)
						case "definition":
							plan := requireJSONObject(t, p["plan"])
							pi := requireJSONObject(t, plan["items"].([]any)[0])
							items = requireJSONObject(t, pi["cancellationPlan"])["RequiresCancelBeforeDelete"].([]any)
						}
						require.Len(t, items, 4)
						for _, raw := range items {
							item := requireJSONObject(t, raw)
							key, _ := item["key"].(string)
							if command == "update" {
								key = item["processInstanceKey"].(string)
							}
							assertObservedBusinessID(t, item, key)
						}
						require.Empty(t, stderr)
					} else if mode == "keys" {
						if command == "definition" || command == "resume" {
							require.Empty(t, out)
						} else {
							require.Equal(t, "2251799813685101\n2251799813685102\n2251799813685103\n2251799813685104\n", out)
						}
						require.Empty(t, stderr)
					} else {
						require.Empty(t, out, "mutation observations must use INFO, not stdout")
						records := bulkCompletionDiagnosticRecords(t, stderr, format)
						var infos []string
						for _, r := range records {
							if r[0] == "INFO" {
								infos = append(infos, r[1])
							} else {
								require.Equal(t, "WARN", r[0])
							}
						}
						if mode == "warn" || mode == "quiet" {
							require.Empty(t, infos)
						} else {
							text := strings.Join(infos, "\n")
							require.Contains(t, text, "business-id:"+strconv.Quote("new,\"ID\"\n"))
							require.Contains(t, text, `business-id:"old"`)
							require.Contains(t, text, `business-id:""`)
							require.Equal(t, 3, strings.Count(text, "business-id:"))
						}
					}
					gets, searches, variables := 0, 0, 0
					for _, r := range f.requests.Snapshot() {
						if strings.HasPrefix(r, "GET /v2/process-instances/") {
							gets++
						}
						if r == "POST /v2/process-instances/search" {
							searches++
						}
						if r == "POST /v2/variables/search" {
							variables++
						}
					}
					switch command {
					case "update":
						require.Equal(t, 4, gets)
						require.Equal(t, 4, variables)
						require.Zero(t, searches)
					case "suspend", "resume":
						require.Equal(t, 1, gets)
						require.Equal(t, 4, searches)
					case "delete":
						require.Equal(t, 2, gets)
						require.Equal(t, 4, searches)
					}
				})
			}
		}
	}
}

// TestProcessInstanceBusinessIDEmptyAndLocalPreviews checks prompt-free terminal completion, logger routing and read-free exact-key previews.
func TestProcessInstanceBusinessIDEmptyAndLocalPreviews(t *testing.T) {
	for _, command := range []string{"get", "cancel", "delete", "suspend", "resume"} {
		for _, dry := range []bool{false, true} {
			if (command == "suspend" || command == "resume") && !dry {
				continue
			}
			for _, control := range []string{"interactive", "--auto-confirm", "--automation", "--no-wait"} {
				if command == "get" && control == "--no-wait" {
					continue
				}
				for i, mode := range []string{"human", "json", "keys", "quiet", "quiet-json", "quiet-keys", "warn"} {
					t.Run(fmt.Sprintf("%s/dry=%t/%s/%s", command, dry, control, mode), func(t *testing.T) {
						f := newObservedBusinessIDFixture(t)
						f.empty = true
						level, format := "info", "plain-time"
						if mode == "warn" {
							level = "warn"
						}
						if i%2 == 1 {
							format = "json"
						}
						args := []string{"--config", runBusinessIDConfig(t, f.server.URL, format, level), command, "pi"}
						local := command == "suspend" || command == "resume"
						if local {
							args = append(args, "--key", "2251799813685101")
						} else {
							args = append(args, "--state", "active")
						}
						if dry {
							if command == "get" {
								args = append(args, "--with-vars")
							} else {
								args = append(args, "--dry-run")
							}
						}
						if control != "interactive" {
							args = append(args, control)
						}
						if strings.Contains(mode, "json") {
							args = append(args, "--json")
						}
						if strings.Contains(mode, "keys") {
							args = append(args, "--keys-only")
						}
						if strings.HasPrefix(mode, "quiet") {
							args = append(args, "--quiet")
						}
						dest := "configured"
						if i%2 == 1 {
							dest = "inherited"
						}
						out, stderr, err := runBusinessIDCommand(t, args, dest, false)
						require.NoError(t, err, stderr)
						if strings.Contains(mode, "json") {
							env := requireSingleJSONObjectDocument(t, out)
							require.Equal(t, "succeeded", env["outcome"])
							require.NotContains(t, out, "businessId")
							require.Empty(t, stderr)
							p := requireJSONObject(t, env["payload"])
							if command == "get" {
								require.NotContains(t, p, "items")
								require.NotContains(t, p, "total")
								require.Equal(t, map[string]any{"withAge": true}, p["meta"])
							} else if local {
								item := requireJSONObject(t, requireJSONItems(t, p["items"], 1)[0])
								require.Equal(t, "preview", item["status"])
								require.NotContains(t, p, "plan")
							} else {
								requireEmptyProcessInstanceSelectorOutput(t, out, stderr, command+" process-instance", command, dry, RenderModeJSON)
							}
						} else if strings.Contains(mode, "keys") {
							if local {
								require.Equal(t, "2251799813685101\n", out)
							} else {
								require.Empty(t, out)
							}
							require.Empty(t, stderr)
						} else if command == "get" {
							want := "found: 0\n"
							if mode == "quiet" {
								want = ""
							}
							require.Equal(t, want, out)
							require.Empty(t, stderr)
						} else {
							require.Empty(t, out)
							records := bulkCompletionDiagnosticRecords(t, stderr, format)
							if mode == "quiet" || mode == "warn" {
								require.Empty(t, records)
							} else if local {
								require.NotEmpty(t, records)
								for _, r := range records {
									require.Equal(t, "INFO", r[0])
									require.NotContains(t, r[1], "business-id:")
								}
							} else {
								require.Equal(t, [][2]string{{"INFO", "found: 0"}}, records)
							}
						}
						if local {
							require.Empty(t, f.requests.Snapshot())
						} else {
							require.Equal(t, []string{"POST /v2/process-instances/search"}, f.requests.Snapshot())
						}
					})
				}
			}
		}
	}
}

// TestProcessInstanceBusinessIDTreeExecution keeps frozen IDs separate from accepted mutation outcomes and routes verbose facts through INFO.
func TestProcessInstanceBusinessIDTreeExecution(t *testing.T) {
	for _, verb := range []string{"suspend", "resume"} {
		for _, format := range []string{"plain-time", "json"} {
			for _, mode := range []string{"verbose", "warn", "quiet", "json"} {
				t.Run(verb+"/"+format+"/"+mode, func(t *testing.T) {
					var url, key string
					var snapshot func() (map[string]int, map[string]int)
					if verb == "suspend" {
						f := newSuspensionTreeFixture(t)
						f.businessIDs = map[string]string{suspendKeyA: "new", suspendKeyB: "old"}
						url, key = f.server.URL, suspendKeyA
						snapshot = func() (map[string]int, map[string]int) {
							_, w, r := f.snapshot()
							require.Len(t, f.discoverySnapshot(), 4)
							return w, r
						}
					} else {
						f := newResumptionTreeFixture(t)
						f.businessIDs = map[string]string{resumeKeyA: "new", resumeKeyB: "old"}
						url, key = f.server.URL, resumeKeyA
						snapshot = func() (map[string]int, map[string]int) {
							_, w, r := f.snapshot()
							require.Len(t, f.discoverySnapshot(), 4)
							return w, r
						}
					}
					level := "info"
					if mode == "warn" {
						level = "warn"
					}
					args := []string{"--config", runBusinessIDConfig(t, url, format, level), verb, "pi", "--key", key, "--tree", "--auto-confirm", "--no-wait", "--workers", "1"}
					if mode == "json" {
						args = append(args, "--json")
					} else {
						args = append(args, "--verbose")
					}
					if mode == "quiet" {
						args = append(args, "--quiet")
					}
					out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
					require.NoError(t, err, stderr)
					writes, reads := snapshot()
					require.Len(t, writes, 3)
					for _, n := range writes {
						require.Equal(t, 1, n)
					}
					require.Empty(t, reads)
					if mode == "json" {
						env := requireSingleJSONObjectDocument(t, out)
						require.Equal(t, "accepted", env["outcome"])
						p := requireJSONObject(t, env["payload"])
						plan := requireJSONObject(t, p["plan"])
						items := requireJSONItems(t, plan["instances"], 3)
						require.Equal(t, "new", requireJSONObject(t, items[0])["businessId"])
						require.Equal(t, "old", requireJSONObject(t, items[1])["businessId"])
						require.NotContains(t, requireJSONObject(t, items[2]), "businessId")
						for _, raw := range requireJSONItems(t, p["items"], 3) {
							require.NotContains(t, requireJSONObject(t, raw), "businessId")
						}
						require.Empty(t, stderr)
					} else {
						require.Empty(t, out)
						records := bulkCompletionDiagnosticRecords(t, stderr, format)
						if mode == "quiet" || mode == "warn" {
							require.Empty(t, records)
						} else {
							var messages []string
							for _, r := range records {
								require.Equal(t, "INFO", r[0])
								messages = append(messages, r[1])
							}
							text := strings.Join(messages, "\n")
							require.Contains(t, text, `business-id:"new"`)
							require.Contains(t, text, `business-id:"old"`)
							require.Equal(t, 2, strings.Count(text, "business-id:"))
						}
					}
				})
			}
		}
	}
}
