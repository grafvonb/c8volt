// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/stretchr/testify/require"
)

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
		confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
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

	t.Run("incident detail filters count matching direct incidents", func(t *testing.T) {
		tests := []struct {
			name string
			args []string
		}{
			{name: "direct incidents only", args: []string{"--total", "--direct-incidents-only", "--incident-error-type", "io_mapping_error", "--incident-error-message", "intentional"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
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
					case "/v2/process-instances/123/incidents/search":
						require.Equal(t, http.MethodPost, r.Method)
						_, _ = w.Write([]byte(`{"items":[{"errorMessage":"Intentional mapping failure","errorType":"IO_MAPPING_ERROR","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
					case "/v2/process-instances/124/incidents/search":
						require.Equal(t, http.MethodPost, r.Method)
						_, _ = w.Write([]byte(`{"items":[{"errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"incident-124","processInstanceKey":"124","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
					default:
						t.Fatalf("unexpected request path: %s", r.URL.Path)
					}
				}))
				t.Cleanup(srv.Close)

				cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
				args := append([]string{"--config", cfgPath, "--tenant", "tenant", "get", "process-instance"}, tt.args...)
				stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, args...)

				require.ElementsMatch(t, []string{
					"POST /v2/process-instances/search",
					"POST /v2/process-instances/123/incidents/search",
					"POST /v2/process-instances/124/incidents/search",
				}, requests)
				require.Equal(t, "1\n", stdout)
				require.Empty(t, stderr)
			})
		}
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
		confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
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
		require.Contains(t, stderr, "INFO page size: 2, current page: 0, total so far: 3, more matches: no, next step: complete")
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
		require.Contains(t, stderr, `DEBUG pi total page; mode offset, from 0, after "", limit 2, items 2, total before 0, total after 2`)
		require.Contains(t, stderr, `reported total 10000, reported kind lower_bound, end cursor "cursor-1"`)
		require.Contains(t, stderr, `DEBUG pi total page; mode cursor, from 2, after "cursor-1", limit 2, items 1, total before 2, total after 3`)
		require.Contains(t, stderr, `DEBUG pi total page; mode cursor, from 3, after "cursor-2", limit 2, items 0, total before 3, total after 3`)
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

// TestGetProcessInstanceSearchMachineOutputStaysProgressFree verifies paged search can later gain shared progress without corrupting JSON or key streams.
func TestGetProcessInstanceSearchMachineOutputStaysProgressFree(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":true,"endCursor":"cursor-1"}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:01:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":false,"startCursor":"cursor-1"}}`,
		)
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--batch-size", "1",
		)

		require.Len(t, requests, 2)
		require.Empty(t, stderr)
		require.NotContains(t, stdout, "scope:")
		require.NotContains(t, stdout, "page size:")
		require.NotContains(t, stdout, "discovering process instances")
		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
		payload := requireJSONObject(t, envelope["payload"])
		items := payload["items"].([]any)
		require.Len(t, items, 2)
	})

	t.Run("keys only", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:01:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--keys-only",
			"get", "process-instance",
		)

		require.Len(t, requests, 1)
		require.Empty(t, stderr)
		require.Equal(t, "123\n124\n", stdout)
		require.NotContains(t, stdout, "scope:")
		require.NotContains(t, stdout, "page size:")
	})
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
		confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
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
		require.Contains(t, output, "process-instance search scope: matched at least 5 process instances; page size: 2; discovery pages: at least 3")
		require.Contains(t, output, "discovering process instances, page 2/~3, 3 seen, user-limited")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.Contains(t, output, "125")
		require.NotContains(t, output, "126 tenant demo")
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
		confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
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
		require.Contains(t, prompts[0], "Fetched 2 process instance(s) on this page (2/3+ loaded)")
		require.Contains(t, output, "process-instance search scope: matched at least 3 process instances; page size: 1000; discovery pages: at least 1")
		require.Contains(t, output, "discovering process instances, page 1/~1, 2 seen")
		require.Contains(t, output, "discovering process instances, page 2/2, 3 seen")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.Contains(t, output, "125")
	})

	t.Run("does not prompt again when exact cursor page reaches the reported total", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false,"endCursor":"cursor-1"}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false,"endCursor":"cursor-final"}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
		prompts := []string{}
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
			prompts = append(prompts, prompt)
			return nil
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
		require.Len(t, pages, 2)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.Equal(t, "cursor-1", pages[1]["after"])
		require.Len(t, prompts, 1)
		require.Contains(t, prompts[0], "Fetched 2 process instance(s) on this page (2/3 loaded). More matching process instances remain")
		require.NotContains(t, prompts[0], "3/3 loaded")
		require.Contains(t, output, "discovering process instances, page 2/2, 3 seen")
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
		confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
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
		require.Contains(t, output, "process-instance search scope: matched at least 3 process instances; page size: 2; discovery pages: at least 2")
		require.Contains(t, output, "discovering process instances, page 1/~2, 2 seen")
		require.Contains(t, output, "discovering process instances, page 2/2, 3 seen")
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
		require.Contains(t, output, "process-instance search scope: matched 1 process instance; page size: 4; discovery pages: 1")
		require.Contains(t, output, "discovering process instances, page 1/1, 1 seen")
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
		confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
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
		require.Contains(t, output, "process-instance search scope: matched at least 6 process instances; page size: 4; discovery pages: at least 2")
		require.Contains(t, output, "discovering process instances, page 1/~2, 2 seen, user-limited")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.NotContains(t, output, "125 tenant demo")
		require.NotContains(t, output, "126 tenant demo")
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
		confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
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
		confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
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
		require.NotContains(t, output, "scope:")
		require.NotContains(t, output, "discovering process instances")
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
		confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
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
		require.Equal(t, 2, strings.Count(stderr, "api #"))
		require.NotContains(t, stdout, "api #")
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
		confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
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
		confirmCmdOrAbortFn = func(writer io.Writer, autoConfirm bool, prompt string) error {
			_, err := writer.Write([]byte("process-instance paging prompt writer\n"))
			require.NoError(t, err)
			require.False(t, autoConfirm)
			require.Contains(t, prompt, "More matching process instances remain")
			return localPreconditionError(ErrCmdAborted)
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Contains(t, stderr, "process-instance paging prompt writer")
		require.NotContains(t, stdout, "process-instance paging prompt writer")
		output := stdout + stderr
		require.Contains(t, output, "process-instance search scope: matched at least 3 process instances; page size: 2; discovery pages: at least 2")
		require.Contains(t, output, "discovering process instances, page 1/~2, 2 seen")
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
		require.Contains(t, output, "process-instance search scope: matched an unknown number of process instances; page size: 2")
		require.Contains(t, output, "discovering process instances, page 1, 2 seen")
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
				require.Contains(t, topLevelFilter, "parentProcessInstanceKey")
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
				confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
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
				require.Contains(t, prompts[0], "Fetched 2 process instance(s) on this page (2 loaded)")
				require.Contains(t, output, "process-instance search scope: matched an unknown number of process instances; page size: 2")
				require.Contains(t, output, "discovering process instances, page 1, 2 seen")
				require.Contains(t, output, "discovering process instances, page 2, 3 seen")
			})
		}
	})
}

// TestPIContinuationProgress protects the translation from backend overflow
// metadata to the prompt/auto-continue/warning states shown in verbose output.
func TestPIContinuationProgress(t *testing.T) {
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
