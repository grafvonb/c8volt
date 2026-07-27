// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestGetProcessDefinitionSelectionFlagsRemainSearchFilters(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	flagGetPDKey = "2251799813685255"
	flagGetPDBpmnProcessId = "invoice"
	flagGetPDProcessVersion = 3
	flagGetPDProcessVersionTag = "stable"
	flagGetPDLatest = true

	filter := populatePDSearchFilterOpts()

	require.Equal(t, process.ProcessDefinitionFilter{
		Key:               "2251799813685255",
		BpmnProcessId:     "invoice",
		ProcessVersion:    3,
		ProcessVersionTag: "stable",
	}, filter)
}

func TestGetProcessDefinitionLatestSearchPreservesSelectionRequest(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	output, err := testx.RunCmdSubprocess(t, "TestGetProcessDefinitionLatestSearchPreservesSelectionRequestHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.NoError(t, err, string(output))
	body := decodeSingleRequestJSON(t, requests)
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "invoice", filter["processDefinitionId"])
	require.Equal(t, float64(3), filter["version"])
	require.Equal(t, "stable", filter["versionTag"])
	require.Equal(t, true, filter["isLatestVersion"])
}

func TestGetProcessDefinitionBpmnSelectorMissingFailsWithExplicitDiagnostic(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))
		writeEmptyProcessDefinitionSearchResponse(w)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	output, err := testx.RunCmdSubprocess(t, "TestGetProcessDefinitionBpmnSelectorMissingFailsWithExplicitDiagnosticHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Len(t, requests, 1)
	body := decodeSingleRequestJSON(t, requests)
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "missing-process", filter["processDefinitionId"])
	require.Contains(t, string(output), "no visible process definition matches the provided selector")
	require.Contains(t, string(output), "[missing-process]")
}

func TestGetProcessDefinitionBpmnSelectorVisiblePreservesListing(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"order-process","name":"Order Process","version":3,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	output := executeRootForTest(t,
		"--config", cfgPath,
		"get", "process-definition",
		"--bpmn-process-id", "order-process",
		"--pd-version", "3",
		"--pd-version-tag", "stable",
	)

	require.Len(t, requests, 1)
	body := decodeSingleRequestJSON(t, requests)
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Equal(t, float64(3), filter["version"])
	require.Equal(t, "stable", filter["versionTag"])
	require.Contains(t, output, "2251799813685255")
	require.Contains(t, output, "tenant order-process v3/stable")
}

// TestGetProcessDefinitionSearchVerboseProgress defines the process-definition progress contract for broad listing.
func TestGetProcessDefinitionSearchVerboseProgress(t *testing.T) {
	var requests []map[string]any
	srv := newProcessDefinitionSearchServerResponses(t, &requests,
		`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"},{"processDefinitionKey":"2251799813685256","processDefinitionId":"payment","name":"payment","version":2,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":3,"hasMoreTotalItems":true,"endCursor":"pd-page-2"}}`,
		`{"items":[{"processDefinitionKey":"2251799813685257","processDefinitionId":"shipping","name":"shipping","version":1,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t,
		"--config", cfgPath,
		"--verbose",
		"--auto-confirm",
		"get", "process-definition",
		"--batch-size", "2",
	)

	require.Len(t, requests, 2)
	firstPage := requireJSONObject(t, requests[0]["page"])
	require.Equal(t, float64(2), firstPage["limit"])
	require.Equal(t, float64(0), firstPage["from"])
	secondPage := requireJSONObject(t, requests[1]["page"])
	require.Equal(t, "pd-page-2", secondPage["after"])
	require.Contains(t, stderr, "process-definition search scope: matched at least 3 process definitions; page size: 2; discovery pages: at least 2")
	require.Contains(t, stderr, "discovering process definitions, page 1/~2, 2 seen")
	require.Contains(t, stderr, "discovering process definitions, page 2/2, 3 seen")
	require.Contains(t, stdout, "2251799813685255")
	require.Contains(t, stdout, "2251799813685256")
	require.Contains(t, stdout, "2251799813685257")
	require.Contains(t, stdout, "found: 3")
}

// TestGetProcessDefinitionSearchMachineOutputStaysProgressFree protects process-definition JSON/key streams during progress rollout.
func TestGetProcessDefinitionSearchMachineOutputStaysProgressFree(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		var requests []map[string]any
		srv := newProcessDefinitionSearchServerResponses(t, &requests,
			`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

		stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t,
			"--config", cfgPath,
			"--json",
			"get", "process-definition",
		)

		require.Len(t, requests, 1)
		require.Empty(t, stderr)
		require.NotContains(t, stdout, "scope:")
		require.NotContains(t, stdout, "page size:")
		require.NotContains(t, stdout, "discovering process definitions")
		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), stdout)
		payload := requireJSONObject(t, envelope["payload"])
		items := payload["items"].([]any)
		require.Len(t, items, 1)
	})

	t.Run("keys only", func(t *testing.T) {
		var requests []map[string]any
		srv := newProcessDefinitionSearchServerResponses(t, &requests,
			`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"},{"processDefinitionKey":"2251799813685256","processDefinitionId":"payment","name":"payment","version":2,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

		stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t,
			"--config", cfgPath,
			"--keys-only",
			"get", "process-definition",
		)

		require.Len(t, requests, 1)
		require.Empty(t, stderr)
		require.Equal(t, "2251799813685255\n2251799813685256\n", stdout)
		require.NotContains(t, stdout, "scope:")
		require.NotContains(t, stdout, "page size:")
		require.NotContains(t, stdout, "discovering process definitions")
	})
}

func TestGetProcessDefinitionXMLOutputRemainsKeyOnlyDisplayMode(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*process.ProcessDefinitionFilter)
		assert func(*testing.T, error)
	}{
		{
			name: "key only accepted",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:  "missing key",
			setup: func(*process.ProcessDefinitionFilter) {},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "xml output requires --key")
			},
		},
		{
			name: "bpmn process id",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				filter.BpmnProcessId = "invoice"
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--bpmn-process-id"),
		},
		{
			name: "process version",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagGetPDProcessVersion = 3
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--pd-version"),
		},
		{
			name: "process version tag",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				filter.ProcessVersionTag = "stable"
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--pd-version-tag"),
		},
		{
			name: "latest",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagGetPDLatest = true
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--latest"),
		},
		{
			name: "stat",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagGetPDWithStat = true
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--stat"),
		},
		{
			name: "json",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagViewAsJson = true
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--json"),
		},
		{
			name: "keys only",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagViewKeysOnly = true
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--keys-only"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGetProcessDefinitionCommandGlobals()
			t.Cleanup(resetGetProcessDefinitionCommandGlobals)

			var filter process.ProcessDefinitionFilter
			tt.setup(&filter)
			tt.assert(t, validateProcessDefinitionXMLFlags(filter))
		})
	}
}

func requireXMLDisplayModeIncompatibleFlag(flag string) func(*testing.T, error) {
	return func(t *testing.T, err error) {
		t.Helper()
		require.Error(t, err)
		require.Contains(t, err.Error(), "xml output only supports --key")
		require.Contains(t, err.Error(), flag)
	}
}

func resetGetProcessDefinitionCommandGlobals() {
	flagGetPDKey = ""
	flagGetPDBpmnProcessId = ""
	flagGetPDProcessVersion = 0
	flagGetPDProcessVersionTag = ""
	flagGetPDLatest = false
	flagGetPDWithStat = false
	flagGetPDAsXML = false
	flagViewAsJson = false
	flagViewKeysOnly = false
}

func TestGetProcessDefinitionLatestSearchPreservesSelectionRequestHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetGetProcessDefinitionCommandGlobals()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"--json",
		"get", "process-definition",
		"--bpmn-process-id", "invoice",
		"--pd-version", "3",
		"--pd-version-tag", "stable",
		"--latest",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestGetProcessDefinitionBpmnSelectorMissingFailsWithExplicitDiagnosticHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetGetProcessDefinitionCommandGlobals()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"get", "process-definition",
		"--bpmn-process-id", "missing-process",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func executeRootForProcessDefinitionTestWithSeparateOutputs(t *testing.T, args ...string) (string, string) {
	t.Helper()

	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	root := Root()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetGetProcessDefinitionCommandGlobals()

	_, err := root.ExecuteC()
	require.NoError(t, err)
	return stdout.String(), stderr.String()
}

func newProcessDefinitionSearchServerResponses(t *testing.T, requests *[]map[string]any, responses ...string) *httptest.Server {
	t.Helper()

	served := 0
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var request map[string]any
		require.NoError(t, json.Unmarshal(body, &request))
		*requests = append(*requests, request)
		require.Less(t, served, len(responses), "unexpected extra process-definition search request")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[served]))
		served++
	}))
}

func countProcessDefinitionSearchRequests(items []string) int {
	count := 0
	for _, item := range items {
		if strings.HasPrefix(item, "POST /v2/process-definitions/search ") {
			count++
		}
	}
	return count
}
