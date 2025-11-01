package v87

import (
	"context"

	tasklistv87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/tasklist"
)

type GenUserTaskClient interface {
	GetVariableByIdWithResponse(ctx context.Context, variableId string, reqEditors ...tasklistv87.RequestEditorFn) (*tasklistv87.GetVariableByIdResponse, error)
	SearchTaskVariablesWithResponse(ctx context.Context, taskId string, body tasklistv87.SearchTaskVariablesJSONRequestBody, reqEditors ...tasklistv87.RequestEditorFn) (*tasklistv87.SearchTaskVariablesResponse, error)
}

var _ GenUserTaskClient = (*tasklistv87.ClientWithResponses)(nil)
