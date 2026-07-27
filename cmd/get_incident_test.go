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
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const testGetIncidentArgsEnv = "C8VOLT_TEST_GET_INCIDENT_ARGS"

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
	require.Contains(t, output, "2251799813685249")
	require.Contains(t, output, "m:No retries left")
	require.Contains(t, output, "2251799813685250")
	require.Contains(t, output, "m:Mapping failed")
	require.Contains(t, output, "found: 2")
}

func TestGetIncidentCommand_JSONOutputUsesIncidentListPayload(t *testing.T) {
	var requests []string
	longMessage := "No retries left with full diagnostic context that must not be truncated in JSON"
	srv := newIncidentLookupServer(t, &requests, map[string]string{
		"2251799813685249": incidentLookupResultJSON("2251799813685249", "2251799813711967", longMessage),
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
	require.Equal(t, longMessage, item["errorMessage"])
	require.Equal(t, "2026-03-23T18:01:00Z", item["creationTime"])
	require.Equal(t, "task-a", item["elementId"])
	require.Equal(t, "2251799813685300", item["elementInstanceKey"])
	require.NotContains(t, output, "flowNode")
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

func TestGetIncidentCommand_PIKeysOnlyKeyedLookupUsesProcessInstanceKeys(t *testing.T) {
	var requests []string
	srv := newIncidentLookupServer(t, &requests, map[string]string{
		"2251799813685249": incidentLookupResultJSON("2251799813685249", "2251799813711967", "No retries left"),
		"2251799813685250": incidentLookupResultJSON("2251799813685250", "", "Missing process instance key"),
	})
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--workers", "1",
		"--key", "2251799813685249",
		"--key", "2251799813685250",
		"--pi-keys-only",
	)

	require.Equal(t, []string{
		"GET /v2/incidents/2251799813685249",
		"GET /v2/incidents/2251799813685250",
	}, requests)
	require.Equal(t, "2251799813711967\n", output)
}

func TestGetIncidentCommand_SearchKeysOnlyOutputUsesIncidentKeys(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"errorMessage":"first","incidentKey":"2251799813685253","processInstanceKey":"2251799813711972","state":"ACTIVE","tenantId":"tenant-a"},{"errorMessage":"second","incidentKey":"2251799813685254","processInstanceKey":"2251799813711973","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"--keys-only",
		"get", "incident",
		"--state", "active",
	)

	require.Len(t, requests, 1)
	require.Equal(t, "2251799813685253\n2251799813685254\n", output)
}

func TestGetIncidentCommand_SearchPIKeysOnlyOutputUsesProcessInstanceKeys(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"errorMessage":"first","incidentKey":"2251799813685253","processInstanceKey":"2251799813711972","state":"ACTIVE","tenantId":"tenant-a"},{"errorMessage":"second","incidentKey":"2251799813685254","processInstanceKey":"2251799813711972","state":"ACTIVE","tenantId":"tenant-a"},{"errorMessage":"missing","incidentKey":"2251799813685255","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--state", "active",
		"--pi-keys-only",
	)

	require.Len(t, requests, 1)
	require.Equal(t, "2251799813711972\n2251799813711972\n", output)
}

func TestGetIncidentBpmnSelectorMissingFailsBeforeSearch(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			writeEmptyProcessDefinitionSearchResponse(w)
		case "/v2/incidents/search":
			t.Fatal("unexpected incident search before selector validation")
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output, code := executeGetIncidentFailureHelper(t, cfgPath,
		"get", "incident",
		"--state", "active",
		"--bpmn-process-id", "missing-process",
	)

	require.Equal(t, exitcode.Error, code)
	require.Equal(t, []string{"POST /v2/process-definitions/search"}, requests)
	require.Contains(t, output, "no visible process definition matches the provided selector")
	require.Contains(t, output, "[missing-process]")
	require.NotContains(t, output, "found: 0")
}

func TestGetIncidentBpmnSelectorVisiblePreservesEmptyOutput(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			writeVisibleProcessDefinitionSearchResponse(w)
		case "/v2/incidents/search":
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--state", "active",
		"--bpmn-process-id", "order-process",
	)

	require.Equal(t, []string{"POST /v2/process-definitions/search", "POST /v2/incidents/search"}, requests)
	require.Equal(t, "found: 0\n", output)
}

func TestGetIncidentBpmnSelectorMissingNoPromptModes(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "total", args: []string{"get", "incident", "--bpmn-process-id", "missing-process", "--total"}},
		{name: "keys only", args: []string{"--keys-only", "get", "incident", "--bpmn-process-id", "missing-process"}},
		{name: "pi keys only", args: []string{"get", "incident", "--bpmn-process-id", "missing-process", "--pi-keys-only"}},
		{name: "json", args: []string{"--json", "get", "incident", "--bpmn-process-id", "missing-process"}},
		{name: "automation", args: []string{"--automation", "get", "incident", "--bpmn-process-id", "missing-process"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []string
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.Method+" "+r.URL.Path)
				require.Equal(t, http.MethodPost, r.Method)
				switch r.URL.Path {
				case "/v2/process-definitions/search":
					writeEmptyProcessDefinitionSearchResponse(w)
				case "/v2/incidents/search":
					t.Fatal("unexpected incident search before selector validation")
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)
			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

			output, code := executeGetIncidentFailureHelper(t, cfgPath, tt.args...)

			require.Equal(t, exitcode.Error, code)
			require.Equal(t, []string{"POST /v2/process-definitions/search"}, requests)
			require.Contains(t, output, "no visible process definition matches the provided selector")
			require.NotContains(t, output, "List visible process definitions?")
		})
	}
}

func TestGetIncidentBpmnSelectorValidationUsesTenantContext(t *testing.T) {
	var processDefinitionSearchBody string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			processDefinitionSearchBody = string(body)
			writeEmptyProcessDefinitionSearchResponse(w)
		case "/v2/incidents/search":
			t.Fatal("unexpected incident search before selector validation")
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output, code := executeGetIncidentFailureHelper(t, cfgPath,
		"--tenant", "tenant-a",
		"get", "incident",
		"--bpmn-process-id", "missing-process",
	)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, output, "no visible process definition matches the provided selector")
	var request map[string]any
	require.NoError(t, json.Unmarshal([]byte(processDefinitionSearchBody), &request))
	filter, ok := request["filter"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "missing-process", filter["processDefinitionId"])
	require.Equal(t, "tenant-a", filter["tenantId"])
	require.NotContains(t, filter, "version")
	require.NotContains(t, filter, "versionTag")
}

// TestGetIncidentCommand_RegressionSelectionAndDisplayFlagsRemainDistinct
// protects the original incident command surface while incident purge reuses
// only the selection subset.
func TestGetIncidentCommand_RegressionSelectionAndDisplayFlagsRemainDistinct(t *testing.T) {
	help := executeRootForIncidentTest(t, "get", "incident", "--help")
	assertHelpOutputContainsAll(t, help,
		"--key strings",
		"--state string",
		"--error-type string",
		"--pi-key string",
		"--pi-keys-only",
		"--total",
		"--error-message-limit int",
		"--with-no-error-message",
	)

	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"2251799813685253","processInstanceKey":"2251799813711972","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--state", "active",
		"--error-type", "job_no_retries",
		"--limit", "1",
		"--pi-keys-only",
	)

	require.Len(t, requests, 1)
	require.Contains(t, requests[0], `"state":"ACTIVE"`)
	require.Contains(t, requests[0], `"errorType":"JOB_NO_RETRIES"`)
	require.Equal(t, "2251799813711972\n", output)
}

func TestGetIncidentCommand_SearchPIKeysOnlyIncrementalPagesOmitFound(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"errorMessage":"first","incidentKey":"2251799813685253","processInstanceKey":"2251799813711972","state":"ACTIVE","tenantId":"tenant-a"},{"errorMessage":"second","incidentKey":"2251799813685254","processInstanceKey":"2251799813711973","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":4,"hasMoreTotalItems":true}}`,
		`{"items":[{"errorMessage":"third","incidentKey":"2251799813685255","processInstanceKey":"2251799813711974","state":"ACTIVE","tenantId":"tenant-a"},{"errorMessage":"fourth","incidentKey":"2251799813685256","processInstanceKey":"2251799813711975","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":4,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
	promptCalls := 0
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		promptCalls++
		require.False(t, autoConfirm)
		require.Contains(t, prompt, "More matching incidents remain")
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--batch-size", "2",
		"--pi-keys-only",
	)

	require.Len(t, requests, 2)
	require.Equal(t, 1, promptCalls)
	require.Equal(t, "2251799813711972\n2251799813711973\n2251799813711974\n2251799813711975\n", output)
	require.NotContains(t, output, "found:")
}

// TestGetIncidentCommand_SearchSkipsPromptForEmptyFilteredPages verifies locally filtered empty pages do not make interactive users confirm before any rows are shown.
func TestGetIncidentCommand_SearchSkipsPromptForEmptyFilteredPages(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
		`{"items":[{"errorMessage":"first","incidentKey":"2251799813685253","processInstanceKey":"2251799813711972","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
		`{"items":[{"errorMessage":"second","incidentKey":"2251799813685254","processInstanceKey":"2251799813711973","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
	var prompts []string
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		require.False(t, autoConfirm)
		prompts = append(prompts, prompt)
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--batch-size", "1",
	)

	require.Len(t, requests, 3)
	require.Len(t, prompts, 1)
	require.Contains(t, prompts[0], "Fetched 1 incident(s) on this page")
	require.Contains(t, output, "2251799813685253")
	require.Contains(t, output, "2251799813685254")
	require.Contains(t, output, "found: 2")
}

func TestGetIncidentCommand_TotalUsesExactReportedBackendTotal(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"errorMessage":"first","incidentKey":"2251799813685253","processInstanceKey":"2251799813711972","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":7,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--total",
	)

	require.Len(t, requests, 1)
	require.Equal(t, "7\n", output)
}

func TestGetIncidentCommand_TotalCountsLocalFilteredMatchesAcrossPages(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"errorMessage":"No retries left","incidentKey":"2251799813685257","processInstanceKey":"2251799813711976","state":"ACTIVE","tenantId":"tenant-a"},{"errorMessage":"Mapping failed","incidentKey":"2251799813685258","processInstanceKey":"2251799813711977","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":4,"hasMoreTotalItems":true}}`,
		`{"items":[{"errorMessage":"first intentional failure","incidentKey":"2251799813685259","processInstanceKey":"2251799813711978","state":"ACTIVE","tenantId":"tenant-a"},{"errorMessage":"second INTENTIONAL issue","incidentKey":"2251799813685260","processInstanceKey":"2251799813711979","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":4,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--batch-size", "2",
		"--total",
		"--error-message", "INTENTIONAL",
	)

	require.Len(t, requests, 2)
	require.NotContains(t, output, "found:")
	require.Equal(t, "2\n", output)
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

func TestGetIncidentCommand_RejectsKeysOnlyErrorMessageLimit(t *testing.T) {
	output, err := executeRootExpectErrorForIncidentTest(t, "--keys-only", "get", "incident", "--key", "2251799813685249", "--error-message-limit", "8")

	require.Error(t, err)
	require.Contains(t, err.Error(), "--error-message-limit cannot be combined with --keys-only")
	require.Empty(t, output)
}

func TestGetIncidentCommand_RejectsPIKeysOnlyMachineOutputModesBeforeRequest(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{
			name: "json",
			args: []string{"--json", "get", "incident", "--key", "2251799813685249", "--pi-keys-only"},
			want: "--pi-keys-only cannot be combined with --json",
		},
		{
			name: "keys only",
			args: []string{"--keys-only", "get", "incident", "--key", "2251799813685249", "--pi-keys-only"},
			want: "--pi-keys-only cannot be combined with --keys-only",
		},
		{
			name: "total",
			args: []string{"get", "incident", "--key", "2251799813685249", "--pi-keys-only", "--total"},
			want: "--pi-keys-only cannot be combined with --total",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests []string
			srv := newIncidentUnexpectedRequestServer(t, &requests)
			t.Cleanup(srv.Close)
			args := append([]string{"--config", writeTestConfigForVersion(t, srv.URL, "8.9")}, tc.args...)

			output, err := executeRootExpectErrorForIncidentTest(t, args...)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
			require.Empty(t, output)
			require.Empty(t, requests)
		})
	}
}

func TestGetIncidentCommand_RejectsPIKeysOnlyMessageOutputModifiersBeforeRequest(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{
			name: "error message limit",
			args: []string{"get", "incident", "--key", "2251799813685249", "--pi-keys-only", "--error-message-limit", "8"},
			want: "--pi-keys-only cannot be combined with --error-message-limit",
		},
		{
			name: "with no error message",
			args: []string{"get", "incident", "--key", "2251799813685249", "--pi-keys-only", "--with-no-error-message"},
			want: "--pi-keys-only cannot be combined with --with-no-error-message",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests []string
			srv := newIncidentUnexpectedRequestServer(t, &requests)
			t.Cleanup(srv.Close)
			args := append([]string{"--config", writeTestConfigForVersion(t, srv.URL, "8.9")}, tc.args...)

			output, err := executeRootExpectErrorForIncidentTest(t, args...)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
			require.Empty(t, output)
			require.Empty(t, requests)
		})
	}
}

func TestGetIncidentCommand_WithNoErrorMessageOmitsHumanMessageTail(t *testing.T) {
	var requests []string
	srv := newIncidentLookupServer(t, &requests, map[string]string{
		"2251799813685249": incidentLookupResultJSON("2251799813685249", "2251799813711967", "No retries left"),
	})
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--key", "2251799813685249",
		"--with-no-error-message",
	)

	require.Equal(t, []string{"GET /v2/incidents/2251799813685249"}, requests)
	require.Contains(t, output, "2251799813685249")
	require.NotContains(t, output, "m:")
	require.NotContains(t, output, "No retries left")
	require.Contains(t, output, "found: 1")
}

func TestGetIncidentCommand_RejectsWithNoErrorMessageMachineOutputAndLimit(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "json", args: []string{"--json", "get", "incident", "--key", "2251799813685249", "--with-no-error-message"}, want: "--with-no-error-message cannot be combined with --json"},
		{name: "keys only", args: []string{"--keys-only", "get", "incident", "--key", "2251799813685249", "--with-no-error-message"}, want: "--with-no-error-message cannot be combined with --keys-only"},
		{name: "message limit", args: []string{"get", "incident", "--key", "2251799813685249", "--with-no-error-message", "--error-message-limit", "8"}, want: "--with-no-error-message cannot be combined with --error-message-limit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := executeRootExpectErrorForIncidentTest(t, tc.args...)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
			require.Empty(t, output)
		})
	}
}

func TestGetIncidentCommand_RejectsTotalWithMachineOutput(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "json", args: []string{"--json", "get", "incident", "--total"}, want: "--total cannot be combined with --json"},
		{name: "keys only", args: []string{"--keys-only", "get", "incident", "--total"}, want: "--total cannot be combined with --keys-only"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := executeRootExpectErrorForIncidentTest(t, tc.args...)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
			require.Empty(t, output)
		})
	}
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

func TestGetIncidentCommand_V87ReportsUnsupported(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "keyed lookup",
			helper: "TestGetIncidentCommand_V87KeyedLookupUnsupportedHelper",
			want:   "direct incident lookup is not tenant-safe in Camunda 8.7",
		},
		{
			name:   "search",
			helper: "TestGetIncidentCommand_V87SearchUnsupportedHelper",
			want:   "incident search is not tenant-safe in Camunda 8.7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, tt.helper, map[string]string{
				"C8VOLT_TEST_CONFIG": cfgPath,
			})

			require.Error(t, err)
			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.Error, exitErr.ExitCode())
			require.Contains(t, string(output), "unsupported capability")
			require.Contains(t, string(output), tt.want)
		})
	}
}

func TestGetIncidentCommand_V87KeyedLookupUnsupportedHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "incident", "--key", "2251799813685249"}

	Execute()
}

func TestGetIncidentCommand_V87SearchUnsupportedHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "incident"}

	Execute()
}

func TestGetIncidentCommand_SearchDefaultsToActiveState(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"2251799813685300","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"2251799813685249","processDefinitionId":"demo","processDefinitionKey":"2251799813685200","processInstanceKey":"2251799813711967","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
	)

	require.Len(t, requests, 1)
	require.Contains(t, requests[0], `"state":"ACTIVE"`)
	require.Contains(t, output, "2251799813685249")
	require.Contains(t, output, "ACTIVE")
	require.Contains(t, output, "found: 1")
}

func TestGetIncidentCommand_SearchStateAllOmitsStateFilter(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"errorMessage":"resolved earlier","errorType":"JOB_NO_RETRIES","incidentKey":"2251799813685250","processInstanceKey":"2251799813711968","state":"RESOLVED","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--state", "all",
	)

	require.Len(t, requests, 1)
	require.NotContains(t, requests[0], `"state"`)
	require.Contains(t, output, "RESOLVED")
	require.Contains(t, output, "found: 1")
}

func TestGetIncidentCommand_SearchNormalizesCaseInsensitiveState(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"errorMessage":"resolved earlier","errorType":"JOB_NO_RETRIES","incidentKey":"2251799813685250","processInstanceKey":"2251799813711968","state":"RESOLVED","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--state", "RESOLVED",
	)

	require.Len(t, requests, 1)
	require.Contains(t, requests[0], `"state":"RESOLVED"`)
	require.Contains(t, output, "RESOLVED")
	require.Contains(t, output, "found: 1")
}

func TestGetIncidentCommand_RejectsInvalidStateBeforeLookup(t *testing.T) {
	output, err := executeRootExpectErrorForIncidentTest(t, "get", "incident", "--state", "done")

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
	require.Contains(t, err.Error(), `invalid value for --state: "done", valid values are: active, pending, resolved, migrated, unknown, all`)
	require.Empty(t, output)
}

func TestGetIncidentCommand_SearchNormalizesCaseInsensitiveErrorType(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"errorMessage":"Mapping failed","errorType":"IO_MAPPING_ERROR","incidentKey":"2251799813685251","processInstanceKey":"2251799813711969","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--error-type", "io_mapping_error",
	)

	require.Len(t, requests, 1)
	require.Contains(t, requests[0], `"errorType":"IO_MAPPING_ERROR"`)
	require.Contains(t, output, "IO_MAPPING_ERROR")
	require.Contains(t, output, "found: 1")
}

func TestGetIncidentCommand_RejectsInvalidErrorType(t *testing.T) {
	output, err := executeRootExpectErrorForIncidentTest(t, "get", "incident", "--error-type", "bad_type")

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
	require.Contains(t, err.Error(), `invalid value for --error-type: "bad_type"`)
	require.NotContains(t, err.Error(), "JOB_NO_RETRIES")
	require.Empty(t, output)
}

func TestGetIncidentCommand_SearchCoreProcessAndElementFilters(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"2251799813685303","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"2251799813685252","processDefinitionId":"order-process","processDefinitionKey":"2251799813685201","processInstanceKey":"2251799813711970","rootProcessInstanceKey":"2251799813711971","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--pi-key", "2251799813711970",
		"--root-key", "2251799813711971",
		"--pd-key", "2251799813685201",
		"--bpmn-process-id", "order-process",
		"--element-id", "task-a",
		"--element-instance-key", "2251799813685303",
	)

	require.Len(t, requests, 1)
	require.Contains(t, requests[0], "2251799813711970")
	require.NotContains(t, requests[0], "2251799813711971")
	require.Contains(t, requests[0], "2251799813685201")
	require.Contains(t, requests[0], "order-process")
	require.Contains(t, requests[0], "task-a")
	require.Contains(t, requests[0], "2251799813685303")
	require.Contains(t, output, "2251799813685252")
	require.Contains(t, output, "e:task-a")
	require.Contains(t, output, "ei:2251799813685303")
	require.NotContains(t, output, "fn:")
	require.NotContains(t, output, "fni:")
	require.Contains(t, output, "pi:2251799813711970")
	require.Contains(t, output, "root:2251799813711971")
	require.Contains(t, output, "order-process")
	require.Contains(t, output, "found: 1")
}

func TestGetIncidentCommand_RejectsLegacyFlowNodeFilterFlags(t *testing.T) {
	tests := []string{"--flow-node-id", "--fni-key"}

	for _, flag := range tests {
		t.Run(flag, func(t *testing.T) {
			output, err := executeRootExpectErrorForIncidentTest(t, "get", "incident", flag, "legacy-value")

			require.Error(t, err)
			require.Contains(t, err.Error(), "unknown flag: "+flag)
			require.Empty(t, output)
		})
	}
}

func TestGetIncidentCommand_SearchCreationTimeWindow(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"creationTime":"2026-05-09T10:15:00Z","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"2251799813685253","processInstanceKey":"2251799813711972","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--creation-time-after", "2026-05-09T09:00:00Z",
		"--creation-time-before", "2026-05-09T11:00:00Z",
	)

	require.Len(t, requests, 1)
	require.Contains(t, requests[0], `"creationTime"`)
	require.Contains(t, requests[0], `"$gte":"2026-05-09T09:00:00Z"`)
	require.Contains(t, requests[0], `"$lte":"2026-05-09T11:00:00Z"`)
	require.Contains(t, output, "2251799813685253")
	require.Contains(t, output, "2026-05-09T10:15:00.000")
	require.Contains(t, output, "found: 1")
}

func TestGetIncidentCommand_SearchCreationTimeAcceptsDateOnlyBounds(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--creation-time-after", "2026-05-09",
		"--creation-time-before", "2026-05-10",
	)

	require.Len(t, requests, 1)
	require.Contains(t, requests[0], `"$gte":"2026-05-09T00:00:00Z"`)
	require.Contains(t, requests[0], `"$lte":"2026-05-10T23:59:59.999999999Z"`)
	require.Contains(t, output, "found: 0")
}

func TestGetIncidentCommand_SearchCreationTimeAcceptsDisplayedTimestamps(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--creation-time-after", "2026-05-25T20:09:11",
		"--creation-time-before", "2026-05-25T20:09:11.000",
	)

	require.Len(t, requests, 1)
	require.Contains(t, requests[0], `"$gte":"2026-05-25T20:09:11Z"`)
	require.Contains(t, requests[0], `"$lte":"2026-05-25T20:09:11Z"`)
	require.Contains(t, output, "found: 0")
}

func TestGetIncidentCommand_SearchCreationTimeRelativeDayBounds(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--creation-time-newer-days", "30",
		"--creation-time-older-days", "7",
	)

	require.Len(t, requests, 1)
	require.Contains(t, requests[0], `"$gte":"2026-04-26T00:00:00Z"`)
	require.Contains(t, requests[0], `"$lte":"2026-05-19T23:59:59.999999999Z"`)
	require.Contains(t, output, "found: 0")
}

func TestGetIncidentCommand_RejectsInvalidCreationTimeBeforeLookup(t *testing.T) {
	output, err := executeRootExpectErrorForIncidentTest(t,
		"get", "incident",
		"--creation-time-after", "last-friday",
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
	require.Contains(t, err.Error(), `invalid value for --creation-time-after: "last-friday", expected RFC3339 timestamp, c8volt timestamp YYYY-MM-DDTHH:MM:SS[.fraction], or YYYY-MM-DD`)
	require.Empty(t, output)
}

func TestGetIncidentCommand_RejectsInvalidCreationTimeCombinationsBeforeLookup(t *testing.T) {
	output, err := executeRootExpectErrorForIncidentTest(t,
		"get", "incident",
		"--creation-time-after", "2026-05-09T09:00:00Z",
		"--creation-time-newer-days", "7",
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
	require.Contains(t, err.Error(), "creation-time absolute and relative day filters cannot be combined")
	require.Empty(t, output)

	output, err = executeRootExpectErrorForIncidentTest(t,
		"get", "incident",
		"--creation-time-newer-days", "7",
		"--creation-time-older-days", "30",
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid input")
	require.Contains(t, err.Error(), "invalid range for --creation-time-newer-days and --creation-time-older-days")
	require.Empty(t, output)
}

func TestGetIncidentCommand_SearchAutoConfirmContinuesPagesAndHonorsLimit(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"errorMessage":"first","incidentKey":"2251799813685253","processInstanceKey":"2251799813711972","state":"ACTIVE","tenantId":"tenant-a"},{"errorMessage":"second","incidentKey":"2251799813685254","processInstanceKey":"2251799813711973","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":4,"hasMoreTotalItems":true}}`,
		`{"items":[{"errorMessage":"third","incidentKey":"2251799813685255","processInstanceKey":"2251799813711974","state":"ACTIVE","tenantId":"tenant-a"},{"errorMessage":"fourth","incidentKey":"2251799813685256","processInstanceKey":"2251799813711975","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":4,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--batch-size", "2",
		"--limit", "3",
		"--auto-confirm",
	)

	require.Len(t, requests, 2)
	require.Contains(t, requests[0], `"limit":2`)
	require.Contains(t, requests[1], `"from":2`)
	require.Contains(t, output, "2251799813685253")
	require.Contains(t, output, "2251799813685254")
	require.Contains(t, output, "2251799813685255")
	require.NotContains(t, output, "2251799813685256")
	require.Contains(t, output, "found: 3")
}

// TestGetIncidentCommand_SearchVerboseProgress verifies paged incident search reports durable progress away from stdout.
func TestGetIncidentCommand_SearchVerboseProgress(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"errorMessage":"first","incidentKey":"2251799813685253","processInstanceKey":"2251799813711972","state":"ACTIVE","tenantId":"tenant-a"},{"errorMessage":"second","incidentKey":"2251799813685254","processInstanceKey":"2251799813711973","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
		`{"items":[{"errorMessage":"third","incidentKey":"2251799813685255","processInstanceKey":"2251799813711974","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"--verbose",
		"--auto-confirm",
		"get", "incident",
		"--batch-size", "2",
	)

	require.Len(t, requests, 2)
	require.Contains(t, requests[0], `"limit":2`)
	require.Contains(t, requests[0], `"from":0`)
	require.Contains(t, requests[1], `"from":2`)
	require.Contains(t, output, "incident search scope: matched at least 3 incidents; page size: 2; discovery pages: at least 2")
	require.Contains(t, output, "discovering incidents, page 1/~2, 2 seen")
	require.Contains(t, output, "discovering incidents, page 2/2, 3 seen")
	require.Contains(t, output, "found: 3")
}

// TestGetIncidentCommand_SearchMachineOutputStaysProgressFree verifies shared progress rollout cannot leak into incident JSON or key streams.
func TestGetIncidentCommand_SearchMachineOutputStaysProgressFree(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		var requests []string
		srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"errorMessage":"first","incidentKey":"2251799813685253","processInstanceKey":"2251799813711972","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":2,"hasMoreTotalItems":true}}`,
			`{"items":[{"errorMessage":"second","incidentKey":"2251799813685254","processInstanceKey":"2251799813711973","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

		stdout, stderr := executeRootForIncidentTestWithSeparateOutputs(t,
			"--config", cfgPath,
			"--json",
			"get", "incident",
			"--batch-size", "1",
		)

		require.Len(t, requests, 2)
		require.Empty(t, stderr)
		require.NotContains(t, stdout, "scope:")
		require.NotContains(t, stdout, "page size:")
		require.NotContains(t, stdout, "discovering incidents")
		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
		payload := requireJSONObject(t, envelope["payload"])
		items := payload["items"].([]any)
		require.Len(t, items, 2)
	})

	t.Run("keys only", func(t *testing.T) {
		var requests []string
		srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"errorMessage":"first","incidentKey":"2251799813685253","processInstanceKey":"2251799813711972","state":"ACTIVE","tenantId":"tenant-a"},{"errorMessage":"second","incidentKey":"2251799813685254","processInstanceKey":"2251799813711973","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

		stdout, stderr := executeRootForIncidentTestWithSeparateOutputs(t,
			"--config", cfgPath,
			"--keys-only",
			"get", "incident",
			"--state", "active",
		)

		require.Len(t, requests, 1)
		require.Empty(t, stderr)
		require.Equal(t, "2251799813685253\n2251799813685254\n", stdout)
		require.NotContains(t, stdout, "scope:")
		require.NotContains(t, stdout, "page size:")
	})
}

func TestGetIncidentCommand_SearchErrorMessageMatchesCaseInsensitiveAcrossPages(t *testing.T) {
	var requests []string
	srv := newIncidentSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"errorMessage":"No retries left","incidentKey":"2251799813685257","processInstanceKey":"2251799813711976","state":"ACTIVE","tenantId":"tenant-a"},{"errorMessage":"Mapping failed","incidentKey":"2251799813685258","processInstanceKey":"2251799813711977","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":4,"hasMoreTotalItems":true}}`,
		`{"items":[{"errorMessage":"first intentional failure","incidentKey":"2251799813685259","processInstanceKey":"2251799813711978","state":"ACTIVE","tenantId":"tenant-a"},{"errorMessage":"second INTENTIONAL issue","incidentKey":"2251799813685260","processInstanceKey":"2251799813711979","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":4,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForIncidentTest(t,
		"--config", cfgPath,
		"get", "incident",
		"--batch-size", "2",
		"--limit", "2",
		"--auto-confirm",
		"--error-message", "INTENTIONAL",
	)

	require.Len(t, requests, 2)
	require.NotContains(t, requests[0], "errorMessage")
	require.Contains(t, requests[0], `"limit":2`)
	require.Contains(t, requests[1], `"from":2`)
	require.NotContains(t, output, "2251799813685257")
	require.NotContains(t, output, "2251799813685258")
	require.Contains(t, output, "2251799813685259")
	require.Contains(t, output, "2251799813685260")
	require.Contains(t, output, "found: 2")
}

func TestGetIncidentCommand_RejectsKeyedLookupWithSearchFilter(t *testing.T) {
	output, err := executeRootExpectErrorForIncidentTest(t,
		"get", "incident",
		"--key", "2251799813685249",
		"--state", "resolved",
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--key cannot be combined with search filters")
	require.Empty(t, output)
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

func newIncidentSearchCaptureServerWithResponses(t *testing.T, requests *[]string, responses ...string) *httptest.Server {
	t.Helper()

	served := 0
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		if r.URL.Path == "/v2/process-definitions/search" {
			writeVisibleProcessDefinitionSearchResponse(w)
			return
		}
		require.Equal(t, "/v2/incidents/search", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		*requests = append(*requests, string(body))
		require.Less(t, served, len(responses), "unexpected extra incident search request")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[served]))
		served++
	}))
}

func newIncidentUnexpectedRequestServer(t *testing.T, requests *[]string) *httptest.Server {
	t.Helper()
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requests = append(*requests, r.Method+" "+r.URL.Path)
		http.Error(w, `{"message":"unexpected request"}`, http.StatusInternalServerError)
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

// executeRootForIncidentTestWithSeparateOutputs lets machine-output tests prove progress stays off stdout.
func executeRootForIncidentTestWithSeparateOutputs(t *testing.T, args ...string) (string, string) {
	t.Helper()

	resetGetIncidentFlagState()
	t.Cleanup(resetGetIncidentFlagState)

	root := Root()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetGetIncidentFlagState()

	_, err := root.ExecuteC()
	require.NoError(t, err)
	return stdout.String(), stderr.String()
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

func executeGetIncidentFailureHelper(t *testing.T, cfgPath string, args ...string) (string, int) {
	t.Helper()

	argsJSON, err := json.Marshal(args)
	require.NoError(t, err)

	output, err := testx.RunCmdSubprocess(t, "TestGetIncidentCommand_HelperProcess", map[string]string{
		"C8VOLT_TEST_CONFIG":   cfgPath,
		testGetIncidentArgsEnv: string(argsJSON),
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	return string(output), exitErr.ExitCode()
}

func TestGetIncidentCommand_HelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv(testGetIncidentArgsEnv)), &args))

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = append([]string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG")}, args...)

	Execute()
}

func strconvQuote(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}
