// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestGetElementCommand_ValidateDirectLookupKey(t *testing.T) {
	resetGetElementFlagState()
	t.Cleanup(resetGetElementFlagState)

	flagGetElementKey = "2251799813689002"

	require.NoError(t, validateGetElementFlags(getElementCmd))
}

func TestGetElementCommand_RejectsInvalidKeys(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func()
		want  string
	}{
		{name: "key", setup: func() { flagGetElementKey = "not-a-key" }, want: `--key value "not-a-key" is not a valid key`},
		{name: "pi key", setup: func() { flagGetElementProcessKey = "not-a-key" }, want: `--pi-key value "not-a-key" is not a valid key`},
		{name: "pd key", setup: func() { flagGetElementProcessDefKey = "not-a-key" }, want: `--pd-key value "not-a-key" is not a valid key`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetGetElementFlagState()
			t.Cleanup(resetGetElementFlagState)
			tc.setup()

			err := validateGetElementFlags(getElementCmd)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestGetElementCommand_KeyedLookupRejectsSearchFiltersBeforeLookup(t *testing.T) {
	resetGetElementFlagState()
	t.Cleanup(resetGetElementFlagState)
	require.NoError(t, getElementCmd.Flags().Set("pi-key", "2251799813688001"))
	flagGetElementKey = "2251799813689002"
	flagGetElementProcessKey = "2251799813688001"

	err := validateGetElementFlags(getElementCmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--key cannot be combined with element search filters: --pi-key")
}

func TestGetElementCommand_ValidateSearchFlagsAndBuildsRequest(t *testing.T) {
	resetGetElementFlagState()
	t.Cleanup(resetGetElementFlagState)
	require.NoError(t, getElementCmd.Flags().Set("pi-key", "2251799813688001"))
	require.NoError(t, getElementCmd.Flags().Set("element-id", "ship-order"))
	require.NoError(t, getElementCmd.Flags().Set("state", "active"))
	require.NoError(t, getElementCmd.Flags().Set("type", "service_task"))
	require.NoError(t, getElementCmd.Flags().Set("pd-key", "2251799813687001"))
	require.NoError(t, getElementCmd.Flags().Set("bpmn-process-id", "order-process"))
	require.NoError(t, getElementCmd.Flags().Set("batch-size", "25"))
	require.NoError(t, getElementCmd.Flags().Set("limit", "50"))

	require.NoError(t, validateGetElementFlags(getElementCmd))

	request := newGetElementSearchRequest(getElementCmd)
	require.Equal(t, "2251799813688001", request.ProcessInstanceKey)
	require.Equal(t, "ship-order", request.ElementId)
	require.Equal(t, "ACTIVE", request.State)
	require.Equal(t, "SERVICE_TASK", request.Type)
	require.Equal(t, "2251799813687001", request.ProcessDefinitionKey)
	require.Equal(t, "order-process", request.BpmnProcessId)
	require.Equal(t, int32(25), request.BatchSize)
	require.Equal(t, int32(50), request.Limit)
}

func TestGetElementCommand_UnfilteredSearchIsAllowed(t *testing.T) {
	resetGetElementFlagState()
	t.Cleanup(resetGetElementFlagState)

	require.NoError(t, validateGetElementFlags(getElementCmd))

	request := newGetElementSearchRequest(getElementCmd)
	require.False(t, request.HasKey())
	require.False(t, request.HasSearchFilters())
	require.Equal(t, int32(consts.MaxPISearchSize), request.BatchSize)
}

func TestGetElementCommand_RejectsInvalidSearchControls(t *testing.T) {
	for _, tc := range []struct {
		name string
		flag string
		val  string
		want string
	}{
		{name: "state", flag: "state", val: "running", want: `invalid value for --state: "running"`},
		{name: "type", flag: "type", val: "mail_task", want: `invalid value for --type: "mail_task"`},
		{name: "batch size", flag: "batch-size", val: "0", want: "invalid value for --batch-size: 0"},
		{name: "limit", flag: "limit", val: "0", want: "--limit must be positive integer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetGetElementFlagState()
			t.Cleanup(resetGetElementFlagState)
			require.NoError(t, getElementCmd.Flags().Set(tc.flag, tc.val))

			err := validateGetElementFlags(getElementCmd)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

// TestGetElementCommand_RejectsTotalModeConflicts keeps total-only output
// script-safe by preventing other machine output modes from changing it.
func TestGetElementCommand_RejectsTotalModeConflicts(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func()
		want  string
	}{
		{name: "json", setup: func() { flagViewAsJson = true }, want: "--total cannot be combined with --json"},
		{name: "keys only", setup: func() { flagViewKeysOnly = true }, want: "--total cannot be combined with --keys-only"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetGetElementFlagState()
			t.Cleanup(func() {
				resetGetElementFlagState()
				flagViewAsJson = false
				flagViewKeysOnly = false
			})
			flagGetElementTotal = true
			tc.setup()

			err := validateGetElementFlags(getElementCmd)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

// TestGetElementCommand_RejectsListenerOutputConflicts ensures nested listener output is validated locally.
func TestGetElementCommand_RejectsListenerOutputConflicts(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func()
		want  string
	}{
		{name: "keys only", setup: func() { flagViewKeysOnly = true }, want: "--with-listeners cannot be combined with --keys-only"},
		{name: "total", setup: func() { flagGetElementTotal = true }, want: "--with-listeners cannot be combined with --total"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetGetElementFlagState()
			t.Cleanup(func() {
				resetGetElementFlagState()
				flagViewKeysOnly = false
			})
			flagGetElementWithListeners = true
			tc.setup()

			err := validateGetElementFlags(getElementCmd)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestGetElementCommand_SearchHumanOutput(t *testing.T) {
	var requests []string
	srv := newElementSearchServer(t, &requests, `{
  "items": [
    {
      "elementInstanceKey": "2251799813689002",
      "elementId": "ship-order",
      "elementName": "Ship order",
      "type": "SERVICE_TASK",
      "state": "ACTIVE",
      "startDate": "2026-07-15T10:12:01Z",
      "processInstanceKey": "2251799813688001",
      "rootProcessInstanceKey": "2251799813688001",
      "processDefinitionId": "order-process",
      "processDefinitionKey": "2251799813687001",
      "tenantId": "tenant-a",
      "hasIncident": false
    }
  ],
  "page": {
    "totalItems": 1,
    "hasMoreTotalItems": false
  }
}`)
	t.Cleanup(srv.Close)
	cfgPath := testx.WriteTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForElementTest(t,
		"--config", cfgPath,
		"get", "element",
		"--pi-key", "2251799813688001",
		"--state", "active",
		"--type", "service_task",
		"--batch-size", "5",
		"--limit", "7",
	)

	require.Equal(t, []string{"POST /v2/element-instances/search"}, requests)
	require.Contains(t, output, "2251799813689002")
	require.Contains(t, output, "SERVICE_TASK")
	require.Contains(t, output, "SERVICE_TASK ship-order ACTIVE")
	require.Contains(t, output, "pi:2251799813688001")
	require.NotContains(t, output, "element:ship-order")
	require.Contains(t, output, "found: 1")
}

// TestGetElementCommand_SearchJSONOutput emits one shared result envelope with
// a stable total/items payload for bounded element search results.
func TestGetElementCommand_SearchJSONOutput(t *testing.T) {
	var bodies []map[string]any
	srv := newElementSearchServerResponses(t, &bodies, `{
  "items": [
    {
      "elementInstanceKey": "2251799813689002",
      "elementId": "ship-order",
      "type": "SERVICE_TASK",
      "state": "ACTIVE",
      "startDate": "2026-07-15T10:12:01Z",
      "processInstanceKey": "2251799813688001",
      "processDefinitionKey": "2251799813687001",
      "tenantId": "tenant-a",
      "hasIncident": true,
      "incidentKey": "2251799813687777"
    },
    {
      "elementInstanceKey": "2251799813689003",
      "elementId": "finish-order",
      "type": "END_EVENT",
      "state": "COMPLETED",
      "startDate": "2026-07-15T10:12:02Z",
      "endDate": "2026-07-15T10:12:03Z",
      "processInstanceKey": "2251799813688001",
      "processDefinitionKey": "2251799813687001",
      "tenantId": "tenant-a",
      "hasIncident": false
    }
  ],
  "page": {
    "totalItems": 2,
    "hasMoreTotalItems": false
  }
}`)
	t.Cleanup(srv.Close)
	cfgPath := testx.WriteTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForElementTest(t, "--config", cfgPath, "--json", "get", "element", "--pi-key", "2251799813688001", "--limit", "2")

	require.Len(t, bodies, 1)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "get element", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, float64(2), payload["total"])
	items := payload["items"].([]any)
	require.Len(t, items, 2)
	first := requireJSONObject(t, items[0])
	require.Equal(t, "2251799813689002", first["elementInstanceKey"])
	require.Equal(t, "ship-order", first["elementId"])
	require.Equal(t, "SERVICE_TASK", first["type"])
	require.Equal(t, true, first["hasIncident"])
	require.Equal(t, "2251799813687777", first["incidentKey"])
}

// TestGetElementCommand_SearchKeysOnlyOutput emits only element instance keys
// so automation can pipe runtime element identifiers without row text.
func TestGetElementCommand_SearchKeysOnlyOutput(t *testing.T) {
	var bodies []map[string]any
	srv := newElementSearchServerResponses(t, &bodies, `{
  "items": [
    {"elementInstanceKey": "2251799813689002", "state": "ACTIVE"},
    {"elementInstanceKey": "2251799813689003", "state": "COMPLETED"}
  ],
  "page": {"totalItems": 2, "hasMoreTotalItems": false}
}`)
	t.Cleanup(srv.Close)
	cfgPath := testx.WriteTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForElementTest(t, "--config", cfgPath, "--keys-only", "get", "element", "--pi-key", "2251799813688001", "--limit", "2")

	require.Equal(t, "2251799813689002\n2251799813689003\n", output)
	require.Len(t, bodies, 1)
}

// TestGetElementCommand_SearchTotalOnlyOutput keeps total mode quiet by
// rendering only the numeric count reported by the backend when exact.
func TestGetElementCommand_SearchTotalOnlyOutput(t *testing.T) {
	var bodies []map[string]any
	srv := newElementSearchServerResponses(t, &bodies, `{
  "items": [
    {"elementInstanceKey": "2251799813689002", "state": "ACTIVE"}
  ],
  "page": {"totalItems": 42, "hasMoreTotalItems": false}
}`)
	t.Cleanup(srv.Close)
	cfgPath := testx.WriteTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForElementTest(t, "--config", cfgPath, "get", "element", "--pi-key", "2251799813688001", "--total")

	require.Equal(t, "42\n", output)
	require.Len(t, bodies, 1)
}

func TestGetElementCommand_KeyedLookupHumanOutput(t *testing.T) {
	var requests []string
	srv := newElementLookupServer(t, &requests, http.StatusOK, `{
  "elementInstanceKey": "2251799813689002",
  "elementId": "ship-order",
  "elementName": "Ship order",
  "type": "SERVICE_TASK",
  "state": "ACTIVE",
  "startDate": "2026-07-15T10:12:01Z",
  "processInstanceKey": "2251799813688001",
  "rootProcessInstanceKey": "2251799813688001",
  "processDefinitionId": "order-process",
  "processDefinitionKey": "2251799813687001",
  "tenantId": "tenant-a",
  "hasIncident": true,
  "incidentKey": "2251799813687777"
}`)
	t.Cleanup(srv.Close)
	cfgPath := testx.WriteTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestGetElementCommand_KeyedLookupHumanOutputHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.NoError(t, err)
	require.Equal(t, []string{"GET /v2/element-instances/2251799813689002"}, requests)
	require.Contains(t, string(output), "2251799813689002")
	require.Contains(t, string(output), "SERVICE_TASK")
	require.Contains(t, string(output), "SERVICE_TASK ship-order ACTIVE")
	require.Contains(t, string(output), "s:2026-07-15T10:12:01.000")
	require.Contains(t, string(output), "pi:2251799813688001")
	require.NotContains(t, string(output), "element:ship-order")
	require.Contains(t, string(output), "inc!:2251799813687777")
	require.NotContains(t, string(output), "found:")
}

func TestGetElementCommand_KeyedLookupHumanOutputHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "element", "--key", "2251799813689002"}

	Execute()
}

func TestGetElementCommand_KeyedLookupJSONOutput(t *testing.T) {
	var requests []string
	srv := newElementLookupServer(t, &requests, http.StatusOK, `{
  "elementInstanceKey": "2251799813689002",
  "elementId": "ship-order",
  "type": "SERVICE_TASK",
  "state": "ACTIVE",
  "startDate": "2026-07-15T10:12:01Z",
  "processInstanceKey": "2251799813688001",
  "processDefinitionKey": "2251799813687001",
  "tenantId": "tenant-a",
  "hasIncident": false
}`)
	t.Cleanup(srv.Close)
	cfgPath := testx.WriteTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForElementTest(t, "--config", cfgPath, "--json", "get", "element", "--key", "2251799813689002")

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "get element", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "2251799813689002", payload["elementInstanceKey"])
	require.Equal(t, "ship-order", payload["elementId"])
	require.Equal(t, "SERVICE_TASK", payload["type"])
	require.Equal(t, []string{"GET /v2/element-instances/2251799813689002"}, requests)
}

// TestGetElementCommand_KeyedLookupWithListenersHumanOutput nests matched listener jobs under the fetched element row.
func TestGetElementCommand_KeyedLookupWithListenersHumanOutput(t *testing.T) {
	var requests []string
	srv := newElementWithListenersServer(t, &requests, []string{`{
  "elementInstanceKey": "2251799813689002",
  "elementId": "ship-order",
  "type": "SERVICE_TASK",
  "state": "ACTIVE",
  "startDate": "2026-07-15T10:12:01Z",
  "processInstanceKey": "2251799813688001",
  "tenantId": "tenant-a",
  "hasIncident": false
}`}, []string{
		`{"items":[{"jobKey":"2251799813689101","kind":"EXECUTION_LISTENER","listenerEventType":"START","type":"audit-start","state":"CREATED","retries":3,"processInstanceKey":"2251799813688001","elementInstanceKey":"2251799813689002","elementId":"ship-order","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
		`{"items":[{"jobKey":"2251799813689102","kind":"TASK_LISTENER","listenerEventType":"COMPLETING","type":"audit-task","state":"FAILED","retries":0,"processInstanceKey":"2251799813688001","elementInstanceKey":"2251799813689002","elementId":"ship-order","tenantId":"tenant-a","errorCode":"LISTENER_FAILED","errorMessage":"worker failed"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
	})
	t.Cleanup(srv.Close)
	cfgPath := testx.WriteTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForElementTest(t, "--config", cfgPath, "get", "element", "--key", "2251799813689002", "--with-listeners")

	require.Equal(t, []string{
		"GET /v2/element-instances/2251799813689002",
		"POST /v2/jobs/search",
		"POST /v2/jobs/search",
	}, requests)
	require.Contains(t, output, "2251799813689002")
	require.Contains(t, output, "└─ listeners:")
	require.Contains(t, output, "2251799813689101 EXECUTION_LISTENER lsnr:START")
	require.Contains(t, output, "2251799813689102 TASK_LISTENER")
	require.Contains(t, output, "lsnr:COMPLETING FAILED")
	require.Contains(t, output, "tp:audit-task")
	require.Contains(t, output, "r:0")
	require.Contains(t, output, "ec:LISTENER_FAILED")
}

// TestGetElementCommand_SearchWithListenersJSONOutput preserves requested-empty arrays and omits unmatched jobs.
func TestGetElementCommand_SearchWithListenersJSONOutput(t *testing.T) {
	var requests []string
	srv := newElementWithListenersServer(t, &requests, []string{`{
  "items": [
    {"elementInstanceKey":"2251799813689002","elementId":"ship-order","type":"SERVICE_TASK","state":"ACTIVE","processInstanceKey":"2251799813688001","tenantId":"tenant-a","hasIncident":false},
    {"elementInstanceKey":"2251799813689003","elementId":"finish-order","type":"END_EVENT","state":"COMPLETED","processInstanceKey":"2251799813688001","tenantId":"tenant-a","hasIncident":false}
  ],
  "page": {"totalItems":2,"hasMoreTotalItems":false}
}`}, []string{
		`{"items":[{"jobKey":"2251799813689101","kind":"EXECUTION_LISTENER","listenerEventType":"START","type":"audit-start","state":"CREATED","retries":3,"processInstanceKey":"2251799813688001","elementInstanceKey":"2251799813689002","elementId":"ship-order","tenantId":"tenant-a"},{"jobKey":"2251799813689999","kind":"EXECUTION_LISTENER","processInstanceKey":"2251799813688001","elementInstanceKey":"2251799813689998"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
		`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`,
	})
	t.Cleanup(srv.Close)
	cfgPath := testx.WriteTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForElementTest(t, "--config", cfgPath, "--json", "get", "element", "--pi-key", "2251799813688001", "--with-listeners")

	require.Equal(t, []string{"POST /v2/element-instances/search", "POST /v2/jobs/search", "POST /v2/jobs/search"}, requests)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	payload := requireJSONObject(t, envelope["payload"])
	items := payload["items"].([]any)
	require.Len(t, items, 2)
	first := requireJSONObject(t, items[0])
	require.NotNil(t, first["listeners"], output)
	firstListeners := first["listeners"].([]any)
	require.Len(t, firstListeners, 1)
	require.Equal(t, "2251799813689101", requireJSONObject(t, firstListeners[0])["jobKey"])
	second := requireJSONObject(t, items[1])
	require.Empty(t, second["listeners"].([]any))
}

func TestGetElementCommand_KeyedLookupUnsupportedV87(t *testing.T) {
	cfgPath := testx.WriteTestConfigForVersion(t, "http://camunda.example.test", "8.7")

	output, err := testx.RunCmdSubprocess(t, "TestGetElementCommand_KeyedLookupUnsupportedV87Helper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	require.Contains(t, string(output), "unsupported")
	require.Contains(t, string(output), "element lookup requires Camunda 8.8 or newer")
}

func TestGetElementCommand_KeyedLookupUnsupportedV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "element", "--key", "2251799813689002"}

	Execute()
}

func newElementLookupServer(t *testing.T, requests *[]string, status int, response string) *httptest.Server {
	t.Helper()
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/element-instances/2251799813689002", r.URL.Path)
		*requests = append(*requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
}

func newElementSearchServer(t *testing.T, requests *[]string, response string) *httptest.Server {
	t.Helper()
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/element-instances/search", r.URL.Path)
		*requests = append(*requests, r.Method+" "+r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		filter := body["filter"].(map[string]any)
		require.Equal(t, "2251799813688001", filter["processInstanceKey"])
		require.Equal(t, "ACTIVE", filter["state"])
		require.Equal(t, "SERVICE_TASK", filter["type"])
		page := body["page"].(map[string]any)
		require.Equal(t, float64(0), page["from"])
		require.Equal(t, float64(5), page["limit"])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
}

// newElementSearchServerResponses captures element search request bodies while
// returning one generated-client-compatible response per request.
func newElementSearchServerResponses(t *testing.T, bodies *[]map[string]any, responses ...string) *httptest.Server {
	t.Helper()
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/element-instances/search", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		*bodies = append(*bodies, body)
		responseIndex := len(*bodies) - 1
		if responseIndex >= len(responses) {
			responseIndex = len(responses) - 1
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[responseIndex]))
	}))
}

// newElementWithListenersServer serves generated-compatible element and job search responses.
func newElementWithListenersServer(t *testing.T, requests *[]string, elementResponses []string, jobResponses []string) *httptest.Server {
	t.Helper()
	var elementCount int
	var jobCount int
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/element-instances/2251799813689002":
			require.Equal(t, http.MethodGet, r.Method)
			*requests = append(*requests, r.Method+" "+r.URL.Path)
			response := elementResponses[min(elementCount, len(elementResponses)-1)]
			elementCount++
			_, _ = w.Write([]byte(response))
		case "/v2/element-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			*requests = append(*requests, r.Method+" "+r.URL.Path)
			response := elementResponses[min(elementCount, len(elementResponses)-1)]
			elementCount++
			_, _ = w.Write([]byte(response))
		case "/v2/jobs/search":
			require.Equal(t, http.MethodPost, r.Method)
			*requests = append(*requests, r.Method+" "+r.URL.Path)
			response := jobResponses[min(jobCount, len(jobResponses)-1)]
			jobCount++
			_, _ = w.Write([]byte(response))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
}

func executeRootForElementTest(t *testing.T, args ...string) string {
	t.Helper()

	resetGetElementFlagState()
	t.Cleanup(resetGetElementFlagState)

	root := Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetGetElementFlagState()

	_, err := root.ExecuteC()
	require.NoError(t, err)
	return buf.String()
}

func resetGetElementFlagState() {
	flagGetElementKey = ""
	flagGetElementProcessKey = ""
	flagGetElementID = ""
	flagGetElementState = ""
	flagGetElementType = ""
	flagGetElementProcessDefKey = ""
	flagGetElementBpmnProcessID = ""
	flagGetElementBatchSize = consts.MaxPISearchSize
	flagGetElementLimit = 0
	flagGetElementTotal = false
	flagGetElementWithListeners = false
	testx.ResetCommandTreeFlags(getElementCmd)
}
