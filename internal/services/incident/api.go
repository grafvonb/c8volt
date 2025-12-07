package incident

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/incident/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/incident/v88"
)

type API interface {
	SearchIncidents(ctx context.Context, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.Incident, error)
	GetIncident(ctx context.Context, key string, opts ...services.CallOption) (d.Incident, error)
	ResolveIncident(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResponse, error)
}

var _ API = (*v87.Service)(nil)
var _ API = (*v88.Service)(nil)
