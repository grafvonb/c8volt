// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	esvc "github.com/grafvonb/c8volt/internal/services/element"
	jsvc "github.com/grafvonb/c8volt/internal/services/job"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// slowProcessAnalysisFixtureTime centralizes timestamp parsing for service analysis fixtures.
func slowProcessAnalysisFixtureTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	require.NoError(t, err)
	return parsed
}

// slowProcessAnalysisFixtureProcessInstance returns a minimal process-instance root for timing tests.
func slowProcessAnalysisFixtureProcessInstance(key string, start time.Time, end time.Time) d.ProcessInstance {
	return d.ProcessInstance{
		Key:                    key,
		RootProcessInstanceKey: key,
		ProcessDefinitionKey:   "2251799813687001",
		BpmnProcessId:          "OrderProcess",
		ProcessVersion:         7,
		State:                  d.StateCompleted,
		StartDate:              start.Format(time.RFC3339Nano),
		EndDate:                end.Format(time.RFC3339Nano),
		TenantId:               "tenant-a",
	}
}

// slowProcessAnalysisFixtureElement returns a runtime element row tied to a process instance.
func slowProcessAnalysisFixtureElement(processInstanceKey string, elementInstanceKey string, elementID string, start time.Time, end time.Time) d.Element {
	return d.Element{
		ElementInstanceKey:     elementInstanceKey,
		ElementId:              elementID,
		Type:                   "SERVICE_TASK",
		State:                  "COMPLETED",
		StartDate:              start.Format(time.RFC3339Nano),
		EndDate:                end.Format(time.RFC3339Nano),
		ProcessInstanceKey:     processInstanceKey,
		RootProcessInstanceKey: processInstanceKey,
		ProcessDefinitionKey:   "2251799813687001",
		TenantId:               "tenant-a",
	}
}

// TestSlowProcessAnalysisFixturesBuildConsistentTimingRows protects the shared fixture assumptions.
func TestSlowProcessAnalysisFixturesBuildConsistentTimingRows(t *testing.T) {
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	end := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:05:00Z")

	pi := slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, end)
	element := slowProcessAnalysisFixtureElement(pi.Key, "2251799813685250", "ReserveStock", start.Add(time.Second), start.Add(5*time.Second))

	require.Equal(t, pi.Key, pi.RootProcessInstanceKey)
	require.Equal(t, pi.Key, element.ProcessInstanceKey)
	require.Equal(t, pi.ProcessDefinitionKey, element.ProcessDefinitionKey)
	require.Equal(t, "ReserveStock", element.ElementId)
}

// TestSlowProcessAnalysisExplicitKeysDeduplicatesLooksUpAndSortsRoots verifies keyed MVP selection semantics.
func TestSlowProcessAnalysisExplicitKeysDeduplicatesLooksUpAndSortsRoots(t *testing.T) {
	captured := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:30:00Z")
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	var lookedUp testx.SafeSlice[string]
	instances := map[string]d.ProcessInstance{
		"2251799813685249": slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(2*time.Minute)),
		"2251799813685250": func() d.ProcessInstance {
			pi := slowProcessAnalysisFixtureProcessInstance("2251799813685250", start, start.Add(5*time.Minute))
			pi.EndDate = ""
			return pi
		}(),
		"2251799813685251": slowProcessAnalysisFixtureProcessInstance("2251799813685251", start, start.Add(5*time.Minute)),
	}
	piAPI := stubProcessInstanceAPI{
		search: func(_ context.Context, filter d.ProcessInstanceFilter, size int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			require.Equal(t, int32(2), size)
			lookedUp.Append(filter.Key)
			if item, ok := instances[filter.Key]; ok {
				return []d.ProcessInstance{item}, nil
			}
			return nil, fmt.Errorf("%w: missing", d.ErrNotFound)
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, stubSlowProcessAnalysisElementAPI{}, toolx.V88).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{"2251799813685249", "2251799813685250", "2251799813685249", "2251799813685251"},
		CapturedNow:   captured,
	})

	require.NoError(t, err)
	require.ElementsMatch(t, []string{"2251799813685249", "2251799813685250", "2251799813685251"}, lookedUp.Snapshot())
	require.Equal(t, []string{"2251799813685251", "2251799813685249", "2251799813685250"}, []string{got.Items[0].Key, got.Items[1].Key, got.Items[2].Key})
	require.Equal(t, "5m0s", got.Items[0].Duration)
	require.Equal(t, int64(300000), got.Items[0].DurationMillis)
	require.True(t, got.Items[0].DurationAvailable)
	require.False(t, got.Items[2].DurationAvailable)
	require.Equal(t, 3, got.Count)
	require.False(t, got.Empty)
	require.Equal(t, typex.Keys{"2251799813685249", "2251799813685250", "2251799813685251"}, got.Request.InputKeys)
	require.Equal(t, captured, got.CapturedAt)
}

// TestSlowProcessAnalysisProgressCallbackReceivesFrozenScopeSnapshots verifies service progress callbacks receive exact frozen counters.
func TestSlowProcessAnalysisProgressCallbackReceivesFrozenScopeSnapshots(t *testing.T) {
	captured := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:30:00Z")
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	root := slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(2*time.Minute))
	piAPI := stubProcessInstanceAPI{
		search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
			return []d.ProcessInstance{root}, nil
		},
	}
	elementAPI := stubSlowProcessAnalysisElementAPI{
		search: func(context.Context, d.ElementSearchQuery, ...services.CallOption) (d.ElementSearchResult, error) {
			return d.ElementSearchResult{Items: []d.Element{
				slowProcessAnalysisFixtureElement(root.Key, "2251799813685250", "ReserveStock", start, start.Add(time.Minute)),
			}}, nil
		},
	}
	var events []d.OpsProgressEvent

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, elementAPI, toolx.V88).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{root.Key},
		CapturedNow:   captured,
		Progress: func(event d.OpsProgressEvent) {
			if event.FrozenScope != nil {
				frozen := *event.FrozenScope
				event.FrozenScope = &frozen
			}
			events = append(events, event)
		},
	})

	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, d.OpsProgressEventKindFrozenScope, events[0].Kind)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading runtime elements", CoreResource: "process instance(s)", Done: 0, Total: 1}, *events[0].FrozenScope)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading runtime elements", CoreResource: "process instance(s)", Done: 1, Total: 1}, *events[1].FrozenScope)
	require.Equal(t, *events[1].FrozenScope, *got.FrozenScopeProgress)
}

// TestSlowProcessAnalysisNilProgressCallbackIsSafe verifies progress plumbing stays optional for callers.
func TestSlowProcessAnalysisNilProgressCallbackIsSafe(t *testing.T) {
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	root := slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(2*time.Minute))
	piAPI := stubProcessInstanceAPI{
		search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
			return []d.ProcessInstance{root}, nil
		},
	}
	elementAPI := stubSlowProcessAnalysisElementAPI{}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, elementAPI, toolx.V89).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{root.Key},
	})

	require.NoError(t, err)
	require.NotNil(t, got.FrozenScopeProgress)
	require.Equal(t, 1, got.FrozenScopeProgress.Done)
	require.Equal(t, 1, got.FrozenScopeProgress.Total)
}

// TestSlowProcessAnalysisExplicitKeysUsesBoundedWorkersForLookup verifies high-volume explicit key analysis overlaps tenant-safe lookups without unbounded fan-out.
func TestSlowProcessAnalysisExplicitKeysUsesBoundedWorkersForLookup(t *testing.T) {
	previousProcs := runtime.GOMAXPROCS(1)
	t.Cleanup(func() { runtime.GOMAXPROCS(previousProcs) })

	captured := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:30:00Z")
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	keys := typex.Keys{"2251799813685249", "2251799813685250", "2251799813685251"}
	instances := map[string]d.ProcessInstance{
		keys[0]: slowProcessAnalysisFixtureProcessInstance(keys[0], start, start.Add(2*time.Minute)),
		keys[1]: slowProcessAnalysisFixtureProcessInstance(keys[1], start, start.Add(3*time.Minute)),
		keys[2]: slowProcessAnalysisFixtureProcessInstance(keys[2], start, start.Add(4*time.Minute)),
	}
	started := make(chan string, len(keys))
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	piAPI := stubProcessInstanceAPI{
		search: func(_ context.Context, filter d.ProcessInstanceFilter, _ int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			started <- filter.Key
			<-release
			return []d.ProcessInstance{instances[filter.Key]}, nil
		},
	}
	done := make(chan struct {
		result d.SlowProcessAnalysisResult
		err    error
	}, 1)

	go func() {
		result, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, stubSlowProcessAnalysisElementAPI{}, toolx.V88).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
			SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
			InputKeys:     keys,
			CapturedNow:   captured,
		})
		done <- struct {
			result d.SlowProcessAnalysisResult
			err    error
		}{result: result, err: err}
	}()

	firstStarted := receiveStartedKeys(t, started, 2)
	require.Subset(t, []string(keys), firstStarted)
	requireNoAdditionalStart(t, started, 25*time.Millisecond)
	close(release)
	out := receiveSlowProcessAnalysisResult(t, done)
	require.NoError(t, out.err)
	require.Equal(t, []string{keys[2], keys[1], keys[0]}, []string{out.result.Items[0].Key, out.result.Items[1].Key, out.result.Items[2].Key})
	require.Equal(t, 3, out.result.Count)
}

// TestSlowProcessAnalysisExplicitKeysMeasuresActiveFromCapturedNow verifies active roots reuse one analysis timestamp.
func TestSlowProcessAnalysisExplicitKeysMeasuresActiveFromCapturedNow(t *testing.T) {
	captured := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:30:00Z")
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	active := slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, time.Time{})
	active.EndDate = ""
	active.State = d.StateActive
	piAPI := stubProcessInstanceAPI{
		search: func(_ context.Context, filter d.ProcessInstanceFilter, _ int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			require.Equal(t, "2251799813685249", filter.Key)
			return []d.ProcessInstance{active}, nil
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, stubSlowProcessAnalysisElementAPI{}, toolx.V89).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{"2251799813685249"},
		CapturedNow:   captured,
	})

	require.NoError(t, err)
	require.Equal(t, "30m0s", got.Items[0].Duration)
	require.Equal(t, int64(1800000), got.Items[0].DurationMillis)
	require.True(t, got.Items[0].DurationAvailable)
}

// TestSlowProcessAnalysisRootDurationFilterKeepsOnlyLongRoots verifies PI thresholds hide whole roots, not details.
func TestSlowProcessAnalysisRootDurationFilterKeepsOnlyLongRoots(t *testing.T) {
	captured := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:06:00Z")
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	active := slowProcessAnalysisFixtureProcessInstance("2251799813685252", start, time.Time{})
	active.State = d.StateActive
	active.EndDate = ""
	unknownDuration := slowProcessAnalysisFixtureProcessInstance("2251799813685253", start, time.Time{})
	unknownDuration.EndDate = ""
	instances := map[string]d.ProcessInstance{
		"2251799813685249": slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(10*time.Minute)),
		"2251799813685250": slowProcessAnalysisFixtureProcessInstance("2251799813685250", start, start.Add(5*time.Minute)),
		"2251799813685251": slowProcessAnalysisFixtureProcessInstance("2251799813685251", start, start.Add(2*time.Minute)),
		"2251799813685252": active,
		"2251799813685253": unknownDuration,
	}
	piAPI := stubProcessInstanceAPI{
		search: func(_ context.Context, filter d.ProcessInstanceFilter, _ int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			if item, ok := instances[filter.Key]; ok {
				return []d.ProcessInstance{item}, nil
			}
			return nil, fmt.Errorf("%w: missing", d.ErrNotFound)
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, stubSlowProcessAnalysisElementAPI{}, toolx.V88).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode:      d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:          typex.Keys{"2251799813685249", "2251799813685250", "2251799813685251", "2251799813685252", "2251799813685253"},
		RootDurationLonger: 5 * time.Minute,
		CapturedNow:        captured,
	})

	require.NoError(t, err)
	require.Equal(t, []string{"2251799813685249", "2251799813685252"}, []string{got.Items[0].Key, got.Items[1].Key})
	require.Equal(t, "10m0s", got.Items[0].Duration)
	require.Equal(t, "6m0s", got.Items[1].Duration)
	require.Equal(t, 2, got.Count)
	require.False(t, got.Empty)
}

// TestSlowProcessAnalysisExplicitKeysPropagatesLookupFailures verifies missing or unauthorized keys do not become partial success.
func TestSlowProcessAnalysisExplicitKeysPropagatesLookupFailures(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "missing", err: fmt.Errorf("%w: process instance 2251799813685249 not found", d.ErrNotFound)},
		{name: "unauthorized", err: fmt.Errorf("%w: process instance 2251799813685249", d.ErrUnauthorized)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			piAPI := stubProcessInstanceAPI{
				search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
					return nil, tc.err
				},
			}

			got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, stubSlowProcessAnalysisElementAPI{}, toolx.V88).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
				SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
				InputKeys:     typex.Keys{"2251799813685249"},
			})

			require.Error(t, err)
			require.True(t, errors.Is(err, errors.Unwrap(tc.err)))
			require.Empty(t, got.Items)
			require.True(t, got.Empty)
		})
	}
}

// TestSlowProcessAnalysisCamunda87Unsupported verifies version compatibility fails before remote lookup.
func TestSlowProcessAnalysisCamunda87Unsupported(t *testing.T) {
	piAPI := stubProcessInstanceAPI{
		search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
			t.Fatal("lookup should not run for unsupported Camunda 8.7")
			return nil, nil
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, stubSlowProcessAnalysisElementAPI{}, toolx.V87).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{"2251799813685249"},
	})

	require.ErrorIs(t, err, d.ErrUnsupported)
	require.Empty(t, got.Items)
	require.True(t, got.Empty)
}

// TestSlowProcessAnalysisProcessDefinitionSearchDiscoversFrozenSelection verifies search filters page into one frozen root set.
func TestSlowProcessAnalysisProcessDefinitionSearchDiscoversFrozenSelection(t *testing.T) {
	captured := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:30:00Z")
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	pageCalls := 0
	piAPI := stubProcessInstanceAPI{
		searchPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			pageCalls++
			require.Equal(t, "OrderProcess", filter.BpmnProcessId)
			require.Empty(t, filter.ProcessDefinitionKey)
			require.Equal(t, d.StateActive, filter.State)
			require.Equal(t, "2026-07-18T09:00:00Z", filter.StartDateAfter)
			require.Equal(t, "2026-07-19T23:59:59.999999999Z", filter.StartDateBefore)
			require.Equal(t, "2026-07-18T10:00:00Z", filter.EndDateAfter)
			require.Equal(t, "2026-07-20T23:59:59.999999999Z", filter.EndDateBefore)
			require.NotNil(t, filter.HasIncident)
			require.False(t, *filter.HasIncident)
			require.EqualValues(t, 2, page.Size)
			switch pageCalls {
			case 1:
				require.Zero(t, page.From)
				return d.ProcessInstancePage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateHasMore,
					Items: []d.ProcessInstance{
						slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(2*time.Minute)),
						slowProcessAnalysisFixtureProcessInstance("2251799813685250", start, start.Add(5*time.Minute)),
					},
				}, nil
			case 2:
				require.EqualValues(t, 2, page.From)
				return d.ProcessInstancePage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateHasMore,
					Items: []d.ProcessInstance{
						slowProcessAnalysisFixtureProcessInstance("2251799813685251", start, start.Add(time.Minute)),
					},
				}, nil
			default:
				t.Fatalf("unexpected discovery page %d", pageCalls)
				return d.ProcessInstancePage{}, nil
			}
		},
		search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
			t.Fatal("search-mode analysis should not perform explicit-key lookup")
			return nil, nil
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, stubSlowProcessAnalysisElementAPI{}, toolx.V88).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeProcessDefinitionSearch,
		ProcessDefinitionSelector: d.SlowProcessAnalysisProcessDefinitionSelector{
			BpmnProcessID: "OrderProcess",
		},
		ProcessInstanceFilters: d.SlowProcessAnalysisProcessInstanceSearchFilters{
			State:           d.StateActive,
			StartDateAfter:  "2026-07-18T09:00:00Z",
			StartDateBefore: "2026-07-19T23:59:59.999999999Z",
			EndDateAfter:    "2026-07-18T10:00:00Z",
			EndDateBefore:   "2026-07-20T23:59:59.999999999Z",
			NoIncidentsOnly: true,
		},
		BatchSize:   2,
		Limit:       3,
		CapturedNow: captured,
	})

	require.NoError(t, err)
	require.Equal(t, 2, pageCalls)
	require.Equal(t, []string{"2251799813685250", "2251799813685249", "2251799813685251"}, []string{got.Items[0].Key, got.Items[1].Key, got.Items[2].Key})
	require.Equal(t, 3, got.Count)
	require.False(t, got.Empty)
	require.Equal(t, d.DiscoveryScopeStatus{
		Complete:         false,
		Limited:          true,
		Limit:            3,
		BatchSize:        2,
		Pages:            2,
		CandidatesSeen:   3,
		CandidatesFrozen: 3,
	}, got.DiscoveredScopeStatus)
	require.Equal(t, captured, got.CapturedAt)
}

// TestSlowProcessAnalysisProcessDefinitionSearchReusesPreflightPage verifies preflight does not refetch the initial page.
func TestSlowProcessAnalysisProcessDefinitionSearchReusesPreflightPage(t *testing.T) {
	captured := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:30:00Z")
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	var preflightEvents []d.OpsPreflightScope
	var confirmed []d.OpsPreflightScope
	pageCalls := 0
	piAPI := stubProcessInstanceAPI{
		searchPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			pageCalls++
			require.Equal(t, "OrderProcess", filter.BpmnProcessId)
			require.EqualValues(t, 2, page.Size)
			switch pageCalls {
			case 1:
				require.Zero(t, page.From)
				require.Empty(t, page.After)
				total := d.ProcessInstanceReportedTotal{Count: 3, Kind: d.ProcessInstanceReportedTotalKindExact}
				return d.ProcessInstancePage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateHasMore,
					ReportedTotal: &total,
					EndCursor:     "cursor-1",
					Items: []d.ProcessInstance{
						slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(2*time.Minute)),
						slowProcessAnalysisFixtureProcessInstance("2251799813685250", start, start.Add(5*time.Minute)),
					},
				}, nil
			case 2:
				require.Equal(t, "cursor-1", page.After)
				return d.ProcessInstancePage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateNoMore,
					Items: []d.ProcessInstance{
						slowProcessAnalysisFixtureProcessInstance("2251799813685251", start, start.Add(time.Minute)),
					},
				}, nil
			default:
				t.Fatalf("unexpected repeated discovery page %d", pageCalls)
				return d.ProcessInstancePage{}, nil
			}
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, stubSlowProcessAnalysisElementAPI{}, toolx.V88).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		CommandName:   "ops analyse slow-process-instances",
		SelectionMode: d.SlowProcessAnalysisSelectionModeProcessDefinitionSearch,
		ProcessDefinitionSelector: d.SlowProcessAnalysisProcessDefinitionSelector{
			BpmnProcessID: "OrderProcess",
		},
		BatchSize:   2,
		CapturedNow: captured,
		Progress: func(event d.OpsProgressEvent) {
			if event.Kind == d.OpsProgressEventKindPreflight && event.Preflight != nil {
				preflightEvents = append(preflightEvents, *event.Preflight)
			}
		},
		ConfirmPreflight: func(scope d.OpsPreflightScope) error {
			confirmed = append(confirmed, scope)
			return nil
		},
	})

	require.NoError(t, err)
	require.Equal(t, 2, pageCalls)
	require.Len(t, preflightEvents, 1)
	require.Len(t, confirmed, 1)
	require.Equal(t, preflightEvents[0], confirmed[0])
	require.NotNil(t, got.PreflightScope)
	require.Equal(t, d.OpsTotalCertaintyExact, got.PreflightScope.TotalKind)
	require.Equal(t, int64(3), *got.PreflightScope.Total)
	require.Equal(t, d.OpsPageCountKindExact, got.PreflightScope.PageCountKind)
	require.Equal(t, int64(2), *got.PreflightScope.PageCount)
	require.Equal(t, "OrderProcess", got.PreflightScope.SelectorSummary)
	require.True(t, got.PreflightScope.RequiresConfirmation)
	require.Contains(t, got.PreflightScope.ConsequenceSummary.ConfirmationText, "Continue slow analysis")
	require.Equal(t, []string{"2251799813685250", "2251799813685249", "2251799813685251"}, []string{got.Items[0].Key, got.Items[1].Key, got.Items[2].Key})
}

// TestSlowProcessAnalysisProcessDefinitionSearchBuildsLowerBoundAndUnknownPreflight verifies non-exact totals remain labeled.
func TestSlowProcessAnalysisProcessDefinitionSearchBuildsLowerBoundAndUnknownPreflight(t *testing.T) {
	tests := []struct {
		name          string
		reportedTotal *d.ProcessInstanceReportedTotal
		wantKind      d.OpsTotalCertainty
		wantPages     *int64
		wantPageKind  d.OpsPageCountKind
	}{
		{name: "lower bound", reportedTotal: &d.ProcessInstanceReportedTotal{Count: 2000, Kind: d.ProcessInstanceReportedTotalKindLowerBound}, wantKind: d.OpsTotalCertaintyLowerBound, wantPages: ptrDomainInt64(2), wantPageKind: d.OpsPageCountKindEstimated},
		{name: "unknown", wantKind: d.OpsTotalCertaintyUnknown, wantPageKind: d.OpsPageCountKindUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			piAPI := stubProcessInstanceAPI{
				searchPage: func(_ context.Context, _ d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
					return d.ProcessInstancePage{Request: page, OverflowState: d.ProcessInstanceOverflowStateNoMore, ReportedTotal: tc.reportedTotal}, nil
				},
			}

			got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, stubSlowProcessAnalysisElementAPI{}, toolx.V89).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
				SelectionMode: d.SlowProcessAnalysisSelectionModeProcessDefinitionSearch,
				ProcessDefinitionSelector: d.SlowProcessAnalysisProcessDefinitionSelector{
					ProcessDefinitionKey: "2251799813687001",
				},
			})

			require.NoError(t, err)
			require.NotNil(t, got.PreflightScope)
			require.Equal(t, tc.wantKind, got.PreflightScope.TotalKind)
			require.Equal(t, tc.wantPageKind, got.PreflightScope.PageCountKind)
			if tc.wantPages == nil {
				require.Nil(t, got.PreflightScope.PageCount)
				require.Nil(t, got.PreflightScope.Total)
			} else {
				require.Equal(t, *tc.wantPages, *got.PreflightScope.PageCount)
			}
		})
	}
}

// TestSlowProcessAnalysisProcessDefinitionSearchSupportsSelectorsAndStates verifies accepted state values and selector modes.
func TestSlowProcessAnalysisProcessDefinitionSearchSupportsSelectorsAndStates(t *testing.T) {
	tests := []struct {
		name       string
		selector   d.SlowProcessAnalysisProcessDefinitionSelector
		state      d.State
		wantFilter d.ProcessInstanceFilter
	}{
		{name: "bpmn all", selector: d.SlowProcessAnalysisProcessDefinitionSelector{BpmnProcessID: "OrderProcess"}, state: d.StateAll, wantFilter: d.ProcessInstanceFilter{BpmnProcessId: "OrderProcess"}},
		{name: "pd active", selector: d.SlowProcessAnalysisProcessDefinitionSelector{ProcessDefinitionKey: "2251799813687001"}, state: d.StateActive, wantFilter: d.ProcessInstanceFilter{ProcessDefinitionKey: "2251799813687001", State: d.StateActive}},
		{name: "pd completed", selector: d.SlowProcessAnalysisProcessDefinitionSelector{ProcessDefinitionKey: "2251799813687001"}, state: d.StateCompleted, wantFilter: d.ProcessInstanceFilter{ProcessDefinitionKey: "2251799813687001", State: d.StateCompleted}},
		{name: "pd canceled", selector: d.SlowProcessAnalysisProcessDefinitionSelector{ProcessDefinitionKey: "2251799813687001"}, state: d.StateCanceled, wantFilter: d.ProcessInstanceFilter{ProcessDefinitionKey: "2251799813687001", State: d.StateCanceled}},
		{name: "pd terminated", selector: d.SlowProcessAnalysisProcessDefinitionSelector{ProcessDefinitionKey: "2251799813687001"}, state: d.StateTerminated, wantFilter: d.ProcessInstanceFilter{ProcessDefinitionKey: "2251799813687001", State: d.StateTerminated}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			piAPI := stubProcessInstanceAPI{
				searchPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
					require.Equal(t, tc.wantFilter, filter)
					require.EqualValues(t, 1000, page.Size)
					return d.ProcessInstancePage{Request: page, OverflowState: d.ProcessInstanceOverflowStateNoMore, Items: nil}, nil
				},
			}

			got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, stubSlowProcessAnalysisElementAPI{}, toolx.V89).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
				SelectionMode:             d.SlowProcessAnalysisSelectionModeProcessDefinitionSearch,
				ProcessDefinitionSelector: tc.selector,
				ProcessInstanceFilters:    d.SlowProcessAnalysisProcessInstanceSearchFilters{State: tc.state},
			})

			require.NoError(t, err)
			require.Empty(t, got.Items)
			require.True(t, got.Empty)
			require.Equal(t, 0, got.Count)
		})
	}
}

// TestSlowProcessAnalysisProcessDefinitionSearchEmptyResultSucceeds verifies no-match discovery is not an error.
func TestSlowProcessAnalysisProcessDefinitionSearchEmptyResultSucceeds(t *testing.T) {
	piAPI := stubProcessInstanceAPI{
		searchPage: func(_ context.Context, _ d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			return d.ProcessInstancePage{Request: page, OverflowState: d.ProcessInstanceOverflowStateNoMore, Items: []d.ProcessInstance{}}, nil
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, nil, toolx.V88).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeProcessDefinitionSearch,
		ProcessDefinitionSelector: d.SlowProcessAnalysisProcessDefinitionSelector{
			BpmnProcessID: "EmptyProcess",
		},
	})

	require.NoError(t, err)
	require.Empty(t, got.Items)
	require.Equal(t, 0, got.Count)
	require.True(t, got.Empty)
	require.Equal(t, 1, got.DiscoveredScopeStatus.Pages)
	require.True(t, got.DiscoveredScopeStatus.Complete)
}

// TestSlowProcessAnalysisRuntimeElementsBuildChronologicalTimeline verifies element lookup, ordering, duration states, and incident markers.
func TestSlowProcessAnalysisRuntimeElementsBuildChronologicalTimeline(t *testing.T) {
	captured := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:10:00Z")
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	root := slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(10*time.Minute))
	elements := []d.Element{
		slowProcessAnalysisFixtureElement(root.Key, "2251799813685251", "ActiveWait", start.Add(2*time.Minute), time.Time{}),
		slowProcessAnalysisFixtureElement(root.Key, "2251799813685250", "ReserveStock", start.Add(10*time.Second), start.Add(time.Minute)),
		slowProcessAnalysisFixtureElement(root.Key, "2251799813685253", "MissingEnd", start.Add(4*time.Minute), time.Time{}),
		slowProcessAnalysisFixtureElement(root.Key, "2251799813685252", "TerminatePath", start.Add(3*time.Minute), start.Add(4*time.Minute)),
	}
	elements[0].State = "ACTIVE"
	elements[0].Type = "USER_TASK"
	elements[0].EndDate = ""
	elements[2].EndDate = ""
	elements[2].HasIncident = true
	elements[2].IncidentKey = "2251799813687777"
	elements[3].State = "TERMINATED"
	piAPI := stubProcessInstanceAPI{
		search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
			return []d.ProcessInstance{root}, nil
		},
	}
	elementAPI := stubSlowProcessAnalysisElementAPI{
		search: func(_ context.Context, query d.ElementSearchQuery, _ ...services.CallOption) (d.ElementSearchResult, error) {
			require.Equal(t, d.ElementSearchQuery{ProcessInstanceKey: root.Key}, query)
			return d.ElementSearchResult{Items: elements}, nil
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, elementAPI, toolx.V88).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{root.Key},
		CapturedNow:   captured,
	})

	require.NoError(t, err)
	elementRows := slowProcessAnalysisTimelineElements(got.Items[0].Timeline)
	require.Equal(t, []string{"2251799813685250", "2251799813685251", "2251799813685252", "2251799813685253"}, []string{elementRows[0].ElementInstanceKey, elementRows[1].ElementInstanceKey, elementRows[2].ElementInstanceKey, elementRows[3].ElementInstanceKey})
	require.Equal(t, "50s", elementRows[0].Duration)
	require.Equal(t, "8m0s", elementRows[1].Duration)
	require.Equal(t, "1m0s", elementRows[2].Duration)
	require.True(t, elementRows[2].DurationAvailable)
	require.False(t, elementRows[3].DurationAvailable)
	require.True(t, elementRows[3].HasIncident)
	require.Equal(t, "2251799813687777", elementRows[3].IncidentKey)
}

// TestSlowProcessAnalysisWithListenersAttachesOnlyMatchingElementJobs verifies listener lookup uses element ownership.
func TestSlowProcessAnalysisWithListenersAttachesOnlyMatchingElementJobs(t *testing.T) {
	captured := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:10:00Z")
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	root := slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(10*time.Minute))
	elements := []d.Element{
		slowProcessAnalysisFixtureElement(root.Key, "2251799813685250", "ReserveStock", start.Add(10*time.Second), start.Add(time.Minute)),
		slowProcessAnalysisFixtureElement(root.Key, "2251799813685251", "PackOrder", start.Add(2*time.Minute), start.Add(3*time.Minute)),
	}
	piAPI := stubProcessInstanceAPI{
		search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
			return []d.ProcessInstance{root}, nil
		},
	}
	elementAPI := stubSlowProcessAnalysisElementAPI{
		search: func(_ context.Context, query d.ElementSearchQuery, _ ...services.CallOption) (d.ElementSearchResult, error) {
			require.Equal(t, root.Key, query.ProcessInstanceKey)
			return d.ElementSearchResult{Items: elements}, nil
		},
	}
	var jobQueries []d.JobSearchQuery
	jobAPI := stubSlowProcessAnalysisJobAPI{
		search: func(_ context.Context, query d.JobSearchQuery, _ ...services.CallOption) (d.JobSearchResult, error) {
			jobQueries = append(jobQueries, query)
			return d.JobSearchResult{Items: []d.Job{
				{Key: "job-match", Kind: query.Kind, ListenerEventType: "START", State: "CREATED", Type: "audit", Retries: 3, ProcessInstanceKey: root.Key, ElementInstanceKey: "2251799813685250"},
				{Key: "job-unmatched-element", Kind: query.Kind, ProcessInstanceKey: root.Key, ElementInstanceKey: "missing"},
				{Key: "job-other-process", Kind: query.Kind, ProcessInstanceKey: "other", ElementInstanceKey: "2251799813685250"},
			}}, nil
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, jobAPI, elementAPI, toolx.V88).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{root.Key},
		CapturedNow:   captured,
		WithListeners: true,
	})

	require.NoError(t, err)
	require.Equal(t, []d.JobSearchQuery{
		{ProcessInstanceKey: root.Key, Kind: d.JobKindExecutionListener},
		{ProcessInstanceKey: root.Key, Kind: d.JobKindTaskListener},
	}, jobQueries)
	elementRows := slowProcessAnalysisTimelineElements(got.Items[0].Timeline)
	require.NotNil(t, elementRows[0].Listeners)
	require.Equal(t, []string{"job-match", "job-match"}, []string{(*elementRows[0].Listeners)[0].JobKey, (*elementRows[0].Listeners)[1].JobKey})
	require.Equal(t, []d.RuntimeListenerJob{}, *elementRows[1].Listeners)
}

// TestSlowProcessAnalysisWithoutListenersDoesNotLookupJobs verifies default output remains listener-free.
func TestSlowProcessAnalysisWithoutListenersDoesNotLookupJobs(t *testing.T) {
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	root := slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(10*time.Minute))
	piAPI := stubProcessInstanceAPI{
		search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
			return []d.ProcessInstance{root}, nil
		},
	}
	elementAPI := stubSlowProcessAnalysisElementAPI{
		search: func(context.Context, d.ElementSearchQuery, ...services.CallOption) (d.ElementSearchResult, error) {
			return d.ElementSearchResult{Items: []d.Element{
				slowProcessAnalysisFixtureElement(root.Key, "2251799813685250", "ReserveStock", start, start.Add(time.Minute)),
			}}, nil
		},
	}
	jobAPI := stubSlowProcessAnalysisJobAPI{
		search: func(context.Context, d.JobSearchQuery, ...services.CallOption) (d.JobSearchResult, error) {
			t.Fatal("listener jobs should not be searched when WithListeners is false")
			return d.JobSearchResult{}, nil
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, jobAPI, elementAPI, toolx.V89).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{root.Key},
		CapturedNow:   start.Add(30 * time.Minute),
	})

	require.NoError(t, err)
	elementRows := slowProcessAnalysisTimelineElements(got.Items[0].Timeline)
	require.Nil(t, elementRows[0].Listeners)
}

// TestSlowProcessAnalysisWithListenersPropagatesUnsupportedJobLookup verifies requested listener failures fail the run.
func TestSlowProcessAnalysisWithListenersPropagatesUnsupportedJobLookup(t *testing.T) {
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	root := slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(10*time.Minute))
	piAPI := stubProcessInstanceAPI{
		search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
			return []d.ProcessInstance{root}, nil
		},
	}
	elementAPI := stubSlowProcessAnalysisElementAPI{
		search: func(context.Context, d.ElementSearchQuery, ...services.CallOption) (d.ElementSearchResult, error) {
			return d.ElementSearchResult{Items: []d.Element{}}, nil
		},
	}
	jobAPI := stubSlowProcessAnalysisJobAPI{
		search: func(context.Context, d.JobSearchQuery, ...services.CallOption) (d.JobSearchResult, error) {
			return d.JobSearchResult{}, fmt.Errorf("%w: search jobs requires Camunda 8.8 or newer", d.ErrUnsupported)
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, jobAPI, elementAPI, toolx.V89).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{root.Key},
		WithListeners: true,
	})

	require.ErrorIs(t, err, d.ErrUnsupported)
	require.Empty(t, got.Items)
	require.True(t, got.Empty)
}

// TestSlowProcessAnalysisTransitionsUseOnlyAdjacentChronologicalElements verifies gap timing without overlap or synthetic bridging.
func TestSlowProcessAnalysisTransitionsUseOnlyAdjacentChronologicalElements(t *testing.T) {
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	root := slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(10*time.Minute))
	elements := []d.Element{
		slowProcessAnalysisFixtureElement(root.Key, "2251799813685250", "A", start, start.Add(time.Minute)),
		slowProcessAnalysisFixtureElement(root.Key, "2251799813685251", "B", start.Add(2*time.Minute), start.Add(3*time.Minute)),
		slowProcessAnalysisFixtureElement(root.Key, "2251799813685252", "C", start.Add(150*time.Second), start.Add(4*time.Minute)),
		slowProcessAnalysisFixtureElement(root.Key, "2251799813685253", "D", start.Add(5*time.Minute), time.Time{}),
		slowProcessAnalysisFixtureElement(root.Key, "2251799813685254", "E", start.Add(6*time.Minute), start.Add(7*time.Minute)),
	}
	elements[3].EndDate = ""
	piAPI := stubProcessInstanceAPI{
		search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
			return []d.ProcessInstance{root}, nil
		},
	}
	elementAPI := stubSlowProcessAnalysisElementAPI{
		search: func(context.Context, d.ElementSearchQuery, ...services.CallOption) (d.ElementSearchResult, error) {
			return d.ElementSearchResult{Items: elements}, nil
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, elementAPI, toolx.V89).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{root.Key},
		CapturedNow:   start.Add(30 * time.Minute),
	})

	require.NoError(t, err)
	transitions := slowProcessAnalysisTimelineTransitions(got.Items[0].Timeline)
	require.Equal(t, []string{"A -> B", "C -> D"}, []string{transitions[0].FromElementID + " -> " + transitions[0].ToElementID, transitions[1].FromElementID + " -> " + transitions[1].ToElementID})
	require.Equal(t, "1m0s", transitions[0].Duration)
	require.Equal(t, "1m0s", transitions[1].Duration)
}

// TestSlowProcessAnalysisDetailFiltersApplyAfterCompleteTimelineCalculations verifies visible rows never bridge hidden elements.
func TestSlowProcessAnalysisDetailFiltersApplyAfterCompleteTimelineCalculations(t *testing.T) {
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	root := slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(10*time.Minute))
	elements := []d.Element{
		slowProcessAnalysisFixtureElement(root.Key, "2251799813685250", "A", start, start.Add(2*time.Minute)),
		slowProcessAnalysisFixtureElement(root.Key, "2251799813685251", "B", start.Add(4*time.Minute), start.Add(5*time.Minute)),
		slowProcessAnalysisFixtureElement(root.Key, "2251799813685252", "C", start.Add(6*time.Minute), start.Add(7*time.Minute)),
	}
	elements[1].Type = "USER_TASK"
	piAPI := stubProcessInstanceAPI{
		search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
			return []d.ProcessInstance{root}, nil
		},
	}
	elementAPI := stubSlowProcessAnalysisElementAPI{
		search: func(context.Context, d.ElementSearchQuery, ...services.CallOption) (d.ElementSearchResult, error) {
			return d.ElementSearchResult{Items: elements}, nil
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, elementAPI, toolx.V88).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{root.Key},
		CapturedNow:   start.Add(30 * time.Minute),
		DetailFilters: d.SlowProcessAnalysisDetailFilters{
			ElementID:     "B",
			Type:          "USER_TASK",
			ElementState:  "COMPLETED",
			DurationAfter: 45 * time.Second,
		},
	})

	require.NoError(t, err)
	require.Equal(t, root.Key, got.Items[0].Key)
	require.Equal(t, 3, len(got.Items[0].Timeline))
	require.Equal(t, d.SlowProcessAnalysisTimelineEntryKindTransition, got.Items[0].Timeline[0].Kind)
	require.Equal(t, "A", got.Items[0].Timeline[0].FromElementID)
	require.Equal(t, "B", got.Items[0].Timeline[0].ToElementID)
	require.Equal(t, d.SlowProcessAnalysisTimelineEntryKindElement, got.Items[0].Timeline[1].Kind)
	require.Equal(t, "B", got.Items[0].Timeline[1].ElementID)
	require.Equal(t, d.SlowProcessAnalysisTimelineEntryKindTransition, got.Items[0].Timeline[2].Kind)
	require.Equal(t, "B", got.Items[0].Timeline[2].FromElementID)
	require.Equal(t, "C", got.Items[0].Timeline[2].ToElementID)
}

// TestSlowProcessAnalysisDetailFiltersDropRootsWithoutMatchingTimelineRows verifies detail filters narrow result roots.
func TestSlowProcessAnalysisDetailFiltersDropRootsWithoutMatchingTimelineRows(t *testing.T) {
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	roots := []d.ProcessInstance{
		slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(800*time.Hour)),
		slowProcessAnalysisFixtureProcessInstance("2251799813685250", start, start.Add(800*time.Hour)),
		slowProcessAnalysisFixtureProcessInstance("2251799813685251", start, start.Add(145*time.Hour)),
	}
	elementSets := map[string][]d.Element{
		"2251799813685249": {
			slowProcessAnalysisFixtureElement("2251799813685249", "2251799813685301", "LongCallActivity", start, start.Add(701*time.Hour)),
		},
		"2251799813685250": {
			slowProcessAnalysisFixtureElement("2251799813685250", "2251799813685302", "ShortCallActivity", start, start.Add(699*time.Hour)),
		},
		"2251799813685251": {
			slowProcessAnalysisFixtureElement("2251799813685251", "2251799813685303", "ShortRecentActivity", start, start.Add(10*time.Hour)),
		},
	}
	piAPI := stubProcessInstanceAPI{
		searchPage: func(_ context.Context, _ d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			return d.ProcessInstancePage{Request: page, OverflowState: d.ProcessInstanceOverflowStateNoMore, Items: roots}, nil
		},
	}
	elementAPI := stubSlowProcessAnalysisElementAPI{
		search: func(_ context.Context, query d.ElementSearchQuery, _ ...services.CallOption) (d.ElementSearchResult, error) {
			return d.ElementSearchResult{Items: elementSets[query.ProcessInstanceKey]}, nil
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, elementAPI, toolx.V89).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeProcessDefinitionSearch,
		ProcessDefinitionSelector: d.SlowProcessAnalysisProcessDefinitionSelector{
			BpmnProcessID: "OrderProcess",
		},
		DetailFilters: d.SlowProcessAnalysisDetailFilters{DurationAfter: 700 * time.Hour},
		CapturedNow:   start.Add(900 * time.Hour),
	})

	require.NoError(t, err)
	require.False(t, got.Empty)
	require.Equal(t, 1, got.Count)
	require.Equal(t, "2251799813685249", got.Items[0].Key)
	require.Len(t, got.Items[0].Timeline, 1)
	require.Equal(t, "LongCallActivity", got.Items[0].Timeline[0].ElementID)
	require.Equal(t, "701h0m0s", got.Items[0].Timeline[0].Duration)
}

// TestSlowProcessAnalysisComparisonIndicatorsUseScopedSamples verifies relative indicators use explicit comparable groups.
func TestSlowProcessAnalysisComparisonIndicatorsUseScopedSamples(t *testing.T) {
	captured := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:30:00Z")
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	roots := []d.ProcessInstance{
		slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(10*time.Minute)),
		slowProcessAnalysisFixtureProcessInstance("2251799813685250", start, start.Add(5*time.Minute)),
		slowProcessAnalysisFixtureProcessInstance("2251799813685251", start, start.Add(5*time.Minute)),
		slowProcessAnalysisFixtureProcessInstance("2251799813685252", start, start.Add(time.Minute)),
		func() d.ProcessInstance {
			pi := slowProcessAnalysisFixtureProcessInstance("2251799813685253", start, start.Add(20*time.Minute))
			pi.ProcessDefinitionKey = "2251799813687999"
			return pi
		}(),
	}
	elementSets := map[string][]d.Element{
		"2251799813685249": {
			slowProcessAnalysisFixtureElement("2251799813685249", "2251799813685301", "ReserveStock", start, start.Add(9*time.Second)),
			slowProcessAnalysisFixtureElement("2251799813685249", "2251799813685302", "OrderFinished", start.Add(20*time.Second), start.Add(30*time.Second)),
		},
		"2251799813685250": {
			slowProcessAnalysisFixtureElement("2251799813685250", "2251799813685303", "ReserveStock", start, start.Add(4*time.Second)),
			slowProcessAnalysisFixtureElement("2251799813685250", "2251799813685304", "OrderFinished", start.Add(10*time.Second), start.Add(20*time.Second)),
		},
		"2251799813685251": {
			slowProcessAnalysisFixtureElement("2251799813685251", "2251799813685305", "ReserveStock", start, start.Add(4*time.Second)),
			slowProcessAnalysisFixtureElement("2251799813685251", "2251799813685306", "OrderFinished", start.Add(10*time.Second), start.Add(20*time.Second)),
		},
		"2251799813685252": {
			slowProcessAnalysisFixtureElement("2251799813685252", "2251799813685307", "ReserveStock", start, start.Add(time.Second)),
			slowProcessAnalysisFixtureElement("2251799813685252", "2251799813685308", "OrderFinished", start.Add(3*time.Second), start.Add(4*time.Second)),
		},
		"2251799813685253": {
			slowProcessAnalysisFixtureElement("2251799813685253", "2251799813685309", "ReserveStock", start, start.Add(30*time.Second)),
		},
	}
	piAPI := stubProcessInstanceAPI{
		searchPage: func(_ context.Context, _ d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			return d.ProcessInstancePage{Request: page, OverflowState: d.ProcessInstanceOverflowStateNoMore, Items: roots}, nil
		},
	}
	elementAPI := stubSlowProcessAnalysisElementAPI{
		search: func(_ context.Context, query d.ElementSearchQuery, _ ...services.CallOption) (d.ElementSearchResult, error) {
			return d.ElementSearchResult{Items: elementSets[query.ProcessInstanceKey]}, nil
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, elementAPI, toolx.V89).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeProcessDefinitionSearch,
		ProcessDefinitionSelector: d.SlowProcessAnalysisProcessDefinitionSelector{
			BpmnProcessID: "OrderProcess",
		},
		CapturedNow: captured,
	})

	require.NoError(t, err)
	require.Equal(t, []string{"2251799813685253", "2251799813685249", "2251799813685250", "2251799813685251", "2251799813685252"}, []string{got.Items[0].Key, got.Items[1].Key, got.Items[2].Key, got.Items[3].Key, got.Items[4].Key})
	require.Zero(t, got.Items[0].ComparisonSampleCount)
	require.Zero(t, got.Items[0].RelativePercentile)
	require.Empty(t, got.Items[0].RelativeBar)
	require.Equal(t, 4, got.Items[1].ComparisonSampleCount)
	require.Equal(t, 88, got.Items[1].RelativePercentile)
	require.Equal(t, "[#########-]", got.Items[1].RelativeBar)
	require.Equal(t, 4, got.Items[2].ComparisonSampleCount)
	require.Equal(t, 50, got.Items[2].RelativePercentile)
	require.Equal(t, got.Items[2].RelativePercentile, got.Items[3].RelativePercentile)
	require.Equal(t, got.Items[2].RelativeBar, got.Items[3].RelativeBar)
	require.Equal(t, 4, got.Items[4].ComparisonSampleCount)
	require.Equal(t, 13, got.Items[4].RelativePercentile)
	require.Equal(t, "[#---------]", got.Items[4].RelativeBar)

	root := got.Items[1]
	elementRows := slowProcessAnalysisTimelineElements(root.Timeline)
	transitions := slowProcessAnalysisTimelineTransitions(root.Timeline)
	require.Equal(t, 4, elementRows[0].ComparisonSampleCount)
	require.Equal(t, 88, elementRows[0].RelativePercentile)
	require.Equal(t, "[#########-]", elementRows[0].RelativeBar)
	require.Equal(t, 4, transitions[0].ComparisonSampleCount)
	require.Equal(t, 88, transitions[0].RelativePercentile)
	require.Equal(t, "[#########-]", transitions[0].RelativeBar)
}

// TestSlowProcessAnalysisComparisonIndicatorsSurviveDetailFiltering verifies metrics are calculated before visibility filters.
func TestSlowProcessAnalysisComparisonIndicatorsSurviveDetailFiltering(t *testing.T) {
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	roots := []d.ProcessInstance{
		slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(10*time.Minute)),
		slowProcessAnalysisFixtureProcessInstance("2251799813685250", start, start.Add(5*time.Minute)),
		slowProcessAnalysisFixtureProcessInstance("2251799813685251", start, start.Add(time.Minute)),
	}
	elementSets := map[string][]d.Element{
		"2251799813685249": {
			slowProcessAnalysisFixtureElement("2251799813685249", "2251799813685301", "ReserveStock", start, start.Add(9*time.Second)),
			slowProcessAnalysisFixtureElement("2251799813685249", "2251799813685302", "OrderFinished", start.Add(20*time.Second), start.Add(30*time.Second)),
		},
		"2251799813685250": {
			slowProcessAnalysisFixtureElement("2251799813685250", "2251799813685303", "ReserveStock", start, start.Add(4*time.Second)),
			slowProcessAnalysisFixtureElement("2251799813685250", "2251799813685304", "OrderFinished", start.Add(10*time.Second), start.Add(20*time.Second)),
		},
		"2251799813685251": {
			slowProcessAnalysisFixtureElement("2251799813685251", "2251799813685305", "ReserveStock", start, start.Add(time.Second)),
			slowProcessAnalysisFixtureElement("2251799813685251", "2251799813685306", "OrderFinished", start.Add(3*time.Second), start.Add(4*time.Second)),
		},
	}
	piAPI := stubProcessInstanceAPI{
		searchPage: func(_ context.Context, _ d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			return d.ProcessInstancePage{Request: page, OverflowState: d.ProcessInstanceOverflowStateNoMore, Items: roots}, nil
		},
	}
	elementAPI := stubSlowProcessAnalysisElementAPI{
		search: func(_ context.Context, query d.ElementSearchQuery, _ ...services.CallOption) (d.ElementSearchResult, error) {
			return d.ElementSearchResult{Items: elementSets[query.ProcessInstanceKey]}, nil
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, elementAPI, toolx.V88).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeProcessDefinitionSearch,
		ProcessDefinitionSelector: d.SlowProcessAnalysisProcessDefinitionSelector{
			BpmnProcessID: "OrderProcess",
		},
		DetailFilters: d.SlowProcessAnalysisDetailFilters{ElementID: "OrderFinished"},
		CapturedNow:   start.Add(30 * time.Minute),
	})

	require.NoError(t, err)
	require.Len(t, got.Items[0].Timeline, 2)
	require.Equal(t, d.SlowProcessAnalysisTimelineEntryKindTransition, got.Items[0].Timeline[0].Kind)
	require.Equal(t, 3, got.Items[0].Timeline[0].ComparisonSampleCount)
	require.Equal(t, 83, got.Items[0].Timeline[0].RelativePercentile)
	require.Equal(t, d.SlowProcessAnalysisTimelineEntryKindElement, got.Items[0].Timeline[1].Kind)
	require.Equal(t, "OrderFinished", got.Items[0].Timeline[1].ElementID)
}

// slowProcessAnalysisTimelineElements extracts element rows from a mixed timeline for assertions.
func slowProcessAnalysisTimelineElements(entries []d.SlowProcessAnalysisTimelineEntry) []d.SlowProcessAnalysisTimelineEntry {
	out := make([]d.SlowProcessAnalysisTimelineEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Kind == d.SlowProcessAnalysisTimelineEntryKindElement {
			out = append(out, entry)
		}
	}
	return out
}

// slowProcessAnalysisTimelineTransitions extracts transition rows from a mixed timeline for assertions.
func slowProcessAnalysisTimelineTransitions(entries []d.SlowProcessAnalysisTimelineEntry) []d.SlowProcessAnalysisTimelineEntry {
	out := make([]d.SlowProcessAnalysisTimelineEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Kind == d.SlowProcessAnalysisTimelineEntryKindTransition {
			out = append(out, entry)
		}
	}
	return out
}

// receiveSlowProcessAnalysisResult bounds tests that intentionally block slow-analysis lookup workers behind a release gate.
func receiveSlowProcessAnalysisResult(t *testing.T, done <-chan struct {
	result d.SlowProcessAnalysisResult
	err    error
}) struct {
	result d.SlowProcessAnalysisResult
	err    error
} {
	t.Helper()
	select {
	case out := <-done:
		return out
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for slow-process analysis")
		return struct {
			result d.SlowProcessAnalysisResult
			err    error
		}{}
	}
}

// ptrDomainInt64 returns a pointer for compact domain progress expectations.
func ptrDomainInt64(value int64) *int64 {
	return &value
}

// stubSlowProcessAnalysisElementAPI provides runtime elements to slow-analysis service tests.
type stubSlowProcessAnalysisElementAPI struct {
	esvc.API
	search func(context.Context, d.ElementSearchQuery, ...services.CallOption) (d.ElementSearchResult, error)
}

// SearchElements delegates runtime element search to the configured test callback.
func (s stubSlowProcessAnalysisElementAPI) SearchElements(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error) {
	if s.search == nil {
		return d.ElementSearchResult{}, nil
	}
	return s.search(ctx, query, opts...)
}

// stubSlowProcessAnalysisJobAPI provides listener jobs to slow-analysis service tests.
type stubSlowProcessAnalysisJobAPI struct {
	jsvc.API
	search func(context.Context, d.JobSearchQuery, ...services.CallOption) (d.JobSearchResult, error)
}

// SearchJobs delegates runtime job search to the configured test callback.
func (s stubSlowProcessAnalysisJobAPI) SearchJobs(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (d.JobSearchResult, error) {
	if s.search == nil {
		return d.JobSearchResult{}, nil
	}
	return s.search(ctx, query, opts...)
}
