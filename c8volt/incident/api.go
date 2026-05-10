// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incident

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	types "github.com/grafvonb/c8volt/typex"
)

type API interface {
	GetIncident(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstanceIncidentDetail, error)
	GetIncidents(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (Incidents, error)
	SearchIncidents(ctx context.Context, filter Filter, size int32, opts ...options.FacadeOption) (Incidents, error)
	SearchIncidentsPage(ctx context.Context, filter Filter, page PageRequest, opts ...options.FacadeOption) (Page, error)
	SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...options.FacadeOption) ([]ProcessInstanceIncidentDetail, error)
	ResolveIncident(ctx context.Context, key string, opts ...options.FacadeOption) (ResolutionResult, error)
	ResolveIncidents(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (ResolutionResults, error)
	ResolveProcessInstanceIncidents(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstanceResolutionResult, error)
	ResolveProcessInstancesIncidents(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (ProcessInstanceResolutionResults, error)
}

var _ API = (*client)(nil)
