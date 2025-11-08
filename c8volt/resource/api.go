package resource

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/foptions"
)

type API interface {
	DeployProcessDefinition(ctx context.Context, tenantId string, units []DeploymentUnitData, opts ...foptions.FacadeOption) ([]ProcessDefinitionDeployment, error)

	DeleteProcessDefinition(ctx context.Context, key string, opts ...foptions.FacadeOption) (DeleteReport, error)
	DeleteProcessDefinitions(ctx context.Context, keys []string, parallel int, failFast bool, opts ...foptions.FacadeOption) (DeleteReports, error)
}

var _ API = (*client)(nil)
