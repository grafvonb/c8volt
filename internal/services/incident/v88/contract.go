// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API exposes incident operations supported by the v8.8 incident service.
type API interface {
	GetIncident(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceIncidentDetail, error)
	ResolveIncident(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error)
	ResolveProcessInstanceIncidents(ctx context.Context, processInstanceKey string, opts ...services.CallOption) (d.IncidentResolutionResponse, error)
	SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
	WaitForIncidentResolved(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error)
	WaitForProcessInstanceIncidentsResolved(ctx context.Context, processInstanceKey string, incidentKeys []string, opts ...services.CallOption) (d.IncidentResolutionResponse, error)
}

// GenIncidentClientCamunda captures the generated Camunda calls used by the v8.8 incident service.
type GenIncidentClientCamunda interface {
	GetIncidentWithResponse(ctx context.Context, incidentKey camundav88.IncidentKey, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetIncidentResponse, error)
	ResolveIncidentWithResponse(ctx context.Context, incidentKey camundav88.IncidentKey, body camundav88.ResolveIncidentJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.ResolveIncidentResponse, error)
	ResolveProcessInstanceIncidentsWithResponse(ctx context.Context, processInstanceKey camundav88.ProcessInstanceKey, reqEditors ...camundav88.RequestEditorFn) (*camundav88.ResolveProcessInstanceIncidentsResponse, error)
	SearchProcessInstanceIncidentsWithResponse(ctx context.Context, processInstanceKey string, body camundav88.SearchProcessInstanceIncidentsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstanceIncidentsResponse, error)
}

var _ API = (*Service)(nil)
var _ GenIncidentClientCamunda = (*camundav88.ClientWithResponses)(nil)
