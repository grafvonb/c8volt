package v87

import (
	"context"

	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
)

type GenClusterClientCamunda interface {
}

type GenClusterClientOperate interface {
	GetProcessDefinitionByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessDefinitionByKeyResponse, error)
	SearchProcessDefinitionsWithResponse(ctx context.Context, body operatev87.SearchProcessDefinitionsJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessDefinitionsResponse, error)
}

var _ GenClusterClientOperate = (*operatev87.ClientWithResponses)(nil)
