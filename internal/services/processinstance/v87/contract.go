package v87

import (
	"context"

	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
)

type GenClusterClientCamunda interface {
	PostProcessInstancesProcessInstanceKeyCancellationWithResponse(ctx context.Context, processInstanceKey string, body camundav87.PostProcessInstancesProcessInstanceKeyCancellationJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesProcessInstanceKeyCancellationResponse, error)
}

type GenClusterClientOperate interface {
	GetProcessDefinitionByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessDefinitionByKeyResponse, error)
	SearchProcessDefinitionsWithResponse(ctx context.Context, body operatev87.SearchProcessDefinitionsJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessDefinitionsResponse, error)
	GetProcessInstanceByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetProcessInstanceByKeyResponse, error)
	SearchProcessInstancesWithResponse(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error)
	DeleteProcessInstanceAndAllDependantDataByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error)
}

var _ GenClusterClientCamunda = (*camundav87.ClientWithResponses)(nil)
var _ GenClusterClientOperate = (*operatev87.ClientWithResponses)(nil)
