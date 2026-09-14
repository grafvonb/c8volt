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
	GetUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error)
	GetNativeUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error)
	SearchUserTasksPage(ctx context.Context, query d.UserTaskSearchQuery, page d.UserTaskPageRequest, opts ...services.CallOption) (d.UserTaskSearchPage, error)
}

// GenUserTaskClientCamunda captures the generated native read and legacy search operations used by this adapter.
type GenUserTaskClientCamunda interface {
	GetUserTaskWithResponse(ctx context.Context, userTaskKey camundav88.UserTaskKey, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetUserTaskResponse, error)
	SearchUserTasksWithResponse(ctx context.Context, body camundav88.SearchUserTasksJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error)
}

type GenUserTaskClientTasklist interface {
	GetTaskByIdWithResponse(ctx context.Context, taskId string, reqEditors ...tasklistv88.RequestEditorFn) (*tasklistv88.GetTaskByIdResponse, error)
}

var _ API = (*Service)(nil)
var _ GenUserTaskClientCamunda = (*camundav88.ClientWithResponses)(nil)
var _ GenUserTaskClientTasklist = (*tasklistv88.ClientWithResponses)(nil)
