// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"errors"
	"testing"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

type updatePlanningUserTaskAPI struct {
	tasks     map[string]d.UserTask
	pages     map[string][]d.UserTaskVariablePage
	getErr    map[string]error
	pageErr   map[string]map[int]error
	pageCalls map[string]int
}

// GetUserTask rejects use of the tenant-scoped legacy resolver during explicit-key planning.
func (a *updatePlanningUserTaskAPI) GetUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	panic("update planning must use native task lookup")
}

// GetNativeUserTask returns the task fixture while recording explicit-key order.
func (a *updatePlanningUserTaskAPI) GetNativeUserTask(_ context.Context, key string, _ ...services.CallOption) (d.UserTask, error) {
	if err := a.getErr[key]; err != nil {
		return d.UserTask{}, err
	}
	return a.tasks[key], nil
}

// SearchUserTasksPage rejects accidental discovery-based selection.
func (a *updatePlanningUserTaskAPI) SearchUserTasksPage(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error) {
	panic("update planning must not search for tasks")
}

// SearchUserTaskEffectiveVariablesPage returns the next complete-traversal fixture page.
func (a *updatePlanningUserTaskAPI) SearchUserTaskEffectiveVariablesPage(_ context.Context, key string, request d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
	if a.pageCalls == nil {
		a.pageCalls = make(map[string]int)
	}
	call := a.pageCalls[key]
	a.pageCalls[key]++
	if err := a.pageErr[key][call]; err != nil {
		return d.UserTaskVariablePage{}, err
	}
	page := a.pages[key][call]
	page.Request = request
	return page, nil
}

type updatePlanningScopeWriter struct {
	calls int
}

// UpdateScopeVariables records forbidden writes from the planning-only workflow.
func (w *updatePlanningScopeWriter) UpdateScopeVariables(context.Context, string, map[string]any, ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
	w.calls++
	return d.ScopeVariableUpdateResponse{}, errors.New("planning must not write")
}

// TestPlanUserTaskVariableUpdatesBuildsCompleteScopeSafePlan verifies local,
// inherited, missing, unchanged, truncated, untouched, tenant, and sparse-page facts.
func TestPlanUserTaskVariableUpdatesBuildsCompleteScopeSafePlan(t *testing.T) {
	t.Parallel()

	api := &updatePlanningUserTaskAPI{
		tasks: map[string]d.UserTask{
			"task-a": {Key: "task-a", ElementInstanceKey: "element-a", ProcessInstanceKey: "root-a", TenantId: "tenant-a"},
		},
		pages: map[string][]d.UserTaskVariablePage{
			"task-a": {
				updatePlanningPage(nil, 0, 5, d.UserTaskReportedTotalKindExact, true),
				updatePlanningPage([]d.ProcessInstanceVariable{
					{Name: "rootChange", Value: `"before"`, VariableKey: "v-root", ProcessInstanceKey: "root-a", ScopeKey: "root-a", TenantId: "tenant-a"},
					{Name: "middleEqual", Value: `{"count":1}`, VariableKey: "v-middle", ProcessInstanceKey: "root-a", ScopeKey: "middle-a", TenantId: "tenant-a"},
					{Name: "localChange", Value: `false`, VariableKey: "v-local", ProcessInstanceKey: "root-a", ScopeKey: "element-a", TenantId: "tenant-a"},
					{Name: "truncated", Value: `{"large":`, VariableKey: "v-truncated", ProcessInstanceKey: "root-a", ScopeKey: "middle-a", TenantId: "tenant-a", APITruncated: true},
					{Name: "untouched", Value: `null`, VariableKey: "v-untouched", ProcessInstanceKey: "root-a", ScopeKey: "root-a", TenantId: "tenant-a"},
				}, 5, 5, d.UserTaskReportedTotalKindExact, false),
			},
		},
	}
	writer := &updatePlanningScopeWriter{}
	cfg := config.New()
	cfg.App.Tenant = "configured-but-not-filtering"
	service := NewVariableUpdates(api, writer, cfg, nil)

	plan, err := service.PlanUserTaskVariableUpdates(context.Background(), typex.Keys{"task-a", "task-a"}, map[string]any{
		"newNull": nil, "rootChange": "after", "middleEqual": map[string]any{"count": float64(1)},
		"localChange": true, "truncated": map[string]any{"large": true},
	})

	require.NoError(t, err)
	require.Equal(t, []string{"task-a"}, plan.RequestedKeys)
	require.Equal(t, 1, plan.RequestedCount)
	require.Equal(t, 1, plan.UpdateCount)
	require.Equal(t, 1, plan.VariableAddCount)
	require.Equal(t, 3, plan.VariableChangeCount)
	require.Equal(t, 1, plan.VariableUnchangedCount)
	require.Equal(t, 1, plan.VariableUntouchedCount)
	require.False(t, plan.MutationSubmitted)
	require.Equal(t, d.TenantContextModeExplicitKeys, plan.TenantContext.Mode)
	require.Equal(t, "configured-but-not-filtering", plan.TenantContext.ConfiguredTenantID)
	require.Equal(t, []string{"tenant-a"}, plan.TenantContext.ResolvedTenantIDs)
	require.Zero(t, plan.TenantContext.UnknownTargetCount)

	require.Len(t, plan.UserTasks, 1)
	task := plan.UserTasks[0]
	require.Equal(t, []string{"newNull"}, plannedValueNames(task.Additions))
	require.Equal(t, []string{"localChange", "rootChange", "truncated"}, plannedChangeNames(task.Changes))
	require.Equal(t, []string{"middleEqual"}, plannedValueNames(task.UnchangedRequested))
	require.Equal(t, []string{"untouched"}, plannedValueNames(task.Untouched))
	require.False(t, task.Additions[0].Inherited)
	require.True(t, task.Changes[1].Inherited)
	require.True(t, task.Changes[2].APITruncated)
	require.Equal(t, `{"large":`, task.Changes[2].Before)
	require.Equal(t, []string{"element-a", "root-a", "middle-a"}, task.TargetScopeKeys)

	require.Len(t, plan.Targets, 3)
	require.Equal(t, "element-a", plan.Targets[0].ScopeKey)
	require.Equal(t, map[string]any{"localChange": true, "newNull": nil}, plan.Targets[0].Variables)
	require.Equal(t, "root-a", plan.Targets[1].ScopeKey)
	require.Equal(t, map[string]any{"rootChange": "after"}, plan.Targets[1].Variables)
	require.Equal(t, "middle-a", plan.Targets[2].ScopeKey)
	require.Equal(t, map[string]any{"truncated": map[string]any{"large": true}}, plan.Targets[2].Variables)
	require.Equal(t, 2, api.pageCalls["task-a"])
	require.Zero(t, writer.calls)
}

// TestPlanUserTaskVariableUpdatesDeduplicatesSharedTargets verifies one scope
// payload serves several tasks while same-name variables at other scopes remain distinct.
func TestPlanUserTaskVariableUpdatesDeduplicatesSharedTargets(t *testing.T) {
	t.Parallel()

	api := &updatePlanningUserTaskAPI{
		tasks: map[string]d.UserTask{
			"task-b": {Key: "task-b", ElementInstanceKey: "element-b", ProcessInstanceKey: "root-shared", TenantId: "tenant-a"},
			"task-a": {Key: "task-a", ElementInstanceKey: "element-a", ProcessInstanceKey: "root-shared", TenantId: "tenant-a"},
		},
		pages: map[string][]d.UserTaskVariablePage{
			"task-b": {updatePlanningPage([]d.ProcessInstanceVariable{
				{Name: "shared", Value: `"old"`, VariableKey: "v-shared", ProcessInstanceKey: "root-shared", ScopeKey: "root-shared", TenantId: "tenant-a"},
				{Name: "separate", Value: `0`, VariableKey: "v-b", ProcessInstanceKey: "root-shared", ScopeKey: "element-b", TenantId: "tenant-a"},
			}, 2, 2, d.UserTaskReportedTotalKindExact, false)},
			"task-a": {updatePlanningPage([]d.ProcessInstanceVariable{
				{Name: "shared", Value: `"old"`, VariableKey: "v-shared", ProcessInstanceKey: "root-shared", ScopeKey: "root-shared", TenantId: "tenant-a"},
				{Name: "separate", Value: `0`, VariableKey: "v-a", ProcessInstanceKey: "root-shared", ScopeKey: "element-a", TenantId: "tenant-a"},
			}, 2, 2, d.UserTaskReportedTotalKindExact, false)},
		},
	}
	service := NewVariableUpdates(api, &updatePlanningScopeWriter{}, config.New(), nil)

	plan, err := service.PlanUserTaskVariableUpdates(context.Background(), typex.Keys{"task-b", "task-a", "task-b"}, map[string]any{"shared": "new", "separate": float64(1)})

	require.NoError(t, err)
	require.Equal(t, []string{"task-b", "task-a"}, plan.RequestedKeys)
	require.Equal(t, []string{"task-b", "task-a"}, []string{plan.UserTasks[0].UserTaskKey, plan.UserTasks[1].UserTaskKey})
	require.Len(t, plan.Targets, 3)
	require.Equal(t, "element-b", plan.Targets[0].ScopeKey)
	require.Equal(t, "root-shared", plan.Targets[1].ScopeKey)
	require.Equal(t, map[string]any{"shared": "new"}, plan.Targets[1].Variables)
	require.Equal(t, []d.ScopeVariableUpdateAssociation{
		{UserTaskKey: "task-b", Names: []string{"shared"}},
		{UserTaskKey: "task-a", Names: []string{"shared"}},
	}, plan.Targets[1].Associations)
	require.Equal(t, "element-a", plan.Targets[2].ScopeKey)
	require.Equal(t, 4, plan.VariableChangeCount)
	require.Equal(t, 2, plan.UpdateCount)
}

// TestPlanUserTaskVariableUpdatesHandlesEmptyInputs verifies service-level no
// work and empty payloads return initialized plans without synthetic targets or writes.
func TestPlanUserTaskVariableUpdatesHandlesEmptyInputs(t *testing.T) {
	t.Parallel()

	t.Run("no keys", func(t *testing.T) {
		writer := &updatePlanningScopeWriter{}
		plan, err := NewVariableUpdates(nil, writer, config.New(), nil).PlanUserTaskVariableUpdates(context.Background(), nil, map[string]any{"ignored": true})

		require.NoError(t, err)
		require.NotNil(t, plan.RequestedKeys)
		require.NotNil(t, plan.UserTasks)
		require.NotNil(t, plan.Targets)
		require.Zero(t, writer.calls)
	})

	t.Run("empty payload still completes discovery", func(t *testing.T) {
		api := &updatePlanningUserTaskAPI{
			tasks: map[string]d.UserTask{"task-a": {Key: "task-a", ElementInstanceKey: "element-a"}},
			pages: map[string][]d.UserTaskVariablePage{"task-a": {updatePlanningPage(nil, 0, 0, d.UserTaskReportedTotalKindExact, false)}},
		}
		writer := &updatePlanningScopeWriter{}
		plan, err := NewVariableUpdates(api, writer, config.New(), nil).PlanUserTaskVariableUpdates(context.Background(), typex.Keys{"task-a"}, map[string]any{})

		require.NoError(t, err)
		require.Len(t, plan.UserTasks, 1)
		require.Empty(t, plan.Targets)
		require.Zero(t, plan.UpdateCount)
		require.Equal(t, 1, plan.TenantContext.UnknownTargetCount)
		require.Zero(t, writer.calls)
	})
}

// TestPlanUserTaskVariableUpdatesRejectsIncompleteOrConflictingDiscovery proves
// no executable partial plan or write escapes malformed identity and later-page failures.
func TestPlanUserTaskVariableUpdatesRejectsIncompleteOrConflictingDiscovery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		api  *updatePlanningUserTaskAPI
		err  error
	}{
		{
			name: "missing task scope",
			api: &updatePlanningUserTaskAPI{
				tasks: map[string]d.UserTask{"task-a": {Key: "task-a"}},
				pages: map[string][]d.UserTaskVariablePage{},
			},
			err: d.ErrValidation,
		},
		{
			name: "conflicting shared identity",
			api: &updatePlanningUserTaskAPI{
				tasks: map[string]d.UserTask{
					"task-a": {Key: "task-a", ElementInstanceKey: "element-a", ProcessInstanceKey: "root", TenantId: "tenant-a"},
					"task-b": {Key: "task-b", ElementInstanceKey: "element-b", ProcessInstanceKey: "root", TenantId: "tenant-a"},
				},
				pages: map[string][]d.UserTaskVariablePage{
					"task-a": {updatePlanningPage([]d.ProcessInstanceVariable{{Name: "shared", Value: `1`, VariableKey: "v-1", ProcessInstanceKey: "root", ScopeKey: "root", TenantId: "tenant-a"}}, 1, 1, d.UserTaskReportedTotalKindExact, false)},
					"task-b": {updatePlanningPage([]d.ProcessInstanceVariable{{Name: "shared", Value: `1`, VariableKey: "v-2", ProcessInstanceKey: "root", ScopeKey: "root", TenantId: "tenant-a"}}, 1, 1, d.UserTaskReportedTotalKindExact, false)},
				},
			},
			err: d.ErrValidation,
		},
		{
			name: "missing variable scope",
			api: &updatePlanningUserTaskAPI{
				tasks: map[string]d.UserTask{"task-a": {Key: "task-a", ElementInstanceKey: "element-a"}},
				pages: map[string][]d.UserTaskVariablePage{"task-a": {updatePlanningPage([]d.ProcessInstanceVariable{{Name: "value", Value: `1`}}, 1, 1, d.UserTaskReportedTotalKindExact, false)}},
			},
			err: d.ErrValidation,
		},
		{
			name: "conflicting tenant evidence",
			api: &updatePlanningUserTaskAPI{
				tasks: map[string]d.UserTask{"task-a": {Key: "task-a", ElementInstanceKey: "element-a", ProcessInstanceKey: "root", TenantId: "tenant-a"}},
				pages: map[string][]d.UserTaskVariablePage{"task-a": {updatePlanningPage([]d.ProcessInstanceVariable{{Name: "value", Value: `1`, ScopeKey: "root", TenantId: "tenant-b"}}, 1, 1, d.UserTaskReportedTotalKindExact, false)}},
			},
			err: d.ErrValidation,
		},
		{
			name: "later variable page fails",
			api: &updatePlanningUserTaskAPI{
				tasks:   map[string]d.UserTask{"task-a": {Key: "task-a", ElementInstanceKey: "element-a"}},
				pages:   map[string][]d.UserTaskVariablePage{"task-a": {updatePlanningPage(nil, 0, 1, d.UserTaskReportedTotalKindExact, true)}},
				pageErr: map[string]map[int]error{"task-a": {1: d.ErrUnavailable}},
			},
			err: d.ErrUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &updatePlanningScopeWriter{}
			plan, err := NewVariableUpdates(tt.api, writer, config.New(), nil).PlanUserTaskVariableUpdates(context.Background(), typex.Keys{"task-a", "task-b"}, map[string]any{"shared": 2, "value": 2})

			require.ErrorIs(t, err, tt.err)
			require.Equal(t, d.UserTaskVariableUpdatePlan{}, plan)
			require.Zero(t, writer.calls)
		})
	}
}

// TestExecuteUserTaskVariableUpdatesRejectsInconsistentPlanBeforeRequests
// verifies the execution boundary revalidates frozen target associations.
func TestExecuteUserTaskVariableUpdatesRejectsInconsistentPlanBeforeRequests(t *testing.T) {
	t.Parallel()

	writer := &updateExecutionScopeWriter{}
	plan := sharedExecutionPlan()
	plan.Targets[0].Associations[0].Names = []string{"local"}

	results, err := NewVariableUpdates(nil, writer, config.New(), nil).
		ExecuteUserTaskVariableUpdates(context.Background(), plan, 1, services.WithNoWait())

	require.ErrorIs(t, err, d.ErrValidation)
	require.Empty(t, results.Items)
	require.Zero(t, writer.callCount())
}

// updatePlanningPage builds traversal metadata while the fake fills the actual request.
func updatePlanningPage(items []d.ProcessInstanceVariable, raw int32, total int64, kind d.UserTaskReportedTotalKind, continuation bool) d.UserTaskVariablePage {
	return d.UserTaskVariablePage{
		Items: items, RawItemCount: raw, ReportedTotal: d.UserTaskReportedTotal{Count: total, Kind: kind},
		HasContinuationEvidence: continuation,
	}
}

// plannedValueNames extracts deterministic planned-value category names.
func plannedValueNames(values []d.UserTaskVariablePlannedValue) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		names = append(names, value.Name)
	}
	return names
}

// plannedChangeNames extracts deterministic change category names.
func plannedChangeNames(values []d.UserTaskVariablePlannedChange) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		names = append(names, value.Name)
	}
	return names
}
