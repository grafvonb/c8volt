package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
)

type GenIncidentClientCamunda interface {
	SearchIncidentsWithResponse(ctx context.Context, body camundav88.SearchIncidentsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchIncidentsResponse, error)
	GetIncidentWithResponse(ctx context.Context, incidentKey string, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetIncidentResponse, error)
	ResolveIncidentWithResponse(ctx context.Context, incidentKey string, body camundav88.ResolveIncidentJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.ResolveIncidentResponse, error)
}

var _ GenIncidentClientCamunda = (*camundav88.ClientWithResponses)(nil)
