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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestGetUserTaskCommand_SearchBuildsAllPredicates verifies normalized state,
// exact string predicates, selector keys, tenant scope, bounds, and AND composition.
func TestGetUserTaskCommand_SearchBuildsAllPredicates(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return userTaskSearchResponse(1, false, "", "2251799815391233")
	})
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--tenant", "tenant-a", "--json", "get", "ut",
		"--pi-key", "2251799813711967", "--pd-key", "2251799813689000",
		"--bpmn-process-id", "invoice", "--element-id", "approve_invoice",
		"--state", "created", "--assignee", "AliceCase", "--candidate-user", "BobCase",
		"--candidate-group", "AccountingCase", "--batch-size", "2", "--limit", "2")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	require.Contains(t, stdout, `"total": 1`)

	got := requests.snapshot(t)
	require.Len(t, got, 1)
	filter := requireJSONMap(t, got[0]["filter"])
	require.Equal(t, "2251799813711967", filter["processInstanceKey"])
	require.Equal(t, "2251799813689000", filter["processDefinitionKey"])
	require.Equal(t, "invoice", filter["processDefinitionId"])
	require.Equal(t, "approve_invoice", filter["elementId"])
	require.Equal(t, "CREATED", jsonFilterValue(t, filter["state"]))
	require.Equal(t, "AliceCase", jsonFilterValue(t, filter["assignee"]))
	require.Equal(t, "BobCase", jsonFilterValue(t, filter["candidateUser"]))
	require.Equal(t, "AccountingCase", jsonFilterValue(t, filter["candidateGroup"]))
	require.Equal(t, "tenant-a", jsonFilterValue(t, filter["tenantId"]))
	require.Equal(t, float64(2), jsonPageLimit(t, got[0]))
}

// TestGetUserTaskCommand_SearchDefaultsAndStates verifies empty implicit stdin
// selects discovery, all omits the state predicate, and all nine states normalize.
func TestGetUserTaskCommand_SearchDefaultsAndStates(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return userTaskSearchResponse(0, false, "")
	})
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", "ut")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	require.Equal(t, "found: 0\n", stdout)
	filter := requireJSONMap(t, requests.snapshot(t)[0]["filter"])
	require.NotContains(t, filter, "state")

	stdout, stderr, err = runGetUserTaskCommand(t, configPath, "", "get", "ut", "--state", "ALL")
	require.NoError(t, err, stderr)
	require.Equal(t, "found: 0\n", stdout)
	filter = requireJSONMap(t, requests.snapshot(t)[1]["filter"])
	require.NotContains(t, filter, "state")

	states := []string{"ASSIGNING", "CANCELED", "CANCELING", "COMPLETED", "COMPLETING", "CREATED", "CREATING", "FAILED", "UPDATING"}
	for index, state := range states {
		_, stderr, err = runGetUserTaskCommand(t, configPath, "", "get", "ut", "--state", strings.ToLower(state))
		require.NoError(t, err, stderr)
		filter = requireJSONMap(t, requests.snapshot(t)[index+2]["filter"])
		require.Equal(t, state, jsonFilterValue(t, filter["state"]))
	}
}

// TestGetUserTaskCommand_SearchEmptyModes verifies completed empty discovery
// renders once without synthetic rows or extra requests in every output mode.
func TestGetUserTaskCommand_SearchEmptyModes(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return userTaskSearchResponse(0, false, "")
	})
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "human", args: []string{"get", "ut"}, want: "found: 0\n"},
		{name: "keys", args: []string{"--keys-only", "get", "ut"}, want: ""},
		{name: "quiet human", args: []string{"--quiet", "get", "ut"}, want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := len(requests.snapshot(t))
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", test.args...)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			require.Equal(t, test.want, stdout)
			require.Len(t, requests.snapshot(t), before+1)
		})
	}

	before := len(requests.snapshot(t))
	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--json", "get", "ut")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	var envelope struct {
		Outcome string `json:"outcome"`
		Payload struct {
			Total int64 `json:"total"`
			Items []any `json:"items"`
		} `json:"payload"`
	}
	decoder := json.NewDecoder(strings.NewReader(stdout))
	require.NoError(t, decoder.Decode(&envelope))
	require.Equal(t, "succeeded", envelope.Outcome)
	require.Zero(t, envelope.Payload.Total)
	require.NotNil(t, envelope.Payload.Items)
	require.Empty(t, envelope.Payload.Items)
	var extra any
	require.Error(t, decoder.Decode(&extra), "JSON output must contain exactly one envelope")
	require.Len(t, requests.snapshot(t), before+1)
}

// TestGetUserTaskCommand_SearchRejectsInvalidInputBeforeRequests verifies state,
// selector, bound, and total-mode conflicts fail before backend discovery.
func TestGetUserTaskCommand_SearchRejectsInvalidInputBeforeRequests(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return userTaskSearchResponse(0, false, "")
	})
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "assigned", args: []string{"get", "ut", "--state", "assigned"}, want: "invalid value for --state"},
		{name: "unknown state", args: []string{"get", "ut", "--state", "waiting"}, want: "invalid value for --state"},
		{name: "bad pi key", args: []string{"get", "ut", "--pi-key", "123"}, want: "--pi-key value"},
		{name: "bad pd key", args: []string{"get", "ut", "--pd-key", "123"}, want: "--pd-key value"},
		{name: "zero batch", args: []string{"get", "ut", "--batch-size", "0"}, want: "invalid value for --batch-size"},
		{name: "zero limit", args: []string{"get", "ut", "--limit", "0"}, want: "--limit must be positive"},
		{name: "total limit", args: []string{"get", "ut", "--total", "--limit", "1"}, want: "--total cannot be combined with --limit"},
		{name: "total json", args: []string{"--json", "get", "ut", "--total"}, want: "--total cannot be combined with --json"},
		{name: "total keys", args: []string{"--keys-only", "get", "ut", "--total"}, want: "--total cannot be combined with --keys-only"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := len(requests.snapshot(t))
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", test.args...)
			require.Error(t, err)
			require.Empty(t, stdout)
			require.Contains(t, stderr, test.want)
			require.Len(t, requests.snapshot(t), before)
		})
	}
}

// TestGetUserTaskCommand_SearchTraversesSparsePagesAndHonorsLimit verifies exact
// totals across cursor pages, sparse intermediate pages, and nonfinal limits on every adapter.
func TestGetUserTaskCommand_SearchTraversesSparsePagesAndHonorsLimit(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, sparse := range []bool{false, true} {
			for _, limited := range []bool{false, true} {
				for _, jsonOutput := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/sparse=%t/limited=%t/json=%t", version, sparse, limited, jsonOutput), func(t *testing.T) {
						responses := []string{
							userTaskSearchResponse(3, false, "cursor-a", "2251799815391233"),
							userTaskSearchResponse(3, false, "cursor-b", "2251799815391234"),
							userTaskSearchResponse(3, false, "cursor-c", "2251799815391235"),
						}
						if sparse {
							responses = append(responses[:1], append([]string{userTaskSearchResponse(3, false, "cursor-sparse")}, responses[1:]...)...)
						}
						server, requests := newGetUserTaskSearchServer(t, func(index int, _ map[string]any) string {
							require.Less(t, index, len(responses))
							return responses[index]
						})
						configPath := testx.WriteTestConfigForVersion(t, server.URL, version)
						args := []string{"--keys-only", "get", "ut", "--batch-size", "1"}
						if jsonOutput {
							args[0] = "--json"
						}
						wantCount := 3
						wantRequests := len(responses)
						if limited {
							args = append(args, "--limit", "2")
							wantCount = 2
							wantRequests--
						}
						stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", args...)
						require.NoError(t, err, stderr)
						require.Empty(t, stderr)
						if jsonOutput {
							requireSucceededUserTaskEnvelope(t, stdout, int64(wantCount))
						} else {
							want := "2251799815391233\n2251799815391234\n"
							if !limited {
								want += "2251799815391235\n"
							}
							require.Equal(t, want, stdout)
						}
						got := requests.snapshot(t)
						require.Len(t, got, wantRequests)
						for i := 1; i < len(got); i++ {
							var previous struct {
								Page struct {
									EndCursor string `json:"endCursor"`
								} `json:"page"`
							}
							require.NoError(t, json.Unmarshal([]byte(responses[i-1]), &previous))
							require.Equal(t, previous.Page.EndCursor, jsonPageAfter(t, got[i]))
						}
					})
				}
			}
		}
	}
}

// TestGetUserTaskCommand_SearchLimitBoundaries verifies limits before, at, and
// after a backend page boundary stop with the exact required request count.
func TestGetUserTaskCommand_SearchLimitBoundaries(t *testing.T) {
	tests := []struct {
		name         string
		limit        string
		wantKeys     string
		wantRequests int
	}{
		{name: "before", limit: "1", wantKeys: "2251799815391233\n", wantRequests: 1},
		{name: "at", limit: "3", wantKeys: "2251799815391233\n2251799815391234\n2251799815391235\n", wantRequests: 1},
		{name: "after", limit: "5", wantKeys: "2251799815391233\n2251799815391234\n2251799815391235\n2251799815391236\n", wantRequests: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newGetUserTaskSearchServer(t, func(index int, _ map[string]any) string {
				if index == 0 {
					return userTaskSearchResponse(4, false, "cursor-a", "2251799815391233", "2251799815391234", "2251799815391235")
				}
				return userTaskSearchResponse(4, false, "", "2251799815391236")
			})
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--keys-only", "get", "ut", "--batch-size", "3", "--limit", test.limit)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			require.Equal(t, test.wantKeys, stdout)
			require.Len(t, requests.snapshot(t), test.wantRequests)
		})
	}
}

// TestGetUserTaskCommand_TotalUsesExactAndCappedTraversal verifies numeric zero,
// exact fast-path totals, capped fallback traversal, and quiet preservation.
func TestGetUserTaskCommand_TotalUsesExactAndCappedTraversal(t *testing.T) {
	tests := []struct {
		name      string
		responses []string
		want      string
	}{
		{name: "zero", responses: []string{userTaskSearchResponse(0, false, "")}, want: "0\n"},
		{name: "exact", responses: []string{userTaskSearchResponse(7, false, "cursor-unused")}, want: "7\n"},
		{name: "capped", responses: []string{
			userTaskSearchResponse(2, true, "cursor-a", "2251799815391233", "2251799815391234"),
			userTaskSearchResponse(3, false, "", "2251799815391235"),
		}, want: "3\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newGetUserTaskSearchServer(t, func(index int, _ map[string]any) string {
				return test.responses[index]
			})
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--quiet", "get", "ut", "--total", "--batch-size", "2")
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			require.Equal(t, test.want, stdout)
			require.Len(t, requests.snapshot(t), len(test.responses))
		})
	}
}

// TestGetUserTaskCommand_SearchSupportsEveryNativeVersion verifies command
// dispatch reaches each supported adapter and V87 rejects without a request.
func TestGetUserTaskCommand_SearchSupportsEveryNativeVersion(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return userTaskSearchResponse(1, false, "", "2251799815391233")
	})
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		configPath := testx.WriteTestConfigForVersion(t, server.URL, version)
		stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--tenant", "tenant-a", "--keys-only", "get", "ut", "--automation",
			"--pi-key", "2251799813711967", "--pd-key", "2251799813689000",
			"--bpmn-process-id", "invoice", "--element-id", "approve_invoice",
			"--state", "created", "--assignee", "alice", "--candidate-user", "bob", "--candidate-group", "accounting")
		require.NoError(t, err, stderr)
		require.Empty(t, stderr)
		require.Equal(t, "2251799815391233\n", stdout)
	}
	before := len(requests.snapshot(t))
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.7")
	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", "ut")
	require.Error(t, err)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "unsupported")
	require.Len(t, requests.snapshot(t), before)
}

// TestGetUserTaskCommand_SearchUsesEffectiveTenant verifies configured,
// explicit override, and all-tenant discovery scopes reach the backend filter.
func TestGetUserTaskCommand_SearchUsesEffectiveTenant(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return userTaskSearchResponse(0, false, "")
	})
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	config := fmt.Sprintf("app:\n  camunda_version: %q\n  tenant: %q\nauth:\n  mode: none\napis:\n  camunda_api:\n    base_url: %q\n", "8.9", "configured-tenant", server.URL)
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o600))

	_, stderr, err := runGetUserTaskCommand(t, configPath, "", "--json", "get", "ut")
	require.NoError(t, err, stderr)
	filter := requireJSONMap(t, requests.snapshot(t)[0]["filter"])
	require.Equal(t, "configured-tenant", jsonFilterValue(t, filter["tenantId"]))

	_, stderr, err = runGetUserTaskCommand(t, configPath, "", "--tenant", "override-tenant", "--json", "get", "ut")
	require.NoError(t, err, stderr)
	filter = requireJSONMap(t, requests.snapshot(t)[1]["filter"])
	require.Equal(t, "override-tenant", jsonFilterValue(t, filter["tenantId"]))

	_, stderr, err = runGetUserTaskCommand(t, configPath, "", "--all-tenants", "--json", "get", "ut")
	require.NoError(t, err, stderr)
	filter = requireJSONMap(t, requests.snapshot(t)[2]["filter"])
	require.NotContains(t, filter, "tenantId")
}

// TestGetUserTaskCommand_SearchRejectsMalformedBackendItems verifies a failed
// traversal cannot be rendered as a successful empty or partial collection.
func TestGetUserTaskCommand_SearchRejectsMalformedBackendItems(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return `{"items":[{"state":"CREATED","processInstanceKey":"2251799813711967"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`
	})
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--json", "get", "ut")
	require.Error(t, err)
	require.Empty(t, stderr)
	require.NotContains(t, stdout, `"outcome": "succeeded"`)
	require.Contains(t, stdout, `"outcome": "failed"`)
	require.Len(t, requests.snapshot(t), 1)
}

// TestGetUserTaskCommand_SearchReportsBackendFailure verifies HTTP failures
// retain command error classification and never emit a successful collection.
func TestGetUserTaskCommand_SearchReportsBackendFailure(t *testing.T) {
	var requests atomic.Int32
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		http.Error(writer, `{"message":"unavailable"}`, http.StatusServiceUnavailable)
	}))
	defer server.Close()
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", "ut")
	require.Error(t, err)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "unavailable")
	require.Equal(t, int32(1), requests.Load())
}

type capturedUserTaskSearchRequests struct {
	mu    sync.Mutex
	items []map[string]any
}

// snapshot returns an independently owned request list for stable assertions.
func (c *capturedUserTaskSearchRequests) snapshot(t *testing.T) []map[string]any {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]map[string]any(nil), c.items...)
}

// newGetUserTaskSearchServer captures generic generated-client request JSON and
// delegates deterministic page responses to each command scenario.
func newGetUserTaskSearchServer(t *testing.T, respond func(int, map[string]any) string) (*httptest.Server, *capturedUserTaskSearchRequests) {
	t.Helper()
	requests := new(capturedUserTaskSearchRequests)
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/v2/user-tasks/search" {
			http.NotFound(writer, request)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		requests.mu.Lock()
		index := len(requests.items)
		requests.items = append(requests.items, body)
		requests.mu.Unlock()
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(writer, respond(index, body))
	}))
	t.Cleanup(server.Close)
	return server, requests
}

// userTaskSearchResponse builds the stable response fields shared by supported versions.
func userTaskSearchResponse(total int64, capped bool, cursor string, keys ...string) string {
	items := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		items = append(items, map[string]any{
			"userTaskKey": key, "state": "CREATED", "name": "Approve invoice",
			"elementId": "approve_invoice", "assignee": "alice",
			"processInstanceKey": "2251799813711967", "processDefinitionKey": "2251799813689000",
			"processDefinitionId": "invoice", "tenantId": "tenant-a",
		})
	}
	payload := map[string]any{"items": items, "page": map[string]any{"totalItems": total, "hasMoreTotalItems": capped}}
	if cursor != "" {
		payload["page"].(map[string]any)["endCursor"] = cursor
	}
	encoded, _ := json.Marshal(payload)
	return string(encoded)
}

// requireJSONMap asserts and returns one decoded object.
func requireJSONMap(t *testing.T, value any) map[string]any {
	t.Helper()
	got, ok := value.(map[string]any)
	require.True(t, ok, "expected JSON object, got %#v", value)
	return got
}

// jsonFilterValue accepts scalar and generated equality-union JSON shapes.
func jsonFilterValue(t *testing.T, value any) string {
	t.Helper()
	if scalar, ok := value.(string); ok {
		return scalar
	}
	object := requireJSONMap(t, value)
	for _, key := range []string{"eq", "$eq"} {
		if scalar, ok := object[key].(string); ok {
			return scalar
		}
	}
	t.Fatalf("unsupported filter JSON %#v", value)
	return ""
}

// jsonPageLimit reads the limit from any supported generated pagination union.
func jsonPageLimit(t *testing.T, request map[string]any) float64 {
	t.Helper()
	page := requireJSONMap(t, request["page"])
	return page["limit"].(float64)
}

// jsonPageAfter reads a forward cursor from a captured request.
func jsonPageAfter(t *testing.T, request map[string]any) string {
	t.Helper()
	page := requireJSONMap(t, request["page"])
	return page["after"].(string)
}
