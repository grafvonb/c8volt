// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

type updateConfirmationUserTaskAPI struct {
	mu       sync.Mutex
	searches map[string]int
	search   func(string, int) ([]d.ProcessInstanceVariable, error)
}

// GetUserTask rejects legacy lookup during update confirmation.
func (a *updateConfirmationUserTaskAPI) GetUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	panic("confirmation must not use legacy lookup")
}

// GetNativeUserTask rejects task rediscovery during update confirmation.
func (a *updateConfirmationUserTaskAPI) GetNativeUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	panic("confirmation must retain the frozen task")
}

// SearchUserTasksPage rejects search selection during update confirmation.
func (a *updateConfirmationUserTaskAPI) SearchUserTasksPage(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error) {
	panic("confirmation must not search tasks")
}

// SearchUserTaskEffectiveVariablesPage returns one exact complete page per poll.
func (a *updateConfirmationUserTaskAPI) SearchUserTaskEffectiveVariablesPage(_ context.Context, key string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
	a.mu.Lock()
	attempt := a.searches[key]
	a.searches[key]++
	a.mu.Unlock()
	items, err := a.search(key, attempt)
	if err != nil {
		return d.UserTaskVariablePage{}, err
	}
	page := updatePlanningPage(items, int32(len(items)), int64(len(items)), d.UserTaskReportedTotalKindExact, false)
	page.Request = request
	return page, nil
}

// count returns the synchronized confirmation read count for one task.
func (a *updateConfirmationUserTaskAPI) count(key string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.searches[key]
}

// TestExecuteUserTaskVariableUpdatesConfirmsExactScopeAndCompleteValue verifies
// polling rejects equal values at the wrong scope and truncated observations.
func TestExecuteUserTaskVariableUpdatesConfirmsExactScopeAndCompleteValue(t *testing.T) {
	t.Parallel()

	api := &updateConfirmationUserTaskAPI{searches: make(map[string]int)}
	api.search = func(key string, attempt int) ([]d.ProcessInstanceVariable, error) {
		require.Equal(t, "task-a", key)
		switch attempt {
		case 0:
			return []d.ProcessInstanceVariable{{Name: "value", ScopeKey: "wrong", Value: `"new"`}}, nil
		case 1:
			return []d.ProcessInstanceVariable{{Name: "value", ScopeKey: "scope-a", Value: `"new"`, APITruncated: true}}, nil
		default:
			return []d.ProcessInstanceVariable{{Name: "value", ScopeKey: "scope-a", Value: `"new"`}}, nil
		}
	}
	cfg := updateConfirmationConfig(5, time.Second)
	plan := singleChangeExecutionPlan("task-a", "scope-a", "value", "new")

	results, err := NewVariableUpdates(api, &updateExecutionScopeWriter{}, cfg, nil).ExecuteUserTaskVariableUpdates(context.Background(), plan, 1)

	require.NoError(t, err)
	require.Equal(t, 3, api.count("task-a"))
	require.Equal(t, d.UserTaskVariableUpdateStatusConfirmed, results.Items[0].Status)
	require.Equal(t, "confirmed", results.Items[0].ConfirmationStatus)
}

// TestExecuteUserTaskVariableUpdatesConfirmationFailuresRetainAcceptance verifies
// authorization loss and timeout remain distinct from accepted mutation facts.
func TestExecuteUserTaskVariableUpdatesConfirmationFailuresRetainAcceptance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		search func(string, int) ([]d.ProcessInstanceVariable, error)
		want   error
	}{
		{name: "authorization loss", search: func(string, int) ([]d.ProcessInstanceVariable, error) { return nil, d.ErrForbidden }, want: d.ErrForbidden},
		{name: "malformed complete value", search: func(string, int) ([]d.ProcessInstanceVariable, error) {
			return []d.ProcessInstanceVariable{{Name: "value", ScopeKey: "scope-a", Value: `{"broken":`}}, nil
		}, want: d.ErrMalformedResponse},
		{name: "timeout", search: func(string, int) ([]d.ProcessInstanceVariable, error) {
			return []d.ProcessInstanceVariable{{Name: "value", ScopeKey: "scope-a", Value: `"old"`}}, nil
		}, want: d.ErrUpstream},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &updateConfirmationUserTaskAPI{searches: make(map[string]int), search: tt.search}
			results, err := NewVariableUpdates(api, &updateExecutionScopeWriter{}, updateConfirmationConfig(1, time.Second), nil).
				ExecuteUserTaskVariableUpdates(context.Background(), singleChangeExecutionPlan("task-a", "scope-a", "value", "new"), 1)

			require.ErrorIs(t, err, tt.want)
			require.True(t, results.Items[0].MutationAccepted)
			require.Equal(t, d.UserTaskVariableUpdateStatusConfirmationFailed, results.Items[0].Status)
			require.Equal(t, "failed", results.Items[0].ConfirmationStatus)
		})
	}
}

// TestExecuteUserTaskVariableUpdatesNoWaitSkipsConfirmation verifies accepted
// writes make no task-variable read when waiting is disabled.
func TestExecuteUserTaskVariableUpdatesNoWaitSkipsConfirmation(t *testing.T) {
	t.Parallel()

	api := &updateConfirmationUserTaskAPI{searches: make(map[string]int), search: func(string, int) ([]d.ProcessInstanceVariable, error) {
		panic("no-wait must not confirm")
	}}
	results, err := NewVariableUpdates(api, &updateExecutionScopeWriter{}, updateConfirmationConfig(1, time.Second), nil).
		ExecuteUserTaskVariableUpdates(context.Background(), singleChangeExecutionPlan("task-a", "scope-a", "value", "new"), 1, services.WithNoWait())

	require.NoError(t, err)
	require.Zero(t, api.count("task-a"))
	require.Equal(t, d.UserTaskVariableUpdateStatusSubmitted, results.Items[0].Status)
	require.Equal(t, "skipped", results.Items[0].ConfirmationStatus)
}

// TestExecuteUserTaskVariableUpdatesFailFastConfirmationLeavesUnstartedSubmitted verifies
// a confirmation error does not rewrite another accepted task as failed.
func TestExecuteUserTaskVariableUpdatesFailFastConfirmationLeavesUnstartedSubmitted(t *testing.T) {
	t.Parallel()

	api := &updateConfirmationUserTaskAPI{searches: make(map[string]int), search: func(key string, _ int) ([]d.ProcessInstanceVariable, error) {
		if key == "task-a" {
			return nil, d.ErrNotFound
		}
		return []d.ProcessInstanceVariable{{Name: "value", ScopeKey: "scope-1", Value: `1`}}, nil
	}}
	results, err := NewVariableUpdates(api, &updateExecutionScopeWriter{}, updateConfirmationConfig(1, time.Second), nil).
		ExecuteUserTaskVariableUpdates(context.Background(), independentExecutionPlan(2), 1, services.WithFailFast())

	require.ErrorIs(t, err, d.ErrNotFound)
	require.Equal(t, d.UserTaskVariableUpdateStatusConfirmationFailed, results.Items[0].Status)
	require.Equal(t, d.UserTaskVariableUpdateStatusSubmitted, results.Items[1].Status)
	require.Equal(t, "skipped", results.Items[1].ConfirmationStatus)
	require.Zero(t, api.count("task-b"))
}

// updateConfirmationConfig provides fast deterministic polling bounds.
func updateConfirmationConfig(maxRetries int, timeout time.Duration) *config.Config {
	cfg := config.New()
	cfg.App.Backoff = config.BackoffConfig{
		Strategy: config.BackoffFixed, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond,
		MaxRetries: maxRetries, Timeout: timeout,
	}
	return cfg
}

// singleChangeExecutionPlan builds one valid inherited or local change plan.
func singleChangeExecutionPlan(taskKey, scopeKey, name string, value any) d.UserTaskVariableUpdatePlan {
	return d.UserTaskVariableUpdatePlan{
		RequestedKeys: []string{taskKey}, RequestedCount: 1, UpdateCount: 1, VariableChangeCount: 1,
		UserTasks: []d.UserTaskVariablePlan{{
			UserTaskKey: taskKey, ElementInstanceKey: scopeKey,
			Changes:         []d.UserTaskVariablePlannedChange{{Name: name, ScopeKey: scopeKey, Before: "old", After: value}},
			TargetScopeKeys: []string{scopeKey},
		}},
		Targets: []d.ScopeVariableUpdateTarget{{
			ScopeKey: scopeKey, Variables: map[string]any{name: value},
			Associations: []d.ScopeVariableUpdateAssociation{{UserTaskKey: taskKey, Names: []string{name}}},
		}},
	}
}
