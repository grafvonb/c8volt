package resource

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/options"
)

type API interface {
	DeployProcessDefinition(ctx context.Context, tenantId string, units []DeploymentUnitData, opts ...options.FacadeOption) ([]ProcessDefinitionDeployment, error)

	DeleteProcessDefinitionByKey(ctx context.Context, key string, opts ...options.FacadeOption) error
	DeleteProcessDefinitionVersionsByBpmnProcessId(ctx context.Context, bpmnProcessId string, opts ...options.FacadeOption) error
}

var _ API = (*client)(nil)
