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
	require.Equal(t, "2251799813711967 tenant-a FAILED pi:2251799813711000 ei:2251799813711001 r:2 d:2026-05-08T10:15:00+00:00 ec:PAYMENT_ERROR err:worker failed\n", output)
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
	require.Equal(t, "2251799814014237 tenant-a FAILED pi:2251799814014230 ei:2251799814014236 r:0 d:2026-04-23T01:07:49+00:00 err:"+longMessage+"\n", output)
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
	flagGetErrorMessageLimit = 0
}
