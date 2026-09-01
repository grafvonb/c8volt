// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

type stubProcessInstanceCreator struct {
	create func(context.Context, d.ProcessInstanceData, ...services.CallOption) (d.ProcessInstanceCreation, error)
}

// CreateProcessInstance delegates creation to the configured test callback.
func (s stubProcessInstanceCreator) CreateProcessInstance(ctx context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error) {
	if s.create == nil {
		return d.ProcessInstanceCreation{}, errors.New("unexpected process-instance creation")
	}
	return s.create(ctx, data, opts...)
}

// TestCreateProcessInstancesPreservesOrderAndOptions verifies the service-owned create-many workflow remains sequential and option-aware.
func TestCreateProcessInstancesPreservesOrderAndOptions(t *testing.T) {
	seen := []string{}
	got, err := CreateProcessInstances(context.Background(), stubProcessInstanceCreator{
		create: func(_ context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error) {
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			seen = append(seen, data.BpmnProcessId)
			return d.ProcessInstanceCreation{Key: "created-" + data.BpmnProcessId, BpmnProcessId: data.BpmnProcessId}, nil
		},
	}, []d.ProcessInstanceData{
		{BpmnProcessId: "alpha"},
		{BpmnProcessId: "beta"},
	}, services.WithIgnoreTenant())

	require.NoError(t, err)
	require.Equal(t, []string{"alpha", "beta"}, seen)
	require.Equal(t, []d.ProcessInstanceCreation{
		{Key: "created-alpha", BpmnProcessId: "alpha"},
		{Key: "created-beta", BpmnProcessId: "beta"},
	}, got)
}

// TestCreateProcessInstancesStopsOnFirstError verifies the previous fail-on-first-error behavior stays intact.
func TestCreateProcessInstancesStopsOnFirstError(t *testing.T) {
	seen := []string{}
	wantErr := errors.New("create failed")
	got, err := CreateProcessInstances(context.Background(), stubProcessInstanceCreator{
		create: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			seen = append(seen, data.BpmnProcessId)
			if data.BpmnProcessId == "beta" {
				return d.ProcessInstanceCreation{}, wantErr
			}
			return d.ProcessInstanceCreation{Key: "created-" + data.BpmnProcessId}, nil
		},
	}, []d.ProcessInstanceData{
		{BpmnProcessId: "alpha"},
		{BpmnProcessId: "beta"},
		{BpmnProcessId: "gamma"},
	})

	require.ErrorIs(t, err, wantErr)
	require.Nil(t, got)
	require.Equal(t, []string{"alpha", "beta"}, seen)
}

type stubBulkProcessInstanceAPI struct {
	API
	create func(context.Context, d.ProcessInstanceData, ...services.CallOption) (d.ProcessInstanceCreation, error)
	cancel func(context.Context, string, ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error)
	delete func(context.Context, string, ...services.CallOption) (d.DeleteResponse, error)
}

type lockedLogBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

type lockedProgressEvents struct {
	mu     sync.Mutex
	events []d.OpsProgressEvent
}

// Append stores progress events safely from concurrent worker callbacks.
func (e *lockedProgressEvents) Append(event d.OpsProgressEvent) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, event)
}

// Completions returns completion facts in arrival order.
func (e *lockedProgressEvents) Completions() []d.OpsCompletionProgress {
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

// FrozenScopes returns frozen-scope progress in arrival order.
func (e *lockedProgressEvents) FrozenScopes() []d.OpsFrozenScopeProgress {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]d.OpsFrozenScopeProgress, 0, len(e.events))
	for _, event := range e.events {
		if event.Kind == d.OpsProgressEventKindFrozenScope && event.FrozenScope != nil {
			out = append(out, *event.FrozenScope)
		}
	}
	return out
}

// Write appends log bytes while allowing concurrent progress logging and assertions.
func (b *lockedLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// String returns the buffered log output while writes may still be in flight.
func (b *lockedLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// CreateProcessInstance delegates creation to the configured test callback.
func (s stubBulkProcessInstanceAPI) CreateProcessInstance(ctx context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error) {
	if s.create == nil {
		return d.ProcessInstanceCreation{}, errors.New("unexpected process-instance creation")
	}
	return s.create(ctx, data, opts...)
}

// CancelProcessInstance delegates cancellation to the configured test callback.
func (s stubBulkProcessInstanceAPI) CancelProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
	if s.cancel == nil {
		return d.CancelResponse{}, nil, errors.New("unexpected process-instance cancellation")
	}
	return s.cancel(ctx, key, opts...)
}

// DeleteProcessInstance delegates deletion to the configured test callback.
func (s stubBulkProcessInstanceAPI) DeleteProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error) {
	if s.delete == nil {
		return d.DeleteResponse{}, errors.New("unexpected process-instance deletion")
	}
	return s.delete(ctx, key, opts...)
}

// TestCreateNProcessInstancesLogsActualSuccessAndFailureCounts verifies partial create failures do not log the requested count as fully created.
func TestCreateNProcessInstancesLogsActualSuccessAndFailureCounts(t *testing.T) {
	var logBuf lockedLogBuffer
	log := slog.New(logging.NewPlainHandler(&logBuf, slog.LevelInfo))
	wantErr := errors.New("create failed")
	attempt := 0
	api := stubBulkProcessInstanceAPI{
		create: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			attempt++
			require.Equal(t, "demo", data.BpmnProcessId)
			if attempt == 2 {
				return d.ProcessInstanceCreation{}, wantErr
			}
			return d.ProcessInstanceCreation{Key: fmt.Sprintf("pi-%d", attempt), BpmnProcessId: data.BpmnProcessId}, nil
		},
	}

	got, err := CreateNProcessInstances(context.Background(), api, log, d.ProcessInstanceData{BpmnProcessId: "demo"}, 3, 1)

	require.ErrorIs(t, err, wantErr)
	require.Len(t, got, 3)
	require.Contains(t, logBuf.String(), "creating pi done; requested 3, created 2, failed 1")
	require.NotContains(t, logBuf.String(), "creating pi done; created 3")
}

// TestCreateNProcessInstancesEmitsFrozenProgress verifies explicit-count starts expose exact create counters.
func TestCreateNProcessInstancesEmitsFrozenProgress(t *testing.T) {
	events := &lockedProgressEvents{}
	api := stubBulkProcessInstanceAPI{
		create: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			return d.ProcessInstanceCreation{Key: "created-" + data.BpmnProcessId, BpmnProcessId: data.BpmnProcessId}, nil
		},
	}

	got, err := CreateNProcessInstances(context.Background(), api, slog.Default(), d.ProcessInstanceData{BpmnProcessId: "demo"}, 2, 1, services.WithProgress(events.Append))

	require.NoError(t, err)
	require.Len(t, got, 2)
	frozenScopes := events.FrozenScopes()
	require.Equal(t, []d.OpsFrozenScopeProgress{
		{Phase: "starting process instances", CoreResource: "process instance(s)", Done: 0, Total: 2},
		{Phase: "starting process instances", CoreResource: "process instance(s)", Done: 1, Total: 2},
		{Phase: "starting process instances", CoreResource: "process instance(s)", Done: 2, Total: 2},
	}, frozenScopes)
}

// TestCreateNProcessInstancesEmitsCompletionFacts verifies explicit-count
// starts report one lifecycle fact per executed creation.
func TestCreateNProcessInstancesEmitsCompletionFacts(t *testing.T) {
	events := &lockedProgressEvents{}
	attempt := 0
	api := stubBulkProcessInstanceAPI{
		create: func(_ context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error) {
			require.True(t, services.ApplyCallOptions(opts).NoWait)
			attempt++
			return d.ProcessInstanceCreation{Key: fmt.Sprintf("pi-%d", attempt), BpmnProcessId: data.BpmnProcessId}, nil
		},
	}

	got, err := CreateNProcessInstances(context.Background(), api, slog.Default(), d.ProcessInstanceData{BpmnProcessId: "demo"}, 2, 1,
		services.WithNoWait(),
		services.WithProgress(events.Append),
	)

	require.NoError(t, err)
	require.Len(t, got, 2)
	completions := events.Completions()
	require.Len(t, completions, 2)
	for i, completion := range completions {
		require.Equal(t, "create", completion.Phase)
		require.Equal(t, "process instance(s)", completion.CoreResource)
		require.Equal(t, 2, completion.Total)
		require.Equal(t, fmt.Sprintf("pi-%d", i+1), completion.Identity)
		require.Equal(t, d.OpsCompletionDispositionSubmitted, completion.Disposition)
		require.Empty(t, completion.FailureDetail)
		require.Equal(t, "process instances", completion.AffectedResource)
		require.NotNil(t, completion.AffectedCount)
		require.Equal(t, 1, *completion.AffectedCount)
	}
}

// TestCancelProcessInstancesEmitsCompletionFacts verifies root-tree
// cancellation reports successes, non-OK failures, and trustworthy affected
// counts from the affected item set returned by the API.
func TestCancelProcessInstancesEmitsCompletionFacts(t *testing.T) {
	events := &lockedProgressEvents{}
	api := stubBulkProcessInstanceAPI{
		cancel: func(_ context.Context, key string, _ ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
			switch key {
			case "root-ok":
				return d.CancelResponse{Ok: true, StatusCode: 200, Status: "200 OK"}, []d.ProcessInstance{
					{Key: "root-ok"},
					{Key: "child-ok"},
				}, nil
			case "root-conflict":
				return d.CancelResponse{Ok: false, StatusCode: 409, Status: "409 Conflict"}, nil, nil
			default:
				return d.CancelResponse{}, nil, fmt.Errorf("unexpected key %s", key)
			}
		},
	}

	got, err := CancelProcessInstances(context.Background(), api, slog.Default(), typex.Keys{"root-ok", "root-conflict"}, 1, 2,
		services.WithProgress(events.Append),
	)

	require.NoError(t, err)
	require.Equal(t, []d.Reporter{
		{Key: "root-ok", Ok: true, StatusCode: 200, Status: "200 OK"},
		{Key: "root-conflict", Ok: false, StatusCode: 409, Status: "409 Conflict"},
	}, got)
	completions := events.Completions()
	require.Len(t, completions, 2)
	require.Equal(t, d.OpsCompletionProgress{
		Phase:            "cancel",
		CoreResource:     "process-instance tree(s)",
		Total:            2,
		Identity:         "root-ok",
		Disposition:      d.OpsCompletionDispositionConfirmed,
		AffectedResource: "affected process instances",
		AffectedCount:    intPtr(2),
	}, completions[0])
	require.Equal(t, d.OpsCompletionProgress{
		Phase:            "cancel",
		CoreResource:     "process-instance tree(s)",
		Total:            2,
		Identity:         "root-conflict",
		Disposition:      d.OpsCompletionDispositionFailed,
		FailureDetail:    "409 Conflict",
		AffectedResource: "affected process instances",
		AffectedCount:    intPtr(0),
	}, completions[1])
}

// TestDeleteProcessInstancesEmitsCompletionFactsWithUnknownExpandedAffectedCount
// verifies deletion avoids estimating per-root affected counts when only an
// aggregate expanded scope is available.
func TestDeleteProcessInstancesEmitsCompletionFactsWithUnknownExpandedAffectedCount(t *testing.T) {
	events := &lockedProgressEvents{}
	api := stubBulkProcessInstanceAPI{
		delete: func(_ context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			return d.DeleteResponse{Ok: true, StatusCode: 204, Status: "204 No Content"}, nil
		},
	}

	got, err := DeleteProcessInstances(context.Background(), api, slog.Default(), typex.Keys{"root-1", "root-2"}, 1, 5,
		services.WithProgress(events.Append),
	)

	require.NoError(t, err)
	require.Len(t, got, 2)
	completions := events.Completions()
	require.Len(t, completions, 2)
	for _, completion := range completions {
		require.Equal(t, "delete", completion.Phase)
		require.Equal(t, "process-instance tree(s)", completion.CoreResource)
		require.Equal(t, 2, completion.Total)
		require.Equal(t, d.OpsCompletionDispositionConfirmed, completion.Disposition)
		require.Empty(t, completion.FailureDetail)
		require.Equal(t, "affected process instances", completion.AffectedResource)
		require.Nil(t, completion.AffectedCount)
	}
}

// TestDeleteProcessInstancesFailFastOmitsUnscheduledCompletionFacts verifies
// fail-fast cancellation does not invent completion facts for worker items that
// never reached the API call.
func TestDeleteProcessInstancesFailFastOmitsUnscheduledCompletionFacts(t *testing.T) {
	events := &lockedProgressEvents{}
	wantErr := errors.New("delete failed")
	api := stubBulkProcessInstanceAPI{
		delete: func(_ context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			require.Equal(t, "root-1", key)
			return d.DeleteResponse{}, wantErr
		},
	}

	got, err := DeleteProcessInstances(context.Background(), api, slog.Default(), typex.Keys{"root-1", "root-2", "root-3"}, 1, 3,
		services.WithFailFast(),
		services.WithProgress(events.Append),
	)

	require.ErrorIs(t, err, wantErr)
	require.Len(t, got, 3)
	completions := events.Completions()
	require.Len(t, completions, 1)
	require.Equal(t, d.OpsCompletionProgress{
		Phase:            "delete",
		CoreResource:     "process-instance tree(s)",
		Total:            3,
		Identity:         "root-1",
		Disposition:      d.OpsCompletionDispositionFailed,
		FailureDetail:    "delete failed",
		AffectedResource: "affected process instances",
	}, completions[0])
}

func intPtr(v int) *int {
	return &v
}

// TestDeleteProcessInstancesLogsProgressWhileRootDeleteRuns verifies long root-tree deletes produce durable progress lines before the final summary.
func TestDeleteProcessInstancesLogsProgressWhileRootDeleteRuns(t *testing.T) {
	oldInterval := processInstanceBulkProgressInterval
	processInstanceBulkProgressInterval = 10 * time.Millisecond
	t.Cleanup(func() { processInstanceBulkProgressInterval = oldInterval })

	var logBuf lockedLogBuffer
	log := slog.New(logging.NewPlainHandler(&logBuf, slog.LevelInfo))
	sink := &activitysink.Sink{}
	started := make(chan struct{})
	release := make(chan struct{})
	var startedOnce sync.Once
	api := stubBulkProcessInstanceAPI{
		delete: func(ctx context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			require.Equal(t, "root-1", key)
			startedOnce.Do(func() { close(started) })
			select {
			case <-ctx.Done():
				return d.DeleteResponse{}, ctx.Err()
			case <-release:
				return d.DeleteResponse{Ok: true, StatusCode: 204, Status: "204 No Content"}, nil
			}
		},
	}
	errCh := make(chan error, 1)
	go func() {
		_, err := DeleteProcessInstances(logging.ToActivityContext(context.Background(), sink), api, log, typex.Keys{"root-1"}, 1, 4)
		errCh <- err
	}()

	<-started
	require.Eventually(t, func() bool {
		return strings.Contains(logBuf.String(), "pi delete progress; roots 0/1 done, affected 4")
	}, time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		return strings.Contains(strings.Join(sink.Updates(), "\n"), "pi delete progress; roots 0/1 done, affected 4")
	}, time.Second, 10*time.Millisecond)
	require.Contains(t, sink.Starts(), activitysink.Start{
		Message:    "deleting 4 pi via 1 root(s)",
		Importance: logging.ActivityImportanceBatch,
	})
	require.Eventually(t, func() bool {
		return containsActivityUpdate(sink.PriorityUpdates(), activitysink.Update{
			Message:    "pi delete progress; roots 0/1 done, affected 4",
			Importance: logging.ActivityImportanceBatch,
		})
	}, time.Second, 10*time.Millisecond)
	close(release)

	require.NoError(t, <-errCh)
	require.Contains(t, logBuf.String(), "pi delete done; roots 1, affected 4, ok 1, failed 0")
}

// containsActivityUpdate matches asynchronous activity updates without depending on exact retry count.
func containsActivityUpdate(items []activitysink.Update, want activitysink.Update) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// TestDeleteProcessInstancesSuppressesLegacyBulkLogs verifies CLI callers can
// rely on structured progress without duplicate durable INFO progress lines.
func TestDeleteProcessInstancesSuppressesLegacyBulkLogs(t *testing.T) {
	oldInterval := processInstanceBulkProgressInterval
	processInstanceBulkProgressInterval = 10 * time.Millisecond
	t.Cleanup(func() { processInstanceBulkProgressInterval = oldInterval })

	var logBuf lockedLogBuffer
	log := slog.New(logging.NewPlainHandler(&logBuf, slog.LevelInfo))
	started := make(chan struct{})
	release := make(chan struct{})
	var startedOnce sync.Once
	api := stubBulkProcessInstanceAPI{
		delete: func(ctx context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error) {
			require.Equal(t, "root-1", key)
			require.True(t, services.ApplyCallOptions(opts).SuppressProcessInstanceDetailLogs)
			startedOnce.Do(func() { close(started) })
			select {
			case <-ctx.Done():
				return d.DeleteResponse{}, ctx.Err()
			case <-release:
				return d.DeleteResponse{Ok: true, StatusCode: 204, Status: "204 No Content"}, nil
			}
		},
	}
	errCh := make(chan error, 1)
	go func() {
		_, err := DeleteProcessInstances(context.Background(), api, log, typex.Keys{"root-1"}, 1, 4,
			services.WithSuppressWorkflowDetailLogs(),
			services.WithSuppressProcessInstanceDetailLogs(),
		)
		errCh <- err
	}()

	<-started
	time.Sleep(35 * time.Millisecond)
	require.NotContains(t, logBuf.String(), "pi delete progress")
	close(release)

	require.NoError(t, <-errCh)
	require.NotContains(t, logBuf.String(), "pi delete done")
}

// TestDeleteProcessInstancesProgressCallbackSuppressesLegacyTimer verifies
// structured progress callbacks replace the old timer stream without hiding the
// ordinary service summary for non-suppressed callers.
func TestDeleteProcessInstancesProgressCallbackSuppressesLegacyTimer(t *testing.T) {
	oldInterval := processInstanceBulkProgressInterval
	processInstanceBulkProgressInterval = 10 * time.Millisecond
	t.Cleanup(func() { processInstanceBulkProgressInterval = oldInterval })

	var logBuf lockedLogBuffer
	log := slog.New(logging.NewPlainHandler(&logBuf, slog.LevelInfo))
	events := &lockedProgressEvents{}
	started := make(chan struct{})
	release := make(chan struct{})
	var startedOnce sync.Once
	api := stubBulkProcessInstanceAPI{
		delete: func(ctx context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			require.Equal(t, "root-1", key)
			startedOnce.Do(func() { close(started) })
			select {
			case <-ctx.Done():
				return d.DeleteResponse{}, ctx.Err()
			case <-release:
				return d.DeleteResponse{Ok: true, StatusCode: 204, Status: "204 No Content"}, nil
			}
		},
	}
	errCh := make(chan error, 1)
	go func() {
		_, err := DeleteProcessInstances(context.Background(), api, log, typex.Keys{"root-1"}, 1, 4,
			services.WithProgress(events.Append),
		)
		errCh <- err
	}()

	<-started
	time.Sleep(35 * time.Millisecond)
	require.NotContains(t, logBuf.String(), "pi delete progress")
	close(release)

	require.NoError(t, <-errCh)
	require.NotContains(t, logBuf.String(), "pi delete progress")
	require.Contains(t, logBuf.String(), "pi delete done; roots 1, affected 4, ok 1, failed 0")
	require.Len(t, events.Completions(), 1)
}

// TestCancelProcessInstancesLogsSlowRootWhenProgressStalls verifies progress output names the in-flight root when completion stops advancing.
func TestCancelProcessInstancesLogsSlowRootWhenProgressStalls(t *testing.T) {
	oldInterval := processInstanceBulkProgressInterval
	oldThreshold := processInstanceBulkStallProgressThreshold
	processInstanceBulkProgressInterval = 10 * time.Millisecond
	processInstanceBulkStallProgressThreshold = 1
	t.Cleanup(func() {
		processInstanceBulkProgressInterval = oldInterval
		processInstanceBulkStallProgressThreshold = oldThreshold
	})

	var logBuf lockedLogBuffer
	log := slog.New(logging.NewPlainHandler(&logBuf, slog.LevelInfo))
	started := make(chan struct{})
	release := make(chan struct{})
	var startedOnce sync.Once
	api := stubBulkProcessInstanceAPI{
		cancel: func(ctx context.Context, key string, _ ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
			require.Equal(t, "root-slow", key)
			startedOnce.Do(func() { close(started) })
			select {
			case <-ctx.Done():
				return d.CancelResponse{}, nil, ctx.Err()
			case <-release:
				return d.CancelResponse{Ok: true, StatusCode: 200, Status: "200 OK"}, nil, nil
			}
		},
	}
	errCh := make(chan error, 1)
	go func() {
		_, err := CancelProcessInstances(context.Background(), api, log, typex.Keys{"root-slow"}, 1, 3)
		errCh <- err
	}()

	<-started
	require.Eventually(t, func() bool {
		out := logBuf.String()
		return strings.Contains(out, "pi cancel slow root; root root-slow") &&
			strings.Contains(out, "phase cancel request or wait")
	}, time.Second, 10*time.Millisecond)
	close(release)

	require.NoError(t, <-errCh)
}
