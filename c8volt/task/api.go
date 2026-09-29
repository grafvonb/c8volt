// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	types "github.com/grafvonb/c8volt/typex"
)

type API interface {
	GetUserTask(ctx context.Context, taskKey string, opts ...options.FacadeOption) (UserTask, error)
	GetUserTasks(ctx context.Context, taskKeys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (UserTasks, error)
	SearchUserTasks(ctx context.Context, request SearchRequest, opts ...options.FacadeOption) (UserTasks, error)
	SearchUserTasksPages(ctx context.Context, request SearchRequest, visitor SearchPageVisitor, opts ...options.FacadeOption) (SearchPagesResult, error)
	SearchUserTasksTotal(ctx context.Context, request SearchRequest, opts ...options.FacadeOption) (int64, error)
	EnrichUserTasksWithVariables(ctx context.Context, tasks UserTasks, opts ...options.FacadeOption) (VariableEnrichedUserTasks, error)
	ResolveProcessInstanceKeyFromUserTask(ctx context.Context, taskKey string, opts ...options.FacadeOption) (string, error)
	ResolveProcessInstanceKeysFromUserTasks(ctx context.Context, taskKeys types.Keys, opts ...options.FacadeOption) (types.Keys, error)
	PlanUserTaskVariableUpdates(ctx context.Context, keys types.Keys, variables map[string]any, opts ...options.FacadeOption) (UserTaskVariableUpdatePlan, error)
	ExecuteUserTaskVariableUpdates(ctx context.Context, plan UserTaskVariableUpdatePlan, wantedWorkers int, opts ...options.FacadeOption) (UserTaskVariableUpdateResults, error)
}

var _ API = (*client)(nil)
