package v87

import (
	"context"

	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/typex"
)

type API interface {
	CreateProcessInstance(ctx context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error)
	GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error)
	GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstance, error)
	FilterProcessInstanceWithOrphanParent(ctx context.Context, items []d.ProcessInstance, opts ...services.CallOption) ([]d.ProcessInstance, error)
	SearchForProcessInstancesPage(ctx context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error)
	SearchForProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error)
	CancelProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error)
	DeleteProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error)
	GetProcessInstanceStateByKey(ctx context.Context, key string, opts ...services.CallOption) (d.State, d.ProcessInstance, error)
	WaitForProcessInstanceState(ctx context.Context, key string, desired d.States, opts ...services.CallOption) (d.StateResponse, d.ProcessInstance, error)
	Ancestry(ctx context.Context, startKey string, opts ...services.CallOption) (rootKey string, path []string, chain map[string]d.ProcessInstance, err error)
	Descendants(ctx context.Context, rootKey string, opts ...services.CallOption) (desc []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error)
	Family(ctx context.Context, startKey string, opts ...services.CallOption) (fam []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error)
	AncestryResult(ctx context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error)
	DescendantsResult(ctx context.Context, rootKey string, opts ...services.CallOption) (pitraversal.Result, error)
	FamilyResult(ctx context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error)
	GetProcessInstances(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) ([]d.ProcessInstance, error)
	WaitForProcessInstancesState(ctx context.Context, keys typex.Keys, desired d.States, wantedWorkers int, opts ...services.CallOption) (d.StateResponses, error)
}

type GenProcessInstanceClientCamunda interface {
	PostProcessInstancesProcessInstanceKeyCancellationWithResponse(ctx context.Context, processInstanceKey string, body camundav87.PostProcessInstancesProcessInstanceKeyCancellationJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesProcessInstanceKeyCancellationResponse, error)
	PostProcessInstancesWithResponse(ctx context.Context, body camundav87.PostProcessInstancesJSONRequestBody, reqEditors ...camundav87.RequestEditorFn) (*camundav87.PostProcessInstancesResponse, error)
}

type GenProcessInstanceClientOperate interface {
	SearchProcessInstancesWithResponse(ctx context.Context, body operatev87.SearchProcessInstancesJSONRequestBody, reqEditors ...operatev87.RequestEditorFn) (*operatev87.SearchProcessInstancesResponse, error)
	DeleteProcessInstanceAndAllDependantDataByKeyWithResponse(ctx context.Context, key int64, reqEditors ...operatev87.RequestEditorFn) (*operatev87.DeleteProcessInstanceAndAllDependantDataByKeyResponse, error)
}

var _ API = (*Service)(nil)
var _ GenProcessInstanceClientCamunda = (*camundav87.ClientWithResponses)(nil)
var _ GenProcessInstanceClientOperate = (*operatev87.ClientWithResponses)(nil)
