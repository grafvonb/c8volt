// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeActivitySink struct {
	mu      sync.Mutex
	started int
	stopped int
	msgs    []string
}

// StartActivity records activity starts for facade activity-indicator assertions.
func (s *fakeActivitySink) StartActivity(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.started++
	s.msgs = append(s.msgs, msg)
}

// StopActivity records activity stops for facade activity-indicator assertions.
func (s *fakeActivitySink) StopActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped++
}

// TestClient_GetProcessDefinitionXML verifies facade option translation for
// XML retrieval. The key and call options must pass through unchanged because
// the service layer owns the Camunda-specific request details.
func TestClient_GetProcessDefinitionXML(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pdAPI := &stubProcessDefinitionAPI{
		getProcessDefinitionXML: func(_ context.Context, key string, opts ...services.CallOption) (string, error) {
			cfg := services.ApplyCallOptions(opts)
			assert.Equal(t, "2251799813685255", key)
			assert.True(t, cfg.Verbose)
			assert.True(t, cfg.WithStat)
			return "<definitions id=\"order-process\"/>", nil
		},
	}

	cli := New(pdAPI, stubProcessInstanceAPI{}, slog.Default())
	xml, err := cli.GetProcessDefinitionXML(ctx, "2251799813685255", options.WithVerbose(), options.WithStat())

	require.NoError(t, err)
	assert.Equal(t, "<definitions id=\"order-process\"/>", xml)
}

// TestClient_GetProcessDefinition_MapsIncidentCountSupportState protects the
// distinction between an incident count of zero and a version where incident
// counts are actually supported.
func TestClient_GetProcessDefinition_MapsIncidentCountSupportState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pdAPI := &stubProcessDefinitionAPI{
		getProcessDefinition: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
			cfg := services.ApplyCallOptions(opts)
			assert.Equal(t, "2251799813685255", key)
			assert.True(t, cfg.WithStat)
			return d.ProcessDefinition{
				Key: "2251799813685255",
				Statistics: &d.ProcessDefinitionStatistics{
					Active:                 7,
					Canceled:               3,
					Completed:              11,
					Incidents:              0,
					IncidentCountSupported: true,
				},
			}, nil
		},
	}

	cli := New(pdAPI, stubProcessInstanceAPI{}, slog.Default())
	pd, err := cli.GetProcessDefinition(ctx, "2251799813685255", options.WithStat())

	require.NoError(t, err)
	require.NotNil(t, pd.Statistics)
	assert.Equal(t, int64(7), pd.Statistics.Active)
	assert.Equal(t, int64(0), pd.Statistics.Incidents)
	assert.True(t, pd.Statistics.IncidentCountSupported)
}

// TestClient_SearchProcessDefinitions_PreservesUnsupportedIncidentCountBoundary
// keeps the public model from implying that zero incidents were measured when a
// Camunda version cannot provide incident-count statistics.
func TestClient_SearchProcessDefinitions_PreservesUnsupportedIncidentCountBoundary(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pdAPI := &stubProcessDefinitionAPI{
		searchProcessDefinitions: func(_ context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
			cfg := services.ApplyCallOptions(opts)
			assert.Equal(t, d.ProcessDefinitionFilter{BpmnProcessId: "order-process"}, filter)
			assert.Equal(t, pdsvc.MaxResultSize, size)
			assert.True(t, cfg.WithStat)
			return []d.ProcessDefinition{
				{
					Key:           "2251799813685255",
					BpmnProcessId: "order-process",
					Statistics: &d.ProcessDefinitionStatistics{
						Active:                 7,
						Canceled:               3,
						Completed:              11,
						Incidents:              0,
						IncidentCountSupported: false,
					},
				},
			}, nil
		},
	}

	cli := New(pdAPI, stubProcessInstanceAPI{}, slog.Default())
	items, err := cli.SearchProcessDefinitions(ctx, ProcessDefinitionFilter{BpmnProcessId: "order-process"}, options.WithStat())

	require.NoError(t, err)
	require.Len(t, items.Items, 1)
	require.NotNil(t, items.Items[0].Statistics)
	assert.Equal(t, int64(7), items.Items[0].Statistics.Active)
	assert.Equal(t, int64(0), items.Items[0].Statistics.Incidents)
	assert.False(t, items.Items[0].Statistics.IncidentCountSupported)
}

// TestClient_SearchProcessInstances_MapsDateBoundsToDomainFilter verifies that
// facade filters, presence booleans, and verbose options reach the domain layer
// without losing exact date-bound strings.
func TestClient_SearchProcessInstances_MapsDateBoundsToDomainFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	hasParent := new(false)
	hasIncident := new(true)
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstances: func(_ context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error) {
			assert.Equal(t, int32(25), size)
			assert.Equal(t, d.ProcessInstanceFilter{
				BpmnProcessId:        "order-process",
				ProcessDefinitionKey: "2251799813685255",
				StartDateAfter:       "2026-01-01",
				StartDateBefore:      "2026-01-31",
				EndDateAfter:         "2026-02-01",
				EndDateBefore:        "2026-02-28",
				State:                d.StateCompleted,
				ParentKey:            "12345",
				HasParent:            hasParent,
				HasIncident:          hasIncident,
			}, filter)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return []d.ProcessInstance{}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	_, err := cli.SearchProcessInstances(ctx, ProcessInstanceFilter{
		BpmnProcessId:        "order-process",
		ProcessDefinitionKey: "2251799813685255",
		StartDateAfter:       "2026-01-01",
		StartDateBefore:      "2026-01-31",
		EndDateAfter:         "2026-02-01",
		EndDateBefore:        "2026-02-28",
		State:                StateCompleted,
		ParentKey:            "12345",
		HasParent:            hasParent,
		HasIncident:          hasIncident,
	}, 25, options.WithVerbose())

	require.NoError(t, err)
}

// TestClient_SearchProcessInstances_PreservesDerivedRelativeDayBoundsAsCanonicalDateFields
// documents that relative-day CLI handling happens before this facade call.
// Once dates arrive here they are canonical absolute strings and must not be
// recomputed or interpreted again.
func TestClient_SearchProcessInstances_PreservesDerivedRelativeDayBoundsAsCanonicalDateFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstances: func(_ context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error) {
			assert.Equal(t, int32(10), size)
			assert.Equal(t, d.ProcessInstanceFilter{
				StartDateAfter:  "2026-03-11",
				StartDateBefore: "2026-04-03",
				EndDateAfter:    "2026-02-09",
				EndDateBefore:   "2026-03-27",
			}, filter)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return []d.ProcessInstance{}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	_, err := cli.SearchProcessInstances(ctx, ProcessInstanceFilter{
		StartDateAfter:  "2026-03-11",
		StartDateBefore: "2026-04-03",
		EndDateAfter:    "2026-02-09",
		EndDateBefore:   "2026-03-27",
	}, 10, options.WithVerbose())

	require.NoError(t, err)
}

// TestClient_SearchProcessInstancesPage_MapsPagingMetadata checks the full page
// contract: request echoing, overflow state, reported total kind, and item
// mapping all need to survive the facade boundary.
func TestClient_SearchProcessInstancesPage_MapsPagingMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			assert.Equal(t, d.ProcessInstanceFilter{BpmnProcessId: "order-process"}, filter)
			assert.Equal(t, d.ProcessInstancePageRequest{From: 25, Size: 10}, page)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateIndeterminate,
				ReportedTotal: &d.ProcessInstanceReportedTotal{
					Count: 17,
					Kind:  d.ProcessInstanceReportedTotalKindExact,
				},
				Items: []d.ProcessInstance{
					{Key: "2251799813711967", BpmnProcessId: "order-process"},
				},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	page, err := cli.SearchProcessInstancesPage(ctx, ProcessInstanceFilter{
		BpmnProcessId: "order-process",
	}, ProcessInstancePageRequest{From: 25, Size: 10}, options.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, ProcessInstancePageRequest{From: 25, Size: 10}, page.Request)
	assert.Equal(t, ProcessInstanceOverflowStateIndeterminate, page.OverflowState)
	require.NotNil(t, page.ReportedTotal)
	assert.Equal(t, int64(17), page.ReportedTotal.Count)
	assert.Equal(t, ProcessInstanceReportedTotalKindExact, page.ReportedTotal.Kind)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "2251799813711967", page.Items[0].Key)
}

// TestClient_SearchProcessInstancesPage_LeavesReportedTotalNilWhenUnavailable
// preserves the "unknown total" state. Nil is meaningful here and must not be
// converted into a misleading zero total.
func TestClient_SearchProcessInstancesPage_LeavesReportedTotalNilWhenUnavailable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			assert.Equal(t, d.ProcessInstanceFilter{BpmnProcessId: "order-process"}, filter)
			assert.Equal(t, d.ProcessInstancePageRequest{Size: 1}, page)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateNoMore,
				Items: []d.ProcessInstance{
					{Key: "2251799813711967", BpmnProcessId: "order-process"},
				},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	page, err := cli.SearchProcessInstancesPage(ctx, ProcessInstanceFilter{
		BpmnProcessId: "order-process",
	}, ProcessInstancePageRequest{Size: 1})

	require.NoError(t, err)
	assert.Nil(t, page.ReportedTotal)
	require.Len(t, page.Items, 1)
}

// TestClient_SearchProcessInstancesPage_MapsLowerBoundReportedTotal protects
// Camunda's lower-bound total semantics, where large result sets report "at
// least this many" rather than an exact count.
func TestClient_SearchProcessInstancesPage_MapsLowerBoundReportedTotal(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			assert.Equal(t, d.ProcessInstanceFilter{BpmnProcessId: "order-process"}, filter)
			assert.Equal(t, d.ProcessInstancePageRequest{From: 100, Size: 25}, page)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateHasMore,
				ReportedTotal: &d.ProcessInstanceReportedTotal{
					Count: 10000,
					Kind:  d.ProcessInstanceReportedTotalKindLowerBound,
				},
				Items: []d.ProcessInstance{
					{Key: "2251799813711967", BpmnProcessId: "order-process"},
				},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	page, err := cli.SearchProcessInstancesPage(ctx, ProcessInstanceFilter{
		BpmnProcessId: "order-process",
	}, ProcessInstancePageRequest{From: 100, Size: 25}, options.WithVerbose())

	require.NoError(t, err)
	require.NotNil(t, page.ReportedTotal)
	assert.Equal(t, int64(10000), page.ReportedTotal.Count)
	assert.Equal(t, ProcessInstanceReportedTotalKindLowerBound, page.ReportedTotal.Kind)
	require.Len(t, page.Items, 1)
}

// TestClient_SearchProcessInstancesPage_MapsPresenceFiltersToDomainFilter
// verifies pointer-backed booleans for has-parent and has-incident filters.
// Pointer identity matters because nil means "unspecified" while false means
// an explicit negative filter.
func TestClient_SearchProcessInstancesPage_MapsPresenceFiltersToDomainFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	hasParent := new(true)
	hasIncident := new(false)
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			assert.Equal(t, d.ProcessInstanceFilter{
				HasParent:   hasParent,
				HasIncident: hasIncident,
			}, filter)
			assert.Equal(t, d.ProcessInstancePageRequest{From: 5, Size: 10}, page)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateNoMore,
				Items: []d.ProcessInstance{
					{Key: "2251799813711967", BpmnProcessId: "order-process"},
				},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	page, err := cli.SearchProcessInstancesPage(ctx, ProcessInstanceFilter{
		HasParent:   hasParent,
		HasIncident: hasIncident,
	}, ProcessInstancePageRequest{From: 5, Size: 10}, options.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, ProcessInstancePageRequest{From: 5, Size: 10}, page.Request)
	require.Len(t, page.Items, 1)
}

// TestClient_SearchProcessInstancesPage_PreservesCrossVersionOverflowStates
// ensures overflow state remains a cross-version domain signal instead of being
// flattened by the facade into item-count heuristics.
func TestClient_SearchProcessInstancesPage_PreservesCrossVersionOverflowStates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			assert.Equal(t, d.ProcessInstanceFilter{BpmnProcessId: "order-process"}, filter)
			assert.Equal(t, d.ProcessInstancePageRequest{Size: 2}, page)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateNoMore,
				Items: []d.ProcessInstance{
					{Key: "2251799813711967", BpmnProcessId: "order-process"},
				},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	page, err := cli.SearchProcessInstancesPage(ctx, ProcessInstanceFilter{
		BpmnProcessId: "order-process",
	}, ProcessInstancePageRequest{Size: 2})

	require.NoError(t, err)
	assert.Equal(t, ProcessInstanceOverflowStateNoMore, page.OverflowState)
	require.Len(t, page.Items, 1)
}

// TestClient_SearchProcessInstances_UsesPagedSearchWrapper documents the legacy
// list facade over the newer paged service call. Total is derived from returned
// items here, while page metadata remains available through the page API.
func TestClient_SearchProcessInstances_UsesPagedSearchWrapper(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			assert.Equal(t, d.ProcessInstanceFilter{BpmnProcessId: "order-process"}, filter)
			assert.Equal(t, d.ProcessInstancePageRequest{Size: 2}, page)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateHasMore,
				Items: []d.ProcessInstance{
					{Key: "2251799813711967", BpmnProcessId: "order-process"},
					{Key: "2251799813711968", BpmnProcessId: "order-process"},
				},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	items, err := cli.SearchProcessInstances(ctx, ProcessInstanceFilter{
		BpmnProcessId: "order-process",
	}, 2, options.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, int32(2), items.Total)
	require.Len(t, items.Items, 2)
	assert.Equal(t, "2251799813711967", items.Items[0].Key)
	assert.Equal(t, "2251799813711968", items.Items[1].Key)
}

// TestClient_LookupProcessInstance_UsesSearchBackedLookup protects the lookup
// strategy for versions where direct key retrieval is implemented through a
// tenant-aware search filter.
func TestClient_LookupProcessInstance_UsesSearchBackedLookup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstances: func(_ context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error) {
			assert.Equal(t, d.ProcessInstanceFilter{Key: "2251799813711967"}, filter)
			assert.Equal(t, int32(2), size)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return []d.ProcessInstance{
				{Key: "2251799813711967", State: d.StateActive, TenantId: "tenant-a"},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	pi, err := cli.LookupProcessInstance(ctx, "2251799813711967", options.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, "2251799813711967", pi.Key)
	assert.Equal(t, StateActive, pi.State)
	assert.Equal(t, "tenant-a", pi.TenantId)
}

// TestClient_LookupProcessInstanceStateByKey_MapsSearchBackedState verifies the
// public state report created from a search-backed lookup, including the stable
// uppercase status string used by command output.
func TestClient_LookupProcessInstanceStateByKey_MapsSearchBackedState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstances: func(_ context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error) {
			assert.Equal(t, d.ProcessInstanceFilter{Key: "2251799813711967"}, filter)
			assert.Equal(t, int32(2), size)
			return []d.ProcessInstance{
				{Key: "2251799813711967", State: d.StateCompleted, TenantId: "tenant-a"},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	report, pi, err := cli.LookupProcessInstanceStateByKey(ctx, "2251799813711967")

	require.NoError(t, err)
	assert.Equal(t, StateCompleted, report.State)
	assert.Equal(t, "COMPLETED", report.Status)
	assert.Equal(t, "2251799813711967", report.Key)
	assert.Equal(t, "2251799813711967", pi.Key)
	assert.Equal(t, StateCompleted, pi.State)
}

// TestClient_DryRunCancelOrDeleteGetPIKeys_DeduplicatesRootsAndCollected covers
// the legacy dry-run expansion contract. Multiple selected children can share a
// root, so roots and collected keys must keep deterministic first-seen order.
func TestClient_DryRunCancelOrDeleteGetPIKeys_DeduplicatesRootsAndCollected(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		ancestry: func(_ context.Context, startKey string, _ ...services.CallOption) (string, []string, map[string]d.ProcessInstance, error) {
			switch startKey {
			case "c1", "c2":
				return "r1", nil, nil, nil
			case "c3":
				return "r2", nil, nil, nil
			default:
				t.Fatalf("unexpected key %q", startKey)
				return "", nil, nil, nil
			}
		},
		descendants: func(_ context.Context, rootKey string, _ ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error) {
			switch rootKey {
			case "r1":
				return []string{"r1", "c1", "c2"}, nil, nil, nil
			case "r2":
				return []string{"r2", "c3"}, nil, nil, nil
			default:
				t.Fatalf("unexpected root %q", rootKey)
				return nil, nil, nil, nil
			}
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	roots, collected, err := cli.DryRunCancelOrDeleteGetPIKeys(ctx, typex.Keys{"c1", "c2", "c3"})

	require.NoError(t, err)
	assert.Equal(t, typex.Keys{"r1", "r2"}, roots)
	assert.Equal(t, typex.Keys{"r1", "c1", "c2", "r2", "c3"}, collected)
}

// TestClient_AncestryResult_MapsStructuredTraversalContract verifies the newer
// structured traversal model, including partial outcomes and missing-ancestor
// metadata used by automation and warning output.
func TestClient_AncestryResult_MapsStructuredTraversalContract(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		ancestryResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			assert.Equal(t, "child", startKey)
			return pitraversal.Result{
				Mode:     pitraversal.ModeAncestry,
				StartKey: "child",
				RootKey:  "child",
				Keys:     []string{"child"},
				Chain: map[string]d.ProcessInstance{
					"child": {Key: "child", ParentKey: "missing"},
				},
				MissingAncestors: []pitraversal.MissingAncestor{{Key: "missing", StartKey: "child"}},
				Warning:          "one or more parent process instances were not found",
				Outcome:          pitraversal.OutcomePartial,
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	got, err := cli.AncestryResult(ctx, "child")

	require.NoError(t, err)
	assert.Equal(t, TraversalModeAncestry, got.Mode)
	assert.Equal(t, "child", got.StartKey)
	assert.Equal(t, "child", got.RootKey)
	assert.Equal(t, []string{"child"}, got.Keys)
	assert.Equal(t, "missing", got.MissingAncestors[0].Key)
	assert.Equal(t, "child", got.Chain["child"].Key)
	assert.Equal(t, TraversalOutcomePartial, got.Outcome)
}

// TestClient_DryRunCancelOrDeletePlan_ReturnsStructuredExpansion exercises the
// full cancellation/deletion preflight: selected children resolve to roots,
// descendants expand per root, and partial ancestry warnings are preserved.
func TestClient_DryRunCancelOrDeletePlan_ReturnsStructuredExpansion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		ancestryResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			switch startKey {
			case "c1":
				return pitraversal.Result{
					Mode:             pitraversal.ModeAncestry,
					StartKey:         "c1",
					RootKey:          "r1",
					Keys:             []string{"c1", "r1"},
					Chain:            map[string]d.ProcessInstance{"c1": {Key: "c1", State: d.StateCanceled}, "r1": {Key: "r1", State: d.StateActive}},
					MissingAncestors: []pitraversal.MissingAncestor{{Key: "missing", StartKey: "c1"}},
					Warning:          "one or more parent process instances were not found",
					Outcome:          pitraversal.OutcomePartial,
				}, nil
			case "c2":
				return pitraversal.Result{
					Mode:     pitraversal.ModeAncestry,
					StartKey: "c2",
					RootKey:  "r2",
					Keys:     []string{"c2", "r2"},
					Chain:    map[string]d.ProcessInstance{"c2": {Key: "c2", State: d.StateActive}, "r2": {Key: "r2", State: d.StateActive}},
					Outcome:  pitraversal.OutcomeComplete,
				}, nil
			default:
				t.Fatalf("unexpected key %q", startKey)
				return pitraversal.Result{}, nil
			}
		},
		descendantsResult: func(_ context.Context, rootKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			switch rootKey {
			case "r1":
				return pitraversal.Result{
					Mode:    pitraversal.ModeDescendants,
					RootKey: "r1",
					Keys:    []string{"r1", "c1"},
					Chain:   map[string]d.ProcessInstance{"r1": {Key: "r1", State: d.StateActive}, "c1": {Key: "c1", State: d.StateCanceled}},
					Outcome: pitraversal.OutcomeComplete,
				}, nil
			case "r2":
				return pitraversal.Result{
					Mode:    pitraversal.ModeDescendants,
					RootKey: "r2",
					Keys:    []string{"r2", "c2"},
					Chain:   map[string]d.ProcessInstance{"r2": {Key: "r2", State: d.StateActive}, "c2": {Key: "c2", State: d.StateActive}},
					Outcome: pitraversal.OutcomeComplete,
				}, nil
			default:
				t.Fatalf("unexpected root %q", rootKey)
				return pitraversal.Result{}, nil
			}
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	got, err := cli.DryRunCancelOrDeletePlan(ctx, typex.Keys{"c1", "c2"})

	require.NoError(t, err)
	assert.Equal(t, typex.Keys{"r1", "r2"}, got.Roots)
	assert.Equal(t, typex.Keys{"r1", "c1", "r2", "c2"}, got.Collected)
	assert.Equal(t, []MissingAncestor{{Key: "missing", StartKey: "c1"}}, got.MissingAncestors)
	assert.Equal(t, []ProcessInstance{{Key: "c1", State: StateCanceled}}, got.SelectedFinalState)
	assert.Equal(t, []ProcessInstance{
		{Key: "r1", State: StateActive},
		{Key: "r2", State: StateActive},
		{Key: "c2", State: StateActive},
	}, got.RequiresCancelBeforeDelete)
	assert.Equal(t, TraversalOutcomePartial, got.Outcome)
	assert.NotEmpty(t, got.Warning)
}

// TestClient_DryRunCancelOrDeletePlan_FailsWhenNoActionableResultsResolve keeps
// unresolved ancestry from silently producing an empty successful plan. That
// protects destructive commands from proceeding when nothing can be targeted.
func TestClient_DryRunCancelOrDeletePlan_FailsWhenNoActionableResultsResolve(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		ancestryResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			assert.Equal(t, "c1", startKey)
			return pitraversal.Result{
				Mode:             pitraversal.ModeAncestry,
				StartKey:         "c1",
				MissingAncestors: []pitraversal.MissingAncestor{{Key: "missing", StartKey: "c1"}},
				Warning:          "one or more parent process instances were not found",
				Outcome:          pitraversal.OutcomeUnresolved,
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	_, err := cli.DryRunCancelOrDeletePlan(ctx, typex.Keys{"c1"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no process instances resolved during dependency expansion")
}

// TestClient_CancelProcessInstances_LogsExpandedAffectedScope verifies cancel
// logs describe expanded dry-run scope counts.
func TestClient_CancelProcessInstances_LogsExpandedAffectedScope(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var logBuf bytes.Buffer
	piAPI := stubProcessInstanceAPI{
		cancelProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
			assert.Equal(t, "root-1", key)
			return d.CancelResponse{Ok: true, StatusCode: 202, Status: "202 Accepted"}, nil, nil
		},
	}
	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.New(logging.NewPlainHandler(&logBuf, slog.LevelDebug)))

	reports, err := cli.CancelProcessInstances(ctx, typex.Keys{"root-1"}, 0, options.WithAffectedProcessInstanceCount(4), options.WithVerbose())

	require.NoError(t, err)
	require.Len(t, reports.Items, 1)
	assert.Contains(t, logBuf.String(), "cancelling process instances requested for 4 affected instance(s) across 1 root key(s)")
	assert.Contains(t, logBuf.String(), "cancelling 4 process instance(s) completed via 1 root request(s): 1 root request(s) succeeded or already cancelled/terminated, 0 failed")
	assert.NotContains(t, logBuf.String(), "cancelling 1 process instance(s) completed")
}

// TestClient_CancelProcessInstances_UsesActivityIndicator verifies bulk cancel emits activity lifecycle messages.
func TestClient_CancelProcessInstances_UsesActivityIndicator(t *testing.T) {
	t.Parallel()

	sink := &fakeActivitySink{}
	ctx := logging.ToActivityContext(context.Background(), sink)
	piAPI := stubProcessInstanceAPI{
		cancelProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
			assert.Equal(t, "root-1", key)
			return d.CancelResponse{Ok: true, StatusCode: 202, Status: "202 Accepted"}, nil, nil
		},
	}
	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())

	_, err := cli.CancelProcessInstances(ctx, typex.Keys{"root-1"}, 0, options.WithAffectedProcessInstanceCount(4))

	require.NoError(t, err)
	sink.mu.Lock()
	defer sink.mu.Unlock()
	assert.Equal(t, 1, sink.started)
	assert.Equal(t, 1, sink.stopped)
	assert.Equal(t, []string{"cancelling 4 process instance(s) via 1 root request(s)"}, sink.msgs)
}

// TestClient_DeleteProcessInstances_LogsExpandedAffectedScope verifies delete
// logs describe expanded dry-run scope counts.
func TestClient_DeleteProcessInstances_LogsExpandedAffectedScope(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var logBuf bytes.Buffer
	piAPI := stubProcessInstanceAPI{
		deleteProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			assert.Equal(t, "root-1", key)
			return d.DeleteResponse{Ok: true, StatusCode: 204, Status: "204 No Content"}, nil
		},
	}
	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.New(logging.NewPlainHandler(&logBuf, slog.LevelDebug)))

	reports, err := cli.DeleteProcessInstances(ctx, typex.Keys{"root-1"}, 0, options.WithAffectedProcessInstanceCount(4), options.WithVerbose())

	require.NoError(t, err)
	require.Len(t, reports.Items, 1)
	assert.Contains(t, logBuf.String(), "deleting process instances requested for 4 affected instance(s) across 1 root key(s)")
	assert.Contains(t, logBuf.String(), "deleting 4 process instance(s) completed via 1 root request(s): 1 root request(s) succeeded, 0 failed")
	assert.NotContains(t, logBuf.String(), "deleting 1 process instances completed")
}

// TestClient_DeleteProcessInstances_LogsConsolidatedWrongStateForExpandedScope
// verifies delete failures summarize non-terminal expanded scope.
func TestClient_DeleteProcessInstances_LogsConsolidatedWrongStateForExpandedScope(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var logBuf bytes.Buffer
	piAPI := stubProcessInstanceAPI{
		deleteProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			assert.Equal(t, "root-1", key)
			return d.DeleteResponse{StatusCode: 409, Status: "409 Conflict"}, nil
		},
	}
	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.New(logging.NewPlainHandler(&logBuf, slog.LevelDebug)))

	reports, err := cli.DeleteProcessInstances(ctx, typex.Keys{"root-1"}, 0, options.WithAffectedProcessInstanceCount(4))

	require.NoError(t, err)
	require.Len(t, reports.Items, 1)
	assert.Contains(t, logBuf.String(), "cannot delete expanded process-instance scope of 4 process instance(s): one or more affected process instances are not in a terminated state; use --force flag to cancel and then delete them")
	assert.Contains(t, logBuf.String(), "deleting 4 process instance(s) completed via 1 root request(s): 0 root request(s) succeeded, 1 failed")
}

type stubProcessDefinitionAPI struct {
	searchProcessDefinitions func(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error)
	getProcessDefinition     func(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error)
	getProcessDefinitionXML  func(ctx context.Context, key string, opts ...services.CallOption) (string, error)
}

// SearchProcessDefinitions delegates to the per-test callback and panics when a
// test did not authorize this service method.
func (s *stubProcessDefinitionAPI) SearchProcessDefinitions(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	if s.searchProcessDefinitions == nil {
		panic("unexpected call")
	}
	return s.searchProcessDefinitions(ctx, filter, size, opts...)
}

// SearchProcessDefinitionsLatest panics when a facade test accidentally takes the latest-definition path.
func (s *stubProcessDefinitionAPI) SearchProcessDefinitionsLatest(context.Context, d.ProcessDefinitionFilter, ...services.CallOption) ([]d.ProcessDefinition, error) {
	panic("unexpected call")
}

// GetProcessDefinition delegates to the per-test callback and panics on
// unexpected lookups to keep facade tests narrowly scoped.
func (s *stubProcessDefinitionAPI) GetProcessDefinition(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
	if s.getProcessDefinition == nil {
		panic("unexpected call")
	}
	return s.getProcessDefinition(ctx, key, opts...)
}

// GetProcessDefinitionXML delegates to the per-test callback for XML-specific
// facade assertions.
func (s *stubProcessDefinitionAPI) GetProcessDefinitionXML(ctx context.Context, key string, opts ...services.CallOption) (string, error) {
	if s.getProcessDefinitionXML == nil {
		panic("unexpected call")
	}
	return s.getProcessDefinitionXML(ctx, key, opts...)
}

var _ pdsvc.API = (*stubProcessDefinitionAPI)(nil)

type stubProcessInstanceAPI struct {
	searchForProcessInstances     func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error)
	searchForProcessInstancesPage func(context.Context, d.ProcessInstanceFilter, d.ProcessInstancePageRequest, ...services.CallOption) (d.ProcessInstancePage, error)
	ancestry                      func(context.Context, string, ...services.CallOption) (string, []string, map[string]d.ProcessInstance, error)
	descendants                   func(context.Context, string, ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error)
	cancelProcessInstance         func(context.Context, string, ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error)
	deleteProcessInstance         func(context.Context, string, ...services.CallOption) (d.DeleteResponse, error)
	ancestryResult                func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
	descendantsResult             func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
	familyResult                  func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
}

// CreateProcessInstance panics when a facade test accidentally starts a process instance.
func (stubProcessInstanceAPI) CreateProcessInstance(context.Context, d.ProcessInstanceData, ...services.CallOption) (d.ProcessInstanceCreation, error) {
	panic("unexpected call")
}

// GetProcessInstance panics when a facade test accidentally performs direct lookup.
func (stubProcessInstanceAPI) GetProcessInstance(context.Context, string, ...services.CallOption) (d.ProcessInstance, error) {
	panic("unexpected call")
}

// GetDirectChildrenOfProcessInstance panics when a test takes the direct-children path unexpectedly.
func (stubProcessInstanceAPI) GetDirectChildrenOfProcessInstance(context.Context, string, ...services.CallOption) ([]d.ProcessInstance, error) {
	panic("unexpected call")
}

// FilterProcessInstanceWithOrphanParent panics when a test takes orphan filtering unexpectedly.
func (stubProcessInstanceAPI) FilterProcessInstanceWithOrphanParent(context.Context, []d.ProcessInstance, ...services.CallOption) ([]d.ProcessInstance, error) {
	panic("unexpected call")
}

// SearchForProcessInstances delegates to the per-test callback and panics when
// a test accidentally takes the legacy list path.
func (s stubProcessInstanceAPI) SearchForProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	if s.searchForProcessInstances == nil {
		panic("unexpected call")
	}
	return s.searchForProcessInstances(ctx, filter, size, opts...)
}

// SearchForProcessInstancesPage delegates to an explicit page callback when a
// test provides one, otherwise it adapts the legacy list callback for older
// facade tests that only care about returned items.
func (s stubProcessInstanceAPI) SearchForProcessInstancesPage(ctx context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
	if s.searchForProcessInstancesPage != nil {
		return s.searchForProcessInstancesPage(ctx, filter, page, opts...)
	}
	if s.searchForProcessInstances == nil {
		panic("unexpected call")
	}
	items, err := s.searchForProcessInstances(ctx, filter, page.Size, opts...)
	if err != nil {
		return d.ProcessInstancePage{}, err
	}
	return d.ProcessInstancePage{
		Request: page,
		Items:   items,
	}, nil
}

// CancelProcessInstance delegates to the per-test callback and panics on unexpected cancel calls.
func (s stubProcessInstanceAPI) CancelProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
	if s.cancelProcessInstance == nil {
		panic("unexpected call")
	}
	return s.cancelProcessInstance(ctx, key, opts...)
}

// DeleteProcessInstance delegates to the per-test callback and panics on unexpected delete calls.
func (s stubProcessInstanceAPI) DeleteProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error) {
	if s.deleteProcessInstance == nil {
		panic("unexpected call")
	}
	return s.deleteProcessInstance(ctx, key, opts...)
}

// GetProcessInstanceStateByKey panics when a facade test accidentally checks process state.
func (stubProcessInstanceAPI) GetProcessInstanceStateByKey(context.Context, string, ...services.CallOption) (d.State, d.ProcessInstance, error) {
	panic("unexpected call")
}

// WaitForProcessInstanceState panics when a facade test accidentally waits for one process instance.
func (stubProcessInstanceAPI) WaitForProcessInstanceState(context.Context, string, d.States, ...services.CallOption) (d.StateResponse, d.ProcessInstance, error) {
	panic("unexpected call")
}

// Ancestry delegates the legacy traversal tuple used by older dry-run helpers.
func (s stubProcessInstanceAPI) Ancestry(ctx context.Context, startKey string, opts ...services.CallOption) (string, []string, map[string]d.ProcessInstance, error) {
	if s.ancestry == nil {
		panic("unexpected call")
	}
	return s.ancestry(ctx, startKey, opts...)
}

// Descendants delegates the legacy traversal tuple used to expand a resolved
// root into actionable process instance keys.
func (s stubProcessInstanceAPI) Descendants(ctx context.Context, rootKey string, opts ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error) {
	if s.descendants == nil {
		panic("unexpected call")
	}
	return s.descendants(ctx, rootKey, opts...)
}

// Family panics when a test unexpectedly takes the legacy full-family path.
func (stubProcessInstanceAPI) Family(context.Context, string, ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error) {
	panic("unexpected call")
}

// AncestryResult delegates structured traversal when supplied, or builds it
// from the legacy tuple callback to keep migration tests compact.
func (s stubProcessInstanceAPI) AncestryResult(ctx context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error) {
	if s.ancestryResult != nil {
		return s.ancestryResult(ctx, startKey, opts...)
	}
	if s.ancestry == nil {
		panic("unexpected call")
	}
	return pitraversal.BuildAncestryResult(ctx, s, startKey, opts...)
}

// DescendantsResult delegates structured traversal when supplied, or builds it
// from the legacy tuple callback for tests that cover compatibility behavior.
func (s stubProcessInstanceAPI) DescendantsResult(ctx context.Context, rootKey string, opts ...services.CallOption) (pitraversal.Result, error) {
	if s.descendantsResult != nil {
		return s.descendantsResult(ctx, rootKey, opts...)
	}
	if s.descendants == nil {
		panic("unexpected call")
	}
	return pitraversal.BuildDescendantsResult(ctx, s, rootKey, opts...)
}

// FamilyResult delegates a structured family traversal when supplied, otherwise
// composes one from the legacy ancestry and descendants callbacks.
func (s stubProcessInstanceAPI) FamilyResult(ctx context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error) {
	if s.familyResult != nil {
		return s.familyResult(ctx, startKey, opts...)
	}
	return pitraversal.BuildFamilyResult(ctx, s, startKey, opts...)
}

// GetProcessInstances panics when a facade test accidentally performs bulk lookup.
func (stubProcessInstanceAPI) GetProcessInstances(context.Context, typex.Keys, int, ...services.CallOption) ([]d.ProcessInstance, error) {
	panic("unexpected call")
}

// WaitForProcessInstancesState panics when a facade test accidentally waits for bulk state changes.
func (stubProcessInstanceAPI) WaitForProcessInstancesState(context.Context, typex.Keys, d.States, int, ...services.CallOption) (d.StateResponses, error) {
	panic("unexpected call")
}

var _ pisvc.API = stubProcessInstanceAPI{}
