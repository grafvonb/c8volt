package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	SearchProcessDefinitions(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error)
	SearchProcessDefinitionsLatest(ctx context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error)
	GetProcessDefinition(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error)
	GetProcessDefinitionXML(ctx context.Context, key string, opts ...services.CallOption) (string, error)
}

type GenProcessDefinitionClientCamunda interface {
	GetProcessDefinitionWithResponse(ctx context.Context, processDefinitionKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error)
	GetProcessDefinitionXMLWithResponse(ctx context.Context, processDefinitionKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionXMLResponse, error)
	SearchProcessDefinitionsWithResponse(ctx context.Context, body camundav88.SearchProcessDefinitionsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessDefinitionsResponse, error)
	GetProcessDefinitionStatisticsWithResponse(ctx context.Context, processDefinitionKey string, body camundav88.GetProcessDefinitionStatisticsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionStatisticsResponse, error)
	SearchProcessInstancesWithResponse(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error)
	SearchIncidentsWithResponse(ctx context.Context, body camundav88.SearchIncidentsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchIncidentsResponse, error)
}

var _ API = (*Service)(nil)
var _ GenProcessDefinitionClientCamunda = (*camundav88.ClientWithResponses)(nil)
