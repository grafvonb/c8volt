// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
)

type GenUserTaskClientCamunda interface {
	GetVariableWithResponse(ctx context.Context, variableKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetVariableResponse, error)
	SearchUserTaskVariablesWithResponse(ctx context.Context, userTaskKey string, params *camundav88.SearchUserTaskVariablesParams, body camundav88.SearchUserTaskVariablesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchUserTaskVariablesResponse, error)
}

var _ GenUserTaskClientCamunda = (*camundav88.ClientWithResponses)(nil)
