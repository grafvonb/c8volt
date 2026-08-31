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
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestCancelProcessInstanceDryRun_SearchPagesAggregateStructuredOutput verifies
// paged cancel dry-runs produce one aggregate JSON summary.
func TestCancelProcessInstanceDryRun_SearchPagesAggregateStructuredOutput(t *testing.T) {
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
		t.Fatal("unexpected confirmation prompt during cancel dry-run search")
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
						{Key: "101", State: process.StateActive},
						{Key: "102", State: process.StateActive},
					},
				}, nil
			case 2:
				return process.ProcessInstancePage{
					Request:       req,
					OverflowState: process.ProcessInstanceOverflowStateNoMore,
					Items: []process.ProcessInstance{
						{Key: "103", State: process.StateActive},
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
			case "101,102":
				return process.DryRunPIKeyExpansion{
					Roots:     typex.Keys{"root-a"},
					Collected: typex.Keys{"root-a", "101", "102"},
					Outcome:   process.TraversalOutcomeComplete,
				}, nil
			case "103":
				return process.DryRunPIKeyExpansion{
					Roots:     typex.Keys{"root-b"},
					Collected: typex.Keys{"root-b"},
					Outcome:   process.TraversalOutcomeComplete,
				}, nil
			default:
				t.Fatalf("unexpected dry-run plan keys %v", keys)
				return process.DryRunPIKeyExpansion{}, nil
			}
		},
		cancelProcessInstances: dryRunCancelMutationGuard(t),
	}

	results, err := cancelProcessInstanceSearchPages(cmd, cli, nil, process.ProcessInstanceFilter{})
	require.NoError(t, err)
	require.Empty(t, results.Reports)
	require.Len(t, results.DryRunPreviews, 2)
	require.Equal(t, []int32{0, 2}, searchedFrom)
	require.Equal(t, []typex.Keys{{"101", "102"}, {"103"}}, planned)

	require.NoError(t, renderProcessInstanceDryRunSummary(cmd, newProcessInstanceDryRunSummary("cancel", results.DryRunPreviews)))
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	payload, ok := envelope["payload"].(map[string]any)
	require.True(t, ok)
	previews := requireDryRunSummaryPayload(t, payload, "cancel", 3, 2, 4, 2)

	firstPreview, ok := previews[0].(map[string]any)
	require.True(t, ok)
	requireDryRunPreviewStringSlice(t, firstPreview, "requestedKeys", typex.Keys{"101", "102"})
	requireDryRunPreviewStringSlice(t, firstPreview, "resolvedRoots", typex.Keys{"root-a"})
	requireDryRunPreviewStringSlice(t, firstPreview, "affectedFamilyKeys", typex.Keys{"root-a", "101", "102"})

	secondPreview, ok := previews[1].(map[string]any)
	require.True(t, ok)
	requireDryRunPreviewStringSlice(t, secondPreview, "requestedKeys", typex.Keys{"103"})
	requireDryRunPreviewStringSlice(t, secondPreview, "resolvedRoots", typex.Keys{"root-b"})
	requireDryRunPreviewStringSlice(t, secondPreview, "affectedFamilyKeys", typex.Keys{"root-b"})
}

// TestCancelProcessInstanceSearchSelectedUsesSemanticCompletionActivity verifies
// search-selected cancellation starts a fresh completion activity after the
// selected mutation scope is confirmed.
func TestCancelProcessInstanceSearchSelectedUsesSemanticCompletionActivity(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true
	flagGetPISize = 1

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))
	cmd.Flags().Int32("batch-size", 1000, "")
	require.NoError(t, cmd.Flags().Set("batch-size", "1"))

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		require.True(t, autoConfirm)
		require.Contains(t, prompt, "cancel")
		return nil
	}

	cli := stubProcessAPI{
		planProcessInstanceMutationPages: func(_ context.Context, request process.ProcessInstanceMutationPlanRequest, visitor process.ProcessInstanceMutationPlanVisitor, opts ...options.FacadeOption) (process.ProcessInstanceMutationPlanPagesResult, error) {
			require.Equal(t, int32(1), request.SearchRequest.Page.Size)
			require.NotNil(t, options.ApplyFacadeOptions(opts).Progress)
			page := process.ProcessInstancePage{
				Items:         []process.ProcessInstance{{Key: "401", State: process.StateActive}},
				Request:       process.ProcessInstancePageRequest{From: 0, Size: 1},
				OverflowState: process.ProcessInstanceOverflowStateNoMore,
				ReportedTotal: &process.ProcessInstanceReportedTotal{Count: 1, Kind: process.ProcessInstanceReportedTotalKindExact},
			}
			plan := process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-401"},
				Collected: typex.Keys{"root-401", "401"},
				Outcome:   process.TraversalOutcomeComplete,
			}
			action, err := visitor(process.ProcessInstanceMutationPlanStep{
				Page:             page,
				RequestedKeys:    []string{"401"},
				Plan:             plan,
				CumulativeCount:  1,
				CumulativeImpact: 2,
			})
			require.NoError(t, err)
			require.Equal(t, process.ProcessInstanceSearchPageActionStop, action)
			return process.ProcessInstanceMutationPlanPagesResult{
				Plans:            []process.ProcessInstanceMutationPlanStep{{Page: page, RequestedKeys: []string{"401"}, Plan: plan, CumulativeCount: 1, CumulativeImpact: 2}},
				Pages:            1,
				RequestedCount:   1,
				CumulativeImpact: 2,
			}, nil
		},
		cancelProcessInstances: func(_ context.Context, keys typex.Keys, _ int, opts ...options.FacadeOption) (process.CancelReports, error) {
			reportProcessInstanceMutationCompletionForTest(t, "cancel", "root-401", 2, keys, opts...)
			return process.CancelReports{Items: []process.CancelReport{{Key: "root-401", Ok: true}}}, nil
		},
	}

	got, err := cancelProcessInstanceSearchPages(cmd, cli, nil, process.ProcessInstanceFilter{State: process.StateActive})

	require.NoError(t, err)
	require.Len(t, got.Reports, 1)
	requireProcessInstanceMutationSemanticActivity(t, sink, "cancel", "root-401")
}

// TestCancelProcessInstanceDryRun_SearchTenantScopedCandidates protects the
// search-derived dry-run path from broadening beyond tenant-scoped candidates.
func TestCancelProcessInstanceDryRun_SearchTenantScopedCandidates(t *testing.T) {
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

			require.Equal(t, "ACTIVE", filter["state"])
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"101","processDefinitionId":"tenant-a-process","processDefinitionKey":"9001","processDefinitionName":"tenant-a-process","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant-a"},{"processInstanceKey":"102","processDefinitionId":"tenant-a-process","processDefinitionKey":"9001","processDefinitionName":"tenant-a-process","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			require.Contains(t, []string{"101", "102"}, key)
			fetched.Append(key)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"tenant-a-process","processDefinitionKey":"9001","processDefinitionName":"tenant-a-process","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant-a"}`, key)))
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
		"cancel", "process-instance",
		"--state", "active",
		"--dry-run",
		"--batch-size", "2",
	)

	topFilters := decodeCapturedTopLevelPISearchFilters(t, requests.Snapshot())
	require.Len(t, topFilters, 1)
	require.Equal(t, tenantAdminKeysSelectedTenant, topFilters[0]["tenantId"])
	require.NotContains(t, fetched.Snapshot(), tenantAdminKeysProcessInstanceKey)
	require.Contains(t, output, `"requestedCount": 2`)
	require.Contains(t, output, `"101"`)
	require.Contains(t, output, `"102"`)
	require.NotContains(t, output, tenantAdminKeysReturnedTenant)
}

// TestCancelProcessInstanceDryRun_SearchTenantContextPrecedesPreview verifies
// selector-based cancel previews explain named and unfiltered discovery scope
// before the dry-run body is rendered.
func TestCancelProcessInstanceDryRun_SearchTenantContextPrecedesPreview(t *testing.T) {
	tests := []struct {
		name   string
		tenant string
		want   string
	}{
		{name: "named", tenant: "tenant-a", want: "selection scope: tenant-a only\n"},
		{name: "empty", tenant: "", want: "selection scope: unfiltered across accessible tenants\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProcessInstanceCommandGlobals()
			t.Cleanup(resetProcessInstanceCommandGlobals)
			flagDryRun = true

			cmd := &cobra.Command{Use: "process-instance"}
			buf := &bytes.Buffer{}
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			prevConfirm := confirmCmdOrAbortFn
			t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
			confirmCmdOrAbortFn = func(bool, string) error {
				t.Fatal("unexpected confirmation prompt during cancel dry-run search")
				return nil
			}

			cli := stubProcessAPI{
				planProcessInstanceMutationPages: func(_ context.Context, _ process.ProcessInstanceMutationPlanRequest, visitor process.ProcessInstanceMutationPlanVisitor, _ ...options.FacadeOption) (process.ProcessInstanceMutationPlanPagesResult, error) {
					plan := process.DryRunPIKeyExpansion{
						Roots:     typex.Keys{"101"},
						Collected: typex.Keys{"101"},
						Outcome:   process.TraversalOutcomeComplete,
					}
					action, err := visitor(process.ProcessInstanceMutationPlanStep{
						Page: process.ProcessInstancePage{
							Items:         []process.ProcessInstance{{Key: "101", State: process.StateActive}},
							OverflowState: process.ProcessInstanceOverflowStateNoMore,
						},
						RequestedKeys:    []string{"101"},
						Plan:             plan,
						CumulativeCount:  1,
						CumulativeImpact: 1,
					})
					require.NoError(t, err)
					require.Equal(t, process.ProcessInstanceSearchPageActionStop, action)
					return process.ProcessInstanceMutationPlanPagesResult{RequestedCount: 1, CumulativeImpact: 1}, nil
				},
				cancelProcessInstances: dryRunCancelMutationGuard(t),
			}

			results, err := cancelProcessInstanceSearchPages(cmd, cli, &config.Config{App: config.App{Tenant: tt.tenant}}, process.ProcessInstanceFilter{})
			require.NoError(t, err)
			require.Len(t, results.DryRunPreviews, 1)
			require.NoError(t, renderProcessInstanceDryRunSummary(cmd, newProcessInstanceDryRunSummary("cancel", results.DryRunPreviews)))

			output := buf.String()
			require.Contains(t, output, tt.want)
			require.Contains(t, output, "dry run: cancel process-instance\n")
			require.Less(t, strings.Index(output, tt.want), strings.Index(output, "dry run: cancel process-instance\n"))
		})
	}
}

// TestCancelProcessInstanceDryRun_SearchBatchSizeLimitUsesLimitedPage verifies
// dry-run paging honors the requested limit before planning.
func TestCancelProcessInstanceDryRun_SearchBatchSizeLimitUsesLimitedPage(t *testing.T) {
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
		t.Fatal("unexpected confirmation prompt during cancel dry-run limited search")
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
					{Key: "201", State: process.StateActive},
					{Key: "202", State: process.StateActive},
					{Key: "203", State: process.StateActive},
					{Key: "204", State: process.StateActive},
				},
			}, nil
		},
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			planned = append(typex.Keys(nil), keys...)
			return process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-limit"},
				Collected: typex.Keys{"root-limit", "201", "202"},
				Outcome:   process.TraversalOutcomeComplete,
			}, nil
		},
		cancelProcessInstances: dryRunCancelMutationGuard(t),
	}

	results, err := cancelProcessInstanceSearchPages(cmd, cli, nil, process.ProcessInstanceFilter{})
	require.NoError(t, err)
	require.Len(t, searchRequests, 1)
	require.EqualValues(t, 0, searchRequests[0].From)
	require.Equal(t, typex.Keys{"201", "202"}, planned)
	require.Empty(t, results.Reports)
	require.Len(t, results.DryRunPreviews, 1)
	require.Equal(t, 2, results.DryRunPreviews[0].RequestedCount)
}

// TestCancelProcessInstanceDryRun_SearchContinuesAfterEmptySelectedPage protects
// sparse search-derived cancel planning: an empty selected page with more
// backend matches must not hide later selectable candidates.
func TestCancelProcessInstanceDryRun_SearchContinuesAfterEmptySelectedPage(t *testing.T) {
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
				Items:         []process.ProcessInstance{{Key: "201", State: process.StateActive}},
				Request:       process.ProcessInstancePageRequest{From: 1, Size: 1},
				OverflowState: process.ProcessInstanceOverflowStateNoMore,
			}
			plan := process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-201"},
				Collected: typex.Keys{"root-201", "201"},
				Outcome:   process.TraversalOutcomeComplete,
			}
			action, err = visitor(process.ProcessInstanceMutationPlanStep{
				Page:             selectedPage,
				RequestedKeys:    []string{"201"},
				Plan:             plan,
				CumulativeCount:  1,
				CumulativeImpact: 2,
			})
			if err != nil {
				return process.ProcessInstanceMutationPlanPagesResult{}, err
			}
			actions = append(actions, action)
			return process.ProcessInstanceMutationPlanPagesResult{
				Plans:            []process.ProcessInstanceMutationPlanStep{{Page: selectedPage, RequestedKeys: []string{"201"}, Plan: plan, CumulativeCount: 1, CumulativeImpact: 2}},
				Pages:            2,
				RequestedCount:   1,
				CumulativeImpact: 2,
			}, nil
		},
		cancelProcessInstances: dryRunCancelMutationGuard(t),
	}

	results, err := cancelProcessInstanceSearchPages(cmd, cli, nil, process.ProcessInstanceFilter{})

	require.NoError(t, err)
	require.Equal(t, []process.ProcessInstanceSearchPageAction{
		process.ProcessInstanceSearchPageActionContinue,
		process.ProcessInstanceSearchPageActionStop,
	}, actions)
	require.Empty(t, results.Reports)
	require.Len(t, results.DryRunPreviews, 1)
	require.Equal(t, []string{"201"}, results.DryRunPreviews[0].RequestedKeys)
	require.NotContains(t, buf.String(), "found: 0")
}

// Verifies search-mode cancellation builds the expected date-filtered search request and no-ops cleanly on empty matches.
func TestCancelProcessInstanceSearchScaffold_UsesTempConfigAndCapturesSearchRequest(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeCancelProcessInstanceSuccessHelper(t, "TestCancelProcessInstanceSearchScaffoldHelper", cfgPath)

	filter := decodeCapturedPISearchFilter(t, requests)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Equal(t, "ACTIVE", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-01-31T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to cancel")
}

func TestCancelProcessInstanceBpmnSelectorMissingFailsBeforeSearch(t *testing.T) {
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

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceBpmnSelectorMissingFailsBeforeSearchHelper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Equal(t, []string{"POST /v2/process-definitions/search"}, requests)
	require.Contains(t, output, "no visible process definition matches the provided selector")
	require.Contains(t, output, "[missing-process]")
	require.NotContains(t, output, "found: 0")
}

func TestCancelProcessInstanceBpmnSelectorMissingMachineModesDoNotPrompt(t *testing.T) {
	tests := []struct {
		name   string
		helper string
	}{
		{name: "json", helper: "TestCancelProcessInstanceBpmnSelectorMissingJSONHelper"},
		{name: "automation", helper: "TestCancelProcessInstanceBpmnSelectorMissingAutomationHelper"},
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

			output, code := executeCancelProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.Error, code)
			require.Equal(t, []string{"POST /v2/process-definitions/search"}, requests)
			require.Contains(t, output, "no visible process definition matches the provided selector")
			require.NotContains(t, output, "List visible process definitions?")
		})
	}
}

func TestCancelProcessInstanceBpmnSelectorVisiblePreservesSearchNoOp(t *testing.T) {
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
		"cancel", "process-instance",
		"--state", "active",
		"--bpmn-process-id", "order-process",
	)

	require.Equal(t, []string{"POST /v2/process-definitions/search", "POST /v2/process-instances/search"}, requests)
	require.Equal(t, "found: 0\n", output)
}

// Verifies date-filtered search selection cancels matched instances and keeps descendant lookup behavior intact.
func TestCancelProcessInstanceCommand_SearchSelectionUsesDateFiltersAndCancelsMatches(t *testing.T) {
	var requests []string
	var cancelled testx.SafeSlice[string]

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
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813711967":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/2251799813711967/cancellation":
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeCancelProcessInstanceSuccessHelper(t, "TestCancelProcessInstanceCommand_SearchSelectionUsesDateFiltersAndCancelsMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 2)
	require.Equal(t, []string{"/v2/process-instances/2251799813711967/cancellation"}, cancelled.Snapshot())
	filter := decodeCapturedPISearchFilter(t, requests[:1])

	require.Equal(t, "ACTIVE", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-01-31T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")

	descendantSearch := decodeCapturedPISearchRequest(t, requests[1])
	descFilter := descendantSearch["filter"].(map[string]any)
	require.Equal(t, "2251799813711967", descFilter["parentProcessInstanceKey"])
	require.NotContains(t, output, "no process instance keys provided or found to cancel")
}

// Verifies relative-day search selection derives canonical start-date bounds before cancelling matches.
func TestCancelProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndCancelsMatches(t *testing.T) {
	var requests []string
	var cancelled testx.SafeSlice[string]

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
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-03-11T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813711967":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-03-11T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/2251799813711967/cancellation":
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeCancelProcessInstanceSuccessHelper(t, "TestCancelProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndCancelsMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 2)
	require.Equal(t, []string{"/v2/process-instances/2251799813711967/cancellation"}, cancelled.Snapshot())
	filter := decodeCapturedPISearchFilter(t, requests[:1])

	require.Equal(t, "ACTIVE", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-03-11T00:00:00Z")

	descendantSearch := decodeCapturedPISearchRequest(t, requests[1])
	descFilter := descendantSearch["filter"].(map[string]any)
	require.Equal(t, "2251799813711967", descFilter["parentProcessInstanceKey"])
	require.NotContains(t, output, "no process instance keys provided or found to cancel")
}

// Verifies cancel no-ops successfully when a date-filtered search returns no process instances.
func TestCancelProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatches(t *testing.T) {
	var requests []string

	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeCancelProcessInstanceSuccessHelper(t, "TestCancelProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to cancel")
}

// Verifies a relative-day-only filter is sufficient to trigger search mode.
func TestCancelProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficient(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeCancelProcessInstanceSuccessHelper(t, "TestCancelProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficientHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.NotContains(t, output, "either at least one --key is required, or sufficient filtering options")
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to cancel")
}

// TestCancelProcessInstanceCommand_SearchPagingPromptFlow verifies paged cancel search prompts between pages.
func TestCancelProcessInstanceCommand_SearchPagingPromptFlow(t *testing.T) {
	var requests testx.SafeSlice[string]
	var cancelled testx.SafeSlice[string]
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
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
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
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"101","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"102","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"103","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/101/cancellation" || r.URL.Path == "/v2/process-instances/102/cancellation" || r.URL.Path == "/v2/process-instances/103/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key)))
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
		"cancel", "process-instance",
		"--state", "active",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 0, pages[0]["from"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.ElementsMatch(t, []string{
		"/v2/process-instances/101/cancellation",
		"/v2/process-instances/102/cancellation",
		"/v2/process-instances/103/cancellation",
	}, cancelled.Snapshot())
	require.Len(t, prompts, 2)
	require.Contains(t, prompts[0], "You are about to cancel 2 process instance(s)")
	require.Contains(t, prompts[1], "Processed 2 process instance(s) on this page (2/3+ requested, 2 including dependencies). More matching process instances remain. Continue?")
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: prompt")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
	require.NotContains(t, output, "next step: auto-continue")
}

// TestCancelProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotals verifies v8.7 paging includes dependency totals.
func TestCancelProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotals(t *testing.T) {
	var requests testx.SafeSlice[string]
	var cancelled testx.SafeSlice[string]
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
					if parent != 701 && parent != 702 {
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(`{"items":[]}`))
						return
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"key":%d,"parentKey":%d,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"}]}`, parent+1000, parent)))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			searchMu.Lock()
			defer searchMu.Unlock()
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"key":701,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"},{"key":702,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"}],"total":3}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"key":701,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"},{"key":702,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"},{"key":703,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"}],"total":3}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/701/cancellation" || r.URL.Path == "/v2/process-instances/702/cancellation" || r.URL.Path == "/v2/process-instances/703/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")
	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotalsHelper", cfgPath)

	sizes := decodeCapturedTopLevelPISearchSizes(t, requests.Snapshot())
	require.Equal(t, exitcode.Error, code)
	require.Equal(t, []float64{2}, sizes)
	require.Empty(t, cancelled.Snapshot())
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "process-instance direct lookup by key is not tenant-safe in Camunda 8.7")
}

// TestCancelProcessInstancePage_PrintsOrphanWarningForPagedImpactCheck verifies paged impact-check warnings are printed.
func TestCancelProcessInstancePage_PrintsOrphanWarningForPagedImpactCheck(t *testing.T) {
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
				Roots:            typex.Keys{"2251799813711967"},
				Collected:        typex.Keys{"2251799813711967"},
				MissingAncestors: []process.MissingAncestor{{Key: "2251799813711999", StartKey: "2251799813711967"}},
				Warning:          "one or more parent process instances were not found",
				Outcome:          process.TraversalOutcomePartial,
			}, nil
		},
		cancelProcessInstances: func(_ context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.CancelReports, error) {
			require.Equal(t, typex.Keys{"2251799813711967"}, keys)
			require.Zero(t, wantedWorkers)
			cfg := options.ApplyFacadeOptions(opts)
			require.Equal(t, 1, cfg.AffectedProcessInstanceCount)
			require.True(t, cfg.SuppressWorkflowDetailLogs)
			require.True(t, cfg.SuppressProcessInstanceDetailLogs)
			return process.CancelReports{Items: []process.CancelReport{{Key: "2251799813711967", Ok: true}}}, nil
		},
	}

	got, err := cancelProcessInstancePage(cmd, cli, typex.Keys{"2251799813711967"}, false)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 1, Roots: 1}, got.Impact)
	require.Len(t, got.Reports, 1)
	require.Contains(t, buf.String(), "one or more parent process instances were not found")
	require.Contains(t, buf.String(), "missing ancestor keys: 2251799813711999")
	require.Contains(t, buf.String(), "cancellation: canceled 1/1 process-instance tree(s)")
}

// TestCancelProcessInstanceCommand_SearchPagingAutoConfirmFlow verifies --auto-confirm continues paged cancel searches.
func TestCancelProcessInstanceCommand_SearchPagingAutoConfirmFlow(t *testing.T) {
	var requests testx.SafeSlice[string]
	var cancelled testx.SafeSlice[string]
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
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
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
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"201","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"202","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"203","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/201/cancellation" || r.URL.Path == "/v2/process-instances/202/cancellation" || r.URL.Path == "/v2/process-instances/203/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key)))
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
		"cancel", "process-instance",
		"--state", "active",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v2/process-instances/201/cancellation",
		"/v2/process-instances/202/cancellation",
		"/v2/process-instances/203/cancellation",
	}, cancelled.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
}

// TestCancelProcessInstanceCommand_SearchPagingLimitFlow verifies cancel search stops at the requested limit.
func TestCancelProcessInstanceCommand_SearchPagingLimitFlow(t *testing.T) {
	var requests testx.SafeSlice[string]
	var cancelled testx.SafeSlice[string]
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
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
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
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"201","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"202","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":5,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"203","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"204","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":5,"hasMoreTotalItems":true}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/201/cancellation" || r.URL.Path == "/v2/process-instances/202/cancellation" || r.URL.Path == "/v2/process-instances/203/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key)))
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
		"cancel", "process-instance",
		"--state", "active",
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
		"/v2/process-instances/201/cancellation",
		"/v2/process-instances/202/cancellation",
		"/v2/process-instances/203/cancellation",
	}, cancelled.Snapshot())
	require.NotContains(t, strings.Join(cancelled.Snapshot(), "\n"), "204")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: yes, next step: limit-reached")
}

// TestCancelProcessInstanceCommand_SearchPagingBatchSizeLimitFlow verifies batch size and limit interact correctly.
func TestCancelProcessInstanceCommand_SearchPagingBatchSizeLimitFlow(t *testing.T) {
	var requests testx.SafeSlice[string]
	var cancelled testx.SafeSlice[string]

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
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"401","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"402","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"403","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"404","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":6,"hasMoreTotalItems":true}}`))
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/401/cancellation" || r.URL.Path == "/v2/process-instances/402/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key)))
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
		"cancel", "process-instance",
		"--state", "active",
		"--no-wait",
		"--batch-size", "4",
		"--limit", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 1)
	require.EqualValues(t, 4, pages[0]["limit"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v2/process-instances/401/cancellation",
		"/v2/process-instances/402/cancellation",
	}, cancelled.Snapshot())
	require.NotContains(t, strings.Join(cancelled.Snapshot(), "\n"), "403")
	require.Contains(t, output, "page size: 4, current page: 2, total so far: 2, more matches: yes, next step: limit-reached")
}

// TestCancelProcessInstanceCommand_SearchPagingAutomationFlow verifies automation mode auto-continues paged cancel searches.
func TestCancelProcessInstanceCommand_SearchPagingAutomationFlow(t *testing.T) {
	var requests testx.SafeSlice[string]
	var cancelled testx.SafeSlice[string]
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
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
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
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"301","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"302","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"303","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/301/cancellation" || r.URL.Path == "/v2/process-instances/302/cancellation" || r.URL.Path == "/v2/process-instances/303/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key)))
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
		"cancel", "process-instance",
		"--state", "active",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v2/process-instances/301/cancellation",
		"/v2/process-instances/302/cancellation",
		"/v2/process-instances/303/cancellation",
	}, cancelled.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
}

// TestCancelProcessInstanceCommand_SearchPagingPartialCompletionSummary verifies aborted paging reports partial completion.
func TestCancelProcessInstanceCommand_SearchPagingPartialCompletionSummary(t *testing.T) {
	var requests testx.SafeSlice[string]
	var cancelled testx.SafeSlice[string]

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
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"211","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"212","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/211/cancellation" || r.URL.Path == "/v2/process-instances/212/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key)))
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
		if callCount == 1 {
			return nil
		}
		return ErrCmdAborted
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"cancel", "process-instance",
		"--state", "active",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 1)
	require.ElementsMatch(t, []string{
		"/v2/process-instances/211/cancellation",
		"/v2/process-instances/212/cancellation",
	}, cancelled.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: partial-complete")
	require.Contains(t, output, "detail: stopped after 2 processed process instance(s); remaining matches were left untouched")
}

// TestCancelProcessInstanceCommand_SearchPagingWarningStopSummary verifies indeterminate overflow emits a warning summary.
func TestCancelProcessInstanceCommand_SearchPagingWarningStopSummary(t *testing.T) {
	var requests testx.SafeSlice[string]
	var cancelled testx.SafeSlice[string]

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
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"221","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"processInstanceKey":"222","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{}}`))
		case r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/221/cancellation" || r.URL.Path == "/v2/process-instances/222/cancellation"):
			cancelled.Append(r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`, key)))
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
		"cancel", "process-instance",
		"--state", "active",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 1)
	require.ElementsMatch(t, []string{
		"/v2/process-instances/221/cancellation",
		"/v2/process-instances/222/cancellation",
	}, cancelled.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: unknown, next step: warning-stop")
	require.Contains(t, output, "warning: stopped after 2 processed process instance(s) because more matching process instances may remain")
}

// Verifies invalid --state values are rejected through the shared invalid-args error path.
func TestCancelProcessInstanceCommand_RejectsInvalidSearchState(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsInvalidSearchStateHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, "invalid value for --state")
}

// Verifies invalid date literals for date flags are rejected with a clear YYYY-MM-DD validation error.
func TestCancelProcessInstanceCommand_RejectsInvalidDateFilter(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsInvalidDateFilterHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, `invalid value for --start-date-after: "2026-02-30", expected YYYY-MM-DD`)
}

// Verifies reversed date ranges are rejected when the after-bound is later than the before-bound.
func TestCancelProcessInstanceCommand_RejectsInvalidDateRange(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsInvalidDateRangeHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, `invalid range for --end-date-after and --end-date-before: "2026-02-01" is later than "2026-01-31"`)
}

// Verifies process-instance date filters are rejected for Camunda 8.7 where the capability is unsupported.
func TestCancelProcessInstanceCommand_RejectsDateFiltersOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsDateFiltersOnV87Helper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "process-instance date filters require Camunda 8.8")
}

// Verifies relative-day process-instance filters are also rejected for Camunda 8.7.
func TestCancelProcessInstanceCommand_RejectsRelativeDayFiltersOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsRelativeDayFiltersOnV87Helper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "process-instance date filters require Camunda 8.8")
}

// Helper-process entrypoint for the search scaffold failure test.
func TestCancelProcessInstanceSearchScaffoldHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31"}

	Execute()
}

func TestCancelProcessInstanceBpmnSelectorMissingFailsBeforeSearchHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "missing-process", "--auto-confirm"}

	Execute()
}

func TestCancelProcessInstanceBpmnSelectorMissingJSONHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--json", "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "missing-process", "--auto-confirm"}

	Execute()
}

func TestCancelProcessInstanceBpmnSelectorMissingAutomationHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--automation", "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "missing-process"}

	Execute()
}

// TestCancelProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotalsHelper is the helper-process entrypoint for v8.7 paging.
func TestCancelProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotalsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant", "--verbose", "cancel", "process-instance", "--state", "active", "--no-wait", "--batch-size", "2"}

	Execute()
}

// Helper-process entrypoint for the successful search-select-and-cancel flow test.
func TestCancelProcessInstanceCommand_SearchSelectionUsesDateFiltersAndCancelsMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31", "--auto-confirm", "--no-state-check", "--no-wait"}

	Execute()
}

// Helper-process entrypoint for the successful relative-day search-select-and-cancel flow test.
func TestCancelProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndCancelsMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-newer-days", "30", "--auto-confirm", "--no-state-check", "--no-wait"}

	Execute()
}

// Helper-process entrypoint for the no-matches failure test.
func TestCancelProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for relative-day-only sufficiency validation.
func TestCancelProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficientHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--start-date-older-days", "72"}

	Execute()
}

// Helper-process entrypoint for invalid --state validation.
func TestCancelProcessInstanceCommand_RejectsInvalidSearchStateHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "broken", "--bpmn-process-id", "order-process"}

	Execute()
}

// Helper-process entrypoint for invalid date format validation.
func TestCancelProcessInstanceCommand_RejectsInvalidDateFilterHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--start-date-after", "2026-02-30"}

	Execute()
}

// Helper-process entrypoint for invalid date range validation.
func TestCancelProcessInstanceCommand_RejectsInvalidDateRangeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--end-date-after", "2026-02-01", "--end-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for version capability validation on Camunda 8.7.
func TestCancelProcessInstanceCommand_RejectsDateFiltersOnV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01"}

	Execute()
}

// Helper-process entrypoint for relative-day version capability validation on Camunda 8.7.
func TestCancelProcessInstanceCommand_RejectsRelativeDayFiltersOnV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-newer-days", "30"}

	Execute()
}
