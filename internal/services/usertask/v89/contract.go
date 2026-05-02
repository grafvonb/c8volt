// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	GetUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error)
}

type GenUserTaskClientCamunda interface {
	SearchUserTasksWithResponse(ctx context.Context, body camundav89.SearchUserTasksJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchUserTasksResponse, error)
}

var _ API = (*Service)(nil)
var _ GenUserTaskClientCamunda = (*camundav89.ClientWithResponses)(nil)
