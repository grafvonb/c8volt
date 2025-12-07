package v87

import (
	"context"

	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
)

type GenIncidentClientCamunda interface {
	PostIncidentsSearchWithResponse(ctx context.Context, body camundav87.PostIncidentsSearchJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostIncidentsSearchResponse, error)
	PostIncidentsSearchWithApplicationVndCamundaAPIKeysNumberPlusJSONBodyWithResponse(ctx context.Context, body camundav87.PostIncidentsSearchApplicationVndCamundaAPIKeysNumberPlusJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostIncidentsSearchResponse, error)
	PostIncidentsSearchWithApplicationVndCamundaAPIKeysStringPlusJSONBodyWithResponse(ctx context.Context, body camundav87.PostIncidentsSearchApplicationVndCamundaAPIKeysStringPlusJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostIncidentsSearchResponse, error)
	GetIncidentsIncidentKeyWithResponse(ctx context.Context, incidentKey string, reqEditors ...camundav87.RequestEditorFn) (*camundav87.GetIncidentsIncidentKeyResponse, error)
	PostIncidentsIncidentKeyResolutionWithResponse(ctx context.Context, incidentKey string, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostIncidentsIncidentKeyResolutionResponse, error)
}

type GenIncidentClientOperate interface {
	SearchIncidentsWithResponse(ctx context.Context, body operatev87.SearchIncidentsJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchIncidentsResponse, error)
	GetIncidentByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.GetIncidentByKeyResponse, error)
}

var _ GenIncidentClientCamunda = (*camundav87.ClientWithResponses)(nil)
var _ GenIncidentClientOperate = (*operatev87.ClientWithResponses)(nil)
