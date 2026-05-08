// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetJobCommand_HumanOutput(t *testing.T) {
	var requests []string
	srv := newJobLookupServer(t, &requests, `{"items":[{"jobKey":"2251799813711967","state":"FAILED","retries":2,"deadline":"2026-05-08T10:15:00Z","processInstanceKey":"2251799813711000","elementInstanceKey":"2251799813711001","errorCode":"PAYMENT_ERROR","errorMessage":"worker failed","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "get", "job", "--key", "2251799813711967")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Contains(t, output, "job 2251799813711967")
	require.Contains(t, output, "state=FAILED")
	require.Contains(t, output, "retries=2")
	require.Contains(t, output, "deadline=2026-05-08T10:15:00Z")
	require.Contains(t, output, "processInstanceKey=2251799813711000")
	require.Contains(t, output, "elementInstanceKey=2251799813711001")
	require.Contains(t, output, "errorCode=PAYMENT_ERROR")
	require.Contains(t, output, "errorMessage=worker failed")
	require.Contains(t, output, "tenantId=tenant-a")
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
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, true, payload["found"])
	require.Equal(t, "2251799813711967", payload["key"])
	job := requireJSONObject(t, payload["job"])
	require.Equal(t, "2251799813711967", job["key"])
	require.Equal(t, "FAILED", job["state"])
	require.Equal(t, float64(2), job["retries"])
	require.Equal(t, "2251799813711000", job["processInstanceKey"])
	require.Equal(t, "2251799813711001", job["elementInstanceKey"])
	require.Equal(t, "tenant-a", job["tenantId"])
}

func TestGetJobCommand_NotFoundHumanAndJSON(t *testing.T) {
	t.Run("human", func(t *testing.T) {
		var requests []string
		srv := newJobLookupServer(t, &requests, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForJobTest(t, "--config", cfgPath, "get", "job", "--key", "missing-job")

		require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
		require.Equal(t, "job missing-job: not found\n", output)
	})

	t.Run("json", func(t *testing.T) {
		var requests []string
		srv := newJobLookupServer(t, &requests, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForJobTest(t, "--config", cfgPath, "--json", "get", "job", "--key", "missing-job")

		require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &envelope))
		payload := requireJSONObject(t, envelope["payload"])
		require.Equal(t, false, payload["found"])
		require.Equal(t, "missing-job", payload["key"])
		require.NotContains(t, payload, "job")
	})
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
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
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
}
