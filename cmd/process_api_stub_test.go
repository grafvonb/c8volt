package cmd

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	types "github.com/grafvonb/c8volt/typex"
)

type stubProcessAPI struct {
	dryRunCancelOrDeletePlan func(context.Context, types.Keys, ...options.FacadeOption) (process.DryRunPIKeyExpansion, error)
	cancelProcessInstances   func(context.Context, types.Keys, int, ...options.FacadeOption) (process.CancelReports, error)
	deleteProcessInstances   func(context.Context, types.Keys, int, ...options.FacadeOption) (process.DeleteReports, error)
	filterOrphanParent       func(context.Context, []process.ProcessInstance, ...options.FacadeOption) ([]process.ProcessInstance, error)
}

func (stubProcessAPI) SearchProcessDefinitions(context.Context, process.ProcessDefinitionFilter, ...options.FacadeOption) (process.ProcessDefinitions, error) {
	panic("unexpected call")
}

func (stubProcessAPI) SearchProcessDefinitionsLatest(context.Context, process.ProcessDefinitionFilter, ...options.FacadeOption) (process.ProcessDefinitions, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetProcessDefinition(context.Context, string, ...options.FacadeOption) (process.ProcessDefinition, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetProcessDefinitionXML(context.Context, string, ...options.FacadeOption) (string, error) {
	panic("unexpected call")
}

func (stubProcessAPI) CreateProcessInstance(context.Context, process.ProcessInstanceData, ...options.FacadeOption) (process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) CreateProcessInstances(context.Context, []process.ProcessInstanceData, ...options.FacadeOption) ([]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetProcessInstance(context.Context, string, ...options.FacadeOption) (process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) LookupProcessInstance(context.Context, string, ...options.FacadeOption) (process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) LookupProcessInstanceStateByKey(context.Context, string, ...options.FacadeOption) (process.StateReport, process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) SearchProcessInstancesPage(context.Context, process.ProcessInstanceFilter, process.ProcessInstancePageRequest, ...options.FacadeOption) (process.ProcessInstancePage, error) {
	panic("unexpected call")
}

func (stubProcessAPI) SearchProcessInstances(context.Context, process.ProcessInstanceFilter, int32, ...options.FacadeOption) (process.ProcessInstances, error) {
	panic("unexpected call")
}

func (stubProcessAPI) CancelProcessInstance(context.Context, string, ...options.FacadeOption) (process.CancelReport, process.ProcessInstances, error) {
	panic("unexpected call")
}

func (stubProcessAPI) DeleteProcessInstance(context.Context, string, ...options.FacadeOption) (process.DeleteReport, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetDirectChildrenOfProcessInstance(context.Context, string, ...options.FacadeOption) (process.ProcessInstances, error) {
	panic("unexpected call")
}

func (s stubProcessAPI) FilterProcessInstanceWithOrphanParent(ctx context.Context, items []process.ProcessInstance, opts ...options.FacadeOption) ([]process.ProcessInstance, error) {
	if s.filterOrphanParent == nil {
		panic("unexpected call")
	}
	return s.filterOrphanParent(ctx, items, opts...)
}

func (stubProcessAPI) WaitForProcessInstanceState(context.Context, string, process.States, ...options.FacadeOption) (process.StateReport, process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) Ancestry(context.Context, string, ...options.FacadeOption) (string, []string, map[string]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) Descendants(context.Context, string, ...options.FacadeOption) ([]string, map[string][]string, map[string]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) Family(context.Context, string, ...options.FacadeOption) ([]string, map[string][]string, map[string]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) AncestryResult(context.Context, string, ...options.FacadeOption) (process.TraversalResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) DescendantsResult(context.Context, string, ...options.FacadeOption) (process.TraversalResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) FamilyResult(context.Context, string, ...options.FacadeOption) (process.TraversalResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetProcessInstances(context.Context, types.Keys, int, ...options.FacadeOption) (process.ProcessInstances, error) {
	panic("unexpected call")
}

func (stubProcessAPI) CreateNProcessInstances(context.Context, process.ProcessInstanceData, int, int, ...options.FacadeOption) ([]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (s stubProcessAPI) CancelProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.CancelReports, error) {
	if s.cancelProcessInstances == nil {
		panic("unexpected call")
	}
	return s.cancelProcessInstances(ctx, keys, wantedWorkers, opts...)
}

func (s stubProcessAPI) DeleteProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.DeleteReports, error) {
	if s.deleteProcessInstances == nil {
		panic("unexpected call")
	}
	return s.deleteProcessInstances(ctx, keys, wantedWorkers, opts...)
}

func (stubProcessAPI) WaitForProcessInstancesState(context.Context, types.Keys, process.States, int, ...options.FacadeOption) (process.StateReports, error) {
	panic("unexpected call")
}

func (stubProcessAPI) DryRunCancelOrDeleteGetPIKeys(context.Context, types.Keys, ...options.FacadeOption) (types.Keys, types.Keys, error) {
	panic("unexpected call")
}

func (s stubProcessAPI) DryRunCancelOrDeletePlan(ctx context.Context, keys types.Keys, opts ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
	if s.dryRunCancelOrDeletePlan == nil {
		panic("unexpected call")
	}
	return s.dryRunCancelOrDeletePlan(ctx, keys, opts...)
}

var _ process.API = stubProcessAPI{}
