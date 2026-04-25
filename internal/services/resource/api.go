package resource

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/resource/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/resource/v88"
	v89 "github.com/grafvonb/c8volt/internal/services/resource/v89"
)

type API interface {
	Deploy(ctx context.Context, units []d.DeploymentUnitData, opts ...services.CallOption) (d.Deployment, error)
	Delete(ctx context.Context, resourceKey string, opts ...services.CallOption) error
	Get(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.Resource, error)
}

// Both supported versioned services must continue to satisfy the shared
// resource service surface while the internals are refactored.
var _ API = (*v87.Service)(nil)
var _ API = (*v88.Service)(nil)
var _ API = (*v89.Service)(nil)
var _ API = (v87.API)(nil)
var _ API = (v88.API)(nil)
var _ API = (v89.API)(nil)
