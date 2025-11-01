package v88

import (
	"context"

	operatev88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/operate"
)

type GenClusterClientOperate interface {
	GetProcessDefinitionByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev88.RequestEditorFn) (*operatev88.GetProcessDefinitionByKeyResponse, error)
	SearchProcessDefinitionsWithResponse(ctx context.Context, body operatev88.SearchProcessDefinitionsJSONRequestBody, reqEditors ...operatev88.RequestEditorFn) (*operatev88.SearchProcessDefinitionsResponse, error)
}

var _ GenClusterClientOperate = (*operatev88.ClientWithResponses)(nil)
