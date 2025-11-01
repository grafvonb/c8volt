package resource

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/resource/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/resource/v88"
)

type API interface {
	Deploy(ctx context.Context, tenantId string, units []d.DeploymentUnitData, opts ...services.CallOption) (d.Deployment, error)
	Delete(ctx context.Context, resourceKey string, opts ...services.CallOption) error
}

var _ API = (*v87.Service)(nil)
var _ API = (*v88.Service)(nil)
