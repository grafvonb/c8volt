// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestDeleteProcessInstanceDryRun_SearchPagesAggregateStructuredOutput verifies
// paged delete dry-runs produce one aggregate JSON summary.
func TestDeleteProcessInstanceDryRun_SearchPagesAggregateStructuredOutput(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true
	flagViewAsJson = true
	flagGetPISize = 2

	cmd := &cobra.Command{Use: "process-instance"}
	setContractSupport(cmd, ContractSupportFull)
	cmd.Flags().Int32("batch-size", 1000, "")
	require.NoError(t, cmd.Flags().Set("batch-size", "2"))
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during delete dry-run search")
		return nil
	}

	var planned []typex.Keys
	var searchedFrom []int32
	cli := stubProcessAPI{
		searchProcessInstancesPage: func(_ context.Context, _ process.ProcessInstanceFilter, req process.ProcessInstancePageRequest, _ ...options.FacadeOption) (process.ProcessInstancePage, error) {
			searchedFrom = append(searchedFrom, req.From)
			require.EqualValues(t, 2, req.Size)
			switch req.From {
			case 0:
				return process.ProcessInstancePage{
					Request:       req,
					OverflowState: process.ProcessInstanceOverflowStateHasMore,
					Items: []process.ProcessInstance{
						{Key: "401", State: process.StateCompleted},
						{Key: "402", State: process.StateCompleted},
					},
				}, nil
			case 2:
				return process.ProcessInstancePage{
					Request:       req,
					OverflowState: process.ProcessInstanceOverflowStateNoMore,
					Items: []process.ProcessInstance{
						{Key: "403", State: process.StateCompleted},
					},
				}, nil
			default:
				t.Fatalf("unexpected search page offset %d", req.From)
				return process.ProcessInstancePage{}, nil
			}
		},
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			planned = append(planned, append(typex.Keys(nil), keys...))
			switch strings.Join(keys, ",") {
			case "401,402":
				return process.DryRunPIKeyExpansion{
					Roots:     typex.Keys{"delete-root-a"},
					Collected: typex.Keys{"delete-root-a", "401", "402"},
					Outcome:   process.TraversalOutcomeComplete,
				}, nil
			case "403":
				return process.DryRunPIKeyExpansion{
					Roots:     typex.Keys{"delete-root-b"},
					Collected: typex.Keys{"delete-root-b"},
					Outcome:   process.TraversalOutcomeComplete,
				}, nil
			default:
				t.Fatalf("unexpected dry-run plan keys %v", keys)
				return process.DryRunPIKeyExpansion{}, nil
			}
		},
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	results, err := planDeleteProcessInstanceSearchPages(cmd, cli, nil, process.ProcessInstanceFilter{})
	require.NoError(t, err)
	require.Empty(t, results.Reports)
	require.Len(t, results.DryRunPreviews, 2)
	require.Equal(t, []int32{0, 2}, searchedFrom)
	require.Equal(t, []typex.Keys{{"401", "402"}, {"403"}}, planned)

	require.NoError(t, renderProcessInstanceDryRunSummary(cmd, newProcessInstanceDryRunSummary("delete", results.DryRunPreviews)))
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	payload, ok := envelope["payload"].(map[string]any)
	require.True(t, ok)
	previews := requireDryRunSummaryPayload(t, payload, "delete", 3, 2, 4, 2)

	firstPreview, ok := previews[0].(map[string]any)
	require.True(t, ok)
	requireDryRunPreviewStringSlice(t, firstPreview, "requestedKeys", typex.Keys{"401", "402"})
	requireDryRunPreviewStringSlice(t, firstPreview, "resolvedRoots", typex.Keys{"delete-root-a"})
	requireDryRunPreviewStringSlice(t, firstPreview, "affectedFamilyKeys", typex.Keys{"delete-root-a", "401", "402"})

	secondPreview, ok := previews[1].(map[string]any)
	require.True(t, ok)
	requireDryRunPreviewStringSlice(t, secondPreview, "requestedKeys", typex.Keys{"403"})
	requireDryRunPreviewStringSlice(t, secondPreview, "resolvedRoots", typex.Keys{"delete-root-b"})
	requireDryRunPreviewStringSlice(t, secondPreview, "affectedFamilyKeys", typex.Keys{"delete-root-b"})
}

// TestDeleteProcessInstanceDryRun_SearchTenantScopedCandidatesAndDependencies
// protects search-derived delete previews from adding unrelated tenants.
func TestDeleteProcessInstanceDryRun_SearchTenantScopedCandidatesAndDependencies(t *testing.T) {
	var requests testx.SafeSlice[string]
	var fetched testx.SafeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			require.Equal(t, tenantAdminKeysSelectedTenant, filter["tenantId"])
			if _, ok := filter["parentProcessInstanceKey"]; ok {
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
				return
			}

			require.Equal(t, "COMPLETED", filter["state"])
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"401","processDefinitionId":"tenant-a-process","processDefinitionKey":"9001","processDefinitionName":"tenant-a-process","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant-a"},{"processInstanceKey":"402","processDefinitionId":"tenant-a-process","processDefinitionKey":"9001","processDefinitionName":"tenant-a-process","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant-a"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			require.Contains(t, []string{"401", "402"}, key)
			fetched.Append(key)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"tenant-a-process","processDefinitionKey":"9001","processDefinitionName":"tenant-a-process","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant-a"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", tenantAdminKeysSelectedTenant,
		"--json",
		"delete", "process-instance",
		"--state", "completed",
		"--dry-run",
		"--batch-size", "2",
	)

	topFilters := decodeCapturedTopLevelPISearchFilters(t, requests.Snapshot())
	require.Len(t, topFilters, 1)
	require.Equal(t, tenantAdminKeysSelectedTenant, topFilters[0]["tenantId"])
	require.NotContains(t, fetched.Snapshot(), tenantAdminKeysProcessInstanceKey)
	require.Contains(t, output, `"requestedCount": 2`)
	require.Contains(t, output, `"401"`)
	require.Contains(t, output, `"402"`)
	require.NotContains(t, output, tenantAdminKeysReturnedTenant)
}

// TestDeleteProcessInstanceDryRun_SearchBatchSizeLimitUsesLimitedPage verifies
// dry-run paging honors the requested limit before planning.
func TestDeleteProcessInstanceDryRun_SearchBatchSizeLimitUsesLimitedPage(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true
	flagGetPISize = 4
	flagGetPILimit = 2

	cmd := &cobra.Command{}
	cmd.Flags().Int32("batch-size", 1000, "")
	require.NoError(t, cmd.Flags().Set("batch-size", "4"))
	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during delete dry-run limited search")
		return nil
	}

	var planned typex.Keys
	var searchRequests []process.ProcessInstancePageRequest
	cli := stubProcessAPI{
		searchProcessInstancesPage: func(_ context.Context, _ process.ProcessInstanceFilter, req process.ProcessInstancePageRequest, _ ...options.FacadeOption) (process.ProcessInstancePage, error) {
			searchRequests = append(searchRequests, req)
			require.EqualValues(t, 4, req.Size)
			return process.ProcessInstancePage{
				Request:       req,
				OverflowState: process.ProcessInstanceOverflowStateHasMore,
				Items: []process.ProcessInstance{
					{Key: "501", State: process.StateCompleted},
					{Key: "502", State: process.StateCompleted},
					{Key: "503", State: process.StateCompleted},
					{Key: "504", State: process.StateCompleted},
				},
			}, nil
		},
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			planned = append(typex.Keys(nil), keys...)
			return process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"delete-root-limit"},
				Collected: typex.Keys{"delete-root-limit", "501", "502"},
				Outcome:   process.TraversalOutcomeComplete,
			}, nil
		},
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	results, err := planDeleteProcessInstanceSearchPages(cmd, cli, nil, process.ProcessInstanceFilter{})
	require.NoError(t, err)
	require.Len(t, searchRequests, 1)
	require.EqualValues(t, 0, searchRequests[0].From)
	require.Equal(t, typex.Keys{"501", "502"}, planned)
	require.Empty(t, results.Reports)
	require.Len(t, results.DryRunPreviews, 1)
	require.Equal(t, 2, results.DryRunPreviews[0].RequestedCount)
}

// Verifies search-mode deletion builds the expected date-filtered search request and no-ops cleanly on empty matches.
func TestDeleteProcessInstanceSearchScaffold_UsesTempConfigAndCapturesSearchRequest(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceSearchScaffoldHelper", cfgPath)

	filter := decodeCapturedPISearchFilter(t, requests)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Equal(t, "COMPLETED", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-01-31T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

func TestDeleteProcessInstanceBpmnSelectorMissingFailsBeforeSearch(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			writeEmptyProcessDefinitionSearchResponse(w)
		case "/v2/process-instances/search":
			t.Fatal("unexpected process-instance search before selector validation")
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceBpmnSelectorMissingFailsBeforeSearchHelper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Equal(t, []string{"POST /v2/process-definitions/search"}, requests)
	require.Contains(t, output, "no visible process definition matches the provided selector")
	require.Contains(t, output, "[missing-process]")
	require.NotContains(t, output, "found: 0")
}

func TestDeleteProcessInstanceBpmnSelectorMissingMachineModesDoNotPrompt(t *testing.T) {
	tests := []struct {
		name   string
		helper string
	}{
		{name: "json", helper: "TestDeleteProcessInstanceBpmnSelectorMissingJSONHelper"},
		{name: "automation", helper: "TestDeleteProcessInstanceBpmnSelectorMissingAutomationHelper"},
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
				case "/v2/process-instances/search":
					t.Fatal("unexpected process-instance search before selector validation")
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output, code := executeDeleteProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.Error, code)
			require.Equal(t, []string{"POST /v2/process-definitions/search"}, requests)
			require.Contains(t, output, "no visible process definition matches the provided selector")
			require.NotContains(t, output, "List visible process definitions?")
		})
	}
}

func TestDeleteProcessInstanceBpmnSelectorVisiblePreservesSearchNoOp(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			writeVisibleProcessDefinitionSearchResponse(w)
		case "/v2/process-instances/search":
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"delete", "process-instance",
		"--state", "completed",
		"--bpmn-process-id", "order-process",
	)

	require.Equal(t, []string{"POST /v2/process-definitions/search", "POST /v2/process-instances/search"}, requests)
	require.Equal(t, "found: 0\n", output)
}

// Verifies reversed date ranges are rejected when the after-bound is later than the before-bound.
func TestDeleteProcessInstanceCommand_RejectsInvalidDateFilter(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsInvalidDateFilterHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, `invalid range for --end-date-after and --end-date-before: "2026-02-01" is later than "2026-01-31"`)
}

// Verifies invalid date literals for date flags are rejected with a clear YYYY-MM-DD validation error.
func TestDeleteProcessInstanceCommand_RejectsInvalidDateValue(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsInvalidDateValueHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, `invalid value for --start-date-after: "2026-02-30", expected YYYY-MM-DD`)
}

// Verifies process-instance date filters are rejected for Camunda 8.7 where the capability is unsupported.
func TestDeleteProcessInstanceCommand_RejectsDateFiltersOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsDateFiltersOnV87Helper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "process-instance date filters require Camunda 8.8")
}

// Verifies relative-day process-instance filters are also rejected for Camunda 8.7.
func TestDeleteProcessInstanceCommand_RejectsRelativeDayFiltersOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsRelativeDayFiltersOnV87Helper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "process-instance date filters require Camunda 8.8")
}

// Verifies date-filtered search selection deletes matched instances and preserves descendant lookup behavior.
func TestDeleteProcessInstanceCommand_SearchSelectionUsesDateFiltersAndDeletesMatches(t *testing.T) {
	var requests []string
	var deleted testx.SafeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
			writeVisibleProcessDefinitionSearchResponse(w)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			var searchBody map[string]any
			require.NoError(t, json.Unmarshal(body, &searchBody))

			filter, _ := searchBody["filter"].(map[string]any)
			parentKey, _ := filter["parentProcessInstanceKey"].(string)

			w.Header().Set("Content-Type", "application/json")
			if parentKey == "2251799813711967" {
				_, _ = w.Write([]byte(`{"items":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-01-03T18:00:00Z","endDate":"2026-01-12T08:30:00Z","state":"COMPLETED","tenantId":"tenant"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813711967":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-01-03T18:00:00Z","endDate":"2026-01-12T08:30:00Z","state":"COMPLETED","tenantId":"tenant"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/process-instances/2251799813711967":
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceCommand_SearchSelectionUsesDateFiltersAndDeletesMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(requests), 2)
	require.Equal(t, []string{"/v1/process-instances/2251799813711967"}, deleted.Snapshot())
	filter := decodeCapturedPISearchFilter(t, requests[:1])

	require.Equal(t, "COMPLETED", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-01-31T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")

	descendantSearch := decodeCapturedPISearchRequest(t, requests[len(requests)-1])
	descFilter := descendantSearch["filter"].(map[string]any)
	require.Equal(t, "2251799813711967", descFilter["parentProcessInstanceKey"])
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

// Verifies relative-day search selection derives canonical end-date bounds before deleting matches.
func TestDeleteProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndDeletesMatches(t *testing.T) {
	var requests []string
	var deleted testx.SafeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
			writeVisibleProcessDefinitionSearchResponse(w)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			var searchBody map[string]any
			require.NoError(t, json.Unmarshal(body, &searchBody))

			filter, _ := searchBody["filter"].(map[string]any)
			parentKey, _ := filter["parentProcessInstanceKey"].(string)

			w.Header().Set("Content-Type", "application/json")
			if parentKey == "2251799813711967" {
				_, _ = w.Write([]byte(`{"items":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-02-10T18:00:00Z","endDate":"2026-03-15T08:30:00Z","state":"COMPLETED","tenantId":"tenant"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813711967":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-02-10T18:00:00Z","endDate":"2026-03-15T08:30:00Z","state":"COMPLETED","tenantId":"tenant"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/process-instances/2251799813711967":
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndDeletesMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(requests), 2)
	require.Equal(t, []string{"/v1/process-instances/2251799813711967"}, deleted.Snapshot())
	filter := decodeCapturedPISearchFilter(t, requests[:1])

	require.Equal(t, "COMPLETED", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-02-09T00:00:00Z")
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-04-03T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")

	descendantSearch := decodeCapturedPISearchRequest(t, requests[len(requests)-1])
	descFilter := descendantSearch["filter"].(map[string]any)
	require.Equal(t, "2251799813711967", descFilter["parentProcessInstanceKey"])
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

// TestDeleteProcessInstanceSearchPages_RequiresForceBeforeAnyMutation verifies
// filter-based delete plans all selected pages before submitting any deletion.
func TestDeleteProcessInstanceSearchPages_RequiresForceBeforeAnyMutation(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true
	flagVerbose = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	searchCalls := 0
	cli := stubProcessAPI{
		searchProcessInstancesPage: func(_ context.Context, _ process.ProcessInstanceFilter, req process.ProcessInstancePageRequest, _ ...options.FacadeOption) (process.ProcessInstancePage, error) {
			searchCalls++
			switch searchCalls {
			case 1:
				return process.ProcessInstancePage{
					Items:         []process.ProcessInstance{{Key: "401", State: process.StateCompleted}},
					Request:       req,
					OverflowState: process.ProcessInstanceOverflowStateHasMore,
				}, nil
			case 2:
				return process.ProcessInstancePage{
					Items:         []process.ProcessInstance{{Key: "402", State: process.StateActive}},
					Request:       req,
					OverflowState: process.ProcessInstanceOverflowStateNoMore,
				}, nil
			default:
				t.Fatalf("unexpected search page call %d", searchCalls)
				return process.ProcessInstancePage{}, nil
			}
		},
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			switch keys.String() {
			case "401":
				return process.DryRunPIKeyExpansion{
					Roots:     typex.Keys{"401"},
					Collected: typex.Keys{"401"},
					Outcome:   process.TraversalOutcomeComplete,
				}, nil
			case "402":
				return process.DryRunPIKeyExpansion{
					Roots:     typex.Keys{"402"},
					Collected: typex.Keys{"402"},
					RequiresCancelBeforeDelete: []process.ProcessInstance{
						{Key: "402", State: process.StateActive},
					},
					Outcome: process.TraversalOutcomeComplete,
				}, nil
			default:
				t.Fatalf("unexpected dry-run plan keys %v", keys)
				return process.DryRunPIKeyExpansion{}, nil
			}
		},
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	got, err := deleteProcessInstanceSearchPages(cmd, cli, nil, process.ProcessInstanceFilter{})

	require.Error(t, err)
	require.Contains(t, err.Error(), "refusing to delete process-instance scope")
	require.Equal(t, 2, searchCalls)
	require.Empty(t, got.Reports)
	require.Len(t, got.DryRunPreviews, 0)
	require.Contains(t, buf.String(), "next step: auto-continue")
}

// TestDeleteProcessInstanceSearchPages_FreezesAllPlansBeforeMutation verifies
// search-derived delete collects every page-level plan before one confirmation
// and one aggregate delete call.
func TestDeleteProcessInstanceSearchPages_FreezesAllPlansBeforeMutation(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true
	flagVerbose = true
	flagGetPISize = 1

	cmd := &cobra.Command{}
	cmd.Flags().Int32("batch-size", 1000, "")
	require.NoError(t, cmd.Flags().Set("batch-size", "1"))
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var events []string
	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(implicit bool, prompt string) error {
		events = append(events, "confirm")
		require.True(t, implicit)
		require.Contains(t, prompt, "total of 4 instance(s) with 2 root instance(s) will be deleted")
		return nil
	}

	cli := stubProcessAPI{
		searchProcessInstancesPage: func(_ context.Context, _ process.ProcessInstanceFilter, req process.ProcessInstancePageRequest, _ ...options.FacadeOption) (process.ProcessInstancePage, error) {
			events = append(events, fmt.Sprintf("search:%d", req.From))
			require.EqualValues(t, 1, req.Size)
			switch req.From {
			case 0:
				return process.ProcessInstancePage{
					Items:         []process.ProcessInstance{{Key: "401", State: process.StateCompleted}},
					Request:       req,
					OverflowState: process.ProcessInstanceOverflowStateHasMore,
				}, nil
			case 1:
				return process.ProcessInstancePage{
					Items:         []process.ProcessInstance{{Key: "402", State: process.StateCompleted}},
					Request:       req,
					OverflowState: process.ProcessInstanceOverflowStateNoMore,
				}, nil
			default:
				t.Fatalf("unexpected search page offset %d", req.From)
				return process.ProcessInstancePage{}, nil
			}
		},
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			events = append(events, "plan:"+keys.String())
			switch keys.String() {
			case "401":
				return process.DryRunPIKeyExpansion{
					Roots:     typex.Keys{"root-a"},
					Collected: typex.Keys{"root-a", "401"},
					Outcome:   process.TraversalOutcomeComplete,
				}, nil
			case "402":
				return process.DryRunPIKeyExpansion{
					Roots:     typex.Keys{"root-b"},
					Collected: typex.Keys{"root-b", "402"},
					Outcome:   process.TraversalOutcomeComplete,
				}, nil
			default:
				t.Fatalf("unexpected dry-run plan keys %v", keys)
				return process.DryRunPIKeyExpansion{}, nil
			}
		},
		deleteProcessInstances: func(_ context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.DeleteReports, error) {
			events = append(events, "delete")
			require.Equal(t, typex.Keys{"root-a", "root-b"}, keys)
			require.Zero(t, wantedWorkers)
			cfg := options.ApplyFacadeOptions(opts)
			require.Equal(t, 4, cfg.AffectedProcessInstanceCount)
			require.True(t, cfg.SuppressWorkflowDetailLogs)
			require.True(t, cfg.SuppressProcessInstanceDetailLogs)
			return process.DeleteReports{Items: []process.DeleteReport{
				{Key: "root-a", Ok: true},
				{Key: "root-b", Ok: true},
			}}, nil
		},
	}

	got, err := deleteProcessInstanceSearchPages(cmd, cli, nil, process.ProcessInstanceFilter{State: process.StateCompleted})

	require.NoError(t, err)
	require.Equal(t, []string{"search:0", "plan:401", "search:1", "plan:402", "confirm", "delete"}, events)
	require.Len(t, got.DryRunPreviews, 2)
	require.Len(t, got.Reports, 2)
	require.Contains(t, buf.String(), "next step: auto-continue")
}

// TestDeleteProcessInstanceDryRun_SearchContinuesAfterEmptySelectedPage protects
// sparse search-derived delete planning: an empty selected page with more
// backend matches must not hide later selectable candidates.
func TestDeleteProcessInstanceDryRun_SearchContinuesAfterEmptySelectedPage(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true
	flagGetPISize = 1

	cmd := &cobra.Command{}
	cmd.Flags().Int32("batch-size", 1000, "")
	require.NoError(t, cmd.Flags().Set("batch-size", "1"))
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	var actions []process.ProcessInstanceSearchPageAction
	cli := stubProcessAPI{
		planProcessInstanceMutationPages: func(_ context.Context, request process.ProcessInstanceMutationPlanRequest, visitor process.ProcessInstanceMutationPlanVisitor, _ ...options.FacadeOption) (process.ProcessInstanceMutationPlanPagesResult, error) {
			require.Equal(t, int32(1), request.SearchRequest.Page.Size)
			emptyPage := process.ProcessInstancePage{
				Request:       process.ProcessInstancePageRequest{From: 0, Size: 1},
				OverflowState: process.ProcessInstanceOverflowStateHasMore,
			}
			action, err := visitor(process.ProcessInstanceMutationPlanStep{
				Page:            emptyPage,
				CumulativeCount: 0,
			})
			if err != nil {
				return process.ProcessInstanceMutationPlanPagesResult{}, err
			}
			actions = append(actions, action)
			if action == process.ProcessInstanceSearchPageActionStop {
				return process.ProcessInstanceMutationPlanPagesResult{Pages: 1, Stopped: true}, nil
			}

			selectedPage := process.ProcessInstancePage{
				Items:         []process.ProcessInstance{{Key: "401", State: process.StateCompleted}},
				Request:       process.ProcessInstancePageRequest{From: 1, Size: 1},
				OverflowState: process.ProcessInstanceOverflowStateNoMore,
			}
			plan := process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"401"},
				Collected: typex.Keys{"401"},
				Outcome:   process.TraversalOutcomeComplete,
			}
			action, err = visitor(process.ProcessInstanceMutationPlanStep{
				Page:             selectedPage,
				RequestedKeys:    []string{"401"},
				Plan:             plan,
				CumulativeCount:  1,
				CumulativeImpact: 1,
			})
			if err != nil {
				return process.ProcessInstanceMutationPlanPagesResult{}, err
			}
			actions = append(actions, action)
			return process.ProcessInstanceMutationPlanPagesResult{
				Plans:            []process.ProcessInstanceMutationPlanStep{{Page: selectedPage, RequestedKeys: []string{"401"}, Plan: plan, CumulativeCount: 1, CumulativeImpact: 1}},
				Pages:            2,
				RequestedCount:   1,
				CumulativeImpact: 1,
			}, nil
		},
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	got, err := deleteProcessInstanceSearchPages(cmd, cli, nil, process.ProcessInstanceFilter{State: process.StateCompleted})

	require.NoError(t, err)
	require.Equal(t, []process.ProcessInstanceSearchPageAction{
		process.ProcessInstanceSearchPageActionContinue,
		process.ProcessInstanceSearchPageActionStop,
	}, actions)
	require.Empty(t, got.Reports)
	require.Len(t, got.DryRunPreviews, 1)
	require.Equal(t, []string{"401"}, got.DryRunPreviews[0].RequestedKeys)
	require.NotContains(t, buf.String(), "found: 0")
}

// TestDeleteProcessInstancePage_PrintsOrphanWarningForPagedImpactCheck verifies paged impact-check warnings are printed.
func TestDeleteProcessInstancePage_PrintsOrphanWarningForPagedImpactCheck(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true
	flagVerbose = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"2251799813711967"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:            typex.Keys{"2251799813711900"},
				Collected:        typex.Keys{"2251799813711900", "2251799813711967"},
				MissingAncestors: []process.MissingAncestor{{Key: "2251799813711999", StartKey: "2251799813711967"}},
				Warning:          "one or more parent process instances were not found",
				Outcome:          process.TraversalOutcomePartial,
			}, nil
		},
		deleteProcessInstances: func(_ context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.DeleteReports, error) {
			require.Equal(t, typex.Keys{"2251799813711900"}, keys)
			require.Zero(t, wantedWorkers)
			cfg := options.ApplyFacadeOptions(opts)
			require.Equal(t, 2, cfg.AffectedProcessInstanceCount)
			require.True(t, cfg.SuppressWorkflowDetailLogs)
			require.True(t, cfg.SuppressProcessInstanceDetailLogs)
			return process.DeleteReports{Items: []process.DeleteReport{{Key: "2251799813711900", Ok: true}}}, nil
		},
	}

	got, err := deleteProcessInstancePage(cmd, cli, typex.Keys{"2251799813711967"}, false)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 2, Roots: 1}, got.Impact)
	require.Len(t, got.Reports, 1)
	require.Contains(t, buf.String(), "one or more parent process instances were not found")
	require.Contains(t, buf.String(), "missing ancestor keys: 2251799813711999")
	require.Contains(t, buf.String(), "deletion: deleted 1/1 process-instance tree(s); affected process instances: 2")
}

// Verifies delete no-ops successfully when a date-filtered search returns no process instances.
func TestDeleteProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatches(t *testing.T) {
	var requests []string

	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

// Verifies a relative-day-only filter is sufficient to trigger search mode.
func TestDeleteProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficient(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficientHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.NotContains(t, output, "either at least one --key is required, or sufficient filtering options")
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

// Verifies paged delete search prompts between pages and continues when confirmations are accepted.
func TestDeleteProcessInstanceCommand_SearchPagingPromptFlow(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	searchPage := 0
	var searchMu sync.Mutex

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			searchMu.Lock()
			defer searchMu.Unlock()
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"401","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"402","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"403","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/401" || r.URL.Path == "/v1/process-instances/402" || r.URL.Path == "/v1/process-instances/403"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	var prompts []string
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
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 0, pages[0]["from"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.ElementsMatch(t, []string{
		"/v1/process-instances/401",
		"/v1/process-instances/402",
		"/v1/process-instances/403",
	}, deleted.Snapshot())
	require.Len(t, prompts, 2)
	require.Contains(t, prompts[0], "Checked delete impact for 2 process instance(s) on this page (2/3+ requested, 2 including dependencies); no changes made yet. More matching process instances remain. Continue checking?")
	require.Contains(t, prompts[1], "You are about to delete 3 process instance(s)")
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: prompt")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
	require.NotContains(t, output, "next step: auto-continue")
}

// Verifies v8.7 paged delete search fails once keyed tenant-safe impact checking reaches the unsupported direct-lookup seam.
func TestDeleteProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotals(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	searchPage := 0
	var searchMu sync.Mutex

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if parentKey, ok := filter["parentKey"]; ok && parentKey != nil {
					parent := int64(parentKey.(float64))
					if parent != 901 && parent != 902 {
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(`{"items":[]}`))
						return
					}
					childKey := "1901"
					if parent == 902 {
						childKey = "1902"
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[{"key":` + childKey + `,"parentKey":` + fmt.Sprintf("%d", parent) + `,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"}]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			searchMu.Lock()
			defer searchMu.Unlock()
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"key":901,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"},{"key":902,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"}],"total":3}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"key":901,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"},{"key":902,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"},{"key":903,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"}],"total":3}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/901" || r.URL.Path == "/v1/process-instances/902" || r.URL.Path == "/v1/process-instances/903" || r.URL.Path == "/v1/process-instances/1901" || r.URL.Path == "/v1/process-instances/1902"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")
	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotalsHelper", cfgPath)

	sizes := decodeCapturedTopLevelPISearchSizes(t, requests.Snapshot())
	require.Equal(t, exitcode.Error, code)
	require.Equal(t, []float64{2}, sizes)
	require.Empty(t, deleted.Snapshot())
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "process-instance direct lookup by key is not tenant-safe in Camunda 8.7")
}

// Verifies paged delete search auto-continues without continuation prompts when --auto-confirm is set.
func TestDeleteProcessInstanceCommand_SearchPagingAutoConfirmFlow(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	searchPage := 0

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"501","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"502","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"503","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/501" || r.URL.Path == "/v1/process-instances/502" || r.URL.Path == "/v1/process-instances/503"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
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
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/501",
		"/v1/process-instances/502",
		"/v1/process-instances/503",
	}, deleted.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
}

// TestDeleteProcessInstanceCommand_SearchPagingLimitFlow verifies delete search stops at the requested limit.
func TestDeleteProcessInstanceCommand_SearchPagingLimitFlow(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	searchPage := 0

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"501","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"502","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":5,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"503","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"504","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":5,"hasMoreTotalItems":true}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/501" || r.URL.Path == "/v1/process-instances/502" || r.URL.Path == "/v1/process-instances/503"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
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
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--batch-size", "2",
		"--limit", "3",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/501",
		"/v1/process-instances/502",
		"/v1/process-instances/503",
	}, deleted.Snapshot())
	require.NotContains(t, strings.Join(deleted.Snapshot(), "\n"), "504")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: yes, next step: limit-reached")
}

// TestDeleteProcessInstanceCommand_SearchPagingBatchSizeLimitFlow verifies batch size and limit interact correctly.
func TestDeleteProcessInstanceCommand_SearchPagingBatchSizeLimitFlow(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"701","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"702","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"703","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"704","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":6,"hasMoreTotalItems":true}}`))
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/701" || r.URL.Path == "/v1/process-instances/702"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
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
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--batch-size", "4",
		"--limit", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 1)
	require.EqualValues(t, 4, pages[0]["limit"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/701",
		"/v1/process-instances/702",
	}, deleted.Snapshot())
	require.NotContains(t, strings.Join(deleted.Snapshot(), "\n"), "703")
	require.Contains(t, output, "page size: 4, current page: 2, total so far: 2, more matches: yes, next step: limit-reached")
}

// TestDeleteProcessInstanceCommand_SearchPagingAutomationFlow verifies automation mode auto-continues paged delete searches.
func TestDeleteProcessInstanceCommand_SearchPagingAutomationFlow(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	searchPage := 0

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"601","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"602","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"603","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/601" || r.URL.Path == "/v1/process-instances/602" || r.URL.Path == "/v1/process-instances/603"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
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
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/601",
		"/v1/process-instances/602",
		"/v1/process-instances/603",
	}, deleted.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
}

// Verifies paged delete reports a partial-completion summary when continuation is aborted.
func TestDeleteProcessInstanceCommand_SearchPagingPartialCompletionSummary(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"511","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"512","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/511" || r.URL.Path == "/v1/process-instances/512"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	callCount := 0
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		callCount++
		return ErrCmdAborted
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 1)
	require.Equal(t, 1, callCount)
	require.Empty(t, deleted.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: partial-complete")
	require.Contains(t, output, "detail: stopped after 2 processed process instance(s); remaining matches were left untouched")
}

// Verifies paged delete emits a warning-stop summary when overflow state is indeterminate.
func TestDeleteProcessInstanceCommand_SearchPagingWarningStopSummary(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"521","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"522","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{}}`))
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/521" || r.URL.Path == "/v1/process-instances/522"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error { return nil }
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 1)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/521",
		"/v1/process-instances/522",
	}, deleted.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: unknown, next step: warning-stop")
	require.Contains(t, output, "warning: stopped after 2 processed process instance(s) because more matching process instances may remain")
}

// Helper-process entrypoint for the search scaffold failure test.
func TestDeleteProcessInstanceSearchScaffoldHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31", "--auto-confirm"}

	Execute()
}

func TestDeleteProcessInstanceBpmnSelectorMissingFailsBeforeSearchHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "missing-process", "--auto-confirm"}

	Execute()
}

func TestDeleteProcessInstanceBpmnSelectorMissingJSONHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--json", "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "missing-process", "--auto-confirm"}

	Execute()
}

func TestDeleteProcessInstanceBpmnSelectorMissingAutomationHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--automation", "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "missing-process"}

	Execute()
}

// TestDeleteProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotalsHelper is the helper-process entrypoint for v8.7 paging.
func TestDeleteProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotalsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant", "--verbose", "delete", "process-instance", "--state", "completed", "--no-wait", "--batch-size", "2"}

	Execute()
}

// Helper-process entrypoint for invalid date range validation.
func TestDeleteProcessInstanceCommand_RejectsInvalidDateFilterHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--end-date-after", "2026-02-01", "--end-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for invalid date format validation.
func TestDeleteProcessInstanceCommand_RejectsInvalidDateValueHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--start-date-after", "2026-02-30"}

	Execute()
}

// Helper-process entrypoint for version capability validation on Camunda 8.7.
func TestDeleteProcessInstanceCommand_RejectsDateFiltersOnV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--auto-confirm"}

	Execute()
}

// Helper-process entrypoint for relative-day version capability validation on Camunda 8.7.
func TestDeleteProcessInstanceCommand_RejectsRelativeDayFiltersOnV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--end-date-newer-days", "7", "--auto-confirm"}

	Execute()
}

// Helper-process entrypoint for the successful search-select-and-delete flow test.
func TestDeleteProcessInstanceCommand_SearchSelectionUsesDateFiltersAndDeletesMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31", "--auto-confirm", "--no-state-check", "--no-wait"}

	Execute()
}

// Helper-process entrypoint for the successful relative-day search-select-and-delete flow test.
func TestDeleteProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndDeletesMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--end-date-older-days", "7", "--end-date-newer-days", "60", "--auto-confirm", "--no-state-check", "--no-wait"}

	Execute()
}

// Helper-process entrypoint for the no-matches failure test.
func TestDeleteProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for relative-day-only sufficiency validation.
func TestDeleteProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficientHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--end-date-older-days", "72"}

	Execute()
}
