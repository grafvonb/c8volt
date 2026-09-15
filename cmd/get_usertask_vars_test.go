// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

type userTaskVariableFixtureValue struct {
	Name               string
	Value              string
	VariableKey        string
	ProcessInstanceKey string
	ScopeKey           string
	TenantID           string
	IsTruncated        *bool
	Truncated          *bool
}

type userTaskVariablePageFixture struct {
	Items     []userTaskVariableFixtureValue
	Total     int64
	Capped    bool
	EndCursor string
	Status    int
	ErrorBody string
}

type getUserTaskVariablesFixture struct {
	SearchRespond func(int, map[string]any) string
	VariablePages map[string][]userTaskVariablePageFixture
}

type capturedUserTaskVariableRequest struct {
	TaskKey        string
	TruncateValues string
	Body           map[string]any
}

type capturedGetUserTaskVariableRequests struct {
	mu               sync.Mutex
	taskKeys         []string
	searchRequests   []map[string]any
	variableRequests []capturedUserTaskVariableRequest
}

// snapshot returns independent request slices for deterministic assertions
// while keyed command reads may be running concurrently.
func (c *capturedGetUserTaskVariableRequests) snapshot() ([]string, []map[string]any, []capturedUserTaskVariableRequest) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.taskKeys...),
		append([]map[string]any(nil), c.searchRequests...),
		append([]capturedUserTaskVariableRequest(nil), c.variableRequests...)
}

// variableRequestCount reports how many effective-variable pages were read for
// one task without coupling command tests to request ordering across tasks.
func (c *capturedGetUserTaskVariableRequests) variableRequestCount(taskKey string) int {
	_, _, requests := c.snapshot()
	count := 0
	for _, request := range requests {
		if request.TaskKey == taskKey {
			count++
		}
	}
	return count
}

// newGetUserTaskVariablesServer provides only the user-task routes needed by
// variable command tests and keeps page/error behavior keyed by task identity.
func newGetUserTaskVariablesServer(t *testing.T, fixture getUserTaskVariablesFixture) (*httptest.Server, *capturedGetUserTaskVariableRequests) {
	t.Helper()
	requests := new(capturedGetUserTaskVariableRequests)
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/v2/user-tasks/search":
			serveUserTaskFixtureSearch(writer, request, fixture.SearchRespond, requests)
		case request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/effective-variables/search"):
			serveUserTaskVariableFixturePage(writer, request, fixture.VariablePages, requests)
		case request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/v2/user-tasks/"):
			serveUserTaskFixtureRead(writer, request, requests)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	return server, requests
}

// serveUserTaskFixtureSearch reuses the existing generic search response shape
// while capturing requests inside the composite variable fixture.
func serveUserTaskFixtureSearch(writer http.ResponseWriter, request *http.Request, respond func(int, map[string]any) string, requests *capturedGetUserTaskVariableRequests) {
	var body map[string]any
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	requests.mu.Lock()
	index := len(requests.searchRequests)
	requests.searchRequests = append(requests.searchRequests, body)
	requests.mu.Unlock()
	if respond == nil {
		respond = func(_ int, _ map[string]any) string { return userTaskSearchResponse(0, false, "") }
	}
	writer.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprint(writer, respond(index, body))
}

// serveUserTaskVariableFixturePage selects a page by task-local request count,
// allowing later-page failures without relying on cross-task request ordering.
func serveUserTaskVariableFixturePage(writer http.ResponseWriter, request *http.Request, pages map[string][]userTaskVariablePageFixture, requests *capturedGetUserTaskVariableRequests) {
	taskKey := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, "/v2/user-tasks/"), "/effective-variables/search")
	if taskKey == request.URL.Path || taskKey == "" || strings.Contains(taskKey, "/") {
		http.NotFound(writer, request)
		return
	}
	var body map[string]any
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	requests.mu.Lock()
	pageIndex := 0
	for _, captured := range requests.variableRequests {
		if captured.TaskKey == taskKey {
			pageIndex++
		}
	}
	requests.variableRequests = append(requests.variableRequests, capturedUserTaskVariableRequest{
		TaskKey: taskKey, TruncateValues: request.URL.Query().Get("truncateValues"), Body: body,
	})
	requests.mu.Unlock()
	taskPages := pages[taskKey]
	if pageIndex >= len(taskPages) {
		http.NotFound(writer, request)
		return
	}
	page := taskPages[pageIndex]
	if page.Status != 0 && page.Status != http.StatusOK {
		body := page.ErrorBody
		if body == "" {
			body = `{"message":"injected user-task variable fixture error"}`
		}
		http.Error(writer, body, page.Status)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write(userTaskVariableFixtureResponse(page))
}

// serveUserTaskFixtureRead returns the stable native task record already used
// by keyed command tests and records only valid-looking task routes.
func serveUserTaskFixtureRead(writer http.ResponseWriter, request *http.Request, requests *capturedGetUserTaskVariableRequests) {
	taskKey := strings.TrimPrefix(request.URL.Path, "/v2/user-tasks/")
	if taskKey == request.URL.Path || taskKey == "" || strings.Contains(taskKey, "/") {
		http.NotFound(writer, request)
		return
	}
	requests.mu.Lock()
	requests.taskKeys = append(requests.taskKeys, taskKey)
	requests.mu.Unlock()
	writer.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(writer, `{"userTaskKey":%q,"state":"CREATED","name":"Approve invoice","elementId":"approve_invoice","assignee":"alice","candidateUsers":["bob"],"candidateGroups":["accounting"],"processInstanceKey":"2251799813711967","elementInstanceKey":"2251799815391200","processDefinitionKey":"2251799813689000","processDefinitionId":"invoice","processDefinitionVersion":7,"tenantId":"tenant-a"}`, taskKey)
}

// userTaskVariableFixtureResponse deliberately builds raw JSON fields omitted
// by generated models so tests can distinguish empty values and both truncation names.
func userTaskVariableFixtureResponse(page userTaskVariablePageFixture) []byte {
	items := make([]map[string]any, 0, len(page.Items))
	for _, item := range page.Items {
		value := map[string]any{
			"name": item.Name, "value": item.Value, "variableKey": item.VariableKey,
			"processInstanceKey": item.ProcessInstanceKey, "scopeKey": item.ScopeKey,
		}
		if item.TenantID != "" {
			value["tenantId"] = item.TenantID
		}
		if item.IsTruncated != nil {
			value["isTruncated"] = *item.IsTruncated
		}
		if item.Truncated != nil {
			value["truncated"] = *item.Truncated
		}
		items = append(items, value)
	}
	payload := map[string]any{
		"items": items,
		"page":  map[string]any{"totalItems": page.Total, "hasMoreTotalItems": page.Capped},
	}
	if page.EndCursor != "" {
		payload["page"].(map[string]any)["endCursor"] = page.EndCursor
	}
	encoded, _ := json.Marshal(payload)
	return encoded
}

// TestGetUserTaskVariablesFixture verifies task-keyed page isolation, raw
// truncation fields, request capture, and deterministic injected failures.
func TestGetUserTaskVariablesFixture(t *testing.T) {
	apiTruncated, legacyTruncated := true, false
	server, requests := newGetUserTaskVariablesServer(t, getUserTaskVariablesFixture{
		VariablePages: map[string][]userTaskVariablePageFixture{
			"2251799815391233": {{
				Total: 1,
				Items: []userTaskVariableFixtureValue{{
					Name: "empty", Value: "", VariableKey: "901", ProcessInstanceKey: "2251799813711967",
					ScopeKey: "2251799815391200", TenantID: "tenant-a", IsTruncated: &apiTruncated, Truncated: &legacyTruncated,
				}},
			}},
			"2251799815391234": {{Status: http.StatusForbidden, ErrorBody: `{"message":"denied"}`}},
		},
	})

	response, err := http.Post(server.URL+"/v2/user-tasks/2251799815391233/effective-variables/search?truncateValues=false", "application/json", bytes.NewBufferString(`{"page":{"from":0,"limit":1000}}`))
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	require.Equal(t, http.StatusOK, response.StatusCode)
	var payload map[string]any
	require.NoError(t, json.NewDecoder(response.Body).Decode(&payload))
	item := payload["items"].([]any)[0].(map[string]any)
	require.Equal(t, "", item["value"])
	require.Equal(t, true, item["isTruncated"])
	require.Equal(t, false, item["truncated"])

	errorResponse, err := http.Post(server.URL+"/v2/user-tasks/2251799815391234/effective-variables/search?truncateValues=false", "application/json", bytes.NewBufferString(`{"page":{"from":0,"limit":1000}}`))
	require.NoError(t, err)
	t.Cleanup(func() { _ = errorResponse.Body.Close() })
	require.Equal(t, http.StatusForbidden, errorResponse.StatusCode)
	require.Equal(t, 1, requests.variableRequestCount("2251799815391233"))
	require.Equal(t, 1, requests.variableRequestCount("2251799815391234"))
	_, _, captured := requests.snapshot()
	require.Equal(t, "false", captured[0].TruncateValues)
	require.Equal(t, float64(0), requireJSONMap(t, captured[0].Body["page"])["from"])
}
