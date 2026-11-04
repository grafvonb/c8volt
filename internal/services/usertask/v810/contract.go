// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API defines the V810 user-task service contract.
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
	AssignUserTaskWithResponse(ctx context.Context, userTaskKey camundav810.UserTaskKey, body camundav810.AssignUserTaskJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.AssignUserTaskResponse, error)
	// UnassignUserTaskWithResponse removes the assignee using a bodyless generated request.
	UnassignUserTaskWithResponse(ctx context.Context, userTaskKey camundav810.UserTaskKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.UnassignUserTaskResponse, error)

	// CompleteUserTaskWithResponse submits the version-specific generated completion request.
	CompleteUserTaskWithResponse(ctx context.Context, userTaskKey camundav810.UserTaskKey, body camundav810.CompleteUserTaskJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CompleteUserTaskResponse, error)
	GetUserTaskWithResponse(ctx context.Context, userTaskKey camundav810.UserTaskKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetUserTaskResponse, error)
	SearchUserTasksWithResponse(ctx context.Context, body camundav810.SearchUserTasksJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchUserTasksResponse, error)
	SearchUserTaskEffectiveVariablesWithResponse(ctx context.Context, userTaskKey camundav810.UserTaskKey, params *camundav810.SearchUserTaskEffectiveVariablesParams, body camundav810.SearchUserTaskEffectiveVariablesJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchUserTaskEffectiveVariablesResponse, error)
}

var _ API = (*Service)(nil)
var _ GenUserTaskClientCamunda = (*camundav810.ClientWithResponses)(nil)
