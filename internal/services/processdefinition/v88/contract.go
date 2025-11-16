package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
)

type GenProcessDefinitionClientCamunda interface {
	GetProcessDefinitionWithResponse(ctx context.Context, processDefinitionKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionResponse, error)
	SearchProcessDefinitionsWithResponse(ctx context.Context, body camundav88.SearchProcessDefinitionsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessDefinitionsResponse, error)
	GetProcessDefinitionStatisticsWithResponse(ctx context.Context, processDefinitionKey string, body camundav88.GetProcessDefinitionStatisticsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessDefinitionStatisticsResponse, error)
}

var _ GenProcessDefinitionClientCamunda = (*camundav88.ClientWithResponses)(nil)
