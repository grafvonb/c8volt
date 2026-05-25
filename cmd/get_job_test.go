// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestGetJobCommand_HumanOutput(t *testing.T) {
	var requests []string
	srv := newJobLookupServer(t, &requests, `{"items":[{"jobKey":"2251799813711967","state":"FAILED","retries":2,"deadline":"2026-05-08T10:15:00Z","processInstanceKey":"2251799813711000","elementInstanceKey":"2251799813711001","errorCode":"PAYMENT_ERROR","errorMessage":"worker failed","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "get", "job", "--key", "2251799813711967")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Equal(t, "2251799813711967 tenant-a FAILED pi:2251799813711000 ei:2251799813711001 r:2 d:2026-05-08T10:15:00.000 ec:PAYMENT_ERROR err:worker failed\n", output)
}

func TestGetJobCommand_HumanOutputKeepsLongErrorMessageInlineByDefault(t *testing.T) {
	longMessage := "Process instance could not be deleted. Error: Failed DELETE to https://example.invalid/orchestration/v1/process-instances/6755399441384051, due to Unsuccessful response: Code 400, body: {\"status\":400,\"message\":\"Process instances needs to be in one of the states [COMPLETED, CANCELED]\"}"
	var requests []string
	response := `{"items":[{"jobKey":"2251799814014237","state":"FAILED","retries":0,"deadline":"2026-04-23T01:07:49Z","processInstanceKey":"2251799814014230","elementInstanceKey":"2251799814014236","errorMessage":` + strconv.Quote(longMessage) + `,"tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`
	srv := newJobLookupServer(t, &requests, response)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForJobTest(t, "--config", cfgPath, "get", "job", "--key", "2251799814014237")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Equal(t, "2251799814014237 tenant-a FAILED pi:2251799814014230 ei:2251799814014236 r:0 d:2026-04-23T01:07:49.000 err:"+longMessage+"\n", output)
}

func TestGetJobCommand_HumanOutputTruncatesErrorMessageWhenLimitIsSet(t *testing.T) {
	var requests []string
	response := `{"items":[{"jobKey":"2251799814014237","state":"FAILED","retries":0,"errorMessage":"Process instance could not be deleted","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`
	srv := newJobLookupServer(t, &requests, response)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForJobTest(t, "--config", cfgPath, "get", "job", "--key", "2251799814014237", "--error-message-limit", "16")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Equal(t, "2251799814014237 tenant-a FAILED r:0 err:Process instance...\n", output)
}

func TestGetJobCommand_RejectsJSONErrorMessageLimit(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetGetJobFlagState()
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
		resetGetJobFlagState()
		flagViewAsJson = false
	})

	flagViewAsJson = true
	flagGetJobKey = "2251799814014237"
	require.NoError(t, getJobCmd.Flags().Set("error-message-limit", "16"))

	err := validateGetJobFlags(getJobCmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--error-message-limit cannot be combined with --json")
}

func TestGetJobCommand_JSONOutput(t *testing.T) {
	var requests []string
	srv := newJobLookupServer(t, &requests, `{"items":[{"jobKey":"2251799813711967","state":"FAILED","retries":2,"processInstanceKey":"2251799813711000","elementInstanceKey":"2251799813711001","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "--json", "get", "job", "--key", "2251799813711967")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "get job", envelope["command"])
	job := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "2251799813711967", job["key"])
	require.Equal(t, "FAILED", job["state"])
	require.Equal(t, float64(2), job["retries"])
	require.Equal(t, "2251799813711000", job["processInstanceKey"])
	require.Equal(t, "2251799813711001", job["elementInstanceKey"])
	require.Equal(t, "tenant-a", job["tenantId"])
}

func TestGetJobCommand_KeyedLookupRejectsSearchFiltersBeforeLookup(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetGetJobFlagState()
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
		resetGetJobFlagState()
	})

	flagGetJobKey = "2251799813711967"
	flagGetJobState = "FAILED"
	require.NoError(t, getJobCmd.Flags().Set("state", "FAILED"))

	err := validateGetJobFlags(getJobCmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--key cannot be combined with job search filters: --state")
}

// TestGetJobCommand_SearchModeAcceptsNoKey verifies omitted --key now selects
// bounded list/search mode instead of the old keyed-only validation path.
func TestGetJobCommand_SearchModeAcceptsNoKey(t *testing.T) {
	var bodies []map[string]any
	srv := newJobSearchServer(t, &bodies, `{"items":[{"jobKey":"2251799813711967","state":"FAILED","retries":0,"type":"payment-worker","worker":"worker-a","kind":"BPMN_ELEMENT","processInstanceKey":"2251799813711000","elementInstanceKey":"2251799813711001","elementId":"charge-card","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForJobTest(t, "--config", cfgPath, "get", "job", "--state", "failed", "--type", "payment-worker", "--pi-key", "2251799813711000", "--element-instance-key", "2251799813711001", "--element-id", "charge-card", "--worker", "worker-a", "--retries", "0", "--kind", "bpmn_element", "--listener-event-type", "unspecified", "--limit", "1")

	require.Equal(t, "2251799813711967 tenant-a BPMN_ELEMENT charge-card FAILED tp:payment-worker worker:worker-a pi:2251799813711000 ei:2251799813711001 r:0\nfound: 1\n", output)
	require.Len(t, bodies, 1)
	filter := requireJSONObject(t, bodies[0]["filter"])
	require.Equal(t, "FAILED", filter["state"])
	require.Equal(t, "payment-worker", filter["type"])
	require.Equal(t, "2251799813711000", filter["processInstanceKey"])
	require.Equal(t, "2251799813711001", filter["elementInstanceKey"])
	require.Equal(t, "charge-card", filter["elementId"])
	require.Equal(t, "worker-a", filter["worker"])
	require.Equal(t, float64(0), filter["retries"])
	require.Equal(t, "BPMN_ELEMENT", filter["kind"])
	require.Equal(t, "UNSPECIFIED", filter["listenerEventType"])
	page := requireJSONObject(t, bodies[0]["page"])
	require.Equal(t, float64(1), page["limit"])
}

func TestGetJobCommand_SearchModeBatchSizeShorthandPagesUntilComplete(t *testing.T) {
	var bodies []map[string]any
	srv := newJobSearchServerResponses(t, &bodies,
		`{"items":[{"jobKey":"2251799813711967","state":"FAILED","retries":0}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
		`{"items":[{"jobKey":"2251799813711968","state":"FAILED","retries":1}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForJobTest(t, "--config", cfgPath, "--auto-confirm", "get", "job", "-n", "2")

	require.Contains(t, output, "2251799813711967")
	require.Contains(t, output, "2251799813711968")
	require.Contains(t, output, "found: 2")
	require.Len(t, bodies, 2)
	page := requireJSONObject(t, bodies[0]["page"])
	require.Equal(t, float64(2), page["limit"])
	require.Equal(t, float64(0), page["from"])
	page = requireJSONObject(t, bodies[1]["page"])
	require.Equal(t, float64(2), page["limit"])
	require.Equal(t, float64(1), page["from"])
}

func TestGetJobCommand_SearchModeLimitShorthandCapsPagedSearch(t *testing.T) {
	var bodies []map[string]any
	srv := newJobSearchServerResponses(t, &bodies,
		`{"items":[{"jobKey":"2251799813711967","state":"FAILED","retries":0}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
		`{"items":[{"jobKey":"2251799813711968","state":"FAILED","retries":1}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForJobTest(t, "--config", cfgPath, "get", "job", "-n", "1", "-l", "1")

	require.Contains(t, output, "2251799813711967")
	require.NotContains(t, output, "2251799813711968")
	require.Contains(t, output, "found: 1")
	require.Len(t, bodies, 1)
	page := requireJSONObject(t, bodies[0]["page"])
	require.Equal(t, float64(1), page["limit"])
	require.Equal(t, float64(0), page["from"])
}

func TestGetJobCommand_SearchModeNormalizesEnumFilters(t *testing.T) {
	req := jobSearchRequestForTest(t, " failed ", " bpmn_element ", " completing ")

	require.Equal(t, "FAILED", req.State)
	require.Equal(t, "BPMN_ELEMENT", req.Kind)
	require.Equal(t, "COMPLETING", req.ListenerEventType)
	require.Equal(t, int32(1000), req.BatchSize)
	require.Zero(t, req.Limit)
}

// TestGetJobCommand_SearchModeJSONOutput keeps searched job output as one
// stable result envelope for machine consumers.
func TestGetJobCommand_SearchModeJSONOutput(t *testing.T) {
	var bodies []map[string]any
	srv := newJobSearchServer(t, &bodies, `{"items":[{"jobKey":"2251799813711967","state":"FAILED","retries":0,"tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "--json", "get", "job", "--state", "FAILED", "--limit", "1")

	require.Len(t, bodies, 1)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	payload := requireJSONObject(t, envelope["payload"])
	items := payload["items"].([]any)
	require.Len(t, items, 1)
	first := requireJSONObject(t, items[0])
	require.Equal(t, "2251799813711967", first["key"])
	require.Equal(t, "FAILED", first["state"])
	require.Equal(t, float64(0), first["retries"])
	require.Equal(t, float64(1), payload["limit"])
}

func jobSearchRequestForTest(t *testing.T, state string, kind string, listenerEventType string) job.SearchRequest {
	t.Helper()
	root := Root()
	resetCommandTreeFlags(root)
	resetGetJobFlagState()
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
		resetGetJobFlagState()
	})

	flagGetJobState = state
	flagGetJobKind = kind
	flagGetJobListenerEvent = listenerEventType
	require.NoError(t, getJobCmd.Flags().Set("state", state))
	require.NoError(t, getJobCmd.Flags().Set("kind", kind))
	require.NoError(t, getJobCmd.Flags().Set("listener-event-type", listenerEventType))
	require.NoError(t, validateGetJobFlags(getJobCmd))
	return newGetJobSearchRequest(getJobCmd)
}

// TestGetJobCommand_SearchModeJSONOutputIncludesEmptyItems keeps empty search
// results explicit for machine consumers.
func TestGetJobCommand_SearchModeJSONOutputIncludesEmptyItems(t *testing.T) {
	var bodies []map[string]any
	srv := newJobSearchServer(t, &bodies, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "--json", "get", "job", "--state", "FAILED", "--limit", "1")

	require.Len(t, bodies, 1)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	payload := requireJSONObject(t, envelope["payload"])
	items, ok := payload["items"].([]any)
	require.True(t, ok, "items should be present as an empty array")
	require.Empty(t, items)
	require.Equal(t, float64(1), payload["limit"])
}

// TestGetJobCommand_SearchModeKeysOnlyOutput emits only job keys so discovery
// can feed later pipeline commands without row text.
func TestGetJobCommand_SearchModeKeysOnlyOutput(t *testing.T) {
	var bodies []map[string]any
	srv := newJobSearchServer(t, &bodies, `{"items":[{"jobKey":"2251799813711967","state":"FAILED","retries":0},{"jobKey":"2251799813711968","state":"FAILED","retries":1}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForJobTest(t, "--config", cfgPath, "--keys-only", "get", "job", "--state", "FAILED", "--limit", "2")

	require.Equal(t, "2251799813711967\n2251799813711968\n", output)
	require.Len(t, bodies, 1)
}

func TestGetJobCommand_SearchUnsupportedV87FailsBeforeLookup(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, err := testx.RunCmdSubprocess(t, "TestGetJobCommand_SearchUnsupportedV87FailsBeforeLookupHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "search jobs")
	require.Contains(t, string(output), "Camunda 8.8")
	require.NotContains(t, string(output), "found:")
}

func TestGetJobCommand_SearchUnsupportedV87FailsBeforeLookupHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "job", "--state", "FAILED", "--limit", "1"}

	Execute()
}

// TestGetJobCommand_SearchValidationRejectsInvalidValues proves search filters
// fail locally for enum, key, count, and limit violations.
func TestGetJobCommand_SearchValidationRejectsInvalidValues(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func()
		want  string
	}{
		{name: "state", setup: func() { flagGetJobState = "OPEN" }, want: `invalid value for --state: "OPEN"`},
		{name: "kind", setup: func() { flagGetJobKind = "SERVICE_TASK" }, want: `invalid value for --kind: "SERVICE_TASK"`},
		{name: "listener event", setup: func() { flagGetJobListenerEvent = "DONE" }, want: `invalid value for --listener-event-type: "DONE"`},
		{name: "process key", setup: func() { flagGetJobProcessKey = "not-a-key" }, want: `--pi-key value "not-a-key" is not a valid key`},
		{name: "element key", setup: func() { flagGetJobElementKey = "not-a-key" }, want: `--element-instance-key value "not-a-key" is not a valid key`},
		{name: "retries", setup: func() {
			flagGetJobRetries = -1
			require.NoError(t, getJobCmd.Flags().Set("retries", "-1"))
		}, want: "--retries must be non-negative"},
		{name: "limit", setup: func() {
			flagGetJobLimit = 0
			require.NoError(t, getJobCmd.Flags().Set("limit", "0"))
		}, want: "--limit must be positive integer"},
		{name: "batch size", setup: func() {
			flagGetJobBatchSize = 0
			require.NoError(t, getJobCmd.Flags().Set("batch-size", "0"))
		}, want: "invalid value for --batch-size"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := Root()
			resetCommandTreeFlags(root)
			resetGetJobFlagState()
			t.Cleanup(func() {
				resetCommandTreeFlags(root)
				resetGetJobFlagState()
			})
			tc.setup()

			err := validateGetJobFlags(getJobCmd)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

// TestGetJobCommand_DoesNotRegisterFlowNodeAliases protects the job-specific
// element terminology required by the CLI contract.
func TestGetJobCommand_DoesNotRegisterFlowNodeAliases(t *testing.T) {
	for _, name := range []string{"flow-node", "flow-node-id", "fni", "fni-key"} {
		require.Nil(t, getJobCmd.Flags().Lookup(name), "job command must not expose --%s", name)
	}
}

func TestGetJobCommand_NotFoundExitsWithNotFound(t *testing.T) {
	for _, tc := range []struct {
		name string
		json bool
	}{
		{name: "human"},
		{name: "json", json: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests []string
			srv := newJobLookupServer(t, &requests, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
			t.Cleanup(srv.Close)
			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			env := map[string]string{"C8VOLT_TEST_CONFIG": cfgPath}
			if tc.json {
				env["C8VOLT_TEST_JSON"] = "1"
			}
			output, err := testx.RunCmdSubprocess(t, "TestGetJobCommand_NotFoundExitsWithNotFoundHelper", env)
			require.Error(t, err)

			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.NotFound, exitErr.ExitCode())
			require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
			require.Contains(t, string(output), "resource not found")
			require.Contains(t, string(output), "job missing-job was not found or is not visible to the configured tenant")
			if tc.json {
				var envelope map[string]any
				require.NoError(t, json.Unmarshal(output, &envelope))
				require.Equal(t, "not_found", envelope["class"])
			}
		})
	}
}

func TestGetJobCommand_NotFoundExitsWithNotFoundHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "job", "--key", "missing-job"}
	if os.Getenv("C8VOLT_TEST_JSON") == "1" {
		os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--json", "get", "job", "--key", "missing-job"}
	}

	Execute()
}

func newJobLookupServer(t *testing.T, requests *[]string, response string) *httptest.Server {
	t.Helper()
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/jobs/search", r.URL.Path)
		*requests = append(*requests, r.Method+" "+r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		filter := requireJSONObject(t, body["filter"])
		require.NotEmpty(t, filter["jobKey"])
		for name, value := range filter {
			if name == "jobKey" {
				continue
			}
			require.Nil(t, value, "keyed lookup should not send %s filter", name)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
}

// newJobSearchServer captures raw search request JSON for command-level search
// assertions while returning a generated-client compatible response.
func newJobSearchServer(t *testing.T, bodies *[]map[string]any, response string) *httptest.Server {
	t.Helper()
	return newJobSearchServerResponses(t, bodies, response)
}

// newJobSearchServerResponses returns one response per request and keeps the
// final response sticky when a test issues more calls than expected.
func newJobSearchServerResponses(t *testing.T, bodies *[]map[string]any, responses ...string) *httptest.Server {
	t.Helper()
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/jobs/search", r.URL.Path)
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

func executeRootForJobTest(t *testing.T, args ...string) string {
	t.Helper()

	resetGetJobFlagState()
	resetUpdateJobFlagState()
	t.Cleanup(func() {
		resetGetJobFlagState()
		resetUpdateJobFlagState()
	})

	root := Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetGetJobFlagState()
	resetUpdateJobFlagState()

	_, err := root.ExecuteC()
	require.NoError(t, err)
	return buf.String()
}

func resetGetJobFlagState() {
	flagGetJobKey = ""
	flagGetJobState = ""
	flagGetJobType = ""
	flagGetJobProcessKey = ""
	flagGetJobElementKey = ""
	flagGetJobElementID = ""
	flagGetJobWorker = ""
	flagGetJobRetries = 0
	flagGetJobKind = ""
	flagGetJobListenerEvent = ""
	flagGetJobBatchSize = consts.MaxPISearchSize
	flagGetJobLimit = 0
	flagGetErrorMessageLimit = 0
}
