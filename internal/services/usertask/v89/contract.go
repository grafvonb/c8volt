// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	tasklistv89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/tasklist"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API defines the V89 user-task service contract.
type API interface {
	GetUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error)
	GetNativeUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error)
	SearchUserTasksPage(ctx context.Context, query d.UserTaskSearchQuery, page d.UserTaskPageRequest, opts ...services.CallOption) (d.UserTaskSearchPage, error)
	SearchUserTaskEffectiveVariablesPage(ctx context.Context, key string, page d.UserTaskVariablePageRequest, opts ...services.CallOption) (d.UserTaskVariablePage, error)
}

// GenUserTaskClientCamunda captures the generated native read and search operations used by this adapter.
type GenUserTaskClientCamunda interface {
	GetUserTaskWithResponse(ctx context.Context, userTaskKey camundav89.UserTaskKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetUserTaskResponse, error)
	SearchUserTasksWithResponse(ctx context.Context, body camundav89.SearchUserTasksJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchUserTasksResponse, error)
	SearchUserTaskEffectiveVariablesWithResponse(ctx context.Context, userTaskKey camundav89.UserTaskKey, params *camundav89.SearchUserTaskEffectiveVariablesParams, body camundav89.SearchUserTaskEffectiveVariablesJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchUserTaskEffectiveVariablesResponse, error)
}

type GenUserTaskClientTasklist interface {
	GetTaskByIdWithResponse(ctx context.Context, taskId string, reqEditors ...tasklistv89.RequestEditorFn) (*tasklistv89.GetTaskByIdResponse, error)
}

var _ API = (*Service)(nil)
var _ GenUserTaskClientCamunda = (*camundav89.ClientWithResponses)(nil)
var _ GenUserTaskClientTasklist = (*tasklistv89.ClientWithResponses)(nil)
