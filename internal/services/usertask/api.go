// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v810 "github.com/grafvonb/c8volt/internal/services/usertask/v810"
	v88 "github.com/grafvonb/c8volt/internal/services/usertask/v88"
	v89 "github.com/grafvonb/c8volt/internal/services/usertask/v89"
)

// API exposes legacy resolver lookup, native reads and explicit-key task mutations.
type API interface {
	// AssignUserTask submits protected native assignment and retains response evidence without polling.
	AssignUserTask(ctx context.Context, key string, assignee string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error)
	// UnassignUserTask submits native removal and retains response evidence without polling.
	UnassignUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error)

	// CompleteUserTask submits one native completion and retains acceptance evidence without polling.
	CompleteUserTask(ctx context.Context, key string, variables map[string]any, opts ...services.CallOption) (d.UserTaskCompletionResult, error)
	GetUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error)
	GetNativeUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error)
	SearchUserTasksPage(ctx context.Context, query d.UserTaskSearchQuery, page d.UserTaskPageRequest, opts ...services.CallOption) (d.UserTaskSearchPage, error)
	SearchUserTaskEffectiveVariablesPage(ctx context.Context, key string, page d.UserTaskVariablePageRequest, opts ...services.CallOption) (d.UserTaskVariablePage, error)
}

var _ API = (*v88.Service)(nil)
var _ API = (*v89.Service)(nil)
var _ API = (*v810.Service)(nil)
var _ API = (v88.API)(nil)
var _ API = (v89.API)(nil)
var _ API = (v810.API)(nil)
