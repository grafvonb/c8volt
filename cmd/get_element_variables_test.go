// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const scopeElementA = "2251799813689002"
const scopeElementB = "2251799813689003"
const scopeOwner = "2251799813688001"

// elementScopeFixture records all requests and serves only test-owned selection and local-variable pages.
type elementScopeFixture struct {
	server   *httptest.Server
	requests *testx.SafeSlice[string]
	bodies   *testx.SafeSlice[map[string]any]
}

// newElementScopeFixture supplies independent element, variable and listener pages, failing unexpected requests.
func newElementScopeFixture(t *testing.T, elements, variables []string, variableStatus int) elementScopeFixture {
	t.Helper()
	f := elementScopeFixture{requests: &testx.SafeSlice[string]{}, bodies: &testx.SafeSlice[map[string]any]{}}
	elementIndex, variableIndex := 0, 0
	f.server = newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requests.Append(r.Method + " " + r.URL.Path)
		var body map[string]any
		if r.Method == http.MethodPost {
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		}
		f.bodies.Append(body)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/element-instances/" + scopeElementA, "/v2/element-instances/search":
			require.Less(t, elementIndex, len(elements), "unexpected extra discovery")
			fmt.Fprint(w, elements[elementIndex])
			elementIndex++
		case "/v2/variables/search":
			require.Equal(t, "false", r.URL.Query().Get("truncateValues"))
			require.Less(t, variableIndex, len(variables), "unexpected extra variable request")
			if variableStatus != 0 && variableIndex == len(variables)-1 {
				w.Header().Set("Set-Cookie", "session=scope-cookie-secret")
				w.WriteHeader(variableStatus)
			}
			fmt.Fprint(w, variables[variableIndex])
			variableIndex++
		case "/v2/jobs/search":
			fmt.Fprintf(w, `{"items":[{"jobKey":"2251799813689101","kind":"EXECUTION_LISTENER","listenerEventType":"START","state":"COMPLETED","processInstanceKey":%q,"elementInstanceKey":%q}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, scopeOwner, scopeElementA)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}

// scopeElementJSON keeps row values stable by omitting wall-clock-dependent timestamps.
func scopeElementJSON(key string) string {
	return fmt.Sprintf(`{"elementInstanceKey":%q,"elementId":"task","type":"SERVICE_TASK","state":"ACTIVE","processInstanceKey":%q,"tenantId":"tenant-a"}`, key, scopeOwner)
}

// scopePage encodes explicit completion evidence, including authoritative empty collections.
func scopePage(items []string, total int, more bool) string {
	return fmt.Sprintf(`{"items":[%s],"page":{"totalItems":%d,"hasMoreTotalItems":%t}}`, strings.Join(items, ","), total, more)
}

// scopeVariableJSON retains raw serialized values and backend truncation evidence.
func scopeVariableJSON(name, value, scope string, truncated bool) string {
	return fmt.Sprintf(`{"name":%q,"value":%q,"variableKey":"400","processInstanceKey":%q,"scopeKey":%q,"tenantId":"tenant-a","isTruncated":%t}`, name, value, scopeOwner, scope, truncated)
}

// TestElementScopeVariablesGetExact proves local-only complete reads, raw JSON, aliasing and explicit cross-tenant selection.
func TestElementScopeVariablesGetExact(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		t.Run(version, func(t *testing.T) {
			f := newElementScopeFixture(t, []string{scopeElementJSON(scopeElementA)}, []string{
				strings.Replace(scopePage([]string{scopeVariableJSON("parent", "false", scopeOwner, false)}, 3, true), `"totalItems":`, `"endCursor":"first","totalItems":`, 1),
				strings.Replace(scopePage(nil, 3, true), `"totalItems":`, `"endCursor":"second","totalItems":`, 1),
				scopePage([]string{scopeVariableJSON("z", ` { "nested": [1, 2] } `, scopeElementA, true), scopeVariableJSON("a", `"full"`, scopeElementA, false)}, 3, false),
			}, 0)
			cfg := testx.WriteTestConfigForVersion(t, f.server.URL, version)
			data, err := os.ReadFile(cfg)
			require.NoError(t, err)
			data = []byte(strings.Replace(string(data), "camunda_version:", "tenant: tenant-other\n  camunda_version:", 1))
			require.NoError(t, os.WriteFile(cfg, data, 0600))
			stdout, stderr := executeRootForElementTestWithSeparateOutputs(t, "--config", cfg, "--json", "get", "ei", "--key", scopeElementA, "--with-vars")
			require.Empty(t, stderr)
			env := requireSingleJSONObjectDocument(t, stdout)
			require.Equal(t, "get element", env["command"])
			payload := requireJSONObject(t, env["payload"])
			vars := payload["variables"].([]any)
			require.Len(t, vars, 2)
			require.Equal(t, "a", requireJSONObject(t, vars[0])["name"])
			require.Equal(t, false, requireJSONObject(t, vars[0])["apiTruncated"])
			require.Equal(t, ` { "nested": [1, 2] } `, requireJSONObject(t, vars[1])["value"])
			require.Equal(t, true, requireJSONObject(t, vars[1])["apiTruncated"])
			require.NotContains(t, payload, "listeners")
			require.Len(t, f.requests.Snapshot(), 4)
			for _, body := range f.bodies.Snapshot()[1:] {
				filter := requireJSONObject(t, body["filter"])
				require.Equal(t, scopeOwner, filter["processInstanceKey"])
				require.Equal(t, scopeElementA, filter["scopeKey"])
				require.NotContains(t, filter, "tenantId")
			}
		})
	}
}

// TestElementScopeVariablesGetModes covers empty selection and local scopes without altering effective output precedence.
func TestElementScopeVariablesGetModes(t *testing.T) {
	for _, empty := range []bool{false, true} {
		for _, policy := range [][]string{nil, {"--auto-confirm"}, {"--automation"}, {"--with-listeners"}} {
			for _, mode := range [][]string{nil, {"--json"}, {"--keys-only"}, {"--quiet"}, {"--quiet", "--json"}, {"--quiet", "--keys-only"}, {"--total"}} {
				if strings.Contains(strings.Join(policy, " "), "--with-listeners") && (strings.Contains(strings.Join(mode, " "), "--keys-only") || strings.Contains(strings.Join(mode, " "), "--total")) {
					continue
				}
				t.Run(fmt.Sprintf("empty=%t/%v/%v", empty, policy, mode), func(t *testing.T) {
					items := []string{scopeElementJSON(scopeElementA)}
					if empty {
						items = nil
					}
					f := newElementScopeFixture(t, []string{scopePage(items, len(items), false)}, []string{scopePage(nil, 0, false)}, 0)
					args := []string{"--config", testx.WriteTestConfig(t, f.server.URL), "get", "element", "--with-vars"}
					args = append(args, policy...)
					args = append(args, mode...)
					stdout, stderr := executeRootForElementTestWithSeparateOutputs(t, args...)
					require.Empty(t, stderr)
					modes := strings.Join(mode, " ")
					switch {
					case strings.Contains(modes, "--json"):
						env := requireSingleJSONObjectDocument(t, stdout)
						require.Equal(t, "succeeded", env["outcome"])
						payload := requireJSONObject(t, env["payload"])
						require.Equal(t, float64(len(items)), payload["total"])
						if empty {
							require.Equal(t, []any{}, payload["items"])
						} else {
							selected := payload["items"].([]any)
							require.Equal(t, []any{}, requireJSONObject(t, selected[0])["variables"])
						}
					case strings.Contains(modes, "--total"):
						require.Equal(t, fmt.Sprintf("%d\n", len(items)), stdout)
					case strings.Contains(modes, "--keys-only"):
						expected := scopeElementA + "\n"
						if empty {
							expected = ""
						}
						require.Equal(t, expected, stdout)
					case strings.Contains(modes, "--quiet"):
						require.Empty(t, stdout)
					case empty:
						require.Equal(t, "found: 0\n", stdout)
					default:
						require.Contains(t, stdout, scopeElementA+" tenant-a SERVICE_TASK task ACTIVE pi:"+scopeOwner+"\n")
						require.Contains(t, stdout, "found: 1\n")
						require.NotContains(t, stdout, "vars:")
					}
					wantVars := 1
					if empty || strings.Contains(modes, "--keys-only") || strings.Contains(modes, "--total") {
						wantVars = 0
					}
					require.Equal(t, wantVars, strings.Count(strings.Join(f.requests.Snapshot(), "\n"), "POST /v2/variables/search"))
					require.Equal(t, 1, strings.Count(strings.Join(f.requests.Snapshot(), "\n"), "POST /v2/element-instances/search"))
				})
			}
		}
	}
}

// TestElementScopeVariablesGetSearch proves sparse selection, limit independence, both enrichments and stable human tree connectors.
func TestElementScopeVariablesGetSearch(t *testing.T) {
	for _, both := range []bool{false, true} {
		for _, keyed := range []bool{false, true} {
			for _, jsonMode := range []bool{false, true} {
				t.Run(fmt.Sprintf("both=%t/keyed=%t/json=%t", both, keyed, jsonMode), func(t *testing.T) {
					elements := []string{scopePage(nil, 2, true), scopePage([]string{scopeElementJSON(scopeElementA)}, 2, true)}
					if keyed {
						elements = []string{scopeElementJSON(scopeElementA)}
					}
					f := newElementScopeFixture(t, elements, []string{
						scopePage([]string{scopeVariableJSON("z", ` { "x": 1 } `, scopeElementA, true)}, 2, true),
						scopePage([]string{scopeVariableJSON("a", `"whole"`, scopeElementA, false)}, 2, false),
					}, 0)
					args := []string{"--config", testx.WriteTestConfig(t, f.server.URL), "get", "ei", "--with-vars"}
					if keyed {
						args = append(args, "--key", scopeElementA)
					} else {
						args = append(args, "--batch-size", "1", "--limit", "1")
					}
					if both {
						args = append(args, "--with-listeners")
					}
					if jsonMode {
						args = append(args, "--json")
					}
					stdout, stderr := executeRootForElementTestWithSeparateOutputs(t, args...)
					require.Empty(t, stderr)
					calls := strings.Join(f.requests.Snapshot(), "\n")
					require.Equal(t, len(elements), strings.Count(calls, "/v2/element-instances/"))
					require.Equal(t, 2, strings.Count(calls, "/v2/variables/search"))
					if both {
						require.Equal(t, 2, strings.Count(calls, "/v2/jobs/search"))
					} else {
						require.NotContains(t, calls, "/jobs/")
					}
					if jsonMode {
						env := requireSingleJSONObjectDocument(t, stdout)
						payload := requireJSONObject(t, env["payload"])
						if !keyed {
							require.Equal(t, float64(1), payload["total"])
							items := payload["items"].([]any)
							require.Len(t, items, 1)
							payload = requireJSONObject(t, items[0])
						}
						require.Len(t, payload["variables"].([]any), 2)
						if both {
							require.NotEmpty(t, payload["listeners"])
						} else {
							require.NotContains(t, payload, "listeners")
						}
					} else {
						expected := scopeElementA + " tenant-a SERVICE_TASK task ACTIVE pi:" + scopeOwner + "\n"
						if both {
							expected += "├─ listeners:\n│  ├─ 2251799813689101 EXECUTION_LISTENER lsnr:START COMPLETED r:0\n│  └─ 2251799813689101 EXECUTION_LISTENER lsnr:START COMPLETED r:0\n"
						}
						expected += "└─ vars:\n   ├─ a=\"whole\"\n   └─ z={\"x\":1} [api-truncated]\n"
						if !keyed {
							expected += "found: 1\n"
						}
						require.Equal(t, expected, stdout)
					}
					if !keyed {
						bodies := f.bodies.Snapshot()
						require.Equal(t, float64(0), requireJSONObject(t, bodies[0]["page"])["from"])
						require.Equal(t, float64(1), requireJSONObject(t, bodies[1]["page"])["from"])
						require.Equal(t, float64(1), requireJSONObject(t, bodies[1]["page"])["limit"])
					}
				})
			}
		}
	}
}

// TestElementScopeVariablesGetUnrequested keeps enrichment omitted and keys-only exact reads free of variable calls.
func TestElementScopeVariablesGetUnrequested(t *testing.T) {
	for _, args := range [][]string{{"--json"}, {"--keys-only", "--with-vars"}, {"--quiet", "--keys-only", "--with-vars"}, {"--quiet", "--with-vars"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			f := newElementScopeFixture(t, []string{scopeElementJSON(scopeElementA)}, []string{scopePage(nil, 0, false)}, 0)
			cmdArgs := append([]string{"--config", testx.WriteTestConfig(t, f.server.URL), "get", "element", "--key", scopeElementA}, args...)
			stdout, stderr := executeRootForElementTestWithSeparateOutputs(t, cmdArgs...)
			require.Empty(t, stderr)
			if args[0] == "--json" {
				require.NotContains(t, requireJSONObject(t, requireSingleJSONObjectDocument(t, stdout)["payload"]), "variables")
			} else if strings.Contains(strings.Join(args, " "), "--keys-only") {
				require.Equal(t, scopeElementA+"\n", stdout)
			} else {
				require.Empty(t, stdout)
			}
			want := 1
			if len(args) == 2 && args[0] == "--quiet" && args[1] == "--with-vars" {
				want = 2
			}
			require.Len(t, f.requests.Snapshot(), want)
		})
	}
}
