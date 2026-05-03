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

type API interface {
	GetUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error)
}

type GenUserTaskClientCamunda interface {
	SearchUserTasksWithResponse(ctx context.Context, body camundav88.SearchUserTasksJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchUserTasksResponse, error)
}

type GenUserTaskClientTasklist interface {
	SearchTasksWithResponse(ctx context.Context, body tasklistv88.SearchTasksJSONRequestBody, reqEditors ...tasklistv88.RequestEditorFn) (*tasklistv88.SearchTasksResponse, error)
}

var _ API = (*Service)(nil)
var _ GenUserTaskClientCamunda = (*camundav88.ClientWithResponses)(nil)
var _ GenUserTaskClientTasklist = (*tasklistv88.ClientWithResponses)(nil)
