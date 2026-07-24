// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// TestDryRunCancelOrDeletePlan_UsesBoundedWorkersForDependencyTraversal proves
// independent ancestry and descendant expansion are not serialized during
// high-volume cancel/delete impact planning.
func TestDryRunCancelOrDeletePlan_UsesBoundedWorkersForDependencyTraversal(t *testing.T) {
	ctx := context.Background()

	var ancestryMax atomic.Int32
	var ancestryActive atomic.Int32
	var ancestryReleased atomic.Bool
	ancestryRelease := make(chan struct{})

	var descendantMax atomic.Int32
	var descendantActive atomic.Int32
	var descendantReleased atomic.Bool
	descendantRelease := make(chan struct{})

	api := stubDryRunProcessInstanceAPI{
		ancestryResult: func(_ context.Context, key string, opts ...services.CallOption) (pitraversal.Result, error) {
			cfg := services.ApplyCallOptions(opts)
			if !cfg.IgnoreTenant {
				return pitraversal.Result{}, errors.New("expected ignore tenant option")
			}
			defer waitForDryRunOverlap(t, &ancestryActive, &ancestryMax, &ancestryReleased, ancestryRelease, "ancestry")()
			return pitraversal.Result{
				Mode:     pitraversal.ModeAncestry,
				StartKey: key,
				RootKey:  "root-" + key,
				Keys:     []string{key, "root-" + key},
				Chain: map[string]d.ProcessInstance{
					key:           {Key: key, State: d.StateActive},
					"root-" + key: {Key: "root-" + key, State: d.StateActive},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
		descendantsResult: func(_ context.Context, root string, opts ...services.CallOption) (pitraversal.Result, error) {
			cfg := services.ApplyCallOptions(opts)
			if !cfg.IgnoreTenant {
				return pitraversal.Result{}, errors.New("expected ignore tenant option")
			}
			defer waitForDryRunOverlap(t, &descendantActive, &descendantMax, &descendantReleased, descendantRelease, "descendants")()
			child := root[len("root-"):]
			return pitraversal.Result{
				Mode:    pitraversal.ModeDescendants,
				RootKey: root,
				Keys:    []string{root, child},
				Chain: map[string]d.ProcessInstance{
					root:  {Key: root, State: d.StateActive},
					child: {Key: child, State: d.StateActive},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
	}

	got, err := DryRunCancelOrDeletePlan(ctx, api, typex.Keys{"child-1", "child-2"}, 2, services.WithIgnoreTenant())

	require.NoError(t, err)
	require.Equal(t, int32(2), ancestryMax.Load())
	require.Equal(t, int32(2), descendantMax.Load())
	require.Equal(t, typex.Keys{"root-child-1", "root-child-2"}, got.Roots)
	require.Equal(t, typex.Keys{"root-child-1", "child-1", "root-child-2", "child-2"}, got.Collected)
	require.Equal(t, d.TraversalOutcomeComplete, got.Outcome)
}

// TestPlanProcessInstanceMutationPages_UsesConcurrentDependencyPlanning verifies
// search-selected cancel/delete planning delegates each page to the same
// worker-bounded dependency traversal used by direct dry-run expansion.
func TestPlanProcessInstanceMutationPages_UsesConcurrentDependencyPlanning(t *testing.T) {
	ctx := context.Background()

	var ancestryMax atomic.Int32
	var ancestryActive atomic.Int32
	var ancestryReleased atomic.Bool
	ancestryRelease := make(chan struct{})

	api := stubDryRunProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			cfg := services.ApplyCallOptions(opts)
			require.True(t, cfg.FailFast)
			require.Equal(t, "order", filter.BpmnProcessId)
			require.Equal(t, int32(10), page.Size)
			return d.ProcessInstancePage{
				Items: []d.ProcessInstance{
					{Key: "child-1", BpmnProcessId: "order"},
					{Key: "child-2", BpmnProcessId: "order"},
				},
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateNoMore,
			}, nil
		},
		ancestryResult: func(_ context.Context, key string, opts ...services.CallOption) (pitraversal.Result, error) {
			cfg := services.ApplyCallOptions(opts)
			if !cfg.FailFast {
				return pitraversal.Result{}, errors.New("expected fail-fast option")
			}
			defer waitForDryRunOverlap(t, &ancestryActive, &ancestryMax, &ancestryReleased, ancestryRelease, "page ancestry")()
			return pitraversal.Result{
				Mode:     pitraversal.ModeAncestry,
				StartKey: key,
				RootKey:  "root-" + key,
				Keys:     []string{key, "root-" + key},
				Chain: map[string]d.ProcessInstance{
					key:           {Key: key, State: d.StateActive},
					"root-" + key: {Key: "root-" + key, State: d.StateActive},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
		descendantsResult: func(_ context.Context, root string, opts ...services.CallOption) (pitraversal.Result, error) {
			cfg := services.ApplyCallOptions(opts)
			if !cfg.FailFast {
				return pitraversal.Result{}, errors.New("expected fail-fast option")
			}
			child := root[len("root-"):]
			return pitraversal.Result{
				Mode:    pitraversal.ModeDescendants,
				RootKey: root,
				Keys:    []string{root, child},
				Chain: map[string]d.ProcessInstance{
					root:  {Key: root, State: d.StateActive},
					child: {Key: child, State: d.StateActive},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
	}

	var visited []d.ProcessInstanceMutationPlanStep
	got, err := PlanProcessInstanceMutationPages(ctx, api, stubDryRunIncidentAPI{}, d.ProcessInstanceMutationPlanRequest{
		SearchRequest: d.ProcessInstanceSearchRequest{
			Filter: d.ProcessInstanceFilter{BpmnProcessId: "order"},
			Page:   d.ProcessInstancePageRequest{Size: 10},
		},
		Workers: 2,
	}, func(step d.ProcessInstanceMutationPlanStep) (d.ProcessInstanceSearchPageAction, error) {
		visited = append(visited, step)
		return d.ProcessInstanceSearchPageActionContinue, nil
	}, services.WithFailFast())

	require.NoError(t, err)
	require.Equal(t, int32(2), ancestryMax.Load())
	require.Equal(t, int32(1), got.Pages)
	require.Equal(t, int32(2), got.RequestedCount)
	require.Equal(t, int32(4), got.CumulativeImpact)
	require.Len(t, got.Plans, 1)
	require.Equal(t, []string{"child-1", "child-2"}, got.Plans[0].RequestedKeys)
	require.Equal(t, typex.Keys{"root-child-1", "root-child-2"}, got.Plans[0].Plan.Roots)
	require.Len(t, visited, 1)
	require.Equal(t, got.Plans[0].Plan.Collected, visited[0].Plan.Collected)
}

func waitForDryRunOverlap(t *testing.T, active, max *atomic.Int32, released *atomic.Bool, release chan struct{}, phase string) func() {
	t.Helper()
	current := active.Add(1)
	for {
		seen := max.Load()
		if current <= seen || max.CompareAndSwap(seen, current) {
			break
		}
	}
	if current >= 2 && released.CompareAndSwap(false, true) {
		close(release)
	}
	select {
	case <-release:
	case <-time.After(2 * time.Second):
		if released.CompareAndSwap(false, true) {
			close(release)
		}
		t.Errorf("%s dependency traversal did not use concurrent workers", phase)
	}
	return func() {
		active.Add(-1)
	}
}

type stubDryRunProcessInstanceAPI struct {
	searchForProcessInstancesPage func(context.Context, d.ProcessInstanceFilter, d.ProcessInstancePageRequest, ...services.CallOption) (d.ProcessInstancePage, error)
	ancestryResult                func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
	descendantsResult             func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
}

func (s stubDryRunProcessInstanceAPI) CreateProcessInstance(context.Context, d.ProcessInstanceData, ...services.CallOption) (d.ProcessInstanceCreation, error) {
	return d.ProcessInstanceCreation{}, errors.New("unexpected create process instance")
}

func (s stubDryRunProcessInstanceAPI) GetProcessInstance(context.Context, string, ...services.CallOption) (d.ProcessInstance, error) {
	return d.ProcessInstance{}, errors.New("unexpected get process instance")
}

func (s stubDryRunProcessInstanceAPI) SearchProcessInstanceVariables(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	return nil, errors.New("unexpected search process instance variables")
}

func (s stubDryRunProcessInstanceAPI) UpdateProcessInstanceVariables(context.Context, string, map[string]any, ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error) {
	return d.ProcessInstanceVariableUpdateResponse{}, errors.New("unexpected update process instance variables")
}

func (s stubDryRunProcessInstanceAPI) GetDirectChildrenOfProcessInstance(context.Context, string, ...services.CallOption) ([]d.ProcessInstance, error) {
	return nil, errors.New("unexpected get direct children")
}

func (s stubDryRunProcessInstanceAPI) FilterProcessInstanceWithOrphanParent(context.Context, []d.ProcessInstance, ...services.CallOption) ([]d.ProcessInstance, error) {
	return nil, errors.New("unexpected orphan parent filter")
}

func (s stubDryRunProcessInstanceAPI) SearchForProcessInstancesPage(ctx context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
	if s.searchForProcessInstancesPage == nil {
		return d.ProcessInstancePage{}, errors.New("unexpected search process instances page")
	}
	return s.searchForProcessInstancesPage(ctx, filter, page, opts...)
}

func (s stubDryRunProcessInstanceAPI) SearchForProcessInstances(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
	return nil, errors.New("unexpected search process instances")
}

func (s stubDryRunProcessInstanceAPI) CancelProcessInstance(context.Context, string, ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
	return d.CancelResponse{}, nil, errors.New("unexpected cancel process instance")
}

func (s stubDryRunProcessInstanceAPI) DeleteProcessInstance(context.Context, string, ...services.CallOption) (d.DeleteResponse, error) {
	return d.DeleteResponse{}, errors.New("unexpected delete process instance")
}

func (s stubDryRunProcessInstanceAPI) GetProcessInstanceStateByKey(context.Context, string, ...services.CallOption) (d.State, d.ProcessInstance, error) {
	return "", d.ProcessInstance{}, errors.New("unexpected get process instance state")
}

func (s stubDryRunProcessInstanceAPI) WaitForProcessInstanceState(context.Context, string, d.States, ...services.CallOption) (d.StateResponse, d.ProcessInstance, error) {
	return d.StateResponse{}, d.ProcessInstance{}, errors.New("unexpected wait for process instance state")
}

func (s stubDryRunProcessInstanceAPI) WaitForProcessInstanceExpectation(context.Context, string, d.ProcessInstanceExpectationRequest, ...services.CallOption) (d.ProcessInstanceExpectationResponse, d.ProcessInstance, error) {
	return d.ProcessInstanceExpectationResponse{}, d.ProcessInstance{}, errors.New("unexpected wait for process instance expectation")
}

func (s stubDryRunProcessInstanceAPI) Ancestry(context.Context, string, ...services.CallOption) (string, []string, map[string]d.ProcessInstance, error) {
	return "", nil, nil, errors.New("unexpected ancestry")
}

func (s stubDryRunProcessInstanceAPI) Descendants(context.Context, string, ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error) {
	return nil, nil, nil, errors.New("unexpected descendants")
}

func (s stubDryRunProcessInstanceAPI) Family(context.Context, string, ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error) {
	return nil, nil, nil, errors.New("unexpected family")
}

func (s stubDryRunProcessInstanceAPI) AncestryResult(ctx context.Context, key string, opts ...services.CallOption) (pitraversal.Result, error) {
	if s.ancestryResult == nil {
		return pitraversal.Result{}, errors.New("unexpected ancestry result")
	}
	return s.ancestryResult(ctx, key, opts...)
}

func (s stubDryRunProcessInstanceAPI) DescendantsResult(ctx context.Context, root string, opts ...services.CallOption) (pitraversal.Result, error) {
	if s.descendantsResult == nil {
		return pitraversal.Result{}, errors.New("unexpected descendants result")
	}
	return s.descendantsResult(ctx, root, opts...)
}

func (s stubDryRunProcessInstanceAPI) FamilyResult(context.Context, string, ...services.CallOption) (pitraversal.Result, error) {
	return pitraversal.Result{}, errors.New("unexpected family result")
}

func (s stubDryRunProcessInstanceAPI) GetProcessInstances(context.Context, typex.Keys, int, ...services.CallOption) ([]d.ProcessInstance, error) {
	return nil, errors.New("unexpected get process instances")
}

func (s stubDryRunProcessInstanceAPI) WaitForProcessInstancesState(context.Context, typex.Keys, d.States, int, ...services.CallOption) (d.StateResponses, error) {
	return d.StateResponses{}, errors.New("unexpected wait for process instances state")
}

func (s stubDryRunProcessInstanceAPI) WaitForProcessInstancesExpectation(context.Context, typex.Keys, d.ProcessInstanceExpectationRequest, int, ...services.CallOption) (d.ProcessInstanceExpectationResponses, error) {
	return d.ProcessInstanceExpectationResponses{}, errors.New("unexpected wait for process instances expectation")
}

type stubDryRunIncidentAPI struct{}

func (stubDryRunIncidentAPI) SearchIncidentsPage(context.Context, d.IncidentFilter, d.IncidentPageRequest, ...services.CallOption) (d.IncidentPage, error) {
	return d.IncidentPage{}, errors.New("unexpected incident page search")
}

func (stubDryRunIncidentAPI) SearchProcessInstanceIncidents(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	return nil, errors.New("unexpected process instance incident search")
}
