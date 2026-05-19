// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processdefinition

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFormatPartialCancellationImpactWarning_HidesMissingAncestorKeysUntilVerbose verifies quiet warnings hide key detail.
func TestFormatPartialCancellationImpactWarning_HidesMissingAncestorKeysUntilVerbose(t *testing.T) {
	t.Parallel()

	plan := d.DryRunPIKeyExpansion{
		MissingAncestors: []d.MissingAncestor{
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

// TestProcessDefinitionDeleteLogSubjectUsesBPMNProcessIDVersionAndKey verifies full process-definition labels.
func TestProcessDefinitionDeleteLogSubjectUsesBPMNProcessIDVersionAndKey(t *testing.T) {
	t.Parallel()

	got := processDefinitionDeleteLogSubject(d.DeleteProcessDefinitionPlanItem{
		Key:               "2251799813685255",
		BpmnProcessId:     "invoice",
		ProcessVersion:    5,
		ProcessVersionTag: "v1.0.0",
		TenantId:          "<default>",
	})

	assert.Equal(t, "pd 2251799813685255 invoice v5/v1.0.0 <default>", got)
}

// TestProcessDefinitionDeleteLogSubjectOmitsMissingVersion verifies labels stay compact without version metadata.
func TestProcessDefinitionDeleteLogSubjectOmitsMissingVersion(t *testing.T) {
	t.Parallel()

	got := processDefinitionDeleteLogSubject(d.DeleteProcessDefinitionPlanItem{
		Key:           "2251799813685255",
		BpmnProcessId: "invoice",
		TenantId:      "tenant-a",
	})

	assert.Equal(t, "pd 2251799813685255 invoice tenant-a", got)
}

// TestProcessDefinitionDeleteLogSubjectFallsBackToKeyOnly verifies key-only labels when BPMN metadata is absent.
func TestProcessDefinitionDeleteLogSubjectFallsBackToKeyOnly(t *testing.T) {
	t.Parallel()

	got := processDefinitionDeleteLogSubject(d.DeleteProcessDefinitionPlanItem{Key: "2251799813685255"})

	assert.Equal(t, "pd 2251799813685255", got)
}

// TestLogProcessDefinitionDeleteResultUsesSequentialLifecycleTerms verifies accepted and confirmed delete wording.
func TestLogProcessDefinitionDeleteResultUsesSequentialLifecycleTerms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		resp d.ResourceDeleteResponse
		want string
	}{
		{
			name: "confirmed batch after wait",
			resp: d.ResourceDeleteResponse{BatchOperationKey: "batch-1", BatchState: "COMPLETED"},
			want: "pd 1 invoice v3 tenant; delete confirmed; batch batch-1, state COMPLETED",
		},
		{
			name: "accepted batch without confirmation",
			resp: d.ResourceDeleteResponse{BatchOperationKey: "batch-1"},
			want: "pd 1 invoice v3 tenant; delete accepted; batch batch-1",
		},
		{
			name: "direct status without batch",
			resp: d.ResourceDeleteResponse{Status: "204 No Content"},
			want: "pd 1 invoice v3 tenant; delete done; status 204 No Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			log := slog.New(slog.NewTextHandler(&buf, nil))

			logProcessDefinitionDeleteResult(log, "pd 1 invoice v3 tenant", tt.resp)

			assert.Contains(t, buf.String(), tt.want)
		})
	}
}

type cleanupProcessInstanceAPI struct {
	pisvc.API
	cancel func(context.Context, string, ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error)
	delete func(context.Context, string, ...services.CallOption) (d.DeleteResponse, error)
}

// CancelProcessInstance delegates cancellation to the configured test callback.
func (s cleanupProcessInstanceAPI) CancelProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
	return s.cancel(ctx, key, opts...)
}

// DeleteProcessInstance delegates deletion to the configured test callback.
func (s cleanupProcessInstanceAPI) DeleteProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error) {
	return s.delete(ctx, key, opts...)
}

type cleanupProcessDefinitionAPI struct {
	API
}

type deleteVisibilityProcessDefinitionAPI struct {
	API
	calls atomic.Int64
}

func (s *deleteVisibilityProcessDefinitionAPI) GetProcessDefinition(_ context.Context, key string, _ ...services.CallOption) (d.ProcessDefinition, error) {
	s.calls.Add(1)
	return d.ProcessDefinition{}, fmt.Errorf("%w: process definition %s not found", d.ErrNotFound, key)
}

type testResourceDeleteAPI struct {
	delete func(context.Context, string, ...services.CallOption) (d.ResourceDeleteResponse, error)
}

func (s testResourceDeleteAPI) Delete(ctx context.Context, key string, opts ...services.CallOption) (d.ResourceDeleteResponse, error) {
	return s.delete(ctx, key, opts...)
}

type unsupportedResourceDeleteAPI struct {
	testResourceDeleteAPI
}

func (unsupportedResourceDeleteAPI) SupportsProcessDefinitionHistoryDeletion() bool { return false }

// TestDeleteProcessDefinitionsRejectsUnsupportedHistoryDeletionBeforeCleanup protects callers from partial v8.8 cleanup.
func TestDeleteProcessDefinitionsRejectsUnsupportedHistoryDeletionBeforeCleanup(t *testing.T) {
	api := unsupportedResourceDeleteAPI{testResourceDeleteAPI{
		delete: func(context.Context, string, ...services.CallOption) (d.ResourceDeleteResponse, error) {
			t.Fatal("resource delete must not be called")
			return d.ResourceDeleteResponse{}, nil
		},
	}}

	got, err := DeleteProcessDefinitions(
		context.Background(),
		api,
		cleanupProcessDefinitionAPI{},
		cleanupProcessInstanceAPI{
			cancel: func(context.Context, string, ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
				t.Fatal("process-instance cancellation must not be called")
				return d.CancelResponse{}, nil, nil
			},
			delete: func(context.Context, string, ...services.CallOption) (d.DeleteResponse, error) {
				t.Fatal("process-instance deletion must not be called")
				return d.DeleteResponse{}, nil
			},
		},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		types.Keys{"pd-1"},
		1,
		services.WithForce(),
	)

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrUnsupported)
	require.Contains(t, err.Error(), "requires Camunda 8.9 or newer")
	require.Nil(t, got)
}

// TestDeleteProcessDefinitionResourcesStopsOnDeleteHistoryRequestShapeError verifies a server-side request-shape mismatch is reported once instead of being repeated for every selected definition.
func TestDeleteProcessDefinitionResourcesStopsOnDeleteHistoryRequestShapeError(t *testing.T) {
	var calls atomic.Int64
	api := testResourceDeleteAPI{
		delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
			calls.Add(1)
			require.Equal(t, "pd-1", key)
			return d.ResourceDeleteResponse{Key: key}, fmt.Errorf("%w: 400 POST /v2/resources/%s/deletion (Request property [deleteHistory] cannot be parsed)", d.ErrBadRequest, key)
		},
	}
	plans := []d.DeleteProcessDefinitionPlanItem{
		{Key: "pd-1"},
		{Key: "pd-2"},
		{Key: "pd-3"},
	}

	got, err := DeleteProcessDefinitionResources(context.Background(), api, cleanupProcessDefinitionAPI{}, slog.New(slog.NewTextHandler(io.Discard, nil)), plans, 10)

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrBadRequest)
	require.ErrorContains(t, err, "deleteHistory")
	require.ErrorContains(t, err, "before submitting 2 remaining process-definition delete request(s)")
	require.Equal(t, int64(1), calls.Load())
	require.Len(t, got, 1)
	require.Equal(t, "pd-1", got[0].Key)
}

// TestDeleteProcessDefinitionResourcesWaitsForDefinitionAbsenceAfterBatchCompletion verifies batch completion is not treated as the final visibility proof.
func TestDeleteProcessDefinitionResourcesWaitsForDefinitionAbsenceAfterBatchCompletion(t *testing.T) {
	var deletes atomic.Int64
	api := testResourceDeleteAPI{
		delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
			deletes.Add(1)
			return d.ResourceDeleteResponse{Key: key, BatchOperationKey: "batch-1", BatchState: "COMPLETED"}, nil
		},
	}
	pdAPI := &deleteVisibilityProcessDefinitionAPI{}

	got, err := DeleteProcessDefinitionResources(
		context.Background(),
		api,
		pdAPI,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		[]d.DeleteProcessDefinitionPlanItem{{Key: "pd-1"}},
		1,
	)

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, int64(1), deletes.Load())
	require.Equal(t, int64(1), pdAPI.calls.Load())
}

// GetProcessDefinition returns inactive statistics so force cleanup can proceed after cancellation.
func (cleanupProcessDefinitionAPI) GetProcessDefinition(_ context.Context, key string, _ ...services.CallOption) (d.ProcessDefinition, error) {
	return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{}}, nil
}

// TestCleanupProcessDefinitionDeletePlanForceScopeUsesRequestedWorkers verifies APD worker settings reach nested PI cleanup.
func TestCleanupProcessDefinitionDeletePlanForceScopeUsesRequestedWorkers(t *testing.T) {
	const roots = 40

	var plans []d.DeleteProcessDefinitionPlanItem
	for i := range roots {
		key := "root-" + strconv.Itoa(i)
		plans = append(plans, d.DeleteProcessDefinitionPlanItem{
			Key: key,
			CancellationPlan: d.DryRunPIKeyExpansion{
				Roots:     []string{key},
				Collected: []string{key},
			},
		})
	}

	var cancelStarted atomic.Int64
	var deleteStarted atomic.Int64
	cancelRelease := make(chan struct{})
	deleteRelease := make(chan struct{})
	var cancelReleaseOnce sync.Once
	var deleteReleaseOnce sync.Once
	t.Cleanup(func() {
		cancelReleaseOnce.Do(func() { close(cancelRelease) })
		deleteReleaseOnce.Do(func() { close(deleteRelease) })
	})

	piAPI := cleanupProcessInstanceAPI{
		cancel: func(ctx context.Context, _ string, _ ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
			cancelStarted.Add(1)
			select {
			case <-ctx.Done():
				return d.CancelResponse{}, nil, ctx.Err()
			case <-cancelRelease:
				return d.CancelResponse{Ok: true, StatusCode: 200, Status: "200 OK"}, nil, nil
			}
		},
		delete: func(ctx context.Context, _ string, _ ...services.CallOption) (d.DeleteResponse, error) {
			deleteStarted.Add(1)
			select {
			case <-ctx.Done():
				return d.DeleteResponse{}, ctx.Err()
			case <-deleteRelease:
				return d.DeleteResponse{Ok: true, StatusCode: 200, Status: "200 OK"}, nil
			}
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- cleanupProcessDefinitionDeletePlanForceScope(
			context.Background(),
			cleanupProcessDefinitionAPI{},
			piAPI,
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			plans,
			roots,
			services.WithSuppressProcessInstanceDetailLogs(),
		)
	}()

	require.Eventually(t, func() bool {
		return cancelStarted.Load() == roots
	}, time.Second, 10*time.Millisecond)
	cancelReleaseOnce.Do(func() { close(cancelRelease) })
	require.Eventually(t, func() bool {
		return deleteStarted.Load() == roots
	}, time.Second, 10*time.Millisecond)
	deleteReleaseOnce.Do(func() { close(deleteRelease) })
	require.NoError(t, <-errCh)
}

type workerPreviewProcessDefinitionAPI struct {
	API
	active int64
}

// GetProcessDefinition returns active statistics for worker propagation preview tests.
func (s workerPreviewProcessDefinitionAPI) GetProcessDefinition(_ context.Context, key string, _ ...services.CallOption) (d.ProcessDefinition, error) {
	return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{Active: s.active}}, nil
}

type workerPreviewProcessInstanceAPI struct {
	pisvc.API
	roots       int
	ancestry    func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
	descendants func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
}

// SearchForProcessInstancesPage returns one active process instance for each configured root.
func (s workerPreviewProcessInstanceAPI) SearchForProcessInstancesPage(_ context.Context, _ d.ProcessInstanceFilter, _ d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
	items := make([]d.ProcessInstance, 0, s.roots)
	for i := range s.roots {
		key := "root-" + strconv.Itoa(i)
		items = append(items, d.ProcessInstance{Key: key, RootProcessInstanceKey: key, State: d.StateActive})
	}
	return d.ProcessInstancePage{Items: items}, nil
}

// AncestryResult delegates ancestry expansion to the configured test callback.
func (s workerPreviewProcessInstanceAPI) AncestryResult(ctx context.Context, key string, opts ...services.CallOption) (pitraversal.Result, error) {
	return s.ancestry(ctx, key, opts...)
}

// DescendantsResult delegates descendant expansion to the configured test callback.
func (s workerPreviewProcessInstanceAPI) DescendantsResult(ctx context.Context, key string, opts ...services.CallOption) (pitraversal.Result, error) {
	return s.descendants(ctx, key, opts...)
}

// TestPreviewDeleteProcessDefinitionImpactUsesRequestedWorkers verifies delete-plan PI traversal honors APD worker settings.
func TestPreviewDeleteProcessDefinitionImpactUsesRequestedWorkers(t *testing.T) {
	const roots = 40

	var ancestryStarted atomic.Int64
	var descendantsStarted atomic.Int64
	ancestryRelease := make(chan struct{})
	descendantsRelease := make(chan struct{})
	var ancestryReleaseOnce sync.Once
	var descendantsReleaseOnce sync.Once
	t.Cleanup(func() {
		ancestryReleaseOnce.Do(func() { close(ancestryRelease) })
		descendantsReleaseOnce.Do(func() { close(descendantsRelease) })
	})

	piAPI := workerPreviewProcessInstanceAPI{
		roots: roots,
		ancestry: func(ctx context.Context, key string, _ ...services.CallOption) (pitraversal.Result, error) {
			ancestryStarted.Add(1)
			select {
			case <-ctx.Done():
				return pitraversal.Result{}, ctx.Err()
			case <-ancestryRelease:
				return pitraversal.Result{StartKey: key, RootKey: key, Keys: []string{key}, Outcome: pitraversal.OutcomeComplete}, nil
			}
		},
		descendants: func(ctx context.Context, key string, _ ...services.CallOption) (pitraversal.Result, error) {
			descendantsStarted.Add(1)
			select {
			case <-ctx.Done():
				return pitraversal.Result{}, ctx.Err()
			case <-descendantsRelease:
				return pitraversal.Result{StartKey: key, RootKey: key, Keys: []string{key}, Outcome: pitraversal.OutcomeComplete}, nil
			}
		},
	}

	type previewResult struct {
		item d.DeleteProcessDefinitionPlanItem
		err  error
	}
	resultCh := make(chan previewResult, 1)
	go func() {
		item, err := previewDeleteProcessDefinitionImpact(
			context.Background(),
			workerPreviewProcessDefinitionAPI{active: roots},
			piAPI,
			"pd-1",
			true,
			false,
			roots,
		)
		resultCh <- previewResult{item: item, err: err}
	}()

	require.Eventually(t, func() bool {
		return ancestryStarted.Load() == roots
	}, time.Second, 10*time.Millisecond)
	ancestryReleaseOnce.Do(func() { close(ancestryRelease) })
	require.Eventually(t, func() bool {
		return descendantsStarted.Load() == roots
	}, time.Second, 10*time.Millisecond)
	descendantsReleaseOnce.Do(func() { close(descendantsRelease) })
	result := <-resultCh
	require.NoError(t, result.err)
	require.Len(t, result.item.CancellationPlan.Roots, roots)
}
