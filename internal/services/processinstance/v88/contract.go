package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	operatev88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/operate"
)

type GenProcessInstanceClientCamunda interface {
	CancelProcessInstanceWithResponse(ctx context.Context, processInstanceKey string, body camundav88.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CancelProcessInstanceResponse, error)
	GetProcessInstanceWithResponse(ctx context.Context, processInstanceKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetProcessInstanceResponse, error)
	CreateProcessInstanceWithResponse(ctx context.Context, body camundav88.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.CreateProcessInstanceResponse, error)
	SearchProcessInstancesWithResponse(ctx context.Context, body camundav88.SearchProcessInstancesJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstancesResponse, error)
}

type GenProcessInstanceClientOperate interface {
	DeleteProcessInstanceAndAllDependantDataByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev88.RequestEditorFn) (*operatev88.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error)
}

var _ GenProcessInstanceClientCamunda = (*camundav88.ClientWithResponses)(nil)
var _ GenProcessInstanceClientOperate = (*operatev88.ClientWithResponses)(nil)
