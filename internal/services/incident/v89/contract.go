// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API exposes incident operations supported by the v8.9 incident service.
type API interface {
	GetIncident(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceIncidentDetail, error)
	ResolveIncident(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error)
	SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
	WaitForIncidentResolved(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error)
	WaitForProcessInstanceIncidentsResolved(ctx context.Context, processInstanceKey string, incidentKeys []string, opts ...services.CallOption) (d.IncidentResolutionResponse, error)
}

// GenIncidentClientCamunda captures the generated Camunda calls used by the v8.9 incident service.
type GenIncidentClientCamunda interface {
	GetIncidentWithResponse(ctx context.Context, incidentKey camundav89.IncidentKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetIncidentResponse, error)
	ResolveIncidentWithResponse(ctx context.Context, incidentKey camundav89.IncidentKey, body camundav89.ResolveIncidentJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.ResolveIncidentResponse, error)
	SearchProcessInstanceIncidentsWithResponse(ctx context.Context, processInstanceKey camundav89.ProcessInstanceKey, body camundav89.SearchProcessInstanceIncidentsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstanceIncidentsResponse, error)
}

var _ API = (*Service)(nil)
var _ GenIncidentClientCamunda = (*camundav89.ClientWithResponses)(nil)
