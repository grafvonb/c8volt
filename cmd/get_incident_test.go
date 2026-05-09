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
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestGetIncidentCommand_KeyedLookupDeduplicatesFlagAndStdinKeys(t *testing.T) {
	var requests []string
	srv := newIncidentLookupServer(t, &requests, map[string]string{
		"2251799813685249": incidentLookupResultJSON("2251799813685249", "2251799813711967", "No retries left"),
		"2251799813685250": incidentLookupResultJSON("2251799813685250", "2251799813711968", "Mapping failed"),
	})
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForIncidentTestWithStdin(t,
		"2251799813685249\n2251799813685250\n",
		"--config", cfgPath,
		"get", "incident",
		"--workers", "1",
		"--key", "2251799813685249",
		"--key", "2251799813685249",
		"-",
	)

	require.Equal(t, []string{
		"GET /v2/incidents/2251799813685249",
		"GET /v2/incidents/2251799813685250",
	}, requests)
	require.Contains(t, output, "key=2251799813685249")
	require.Contains(t, output, "message=No retries left")
	require.Contains(t, output, "key=2251799813685250")
	require.Contains(t, output, "message=Mapping failed")
	require.Contains(t, output, "found: 2")
}

func TestGetIncidentCommand_JSONOutputUsesIncidentListPayload(t *testing.T) {
	var requests []string
	srv := newIncidentLookupServer(t, &requests, map[string]string{
		"2251799813685249": incidentLookupResultJSON("2251799813685249", "2251799813711967", "No retries left"),
	})
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"--json",
		"get", "incident",
		"--key", "2251799813685249",
	)

	require.Equal(t, []string{"GET /v2/incidents/2251799813685249"}, requests)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "get incident", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, float64(1), payload["total"])
	items, ok := payload["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "2251799813685249", item["incidentKey"])
	require.Equal(t, "No retries left", item["errorMessage"])
	require.Equal(t, "2026-03-23T18:01:00Z", item["creationTime"])
}

func TestGetIncidentCommand_KeysOnlyOutputUsesIncidentKeys(t *testing.T) {
	var requests []string
	srv := newIncidentLookupServer(t, &requests, map[string]string{
		"2251799813685249": incidentLookupResultJSON("2251799813685249", "2251799813711967", "No retries left"),
	})
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"--keys-only",
		"get", "inc",
		"--key", "2251799813685249",
	)

	require.Equal(t, []string{"GET /v2/incidents/2251799813685249"}, requests)
	require.Equal(t, "2251799813685249\n", output)
}

func TestGetIncidentCommand_RejectsInvalidKeyBeforeLookup(t *testing.T) {
	output, err := executeRootExpectErrorForIncidentTest(t, "get", "incident", "--key", "bad-key")

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
	require.Contains(t, err.Error(), "incident key \"bad-key\" is not a valid key")
	require.Empty(t, output)
}

func TestGetIncidentCommand_RejectsJSONErrorMessageLimit(t *testing.T) {
	output, err := executeRootExpectErrorForIncidentTest(t, "--json", "get", "incident", "--key", "2251799813685249", "--error-message-limit", "8")

	require.Error(t, err)
	require.Contains(t, err.Error(), "--error-message-limit cannot be combined with --json")
	require.Empty(t, output)
}

func TestGetIncidentCommand_NotFoundExitsWithNotFound(t *testing.T) {
	var requests []string
	srv := newIncidentLookupServer(t, &requests, map[string]string{})
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestGetIncidentCommand_NotFoundExitsWithNotFoundHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.NotFound, exitErr.ExitCode())
	require.Equal(t, []string{"GET /v2/incidents/2251799813685249"}, requests)
	require.Contains(t, string(output), "resource not found")
	require.Contains(t, string(output), "get incidents")
}

func TestGetIncidentCommand_NotFoundExitsWithNotFoundHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "incident", "--key", "2251799813685249"}

	Execute()
}

func newIncidentLookupServer(t *testing.T, requests *[]string, responses map[string]string) *httptest.Server {
	t.Helper()
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.True(t, strings.HasPrefix(r.URL.Path, "/v2/incidents/"))
		*requests = append(*requests, r.Method+" "+r.URL.Path)
		key := strings.TrimPrefix(r.URL.Path, "/v2/incidents/")
		response, ok := responses[key]
		if !ok {
			http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
}

func incidentLookupResultJSON(incidentKey string, processInstanceKey string, message string) string {
	return `{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"2251799813685300","errorMessage":` + strconvQuote(message) + `,"errorType":"JOB_NO_RETRIES","incidentKey":"` + incidentKey + `","processDefinitionId":"demo","processDefinitionKey":"2251799813685200","processInstanceKey":"` + processInstanceKey + `","state":"ACTIVE","tenantId":"tenant-a"}`
}

func executeRootForIncidentTest(t *testing.T, args ...string) string {
	t.Helper()

	resetGetIncidentFlagState()
	t.Cleanup(resetGetIncidentFlagState)

	root := Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetGetIncidentFlagState()

	_, err := root.ExecuteC()
	require.NoError(t, err)
	return buf.String()
}

func executeRootForIncidentTestWithStdin(t *testing.T, stdin string, args ...string) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	_, err = writer.WriteString(stdin)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	prevStdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = prevStdin
		_ = reader.Close()
	})

	return executeRootForIncidentTest(t, args...)
}

func executeRootExpectErrorForIncidentTest(t *testing.T, args ...string) (string, error) {
	t.Helper()

	resetGetIncidentFlagState()
	t.Cleanup(resetGetIncidentFlagState)

	root := Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetGetIncidentFlagState()

	_, err := root.ExecuteC()
	return buf.String(), err
}

func strconvQuote(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
