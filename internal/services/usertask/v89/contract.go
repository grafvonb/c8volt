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
	GetUserTaskWithResponse(ctx context.Context, userTaskKey camundav89.UserTaskKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetUserTaskResponse, error)
}

var _ API = (*Service)(nil)
var _ GenUserTaskClientCamunda = (*camundav89.ClientWithResponses)(nil)
