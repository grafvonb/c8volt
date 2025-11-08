package processinstance

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/processinstance/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/processinstance/v88"
)

type API interface {
	CreateProcessInstance(ctx context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error)
	GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error)
	GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstance, error)
	FilterProcessInstanceWithOrphanParent(ctx context.Context, items []d.ProcessInstance, opts ...services.CallOption) ([]d.ProcessInstance, error)
	SearchForProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error)
	CancelProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.CancelResponse, error)
	DeleteProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error)
	GetProcessInstanceStateByKey(ctx context.Context, key string, opts ...services.CallOption) (d.State, d.ProcessInstance, error)
	WaitForProcessInstanceState(ctx context.Context, key string, desired d.States, opts ...services.CallOption) (d.State, d.ProcessInstance, error)
	Ancestry(ctx context.Context, startKey string, opts ...services.CallOption) (rootKey string, path []string, chain map[string]d.ProcessInstance, err error)
	Descendants(ctx context.Context, rootKey string, opts ...services.CallOption) (desc []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error)
	Family(ctx context.Context, startKey string, opts ...services.CallOption) (fam []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error)
}

var _ API = (*v87.Service)(nil)
var _ API = (*v88.Service)(nil)
