package process

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/options"
	d "github.com/grafvonb/c8volt/internal/domain"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/toolx"
)

type API interface {
	SearchProcessDefinitions(ctx context.Context, filter ProcessDefinitionSearchFilterOpts, size int32, opts ...options.FacadeOption) (ProcessDefinitions, error)
	GetProcessDefinitionsLatest(ctx context.Context) (ProcessDefinitions, error)
	GetProcessDefinitionByKey(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessDefinition, error)
	GetProcessDefinitionsByBpmnProcessId(ctx context.Context, bpmnProcessId string, opts ...options.FacadeOption) (ProcessDefinitions, error)
	GetProcessDefinitionByBpmnProcessIdLatest(ctx context.Context, bpmnProcessId string, opts ...options.FacadeOption) (ProcessDefinition, error)
	GetProcessDefinitionByBpmnProcessIdAndVersion(ctx context.Context, bpmnProcessId string, version int32, opts ...options.FacadeOption) (ProcessDefinition, error)

	GetProcessInstanceByKey(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstance, error)
	SearchProcessInstances(ctx context.Context, filter ProcessInstanceSearchFilterOpts, size int32, opts ...options.FacadeOption) (ProcessInstances, error)
	CancelProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (CancelResponse, error)
	GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstances, error)
	FilterProcessInstanceWithOrphanParent(ctx context.Context, items []ProcessInstance, opts ...options.FacadeOption) ([]ProcessInstance, error)
	DeleteProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ChangeStatus, error)
	WaitForProcessInstanceState(ctx context.Context, key string, desired States, opts ...options.FacadeOption) (State, error)
	Walker
}

type client struct {
	pdApi pdsvc.API
	piApi pisvc.API
}

func New(pdApi pdsvc.API, piApi pisvc.API) API {
	return &client{
		pdApi: pdApi,
		piApi: piApi,
	}
}

func (c *client) SearchProcessDefinitions(ctx context.Context, filter ProcessDefinitionSearchFilterOpts, size int32, opts ...options.FacadeOption) (ProcessDefinitions, error) {
	pds, err := c.pdApi.SearchProcessDefinitions(ctx, toDomainProcessDefinitionFilter(filter), size, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinitions{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessDefinitions(pds), nil
}

func (c *client) GetProcessDefinitionsLatest(ctx context.Context) (ProcessDefinitions, error) {
	pds, err := c.pdApi.GetProcessDefinitionsLatest(ctx)
	if err != nil {
		return ProcessDefinitions{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessDefinitions(pds), nil
}

func (c *client) GetProcessDefinitionByKey(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessDefinition, error) {
	pd, err := c.pdApi.GetProcessDefinitionByKey(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinition{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessDefinition(pd), nil
}

func (c *client) GetProcessDefinitionsByBpmnProcessId(ctx context.Context, bpmnProcessId string, opts ...options.FacadeOption) (ProcessDefinitions, error) {
	pds, err := c.pdApi.GetProcessDefinitionVersionsByBpmnProcessId(ctx, bpmnProcessId, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinitions{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessDefinitions(pds), nil
}

func (c *client) GetProcessDefinitionByBpmnProcessIdLatest(ctx context.Context, bpmnProcessId string, opts ...options.FacadeOption) (ProcessDefinition, error) {
	pd, err := c.pdApi.GetProcessDefinitionByBpmnProcessIdLatest(ctx, bpmnProcessId, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinition{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessDefinition(pd), nil
}

func (c *client) GetProcessDefinitionByBpmnProcessIdAndVersion(ctx context.Context, bpmnProcessId string, version int32, opts ...options.FacadeOption) (ProcessDefinition, error) {
	pd, err := c.pdApi.GetProcessDefinitionByBpmnProcessIdAndVersion(ctx, bpmnProcessId, version, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinition{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessDefinition(pd), nil
}

func (c *client) GetProcessInstanceByKey(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstance, error) {
	pi, err := c.piApi.GetProcessInstanceByKey(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstance{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessInstance(pi), nil
}

func (c *client) SearchProcessInstances(ctx context.Context, filter ProcessInstanceSearchFilterOpts, size int32, opts ...options.FacadeOption) (ProcessInstances, error) {
	pis, err := c.piApi.SearchForProcessInstances(ctx, toDomainProcessInstanceFilter(filter), size, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstances{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessInstances(pis), nil
}

func (c *client) CancelProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (CancelResponse, error) {
	resp, err := c.piApi.CancelProcessInstance(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return CancelResponse{}, ferrors.FromDomain(err)
	}
	return CancelResponse{StatusCode: resp.StatusCode, Status: resp.Status}, nil
}

func (c *client) GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstances, error) {
	children, err := c.piApi.GetDirectChildrenOfProcessInstance(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstances{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessInstances(children), nil
}

func (c *client) FilterProcessInstanceWithOrphanParent(ctx context.Context, items []ProcessInstance, opts ...options.FacadeOption) ([]ProcessInstance, error) {
	in := toolx.MapSlice(items, toDomainProcessInstance)
	out, err := c.piApi.FilterProcessInstanceWithOrphanParent(ctx, in, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return nil, ferrors.FromDomain(err)
	}
	return toolx.MapSlice(out, fromDomainProcessInstance), nil
}

func (c *client) DeleteProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ChangeStatus, error) {
	s, err := c.piApi.DeleteProcessInstance(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ChangeStatus{}, ferrors.FromDomain(err)
	}
	return ChangeStatus{Deleted: s.Deleted, Message: s.Message}, nil
}

func (c *client) WaitForProcessInstanceState(ctx context.Context, key string, desired States, opts ...options.FacadeOption) (State, error) {
	got, err := c.piApi.WaitForProcessInstanceState(ctx, key, toolx.MapSlice(desired, func(s State) d.State { return d.State(s) }), options.MapFacadeOptionsToCallOptions(opts)...)
	return State(got), ferrors.FromDomain(err)
}
