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
	SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
}

// GenIncidentClientCamunda captures the generated Camunda calls used by the v8.8 incident service.
type GenIncidentClientCamunda interface {
	SearchProcessInstanceIncidentsWithResponse(ctx context.Context, processInstanceKey string, body camundav88.SearchProcessInstanceIncidentsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchProcessInstanceIncidentsResponse, error)
}

var _ API = (*Service)(nil)
var _ GenIncidentClientCamunda = (*camundav88.ClientWithResponses)(nil)
