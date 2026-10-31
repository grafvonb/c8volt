// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const listenerBusinessIDRootKey = "2251799813685249"
const listenerBusinessIDOld = " old,\"|<tag>\n\t "

// listenerBusinessIDFixture owns all reads and rejects mutation and identity-enrichment requests.
func listenerBusinessIDFixture(t *testing.T, version string, rootIDs ...*string) (*httptest.Server, *testx.SafeSlice[string]) {
	t.Helper()
	requests := &testx.SafeSlice[string]{}
	root := map[string]any{"processInstanceKey": listenerBusinessIDRootKey, "processDefinitionKey": "9001", "processDefinitionId": "demo", "processDefinitionVersion": 1, "state": "ACTIVE", "startDate": "2026-09-16T12:00:00Z", "tenantId": "tenant", "hasIncident": false, "businessId": "new"}
	if len(rootIDs) > 0 {
		delete(root, "businessId")
		if rootIDs[0] != nil {
			root["businessId"] = *rootIDs[0]
		}
	}
	srv := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, version) {
			return
		}
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		var result any
		rows := []map[string]any{}
		switch r.Method + " " + r.URL.Path {
		case "GET /v2/process-instances/" + listenerBusinessIDRootKey:
			result = root
		case "POST /v2/process-instances/search":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter, _ := body["filter"].(map[string]any)
			if _, children := filter["parentProcessInstanceKey"]; !children {
				rows = append(rows, root)
			}
		case "POST /v2/element-instances/search":
			for _, key := range []string{"element-1", "element-empty"} {
				rows = append(rows, map[string]any{"elementInstanceKey": key, "elementId": key, "type": "SERVICE_TASK", "state": "ACTIVE", "startDate": "2026-09-16T12:00:00Z", "processInstanceKey": listenerBusinessIDRootKey, "processDefinitionKey": "9001", "tenantId": "tenant", "hasIncident": false})
			}
		case "POST /v2/jobs/search":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			if filter["kind"] == "EXECUTION_LISTENER" {
				for i, key := range []string{"job-old", "job-nil", "job-empty"} {
					row := map[string]any{"jobKey": key, "kind": "EXECUTION_LISTENER", "listenerEventType": "START", "type": "audit", "state": "CREATED", "retries": 1, "processInstanceKey": listenerBusinessIDRootKey, "elementInstanceKey": "element-1"}
					if i == 0 {
						row["businessId"] = listenerBusinessIDOld
					}
					if i == 2 {
						row["businessId"] = ""
					}
					rows = append(rows, row)
				}
			}
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if result == nil {
			result = map[string]any{"items": rows, "page": map[string]any{"totalItems": len(rows), "hasMoreTotalItems": false}}
		}
		require.NoError(t, json.NewEncoder(w).Encode(result))
	}))
	t.Cleanup(srv.Close)
	return srv, requests
}

// TestListenerBusinessIDCommands proves native-to-row identity through every listener adapter and both analysis views.
func TestListenerBusinessIDCommands(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, command := range []struct {
			name     string
			args     []string
			requests int
		}{
			{"element", []string{"get", "element", "--pi-key", listenerBusinessIDRootKey, "--with-listeners"}, 3},
			{"activity", []string{"get", "pi", "--key", listenerBusinessIDRootKey, "--with-elements", "--with-listeners"}, 4},
			{"walk", []string{"walk", "pi", "--key", listenerBusinessIDRootKey, "--with-elements", "--with-listeners"}, 6},
			{"analysis", []string{"ops", "analyse", "spi", "--key", listenerBusinessIDRootKey, "--with-listeners"}, 4},
			{"timeline", []string{"ops", "analyse", "spi", "--key", listenerBusinessIDRootKey, "--with-listeners", "--with-full-timeline"}, 4},
		} {
			for _, mode := range []string{"human", "quiet-human", "json", "quiet-json"} {
				t.Run(version+"/"+command.name+"/"+mode, func(t *testing.T) {
					srv, requests := listenerBusinessIDFixture(t, version)
					args := []string{"--config", testx.WriteTestConfigForVersion(t, srv.URL, version), "--no-indicator"}
					if strings.Contains(mode, "json") {
						args = append(args, "--json")
					}
					if strings.HasPrefix(mode, "quiet-") {
						args = append(args, "--quiet")
					}
					out, errOut := executeRootForProcessInstanceWithSeparateOutputs(t, append(args, command.args...)...)
					require.Empty(t, errOut)
					require.Len(t, requests.Snapshot(), command.requests, requests.Snapshot())
					if strings.Contains(mode, "json") {
						payload := decodeJobTaskBusinessIDEnvelope(t, out)
						seen := map[string]bool{}
						inspectListenerBusinessIDJSON(t, payload, version, seen)
						for _, key := range []string{"job-old", "job-nil", "job-empty", "element-empty"} {
							require.True(t, seen[key], "missing %s: %s", key, out)
						}
						if command.name != "element" {
							require.True(t, seen[listenerBusinessIDRootKey], out)
						}
						return
					}
					if mode == "quiet-human" && command.name == "element" {
						require.Empty(t, out)
						return
					}
					for _, key := range []string{"job-old", "job-nil", "job-empty"} {
						found := false
						for _, line := range strings.Split(out, "\n") {
							if !strings.Contains(line, key+" ") {
								continue
							}
							found = true
							if version != "8.10" || key == "job-nil" {
								require.NotContains(t, line, "business-id:")
								continue
							}
							want := ""
							if key == "job-old" {
								want = listenerBusinessIDOld
							}
							require.Contains(t, line, "business-id:"+strconv.Quote(want))
							require.NotContains(t, line, `business-id:"new"`)
						}
						require.True(t, found, "missing listener %s: %s", key, out)
					}
					if command.name != "element" && version == "8.10" {
						require.Equal(t, 1, strings.Count(out, `business-id:"new"`))
					}
				})
			}
		}
	}
}

// inspectListenerBusinessIDJSON checks independent facts and requested-empty sections without depending on wrapper layout.
func inspectListenerBusinessIDJSON(t *testing.T, value any, version string, seen map[string]bool) {
	t.Helper()
	switch x := value.(type) {
	case map[string]any:
		if key, ok := x["jobKey"].(string); ok {
			seen[key] = true
			if version != "8.10" || key == "job-nil" {
				require.NotContains(t, x, "businessId")
			} else if key == "job-old" {
				require.Equal(t, listenerBusinessIDOld, x["businessId"])
			} else {
				require.Equal(t, "", x["businessId"])
			}
		}
		if x["key"] == listenerBusinessIDRootKey {
			seen[listenerBusinessIDRootKey] = true
			if version == "8.10" {
				require.Equal(t, "new", x["businessId"])
			} else {
				require.NotContains(t, x, "businessId")
			}
		}
		if x["elementInstanceKey"] == "element-empty" {
			seen["element-empty"] = true
			require.Equal(t, []any{}, x["listeners"])
		}
		for _, child := range x {
			inspectListenerBusinessIDJSON(t, child, version, seen)
		}
	case []any:
		for _, child := range x {
			inspectListenerBusinessIDJSON(t, child, version, seen)
		}
	}
}

// TestSlowAnalysisBusinessIDKeys preserves keys-only output without incompatible listener enrichment.
func TestSlowAnalysisBusinessIDKeys(t *testing.T) {
	for _, quiet := range []bool{false, true} {
		t.Run(fmt.Sprint(quiet), func(t *testing.T) {
			srv, requests := listenerBusinessIDFixture(t, "8.10")
			args := []string{"--config", testx.WriteTestConfigForVersion(t, srv.URL, "8.10"), "--no-indicator", "--keys-only"}
			if quiet {
				args = append(args, "--quiet")
			}
			args = append(args, "ops", "analyse", "spi", "--key", listenerBusinessIDRootKey, "--with-full-timeline")
			out, errOut := executeRootForProcessInstanceWithSeparateOutputs(t, args...)
			require.Equal(t, listenerBusinessIDRootKey+"\n", out)
			require.Empty(t, errOut)
			require.Len(t, requests.Snapshot(), 2)
		})
	}
}

// TestSlowAnalysisBusinessIDAbsentRoot prevents a listener's identity from filling an absent or empty root observation.
func TestSlowAnalysisBusinessIDAbsentRoot(t *testing.T) {
	for _, rootID := range []*string{nil, new("")} {
		srv, requests := listenerBusinessIDFixture(t, "8.10", rootID)
		args := []string{"--config", testx.WriteTestConfigForVersion(t, srv.URL, "8.10"), "--no-indicator", "--json", "ops", "analyse", "spi", "--key", listenerBusinessIDRootKey, "--with-listeners"}
		out, errOut := executeRootForProcessInstanceWithSeparateOutputs(t, args...)
		require.Empty(t, errOut)
		require.Len(t, requests.Snapshot(), 4)
		payload := decodeJobTaskBusinessIDEnvelope(t, out)
		root := requireJSONObject(t, payload["items"].([]any)[0])
		if rootID == nil {
			require.NotContains(t, root, "businessId")
		} else {
			require.Equal(t, "", root["businessId"])
		}
		timeline := root["timeline"].([]any)
		listeners := requireJSONObject(t, timeline[0])["listeners"].([]any)
		require.Len(t, listeners, 3)
		seen := map[string]bool{}
		inspectListenerBusinessIDJSON(t, timeline, "8.10", seen)
		require.True(t, seen["job-old"])
	}
}

// TestListenerBusinessIDUnrequested avoids listener reads and preserves omitted sections when enrichment is absent.
func TestListenerBusinessIDUnrequested(t *testing.T) {
	for _, command := range []struct {
		name     string
		args     []string
		requests int
	}{
		{"element", []string{"get", "element", "--pi-key", listenerBusinessIDRootKey}, 1},
		{"activity", []string{"get", "pi", "--key", listenerBusinessIDRootKey, "--with-elements"}, 2},
		{"analysis", []string{"ops", "analyse", "spi", "--key", listenerBusinessIDRootKey}, 2},
	} {
		t.Run(command.name, func(t *testing.T) {
			srv, requests := listenerBusinessIDFixture(t, "8.10")
			args := []string{"--config", testx.WriteTestConfigForVersion(t, srv.URL, "8.10"), "--no-indicator", "--json"}
			out, errOut := executeRootForProcessInstanceWithSeparateOutputs(t, append(args, command.args...)...)
			require.Empty(t, errOut)
			require.Len(t, requests.Snapshot(), command.requests)
			decodeJobTaskBusinessIDEnvelope(t, out)
			require.NotContains(t, out, `"listeners":`)
		})
	}
}
