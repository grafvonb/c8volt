package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
)

type GenVariableClientCamunda interface {
	GetVariableWithResponse(ctx context.Context, variableKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetVariableResponse, error)
	SearchVariablesWithResponse(ctx context.Context, params *camundav88.SearchVariablesParams, body camundav88.SearchVariablesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchVariablesResponse, error)
}

var _ GenVariableClientCamunda = (*camundav88.ClientWithResponses)(nil)
