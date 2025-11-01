package v87

import (
	"context"

	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
)

type GenVariableClient interface {
	GetVariableByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetVariableByKeyResponse, error)
	SearchVariablesForProcessInstancesWithResponse(ctx context.Context, body operatev87.SearchVariablesForProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchVariablesForProcessInstancesResponse, error)
}

var _ GenVariableClient = (*operatev87.ClientWithResponses)(nil)
