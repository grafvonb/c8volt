// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
)

// GenVariableClientCamunda captures the generated Camunda calls used by the v8.10 variable service.
type GenVariableClientCamunda interface {
	SearchVariablesWithResponse(ctx context.Context, params *camundav810.SearchVariablesParams, body camundav810.SearchVariablesJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error)
	CreateElementInstanceVariablesWithResponse(ctx context.Context, elementInstanceKey camundav810.ElementInstanceKey, body camundav810.CreateElementInstanceVariablesJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error)
}

var _ GenVariableClientCamunda = (*camundav810.ClientWithResponses)(nil)
