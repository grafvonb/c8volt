// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package resource

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/batchoperation"
	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	rsvc "github.com/grafvonb/c8volt/internal/services/resource"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClient_GetResource verifies facade-to-service option mapping and domain
// resource conversion. The facade should not expose generated Camunda response
// shapes to callers.
func TestClient_GetResource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := &stubResourceAPI{
		get: func(_ context.Context, key string, opts ...services.CallOption) (d.Resource, error) {
			cfg := services.ApplyCallOptions(opts)
			assert.Equal(t, "resource-1", key)
			assert.True(t, cfg.Verbose)
			return d.Resource{
				ID:         "demo-process",
				Key:        "resource-1",
				Name:       "demo.bpmn",
				TenantId:   "tenant-a",
				Version:    7,
				VersionTag: "v7",
			}, nil
		},
	}

	cli := New(api, nil, nil, slog.Default())
	got, err := cli.GetResource(ctx, "resource-1", options.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, Resource{
		ID:         "demo-process",
		Key:        "resource-1",
		Name:       "demo.bpmn",
		TenantId:   "tenant-a",
		Version:    7,
		VersionTag: "v7",
	}, got)
}

// TestClient_GetResource_MapsDomainErrors ensures domain lookup failures are
// normalized into facade errors so commands can share one exit-code model.
func TestClient_GetResource_MapsDomainErrors(t *testing.T) {
	t.Parallel()

	api := &stubResourceAPI{
		get: func(context.Context, string, ...services.CallOption) (d.Resource, error) {
			return d.Resource{}, d.ErrNotFound
		},
	}

	cli := New(api, nil, nil, slog.Default())
	_, err := cli.GetResource(context.Background(), "missing")

	require.Error(t, err)
	assert.ErrorIs(t, err, ferr.ErrNotFound)
}

// TestClient_DeleteProcessDefinition_UsesRootCancellationPlan covers the safety
// impact check before deleting a process definition resource. Active instances
// may be child/called instances, so c8volt expands them to root instances before
// cancellation, deletes the planned process-instance tree, and only then deletes
// the process definition resource.
func TestClient_DeleteProcessDefinition_UsesRootCancellationPlan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := &stubResourceAPI{
		delete: func(_ context.Context, resourceKey string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
			assert.Equal(t, "pd-1", resourceKey)
			return d.ResourceDeleteResponse{
				Ok:                true,
				StatusCode:        http.StatusOK,
				Status:            "200 OK; history deletion batch batch-1 completed",
				DeleteHistory:     true,
				BatchOperationKey: "batch-1",
				BatchState:        "COMPLETED",
			}, nil
		},
	}

	var canceledRoots typex.Keys
	var deletedRoots typex.Keys
	var affectedCount int
	var deletedAffectedCount int
	var statsCalls atomic.Int32
	papi := stubProcessAPI{
		getProcessDefinition: func(_ context.Context, key string, opts ...options.FacadeOption) (process.ProcessDefinition, error) {
			assert.Equal(t, "pd-1", key)
			assert.True(t, options.ApplyFacadeOptions(opts).Stat)
			active := int64(1)
			if statsCalls.Add(1) > 1 {
				active = 0
			}
			return process.ProcessDefinition{
				Key:        "pd-1",
				Statistics: &process.ProcessDefinitionStatistics{Active: active},
			}, nil
		},
		searchProcessInstancesPage: func(_ context.Context, filter process.ProcessInstanceFilter, page process.ProcessInstancePageRequest, _ ...options.FacadeOption) (process.ProcessInstancePage, error) {
			assert.Equal(t, process.ProcessInstanceFilter{ProcessDefinitionKey: "pd-1", State: process.StateActive}, filter)
			assert.Equal(t, int32(500), page.Size)
			return process.ProcessInstancePage{
				OverflowState: process.ProcessInstanceOverflowStateNoMore,
				Items: []process.ProcessInstance{
					{Key: "child-1", ParentKey: "root-1", RootProcessInstanceKey: "root-1"},
					{Key: "child-2", ParentKey: "root-1", RootProcessInstanceKey: "root-1"},
				},
			}, nil
		},
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			assert.Equal(t, typex.Keys{"root-1"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-1"},
				Collected: typex.Keys{"root-1", "child-1", "child-2"},
				Outcome:   process.TraversalOutcomeComplete,
			}, nil
		},
		cancelProcessInstances: func(_ context.Context, keys typex.Keys, _ int, opts ...options.FacadeOption) (process.CancelReports, error) {
			canceledRoots = keys
			affectedCount = options.ApplyFacadeOptions(opts).AffectedProcessInstanceCount
			return process.CancelReports{Items: []process.CancelReport{{Key: "root-1", Ok: true}}}, nil
		},
		deleteProcessInstances: func(_ context.Context, keys typex.Keys, _ int, opts ...options.FacadeOption) (process.DeleteReports, error) {
			deletedRoots = keys
			deletedAffectedCount = options.ApplyFacadeOptions(opts).AffectedProcessInstanceCount
			return process.DeleteReports{Items: []process.DeleteReport{{Key: "root-1", Ok: true}}}, nil
		},
	}

	cli := New(api, papi, nil, slog.Default())
	report, err := cli.DeleteProcessDefinition(ctx, "pd-1", options.WithForce())

	require.NoError(t, err)
	assert.True(t, report.Ok)
	assert.True(t, report.DeleteHistory)
	assert.Equal(t, "batch-1", report.BatchOperationKey)
	assert.Equal(t, "COMPLETED", report.BatchState)
	assert.Equal(t, typex.Keys{"root-1"}, canceledRoots)
	assert.Equal(t, 3, affectedCount)
	assert.Equal(t, typex.Keys{"root-1"}, deletedRoots)
	assert.Equal(t, 3, deletedAffectedCount)
	assert.Equal(t, int32(2), statsCalls.Load())
}

// TestClient_DeleteProcessDefinition_ForwardsContextToRootCancellation verifies
// root cancellation and process-instance deletion receive caller context.
func TestClient_DeleteProcessDefinition_ForwardsContextToRootCancellation(t *testing.T) {
	t.Parallel()

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "request-ctx")
	api := &stubResourceAPI{
		delete: func(_ context.Context, resourceKey string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
			assert.Equal(t, "pd-1", resourceKey)
			return d.ResourceDeleteResponse{Ok: true, StatusCode: http.StatusOK, Status: "200 OK"}, nil
		},
	}
	var statsCalls atomic.Int32
	papi := stubProcessAPI{
		getProcessDefinition: func(_ context.Context, key string, opts ...options.FacadeOption) (process.ProcessDefinition, error) {
			assert.Equal(t, "pd-1", key)
			assert.True(t, options.ApplyFacadeOptions(opts).Stat)
			active := int64(1)
			if statsCalls.Add(1) > 1 {
				active = 0
			}
			return process.ProcessDefinition{
				Key:        "pd-1",
				Statistics: &process.ProcessDefinitionStatistics{Active: active},
			}, nil
		},
		searchProcessInstancesPage: func(got context.Context, filter process.ProcessInstanceFilter, _ process.ProcessInstancePageRequest, _ ...options.FacadeOption) (process.ProcessInstancePage, error) {
			if got.Value(ctxKey{}) != "request-ctx" {
				return process.ProcessInstancePage{}, errors.New("active instance search did not receive caller context")
			}
			assert.Equal(t, process.ProcessInstanceFilter{ProcessDefinitionKey: "pd-1", State: process.StateActive}, filter)
			return process.ProcessInstancePage{
				OverflowState: process.ProcessInstanceOverflowStateNoMore,
				Items:         []process.ProcessInstance{{Key: "child-1", ParentKey: "root-1", RootProcessInstanceKey: "root-1"}},
			}, nil
		},
		dryRunCancelOrDeletePlan: func(got context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			if got.Value(ctxKey{}) != "request-ctx" {
				return process.DryRunPIKeyExpansion{}, errors.New("cancellation planning did not receive caller context")
			}
			assert.Equal(t, typex.Keys{"root-1"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-1"},
				Collected: typex.Keys{"root-1", "child-1"},
				Outcome:   process.TraversalOutcomeComplete,
			}, nil
		},
		cancelProcessInstances: func(got context.Context, keys typex.Keys, _ int, _ ...options.FacadeOption) (process.CancelReports, error) {
			if got.Value(ctxKey{}) != "request-ctx" {
				return process.CancelReports{}, errors.New("root cancellation did not receive caller context")
			}
			assert.Equal(t, typex.Keys{"root-1"}, keys)
			return process.CancelReports{Items: []process.CancelReport{{Key: "root-1", Ok: true}}}, nil
		},
		deleteProcessInstances: func(got context.Context, keys typex.Keys, _ int, _ ...options.FacadeOption) (process.DeleteReports, error) {
			if got.Value(ctxKey{}) != "request-ctx" {
				return process.DeleteReports{}, errors.New("process-instance deletion did not receive caller context")
			}
			assert.Equal(t, typex.Keys{"root-1"}, keys)
			return process.DeleteReports{Items: []process.DeleteReport{{Key: "root-1", Ok: true}}}, nil
		},
	}

	cli := New(api, papi, nil, slog.Default())
	report, err := cli.DeleteProcessDefinition(ctx, "pd-1", options.WithForce())

	require.NoError(t, err)
	assert.True(t, report.Ok)
	assert.Equal(t, int32(2), statsCalls.Load())
}

// TestFormatPartialCancellationImpactWarning_HidesMissingAncestorKeysUntilVerbose verifies warning details respect verbosity.
func TestFormatPartialCancellationImpactWarning_HidesMissingAncestorKeysUntilVerbose(t *testing.T) {
	t.Parallel()

	plan := process.DryRunPIKeyExpansion{
		MissingAncestors: []process.MissingAncestor{
			{Key: "missing-1", StartKey: "child-1"},
			{Key: "missing-2", StartKey: "child-2"},
		},
		Warning: "one or more parent process instances were not found",
	}

	quiet := formatPartialCancellationImpactWarning("pd-1", plan, false)
	verbose := formatPartialCancellationImpactWarning("pd-1", plan, true)

	assert.Contains(t, quiet, "2 missing ancestor key(s)")
	assert.Contains(t, quiet, "use --verbose to list keys")
	assert.NotContains(t, quiet, "missing-1")
	assert.NotContains(t, quiet, "missing-2")
	assert.Contains(t, verbose, "missing ancestor keys: missing-1, missing-2")
}

// TestClient_PreviewDeleteProcessDefinition_UsesStatsForNonForceImpactCheck keeps
// the default safety check cheap: without --force, deletion only needs an active
// count, not a full list of process-instance keys.
func TestClient_PreviewDeleteProcessDefinition_UsesStatsForNonForceImpactCheck(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := &stubResourceAPI{}
	papi := stubProcessAPI{
		getProcessDefinition: func(_ context.Context, key string, opts ...options.FacadeOption) (process.ProcessDefinition, error) {
			assert.Equal(t, "pd-1", key)
			assert.True(t, options.ApplyFacadeOptions(opts).Stat)
			return process.ProcessDefinition{
				Key: "pd-1",
				Statistics: &process.ProcessDefinitionStatistics{
					Active: 2,
				},
			}, nil
		},
		searchProcessInstancesPage: func(context.Context, process.ProcessInstanceFilter, process.ProcessInstancePageRequest, ...options.FacadeOption) (process.ProcessInstancePage, error) {
			t.Fatalf("non-force impact check should use process-definition statistics, not process-instance search")
			return process.ProcessInstancePage{}, nil
		},
	}

	cli := New(api, papi, nil, slog.Default())
	plan, err := cli.PreviewDeleteProcessDefinitions(ctx, typex.Keys{"pd-1"})

	require.NoError(t, err)
	require.Len(t, plan.Items, 1)
	assert.Equal(t, int64(2), plan.Items[0].ActiveProcessInstances())
	assert.Equal(t, int64(2), plan.Totals().ActiveProcessInstances)
}

// TestClient_PreviewDeleteProcessDefinitions_UsesActivityIndicator verifies
// slow delete-pd impact checks have one high-level progress message instead of only
// per-request HTTP activity.
func TestClient_PreviewDeleteProcessDefinitions_UsesActivityIndicator(t *testing.T) {
	t.Parallel()

	sink := &activitysink.Sink{}
	ctx := logging.ToActivityContext(context.Background(), sink)
	api := &stubResourceAPI{}

	cli := New(api, nil, nil, slog.Default())
	plan, err := cli.PreviewDeleteProcessDefinitions(ctx, typex.Keys{"pd-1", "pd-2"}, options.WithNoStateCheck())

	require.NoError(t, err)
	require.Len(t, plan.Items, 2)
	started, stopped, msgs := sink.Snapshot()
	assert.Equal(t, 1, started)
	assert.Equal(t, 1, stopped)
	assert.Equal(t, []string{"checking delete impact for 2 process definition(s); process-instance state check is skipped; no changes are being made"}, msgs)
}

// TestClient_PreviewDeleteProcessDefinitions_ExpandsRootsForForce verifies
// forced deletion impact checking reports root cancellation scope so child/called
// process instances are handled correctly.
func TestClient_PreviewDeleteProcessDefinitions_ExpandsRootsForForce(t *testing.T) {
	t.Parallel()

	sink := &activitysink.Sink{}
	ctx := logging.ToActivityContext(context.Background(), sink)
	api := &stubResourceAPI{}
	papi := stubProcessAPI{
		getProcessDefinition: func(_ context.Context, key string, opts ...options.FacadeOption) (process.ProcessDefinition, error) {
			assert.Equal(t, "pd-1", key)
			assert.True(t, options.ApplyFacadeOptions(opts).Stat)
			return process.ProcessDefinition{
				Key:        "pd-1",
				Statistics: &process.ProcessDefinitionStatistics{Active: 2},
			}, nil
		},
		searchProcessInstancesPage: func(_ context.Context, filter process.ProcessInstanceFilter, _ process.ProcessInstancePageRequest, _ ...options.FacadeOption) (process.ProcessInstancePage, error) {
			assert.Equal(t, process.ProcessInstanceFilter{ProcessDefinitionKey: "pd-1", State: process.StateActive}, filter)
			return process.ProcessInstancePage{
				OverflowState: process.ProcessInstanceOverflowStateNoMore,
				Items: []process.ProcessInstance{
					{Key: "child-1", ParentKey: "root-1", RootProcessInstanceKey: "root-1"},
					{Key: "child-2", ParentKey: "root-1", RootProcessInstanceKey: "root-1"},
				},
			}, nil
		},
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			assert.Equal(t, typex.Keys{"root-1"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-1"},
				Collected: typex.Keys{"root-1", "child-1", "child-2"},
				Outcome:   process.TraversalOutcomeComplete,
			}, nil
		},
	}

	cli := New(api, papi, nil, slog.Default())
	plan, err := cli.PreviewDeleteProcessDefinitions(ctx, typex.Keys{"pd-1"}, options.WithForce())

	require.NoError(t, err)
	require.Len(t, plan.Items, 1)
	assert.Equal(t, int64(2), plan.Totals().ActiveProcessInstances)
	assert.Equal(t, 1, plan.Totals().CancellationRoots)
	assert.Equal(t, 3, plan.Totals().CancellationAffected)
	started, stopped, msgs := sink.Snapshot()
	assert.Equal(t, 1, started)
	assert.Equal(t, 1, stopped)
	assert.Equal(t, []string{
		"checking active process instances and cancellation roots for 1 process definition(s); no changes are being made",
	}, msgs)
}

// TestClient_DeleteProcessDefinitions_UsesActivityIndicator verifies bulk deletion
// wraps each direct item delete with its own delete-impact activity scope.
func TestClient_DeleteProcessDefinitions_UsesActivityIndicator(t *testing.T) {
	t.Parallel()

	sink := &activitysink.Sink{}
	ctx := logging.ToActivityContext(context.Background(), sink)
	api := &stubResourceAPI{
		delete: func(_ context.Context, _ string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
			return d.ResourceDeleteResponse{Ok: true, StatusCode: http.StatusOK, Status: "200 OK"}, nil
		},
	}
	cli := New(api, nil, nil, slog.Default())

	reports, err := cli.DeleteProcessDefinitions(ctx, typex.Keys{"pd-1", "pd-2"}, 1, options.WithNoStateCheck())

	require.NoError(t, err)
	require.Len(t, reports.Items, 2)
	started, stopped, msgs := sink.Snapshot()
	assert.Equal(t, 3, started)
	assert.Equal(t, 3, stopped)
	assert.Equal(t, []string{
		"deleting 2 process definition(s)",
		"checking delete impact for 1 process definition(s); process-instance state check is skipped; no changes are being made",
		"checking delete impact for 1 process definition(s); process-instance state check is skipped; no changes are being made",
	}, msgs)
}

type stubResourceAPI struct {
	get    func(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.Resource, error)
	delete func(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.ResourceDeleteResponse, error)
}

func (s *stubResourceAPI) Deploy(context.Context, []d.DeploymentUnitData, ...services.CallOption) (d.Deployment, error) {
	panic("unexpected call")
}

// Delete delegates to the per-test callback and panics if a test did not expect
// the resource deletion endpoint.
func (s *stubResourceAPI) Delete(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.ResourceDeleteResponse, error) {
	if s.delete == nil {
		panic("unexpected call")
	}
	return s.delete(ctx, resourceKey, opts...)
}

// Get delegates to the per-test callback and panics when an unrelated resource
// lookup path is exercised.
func (s *stubResourceAPI) Get(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.Resource, error) {
	if s.get == nil {
		panic("unexpected call")
	}
	return s.get(ctx, resourceKey, opts...)
}

var _ rsvc.API = (*stubResourceAPI)(nil)

type stubBatchOperationAPI struct {
	checkBatchOperationReadAccess func(context.Context, ...options.FacadeOption) error
	cancelProcessInstancesBatch   func(context.Context, process.ProcessInstanceFilter, ...options.FacadeOption) (batchoperation.BatchOperation, error)
	waitBatchOperation            func(context.Context, string, ...options.FacadeOption) (batchoperation.BatchOperation, error)
}

func (s stubBatchOperationAPI) CheckBatchOperationReadAccess(ctx context.Context, opts ...options.FacadeOption) error {
	if s.checkBatchOperationReadAccess == nil {
		panic("unexpected call")
	}
	return s.checkBatchOperationReadAccess(ctx, opts...)
}

func (s stubBatchOperationAPI) CancelProcessInstancesBatch(ctx context.Context, filter process.ProcessInstanceFilter, opts ...options.FacadeOption) (batchoperation.BatchOperation, error) {
	if s.cancelProcessInstancesBatch == nil {
		panic("unexpected call")
	}
	return s.cancelProcessInstancesBatch(ctx, filter, opts...)
}

func (s stubBatchOperationAPI) WaitBatchOperation(ctx context.Context, key string, opts ...options.FacadeOption) (batchoperation.BatchOperation, error) {
	if s.waitBatchOperation == nil {
		panic("unexpected call")
	}
	return s.waitBatchOperation(ctx, key, opts...)
}

var _ batchoperation.API = (*stubBatchOperationAPI)(nil)

type stubProcessAPI struct {
	getProcessDefinition       func(context.Context, string, ...options.FacadeOption) (process.ProcessDefinition, error)
	searchProcessInstancesPage func(context.Context, process.ProcessInstanceFilter, process.ProcessInstancePageRequest, ...options.FacadeOption) (process.ProcessInstancePage, error)
	searchProcessInstances     func(context.Context, process.ProcessInstanceFilter, int32, ...options.FacadeOption) (process.ProcessInstances, error)
	dryRunCancelOrDeletePlan   func(context.Context, typex.Keys, ...options.FacadeOption) (process.DryRunPIKeyExpansion, error)
	cancelProcessInstances     func(context.Context, typex.Keys, int, ...options.FacadeOption) (process.CancelReports, error)
	deleteProcessInstances     func(context.Context, typex.Keys, int, ...options.FacadeOption) (process.DeleteReports, error)
}

func (stubProcessAPI) SearchProcessDefinitions(context.Context, process.ProcessDefinitionFilter, ...options.FacadeOption) (process.ProcessDefinitions, error) {
	panic("unexpected call")
}

func (stubProcessAPI) SearchProcessDefinitionsLatest(context.Context, process.ProcessDefinitionFilter, ...options.FacadeOption) (process.ProcessDefinitions, error) {
	panic("unexpected call")
}

func (s stubProcessAPI) GetProcessDefinition(ctx context.Context, key string, opts ...options.FacadeOption) (process.ProcessDefinition, error) {
	if s.getProcessDefinition == nil {
		panic("unexpected call")
	}
	return s.getProcessDefinition(ctx, key, opts...)
}

func (stubProcessAPI) GetProcessDefinitionXML(context.Context, string, ...options.FacadeOption) (string, error) {
	panic("unexpected call")
}

func (stubProcessAPI) CreateProcessInstance(context.Context, process.ProcessInstanceData, ...options.FacadeOption) (process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) CreateProcessInstances(context.Context, []process.ProcessInstanceData, ...options.FacadeOption) ([]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetProcessInstance(context.Context, string, ...options.FacadeOption) (process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) LookupProcessInstance(context.Context, string, ...options.FacadeOption) (process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) LookupProcessInstanceStateByKey(context.Context, string, ...options.FacadeOption) (process.StateReport, process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) SearchProcessInstanceIncidents(context.Context, string, ...options.FacadeOption) ([]process.ProcessInstanceIncidentDetail, error) {
	panic("unexpected call")
}

func (stubProcessAPI) ResolveIncident(context.Context, string, ...options.FacadeOption) (process.IncidentResolutionResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) ResolveIncidents(context.Context, typex.Keys, int, ...options.FacadeOption) (process.IncidentResolutionResults, error) {
	panic("unexpected call")
}

func (stubProcessAPI) ResolveProcessInstanceIncidents(context.Context, string, ...options.FacadeOption) (process.ProcessInstanceResolutionResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) ResolveProcessInstancesIncidents(context.Context, typex.Keys, int, ...options.FacadeOption) (process.ProcessInstanceResolutionResults, error) {
	panic("unexpected call")
}

func (stubProcessAPI) SearchProcessInstanceVariables(context.Context, string, ...options.FacadeOption) ([]process.ProcessInstanceVariable, error) {
	panic("unexpected call")
}

func (stubProcessAPI) UpdateProcessInstanceVariables(context.Context, process.ProcessInstanceVariableUpdateRequest, ...options.FacadeOption) (process.ProcessInstanceVariableUpdateResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) EnrichProcessInstancesWithIncidents(context.Context, process.ProcessInstances, ...options.FacadeOption) (process.IncidentEnrichedProcessInstances, error) {
	panic("unexpected call")
}

func (stubProcessAPI) EnrichProcessInstancesWithVariables(context.Context, process.ProcessInstances, ...options.FacadeOption) (process.VariableEnrichedProcessInstances, error) {
	panic("unexpected call")
}

func (stubProcessAPI) EnrichTraversalWithIncidents(context.Context, process.TraversalResult, ...options.FacadeOption) (process.IncidentEnrichedTraversalResult, error) {
	panic("unexpected call")
}

func (s stubProcessAPI) SearchProcessInstancesPage(ctx context.Context, filter process.ProcessInstanceFilter, page process.ProcessInstancePageRequest, opts ...options.FacadeOption) (process.ProcessInstancePage, error) {
	if s.searchProcessInstancesPage == nil {
		panic("unexpected call")
	}
	return s.searchProcessInstancesPage(ctx, filter, page, opts...)
}

// SearchProcessInstances delegates to the per-test callback used by process
// definition deletion to discover active process instances.
func (s stubProcessAPI) SearchProcessInstances(ctx context.Context, filter process.ProcessInstanceFilter, size int32, opts ...options.FacadeOption) (process.ProcessInstances, error) {
	if s.searchProcessInstances == nil {
		panic("unexpected call")
	}
	return s.searchProcessInstances(ctx, filter, size, opts...)
}

func (stubProcessAPI) CancelProcessInstance(context.Context, string, ...options.FacadeOption) (process.CancelReport, process.ProcessInstances, error) {
	panic("unexpected call")
}

func (stubProcessAPI) DeleteProcessInstance(context.Context, string, ...options.FacadeOption) (process.DeleteReport, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetDirectChildrenOfProcessInstance(context.Context, string, ...options.FacadeOption) (process.ProcessInstances, error) {
	panic("unexpected call")
}

func (stubProcessAPI) FilterProcessInstanceWithOrphanParent(context.Context, []process.ProcessInstance, ...options.FacadeOption) ([]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) WaitForProcessInstanceState(context.Context, string, process.States, ...options.FacadeOption) (process.StateReport, process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) WaitForProcessInstanceExpectation(context.Context, string, process.ProcessInstanceExpectationRequest, ...options.FacadeOption) (process.ProcessInstanceExpectationReport, process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) Ancestry(context.Context, string, ...options.FacadeOption) (string, []string, map[string]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) Descendants(context.Context, string, ...options.FacadeOption) ([]string, map[string][]string, map[string]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) Family(context.Context, string, ...options.FacadeOption) ([]string, map[string][]string, map[string]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) AncestryResult(context.Context, string, ...options.FacadeOption) (process.TraversalResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) DescendantsResult(context.Context, string, ...options.FacadeOption) (process.TraversalResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) FamilyResult(context.Context, string, ...options.FacadeOption) (process.TraversalResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetProcessInstances(context.Context, typex.Keys, int, ...options.FacadeOption) (process.ProcessInstances, error) {
	panic("unexpected call")
}

func (stubProcessAPI) CreateNProcessInstances(context.Context, process.ProcessInstanceData, int, int, ...options.FacadeOption) ([]process.ProcessInstance, error) {
	panic("unexpected call")
}

// CancelProcessInstances delegates to the per-test callback and records the
// root keys chosen after dry-run expansion.
func (s stubProcessAPI) CancelProcessInstances(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.CancelReports, error) {
	if s.cancelProcessInstances == nil {
		panic("unexpected call")
	}
	return s.cancelProcessInstances(ctx, keys, wantedWorkers, opts...)
}

func (s stubProcessAPI) DeleteProcessInstances(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.DeleteReports, error) {
	if s.deleteProcessInstances == nil {
		panic("unexpected call")
	}
	return s.deleteProcessInstances(ctx, keys, wantedWorkers, opts...)
}

func (stubProcessAPI) UpdateProcessInstancesVariables(context.Context, typex.Keys, map[string]any, int, ...options.FacadeOption) (process.ProcessInstanceVariableUpdateResults, error) {
	panic("unexpected call")
}

func (stubProcessAPI) WaitForProcessInstancesState(context.Context, typex.Keys, process.States, int, ...options.FacadeOption) (process.StateReports, error) {
	panic("unexpected call")
}

func (stubProcessAPI) WaitForProcessInstancesExpectation(context.Context, typex.Keys, process.ProcessInstanceExpectationRequest, int, ...options.FacadeOption) (process.ProcessInstanceExpectationReports, error) {
	panic("unexpected call")
}

func (stubProcessAPI) DryRunCancelOrDeleteGetPIKeys(context.Context, typex.Keys, int, ...options.FacadeOption) (typex.Keys, typex.Keys, error) {
	panic("unexpected call")
}

func (s stubProcessAPI) DryRunCancelOrDeletePlan(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
	if s.dryRunCancelOrDeletePlan == nil {
		panic("unexpected call")
	}
	return s.dryRunCancelOrDeletePlan(ctx, keys, opts...)
}

var _ process.API = stubProcessAPI{}
