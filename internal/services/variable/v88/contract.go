// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
)

// GenVariableClientCamunda captures the generated Camunda calls used by the v8.8 variable service.
type GenVariableClientCamunda interface {
	SearchVariablesWithResponse(ctx context.Context, params *camundav88.SearchVariablesParams, body camundav88.SearchVariablesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchVariablesResponse, error)
	CreateElementInstanceVariablesWithResponse(ctx context.Context, elementInstanceKey camundav88.ElementInstanceKey, body camundav88.CreateElementInstanceVariablesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateElementInstanceVariablesResponse, error)
}

var _ GenVariableClientCamunda = (*camundav88.ClientWithResponses)(nil)
