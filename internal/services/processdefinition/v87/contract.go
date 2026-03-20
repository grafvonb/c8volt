package v87

import (
	"context"

	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	SearchProcessDefinitions(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error)
	SearchProcessDefinitionsLatest(ctx context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error)
	GetProcessDefinition(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error)
}

type GenProcessDefinitionClientOperate interface {
	GetProcessDefinitionByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessDefinitionByKeyResponse, error)
	SearchProcessDefinitionsWithResponse(ctx context.Context, body operatev87.SearchProcessDefinitionsJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessDefinitionsResponse, error)
}

var _ API = (*Service)(nil)
var _ GenProcessDefinitionClientOperate = (*operatev87.ClientWithResponses)(nil)
