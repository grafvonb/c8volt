package v89

import (
	"context"
	"io"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
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
	GetProcessDefinitionWithResponse(ctx context.Context, processDefinitionKey string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionResponse, error)
	GetProcessDefinitionXMLWithResponse(ctx context.Context, processDefinitionKey string, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionXMLResponse, error)
	SearchProcessDefinitionsWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessDefinitionsResponse, error)
	GetProcessDefinitionStatisticsWithResponse(ctx context.Context, processDefinitionKey string, body camundav89.GetProcessDefinitionStatisticsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionStatisticsResponse, error)
	GetProcessDefinitionInstanceVersionStatisticsWithResponse(ctx context.Context, body camundav89.GetProcessDefinitionInstanceVersionStatisticsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessDefinitionInstanceVersionStatisticsResponse, error)
}

var _ GenProcessDefinitionClientCamunda = (*camundav89.ClientWithResponses)(nil)
