// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API exposes incident operations supported by the v8.10 incident service.
type API interface {
	GetIncident(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceIncidentDetail, error)
	ResolveIncident(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error)
	SearchIncidents(ctx context.Context, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
	SearchIncidentsPages(ctx context.Context, filter d.IncidentFilter, page d.IncidentPageRequest, limit int32, visitor d.IncidentSearchPageVisitor, opts ...services.CallOption) (d.IncidentSearchPagesResult, error)
	SearchIncidentsPage(ctx context.Context, filter d.IncidentFilter, page d.IncidentPageRequest, opts ...services.CallOption) (d.IncidentPage, error)
	SearchIncidentsTotal(ctx context.Context, filter d.IncidentFilter, page d.IncidentPageRequest, opts ...services.CallOption) (int64, error)
	SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
	WaitForIncidentResolved(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error)
	WaitForProcessInstanceIncidentsResolved(ctx context.Context, processInstanceKey string, incidentKeys []string, opts ...services.CallOption) (d.IncidentResolutionResponse, error)
}

// GenIncidentClientCamunda captures the generated Camunda calls used by the v8.10 incident service.
type GenIncidentClientCamunda interface {
	GetIncidentWithResponse(ctx context.Context, incidentKey camundav810.IncidentKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetIncidentResponse, error)
	ResolveIncidentWithResponse(ctx context.Context, incidentKey camundav810.IncidentKey, body camundav810.ResolveIncidentJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.ResolveIncidentResponse, error)
	SearchIncidentsWithResponse(ctx context.Context, body camundav810.SearchIncidentsJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchIncidentsResponse, error)
	SearchProcessInstanceIncidentsWithResponse(ctx context.Context, processInstanceKey camundav810.ProcessInstanceKey, body camundav810.SearchProcessInstanceIncidentsJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstanceIncidentsResponse, error)
}

var _ API = (*Service)(nil)
var _ GenIncidentClientCamunda = (*camundav810.ClientWithResponses)(nil)
