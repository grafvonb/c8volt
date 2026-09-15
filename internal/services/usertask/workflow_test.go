// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

type orderedUserTaskAPI struct {
	requested types.Keys
	tasks     map[string]d.UserTask
}

// GetUserTask records the legacy resolver's lookup order and returns the configured owning task.
func (a *orderedUserTaskAPI) GetUserTask(_ context.Context, key string, _ ...services.CallOption) (d.UserTask, error) {
	a.requested = append(a.requested, key)
	return a.tasks[key], nil
}

// GetNativeUserTask fails if the legacy resolver accidentally switches to the direct-read path.
func (a *orderedUserTaskAPI) GetNativeUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	panic("legacy resolver must not call GetNativeUserTask")
}

// SearchUserTasksPage fails if legacy ownership resolution switches to native discovery.
func (a *orderedUserTaskAPI) SearchUserTasksPage(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error) {
	panic("legacy resolver must not call SearchUserTasksPage")
}

// SearchUserTaskEffectiveVariablesPage fails if legacy ownership resolution starts variable enrichment.
func (a *orderedUserTaskAPI) SearchUserTaskEffectiveVariablesPage(context.Context, string, d.UserTaskVariablePageRequest, ...services.CallOption) (d.UserTaskVariablePage, error) {
	panic("legacy resolver must not call SearchUserTaskEffectiveVariablesPage")
}

// TestResolveProcessInstanceKeys_PreservesInputOrder pins the legacy resolver's one-for-one task and owning-process ordering.
func TestResolveProcessInstanceKeys_PreservesInputOrder(t *testing.T) {
	t.Parallel()

	api := &orderedUserTaskAPI{tasks: map[string]d.UserTask{
		"task-c": {Key: "task-c", ProcessInstanceKey: "process-3"},
		"task-a": {Key: "task-a", ProcessInstanceKey: "process-1"},
		"task-b": {Key: "task-b", ProcessInstanceKey: "process-2"},
	}}
	taskKeys := types.Keys{"task-c", "task-a", "task-b"}

	processInstanceKeys, err := ResolveProcessInstanceKeys(context.Background(), api, taskKeys)

	require.NoError(t, err)
	require.Equal(t, taskKeys, api.requested)
	require.Equal(t, types.Keys{"process-3", "process-1", "process-2"}, processInstanceKeys)
}
