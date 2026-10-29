// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	tasklistv88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/tasklist"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API defines the V88 user-task service contract.
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

// GenUserTaskClientCamunda captures the generated native read, search and mutation operations used by this adapter.
type GenUserTaskClientCamunda interface {
	// AssignUserTaskWithResponse submits the generated assignment request with explicit override policy.
	AssignUserTaskWithResponse(ctx context.Context, userTaskKey camundav88.UserTaskKey, body camundav88.AssignUserTaskJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.AssignUserTaskResponse, error)
	// UnassignUserTaskWithResponse removes the assignee using a bodyless generated request.
	UnassignUserTaskWithResponse(ctx context.Context, userTaskKey camundav88.UserTaskKey, reqEditors ...camundav88.RequestEditorFn) (*camundav88.UnassignUserTaskResponse, error)

	// CompleteUserTaskWithResponse submits the version-specific generated completion request.
	CompleteUserTaskWithResponse(ctx context.Context, userTaskKey camundav88.UserTaskKey, body camundav88.CompleteUserTaskJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CompleteUserTaskResponse, error)
	GetUserTaskWithResponse(ctx context.Context, userTaskKey camundav88.UserTaskKey, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetUserTaskResponse, error)
	SearchUserTasksWithResponse(ctx context.Context, body camundav88.SearchUserTasksJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error)
	SearchUserTaskEffectiveVariablesWithResponse(ctx context.Context, userTaskKey camundav88.UserTaskKey, params *camundav88.SearchUserTaskEffectiveVariablesParams, body camundav88.SearchUserTaskEffectiveVariablesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchUserTaskEffectiveVariablesResponse, error)
}

type GenUserTaskClientTasklist interface {
	GetTaskByIdWithResponse(ctx context.Context, taskId string, reqEditors ...tasklistv88.RequestEditorFn) (*tasklistv88.GetTaskByIdResponse, error)
}

var _ API = (*Service)(nil)
var _ GenUserTaskClientCamunda = (*camundav88.ClientWithResponses)(nil)
var _ GenUserTaskClientTasklist = (*tasklistv88.ClientWithResponses)(nil)
