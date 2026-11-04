// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDSelectionNativeSearch proves both query paths send an opaque scalar AND predicate, not a related-resource lookup.
func TestBusinessIDSelectionNativeSearch(t *testing.T) {
	for _, resource := range []string{"pi", "ut"} {
		for _, bid := range []string{" ORDER,零*? ", "000123", strings.Repeat("界", 300)} {
			t.Run(resource+"/"+bid[:min(len(bid), 12)], func(t *testing.T) {
				requests := &testx.SafeSlice[string]{}
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if serveVersionCheckTopology(w, r, "8.10") {
						return
					}
					requests.Append(r.Method + " " + r.URL.Path)
					var body struct {
						Filter map[string]any `json:"filter"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					if resource == "pi" && r.URL.Path == "/v2/process-definitions/search" {
						w.Header().Set("Content-Type", "application/json")
						require.Equal(t, "tenant-a", body.Filter["tenantId"])
						fmt.Fprint(w, `{"items":[{"processDefinitionId":"order","processDefinitionKey":"9001","tenantId":"tenant-a","version":1}],"page":{"totalItems":1}}`)
						return
					}
					require.Equal(t, bid, body.Filter["businessId"])
					require.Equal(t, "tenant-a", body.Filter["tenantId"])
					require.Equal(t, "order", body.Filter["processDefinitionId"])
					variableField := "variables"
					if resource == "ut" {
						variableField = "localVariables"
					}
					require.Equal(t, []any{map[string]any{"name": "status", "value": map[string]any{"$eq": `"approved"`}}}, body.Filter[variableField])
					w.Header().Set("Content-Type", "application/json")
					path := "/v2/process-instances/search"
					if resource == "ut" {
						path = "/v2/user-tasks/search"
						require.Equal(t, "2251799813685900", body.Filter["processInstanceKey"])
						require.Equal(t, "alice", body.Filter["assignee"])
					}
					require.Equal(t, path, r.URL.Path)
					fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
				}))
				defer server.Close()
				args := []string{"--config", testx.WriteTestConfigForVersion(t, server.URL, "8.10"), "--tenant", "tenant-a", "get", resource, "--bid", bid, "--bpmn-process-id", "order", "--var", `status="approved"`, "--json"}
				if resource == "ut" {
					args = append(args, "--pi-key", "2251799813685900", "--assignee", "alice")
				}
				out, stderr, err := runBusinessIDCommand(t, args, "inherited", false)
				require.NoError(t, err, stderr+out)
				require.Empty(t, stderr)
				decodeJobTaskBusinessIDEnvelope(t, out)
				wantRequests := 1
				if resource == "pi" {
					wantRequests = 2
				}
				require.Len(t, requests.Snapshot(), wantRequests)
			})
		}
	}
}

// TestBusinessIDSelectionModesAndPaging preserves sparse native traversal, selected identity, enrichment and output conventions under terminal stdin.
func TestBusinessIDSelectionModesAndPaging(t *testing.T) {
	for _, resource := range []string{"pi", "ut"} {
		for _, mode := range []string{"human", "json", "quiet-json", "keys", "quiet-keys", "quiet", "total", "vars", "limit"} {
			for _, empty := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/empty=%t", resource, mode, empty), func(t *testing.T) {
					requests := &testx.SafeSlice[string]{}
					server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if serveVersionCheckTopology(w, r, "8.10") {
							return
						}
						requests.Append(r.URL.Path)
						w.Header().Set("Content-Type", "application/json")
						var body struct {
							Filter map[string]any `json:"filter"`
							Page   map[string]any `json:"page"`
						}
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						path := "/v2/process-instances/search"
						if resource == "ut" {
							path = "/v2/user-tasks/search"
						}
						if r.URL.Path != path {
							require.Equal(t, "vars", mode)
							if resource == "pi" {
								require.Equal(t, "/v2/variables/search", r.URL.Path)
							} else {
								require.True(t, strings.HasSuffix(r.URL.Path, "/effective-variables/search"))
							}
							fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
							return
						}
						require.Equal(t, "snapshot,OLD", body.Filter["businessId"])
						if empty {
							fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
							return
						}
						if body.Page["after"] == nil {
							// A lower-bound empty page cannot imply an empty selection or exact total.
							fmt.Fprint(w, `{"items":[],"page":{"totalItems":2,"hasMoreTotalItems":true,"endCursor":"next"}}`)
							return
						}
						require.Equal(t, "next", body.Page["after"])
						items := []map[string]any{}
						for _, key := range []string{"2251799813685101", "2251799813685102"} {
							row := map[string]any{"processInstanceKey": key, "processDefinitionKey": "9001", "processDefinitionId": "order", "processDefinitionVersion": 1, "state": "ACTIVE", "businessId": "snapshot,OLD", "tenantId": "<default>"}
							if resource == "ut" {
								row["userTaskKey"] = key
								row["state"] = "CREATED"
								row["elementId"] = "task"
								row["elementInstanceKey"] = "8001"
								row["processInstanceKey"] = "2251799813685999"
							}
							items = append(items, row)
						}
						require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "page": map[string]any{"totalItems": 2, "hasMoreTotalItems": false}}))
					}))
					defer server.Close()
					args := []string{"--config", runBusinessIDConfig(t, server.URL, "plain-time", "info"), "get", resource, "--bid", "snapshot,OLD", "--batch-size", "2"}
					if !empty || mode == "keys" {
						args = append(args, "--auto-confirm")
					}
					if mode == "quiet-json" {
						args = append(args, "--automation")
					}
					if strings.Contains(mode, "json") || mode == "vars" || mode == "limit" {
						args = append(args, "--json")
					}
					if strings.Contains(mode, "keys") {
						args = append(args, "--keys-only")
					}
					if strings.HasPrefix(mode, "quiet") {
						args = append(args, "--quiet")
					}
					if mode == "total" {
						args = append(args, "--total")
					}
					if mode == "vars" {
						args = append(args, "--with-vars")
					}
					if mode == "limit" {
						args = append(args, "--limit", "1")
					}
					out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
					require.NoError(t, err, stderr+out)
					require.Empty(t, stderr)
					count := 2
					if empty {
						count = 0
					} else if mode == "limit" {
						count = 1
					}
					switch {
					case strings.Contains(mode, "json") || mode == "vars" || mode == "limit":
						payload := decodeJobTaskBusinessIDEnvelope(t, out)
						if empty {
							if resource == "ut" {
								require.Equal(t, map[string]any{"items": []any{}, "total": float64(0)}, payload)
							} else {
								require.Empty(t, payload["items"])
							}
						} else {
							items := payload["items"].([]any)
							require.Len(t, items, count)
							for _, raw := range items {
								row := requireJSONObject(t, raw)
								if mode == "vars" {
									row = requireJSONObject(t, row["item"])
								}
								require.Equal(t, "snapshot,OLD", row["businessId"])
							}
						}
					case strings.Contains(mode, "keys"):
						want := "2251799813685101\n2251799813685102\n"
						if empty {
							want = ""
						}
						require.Equal(t, want, out)
					case mode == "total":
						require.Equal(t, fmt.Sprintf("%d\n", count), out)
					case mode == "quiet" && (empty || resource == "ut"):
						require.Empty(t, out)
					case empty:
						require.Equal(t, "found: 0\n", out)
					default:
						require.Contains(t, out, "2251799813685101")
						require.Contains(t, out, "snapshot,OLD")
					}
					wantRequests := 2
					if empty {
						wantRequests = 1
					} else if mode == "vars" {
						wantRequests += 2
					}
					require.Len(t, requests.Snapshot(), wantRequests)
				})
			}
		}
	}
}

// TestBusinessIDSelectionRejectedInputs verifies CLI key/stdin exclusivity and old-version rejection before related discovery, including empty paths.
func TestBusinessIDSelectionRejectedInputs(t *testing.T) {
	for _, resource := range []string{"pi", "ut"} {
		for _, version := range []string{"8.8", "8.9", "8.10"} {
			for _, selection := range []string{"empty", "key", "stdin", "definition", "total"} {
				if version == "8.10" && (selection == "definition" || selection == "total") {
					continue
				}
				t.Run(resource+"/"+version+"/"+selection, func(t *testing.T) {
					calls := &testx.SafeSlice[string]{}
					server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if serveVersionCheckTopology(w, r, version) {
							return
						}
						calls.Append(r.URL.Path)
						http.Error(w, "unexpected backend request", 500)
					}))
					defer server.Close()
					args := []string{"get", resource, "--bid", "ORDER", "--json"}
					stdin := ""
					switch selection {
					case "empty":
						args = []string{"get", resource, "--bid=", "--json"}
					case "key":
						args = append(args, "--key", "2251799813685101")
					case "stdin":
						args = append(args, "-")
						stdin = "2251799813685101\n"
					case "definition":
						args = append(args, "--bpmn-process-id", "order")
					case "total":
						args = []string{"get", resource, "--bid", "ORDER", "--total"}
					}
					out, stderr, err := runGetUserTaskCommand(t, testx.WriteTestConfigForVersion(t, server.URL, version), stdin, args...)
					require.Error(t, err)
					require.Empty(t, calls.Snapshot())
					if selection != "total" {
						dec := json.NewDecoder(strings.NewReader(out))
						var envelope map[string]any
						require.NoError(t, dec.Decode(&envelope))
						require.ErrorIs(t, dec.Decode(new(any)), io.EOF)
						require.Empty(t, stderr)
						if selection == "definition" {
							require.Equal(t, "failed", envelope["outcome"])
						} else {
							require.Equal(t, "invalid", envelope["outcome"])
						}
					} else {
						require.Empty(t, out)
						require.Contains(t, stderr, "business-ID search requires Camunda 8.10")
					}
				})
			}
		}
	}
}

// TestBusinessIDSelectionDirectIncidentGuard proves native selection precedes the incident check and a selected limit never hides a later matching instance.
func TestBusinessIDSelectionDirectIncidentGuard(t *testing.T) {
	requests := &testx.SafeSlice[string]{}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, "8.10") {
			return
		}
		requests.Append(r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		var body struct {
			Filter map[string]any `json:"filter"`
			Page   map[string]any `json:"page"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, "ORDER", body.Filter["businessId"])
			if body.Page["after"] == nil {
				fmt.Fprint(w, `{"items":[{"processInstanceKey":"2251799813685101","state":"ACTIVE"}],"page":{"totalItems":2,"endCursor":"next"}}`)
			} else {
				fmt.Fprint(w, `{"items":[{"processInstanceKey":"2251799813685102","state":"ACTIVE"}],"page":{"totalItems":1}}`)
			}
		case "/v2/process-instances/2251799813685101/incidents/search":
			fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
		case "/v2/process-instances/2251799813685102/incidents/search":
			fmt.Fprint(w, `{"items":[{"incidentKey":"i","processInstanceKey":"2251799813685102","state":"ACTIVE","errorMessage":"needle"}],"page":{"totalItems":1}}`)
		default:
			t.Errorf("unexpected shortcut/read %s", r.URL.Path)
			http.Error(w, "unexpected", 400)
		}
	}))
	defer server.Close()
	args := []string{"--config", runBusinessIDConfig(t, server.URL, "plain-time", "info"), "get", "pi", "--bid", "ORDER", "--direct-incidents-only", "--incident-error-message", "needle", "--limit", "1", "--batch-size", "1", "--keys-only", "--auto-confirm"}
	out, stderr, err := runBusinessIDCommand(t, args, "inherited", false)
	require.NoError(t, err, stderr+out)
	require.Empty(t, stderr)
	require.Equal(t, "2251799813685102\n", out)
	require.Equal(t, []string{"/v2/process-instances/search", "/v2/process-instances/2251799813685101/incidents/search", "/v2/process-instances/search", "/v2/process-instances/2251799813685102/incidents/search"}, requests.Snapshot())
}

// TestBusinessIDSelectionFailureLogging requires one configured diagnostic or one machine envelope, with HTTP evidence confined to redacted DEBUG.
func TestBusinessIDSelectionFailureLogging(t *testing.T) {
	for _, resource := range []string{"pi", "ut", "walk"} {
		for _, format := range []string{"plain-time", "json"} {
			for _, destination := range []string{"configured", "inherited"} {
				for _, mode := range []string{"normal", "verbose", "quiet", "debug-flag", "debug-config", "json"} {
					t.Run(resource+"/"+format+"/"+destination+"/"+mode, func(t *testing.T) {
						calls := &testx.SafeSlice[string]{}
						server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							if serveVersionCheckTopology(w, r, "8.10") {
								return
							}
							calls.Append(r.URL.Path)
							w.Header().Set("Content-Type", "application/json")
							if resource == "walk" && r.URL.Path == "/v2/process-instances/2251799813685101" {
								fmt.Fprint(w, `{"processInstanceKey":"2251799813685101","state":"ACTIVE"}`)
								return
							}
							if resource == "walk" && r.URL.Path == "/v2/process-instances/search" {
								var body struct {
									Filter map[string]any `json:"filter"`
								}
								require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
								switch {
								case body.Filter["businessId"] != nil:
									fmt.Fprint(w, `{"items":[{"processInstanceKey":"2251799813685101","state":"ACTIVE"}],"page":{"totalItems":1}}`)
								case body.Filter["parentProcessInstanceKey"] == "2251799813685101":
									fmt.Fprint(w, `{"items":[{"processInstanceKey":"2251799813685102","parentProcessInstanceKey":"2251799813685101","state":"ACTIVE"}],"page":{"totalItems":1}}`)
								default:
									fmt.Fprint(w, `{"items":[],"page":{"totalItems":0}}`)
								}
								return
							}
							if resource == "pi" && r.URL.Path == "/v2/process-instances/search" {
								fmt.Fprint(w, `{"items":[{"processInstanceKey":"2251799813685101","state":"ACTIVE"},{"processInstanceKey":"2251799813685102","state":"ACTIVE"}],"page":{"totalItems":2}}`)
								return
							}
							w.Header().Set("Set-Cookie", "secret-cookie")
							w.WriteHeader(http.StatusBadRequest)
							fmt.Fprint(w, `{"detail":"POST https://private.invalid/path?token=secret-token body-only-secret resource-key-secret; POST https://private.invalid/path?token=secret-token body-only-secret resource-key-secret"}`)
						}))
						defer server.Close()
						level := "info"
						if mode == "debug-config" {
							level = "debug"
						}
						args := []string{"--config", runBusinessIDConfig(t, server.URL, format, level), "get", resource, "--bid", "ORDER", "--auto-confirm"}
						if resource == "walk" {
							args = []string{"--config", runBusinessIDConfig(t, server.URL, format, level), "walk", "pi", "--bid", "ORDER", "--with-incidents"}
						}
						if resource == "pi" {
							args = append(args, "--with-incidents")
						}
						switch mode {
						case "verbose":
							args = append(args, "--verbose")
						case "quiet":
							args = append(args, "--quiet")
						case "debug-flag":
							args = append(args, "--debug")
						case "json":
							args = append(args, "--json")
						}
						out, stderr, err := runBusinessIDCommand(t, args, destination, false)
						assertRunBusinessIDExit(t, err, 2)
						records := bulkCompletionDiagnosticRecords(t, stderr, format)
						errors, debugs := []string{}, []string{}
						for _, record := range records {
							switch record[0] {
							case "ERROR":
								errors = append(errors, record[1])
							case "DEBUG":
								debugs = append(debugs, record[1])
							}
						}
						if mode == "json" {
							require.Empty(t, stderr)
							dec := json.NewDecoder(strings.NewReader(out))
							var envelope map[string]any
							require.NoError(t, dec.Decode(&envelope))
							require.ErrorIs(t, dec.Decode(new(any)), io.EOF)
							require.Equal(t, "invalid_input", envelope["class"])
							require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "body-only-secret")
							if resource == "walk" {
								require.Len(t, requireJSONObject(t, envelope["payload"])["keys"], 2)
							}
						} else {
							require.Empty(t, out)
							want := "business-ID search failed: invalid input"
							if resource == "pi" {
								want = "business-ID search failed: 2 selected; invalid input"
							}
							if resource == "walk" {
								want = "business-ID search failed: 1 selected; invalid input"
							}
							require.Equal(t, []string{want}, errors)
						}
						for _, secret := range []string{"private.invalid", "secret-token", "secret-cookie", "body-only-secret", "resource-key-secret"} {
							require.NotContains(t, stderr, secret)
						}
						debug := strings.HasPrefix(mode, "debug-")
						if debug {
							require.NotEmpty(t, debugs)
							require.Contains(t, stderr, "POST /v2/")
						} else {
							require.Empty(t, debugs)
							require.NotContains(t, stderr, "POST ")
							require.NotContains(t, stderr, server.URL)
						}
						want := 1
						if resource == "pi" {
							want = 3
						}
						if resource == "walk" {
							want = 7
						}
						require.Len(t, calls.Snapshot(), want)
					})
				}
			}
		}
	}
}
