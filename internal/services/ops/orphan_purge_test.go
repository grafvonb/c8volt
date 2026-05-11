// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

func TestNewCreatesOrphanPurgeServiceBoundary(t *testing.T) {
	t.Parallel()

	api := New(nil)

	require.NotNil(t, api)
	require.Implements(t, (*API)(nil), api)
}

func TestPurgeOrphanProcessInstancesDryRunDiscoversOrphansAndPlansDeletion(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	hasParent := true
	var searchedFilter d.ProcessInstanceFilter
	request := d.OrphanPurgeRequest{
		CommandName: "ops purge orphan-process-instances",
		DryRun:      true,
		BatchSize:   50,
		Limit:       10,
		Workers:     2,
		StartedAt:   started,
		Selection: d.ProcessInstanceFilter{
			BpmnProcessId: "invoice",
			State:         d.StateActive,
		},
	}
	piAPI := stubProcessInstanceAPI{
		searchPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			searchedFilter = filter
			require.EqualValues(t, 50, page.Size)
			require.EqualValues(t, 0, page.From)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateNoMore,
				Items: []d.ProcessInstance{
					{Key: "child-1", ParentKey: "missing-parent", State: d.StateActive},
					{Key: "child-2", ParentKey: "existing-parent", State: d.StateActive},
				},
			}, nil
		},
		filterOrphans: func(_ context.Context, items []d.ProcessInstance, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			require.Len(t, items, 2)
			return []d.ProcessInstance{items[0]}, nil
		},
		ancestryResult: func(_ context.Context, key string, _ ...services.CallOption) (pitraversal.Result, error) {
			require.Equal(t, "child-1", key)
			return pitraversal.Result{
				StartKey: key,
				RootKey:  key,
				Keys:     []string{key},
				Chain: map[string]d.ProcessInstance{
					key: {Key: key, ParentKey: "missing-parent", State: d.StateActive},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
		descendantsResult: func(_ context.Context, rootKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			require.Equal(t, "child-1", rootKey)
			return pitraversal.Result{
				StartKey: rootKey,
				RootKey:  rootKey,
				Keys:     []string{rootKey},
				Chain: map[string]d.ProcessInstance{
					rootKey: {Key: rootKey, ParentKey: "missing-parent", State: d.StateActive},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
	}

	got, err := New(piAPI).PurgeOrphanProcessInstances(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, request, got.Request)
	require.Equal(t, &hasParent, searchedFilter.HasParent)
	require.Equal(t, "invoice", searchedFilter.BpmnProcessId)
	require.Equal(t, d.StateActive, searchedFilter.State)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Discovery.Status)
	require.Equal(t, typexKeys("child-1"), got.Discovery.Keys)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.DeletionPlan.Status)
	require.Equal(t, typexKeys("child-1"), got.DeletionPlan.RequestedKeys)
	require.Equal(t, typexKeys("child-1"), got.DeletionPlan.RootKeys)
	require.Equal(t, typexKeys("child-1"), got.DeletionPlan.AffectedKeys)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
	require.Equal(t, d.OrphanPurgeOutcomePlanned, got.Outcome)
	require.Equal(t, d.OrphanPurgeOutcomePlanned, got.Report.Outcome)
}

func TestPurgeOrphanProcessInstancesDryRunNoTargetsSkipsPlan(t *testing.T) {
	t.Parallel()

	request := d.OrphanPurgeRequest{
		CommandName: "ops purge orphan-process-instances",
		DryRun:      true,
		StartedAt:   time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
	}
	piAPI := stubProcessInstanceAPI{
		searchPage: func(_ context.Context, _ d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			return d.ProcessInstancePage{Request: page, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
		},
		filterOrphans: func(_ context.Context, items []d.ProcessInstance, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			require.Empty(t, items)
			return nil, nil
		},
		ancestryResult: func(context.Context, string, ...services.CallOption) (pitraversal.Result, error) {
			t.Fatal("unexpected dry-run plan for empty discovery")
			return pitraversal.Result{}, nil
		},
	}

	got, err := New(piAPI).PurgeOrphanProcessInstances(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Discovery.Status)
	require.Zero(t, got.Discovery.Count)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.DeletionPlan.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
	require.Equal(t, d.OrphanPurgeOutcomePlanned, got.Outcome)
}

type stubProcessInstanceAPI struct {
	pisvc.API
	searchPage        func(context.Context, d.ProcessInstanceFilter, d.ProcessInstancePageRequest, ...services.CallOption) (d.ProcessInstancePage, error)
	filterOrphans     func(context.Context, []d.ProcessInstance, ...services.CallOption) ([]d.ProcessInstance, error)
	ancestryResult    func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
	descendantsResult func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
}

func (s stubProcessInstanceAPI) SearchForProcessInstancesPage(ctx context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
	if s.searchPage == nil {
		panic("unexpected search")
	}
	return s.searchPage(ctx, filter, page, opts...)
}

func (s stubProcessInstanceAPI) FilterProcessInstanceWithOrphanParent(ctx context.Context, items []d.ProcessInstance, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	if s.filterOrphans == nil {
		panic("unexpected orphan filter")
	}
	return s.filterOrphans(ctx, items, opts...)
}

func (s stubProcessInstanceAPI) AncestryResult(ctx context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error) {
	if s.ancestryResult == nil {
		panic("unexpected ancestry")
	}
	return s.ancestryResult(ctx, startKey, opts...)
}

func (s stubProcessInstanceAPI) DescendantsResult(ctx context.Context, rootKey string, opts ...services.CallOption) (pitraversal.Result, error) {
	if s.descendantsResult == nil {
		panic("unexpected descendants")
	}
	return s.descendantsResult(ctx, rootKey, opts...)
}

func typexKeys(keys ...string) typex.Keys {
	return keys
}
