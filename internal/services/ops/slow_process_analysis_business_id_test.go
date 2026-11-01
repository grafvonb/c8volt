// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDOperationsSlowSelection verifies complete bounded discovery, copies and confirmation before enrichment.
func TestBusinessIDOperationsSlowSelection(t *testing.T) {
	for _, limit := range []int32{0, 2} {
		for _, definition := range []bool{false, true} {
			t.Run(fmt.Sprintf("limit=%d/definition=%t", limit, definition), func(t *testing.T) {
				bid := " ref,Ω *\n "
				original := bid
				searches, confirmations, enrichments := 0, 0, 0
				pi := stubProcessInstanceAPI{validateStates: func(d.States) error { return nil }, searchPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
					searches++
					require.Equal(t, original, *filter.BusinessId)
					require.False(t, services.ApplyCallOptions(opts).IgnoreTenant)
					require.Equal(t, d.StateCompleted, filter.State)
					require.False(t, *filter.HasIncident)
					require.Equal(t, "2026-01-01", filter.StartDateAfter)
					if definition {
						require.Equal(t, "order", filter.BpmnProcessId)
					}
					p := d.ProcessInstancePage{Request: page, OverflowState: d.ProcessInstanceOverflowStateHasMore, EndCursor: fmt.Sprint(searches)}
					switch searches {
					case 1:
						p.Items = []d.ProcessInstance{{Key: "101", BusinessId: new("observed"), TenantId: "a"}, {Key: "101"}}
					case 2:
					case 3:
						p.Items = []d.ProcessInstance{{Key: "102", TenantId: "a"}}
						p.OverflowState = d.ProcessInstanceOverflowStateNoMore
					default:
						t.Fatal("unexpected discovery")
					}
					return p, nil
				}}
				elements := stubSlowProcessAnalysisElementAPI{search: func(_ context.Context, q d.ElementSearchQuery, _ ...services.CallOption) (d.ElementSearchResult, error) {
					require.Equal(t, 3, searches)
					require.Equal(t, 1, confirmations)
					enrichments++
					require.Contains(t, []string{"101", "102"}, q.ProcessInstanceKey)
					return d.ElementSearchResult{}, nil
				}}
				req := d.SlowProcessAnalysisRequest{SelectionMode: d.SlowProcessAnalysisSelectionModeProcessInstanceSearch, Limit: limit, BatchSize: 2,
					ProcessInstanceFilters: d.SlowProcessAnalysisProcessInstanceSearchFilters{BusinessId: &bid, State: d.StateCompleted, NoIncidentsOnly: true, StartDateAfter: "2026-01-01"},
					ConfirmPreflight: func(scope d.OpsPreflightScope) error {
						confirmations++
						require.Zero(t, enrichments)
						require.EqualValues(t, 2, *scope.Total)
						require.Contains(t, scope.SelectorSummary, `business ID " ref,Ω *\n "`)
						bid = "caller changed"
						return nil
					},
				}
				if definition {
					req.SelectionMode = d.SlowProcessAnalysisSelectionModeProcessDefinitionSearch
					req.ProcessDefinitionSelector.BpmnProcessID = "order"
				}
				result, err := NewWithAnalysisDependencies(nil, pi, nil, nil, nil, nil, elements, toolx.V810).AnalyseSlowProcessInstances(context.Background(), req)
				require.NoError(t, err)
				require.Equal(t, 2, enrichments)
				require.Equal(t, 2, result.Count)
				require.Equal(t, original, *result.Request.ProcessInstanceFilters.BusinessId)
				require.Equal(t, original, *result.Selection.Request.SearchRequest.Filter.BusinessId)
				require.Equal(t, "observed", *result.Items[0].BusinessId)
				require.Nil(t, result.Items[1].BusinessId)
				require.Equal(t, limit > 0, result.DiscoveredScopeStatus.Limited)
				require.Equal(t, limit == 0, result.DiscoveredScopeStatus.Complete)
				require.Equal(t, 3, result.DiscoveredScopeStatus.Pages)
				require.False(t, result.Selection.IgnoreTenant)
			})
		}
	}
}

// TestBusinessIDOperationsSlowIncomplete rejects cyclic, failed, canceled and malformed discovery before enrichment or confirmation.
func TestBusinessIDOperationsSlowIncomplete(t *testing.T) {
	for _, mode := range []string{"late-error", "cycle", "missing-key", "cancel", "empty"} {
		t.Run(mode, func(t *testing.T) {
			calls, confirms, enrichments := 0, 0, 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			pi := stubProcessInstanceAPI{validateStates: func(d.States) error { return nil }, searchPage: func(_ context.Context, _ d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
				calls++
				p := d.ProcessInstancePage{Request: page, OverflowState: d.ProcessInstanceOverflowStateHasMore, EndCursor: "next"}
				if calls == 1 {
					if mode != "empty" {
						p.Items = []d.ProcessInstance{{Key: "101"}}
					}
					if mode == "missing-key" {
						p.Items = []d.ProcessInstance{{}}
						p.OverflowState = d.ProcessInstanceOverflowStateNoMore
					}
					if mode == "cancel" {
						cancel()
					}
					return p, nil
				}
				if mode == "late-error" {
					return p, fmt.Errorf("%w: backend failure", d.ErrNotFound)
				}
				if mode == "empty" {
					p.OverflowState = d.ProcessInstanceOverflowStateNoMore
					p.EndCursor = ""
				}
				return p, nil
			}}
			elements := stubSlowProcessAnalysisElementAPI{search: func(context.Context, d.ElementSearchQuery, ...services.CallOption) (d.ElementSearchResult, error) {
				enrichments++
				return d.ElementSearchResult{}, nil
			}}
			req := d.SlowProcessAnalysisRequest{SelectionMode: d.SlowProcessAnalysisSelectionModeProcessInstanceSearch, ProcessInstanceFilters: d.SlowProcessAnalysisProcessInstanceSearchFilters{BusinessId: new("ID")}, ConfirmPreflight: func(d.OpsPreflightScope) error { confirms++; return nil }}
			got, err := NewWithAnalysisDependencies(nil, pi, nil, nil, nil, nil, elements, toolx.V810).AnalyseSlowProcessInstances(ctx, req)
			require.Zero(t, confirms)
			require.Zero(t, enrichments)
			require.NotNil(t, got.Selection)
			if mode == "empty" {
				require.NoError(t, err)
				require.True(t, got.DiscoveredScopeStatus.Complete)
				require.False(t, got.PreflightScope.RequiresConfirmation)
			} else {
				require.Error(t, err)
				require.False(t, got.DiscoveredScopeStatus.Complete)
				require.Equal(t, d.ProcessInstanceSearchCompletionIncomplete, got.Selection.Completion)
				if mode == "late-error" {
					require.ErrorIs(t, err, d.ErrNotFound)
					require.Len(t, got.Selection.Keys, 1)
				}
			}
		})
	}
}

// TestBusinessIDOperationsSlowValidation prevents unsupported and keyed bypasses before any backend call.
func TestBusinessIDOperationsSlowValidation(t *testing.T) {
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		for _, mode := range []string{"bid", "empty", "utf8", "keyed", "input-keys", "both-definitions", "mismatched-mode"} {
			t.Run(string(version)+"/"+mode, func(t *testing.T) {
				req := d.SlowProcessAnalysisRequest{SelectionMode: d.SlowProcessAnalysisSelectionModeProcessInstanceSearch, ProcessInstanceFilters: d.SlowProcessAnalysisProcessInstanceSearchFilters{BusinessId: new("ID")}}
				switch mode {
				case "empty":
					req.ProcessInstanceFilters.BusinessId = new("")
				case "utf8":
					req.ProcessInstanceFilters.BusinessId = new(string([]byte{255}))
				case "keyed":
					req.SelectionMode = d.SlowProcessAnalysisSelectionModeExplicitKeys
					req.InputKeys = []string{"101"}
				case "input-keys":
					req.InputKeys = []string{"101"}
				case "both-definitions":
					req.SelectionMode = d.SlowProcessAnalysisSelectionModeProcessDefinitionSearch
					req.ProcessDefinitionSelector = d.SlowProcessAnalysisProcessDefinitionSelector{BpmnProcessID: "order", ProcessDefinitionKey: "900"}
				case "mismatched-mode":
					req.SelectionMode = d.SlowProcessAnalysisSelectionModeProcessDefinitionSearch
				}
				if version == toolx.V810 && mode == "bid" {
					return
				}
				_, err := NewWithAnalysisDependencies(nil, stubProcessInstanceAPI{}, nil, nil, nil, nil, nil, version).AnalyseSlowProcessInstances(context.Background(), req)
				if version != toolx.V810 {
					require.ErrorIs(t, err, d.ErrUnsupported)
				} else {
					require.ErrorIs(t, err, d.ErrValidation)
				}
			})
		}
	}
}

// TestBusinessIDOperationsSlowPartial keeps successful rows and original joined failures, with fail-fast and listener behavior.
func TestBusinessIDOperationsSlowPartial(t *testing.T) {
	for _, failFast := range []bool{false, true} {
		for _, listeners := range []bool{false, true} {
			t.Run(fmt.Sprintf("failfast=%t/listeners=%t", failFast, listeners), func(t *testing.T) {
				calls := 0
				pi := stubProcessInstanceAPI{validateStates: func(d.States) error { return nil }, searchPage: func(_ context.Context, _ d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
					return d.ProcessInstancePage{Request: page, Items: []d.ProcessInstance{{Key: "101"}, {Key: "102"}, {Key: "103"}}, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
				}}
				elements := stubSlowProcessAnalysisElementAPI{search: func(_ context.Context, q d.ElementSearchQuery, _ ...services.CallOption) (d.ElementSearchResult, error) {
					calls++
					if !listeners && q.ProcessInstanceKey != "101" {
						return d.ElementSearchResult{}, fmt.Errorf("%w: POST private body %s", d.ErrNotFound, q.ProcessInstanceKey)
					}
					return d.ElementSearchResult{}, nil
				}}
				jobs := stubSlowProcessAnalysisJobAPI{search: func(_ context.Context, q d.JobSearchQuery, _ ...services.CallOption) (d.JobSearchResult, error) {
					if q.ProcessInstanceKey != "101" {
						return d.JobSearchResult{}, fmt.Errorf("%w: POST private body %s", d.ErrNotFound, q.ProcessInstanceKey)
					}
					return d.JobSearchResult{}, nil
				}}
				req := d.SlowProcessAnalysisRequest{SelectionMode: d.SlowProcessAnalysisSelectionModeProcessInstanceSearch, ProcessInstanceFilters: d.SlowProcessAnalysisProcessInstanceSearchFilters{BusinessId: new("ID")}, WithListeners: listeners,
					Progress: func(e d.OpsProgressEvent) {
						if e.Completion != nil {
							require.Empty(t, e.Completion.FailureDetail)
						}
					},
				}
				opts := []services.CallOption{}
				if failFast {
					opts = append(opts, services.WithFailFast())
				}
				got, err := NewWithAnalysisDependencies(nil, pi, nil, nil, nil, jobs, elements, toolx.V810).AnalyseSlowProcessInstances(context.Background(), req, opts...)
				require.ErrorIs(t, err, d.ErrNotFound)
				require.Equal(t, 1, got.Count)
				require.Equal(t, "101", got.Items[0].Key)
				require.Len(t, got.Selection.Keys, 3)
				if failFast {
					require.Equal(t, 2, calls)
				} else {
					require.Equal(t, 3, calls)
					require.Contains(t, err.Error(), "103")
				}
			})
		}
	}
}

// TestBusinessIDOperationsSlowProgressAfterFailure never counts a failed root as loaded when later enrichment succeeds.
func TestBusinessIDOperationsSlowProgressAfterFailure(t *testing.T) {
	pi := stubProcessInstanceAPI{validateStates: func(d.States) error { return nil }, searchPage: func(_ context.Context, _ d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
		return d.ProcessInstancePage{Request: page, Items: []d.ProcessInstance{{Key: "101"}, {Key: "102"}, {Key: "103"}}, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
	}}
	elements := stubSlowProcessAnalysisElementAPI{search: func(_ context.Context, q d.ElementSearchQuery, _ ...services.CallOption) (d.ElementSearchResult, error) {
		if q.ProcessInstanceKey == "102" {
			return d.ElementSearchResult{}, d.ErrNotFound
		}
		return d.ElementSearchResult{}, nil
	}}
	var done []int
	req := d.SlowProcessAnalysisRequest{SelectionMode: d.SlowProcessAnalysisSelectionModeProcessInstanceSearch, ProcessInstanceFilters: d.SlowProcessAnalysisProcessInstanceSearchFilters{BusinessId: new("ID")},
		Progress: func(e d.OpsProgressEvent) {
			if e.FrozenScope != nil {
				done = append(done, e.FrozenScope.Done)
			}
		},
	}
	got, err := NewWithAnalysisDependencies(nil, pi, nil, nil, nil, nil, elements, toolx.V810).AnalyseSlowProcessInstances(context.Background(), req)
	require.ErrorIs(t, err, d.ErrNotFound)
	require.Equal(t, 2, got.Count)
	require.Equal(t, 2, got.FrozenScopeProgress.Done)
	require.Equal(t, 3, got.FrozenScopeProgress.Total)
	require.Equal(t, []int{0, 1, 2}, done)
}
