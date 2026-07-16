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
	require.Contains(t, string(output), "pi:2251799813688001")
	require.Contains(t, string(output), "element:ship-order")
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

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &payload))
	require.Equal(t, "2251799813689002", payload["elementInstanceKey"])
	require.Equal(t, "ship-order", payload["elementId"])
	require.Equal(t, "SERVICE_TASK", payload["type"])
	require.Equal(t, []string{"GET /v2/element-instances/2251799813689002"}, requests)
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
	flagGetElementBatchSize = 100
	flagGetElementLimit = 0
	flagGetElementTotal = false
	testx.ResetCommandTreeFlags(getElementCmd)
}
