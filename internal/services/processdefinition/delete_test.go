// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processdefinition

import (
	"bytes"
	"context"
	"errors"
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
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
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

type cleanupEligibilityProcessInstanceAPI struct {
	pisvc.API
	searchPage func(context.Context, d.ProcessInstanceFilter, d.ProcessInstancePageRequest, ...services.CallOption) (d.ProcessInstancePage, error)
}

// SearchForProcessInstancesPage delegates cleanup-eligibility discovery to the configured page callback.
func (s cleanupEligibilityProcessInstanceAPI) SearchForProcessInstancesPage(ctx context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
	return s.searchPage(ctx, filter, page, opts...)
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

type processDefinitionProgressEvents struct {
	mu     sync.Mutex
	events []d.OpsProgressEvent
}

// Append stores progress events safely from concurrent delete workers.
func (e *processDefinitionProgressEvents) Append(event d.OpsProgressEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, event)
}

// Completions returns completion facts in arrival order.
func (e *processDefinitionProgressEvents) Completions() []d.OpsCompletionProgress {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]d.OpsCompletionProgress, 0, len(e.events))
	for _, event := range e.events {
		if event.Kind == d.OpsProgressEventKindCompletion && event.Completion != nil {
			out = append(out, *event.Completion)
		}
	}
	return out
}

// Stages returns stage-entry facts in arrival order.
func (e *processDefinitionProgressEvents) Stages() []d.OpsStageProgress {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]d.OpsStageProgress, 0, len(e.events))
	for _, event := range e.events {
		if event.Kind == d.OpsProgressEventKindStage && event.Stage != nil {
			out = append(out, *event.Stage)
		}
	}
	return out
}

type processDefinitionStageSequence struct {
	mu    sync.Mutex
	items []string
}

// Append records the observable ordering between stage events and backend operations.
func (s *processDefinitionStageSequence) Append(item string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
}

// Items returns the observed stage and operation sequence.
func (s *processDefinitionStageSequence) Items() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.items...)
}

type processDefinitionCallOptionSnapshots struct {
	mu    sync.Mutex
	items []services.CallCfg
}

// Append captures the effective call options observed by a fake backend call.
func (s *processDefinitionCallOptionSnapshots) Append(opts []services.CallOption) {
	cfg := services.ApplyCallOptions(opts)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, *cfg)
}

// Items returns the captured call-option snapshots in arrival order.
func (s *processDefinitionCallOptionSnapshots) Items() []services.CallCfg {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]services.CallCfg(nil), s.items...)
}

type unsupportedResourceDeleteAPI struct {
	testResourceDeleteAPI
}

func (unsupportedResourceDeleteAPI) SupportsProcessDefinitionHistoryDeletion() bool { return false }

// TestFindUnrelatedProcessInstancesForDefinitionPagesThroughMatches protects smoke-test cleanup eligibility from first-page blocker blind spots.
func TestFindUnrelatedProcessInstancesForDefinitionPagesThroughMatches(t *testing.T) {
	t.Parallel()

	pages := []d.ProcessInstancePage{
		{
			Items: []d.ProcessInstance{
				{Key: "owned-1"},
			},
			EndCursor:     "cursor-1",
			OverflowState: d.ProcessInstanceOverflowStateHasMore,
		},
		{
			Items: []d.ProcessInstance{
				{Key: "unrelated-1"},
			},
			OverflowState: d.ProcessInstanceOverflowStateNoMore,
		},
	}
	var gotFilters []d.ProcessInstanceFilter
	var gotPages []d.ProcessInstancePageRequest
	api := cleanupEligibilityProcessInstanceAPI{
		searchPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			call := len(gotPages)
			gotFilters = append(gotFilters, filter)
			gotPages = append(gotPages, page)
			require.Less(t, call, len(pages))
			return pages[call], nil
		},
	}

	got, err := FindUnrelatedProcessInstancesForDefinition(context.Background(), api, "pd-1", "", types.Keys{"owned-1"})

	require.NoError(t, err)
	require.Equal(t, []d.ProcessInstance{{Key: "unrelated-1"}}, got)
	require.Len(t, gotFilters, 2)
	assert.Equal(t, d.ProcessInstanceFilter{ProcessDefinitionKey: "pd-1"}, gotFilters[0])
	assert.Equal(t, d.ProcessInstanceFilter{ProcessDefinitionKey: "pd-1"}, gotFilters[1])
	require.Equal(t, []d.ProcessInstancePageRequest{
		{Size: MaxResultSize},
		{From: 1, Size: MaxResultSize, After: "cursor-1"},
	}, gotPages)
}

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

// TestPreviewDeleteProcessDefinitionsNoStateCheckUsesBatchActivity verifies delete impact planning stays below workflow progress.
func TestPreviewDeleteProcessDefinitionsNoStateCheckUsesBatchActivity(t *testing.T) {
	sink := &activitysink.Sink{}

	got, err := PreviewDeleteProcessDefinitions(
		logging.ToActivityContext(context.Background(), sink),
		nil,
		nil,
		slog.Default(),
		types.Keys{"pd-1", "pd-2"},
		services.WithNoStateCheck(),
	)

	require.NoError(t, err)
	require.Len(t, got.Items, 2)
	require.Equal(t, []activitysink.Start{{
		Message:    "checking 2 pd delete impact; pi state skipped, dry run",
		Importance: logging.ActivityImportanceBatch,
	}}, sink.Starts())
}

// TestPreviewDeleteProcessDefinitionsAggregatesTenantEvidence verifies process-definition
// impact planning preserves distinct tenant evidence from definitions and nested
// cancellation subplans while counting unavailable tenant metadata once per target.
func TestPreviewDeleteProcessDefinitionsAggregatesTenantEvidence(t *testing.T) {
	pdAPI := tenantEvidenceProcessDefinitionAPI{
		definitions: map[string]d.ProcessDefinition{
			"pd-cross": {
				Key:           "pd-cross",
				TenantId:      "tenant-b",
				Statistics:    &d.ProcessDefinitionStatistics{Active: 3},
				BpmnProcessId: "invoice",
			},
			"pd-unknown": {
				Key:           "pd-unknown",
				Statistics:    &d.ProcessDefinitionStatistics{},
				BpmnProcessId: "invoice",
			},
		},
	}
	piAPI := tenantEvidenceProcessInstanceAPI{}

	got, err := PreviewDeleteProcessDefinitionsWithWorkers(
		context.Background(),
		pdAPI,
		piAPI,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		types.Keys{"pd-cross", "pd-unknown"},
		1,
		services.WithForce(),
	)

	require.NoError(t, err)
	require.Equal(t, d.TenantEvidence{
		ResolvedTenantIDs:  []string{"tenant-a", "tenant-b"},
		UnknownTargetCount: 2,
		TargetCount:        5,
		Targets: []d.TenantEvidenceTarget{
			{Key: "pd-cross", TenantID: "tenant-b"},
			{Key: "root-a", TenantID: "tenant-a"},
			{Key: "root-b", TenantID: "tenant-b"},
			{Key: "root-unknown"},
			{Key: "pd-unknown"},
		},
	}, got.TenantEvidence)
}

// TestProcessDefinitionPlanTenantEvidenceAggregateOnlyFallbackDoesNotDoubleCountKnownTenants
// verifies legacy nested evidence can contribute tenant IDs without turning those IDs
// into synthetic affected targets in the aggregate count.
func TestProcessDefinitionPlanTenantEvidenceAggregateOnlyFallbackDoesNotDoubleCountKnownTenants(t *testing.T) {
	t.Parallel()

	got := processDefinitionPlanTenantEvidence([]d.DeleteProcessDefinitionPlanItem{
		{
			Key:      "pd-1",
			TenantId: "tenant-b",
			CancellationPlan: d.DryRunPIKeyExpansion{TenantEvidence: d.TenantEvidence{
				ResolvedTenantIDs:  []string{"tenant-a"},
				UnknownTargetCount: 1,
				TargetCount:        3,
			}},
		},
	})

	require.Equal(t, d.TenantEvidence{
		ResolvedTenantIDs:  []string{"tenant-a", "tenant-b"},
		UnknownTargetCount: 1,
		TargetCount:        4,
		Targets: []d.TenantEvidenceTarget{
			{Key: "pd-1", TenantID: "tenant-b"},
		},
	}, got)
}

// TestDeleteProcessDefinitionResourcesStopsOnDeleteHistoryRequestShapeError verifies a server-side request-shape mismatch is reported once instead of being repeated for every selected definition.
func TestDeleteProcessDefinitionResourcesStopsOnDeleteHistoryRequestShapeError(t *testing.T) {
	var calls atomic.Int64
	events := &processDefinitionProgressEvents{}
	sequence := &processDefinitionStageSequence{}
	progress := func(event d.OpsProgressEvent) {
		if event.Kind == d.OpsProgressEventKindStage && event.Stage != nil {
			sequence.Append("stage:" + event.Stage.Phase)
		}
		events.Append(event)
	}
	api := testResourceDeleteAPI{
		delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
			calls.Add(1)
			sequence.Append("delete:" + key)
			require.Equal(t, "pd-1", key)
			return d.ResourceDeleteResponse{Key: key}, fmt.Errorf("%w: 400 POST /v2/resources/%s/deletion (Request property [deleteHistory] cannot be parsed)", d.ErrBadRequest, key)
		},
	}
	plans := []d.DeleteProcessDefinitionPlanItem{
		{Key: "pd-1"},
		{Key: "pd-2"},
		{Key: "pd-3"},
	}

	got, err := DeleteProcessDefinitionResources(
		context.Background(),
		api,
		cleanupProcessDefinitionAPI{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		plans,
		10,
		services.WithProgress(progress),
	)

	require.Error(t, err)
	require.ErrorIs(t, err, d.ErrBadRequest)
	require.ErrorContains(t, err, "deleteHistory")
	require.ErrorContains(t, err, "before submitting 2 remaining process-definition delete request(s)")
	require.Equal(t, int64(1), calls.Load())
	require.Len(t, got, 1)
	require.Equal(t, "pd-1", got[0].Key)
	completions := events.Completions()
	require.Len(t, completions, 1)
	require.Equal(t, "delete process definitions", completions[0].Phase)
	require.Equal(t, "process definition(s)", completions[0].CoreResource)
	require.Equal(t, 3, completions[0].Total)
	require.Equal(t, "pd-1", completions[0].Identity)
	require.Equal(t, d.OpsCompletionDispositionFailed, completions[0].Disposition)
	require.Contains(t, completions[0].FailureDetail, "deleteHistory")
	require.Empty(t, completions[0].AffectedResource)
	require.Nil(t, completions[0].AffectedCount)
	total := 3
	require.Equal(t, []d.OpsStageProgress{{
		Phase:        "delete process definitions",
		CoreResource: "process definition(s)",
		Total:        &total,
	}}, events.Stages())
	require.Equal(t, []string{
		"stage:delete process definitions",
		"delete:pd-1",
	}, sequence.Items())
}

// TestDeleteProcessDefinitionResourcesEmitsCompletionFactsForSerialProbeAndRemainder
// verifies basic process-definition deletion reports the serial first probe and
// every executed remainder delete as one process-definition completion.
func TestDeleteProcessDefinitionResourcesEmitsCompletionFactsForSerialProbeAndRemainder(t *testing.T) {
	events := &processDefinitionProgressEvents{}
	api := testResourceDeleteAPI{
		delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
			return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: 200, Status: "200 OK", BatchOperationKey: "batch-" + key, BatchState: "COMPLETED"}, nil
		},
	}
	plans := []d.DeleteProcessDefinitionPlanItem{
		{Key: "pd-1"},
		{Key: "pd-2"},
		{Key: "pd-3"},
	}

	got, err := DeleteProcessDefinitionResources(
		context.Background(),
		api,
		&deleteVisibilityProcessDefinitionAPI{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		plans,
		1,
		services.WithProgress(events.Append),
	)

	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, []d.OpsCompletionProgress{
		{
			Phase:        "delete process definitions",
			CoreResource: "process definition(s)",
			Total:        3,
			Identity:     "pd-1",
			Disposition:  d.OpsCompletionDispositionConfirmed,
		},
		{
			Phase:        "delete process definitions",
			CoreResource: "process definition(s)",
			Total:        3,
			Identity:     "pd-2",
			Disposition:  d.OpsCompletionDispositionConfirmed,
		},
		{
			Phase:        "delete process definitions",
			CoreResource: "process definition(s)",
			Total:        3,
			Identity:     "pd-3",
			Disposition:  d.OpsCompletionDispositionConfirmed,
		},
	}, events.Completions())
}

// TestDeleteProcessDefinitionResourcesEmitsSubmittedCompletionFactsForNoWait
// verifies accepted no-wait process-definition deletes are not promoted to a
// confirmed lifecycle disposition by service progress facts.
func TestDeleteProcessDefinitionResourcesEmitsSubmittedCompletionFactsForNoWait(t *testing.T) {
	events := &processDefinitionProgressEvents{}
	api := testResourceDeleteAPI{
		delete: func(_ context.Context, key string, opts ...services.CallOption) (d.ResourceDeleteResponse, error) {
			require.True(t, services.ApplyCallOptions(opts).NoWait)
			return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: 202, Status: "202 Accepted", BatchOperationKey: "batch-" + key}, nil
		},
	}

	got, err := DeleteProcessDefinitionResources(
		context.Background(),
		api,
		cleanupProcessDefinitionAPI{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		[]d.DeleteProcessDefinitionPlanItem{{Key: "pd-1"}, {Key: "pd-2"}},
		1,
		services.WithNoWait(),
		services.WithProgress(events.Append),
	)

	require.NoError(t, err)
	require.Len(t, got, 2)
	completions := events.Completions()
	require.Len(t, completions, 2)
	for _, completion := range completions {
		require.Equal(t, "delete process definitions", completion.Phase)
		require.Equal(t, "process definition(s)", completion.CoreResource)
		require.Equal(t, 2, completion.Total)
		require.Equal(t, d.OpsCompletionDispositionSubmitted, completion.Disposition)
		require.Empty(t, completion.FailureDetail)
	}
}

// TestDeleteProcessDefinitionResourcesFailFastOmitsUnscheduledCompletionFacts
// verifies fail-fast still preserves the first serial request probe, then stops
// worker scheduling without inventing progress for definitions never submitted.
func TestDeleteProcessDefinitionResourcesFailFastOmitsUnscheduledCompletionFacts(t *testing.T) {
	wantErr := errors.New("resource delete failed")
	events := &processDefinitionProgressEvents{}
	sequence := &processDefinitionStageSequence{}
	progress := func(event d.OpsProgressEvent) {
		if event.Kind == d.OpsProgressEventKindStage && event.Stage != nil {
			sequence.Append("stage:" + event.Stage.Phase)
		}
		events.Append(event)
	}
	api := testResourceDeleteAPI{
		delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
			sequence.Append("delete:" + key)
			if key == "pd-2" {
				return d.ResourceDeleteResponse{Key: key, Status: "500 Internal Server Error"}, wantErr
			}
			return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: 200, Status: "200 OK", BatchOperationKey: "batch-" + key, BatchState: "COMPLETED"}, nil
		},
	}
	pdAPI := &deleteVisibilityProcessDefinitionAPI{}

	got, err := DeleteProcessDefinitionResources(
		context.Background(),
		api,
		pdAPI,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		[]d.DeleteProcessDefinitionPlanItem{{Key: "pd-1"}, {Key: "pd-2"}, {Key: "pd-3"}},
		1,
		services.WithFailFast(),
		services.WithProgress(progress),
	)

	require.ErrorIs(t, err, wantErr)
	require.Len(t, got, 3)
	require.Equal(t, "pd-1", got[0].Key)
	require.Equal(t, "pd-2", got[1].Key)
	require.Empty(t, got[2].Key)
	require.Equal(t, int64(1), pdAPI.calls.Load())
	total := 3
	require.Equal(t, []d.OpsStageProgress{{
		Phase:        "delete process definitions",
		CoreResource: "process definition(s)",
		Total:        &total,
	}}, events.Stages())
	require.Equal(t, []d.OpsCompletionProgress{
		{
			Phase:        "delete process definitions",
			CoreResource: "process definition(s)",
			Total:        3,
			Identity:     "pd-1",
			Disposition:  d.OpsCompletionDispositionConfirmed,
		},
		{
			Phase:         "delete process definitions",
			CoreResource:  "process definition(s)",
			Total:         3,
			Identity:      "pd-2",
			Disposition:   d.OpsCompletionDispositionFailed,
			FailureDetail: "resource delete failed",
		},
	}, events.Completions())
	require.Equal(t, []string{
		"stage:delete process definitions",
		"delete:pd-1",
		"delete:pd-2",
	}, sequence.Items())
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

// TestDeleteProcessDefinitionResourcesEmitsStageEntryBeforeSerialProbe verifies
// preplanned definition deletion announces the exact definition total before the
// first serial resource request.
func TestDeleteProcessDefinitionResourcesEmitsStageEntryBeforeSerialProbe(t *testing.T) {
	events := &processDefinitionProgressEvents{}
	sequence := &processDefinitionStageSequence{}
	progress := func(event d.OpsProgressEvent) {
		if event.Kind == d.OpsProgressEventKindStage && event.Stage != nil {
			sequence.Append("stage:" + event.Stage.Phase)
		}
		events.Append(event)
	}
	api := testResourceDeleteAPI{
		delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
			sequence.Append("delete:" + key)
			return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: 202, Status: "202 Accepted", BatchOperationKey: "batch-" + key}, nil
		},
	}

	got, err := DeleteProcessDefinitionResources(
		context.Background(),
		api,
		cleanupProcessDefinitionAPI{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		[]d.DeleteProcessDefinitionPlanItem{{Key: "pd-1"}, {Key: "pd-2"}},
		4,
		services.WithNoWait(),
		services.WithProgress(progress),
	)

	require.NoError(t, err)
	require.Len(t, got, 2)
	total := 2
	require.Equal(t, []d.OpsStageProgress{{
		Phase:        "delete process definitions",
		CoreResource: "process definition(s)",
		Total:        &total,
	}}, events.Stages())
	require.Equal(t, []string{
		"stage:delete process definitions",
		"delete:pd-1",
		"delete:pd-2",
	}, sequence.Items())
}

type staticProcessDefinitionAPI struct {
	API
}

// GetProcessDefinition returns inactive process-definition statistics for
// ordinary worker-path delete validation.
func (staticProcessDefinitionAPI) GetProcessDefinition(_ context.Context, key string, _ ...services.CallOption) (d.ProcessDefinition, error) {
	return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{}}, nil
}

// TestDeleteProcessDefinitionsNonForceEmitsOneDefinitionStageAfterValidation
// verifies the ordinary worker path emits one exact definition-entry event only
// after a validated item reaches resource deletion.
func TestDeleteProcessDefinitionsNonForceEmitsOneDefinitionStageAfterValidation(t *testing.T) {
	events := &processDefinitionProgressEvents{}
	sequence := &processDefinitionStageSequence{}
	progress := func(event d.OpsProgressEvent) {
		if event.Kind == d.OpsProgressEventKindStage && event.Stage != nil {
			sequence.Append("stage:" + event.Stage.Phase)
		}
		events.Append(event)
	}
	api := testResourceDeleteAPI{
		delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
			sequence.Append("delete:" + key)
			return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: 202, Status: "202 Accepted", BatchOperationKey: "batch-" + key}, nil
		},
	}

	got, err := DeleteProcessDefinitions(
		context.Background(),
		api,
		staticProcessDefinitionAPI{},
		cleanupProcessInstanceAPI{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		types.Keys{"pd-1", "pd-1", "pd-2"},
		1,
		services.WithNoWait(),
		services.WithProgress(progress),
	)

	require.NoError(t, err)
	require.Len(t, got, 2)
	total := 2
	require.Equal(t, []d.OpsStageProgress{{
		Phase:        "delete process definitions",
		CoreResource: "process definition(s)",
		Total:        &total,
	}}, events.Stages())
	require.Equal(t, []string{
		"stage:delete process definitions",
		"delete:pd-1",
		"delete:pd-2",
	}, sequence.Items())
}

// GetProcessDefinition returns inactive statistics so force cleanup can proceed after cancellation.
func (cleanupProcessDefinitionAPI) GetProcessDefinition(_ context.Context, key string, _ ...services.CallOption) (d.ProcessDefinition, error) {
	return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{}}, nil
}

type sequenceProcessDefinitionAPI struct {
	API
	sequence *processDefinitionStageSequence
	active   int64
}

// GetProcessDefinition records drain polling while returning configured active statistics.
func (s sequenceProcessDefinitionAPI) GetProcessDefinition(_ context.Context, key string, _ ...services.CallOption) (d.ProcessDefinition, error) {
	s.sequence.Append("get:" + key)
	return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{Active: s.active}}, nil
}

type controlledProcessDefinitionAPI struct {
	API
	sequence *processDefinitionStageSequence
	options  *processDefinitionCallOptionSnapshots
	active   int64
	err      error
	afterGet func()
}

// GetProcessDefinition records drain polling and returns configured active
// statistics or an injected error for failure-boundary tests.
func (s controlledProcessDefinitionAPI) GetProcessDefinition(_ context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
	if s.sequence != nil {
		s.sequence.Append("get:" + key)
	}
	if s.options != nil {
		s.options.Append(opts)
	}
	if s.afterGet != nil {
		s.afterGet()
	}
	if s.err != nil {
		return d.ProcessDefinition{}, s.err
	}
	return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{Active: s.active}}, nil
}

// TestCleanupProcessDefinitionDeletePlanForceScopeEmitsStageEntriesBeforeOperations
// verifies force cleanup enters cancellation, drain, and history stages before
// their backing service operations while keeping waiting unquantified.
func TestCleanupProcessDefinitionDeletePlanForceScopeEmitsStageEntriesBeforeOperations(t *testing.T) {
	events := &processDefinitionProgressEvents{}
	sequence := &processDefinitionStageSequence{}
	progress := func(event d.OpsProgressEvent) {
		if event.Kind == d.OpsProgressEventKindStage && event.Stage != nil {
			sequence.Append("stage:" + event.Stage.Phase)
		}
		events.Append(event)
	}
	piAPI := cleanupProcessInstanceAPI{
		cancel: func(_ context.Context, key string, _ ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
			sequence.Append("cancel:" + key)
			return d.CancelResponse{Ok: true, StatusCode: 200, Status: "200 OK"}, []d.ProcessInstance{{Key: key}}, nil
		},
		delete: func(_ context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			sequence.Append("delete:" + key)
			return d.DeleteResponse{Ok: true, StatusCode: 200, Status: "200 OK"}, nil
		},
	}
	plans := []d.DeleteProcessDefinitionPlanItem{
		{
			Key: "pd-1",
			CancellationPlan: d.DryRunPIKeyExpansion{
				Roots:     []string{"root-1", "root-shared"},
				Collected: []string{"root-1", "child-1", "root-shared"},
			},
		},
		{
			Key: "pd-2",
			CancellationPlan: d.DryRunPIKeyExpansion{
				Roots:     []string{"root-shared", "root-2"},
				Collected: []string{"root-shared", "child-2", "root-2"},
			},
		},
	}

	err := cleanupProcessDefinitionDeletePlanForceScope(
		context.Background(),
		sequenceProcessDefinitionAPI{sequence: sequence},
		piAPI,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		plans,
		1,
		services.WithProgress(progress),
	)

	require.NoError(t, err)
	totalRoots := 3
	plannedAffected := 5
	require.Equal(t, []d.OpsStageProgress{
		{
			Phase:                "cancel",
			CoreResource:         "process-instance tree(s)",
			Total:                &totalRoots,
			PlannedAffectedCount: &plannedAffected,
		},
		{
			Phase:        "drain process instances",
			CoreResource: "",
		},
		{
			Phase:        "delete",
			CoreResource: "process-instance tree(s)",
			Total:        &totalRoots,
		},
	}, events.Stages())
	require.Equal(t, []string{
		"stage:cancel",
		"cancel:root-1",
		"cancel:root-shared",
		"cancel:root-2",
		"stage:drain process instances",
		"get:pd-1",
		"get:pd-2",
		"stage:delete",
		"delete:root-1",
		"delete:root-shared",
		"delete:root-2",
	}, sequence.Items())
}

// TestCleanupProcessDefinitionDeletePlanForceScopeStopsAfterCancellationFailure
// verifies a failed root cancellation reports only the entered cancel stage and
// preserves caller options while suppressing nested detail logs.
func TestCleanupProcessDefinitionDeletePlanForceScopeStopsAfterCancellationFailure(t *testing.T) {
	wantErr := errors.New("cancel failed")
	events := &processDefinitionProgressEvents{}
	sequence := &processDefinitionStageSequence{}
	cancelOptions := &processDefinitionCallOptionSnapshots{}
	var deleteAttempts atomic.Int64
	progress := func(event d.OpsProgressEvent) {
		if event.Kind == d.OpsProgressEventKindStage && event.Stage != nil {
			sequence.Append("stage:" + event.Stage.Phase)
		}
		events.Append(event)
	}
	piAPI := cleanupProcessInstanceAPI{
		cancel: func(_ context.Context, key string, opts ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
			sequence.Append("cancel:" + key)
			cancelOptions.Append(opts)
			return d.CancelResponse{Status: "500 Internal Server Error"}, nil, wantErr
		},
		delete: func(context.Context, string, ...services.CallOption) (d.DeleteResponse, error) {
			deleteAttempts.Add(1)
			return d.DeleteResponse{}, nil
		},
	}
	plans := []d.DeleteProcessDefinitionPlanItem{{
		Key: "pd-1",
		CancellationPlan: d.DryRunPIKeyExpansion{
			Roots:     []string{"root-1", "root-2"},
			Collected: []string{"root-1", "root-2", "child-1"},
		},
	}}

	err := cleanupProcessDefinitionDeletePlanForceScope(
		context.Background(),
		cleanupProcessDefinitionAPI{},
		piAPI,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		plans,
		1,
		services.WithNoWait(),
		services.WithFailFast(),
		services.WithProgress(progress),
	)

	require.ErrorIs(t, err, wantErr)
	totalRoots := 2
	plannedAffected := 3
	require.Equal(t, []d.OpsStageProgress{{
		Phase:                "cancel",
		CoreResource:         "process-instance tree(s)",
		Total:                &totalRoots,
		PlannedAffectedCount: &plannedAffected,
	}}, events.Stages())
	require.Equal(t, []d.OpsCompletionProgress{{
		Phase:            "cancel",
		CoreResource:     "process-instance tree(s)",
		Total:            2,
		Identity:         "root-1",
		Disposition:      d.OpsCompletionDispositionFailed,
		FailureDetail:    "cancel failed",
		AffectedResource: "affected process instances",
	}}, events.Completions())
	require.Equal(t, []string{
		"stage:cancel",
		"cancel:root-1",
	}, sequence.Items())
	require.Equal(t, int64(0), deleteAttempts.Load())
	require.Len(t, cancelOptions.Items(), 1)
	cfg := cancelOptions.Items()[0]
	require.True(t, cfg.NoWait)
	require.True(t, cfg.FailFast)
	require.True(t, cfg.SuppressWorkflowDetailLogs)
	require.True(t, cfg.SuppressProcessInstanceDetailLogs)
	require.Equal(t, 3, cfg.AffectedProcessInstanceCount)
}

// TestCleanupProcessDefinitionDeletePlanForceScopeStopsAfterDrainFailure
// verifies drain errors and interruptions never advance to history deletion.
func TestCleanupProcessDefinitionDeletePlanForceScopeStopsAfterDrainFailure(t *testing.T) {
	tests := []struct {
		name        string
		configurePD func(*testing.T, *processDefinitionStageSequence) (context.Context, API, error)
	}{
		{
			name: "backend error",
			configurePD: func(t *testing.T, sequence *processDefinitionStageSequence) (context.Context, API, error) {
				t.Helper()
				wantErr := errors.New("active count failed")
				return context.Background(), controlledProcessDefinitionAPI{
					sequence: sequence,
					err:      wantErr,
				}, wantErr
			},
		},
		{
			name: "deadline exceeded",
			configurePD: func(t *testing.T, sequence *processDefinitionStageSequence) (context.Context, API, error) {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
				t.Cleanup(cancel)
				return ctx, controlledProcessDefinitionAPI{
					sequence: sequence,
					active:   1,
				}, context.DeadlineExceeded
			},
		},
		{
			name: "context canceled",
			configurePD: func(t *testing.T, sequence *processDefinitionStageSequence) (context.Context, API, error) {
				t.Helper()
				ctx, cancel := context.WithCancel(context.Background())
				t.Cleanup(cancel)
				return ctx, controlledProcessDefinitionAPI{
					sequence: sequence,
					active:   1,
					afterGet: cancel,
				}, context.Canceled
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := &processDefinitionProgressEvents{}
			sequence := &processDefinitionStageSequence{}
			progress := func(event d.OpsProgressEvent) {
				if event.Kind == d.OpsProgressEventKindStage && event.Stage != nil {
					sequence.Append("stage:" + event.Stage.Phase)
				}
				events.Append(event)
			}
			ctx, pdAPI, wantErr := tt.configurePD(t, sequence)
			var deleteAttempts atomic.Int64
			piAPI := cleanupProcessInstanceAPI{
				cancel: func(_ context.Context, key string, _ ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
					sequence.Append("cancel:" + key)
					return d.CancelResponse{Ok: true, StatusCode: 200, Status: "200 OK"}, []d.ProcessInstance{{Key: key}}, nil
				},
				delete: func(context.Context, string, ...services.CallOption) (d.DeleteResponse, error) {
					deleteAttempts.Add(1)
					return d.DeleteResponse{}, nil
				},
			}

			err := cleanupProcessDefinitionDeletePlanForceScope(
				ctx,
				pdAPI,
				piAPI,
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				[]d.DeleteProcessDefinitionPlanItem{{
					Key: "pd-1",
					CancellationPlan: d.DryRunPIKeyExpansion{
						Roots:     []string{"root-1"},
						Collected: []string{"root-1"},
					},
				}},
				1,
				services.WithProgress(progress),
			)

			require.ErrorIs(t, err, wantErr)
			totalRoots := 1
			plannedAffected := 1
			require.Equal(t, []d.OpsStageProgress{
				{
					Phase:                "cancel",
					CoreResource:         "process-instance tree(s)",
					Total:                &totalRoots,
					PlannedAffectedCount: &plannedAffected,
				},
				{
					Phase:        "drain process instances",
					CoreResource: "",
				},
			}, events.Stages())
			require.Equal(t, []string{
				"stage:cancel",
				"cancel:root-1",
				"stage:drain process instances",
				"get:pd-1",
			}, sequence.Items())
			require.Equal(t, int64(0), deleteAttempts.Load())
		})
	}
}

// TestCleanupProcessDefinitionDeletePlanForceScopeStopsAfterHistoryFailure
// verifies history-delete failures retain the delete stage without announcing
// any process-definition resource deletion.
func TestCleanupProcessDefinitionDeletePlanForceScopeStopsAfterHistoryFailure(t *testing.T) {
	wantErr := errors.New("history delete failed")
	events := &processDefinitionProgressEvents{}
	sequence := &processDefinitionStageSequence{}
	deleteOptions := &processDefinitionCallOptionSnapshots{}
	progress := func(event d.OpsProgressEvent) {
		if event.Kind == d.OpsProgressEventKindStage && event.Stage != nil {
			sequence.Append("stage:" + event.Stage.Phase)
		}
		events.Append(event)
	}
	piAPI := cleanupProcessInstanceAPI{
		cancel: func(_ context.Context, key string, _ ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
			sequence.Append("cancel:" + key)
			return d.CancelResponse{Ok: true, StatusCode: 200, Status: "200 OK"}, []d.ProcessInstance{{Key: key}}, nil
		},
		delete: func(_ context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error) {
			sequence.Append("delete:" + key)
			deleteOptions.Append(opts)
			return d.DeleteResponse{Status: "500 Internal Server Error"}, wantErr
		},
	}

	err := cleanupProcessDefinitionDeletePlanForceScope(
		context.Background(),
		controlledProcessDefinitionAPI{sequence: sequence},
		piAPI,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		[]d.DeleteProcessDefinitionPlanItem{{
			Key: "pd-1",
			CancellationPlan: d.DryRunPIKeyExpansion{
				Roots:     []string{"root-1"},
				Collected: []string{"root-1", "child-1"},
			},
		}},
		1,
		services.WithNoWait(),
		services.WithFailFast(),
		services.WithProgress(progress),
	)

	require.ErrorIs(t, err, wantErr)
	totalRoots := 1
	plannedAffected := 2
	require.Equal(t, []d.OpsStageProgress{
		{
			Phase:                "cancel",
			CoreResource:         "process-instance tree(s)",
			Total:                &totalRoots,
			PlannedAffectedCount: &plannedAffected,
		},
		{
			Phase:        "drain process instances",
			CoreResource: "",
		},
		{
			Phase:        "delete",
			CoreResource: "process-instance tree(s)",
			Total:        &totalRoots,
		},
	}, events.Stages())
	require.Equal(t, []string{
		"stage:cancel",
		"cancel:root-1",
		"stage:drain process instances",
		"get:pd-1",
		"stage:delete",
		"delete:root-1",
	}, sequence.Items())
	completions := events.Completions()
	require.Len(t, completions, 2)
	require.Equal(t, d.OpsCompletionDispositionSubmitted, completions[0].Disposition)
	require.Equal(t, "cancel", completions[0].Phase)
	require.Equal(t, d.OpsCompletionDispositionFailed, completions[1].Disposition)
	require.Equal(t, "delete", completions[1].Phase)
	require.Equal(t, "history delete failed", completions[1].FailureDetail)
	require.Len(t, deleteOptions.Items(), 1)
	cfg := deleteOptions.Items()[0]
	require.True(t, cfg.NoWait)
	require.True(t, cfg.FailFast)
	require.True(t, cfg.SuppressWorkflowDetailLogs)
	require.True(t, cfg.SuppressProcessInstanceDetailLogs)
	require.Equal(t, 2, cfg.AffectedProcessInstanceCount)
}

// TestCleanupProcessDefinitionDeletePlanForceScopeSkipsEmptyCleanupStages
// verifies no stage entry is invented when a bulk plan has no cleanup work.
func TestCleanupProcessDefinitionDeletePlanForceScopeSkipsEmptyCleanupStages(t *testing.T) {
	events := &processDefinitionProgressEvents{}
	piAPI := cleanupProcessInstanceAPI{
		cancel: func(context.Context, string, ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
			t.Fatal("process-instance cancellation must not be called")
			return d.CancelResponse{}, nil, nil
		},
		delete: func(context.Context, string, ...services.CallOption) (d.DeleteResponse, error) {
			t.Fatal("process-instance deletion must not be called")
			return d.DeleteResponse{}, nil
		},
	}

	err := cleanupProcessDefinitionDeletePlanForceScope(
		context.Background(),
		cleanupProcessDefinitionAPI{},
		piAPI,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		[]d.DeleteProcessDefinitionPlanItem{{Key: "pd-1"}},
		1,
		services.WithProgress(events.Append),
	)

	require.NoError(t, err)
	require.Empty(t, events.Stages())
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

type tenantEvidenceProcessDefinitionAPI struct {
	API
	definitions map[string]d.ProcessDefinition
}

// GetProcessDefinition returns configured process-definition metadata for
// tenant-evidence aggregation tests.
func (s tenantEvidenceProcessDefinitionAPI) GetProcessDefinition(_ context.Context, key string, _ ...services.CallOption) (d.ProcessDefinition, error) {
	pd, ok := s.definitions[key]
	if !ok {
		return d.ProcessDefinition{}, fmt.Errorf("%w: process definition %s not found", d.ErrNotFound, key)
	}
	return pd, nil
}

type tenantEvidenceProcessInstanceAPI struct {
	pisvc.API
}

// SearchForProcessInstancesPage returns duplicate known tenants plus one
// unknown active process-instance tenant for one process definition.
func (tenantEvidenceProcessInstanceAPI) SearchForProcessInstancesPage(_ context.Context, filter d.ProcessInstanceFilter, _ d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
	if filter.ProcessDefinitionKey != "pd-cross" {
		return d.ProcessInstancePage{Items: []d.ProcessInstance{}}, nil
	}
	return d.ProcessInstancePage{
		Items: []d.ProcessInstance{
			{Key: "root-a", RootProcessInstanceKey: "root-a", TenantId: "tenant-a", State: d.StateActive},
			{Key: "root-b", RootProcessInstanceKey: "root-b", TenantId: "tenant-b", State: d.StateActive},
			{Key: "root-unknown", RootProcessInstanceKey: "root-unknown", State: d.StateActive},
		},
	}, nil
}

// AncestryResult returns one known or unknown tenant observation for the
// requested root without making any enrichment-style lookup.
func (tenantEvidenceProcessInstanceAPI) AncestryResult(_ context.Context, key string, _ ...services.CallOption) (pitraversal.Result, error) {
	return tenantEvidenceTraversalResult(key), nil
}

// DescendantsResult mirrors ancestry evidence so duplicate traversal branches
// must be deduplicated by the shared process-instance dry-run planner.
func (tenantEvidenceProcessInstanceAPI) DescendantsResult(_ context.Context, key string, _ ...services.CallOption) (pitraversal.Result, error) {
	return tenantEvidenceTraversalResult(key), nil
}

// tenantEvidenceTraversalResult creates a single-node traversal result with
// deterministic tenant metadata for aggregation assertions.
func tenantEvidenceTraversalResult(key string) pitraversal.Result {
	tenantID := map[string]string{
		"root-a": "tenant-a",
		"root-b": "tenant-b",
	}[key]
	return pitraversal.Result{
		StartKey: key,
		RootKey:  key,
		Keys:     []string{key},
		Chain: map[string]d.ProcessInstance{
			key: {Key: key, TenantId: tenantID, State: d.StateActive},
		},
		Outcome: pitraversal.OutcomeComplete,
	}
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
