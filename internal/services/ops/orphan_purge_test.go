// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"bytes"
	"context"
	"log/slog"
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

	api := New(nil, nil)

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

	got, err := New(piAPI, nil).PurgeOrphanProcessInstances(context.Background(), request)

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

	got, err := New(piAPI, nil).PurgeOrphanProcessInstances(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Discovery.Status)
	require.Zero(t, got.Discovery.Count)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.DeletionPlan.Status)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
	require.Equal(t, d.OrphanPurgeOutcomePlanned, got.Outcome)
}

func TestPurgeOrphanProcessInstancesConfirmedDeletesImmutableDiscoveredSet(t *testing.T) {
	t.Parallel()

	searches := 0
	var deleted typex.Keys
	request := d.OrphanPurgeRequest{
		CommandName: "ops purge orphan-process-instances",
		AutoConfirm: true,
		StartedAt:   time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
	}
	piAPI := stubProcessInstanceAPI{
		searchPage: func(_ context.Context, _ d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			searches++
			require.Equal(t, 1, searches)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateNoMore,
				Items: []d.ProcessInstance{
					{Key: "child-1", ParentKey: "missing-parent", State: d.StateTerminated},
				},
			}, nil
		},
		filterOrphans: func(_ context.Context, items []d.ProcessInstance, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			return []d.ProcessInstance{items[0]}, nil
		},
		ancestryResult: func(_ context.Context, key string, _ ...services.CallOption) (pitraversal.Result, error) {
			require.Equal(t, "child-1", key)
			return pitraversal.Result{
				StartKey: key,
				RootKey:  key,
				Keys:     []string{key},
				Chain: map[string]d.ProcessInstance{
					key: {Key: key, ParentKey: "missing-parent", State: d.StateTerminated},
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
					rootKey: {Key: rootKey, ParentKey: "missing-parent", State: d.StateTerminated},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
		deleteProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			deleted = append(deleted, key)
			return d.DeleteResponse{Ok: true, StatusCode: 202, Status: "accepted"}, nil
		},
	}

	got, err := New(piAPI, nil).PurgeOrphanProcessInstances(context.Background(), request, services.WithNoWait())

	require.NoError(t, err)
	require.Equal(t, 1, searches)
	require.Equal(t, typexKeys("child-1"), got.Discovery.Keys)
	require.Equal(t, typexKeys("child-1"), got.DeletionPlan.RequestedKeys)
	require.Equal(t, typexKeys("child-1"), deleted)
	require.True(t, got.DeleteRequested)
	require.Equal(t, d.OpsWorkflowStepStatusSubmitted, got.Deletion.Status)
	require.True(t, got.Request.NoWait)
	require.True(t, got.Deletion.NoWait)
	require.True(t, got.Report.NoWait)
	require.Equal(t, d.OrphanPurgeOutcomeDeleted, got.Outcome)
}

func TestPurgeOrphanProcessInstancesUsesSuppliedLoggerForDeleteSummary(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	request := d.OrphanPurgeRequest{
		CommandName: "ops purge orphan-process-instances",
		AutoConfirm: true,
		DiscoveredKeys: typex.Keys{
			"child-1",
		},
		StartedAt: time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
	}
	piAPI := stubProcessInstanceAPI{
		ancestryResult: func(_ context.Context, key string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{
				StartKey: key,
				RootKey:  key,
				Keys:     []string{key},
				Chain: map[string]d.ProcessInstance{
					key: {Key: key, State: d.StateTerminated},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
		descendantsResult: func(_ context.Context, rootKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			return pitraversal.Result{
				StartKey: rootKey,
				RootKey:  rootKey,
				Keys:     []string{rootKey},
				Chain: map[string]d.ProcessInstance{
					rootKey: {Key: rootKey, State: d.StateTerminated},
				},
				Outcome: pitraversal.OutcomeComplete,
			}, nil
		},
		deleteProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			require.Equal(t, "child-1", key)
			return d.DeleteResponse{Ok: true, StatusCode: 202, Status: "accepted"}, nil
		},
	}

	got, err := New(piAPI, nil, logger).PurgeOrphanProcessInstances(context.Background(), request)

	require.NoError(t, err)
	require.Equal(t, d.OrphanPurgeOutcomeDeleted, got.Outcome)
	require.Contains(t, logBuf.String(), "pi delete done; requested 1, ok 1, failed 0")
}

type stubProcessInstanceAPI struct {
	pisvc.API
	createProcessInstance func(context.Context, d.ProcessInstanceData, ...services.CallOption) (d.ProcessInstanceCreation, error)
	search                func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error)
	searchPage            func(context.Context, d.ProcessInstanceFilter, d.ProcessInstancePageRequest, ...services.CallOption) (d.ProcessInstancePage, error)
	filterOrphans         func(context.Context, []d.ProcessInstance, ...services.CallOption) ([]d.ProcessInstance, error)
	ancestryResult        func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
	descendantsResult     func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
	familyResult          func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
	cancelProcessInstance func(context.Context, string, ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error)
	deleteProcessInstance func(context.Context, string, ...services.CallOption) (d.DeleteResponse, error)
}

func (s stubProcessInstanceAPI) CreateProcessInstance(ctx context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error) {
	if s.createProcessInstance == nil {
		panic("unexpected create")
	}
	return s.createProcessInstance(ctx, data, opts...)
}

func (s stubProcessInstanceAPI) SearchForProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	if s.search == nil {
		panic("unexpected search")
	}
	return s.search(ctx, filter, size, opts...)
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

func (s stubProcessInstanceAPI) FamilyResult(ctx context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error) {
	if s.familyResult == nil {
		panic("unexpected family")
	}
	return s.familyResult(ctx, startKey, opts...)
}

func (s stubProcessInstanceAPI) CancelProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
	if s.cancelProcessInstance == nil {
		panic("unexpected cancel")
	}
	return s.cancelProcessInstance(ctx, key, opts...)
}

func (s stubProcessInstanceAPI) DeleteProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error) {
	if s.deleteProcessInstance == nil {
		panic("unexpected delete")
	}
	return s.deleteProcessInstance(ctx, key, opts...)
}

func typexKeys(keys ...string) typex.Keys {
	return keys
}
