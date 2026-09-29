// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

type updateExecutionScopeWriter struct {
	mu      sync.Mutex
	calls   []string
	write   func(context.Context, string, map[string]any, ...services.CallOption) (d.ScopeVariableUpdateResponse, error)
	current atomic.Int32
	maximum atomic.Int32
}

// UpdateScopeVariables records execution calls and delegates controlled outcomes.
func (w *updateExecutionScopeWriter) UpdateScopeVariables(ctx context.Context, scope string, variables map[string]any, opts ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
	w.mu.Lock()
	w.calls = append(w.calls, scope)
	w.mu.Unlock()
	current := w.current.Add(1)
	defer w.current.Add(-1)
	for current > w.maximum.Load() && !w.maximum.CompareAndSwap(w.maximum.Load(), current) {
	}
	if w.write != nil {
		return w.write(ctx, scope, variables, opts...)
	}
	return d.ScopeVariableUpdateResponse{ScopeKey: scope, Accepted: true, StatusCode: 204, Status: "accepted"}, nil
}

// callCount returns the synchronized number of physical writes.
func (w *updateExecutionScopeWriter) callCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.calls)
}

// TestExecuteUserTaskVariableUpdatesPropagatesSharedAndPartialOutcomes verifies
// one shared write fans out consistently while a task retains another accepted scope.
func TestExecuteUserTaskVariableUpdatesPropagatesSharedAndPartialOutcomes(t *testing.T) {
	t.Parallel()

	writer := &updateExecutionScopeWriter{write: func(_ context.Context, scope string, _ map[string]any, _ ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
		if scope == "local-a" {
			return d.ScopeVariableUpdateResponse{ScopeKey: scope, StatusCode: 500, Status: "failed"}, d.ErrUnavailable
		}
		return d.ScopeVariableUpdateResponse{ScopeKey: scope, Accepted: true, StatusCode: 204, Status: "accepted"}, nil
	}}
	service := NewVariableUpdates(nil, writer, config.New(), nil)

	results, err := service.ExecuteUserTaskVariableUpdates(context.Background(), sharedExecutionPlan(), 2, services.WithNoWait())

	require.ErrorIs(t, err, d.ErrUnavailable)
	require.Equal(t, 2, writer.callCount(), "shared target must be submitted once")
	require.Len(t, results.Items, 2)
	require.Equal(t, d.UserTaskVariableUpdateStatusMutationFailed, results.Items[0].Status)
	require.True(t, results.Items[0].MutationAccepted, "accepted is a historical partial-work fact")
	require.Equal(t, []d.ScopeVariableUpdateStatus{d.ScopeVariableUpdateStatusSubmitted, d.ScopeVariableUpdateStatusMutationFailed}, scopeOutcomeStatuses(results.Items[0].Scopes))
	require.Equal(t, d.UserTaskVariableUpdateStatusSubmitted, results.Items[1].Status)
	require.True(t, results.Items[1].MutationAccepted)
	require.Equal(t, []string{"shared"}, results.Items[1].Scopes[0].Names)
}

// TestExecuteUserTaskVariableUpdatesFansSharedFailureToEveryTask verifies one
// failed physical target produces the same failure fact for all dependents.
func TestExecuteUserTaskVariableUpdatesFansSharedFailureToEveryTask(t *testing.T) {
	t.Parallel()

	writer := &updateExecutionScopeWriter{write: func(_ context.Context, scope string, _ map[string]any, _ ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
		if scope == "shared-scope" {
			return d.ScopeVariableUpdateResponse{ScopeKey: scope}, d.ErrUnavailable
		}
		return d.ScopeVariableUpdateResponse{ScopeKey: scope, Accepted: true}, nil
	}}
	results, err := NewVariableUpdates(nil, writer, config.New(), nil).
		ExecuteUserTaskVariableUpdates(context.Background(), sharedExecutionPlan(), 1, services.WithNoWait())

	require.ErrorIs(t, err, d.ErrUnavailable)
	for _, result := range results.Items {
		require.Equal(t, d.UserTaskVariableUpdateStatusMutationFailed, result.Status)
		require.Equal(t, d.ScopeVariableUpdateStatusMutationFailed, result.Scopes[0].Status)
		require.Contains(t, result.Error, d.ErrUnavailable.Error())
	}
	require.True(t, results.Items[0].MutationAccepted, "task-a retains its accepted local write")
	require.False(t, results.Items[1].MutationAccepted)
}

// TestExecuteUserTaskVariableUpdatesKeepsMixedUnchangedTaskFacts verifies a
// no-change task remains unchanged while another selected task is submitted.
func TestExecuteUserTaskVariableUpdatesKeepsMixedUnchangedTaskFacts(t *testing.T) {
	t.Parallel()

	plan := singleChangeExecutionPlan("task-a", "scope-a", "value", "new")
	plan.RequestedKeys = append(plan.RequestedKeys, "task-b")
	plan.RequestedCount++
	plan.VariableUnchangedCount = 1
	plan.UserTasks = append(plan.UserTasks, d.UserTaskVariablePlan{
		UserTaskKey: "task-b", ElementInstanceKey: "scope-b",
		UnchangedRequested: []d.UserTaskVariablePlannedValue{{Name: "same", ScopeKey: "scope-b", Value: true}},
	})
	results, err := NewVariableUpdates(nil, &updateExecutionScopeWriter{}, config.New(), nil).
		ExecuteUserTaskVariableUpdates(context.Background(), plan, 1, services.WithNoWait())

	require.NoError(t, err)
	require.Equal(t, d.UserTaskVariableUpdateStatusSubmitted, results.Items[0].Status)
	require.Equal(t, d.UserTaskVariableUpdateStatusUnchanged, results.Items[1].Status)
	require.False(t, results.Items[1].MutationAccepted)
	require.Equal(t, "skipped", results.Items[1].ConfirmationStatus)
}

// TestExecuteUserTaskVariableUpdatesHonorsDryRunNoOpAndValidation verifies all
// pre-submission exits issue zero writes and invalid plans fail before transport.
func TestExecuteUserTaskVariableUpdatesHonorsDryRunNoOpAndValidation(t *testing.T) {
	t.Parallel()

	writer := &updateExecutionScopeWriter{}
	service := NewVariableUpdates(nil, writer, config.New(), nil)

	dryRun, err := service.ExecuteUserTaskVariableUpdates(context.Background(), sharedExecutionPlan(), 1, services.WithDryRun())
	require.NoError(t, err)
	require.Empty(t, dryRun.Items)

	noOp := noOpExecutionPlan()
	noOpResult, err := service.ExecuteUserTaskVariableUpdates(context.Background(), noOp, 1)
	require.NoError(t, err)
	require.Empty(t, noOpResult.Items)

	invalid := sharedExecutionPlan()
	invalid.Targets[0].Variables["shared"] = "tampered"
	_, err = service.ExecuteUserTaskVariableUpdates(context.Background(), invalid, 1)
	require.ErrorIs(t, err, d.ErrValidation)
	require.Zero(t, writer.callCount())
}

// TestExecuteUserTaskVariableUpdatesBoundsWorkersAndMarksFailFastSkips verifies
// configured concurrency and deterministic unscheduled target facts.
func TestExecuteUserTaskVariableUpdatesBoundsWorkersAndMarksFailFastSkips(t *testing.T) {
	t.Parallel()

	t.Run("worker bound", func(t *testing.T) {
		writer := &updateExecutionScopeWriter{write: func(ctx context.Context, scope string, _ map[string]any, _ ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
			select {
			case <-time.After(10 * time.Millisecond):
				return d.ScopeVariableUpdateResponse{ScopeKey: scope, Accepted: true}, nil
			case <-ctx.Done():
				return d.ScopeVariableUpdateResponse{}, ctx.Err()
			}
		}}
		plan := independentExecutionPlan(6)
		_, err := NewVariableUpdates(nil, writer, config.New(), nil).ExecuteUserTaskVariableUpdates(context.Background(), plan, 2, services.WithNoWait())

		require.NoError(t, err)
		require.LessOrEqual(t, writer.maximum.Load(), int32(2))
		require.Equal(t, 6, writer.callCount())
	})

	t.Run("fail fast", func(t *testing.T) {
		writer := &updateExecutionScopeWriter{write: func(_ context.Context, scope string, _ map[string]any, _ ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
			if scope == "scope-0" {
				return d.ScopeVariableUpdateResponse{ScopeKey: scope}, d.ErrForbidden
			}
			return d.ScopeVariableUpdateResponse{ScopeKey: scope, Accepted: true}, nil
		}}
		plan := independentExecutionPlan(3)
		results, err := NewVariableUpdates(nil, writer, config.New(), nil).ExecuteUserTaskVariableUpdates(context.Background(), plan, 1, services.WithNoWait(), services.WithFailFast())

		require.ErrorIs(t, err, d.ErrForbidden)
		require.Equal(t, 1, writer.callCount())
		require.Equal(t, d.UserTaskVariableUpdateStatusMutationFailed, results.Items[0].Status)
		require.Equal(t, d.UserTaskVariableUpdateStatusSkipped, results.Items[1].Status)
		require.Equal(t, d.ScopeVariableUpdateStatusSkipped, results.Items[2].Scopes[0].Status)
	})
}

// TestExecuteUserTaskVariableUpdatesPreservesCancellation verifies caller
// cancellation returns partial facts and never promotes unfinished work.
func TestExecuteUserTaskVariableUpdatesPreservesCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	writer := &updateExecutionScopeWriter{write: func(ctx context.Context, scope string, _ map[string]any, _ ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
		cancel()
		<-ctx.Done()
		return d.ScopeVariableUpdateResponse{ScopeKey: scope}, ctx.Err()
	}}
	results, err := NewVariableUpdates(nil, writer, config.New(), nil).ExecuteUserTaskVariableUpdates(ctx, independentExecutionPlan(3), 1, services.WithNoWait())

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, d.UserTaskVariableUpdateStatusMutationFailed, results.Items[0].Status)
	for _, result := range results.Items[1:] {
		require.Equal(t, d.UserTaskVariableUpdateStatusSkipped, result.Status)
	}
}

// sharedExecutionPlan builds two tasks sharing one target while the first has a second scope.
func sharedExecutionPlan() d.UserTaskVariableUpdatePlan {
	return d.UserTaskVariableUpdatePlan{
		RequestedKeys: []string{"task-a", "task-b"}, RequestedCount: 2, UpdateCount: 2, VariableChangeCount: 3,
		UserTasks: []d.UserTaskVariablePlan{
			{UserTaskKey: "task-a", ElementInstanceKey: "local-a", Changes: []d.UserTaskVariablePlannedChange{
				{Name: "shared", ScopeKey: "shared-scope", Inherited: true, Before: "old", After: "new"},
				{Name: "local", ScopeKey: "local-a", Before: false, After: true},
			}, TargetScopeKeys: []string{"shared-scope", "local-a"}},
			{UserTaskKey: "task-b", ElementInstanceKey: "local-b", Changes: []d.UserTaskVariablePlannedChange{
				{Name: "shared", ScopeKey: "shared-scope", Inherited: true, Before: "old", After: "new"},
			}, TargetScopeKeys: []string{"shared-scope"}},
		},
		Targets: []d.ScopeVariableUpdateTarget{
			{ScopeKey: "shared-scope", Variables: map[string]any{"shared": "new"}, Associations: []d.ScopeVariableUpdateAssociation{
				{UserTaskKey: "task-a", Names: []string{"shared"}}, {UserTaskKey: "task-b", Names: []string{"shared"}},
			}},
			{ScopeKey: "local-a", Variables: map[string]any{"local": true}, Associations: []d.ScopeVariableUpdateAssociation{{UserTaskKey: "task-a", Names: []string{"local"}}}},
		},
	}
}

// noOpExecutionPlan builds a validated unchanged-only plan.
func noOpExecutionPlan() d.UserTaskVariableUpdatePlan {
	return d.UserTaskVariableUpdatePlan{
		RequestedKeys: []string{"task-a"}, RequestedCount: 1, VariableUnchangedCount: 1,
		UserTasks: []d.UserTaskVariablePlan{{
			UserTaskKey: "task-a", ElementInstanceKey: "local-a",
			UnchangedRequested: []d.UserTaskVariablePlannedValue{{Name: "same", ScopeKey: "local-a", Value: true}},
		}}, Targets: []d.ScopeVariableUpdateTarget{},
	}
}

// independentExecutionPlan builds one unique local target per task.
func independentExecutionPlan(count int) d.UserTaskVariableUpdatePlan {
	plan := d.UserTaskVariableUpdatePlan{RequestedCount: count, UpdateCount: count, VariableAddCount: count}
	for i := 0; i < count; i++ {
		key := "task-" + string(rune('a'+i))
		scope := "scope-" + string(rune('0'+i))
		plan.RequestedKeys = append(plan.RequestedKeys, key)
		plan.UserTasks = append(plan.UserTasks, d.UserTaskVariablePlan{
			UserTaskKey: key, ElementInstanceKey: scope,
			Additions: []d.UserTaskVariablePlannedValue{{Name: "value", ScopeKey: scope, Value: i}}, TargetScopeKeys: []string{scope},
		})
		plan.Targets = append(plan.Targets, d.ScopeVariableUpdateTarget{
			ScopeKey: scope, Variables: map[string]any{"value": i},
			Associations: []d.ScopeVariableUpdateAssociation{{UserTaskKey: key, Names: []string{"value"}}},
		})
	}
	return plan
}

// scopeOutcomeStatuses extracts task-local scope states for compact assertions.
func scopeOutcomeStatuses(outcomes []d.ScopeVariableUpdateOutcome) []d.ScopeVariableUpdateStatus {
	statuses := make([]d.ScopeVariableUpdateStatus, len(outcomes))
	for i := range outcomes {
		statuses[i] = outcomes[i].Status
	}
	return statuses
}
