package process

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	types "github.com/grafvonb/c8volt/typex"
)

type API interface {
	SearchProcessDefinitions(ctx context.Context, filter ProcessDefinitionFilter, opts ...options.FacadeOption) (ProcessDefinitions, error)
	SearchProcessDefinitionsLatest(ctx context.Context, filter ProcessDefinitionFilter, opts ...options.FacadeOption) (ProcessDefinitions, error)
	GetProcessDefinition(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessDefinition, error)

	CreateProcessInstance(ctx context.Context, data ProcessInstanceData, opts ...options.FacadeOption) (ProcessInstance, error)
	CreateProcessInstances(ctx context.Context, datas []ProcessInstanceData, opts ...options.FacadeOption) ([]ProcessInstance, error)
	GetProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstance, error)
	SearchProcessInstances(ctx context.Context, filter ProcessInstanceFilter, size int32, opts ...options.FacadeOption) (ProcessInstances, error)
	CancelProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (CancelReport, ProcessInstances, error)
	DeleteProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (DeleteReport, error)
	GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstances, error)
	FilterProcessInstanceWithOrphanParent(ctx context.Context, items []ProcessInstance, opts ...options.FacadeOption) ([]ProcessInstance, error)
	WaitForProcessInstanceState(ctx context.Context, key string, desired States, opts ...options.FacadeOption) (StateReport, ProcessInstance, error)
	Walker

	GetProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (ProcessInstances, error)
	CreateNProcessInstances(ctx context.Context, data ProcessInstanceData, n int, wantedWorkers int, opts ...options.FacadeOption) ([]ProcessInstance, error)
	CancelProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (CancelReports, error)
	DeleteProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (DeleteReports, error)
	WaitForProcessInstancesState(ctx context.Context, keys types.Keys, desired States, wantedWorkers int, opts ...options.FacadeOption) (StateReports, error)
}

var _ API = (*client)(nil)
