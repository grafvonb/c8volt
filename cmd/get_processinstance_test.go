// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

// TestGetProcessInstanceHelp_DocumentsPagingAndAutomationSurface verifies help text exposes paging and automation contracts.
func TestGetProcessInstanceHelp_DocumentsPagingAndAutomationSurface(t *testing.T) {
	output := executeRootForProcessInstanceTest(t, "get", "process-instance", "--help")

	require.Contains(t, output, "Get process instances by key or by search criteria.")
	require.Contains(t, output, "Search results support interactive paging, scriptable JSON aggregation, and count-only workflows.")
	require.Contains(t, output, "matching process instances by process definition")
	require.Contains(t, output, "Direct key lookup stays strict")
	require.Contains(t, output, "Run `c8volt get pi --help` for the complete flag reference.")
	require.Contains(t, output, "./c8volt get pi --bpmn-process-id <bpmn-process-id> --state active")
	require.Contains(t, output, "./c8volt get pi --key <process-instance-key>")
	require.Contains(t, output, "./c8volt get pi --state active --total")
	require.Contains(t, output, "./c8volt get pi --state active --json")
	require.Contains(t, output, "./c8volt get pi --state active --limit 25 --auto-confirm")
	require.Contains(t, output, "capped backend totals are counted by paging")
	require.Contains(t, output, "--auto-confirm")
	require.Contains(t, output, "--batch-size int32")
	require.Contains(t, output, "number of process instances to fetch per page")
	require.Contains(t, output, "--incident-message-limit int")
	require.Contains(t, output, "maximum characters to show for human incident messages when --with-incidents is set")
	require.Contains(t, output, "--limit int32")
	require.Contains(t, output, "maximum number of matching process instances to return or process across all pages")
	require.NotContains(t, output, "--count")
}

// Verifies help text documents has-user-tasks as a compact lookup selector without overloaded examples.
func TestGetProcessInstanceHelp_DocumentsHasUserTasksLookup(t *testing.T) {
	output := executeRootForProcessInstanceTest(t, "get", "process-instance", "--help")

	require.Contains(t, output, "--has-user-tasks strings")
	require.Contains(t, output, "user task key(s) whose owning process instances should be fetched")
	require.Contains(t, output, "./c8volt get pi --has-user-tasks <user-task-key>")
	require.NotContains(t, output, "./c8volt get pi --has-user-tasks 2251799815391233 --has-user-tasks 2251799815391244")
	require.Contains(t, output, "Camunda v2 user-task search first")
	require.Contains(t, output, "Tasklist V1 lookup for legacy user-task compatibility")
	require.Contains(t, output, "Camunda 8.7 remains unsupported")
	require.NotContains(t, output, "There is no Tasklist or Operate fallback")
}

// Verifies search-mode get process-instance sends the expected filter and pagination request shape.
func TestGetProcessInstanceSearchScaffold_UsesTempConfigAndCapturesSearchRequest(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--state", "active",
		"--batch-size", "5",
	)

	filter := decodeCapturedPISearchFilter(t, requests)
	page := decodeCapturedPISearchPage(t, requests)

	require.Equal(t, "ACTIVE", filter["state"])
	require.EqualValues(t, 5, page["limit"])

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, string(OutcomeSucceeded), got["outcome"])
	require.Equal(t, "get process-instance", got["command"])
}

// TestGetProcessInstanceJSON_AddsAgeMetaField verifies JSON rows include age metadata.
func TestGetProcessInstanceJSON_AddsAgeMetaField(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
	)

	require.NotEmpty(t, requests)
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	payload, ok := got["payload"].(map[string]any)
	require.True(t, ok)
	meta, ok := payload["meta"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, meta["withAge"])
}

// Protects default paged search output, which renders incrementally before the final collected list path can align rows.
func TestGetProcessInstanceSearch_HumanOutputAlignsIncrementalPage(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 3, 23, 19, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	var requests []string
	srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"hasIncident":false,"processDefinitionId":"Short","processDefinitionKey":"9001","processDefinitionName":"Short","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"MuchLongerProcess","processDefinitionKey":"9002","processDefinitionName":"MuchLongerProcess","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--batch-size", "2",
	)

	require.NotEmpty(t, requests)
	expectedLines := formatProcessInstanceFlatRows([]process.ProcessInstance{
		{
			Key:            "123",
			TenantId:       "tenant",
			BpmnProcessId:  "Short",
			ProcessVersion: 3,
			State:          process.StateActive,
			StartDate:      "2026-03-23T18:00:00Z",
		},
		{
			Key:            "124",
			TenantId:       "tenant",
			BpmnProcessId:  "MuchLongerProcess",
			ProcessVersion: 3,
			State:          process.StateCompleted,
			StartDate:      "2026-03-23T18:00:00Z",
		},
	})
	require.Equal(t, strings.Join(append(expectedLines, "found: 2", ""), "\n"), output)
	require.Contains(t, output, "Short             v3 ACTIVE")
}

// TestGetProcessInstanceTotalOutput verifies --total output uses exact fallback counting when backend totals are capped.
func TestGetProcessInstanceTotalOutput(t *testing.T) {
	t.Run("reported total prints only the numeric count without fetching later pages", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"get", "process-instance",
			"--batch-size", "2",
			"--total",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Zero(t, promptCalls)
		require.Equal(t, "3\n", stdout)
		require.Empty(t, stderr)
	})

	t.Run("capped reported total falls back to cursor paging for exact count", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":"cursor-1","startCursor":null}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":"cursor-2","startCursor":"cursor-1"}}`,
			`{"items":[],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":null,"startCursor":"cursor-2"}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"get", "process-instance",
			"--batch-size", "2",
			"--total",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 3)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.Equal(t, "cursor-1", pages[1]["after"])
		require.NotContains(t, pages[1], "from")
		require.Equal(t, "cursor-2", pages[2]["after"])
		require.Zero(t, promptCalls)
		require.Equal(t, "3\n", stdout)
		require.Empty(t, stderr)
	})

	t.Run("verbose capped total logs progress through logger", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":"cursor-1","startCursor":null}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":"cursor-2","startCursor":"cursor-1"}}`,
			`{"items":[],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":null,"startCursor":"cursor-2"}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"get", "process-instance",
			"--batch-size", "2",
			"--total",
			"--verbose",
		)

		require.Equal(t, "3\n", stdout)
		require.Contains(t, stderr, "INFO page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
		require.Contains(t, stderr, "INFO page size: 2, current page: 1, total so far: 3, more matches: yes, next step: auto-continue")
		require.Contains(t, stderr, "INFO page size: 2, current page: 0, total so far: 3, more matches: yes, next step: auto-continue")
		require.NotContains(t, stderr, "\npage size:")
	})

	t.Run("debug capped total includes paging values", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":"cursor-1","startCursor":null}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":"cursor-2","startCursor":"cursor-1"}}`,
			`{"items":[],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":null,"startCursor":"cursor-2"}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--debug",
			"get", "process-instance",
			"--batch-size", "2",
			"--total",
		)

		require.Equal(t, "3\n", stdout)
		require.Contains(t, stderr, `DEBUG process-instance total page: mode=offset, from=0, after="", limit=2, items=2, total before=0, total after=2`)
		require.Contains(t, stderr, `reported total=10000, reported kind=lower_bound, end cursor="cursor-1"`)
		require.Contains(t, stderr, `DEBUG process-instance total page: mode=cursor, from=0, after="cursor-1", limit=2, items=1, total before=2, total after=3`)
		require.Contains(t, stderr, `DEBUG process-instance total page: mode=cursor, from=0, after="cursor-2", limit=2, items=0, total before=3, total after=3`)
		require.NotContains(t, stderr, "INFO page size:")
	})

	t.Run("zero matches still print zero only", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"get", "process-instance",
			"--total",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Equal(t, "0\n", stdout)
		require.Empty(t, stderr)
	})
}

// TestGetProcessInstanceTotalValidation verifies --total rejects incompatible output and lookup modes.
func TestGetProcessInstanceTotalValidation(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "key lookup stays on the strict single-resource path",
			helper: "TestGetProcessInstanceTotalWithKeyHelper",
			want:   "--total cannot be combined with --key",
		},
		{
			name:   "json output is rejected",
			helper: "TestGetProcessInstanceTotalWithJSONHelper",
			want:   "--total cannot be combined with --json",
		},
		{
			name:   "keys-only output is rejected",
			helper: "TestGetProcessInstanceTotalWithKeysOnlyHelper",
			want:   "--total cannot be combined with --keys-only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, tt.want)
		})
	}
}

// TestGetProcessInstanceWithIncidentsValidation rejects enrichment combinations that cannot render incident details safely.
func TestGetProcessInstanceWithIncidentsValidation(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "rejects search-mode incident filters",
			helper: "TestGetProcessInstanceWithIncidentsWithSearchFilterHelper",
			want:   "--with-incidents cannot be combined with search-mode filters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, tt.want)
		})
	}
}

// TestGetProcessInstanceListWithIncidents_HumanOutputShowsDirectIncidentLines verifies list/search incident enrichment keeps incidents under their owning rows.
func TestGetProcessInstanceListWithIncidents_HumanOutputShowsDirectIncidentLines(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var got map[string]any
			require.NoError(t, json.Unmarshal(body, &got))
			filter := requireJSONObject(t, got["filter"])
			require.Equal(t, true, filter["hasIncident"])
			_, _ = w.Write([]byte(`{"items":[
				{"hasIncident":true,"processDefinitionId":"demo-a","processDefinitionKey":"9001","processDefinitionName":"demo-a","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},
				{"hasIncident":true,"processDefinitionId":"demo-b","processDefinitionKey":"9002","processDefinitionName":"demo-b","processDefinitionVersion":4,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"First key failed","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/124/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"Second key failed","incidentKey":"incident-124","processInstanceKey":"124","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--incidents-only",
		"--with-incidents",
	)

	require.Equal(t, []string{
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/123/incidents/search",
		"POST /v2/process-instances/124/incidents/search",
	}, requests)
	require.Contains(t, output, "123 tenant demo-a v3 ACTIVE")
	require.Contains(t, output, "  inc incident-123: First key failed")
	require.Contains(t, output, "124 tenant demo-b v4 ACTIVE")
	require.Contains(t, output, "  inc incident-124: Second key failed")
	require.Contains(t, output, "found: 2")
	require.Less(t, strings.Index(output, "123 tenant demo-a"), strings.Index(output, "  inc incident-123"))
	require.Less(t, strings.Index(output, "  inc incident-123"), strings.Index(output, "124 tenant demo-b"))
	require.Less(t, strings.Index(output, "124 tenant demo-b"), strings.Index(output, "  inc incident-124"))
}

// TestGetProcessInstanceListWithIncidents_LooksUpOnlyLimitedRows guards paging and --limit compatibility for incident lookups.
func TestGetProcessInstanceListWithIncidents_LooksUpOnlyLimitedRows(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[
				{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},
				{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"First key failed","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/124/incidents/search":
			t.Fatalf("incident lookup should not run for rows outside --limit: %s", r.URL.Path)
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--batch-size", "2",
		"--limit", "1",
		"--with-incidents",
	)

	require.Equal(t, []string{
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/123/incidents/search",
	}, requests)
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "  inc incident-123: First key failed")
	require.NotContains(t, output, "124 tenant")
	require.Contains(t, output, "found: 1")
}

// TestGetProcessInstanceListWithIncidents_HumanIndirectMarkerExplainsEmptyDirectIncidents verifies list rows marked inc! stay explainable when direct lookup is empty.
func TestGetProcessInstanceListWithIncidents_HumanIndirectMarkerExplainsEmptyDirectIncidents(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[
				{"hasIncident":true,"processDefinitionId":"demo-a","processDefinitionKey":"9001","processDefinitionName":"demo-a","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},
				{"hasIncident":true,"processDefinitionId":"demo-b","processDefinitionKey":"9002","processDefinitionName":"demo-b","processDefinitionVersion":4,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123/incidents/search", "/v2/process-instances/124/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--incidents-only",
		"--with-incidents",
	)

	require.Equal(t, []string{
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/123/incidents/search",
		"POST /v2/process-instances/124/incidents/search",
	}, requests)
	require.Contains(t, stdout, "123 tenant demo-a v3 ACTIVE")
	require.Contains(t, stdout, "124 tenant demo-b v4 ACTIVE")
	require.Equal(t, 2, strings.Count(stdout, "  "+indirectProcessTreeIncidentNote))
	require.Contains(t, stdout, "found: 2")
	require.NotContains(t, stdout, indirectProcessTreeIncidentWarning)
	require.Equal(t, 1, strings.Count(stderr, indirectProcessTreeIncidentWarning))
	require.Less(t, strings.Index(stdout, "123 tenant demo-a"), strings.Index(stdout, "  "+indirectProcessTreeIncidentNote))
	require.Less(t, strings.Index(stdout, "124 tenant demo-b"), strings.LastIndex(stdout, "  "+indirectProcessTreeIncidentNote))
	require.Less(t, strings.LastIndex(stdout, "  "+indirectProcessTreeIncidentNote), strings.Index(stdout, "found: 2"))
}

// TestGetProcessInstanceIncidentMessageLimitValidation rejects unsafe incident message limit usage.
func TestGetProcessInstanceIncidentMessageLimitValidation(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "requires with-incidents",
			helper: "TestGetProcessInstanceIncidentMessageLimitWithoutIncidentsHelper",
			want:   "--incident-message-limit requires --with-incidents",
		},
		{
			name:   "rejects negative limit",
			helper: "TestGetProcessInstanceIncidentMessageLimitNegativeHelper",
			want:   "invalid value for --incident-message-limit: -1, expected non-negative integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, tt.want)
		})
	}
}

// TestGetProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags verifies paging flag validation errors stay user-facing.
func TestGetProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "removed count flag is rejected",
			helper: "TestGetProcessInstanceCommand_RejectsRemovedCountFlagHelper",
			want:   "unknown flag: --count",
		},
		{
			name:   "non-positive limit is rejected",
			helper: "TestGetProcessInstanceCommand_RejectsInvalidLimitHelper",
			want:   "--limit must be positive integer",
		},
		{
			name:   "limit cannot be combined with key",
			helper: "TestGetProcessInstanceCommand_RejectsLimitWithKeyHelper",
			want:   "--limit cannot be combined with --key",
		},
		{
			name:   "limit cannot be combined with total",
			helper: "TestGetProcessInstanceCommand_RejectsLimitWithTotalHelper",
			want:   "--total cannot be combined with --limit",
		},
		{
			name:   "invalid batch size is rejected",
			helper: "TestGetProcessInstanceCommand_RejectsInvalidBatchSizeHelper",
			want:   "invalid value for --batch-size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, tt.want)
		})
	}
}

// TestApplyPISearchResultFilters_OrphanChildrenUseCommandActivity verifies orphan filtering is wrapped in command activity output.
func TestApplyPISearchResultFilters_OrphanChildrenUseCommandActivity(t *testing.T) {
	prevOrphanOnly := flagGetPIOrphanChildrenOnly
	t.Cleanup(func() {
		flagGetPIOrphanChildrenOnly = prevOrphanOnly
	})
	flagGetPIOrphanChildrenOnly = true

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))
	cliFilterCalls := 0
	cli := stubProcessAPI{filterOrphanParent: func(ctx context.Context, items []process.ProcessInstance, opts ...options.FacadeOption) ([]process.ProcessInstance, error) {
		cliFilterCalls++
		return items[:1], nil
	}}

	pis := process.ProcessInstances{
		Total: 2,
		Items: []process.ProcessInstance{
			{Key: "123", ParentKey: "456"},
			{Key: "124", ParentKey: "457"},
		},
	}

	got, err := applyPISearchResultFilters(cmd, cli, pis)
	require.NoError(t, err)
	require.Equal(t, 1, cliFilterCalls)
	require.Len(t, got.Items, 1)
	require.EqualValues(t, 1, got.Total)

	started, stopped, msgs := sink.Snapshot()
	require.Equal(t, 1, started)
	require.Equal(t, 1, stopped)
	require.Equal(t, []string{"checking orphan parents for 2 process instance(s)"}, msgs)
}

// TestGetProcessInstanceKeyLookup_UsesGeneratedLookupEndpoint verifies direct key lookup uses the versioned generated endpoint.
func TestGetProcessInstanceKeyLookup_UsesGeneratedLookupEndpoint(t *testing.T) {
	response := `{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant-a"}`

	t.Run("explicit flag tenant", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/process-instances/123", r.URL.Path)
			requests = append(requests, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
  tenant: base-tenant
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"--tenant", "tenant-a",
			"get", "process-instance",
			"--key", "123",
		)

		require.Equal(t, []string{"/v2/process-instances/123"}, requests)
		require.Contains(t, output, `"tenantId": "tenant-a"`)
	})

	t.Run("environment tenant", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/process-instances/123", r.URL.Path)
			requests = append(requests, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

		output := executeRootForProcessInstanceTestWithEnv(t,
			[]string{"C8VOLT_APP_TENANT=tenant-a"},
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--key", "123",
		)

		require.Equal(t, []string{"/v2/process-instances/123"}, requests)
		require.Contains(t, output, `"tenantId": "tenant-a"`)
	})

	t.Run("profile tenant", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/process-instances/123", r.URL.Path)
			requests = append(requests, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeRawTestConfig(t, `active_profile: base
app:
  camunda_version: 8.8
  tenant: base-tenant
apis:
  camunda_api:
    base_url: `+srv.URL+`
profiles:
  dev:
    app:
      tenant: tenant-a
    apis:
      camunda_api:
        base_url: `+srv.URL+`
`)

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"--profile", "dev",
			"get", "process-instance",
			"--key", "123",
		)

		require.Equal(t, []string{"/v2/process-instances/123"}, requests)
		require.Contains(t, output, `"tenantId": "tenant-a"`)
	})

	t.Run("base config tenant", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/process-instances/123", r.URL.Path)
			requests = append(requests, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
  tenant: tenant-a
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--key", "123",
		)

		require.Equal(t, []string{"/v2/process-instances/123"}, requests)
		require.Contains(t, output, `"tenantId": "tenant-a"`)
	})
}

// TestGetProcessInstanceWithIncidents_HumanOutputShowsOneIncident verifies the direct incident line includes the incident key.
func TestGetProcessInstanceWithIncidents_HumanOutputShowsOneIncident(t *testing.T) {
	var requests []string
	var incidentBodies []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			incidentBodies = append(incidentBodies, string(body))
			_, _ = w.Write([]byte(`{"items":[{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"incident-123","jobKey":"job-123","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "process-instance",
		"--key", "123",
		"--with-incidents",
	)

	require.Equal(t, []string{"GET /v2/process-instances/123", "POST /v2/process-instances/123/incidents/search"}, requests)
	require.Len(t, incidentBodies, 1)
	require.NotContains(t, incidentBodies[0], "processInstanceKey")
	require.Contains(t, output, "123")
	require.Contains(t, output, "demo v3")
	require.Contains(t, output, "inc!")
	require.Contains(t, output, "  inc incident-123: No retries left")
	require.Contains(t, output, "found: 1")
}

func TestGetProcessInstanceWithIncidents_HumanIncidentMessageLimitTruncatesMessageOnly(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo-process","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"No retries left after worker failure","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "process-instance",
		"--key", "123",
		"--with-incidents",
		"--incident-message-limit", "7",
	)

	require.Equal(t, []string{"GET /v2/process-instances/123", "POST /v2/process-instances/123/incidents/search"}, requests)
	require.Contains(t, output, "123 tenant demo-process v3 ACTIVE")
	require.Contains(t, output, "  inc incident-123: No retr...")
	require.NotContains(t, output, "No retries left after worker failure")
}

func TestGetProcessInstanceWithIncidents_HumanIncidentMessageLimitDefaultLeavesMessageUnchanged(t *testing.T) {
	fullMessage := "No retries left after worker failure"
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"` + fullMessage + `","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "process-instance",
		"--key", "123",
		"--with-incidents",
	)

	require.Contains(t, output, "  inc incident-123: "+fullMessage)
	require.NotContains(t, output, fullMessage[:7]+"...")
}

// TestGetProcessInstanceWithIncidents_HumanOutputShowsMultipleAndNoIncidents covers both direct incident rendering and tree-propagated incident warnings.
func TestGetProcessInstanceWithIncidents_HumanOutputShowsMultipleAndNoIncidents(t *testing.T) {
	tests := []struct {
		name             string
		incidentResponse string
		wantMessages     []string
	}{
		{
			name: "multiple incident lines",
			incidentResponse: `{"items":[
				{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"incident-123","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"},
				{"creationTime":"2026-03-23T18:02:00Z","elementId":"task-b","elementInstanceKey":"element-124","errorMessage":"Gateway failed","errorType":"EXTRACT_VALUE_ERROR","incidentKey":"incident-124","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
			wantMessages: []string{"  inc incident-123: No retries left", "  inc incident-124: Gateway failed"},
		},
		{
			name:             "no incident lines",
			incidentResponse: `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`,
			wantMessages:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/process-instances/123":
					require.Equal(t, http.MethodGet, r.Method)
					_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
				case "/v2/process-instances/123/incidents/search":
					require.Equal(t, http.MethodPost, r.Method)
					_, _ = w.Write([]byte(tt.incidentResponse))
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"get", "process-instance",
				"--key", "123",
				"--with-incidents",
			)

			require.Contains(t, output, "123")
			require.Contains(t, output, "found: 1")
			for _, msg := range tt.wantMessages {
				require.Contains(t, output, msg)
			}
			if len(tt.wantMessages) == 0 {
				require.NotContains(t, output, "  inc ")
				require.Contains(t, output, indirectProcessTreeIncidentNote)
				require.Contains(t, output, indirectProcessTreeIncidentWarning)
			}
		})
	}
}

// TestGetProcessInstanceWithIncidents_JSONOutputShowsIncidentDetails preserves the structured incident detail payload.
func TestGetProcessInstanceWithIncidents_JSONOutputShowsIncidentDetails(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"incident-123","jobKey":"job-123","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "123",
		"--with-incidents",
	)

	payload := requireProcessInstanceIncidentJSONPayload(t, output)
	require.Equal(t, float64(1), payload["total"])
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	item := requireJSONObject(t, first["item"])
	require.Equal(t, "123", item["key"])

	incidents := requireJSONItems(t, first["incidents"], 1)
	incident := requireJSONObject(t, incidents[0])
	require.Equal(t, "incident-123", incident["incidentKey"])
	require.Equal(t, "123", incident["processInstanceKey"])
	require.Equal(t, "No retries left", incident["errorMessage"])
	require.Equal(t, "task-a", incident["flowNodeId"])
}

// TestGetProcessInstanceWithIncidents_JSONOutputAssociatesMultipleKeys prevents incident details from crossing keyed lookup boundaries.
func TestGetProcessInstanceWithIncidents_JSONOutputAssociatesMultipleKeys(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo-a","processDefinitionKey":"9001","processDefinitionName":"demo-a","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/124":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo-b","processDefinitionKey":"9002","processDefinitionName":"demo-b","processDefinitionVersion":4,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[
				{"errorMessage":"First key failed","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"},
				{"errorMessage":"wrong association","incidentKey":"incident-wrong","processInstanceKey":"124","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/124/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"Second key failed","incidentKey":"incident-124","processInstanceKey":"124","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "123",
		"--key", "124",
		"--workers", "1",
		"--with-incidents",
	)

	payload := requireProcessInstanceIncidentJSONPayload(t, output)
	require.Equal(t, float64(2), payload["total"])
	items := requireJSONItems(t, payload["items"], 2)

	first := requireJSONObject(t, items[0])
	firstItem := requireJSONObject(t, first["item"])
	require.Equal(t, "123", firstItem["key"])
	firstIncidents := requireJSONItems(t, first["incidents"], 1)
	require.Equal(t, "First key failed", requireJSONObject(t, firstIncidents[0])["errorMessage"])

	second := requireJSONObject(t, items[1])
	secondItem := requireJSONObject(t, second["item"])
	require.Equal(t, "124", secondItem["key"])
	secondIncidents := requireJSONItems(t, second["incidents"], 1)
	require.Equal(t, "Second key failed", requireJSONObject(t, secondIncidents[0])["errorMessage"])
}

// TestGetProcessInstanceWithIncidents_JSONOutputShowsEmptyIncidentCollection keeps empty enrichment explicit for automation.
func TestGetProcessInstanceWithIncidents_JSONOutputShowsEmptyIncidentCollection(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "123",
		"--with-incidents",
	)

	payload := requireProcessInstanceIncidentJSONPayload(t, output)
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	incidents := requireJSONItems(t, first["incidents"], 0)
	require.Empty(t, incidents)
}

func TestGetProcessInstanceJSONWithIncidents_ListSearchUsesEnrichedPayloadShape(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"hasIncident":true,"processDefinitionId":"demo-a","processDefinitionKey":"9001","processDefinitionName":"demo-a","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":true,"processDefinitionId":"demo-b","processDefinitionKey":"9002","processDefinitionName":"demo-b","processDefinitionVersion":4,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"First direct incident","errorType":"JOB_NO_RETRIES","incidentKey":"incident-123","jobKey":"job-123","processDefinitionId":"demo-a","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/124/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"creationTime":"2026-03-23T18:06:00Z","elementId":"task-b","elementInstanceKey":"element-124","errorMessage":"Second direct incident","errorType":"JOB_NO_RETRIES","incidentKey":"incident-124","jobKey":"job-124","processDefinitionId":"demo-b","processDefinitionKey":"9002","processInstanceKey":"124","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--state", "active",
		"--with-incidents",
		"--batch-size", "2",
	)

	payload := requireProcessInstanceIncidentJSONPayload(t, output)
	require.Equal(t, float64(2), payload["total"])
	meta := requireJSONObject(t, payload["meta"])
	require.Equal(t, true, meta["withAge"])
	items := requireJSONItems(t, payload["items"], 2)

	first := requireJSONObject(t, items[0])
	firstItem := requireJSONObject(t, first["item"])
	require.Equal(t, "123", firstItem["key"])
	firstIncidents := requireJSONItems(t, first["incidents"], 1)
	firstIncident := requireJSONObject(t, firstIncidents[0])
	require.Equal(t, "incident-123", firstIncident["incidentKey"])
	require.Equal(t, "123", firstIncident["processInstanceKey"])
	require.Equal(t, "First direct incident", firstIncident["errorMessage"])
	require.Equal(t, "task-a", firstIncident["flowNodeId"])

	second := requireJSONObject(t, items[1])
	secondItem := requireJSONObject(t, second["item"])
	require.Equal(t, "124", secondItem["key"])
	secondIncidents := requireJSONItems(t, second["incidents"], 1)
	secondIncident := requireJSONObject(t, secondIncidents[0])
	require.Equal(t, "incident-124", secondIncident["incidentKey"])
	require.Equal(t, "124", secondIncident["processInstanceKey"])
	require.Equal(t, "Second direct incident", secondIncident["errorMessage"])
	require.Equal(t, "task-b", secondIncident["flowNodeId"])
}

func TestGetProcessInstanceJSONWithIncidents_IncidentMessageLimitKeepsFullMessages(t *testing.T) {
	fullMessage := "This long incident message must remain complete in JSON output"
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"` + fullMessage + `","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--state", "active",
		"--with-incidents",
		"--incident-message-limit", "5",
	)

	payload := requireProcessInstanceIncidentJSONPayload(t, output)
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	incidents := requireJSONItems(t, first["incidents"], 1)
	incident := requireJSONObject(t, incidents[0])
	require.Equal(t, fullMessage, incident["errorMessage"])
	require.NotEqual(t, "This ...", incident["errorMessage"])
}

func TestGetProcessInstanceJSONWithIncidents_KeyedLookupShapeRemainsUnchanged(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"No retries left","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "123",
		"--with-incidents",
	)

	payload := requireProcessInstanceIncidentJSONPayload(t, output)
	require.Equal(t, float64(1), payload["total"])
	meta := requireJSONObject(t, payload["meta"])
	require.Equal(t, true, meta["withAge"])
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	require.Contains(t, first, "item")
	require.Contains(t, first, "incidents")
	item := requireJSONObject(t, first["item"])
	require.Equal(t, "123", item["key"])
	incidents := requireJSONItems(t, first["incidents"], 1)
	incident := requireJSONObject(t, incidents[0])
	require.Equal(t, "incident-123", incident["incidentKey"])
	require.Equal(t, "123", incident["processInstanceKey"])
	require.Equal(t, "No retries left", incident["errorMessage"])
}

// TestGetProcessInstanceWithIncidents_V87ReportsUnsupported preserves the tenant-safe version boundary.
func TestGetProcessInstanceWithIncidents_V87ReportsUnsupported(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, err := testx.RunCmdSubprocess(t, "TestGetProcessInstanceWithIncidentsUnsupportedV87Helper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "not tenant-safe in Camunda 8.7")
}

// TestGetProcessInstanceWithoutIncidents_HumanOutputPreservesDefault keeps default keyed output free of enrichment lines.
func TestGetProcessInstanceWithoutIncidents_HumanOutputPreservesDefault(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "process-instance",
		"--key", "123",
	)

	wantItem := process.ProcessInstance{
		Key:            "123",
		TenantId:       "tenant",
		BpmnProcessId:  "demo",
		ProcessVersion: 3,
		State:          process.StateActive,
		StartDate:      "2026-03-23T18:00:00Z",
		Incident:       true,
	}
	require.Equal(t, []string{"GET /v2/process-instances/123"}, requests)
	require.Equal(t, strings.TrimSpace(oneLinePI(wantItem))+"\nfound: 1\n", output)
	require.NotContains(t, output, "  inc ")
}

// TestGetProcessInstanceWithoutIncidents_JSONOutputPreservesDefaultShape keeps default JSON free of enrichment wrappers.
func TestGetProcessInstanceWithoutIncidents_JSONOutputPreservesDefaultShape(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "123",
	)

	require.Equal(t, []string{"GET /v2/process-instances/123"}, requests)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "get process-instance", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.NotContains(t, payload, "item")
	require.NotContains(t, payload, "incidents")
	require.Equal(t, float64(1), payload["total"])
	items := requireJSONItems(t, payload["items"], 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "123", item["key"])
	require.Equal(t, true, item["incident"])
	require.NotContains(t, item, "incidents")
}

// TestGetProcessInstanceSearchIncidentFilters_PreserveDefaultSearchMode keeps incident presence filters on the paged search path.
func TestGetProcessInstanceSearchIncidentFilters_PreserveDefaultSearchMode(t *testing.T) {
	tests := []struct {
		name         string
		flag         string
		wantIncident bool
		response     string
	}{
		{
			name:         "incidents only",
			flag:         "--incidents-only",
			wantIncident: true,
			response:     `{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
		},
		{
			name:         "no incidents only",
			flag:         "--no-incidents-only",
			wantIncident: false,
			response:     `{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests, tt.response)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				tt.flag,
			)

			filter := decodeCapturedPISearchFilter(t, requests)
			require.Equal(t, tt.wantIncident, filter["hasIncident"])
			require.NotContains(t, output, `"incidents"`)
			require.Contains(t, output, `"total": 1`)
		})
	}
}

// TestGetProcessInstanceSearch_V87StillSupportsTenantScopedSearch verifies v8.7 search keeps tenant scoping available.
func TestGetProcessInstanceSearch_V87StillSupportsTenantScopedSearch(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/process-instances/search", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"key":123,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"<default>"}]}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--state", "active",
	)

	filter := decodeCapturedPISearchFilter(t, requests)
	require.Equal(t, "<default>", filter["tenantId"])
	require.Equal(t, "ACTIVE", filter["state"])
	require.Contains(t, output, `"tenantId": "<default>"`)
}

// TestGetProcessInstanceCommand_V89KeyLookupUsesNativeSearchPath verifies v8.9 direct lookup uses the native single-instance endpoint.
func TestGetProcessInstanceCommand_V89KeyLookupUsesNativeSearchPath(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/2251799813711967", r.URL.Path)
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "2251799813711967",
	)

	require.Equal(t, []string{"/v2/process-instances/2251799813711967"}, requests)
	require.Contains(t, output, `"key": "2251799813711967"`)
}

// Verifies has-user-tasks resolves through native user-task search, then reuses keyed process-instance rendering.
func TestGetProcessInstanceCommand_HasUserTasksLookupUsesNativeUserTaskAndKeyedProcessInstance(t *testing.T) {
	for _, version := range []string{"8.8", "8.9"} {
		t.Run(version, func(t *testing.T) {
			var requests []string
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/user-tasks/search":
					requireUserTaskSearchRequest(t, r, "2251799815391233", "")
					_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
				case "/v2/process-instances/2251799813711967":
					require.Equal(t, http.MethodGet, r.Method)
					_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, version)

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"get", "pi",
				"--has-user-tasks", "2251799815391233",
			)

			require.Equal(t, []string{
				"/v2/user-tasks/search",
				"/v2/process-instances/2251799813711967",
			}, requests)
			require.Contains(t, output, "2251799813711967")
			require.NotContains(t, output, "2251799815391233")
		})
	}
}

// Verifies has-user-tasks falls back through Tasklist V1 after a native lookup miss and renders the resolved process instance.
func TestGetProcessInstanceCommand_HasUserTasksFallbackUsesTasklistAndKeyedProcessInstance(t *testing.T) {
	for _, version := range []string{"8.8", "8.9"} {
		t.Run(version, func(t *testing.T) {
			var requests []string
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/user-tasks/search":
					requireUserTaskSearchRequest(t, r, "2251799815391233", "")
					_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
				case "/v1/tasks/2251799815391233":
					requireTasklistFallbackTaskRequest(t, r, "2251799815391233")
					_, _ = w.Write([]byte(`{"id":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant","implementation":"JOB_WORKER"}`))
				case "/v2/process-instances/2251799813711967":
					require.Equal(t, http.MethodGet, r.Method)
					_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, version)

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"get", "pi",
				"--has-user-tasks", "2251799815391233",
			)

			require.Equal(t, []string{
				"/v2/user-tasks/search",
				"/v1/tasks/2251799815391233",
				"/v2/process-instances/2251799813711967",
			}, requests)
			require.Contains(t, output, "2251799813711967")
			require.NotContains(t, output, "2251799815391233")
		})
	}
}

// Verifies has-user-tasks lookup applies the effective tenant while resolving the owning process instance.
func TestGetProcessInstanceCommand_HasUserTasksLookupIncludesEffectiveTenant(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user-tasks/search":
			requireUserTaskSearchRequest(t, r, "2251799815391233", "tenant-a")
			_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/2251799813711967":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant-a"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant-a",
		"get", "pi",
		"--has-user-tasks", "2251799815391233",
	)

	require.Contains(t, output, "2251799813711967")
}

// Verifies repeated has-user-tasks values resolve each task and render the resulting process instances.
func TestGetProcessInstanceCommand_HasUserTasksLookupAcceptsMultipleKeys(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user-tasks/search":
			body := requireUserTaskSearchRequest(t, r, "", "")
			switch body["filter"].(map[string]any)["userTaskKey"] {
			case "2251799815391233":
				_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			case "2251799815391244":
				_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391244","processInstanceKey":"2251799813711977","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected user task search body: %v", body)
			}
		case "/v2/process-instances/2251799813711967":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/2251799813711977":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711977","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "pi",
		"--has-user-tasks", "2251799815391233",
		"--has-user-tasks", "2251799815391244",
		"--workers", "1",
	)

	require.Equal(t, []string{
		"/v2/user-tasks/search",
		"/v2/user-tasks/search",
		"/v2/process-instances/2251799813711967",
		"/v2/process-instances/2251799813711977",
	}, requests)
	require.Contains(t, output, "2251799813711967")
	require.Contains(t, output, "2251799813711977")
}

// Verifies repeated has-user-tasks values resolve each task through the first successful path for that task.
func TestGetProcessInstanceCommand_HasUserTasksLookupMixesPrimaryAndFallbackKeys(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user-tasks/search":
			body := requireUserTaskSearchRequest(t, r, "", "")
			switch body["filter"].(map[string]any)["userTaskKey"] {
			case "2251799815391233":
				_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			case "2251799815391244":
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected user task search body: %v", body)
			}
		case "/v1/tasks/2251799815391244":
			requireTasklistFallbackTaskRequest(t, r, "2251799815391244")
			_, _ = w.Write([]byte(`{"id":"2251799815391244","processInstanceKey":"2251799813711977","tenantId":"tenant","implementation":"JOB_WORKER"}`))
		case "/v2/process-instances/2251799813711967":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/2251799813711977":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711977","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "pi",
		"--has-user-tasks", "2251799815391233",
		"--has-user-tasks", "2251799815391244",
		"--workers", "1",
	)

	require.Equal(t, []string{
		"/v2/user-tasks/search",
		"/v2/user-tasks/search",
		"/v1/tasks/2251799815391244",
		"/v2/process-instances/2251799813711967",
		"/v2/process-instances/2251799813711977",
	}, requests)
	require.Contains(t, output, "2251799813711967")
	require.Contains(t, output, "2251799813711977")
}

// Verifies has-user-tasks JSON output stays identical to direct keyed lookup for the resolved process instance.
func TestGetProcessInstanceCommand_HasUserTasksJSONMatchesDirectKeyedJSON(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user-tasks/search":
			requireUserTaskSearchRequest(t, r, "2251799815391233", "")
			_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/2251799813711967":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	taskKeyOutput := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--has-user-tasks", "2251799815391233",
	)
	directKeyOutput := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "2251799813711967",
	)

	require.Equal(t, []string{
		"/v2/user-tasks/search",
		"/v2/process-instances/2251799813711967",
		"/v2/process-instances/2251799813711967",
	}, requests)
	require.JSONEq(t, directKeyOutput, taskKeyOutput)
}

// Verifies fallback-resolved JSON output stays identical to direct keyed lookup for the resolved process instance.
func TestGetProcessInstanceCommand_HasUserTasksFallbackJSONMatchesDirectKeyedJSON(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user-tasks/search":
			requireUserTaskSearchRequest(t, r, "2251799815391233", "")
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		case "/v1/tasks/2251799815391233":
			requireTasklistFallbackTaskRequest(t, r, "2251799815391233")
			_, _ = w.Write([]byte(`{"id":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant","implementation":"JOB_WORKER"}`))
		case "/v2/process-instances/2251799813711967":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	taskKeyOutput := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--has-user-tasks", "2251799815391233",
	)
	directKeyOutput := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "2251799813711967",
	)

	require.Equal(t, []string{
		"/v2/user-tasks/search",
		"/v1/tasks/2251799815391233",
		"/v2/process-instances/2251799813711967",
		"/v2/process-instances/2251799813711967",
	}, requests)
	require.JSONEq(t, directKeyOutput, taskKeyOutput)
}

// Verifies has-user-tasks lookup preserves render flags that are valid for direct single-instance lookup.
func TestGetProcessInstanceCommand_HasUserTasksPreservesSingleLookupRenderFlags(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "default age",
			args: nil,
			want: "(2 days ago)",
		},
		{
			name: "keys only",
			args: []string{"--keys-only"},
			want: "2251799813711967\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []string
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/user-tasks/search":
					requireUserTaskSearchRequest(t, r, "2251799815391233", "")
					_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
				case "/v2/process-instances/2251799813711967":
					require.Equal(t, http.MethodGet, r.Method)
					_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
			args := append([]string{
				"--config", cfgPath,
				"get", "process-instance",
				"--has-user-tasks", "2251799815391233",
			}, tt.args...)

			output := executeRootForProcessInstanceTest(t, args...)

			require.Equal(t, []string{
				"/v2/user-tasks/search",
				"/v2/process-instances/2251799813711967",
			}, requests)
			require.Contains(t, output, tt.want)
		})
	}
}

// Verifies a missing resolved process instance keeps the not-found behavior of direct keyed lookup.
func TestGetProcessInstanceCommand_HasUserTasksPreservesResolvedProcessInstanceNotFound(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user-tasks/search":
			requireUserTaskSearchRequest(t, r, "2251799815391233", "")
			_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/2251799813711967":
			require.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceCommand_HasUserTasksResolvedProcessInstanceNotFoundHelper", cfgPath)

	require.Equal(t, exitcode.NotFound, code)
	require.Contains(t, output, "resource not found")
	require.Contains(t, output, "get process instance(s) resolved from user task key(s) [2251799815391233]")
	require.Contains(t, output, "/v2/process-instances/2251799813711967")
	require.Equal(t, []string{
		"/v2/user-tasks/search",
		"/v2/process-instances/2251799813711967",
	}, requests)
}

// Verifies numeric but unknown user-task keys reach native lookup and return not-found, not validation failure.
func TestGetProcessInstanceCommand_HasUserTasksMissingTaskReturnsNotFoundForShortNumericKey(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user-tasks/search":
			requireUserTaskSearchRequest(t, r, "225179981539123", "")
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		case "/v1/tasks/225179981539123":
			requireTasklistFallbackTaskRequest(t, r, "225179981539123")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, code := executeProcessInstanceFailureHelperWithEnv(t,
		"TestGetProcessInstanceCommand_HasUserTasksLookupFailureHelper",
		cfgPath,
		map[string]string{"C8VOLT_TEST_HAS_USER_TASKS_KEY": "225179981539123"},
	)

	require.Equal(t, exitcode.NotFound, code)
	require.Contains(t, output, "resource not found")
	require.Contains(t, output, "fallback user task 225179981539123 was not found or is not visible to the configured tenant")
	require.NotContains(t, output, "invalid input")
	require.Equal(t, []string{"/v2/user-tasks/search", "/v1/tasks/225179981539123"}, requests)
}

// Verifies malformed has-user-tasks values fail validation before any network lookup is attempted.
func TestGetProcessInstanceCommand_HasUserTasksRejectsNonDecimalKeyBeforeLookup(t *testing.T) {
	var requestCount int32
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, code := executeProcessInstanceFailureHelperWithEnv(t,
		"TestGetProcessInstanceCommand_HasUserTasksLookupFailureHelper",
		cfgPath,
		map[string]string{"C8VOLT_TEST_HAS_USER_TASKS_KEY": "not-a-key"},
	)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, `invalid value for --has-user-tasks: "not-a-key" at index 0 is not a positive decimal user task key`)
	require.Equal(t, int32(0), atomic.LoadInt32(&requestCount))
}

// Verifies has-user-tasks selector conflicts fail before any user-task or process-instance request is made.
func TestGetProcessInstanceCommand_RejectsHasUserTasksConflictsBeforeLookup(t *testing.T) {
	var requestCount int32
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	tests := []struct {
		name string
		mode string
		want string
	}{
		{
			name: "key selector",
			mode: "key",
			want: "--has-user-tasks cannot be combined with --key or stdin key input",
		},
		{
			name: "stdin key selector",
			mode: "stdin",
			want: "--has-user-tasks cannot be combined with --key or stdin key input",
		},
		{
			name: "state filter",
			mode: "state",
			want: "--has-user-tasks cannot be combined with process-instance search filters",
		},
		{
			name: "process definition filter",
			mode: "bpmn-process-id",
			want: "--has-user-tasks cannot be combined with process-instance search filters",
		},
		{
			name: "date filter",
			mode: "start-date-after",
			want: "--has-user-tasks cannot be combined with process-instance search filters",
		},
		{
			name: "derived search filter",
			mode: "roots-only",
			want: "--has-user-tasks cannot be combined with process-instance search filters",
		},
		{
			name: "total mode",
			mode: "total",
			want: "--has-user-tasks cannot be combined with --total",
		},
		{
			name: "limit mode",
			mode: "limit",
			want: "--has-user-tasks cannot be combined with --limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := atomic.LoadInt32(&requestCount)
			output, code := executeProcessInstanceFailureHelperWithEnv(t,
				"TestGetProcessInstanceCommand_RejectsHasUserTasksConflictHelper",
				cfgPath,
				map[string]string{"C8VOLT_TEST_HAS_USER_TASKS_CONFLICT": tt.mode},
			)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, tt.want)
			require.Equal(t, before, atomic.LoadInt32(&requestCount))
		})
	}
}

// Verifies Camunda 8.7 reports has-user-tasks as unsupported instead of falling back to another API.
func TestGetProcessInstanceCommand_HasUserTasksUnsupportedOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceCommand_HasUserTasksUnsupportedOnV87Helper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "has-user-tasks lookup is unsupported in Camunda 8.7")
	require.Contains(t, output, "requires Camunda 8.8 or 8.9")
}

// requireUserTaskSearchRequest validates the native user-task search request and returns its decoded body for scenario-specific assertions.
func requireUserTaskSearchRequest(t *testing.T, r *http.Request, taskKey, tenantID string) map[string]any {
	t.Helper()
	require.Equal(t, http.MethodPost, r.Method)
	require.Equal(t, "/v2/user-tasks/search", r.URL.Path)
	raw, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok, "expected user task search filter in %s", string(raw))
	if taskKey != "" {
		require.Equal(t, taskKey, filter["userTaskKey"])
	}
	if tenantID != "" {
		require.Equal(t, tenantID, filter["tenantId"])
	}
	return body
}

// requireTasklistFallbackTaskRequest protects the contract that legacy Tasklist URL ids are looked up directly.
func requireTasklistFallbackTaskRequest(t *testing.T, r *http.Request, taskKey string) {
	t.Helper()
	require.Equal(t, http.MethodGet, r.Method)
	require.Equal(t, "/v1/tasks/"+taskKey, r.URL.Path)
}

// Verifies get process-instance date filters map to expected API query fields and invalid combinations are rejected.
func TestGetProcessInstanceDateFilterScaffold(t *testing.T) {
	t.Run("start date command coverage", func(t *testing.T) {
		t.Run("lower bound only", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--start-date-after", "2026-01-01",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
			require.NotContains(t, filter["startDate"], "$lte")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})

		t.Run("inclusive range", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--start-date-after", "2026-01-01",
				"--start-date-before", "2026-01-31",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
			requireCapturedPISearchDateBound(t, filter, "startDate", "$lte", "2026-01-31T23:59:59.999999999Z")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})
	})

	t.Run("end date command coverage", func(t *testing.T) {
		t.Run("lower bound only", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--end-date-after", "2026-02-01",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-02-01T00:00:00Z")
			requireCapturedPISearchDateExists(t, filter, "endDate")
			require.NotContains(t, filter["endDate"], "$lte")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})

		t.Run("inclusive range composed with state filter", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--state", "completed",
				"--end-date-after", "2026-02-01",
				"--end-date-before", "2026-03-31",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			require.Equal(t, "COMPLETED", filter["state"])
			requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-02-01T00:00:00Z")
			requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-03-31T23:59:59.999999999Z")
			requireCapturedPISearchDateExists(t, filter, "endDate")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})
	})

	t.Run("invalid date command coverage", func(t *testing.T) {
		t.Run("invalid start-date format exits through shared invalid-input path", func(t *testing.T) {
			cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

			output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceInvalidDateFormatHelper", cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, `invalid value for --start-date-after: "2026-02-30", expected YYYY-MM-DD`)
		})

		t.Run("invalid start-date range exits through shared invalid-input path", func(t *testing.T) {
			cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

			output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceInvalidStartDateRangeHelper", cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, `invalid range for --start-date-after and --start-date-before: "2026-02-01" is later than "2026-01-31"`)
		})

		t.Run("date filters are rejected for direct key lookup", func(t *testing.T) {
			cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

			output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceDateFiltersWithKeyHelper", cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, "date filters are only supported for list/search usage and cannot be combined with --key")
		})
	})
}

// TestGetProcessInstanceRelativeDayFilterScaffold verifies relative-day filters derive stable date bounds for search.
func TestGetProcessInstanceRelativeDayFilterScaffold(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	t.Run("start-day range search request uses derived absolute bounds", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServer(t, &requests)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--start-date-older-days", "7",
			"--start-date-newer-days", "30",
		)

		filter := decodeCapturedPISearchFilter(t, requests)

		requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-03-11T00:00:00Z")
		requireCapturedPISearchDateBound(t, filter, "startDate", "$lte", "2026-04-03T23:59:59.999999999Z")

		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.NotContains(t, got, "error")
	})

	t.Run("end-day upper bound search request uses derived absolute bounds", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServer(t, &requests)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--state", "completed",
			"--end-date-newer-days", "14",
		)

		filter := decodeCapturedPISearchFilter(t, requests)

		require.Equal(t, "COMPLETED", filter["state"])
		requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-03-27T00:00:00Z")
		requireCapturedPISearchDateExists(t, filter, "endDate")

		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.NotContains(t, got, "error")
	})
}

// TestGetProcessInstanceRelativeDayValidation verifies invalid relative-day ranges and combinations are rejected.
func TestGetProcessInstanceRelativeDayValidation(t *testing.T) {
	t.Run("negative relative-day values exit through shared invalid-input path", func(t *testing.T) {
		cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

		output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceNegativeRelativeDayHelper", cfgPath)

		require.Equal(t, exitcode.InvalidArgs, code)
		require.Contains(t, output, "invalid input")
		require.Contains(t, output, "invalid value for --start-date-older-days: -2, expected non-negative integer")
	})

	t.Run("mixed absolute and relative start-date filters are rejected", func(t *testing.T) {
		cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

		output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceMixedAbsoluteAndRelativeDateFiltersHelper", cfgPath)

		require.Equal(t, exitcode.InvalidArgs, code)
		require.Contains(t, output, "invalid input")
		require.Contains(t, output, "start-date absolute and relative day filters cannot be combined")
	})

	t.Run("invalid derived relative-day ranges are rejected", func(t *testing.T) {
		cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

		output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceInvalidRelativeDayRangeHelper", cfgPath)

		require.Equal(t, exitcode.InvalidArgs, code)
		require.Contains(t, output, "invalid input")
		require.Contains(t, output, `invalid range for --start-date-newer-days and --start-date-older-days: "2026-04-03" is later than "2026-03-11"`)
	})

	t.Run("local-day derivation honors the configured day boundary override", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServer(t, &requests)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTestWithEnv(t,
			[]string{testRelativeDayNowEnv + "=2026-04-10T00:30:00+02:00"},
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--start-date-older-days", "0",
		)

		filter := decodeCapturedPISearchFilter(t, requests)

		requireCapturedPISearchDateBound(t, filter, "startDate", "$lte", "2026-04-10T23:59:59.999999999Z")

		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.NotContains(t, got, "error")
	})
}

// TestPopulatePISearchFilterOpts_DerivesRelativeDayBounds verifies command options convert relative days to canonical dates.
func TestPopulatePISearchFilterOpts_DerivesRelativeDayBounds(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	flagGetPIStartAfterDays = 7
	flagGetPIStartBeforeDays = 30
	flagGetPIEndAfterDays = 14
	flagGetPIEndBeforeDays = 1

	filter := populatePISearchFilterOpts()

	require.Equal(t, "2026-03-11", filter.StartDateAfter)
	require.Equal(t, "2026-04-03", filter.StartDateBefore)
	require.Equal(t, "2026-04-09", filter.EndDateAfter)
	require.Equal(t, "2026-03-27", filter.EndDateBefore)
}

// TestPopulatePISearchFilterOpts_TranslatesSupportedPresenceFlags verifies parent and incident flags become facade options.
func TestPopulatePISearchFilterOpts_TranslatesSupportedPresenceFlags(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIChildrenOnly = true
	flagGetPIIncidentsOnly = true

	filter := populatePISearchFilterOpts()

	require.Equal(t, new(true), filter.HasParent)
	require.Equal(t, new(true), filter.HasIncident)
}

// TestValidatePISearchFlags_RejectsMixedAbsoluteAndRelativeInputs verifies absolute and relative date modes are exclusive.
func TestValidatePISearchFlags_RejectsMixedAbsoluteAndRelativeInputs(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIStartDateAfter = "2026-04-03"
	flagGetPIStartBeforeDays = 7

	err := validatePISearchFlags()

	require.Error(t, err)
	require.Contains(t, err.Error(), "start-date absolute and relative day filters cannot be combined")
}

func TestResetProcessInstanceCommandGlobals_ResetsIncidentMessageLimit(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIIncidentMessageLimit = 80

	resetProcessInstanceCommandGlobals()

	require.Zero(t, flagGetPIIncidentMessageLimit)
}

// TestHasPISearchFilterFlags_WithRelativeDaysOnly verifies relative-day flags activate search mode.
func TestHasPISearchFilterFlags_WithRelativeDaysOnly(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIStartAfterDays = 72

	require.True(t, hasPISearchFilterFlags())
}

// TestResolvePISearchSize verifies page-size precedence from flags, config, and defaults.
func TestResolvePISearchSize(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	cmd := getProcessInstanceCmd
	resetPISearchBatchSizeFlag(t, cmd)

	t.Run("uses shared config default when batch-size flag is unchanged", func(t *testing.T) {
		resetPISearchBatchSizeFlag(t, cmd)
		cfg := &config.Config{}
		cfg.App.ProcessInstancePageSize = 250

		require.Equal(t, int32(250), resolvePISearchSize(cmd, cfg))
	})

	t.Run("uses batch-size override when the flag is changed", func(t *testing.T) {
		resetPISearchBatchSizeFlag(t, cmd)
		require.NoError(t, cmd.Flags().Set("batch-size", "125"))
		cfg := &config.Config{}
		cfg.App.ProcessInstancePageSize = 250

		require.Equal(t, int32(125), resolvePISearchSize(cmd, cfg))
	})

	t.Run("falls back to repository default for invalid config values", func(t *testing.T) {
		resetProcessInstanceCommandGlobals()
		resetPISearchBatchSizeFlag(t, cmd)
		cfg := &config.Config{}
		cfg.App.ProcessInstancePageSize = 0

		require.Equal(t, int32(consts.MaxPISearchSize), resolvePISearchSize(cmd, cfg))
	})
}

// TestGetProcessInstancePagingFlow verifies interactive, automatic, and limited paging behavior.
func TestGetProcessInstancePagingFlow(t *testing.T) {
	t.Run("limit truncates results across pages and stops without continuation prompt", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":5,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"126","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":5,"hasMoreTotalItems":true}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--auto-confirm",
			"get", "process-instance",
			"--state", "active",
			"--batch-size", "2",
			"--limit", "3",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: yes, next step: limit-reached")
		require.Contains(t, output, "detail: stopped after reaching limit of 3 process instance(s)")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.Contains(t, output, "125")
		require.NotContains(t, output, "126")
	})

	t.Run("uses shared config default and prompts before the next page", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		prompts := []string{}
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			prompts = append(prompts, prompt)
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"get", "process-instance",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 1000, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Len(t, prompts, 1)
		require.Contains(t, prompts[0], "More matching process instances remain")
		require.Contains(t, output, "page size: 1000, current page: 2, total so far: 2, more matches: yes, next step: prompt")
		require.Contains(t, output, "page size: 1000, current page: 1, total so far: 3, more matches: no, next step: complete")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.Contains(t, output, "125")
	})

	t.Run("batch-size override and auto-confirm fetch every page without prompt", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--auto-confirm",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
		require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
	})

	t.Run("short n controls per-page batch size", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"get", "process-instance",
			"--state", "active",
			"-n", "4",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.EqualValues(t, 4, pages[0]["limit"])
		require.Contains(t, output, "page size: 4, current page: 1, total so far: 1, more matches: no, next step: complete")
		require.Contains(t, output, "123")
	})

	t.Run("batch-size and limit remain independent when limit is smaller", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"126","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":6,"hasMoreTotalItems":true}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--auto-confirm",
			"get", "process-instance",
			"--state", "active",
			"--batch-size", "4",
			"--limit", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.EqualValues(t, 4, pages[0]["limit"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, "page size: 4, current page: 2, total so far: 2, more matches: yes, next step: limit-reached")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.NotContains(t, output, "125")
		require.NotContains(t, output, "126")
	})

	t.Run("json mode fetches every page without prompt", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--json",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, `"outcome": "succeeded"`)
		require.Contains(t, output, `"total": 3`)
		require.Contains(t, output, `"key": "123"`)
		require.Contains(t, output, `"key": "124"`)
		require.Contains(t, output, `"key": "125"`)
	})

	t.Run("automation mode fetches every page without prompt", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--automation",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
		require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.Contains(t, output, "125")
	})

	t.Run("automation json mode keeps stdout machine-readable", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--automation",
			"--json",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.Zero(t, promptCalls)
		require.Contains(t, stdout, `"outcome": "succeeded"`)
		require.Contains(t, stdout, `"total": 3`)
		require.NotContains(t, stdout, "page size:")
		require.Empty(t, stderr)
	})

	t.Run("automation json mode keeps stdout machine-readable even with debug logs", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--debug",
			"--automation",
			"--json",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.Zero(t, promptCalls)
		require.Contains(t, stdout, `"outcome": "succeeded"`)
		require.Contains(t, stdout, `"total": 3`)
		require.NotContains(t, stdout, "DEBUG")
		require.NotContains(t, stdout, "config loaded")
		require.NotEmpty(t, stderr)
		require.Contains(t, stderr, "DEBUG")
	})

	t.Run("declined continuation reports partial completion summary", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			return localPreconditionError(ErrCmdAborted)
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: prompt")
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: partial-complete")
		require.Contains(t, output, "detail: stopped after 2 processed process instance(s); remaining matches were left untouched")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
	})

	t.Run("indeterminate overflow stops with warning summary", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: unknown, next step: warning-stop")
		require.Contains(t, output, "warning: stopped after 2 processed process instance(s) because more matching process instances may remain")
	})

	t.Run("v87 fallback keeps final filtered results even when the request stays broad", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "/v1/process-instances/search", r.URL.Path)

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"key":123,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant","parentKey":456,"incident":true},{"key":124,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant","incident":false}]}`))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--json",
			"get", "process-instance",
			"--children-only",
			"--incidents-only",
		)

		filter := decodeCapturedPISearchFilter(t, requests)
		require.NotContains(t, filter, "parentKey")
		require.NotContains(t, filter, "hasIncident")
		require.Contains(t, output, `"total": 1`)
		require.Contains(t, output, `"key": "123"`)
		require.NotContains(t, output, `"key": "124"`)
	})

	t.Run("orphan-child filtering stays on follow-up lookups for supported versions", func(t *testing.T) {
		for _, version := range []string{"8.8", "8.9"} {
			t.Run(version, func(t *testing.T) {
				var searchRequests []string
				var getPaths []string
				call := 0
				srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					call++
					w.Header().Set("Content-Type", "application/json")
					if call == 1 {
						require.Equal(t, http.MethodPost, r.Method)
						require.Equal(t, "/v2/process-instances/search", r.URL.Path)
						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)
						searchRequests = append(searchRequests, string(body))
						_, _ = w.Write([]byte(`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","parentProcessInstanceKey":"456","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
						return
					}
					require.Equal(t, http.MethodGet, r.Method)
					require.Equal(t, "/v2/process-instances/456", r.URL.Path)
					getPaths = append(getPaths, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"message":"not found"}`))
				}))
				t.Cleanup(srv.Close)

				cfgPath := writeTestConfigForVersion(t, srv.URL, version)

				output := executeRootForProcessInstanceTest(t,
					"--config", cfgPath,
					"--tenant", "tenant",
					"--json",
					"get", "process-instance",
					"--orphan-children-only",
				)

				filters := decodeCapturedPISearchRequests(t, searchRequests)
				require.Len(t, filters, 1)

				topLevelFilter, ok := filters[0]["filter"].(map[string]any)
				require.True(t, ok)
				require.NotContains(t, topLevelFilter, "parentProcessInstanceKey")
				require.NotContains(t, topLevelFilter, "processInstanceKey")
				require.Equal(t, []string{"/v2/process-instances/456"}, getPaths)

				require.Contains(t, output, `"total": 1`)
				require.Contains(t, output, `"key": "123"`)
			})
		}
	})

	t.Run("supported filters keep paging summaries aligned with server-filtered pages", func(t *testing.T) {
		for _, version := range []string{"8.8", "8.9"} {
			t.Run(version, func(t *testing.T) {
				var requests []string
				srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
					`{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant","parentProcessInstanceKey":"456"},{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant","parentProcessInstanceKey":"457"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
					`{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant","parentProcessInstanceKey":"458"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
				)
				t.Cleanup(srv.Close)

				cfgPath := writeTestConfigForVersion(t, srv.URL, version)
				prompts := []string{}
				prevConfirm := confirmCmdOrAbortFn
				confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
					prompts = append(prompts, prompt)
					return nil
				}
				t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

				output := executeRootForProcessInstanceTest(t,
					"--config", cfgPath,
					"--tenant", "tenant",
					"--verbose",
					"get", "process-instance",
					"--children-only",
					"--incidents-only",
					"--batch-size", "2",
				)

				pages := decodeCapturedPISearchPages(t, requests)
				decoded := decodeCapturedPISearchRequests(t, requests)
				require.Len(t, pages, 2)
				require.Len(t, decoded, 2)
				require.EqualValues(t, 2, pages[0]["limit"])
				require.EqualValues(t, 0, pages[0]["from"])
				require.EqualValues(t, 2, pages[1]["from"])
				filter, ok := decoded[0]["filter"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, true, filter["hasIncident"])

				parentFilter, ok := filter["parentProcessInstanceKey"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, true, parentFilter["$exists"])

				require.Len(t, prompts, 1)
				require.Contains(t, prompts[0], "Fetched 2 process instance(s) on this page (2 total so far)")
				require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: prompt")
				require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
			})
		}
	})
}

// TestPIContinuationHelpers verifies paging progress summary and continuation decisions.
func TestPIContinuationHelpers(t *testing.T) {
	t.Run("auto-confirm chooses auto-continue for overflow", func(t *testing.T) {
		page := process.ProcessInstancePage{
			Request:       process.ProcessInstancePageRequest{Size: 50},
			OverflowState: process.ProcessInstanceOverflowStateHasMore,
			Items:         []process.ProcessInstance{{Key: "1"}, {Key: "2"}},
		}

		summary := newPIProgressSummary(page, 2, true)

		require.Equal(t, processInstanceContinuationAutoContinue, summary.ContinuationState)
		require.Equal(t, 50, int(summary.PageSize))
		require.Equal(t, 2, summary.CurrentPageCount)
		require.Equal(t, 2, summary.CumulativeCount)
	})

	t.Run("indeterminate overflow stops with warning", func(t *testing.T) {
		page := process.ProcessInstancePage{
			Request:       process.ProcessInstancePageRequest{Size: 25},
			OverflowState: process.ProcessInstanceOverflowStateIndeterminate,
		}

		summary := newPIProgressSummary(page, 0, false)

		require.Equal(t, processInstanceContinuationWarningStop, summary.ContinuationState)
	})
}

// decodeSingleRequestJSON decodes the single captured request body for request-shape assertions.
func decodeSingleRequestJSON(t *testing.T, requests []string) map[string]any {
	t.Helper()

	require.Len(t, requests, 1)

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(requests[0]), &got))
	return got
}

// requireProcessInstanceIncidentJSONPayload unwraps the shared JSON envelope used by incident-enriched keyed lookups.
func requireProcessInstanceIncidentJSONPayload(t *testing.T, output string) map[string]any {
	t.Helper()

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "get process-instance", envelope["command"])
	return requireJSONObject(t, envelope["payload"])
}

func requireJSONObject(t *testing.T, value any) map[string]any {
	t.Helper()

	got, ok := value.(map[string]any)
	require.True(t, ok, "expected JSON object")
	return got
}

func requireJSONItems(t *testing.T, value any, wantLen int) []any {
	t.Helper()

	items, ok := value.([]any)
	require.True(t, ok, "expected JSON array")
	require.Len(t, items, wantLen)
	return items
}

// newIPv4Server creates an IPv4-only test server for command tests that must avoid IPv6 listeners.
func newIPv4Server(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	return testx.NewIPv4Server(t, handler)
}

// executeRootForProcessInstanceTest runs the root command with process-instance globals reset.
func executeRootForProcessInstanceTest(t *testing.T, args ...string) string {
	t.Helper()

	prevConfirm := confirmCmdOrAbortFn
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm
	t.Cleanup(resetProcessInstanceCommandGlobals)

	root := Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm

	_, err := root.ExecuteC()
	require.NoError(t, err)

	return buf.String()
}

// executeRootForProcessInstanceWithSeparateOutputs runs the root command and returns stdout and stderr independently.
func executeRootForProcessInstanceWithSeparateOutputs(t *testing.T, args ...string) (string, string) {
	t.Helper()

	prevConfirm := confirmCmdOrAbortFn
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm
	t.Cleanup(resetProcessInstanceCommandGlobals)

	root := Root()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm

	_, err := root.ExecuteC()
	require.NoError(t, err)

	return stdout.String(), stderr.String()
}

// executeRootForProcessInstanceTestWithEnv runs the root command with temporary environment overrides.
func executeRootForProcessInstanceTestWithEnv(t *testing.T, env []string, args ...string) string {
	t.Helper()

	prevNow := relativeDayNow
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	for _, kv := range env {
		key, value, ok := strings.Cut(kv, "=")
		require.True(t, ok)
		prevValue, hadValue := os.LookupEnv(key)
		require.NoError(t, os.Setenv(key, value))
		t.Cleanup(func() {
			if hadValue {
				require.NoError(t, os.Setenv(key, prevValue))
				return
			}
			require.NoError(t, os.Unsetenv(key))
		})
	}
	applyRelativeDayNowOverrideFromEnv(t)

	return executeRootForProcessInstanceTest(t, args...)
}

// resetProcessInstanceCommandGlobals restores process-instance command globals between tests.
func resetProcessInstanceCommandGlobals() {
	flagCancelPIKeys = nil
	flagDeletePIKeys = nil
	flagDeletePDKeys = nil
	flagDeletePDBpmnProcessId = ""
	flagDeletePDProcessVersion = 0
	flagDeletePDProcessVersionTag = ""
	flagDeletePDLatest = false
	flagGetPIKeys = nil
	flagGetPIHasUserTasks = nil
	flagRunPIProcessDefinitionBpmnProcessIds = nil
	flagRunPIProcessDefinitionKey = nil
	flagRunPIProcessDefinitionVersion = 0
	flagRunPICount = 1
	flagRunPIVars = ""
	flagGetPIBpmnProcessID = ""
	flagGetPIProcessVersion = 0
	flagGetPIProcessVersionTag = ""
	flagGetPIProcessDefinitionKey = ""
	flagGetPIStartDateAfter = ""
	flagGetPIStartDateBefore = ""
	flagGetPIEndDateAfter = ""
	flagGetPIEndDateBefore = ""
	flagGetPIStartAfterDays = -1
	flagGetPIStartBeforeDays = -1
	flagGetPIEndAfterDays = -1
	flagGetPIEndBeforeDays = -1
	flagGetPITotal = false
	flagGetPIState = "all"
	flagGetPIParentKey = ""
	flagGetPISize = consts.MaxPISearchSize
	flagGetPILimit = 0
	flagGetPIWithIncidents = false
	flagGetPIIncidentMessageLimit = 0
	flagGetPIRootsOnly = false
	flagGetPIChildrenOnly = false
	flagGetPIOrphanChildrenOnly = false
	flagGetPIIncidentsOnly = false
	flagGetPINoIncidentsOnly = false
	flagWalkPIKey = ""
	flagWalkPIModeParent = false
	flagWalkPIModeChildren = false
	flagWalkPIFlat = false
	flagWalkPIWithIncidents = false
	flagCmdAutoConfirm = false
	flagVerbose = false
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagNoWait = false
	flagForce = false
	flagNoStateCheck = false
	flagDryRun = false
	flagWorkers = 0
	flagNoWorkerLimit = false
	flagFailFast = false
	flagExpectPIKeys = nil
	flagExpectPIStates = nil
	flagExpectPIIncident = ""
	confirmCmdOrAbortFn = confirmCmdOrAbort
}

// resetPISearchBatchSizeFlag restores the process-instance batch-size flag default.
func resetPISearchBatchSizeFlag(t *testing.T, cmd *cobra.Command) {
	t.Helper()

	flag := cmd.Flags().Lookup("batch-size")
	require.NotNil(t, flag)
	require.NoError(t, flag.Value.Set("1000"))
	flag.Changed = false
}

// resetRootPersistentFlags clears root persistent flag globals that can leak across command tests.
func resetRootPersistentFlags(t *testing.T, root *cobra.Command) {
	t.Helper()

	root.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		require.NoError(t, flag.Value.Set(flag.DefValue))
		flag.Changed = false
	})
}

// executeProcessInstanceFailureHelper runs a helper subprocess expected to fail and returns output with exit code.
func executeProcessInstanceFailureHelper(t *testing.T, helperName string, cfgPath string) (string, int) {
	t.Helper()

	return executeProcessInstanceFailureHelperWithEnv(t, helperName, cfgPath, nil)
}

// executeProcessInstanceFailureHelperWithEnv runs a failing helper subprocess with extra environment for scenario selection.
func executeProcessInstanceFailureHelperWithEnv(t *testing.T, helperName string, cfgPath string, extraEnv map[string]string) (string, int) {
	t.Helper()

	env := map[string]string{
		"C8VOLT_TEST_CONFIG":  cfgPath,
		testRelativeDayNowEnv: cancelDeleteRelativeDayNow,
	}
	for k, v := range extraEnv {
		env[k] = v
	}
	var output []byte
	var err error
	if extraEnv["C8VOLT_TEST_HAS_USER_TASKS_CONFLICT"] == "stdin" {
		output, err = testx.RunCmdSubprocessWithStdin(t, helperName, env, "2251799813711967\n")
	} else {
		output, err = testx.RunCmdSubprocess(t, helperName, env)
	}
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	return string(output), exitErr.ExitCode()
}

// TestGetProcessInstanceCommand_RejectsHasUserTasksConflictHelper drives conflict cases that must exercise real Execute exit behavior.
func TestGetProcessInstanceCommand_RejectsHasUserTasksConflictHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })

	args := []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--has-user-tasks", "2251799815391233"}
	switch os.Getenv("C8VOLT_TEST_HAS_USER_TASKS_CONFLICT") {
	case "key":
		args = append(args, "--key", "2251799813711967")
	case "stdin":
		args = append(args, "-")
	case "state":
		args = append(args, "--state", "active")
	case "bpmn-process-id":
		args = append(args, "--bpmn-process-id", "C88_SimpleUserTask_Process")
	case "start-date-after":
		args = append(args, "--start-date-after", "2026-01-01")
	case "roots-only":
		args = append(args, "--roots-only")
	case "total":
		args = append(args, "--total")
	case "limit":
		args = append(args, "--limit", "1")
	default:
		t.Fatalf("unknown has-user-tasks conflict mode %q", os.Getenv("C8VOLT_TEST_HAS_USER_TASKS_CONFLICT"))
	}
	os.Args = args

	Execute()
}

// TestGetProcessInstanceCommand_HasUserTasksUnsupportedOnV87Helper drives the unsupported-version path in a helper process.
func TestGetProcessInstanceCommand_HasUserTasksUnsupportedOnV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--has-user-tasks", "2251799815391233"}

	Execute()
}

// TestGetProcessInstanceCommand_HasUserTasksResolvedProcessInstanceNotFoundHelper preserves process exit behavior for resolved-key not-found.
func TestGetProcessInstanceCommand_HasUserTasksResolvedProcessInstanceNotFoundHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--has-user-tasks", "2251799815391233"}

	Execute()
}

// TestGetProcessInstanceCommand_HasUserTasksLookupFailureHelper drives invalid and missing task-key lookups in a helper process.
func TestGetProcessInstanceCommand_HasUserTasksLookupFailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	taskKey := os.Getenv("C8VOLT_TEST_HAS_USER_TASKS_KEY")
	if taskKey == "" {
		taskKey = "2251799815391233"
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--has-user-tasks", taskKey}

	Execute()
}

// TestGetProcessInstanceCommand_RejectsRemovedCountFlagHelper is the helper-process entrypoint for removed --count validation.
func TestGetProcessInstanceCommand_RejectsRemovedCountFlagHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--count", "2"}

	Execute()
}

// TestGetProcessInstanceCommand_RejectsInvalidLimitHelper is the helper-process entrypoint for invalid --limit validation.
func TestGetProcessInstanceCommand_RejectsInvalidLimitHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--limit", "0"}

	Execute()
}

// TestGetProcessInstanceCommand_RejectsLimitWithKeyHelper is the helper-process entrypoint for --limit with --key validation.
func TestGetProcessInstanceCommand_RejectsLimitWithKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--limit", "1"}

	Execute()
}

// TestGetProcessInstanceCommand_RejectsLimitWithTotalHelper is the helper-process entrypoint for --limit with --total validation.
func TestGetProcessInstanceCommand_RejectsLimitWithTotalHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--total", "--limit", "10"}

	Execute()
}

// TestGetProcessInstanceCommand_RejectsInvalidBatchSizeHelper is the helper-process entrypoint for invalid --batch-size validation.
func TestGetProcessInstanceCommand_RejectsInvalidBatchSizeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--batch-size", "0"}

	Execute()
}

// Helper-process entrypoint for negative relative-day validation.
func TestGetProcessInstanceNegativeRelativeDayHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-older-days", "-2"}

	Execute()
}

// Helper-process entrypoint for mixed absolute-plus-relative start-date validation.
func TestGetProcessInstanceMixedAbsoluteAndRelativeDateFiltersHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-after", "2026-04-03", "--start-date-newer-days", "7"}

	Execute()
}

// Helper-process entrypoint for invalid relative-day range validation.
func TestGetProcessInstanceInvalidRelativeDayRangeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-older-days", "30", "--start-date-newer-days", "7"}

	Execute()
}

// Helper-process entrypoint for --total with --key validation.
func TestGetProcessInstanceTotalWithKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--total"}

	Execute()
}

// Helper-process entrypoint for --total with --json validation.
func TestGetProcessInstanceTotalWithJSONHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--json", "--total"}

	Execute()
}

// Helper-process entrypoint for --total with --keys-only validation.
func TestGetProcessInstanceTotalWithKeysOnlyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--keys-only", "--total"}

	Execute()
}

// Helper-process entrypoint for --with-incidents without --key validation.
func TestGetProcessInstanceWithIncidentsWithoutKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--with-incidents"}

	Execute()
}

// Helper-process entrypoint for --with-incidents with search-mode filter validation.
func TestGetProcessInstanceWithIncidentsWithSearchFilterHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--with-incidents", "--incidents-only"}

	Execute()
}

// Helper-process entrypoint for --incident-message-limit without --with-incidents validation.
func TestGetProcessInstanceIncidentMessageLimitWithoutIncidentsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--incident-message-limit", "80"}

	Execute()
}

// Helper-process entrypoint for negative --incident-message-limit validation.
func TestGetProcessInstanceIncidentMessageLimitNegativeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--with-incidents", "--incident-message-limit", "-1"}

	Execute()
}

// Helper-process entrypoint for unsupported v8.7 --with-incidents coverage.
func TestGetProcessInstanceWithIncidentsUnsupportedV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--with-incidents"}

	Execute()
}

// Helper-process entrypoint for invalid date format validation.
func TestGetProcessInstanceInvalidDateFormatHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-after", "2026-02-30"}

	Execute()
}

// Helper-process entrypoint for invalid start-date range validation.
func TestGetProcessInstanceInvalidStartDateRangeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-after", "2026-02-01", "--start-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for key-and-date-filter exclusivity validation.
func TestGetProcessInstanceDateFiltersWithKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "2251799813711967", "--start-date-after", "2026-01-01"}

	Execute()
}
