package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
)

type GenUserTaskClient interface {
	GetVariableWithResponse(ctx context.Context, variableKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetVariableResponse, error)
	SearchUserTaskVariablesWithResponse(ctx context.Context, userTaskKey string, body camundav88.SearchUserTaskVariablesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchUserTaskVariablesResponse, error)
}

var _ GenUserTaskClient = (*camundav88.ClientWithResponses)(nil)
