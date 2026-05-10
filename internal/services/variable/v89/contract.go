// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
)

// GenVariableClientCamunda captures the generated Camunda calls used by the v8.9 variable service.
type GenVariableClientCamunda interface {
	SearchVariablesWithResponse(ctx context.Context, params *camundav89.SearchVariablesParams, body camundav89.SearchVariablesJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchVariablesResponse, error)
	CreateElementInstanceVariablesWithResponse(ctx context.Context, elementInstanceKey camundav89.ElementInstanceKey, body camundav89.CreateElementInstanceVariablesJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateElementInstanceVariablesResponse, error)
}

var _ GenVariableClientCamunda = (*camundav89.ClientWithResponses)(nil)
