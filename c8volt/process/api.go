package process

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/options"
)

type API interface {
	SearchProcessDefinitions(ctx context.Context, filter ProcessDefinitionSearchFilterOpts, size int32, opts ...options.FacadeOption) (ProcessDefinitions, error)
	GetProcessDefinitionsLatest(ctx context.Context) (ProcessDefinitions, error)
	GetProcessDefinitionByKey(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessDefinition, error)
	GetProcessDefinitionsByBpmnProcessId(ctx context.Context, bpmnProcessId string, opts ...options.FacadeOption) (ProcessDefinitions, error)
	GetProcessDefinitionByBpmnProcessIdLatest(ctx context.Context, bpmnProcessId string, opts ...options.FacadeOption) (ProcessDefinition, error)
	GetProcessDefinitionByBpmnProcessIdAndVersion(ctx context.Context, bpmnProcessId string, version int32, opts ...options.FacadeOption) (ProcessDefinition, error)

	CreateProcessInstance(ctx context.Context, data ProcessInstanceData, opts ...options.FacadeOption) (ProcessInstance, error)
	CreateProcessInstances(ctx context.Context, datas []ProcessInstanceData, opts ...options.FacadeOption) ([]ProcessInstance, error)
	GetProcessInstanceByKey(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstance, error)
	SearchProcessInstances(ctx context.Context, filter ProcessInstanceSearchFilterOpts, size int32, opts ...options.FacadeOption) (ProcessInstances, error)
	CancelProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (CancelResponse, error)
	GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstances, error)
	FilterProcessInstanceWithOrphanParent(ctx context.Context, items []ProcessInstance, opts ...options.FacadeOption) ([]ProcessInstance, error)
	DeleteProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ChangeStatus, error)
	WaitForProcessInstanceState(ctx context.Context, key string, desired States, opts ...options.FacadeOption) (State, error)
	Walker

	CreateNProcessInstances(ctx context.Context, data ProcessInstanceData, n int, parallel int, opts ...options.FacadeOption) ([]ProcessInstance, error)
}

var _ API = (*client)(nil)
