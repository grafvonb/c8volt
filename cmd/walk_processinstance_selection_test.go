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

// TestBusinessIDSelectionWalkModes exercises tenant-scoped unique discovery, duplicate/sparse pages and each existing rendering/enrichment scope.
func TestBusinessIDSelectionWalkModes(t *testing.T) {
	for _, mode := range [][]string{nil, {"--flat"}, {"--children"}, {"--parent"}, {"--keys-only"}, {"--quiet", "--keys-only"}, {"--json"}, {"--quiet", "--json"}, {"--with-vars", "--with-incidents", "--with-elements", "--with-listeners", "--json"}} {
		t.Run(strings.Join(mode, " "), func(t *testing.T) {
			searches := &testx.SafeSlice[map[string]any]{}
			calls := &testx.SafeSlice[string]{}
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveVersionCheckTopology(w, r, "8.10") {
					return
				}
				w.Header().Set("Content-Type", "application/json")
				calls.Append(r.URL.Path)
				switch {
				case r.URL.Path == "/v2/process-instances/search":
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					filter := requireJSONObject(t, body["filter"])
					if bid, ok := filter["businessId"]; ok {
						require.Equal(t, " exact,* Ω ", bid)
						require.Equal(t, "tenant-a", filter["tenantId"])
						searches.Append(body)
						switch len(searches.Snapshot()) {
						case 1:
							fmt.Fprint(w, `{"items":[],"page":{"totalItems":1,"hasMoreTotalItems":true,"endCursor":"sparse"}}`)
						case 2:
							fmt.Fprint(w, `{"items":[{"processInstanceKey":"101","state":"ACTIVE"},{"processInstanceKey":"101","state":"ACTIVE"}],"page":{"totalItems":1}}`)
						default:
							t.Error("unexpected selection rediscovery")
						}
					} else {
						require.NotContains(t, filter, "tenantId")
						fmt.Fprint(w, `{"items":[],"page":{"totalItems":0}}`)
					}
				case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/101":
					fmt.Fprint(w, `{"processInstanceKey":"101","state":"ACTIVE","tenantId":"<default>","businessId":"observed"}`)
				case strings.HasSuffix(r.URL.Path, "/jobs/search"), strings.HasSuffix(r.URL.Path, "/incidents/search"), strings.HasSuffix(r.URL.Path, "/element-instances/search"), strings.HasSuffix(r.URL.Path, "/variables/search"), strings.Contains(r.URL.Path, "/variables"):
					fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
				default:
					t.Errorf("unexpected walk request %s %s", r.Method, r.URL.Path)
					w.WriteHeader(400)
				}
			}))
			defer server.Close()
			args := append([]string{"--config", runBusinessIDConfig(t, server.URL, "plain-time", "info"), "--tenant", "tenant-a", "walk", "pi", "--bid", " exact,* Ω "}, mode...)
			out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
			require.NoError(t, err, stderr+out)
			require.Empty(t, stderr)
			require.Len(t, searches.Snapshot(), 2)
			if containsArg(mode, "--json") {
				dec := json.NewDecoder(strings.NewReader(out))
				var envelope map[string]any
				require.NoError(t, dec.Decode(&envelope))
				require.ErrorIs(t, dec.Decode(new(any)), io.EOF)
				require.Contains(t, out, `"observed"`)
			} else if containsArg(mode, "--keys-only") {
				require.Equal(t, "101\n", out)
			} else {
				require.Contains(t, out, "101")
			}
			require.Greater(t, len(calls.Snapshot()), 2)
		})
	}
}

// containsArg keeps test mode checks independent of command globals.
func containsArg(args []string, value string) bool {
	for _, arg := range args {
		if arg == value {
			return true
		}
	}
	return false
}

// TestBusinessIDSelectionWalkEmpty proves real-terminal empty completion never walks, enriches or prompts, including machine/quiet combinations.
func TestBusinessIDSelectionWalkEmpty(t *testing.T) {
	for _, mode := range [][]string{nil, {"--flat"}, {"--parent"}, {"--children"}, {"--quiet"}, {"--keys-only"}, {"--quiet", "--keys-only"}, {"--json"}, {"--quiet", "--json"}, {"--with-incidents", "--json"}, {"--with-vars", "--with-elements", "--json"}, {"--auto-confirm"}} {
		t.Run(strings.Join(mode, " "), func(t *testing.T) {
			calls := &testx.SafeSlice[string]{}
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveVersionCheckTopology(w, r, "8.10") {
					return
				}
				calls.Append(r.URL.Path)
				require.Equal(t, "/v2/process-instances/search", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"items":[],"page":{"totalItems":0}}`)
			}))
			defer server.Close()
			out, stderr, err := runBusinessIDCommand(t, append([]string{"--config", runBusinessIDConfig(t, server.URL, "plain-time", "info"), "walk", "pi", "--bid", "absent"}, mode...), "inherited", false)
			require.NoError(t, err, stderr+out)
			require.Empty(t, stderr)
			require.Len(t, calls.Snapshot(), 1)
			if containsArg(mode, "--json") {
				dec := json.NewDecoder(strings.NewReader(out))
				var envelope map[string]any
				require.NoError(t, dec.Decode(&envelope))
				require.ErrorIs(t, dec.Decode(new(any)), io.EOF)
				require.Equal(t, "succeeded", envelope["outcome"])
				payload := requireJSONObject(t, envelope["payload"])
				require.Equal(t, "complete", payload["outcome"])
				require.NotContains(t, payload, "keys")
				require.NotContains(t, payload, "items")
			} else {
				require.Empty(t, out)
			}
		})
	}
}

// TestBusinessIDSelectionWalkRejected proves ambiguity, input and version failures do not start traversal.
func TestBusinessIDSelectionWalkRejected(t *testing.T) {
	for _, tc := range []struct {
		name, version, bid string
		extra              []string
		searches           int
	}{
		{"ambiguous", "8.10", "same", nil, 1}, {"missing key", "8.10", "same", nil, 1}, {"empty value", "8.10", "", nil, 0}, {"conflict", "8.10", "same", []string{"--key", "101"}, 0}, {"old 88", "8.8", "same", nil, 0}, {"old 89", "8.9", "same", nil, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := &testx.SafeSlice[string]{}
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveVersionCheckTopology(w, r, tc.version) {
					return
				}
				calls.Append(r.URL.Path)
				require.Equal(t, "/v2/process-instances/search", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				if tc.name == "missing key" {
					fmt.Fprint(w, `{"items":[{}],"page":{"totalItems":1}}`)
					return
				}
				fmt.Fprint(w, `{"items":[{"processInstanceKey":"101","parentProcessInstanceKey":"100"},{"processInstanceKey":"102","parentProcessInstanceKey":"100"}],"page":{"totalItems":2}}`)
			}))
			defer server.Close()
			cfg := runBusinessIDConfig(t, server.URL, "plain-time", "info")
			// The configured adapter, not topology auto-detection, controls capability checks.
			args := append([]string{"--config", cfg, "--camunda-version", tc.version, "walk", "pi", "--bid", tc.bid, "--json"}, tc.extra...)
			out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
			require.Error(t, err)
			require.Empty(t, stderr)
			dec := json.NewDecoder(strings.NewReader(out))
			var envelope map[string]any
			require.NoError(t, dec.Decode(&envelope))
			require.ErrorIs(t, dec.Decode(new(any)), io.EOF)
			require.NotEqual(t, "succeeded", envelope["outcome"])
			require.Len(t, calls.Snapshot(), tc.searches)
			if tc.name == "ambiguous" {
				require.Contains(t, out, "ambiguous")
			}
		})
	}
}

// TestBusinessIDSelectionWalkLargeFamily proves complete command output beyond one child page and failure evidence without a success envelope.
func TestBusinessIDSelectionWalkLargeFamily(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			rootPages := &testx.SafeSlice[string]{}
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveVersionCheckTopology(w, r, "8.10") {
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					fmt.Fprint(w, `{"processInstanceKey":"1","state":"ACTIVE"}`)
					return
				}
				var body struct {
					Filter map[string]any `json:"filter"`
					Page   map[string]any `json:"page"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				items := []map[string]any{}
				page := map[string]any{"totalItems": 0}
				switch {
				case body.Filter["businessId"] != nil:
					items = append(items, map[string]any{"processInstanceKey": "1"})
					page["totalItems"] = 1
				case body.Filter["parentProcessInstanceKey"] == "1":
					rootPages.Append(fmt.Sprint(body.Page["after"]))
					if len(rootPages.Snapshot()) == 1 {
						for i := 2; i <= 1001; i++ {
							items = append(items, map[string]any{"processInstanceKey": fmt.Sprint(i), "parentProcessInstanceKey": "1", "state": "ACTIVE"})
						}
						page = map[string]any{"totalItems": 1001, "hasMoreTotalItems": true, "endCursor": "next"}
					} else {
						require.Equal(t, "next", body.Page["after"])
						if fail {
							w.WriteHeader(400)
							fmt.Fprint(w, `{"detail":"owned child-page error"}`)
							return
						}
						items = append(items, map[string]any{"processInstanceKey": "1002", "parentProcessInstanceKey": "1", "state": "ACTIVE"})
						page["totalItems"] = 1001
					}
				case body.Filter["parentProcessInstanceKey"] == "1002":
					items = append(items, map[string]any{"processInstanceKey": "1003", "parentProcessInstanceKey": "1002", "state": "ACTIVE"})
					page["totalItems"] = 1
				}
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "page": page}))
			}))
			defer server.Close()
			out, stderr, err := runBusinessIDCommand(t, []string{"--config", runBusinessIDConfig(t, server.URL, "plain-time", "info"), "walk", "pi", "--bid", "order", "--json"}, "configured", false)
			require.Empty(t, stderr)
			require.Len(t, rootPages.Snapshot(), 2)
			dec := json.NewDecoder(strings.NewReader(out))
			var envelope map[string]any
			require.NoError(t, dec.Decode(&envelope))
			require.ErrorIs(t, dec.Decode(new(any)), io.EOF)
			payload := requireJSONObject(t, envelope["payload"])
			if fail {
				assertRunBusinessIDExit(t, err, 2)
				require.Equal(t, "invalid", envelope["outcome"])
				require.Equal(t, "unresolved", payload["outcome"])
				require.Len(t, requireJSONObject(t, payload["edges"])["1"], 1000)
			} else {
				require.NoError(t, err)
				require.Equal(t, "succeeded", envelope["outcome"])
				require.Equal(t, "complete", payload["outcome"])
				require.Len(t, payload["keys"], 1003)
				require.Len(t, payload["items"], 1003)
				require.Len(t, requireJSONObject(t, payload["edges"])["1"], 1001)
			}
		})
	}
}

// TestBusinessIDSelectionWalkIncomplete never walks a single observation whose later discovery fails or cycles.
func TestBusinessIDSelectionWalkIncomplete(t *testing.T) {
	for _, mode := range []string{"backend error", "cursor cycle"} {
		t.Run(mode, func(t *testing.T) {
			calls := &testx.SafeSlice[string]{}
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveVersionCheckTopology(w, r, "8.10") {
					return
				}
				calls.Append(r.URL.Path)
				require.Equal(t, "/v2/process-instances/search", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				if len(calls.Snapshot()) == 1 {
					fmt.Fprint(w, `{"items":[{"processInstanceKey":"101"}],"page":{"totalItems":3,"hasMoreTotalItems":true,"endCursor":"next"}}`)
					return
				}
				if mode == "backend error" {
					w.WriteHeader(400)
					fmt.Fprint(w, `{"detail":"owned later error"}`)
					return
				}
				fmt.Fprint(w, `{"items":[],"page":{"totalItems":3,"hasMoreTotalItems":true,"endCursor":"next"}}`)
			}))
			defer server.Close()
			out, stderr, err := runBusinessIDCommand(t, []string{"--config", runBusinessIDConfig(t, server.URL, "json", "info"), "walk", "pi", "--bid", "order", "--json"}, "configured", false)
			require.Error(t, err)
			require.Empty(t, stderr)
			require.Len(t, calls.Snapshot(), 2)
			dec := json.NewDecoder(strings.NewReader(out))
			var envelope map[string]any
			require.NoError(t, dec.Decode(&envelope))
			require.ErrorIs(t, dec.Decode(new(any)), io.EOF)
			payload := requireJSONObject(t, envelope["payload"])
			require.Equal(t, "incomplete", payload["completion"])
			require.Equal(t, []any{"101"}, payload["keys"])
		})
	}
}

// TestBusinessIDSelectionWalkOutputFailure detects any second JSON write after an empty-result destination failure.
func TestBusinessIDSelectionWalkOutputFailure(t *testing.T) {
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, "8.10") {
			return
		}
		require.Equal(t, "/v2/process-instances/search", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"items":[],"page":{"totalItems":0}}`)
	}))
	defer server.Close()
	out, stderr, err := runBusinessIDCommand(t, []string{"--config", runBusinessIDConfig(t, server.URL, "plain-time", "info"), "walk", "pi", "--bid", "empty", "--json"}, "configured", true)
	require.Error(t, err)
	require.Empty(t, stderr)
	require.Equal(t, "attempt\n", out)
}
