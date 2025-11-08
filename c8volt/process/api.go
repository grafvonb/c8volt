package process

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/foptions"
)

type API interface {
	SearchProcessDefinitions(ctx context.Context, filter ProcessDefinitionFilter, opts ...foptions.FacadeOption) (ProcessDefinitions, error)
	SearchProcessDefinitionsLatest(ctx context.Context, filter ProcessDefinitionFilter, opts ...foptions.FacadeOption) (ProcessDefinitions, error)
	GetProcessDefinition(ctx context.Context, key string, opts ...foptions.FacadeOption) (ProcessDefinition, error)

	CreateProcessInstance(ctx context.Context, data ProcessInstanceData, opts ...foptions.FacadeOption) (ProcessInstance, error)
	CreateProcessInstances(ctx context.Context, datas []ProcessInstanceData, opts ...foptions.FacadeOption) ([]ProcessInstance, error)
	GetProcessInstance(ctx context.Context, key string, opts ...foptions.FacadeOption) (ProcessInstance, error)
	SearchProcessInstances(ctx context.Context, filter ProcessInstanceFilter, size int32, opts ...foptions.FacadeOption) (ProcessInstances, error)
	CancelProcessInstance(ctx context.Context, key string, opts ...foptions.FacadeOption) (CancelReport, error)
	DeleteProcessInstance(ctx context.Context, key string, opts ...foptions.FacadeOption) (DeleteReport, error)
	GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...foptions.FacadeOption) (ProcessInstances, error)
	FilterProcessInstanceWithOrphanParent(ctx context.Context, items []ProcessInstance, opts ...foptions.FacadeOption) ([]ProcessInstance, error)
	WaitForProcessInstanceState(ctx context.Context, key string, desired States, opts ...foptions.FacadeOption) (State, error)
	Walker

	CreateNProcessInstances(ctx context.Context, data ProcessInstanceData, n int, parallel int, opts ...foptions.FacadeOption) ([]ProcessInstance, error)
	CancelProcessInstances(ctx context.Context, keys []string, parallel int, failFast bool, opts ...foptions.FacadeOption) (CancelReports, error)
	DeleteProcessInstances(ctx context.Context, keys []string, parallel int, failFast bool, opts ...foptions.FacadeOption) (DeleteReports, error)
}

var _ API = (*client)(nil)
