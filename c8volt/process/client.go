package process

import (
	"context"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/toolx"
)

type client struct {
	pdApi pdsvc.API
	piApi pisvc.API
	log   *slog.Logger
}

func New(pdApi pdsvc.API, piApi pisvc.API, log *slog.Logger) API {
	return &client{
		pdApi: pdApi,
		piApi: piApi,
		log:   log,
	}
}

func (c *client) SearchProcessDefinitions(ctx context.Context, filter ProcessDefinitionFilter, opts ...foptions.FacadeOption) (ProcessDefinitions, error) {
	pds, err := c.pdApi.SearchProcessDefinitions(ctx, toDomainProcessDefinitionFilter(filter), pdsvc.MaxResultSize, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinitions{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessDefinitions(pds), nil
}

func (c *client) SearchProcessDefinitionsLatest(ctx context.Context, filter ProcessDefinitionFilter, opts ...foptions.FacadeOption) (ProcessDefinitions, error) {
	pds, err := c.pdApi.SearchProcessDefinitionsLatest(ctx, toDomainProcessDefinitionFilter(filter), foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinitions{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessDefinitions(pds), nil
}

func (c *client) GetProcessDefinition(ctx context.Context, key string, opts ...foptions.FacadeOption) (ProcessDefinition, error) {
	pd, err := c.pdApi.GetProcessDefinition(ctx, key, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinition{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessDefinition(pd), nil
}

func (c *client) GetProcessInstance(ctx context.Context, key string, opts ...foptions.FacadeOption) (ProcessInstance, error) {
	pi, err := c.piApi.GetProcessInstance(ctx, key, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstance{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessInstance(pi), nil
}

func (c *client) CreateProcessInstance(ctx context.Context, data ProcessInstanceData, opts ...foptions.FacadeOption) (ProcessInstance, error) {
	pic, err := c.piApi.CreateProcessInstance(ctx, toProcessInstanceData(data), foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstance{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessInstanceCreation(pic), nil
}

func (c *client) CreateProcessInstances(ctx context.Context, datas []ProcessInstanceData, opts ...foptions.FacadeOption) ([]ProcessInstance, error) {
	pis := make([]ProcessInstance, 0, len(datas))
	for _, data := range datas {
		pic, err := c.piApi.CreateProcessInstance(ctx, toProcessInstanceData(data), foptions.MapFacadeOptionsToCallOptions(opts)...)
		if err != nil {
			return nil, ferrors.FromDomain(err)
		}
		pis = append(pis, fromDomainProcessInstanceCreation(pic))
	}
	return pis, nil
}

func (c *client) SearchProcessInstances(ctx context.Context, filter ProcessInstanceFilter, size int32, opts ...foptions.FacadeOption) (ProcessInstances, error) {
	pis, err := c.piApi.SearchForProcessInstances(ctx, toDomainProcessInstanceFilter(filter), size, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstances{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessInstances(pis), nil
}

func (c *client) CancelProcessInstance(ctx context.Context, key string, opts ...foptions.FacadeOption) (CancelReport, error) {
	resp, err := c.piApi.CancelProcessInstance(ctx, key, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return CancelReport{Key: key, Ok: false, StatusCode: resp.StatusCode, Status: resp.Status}, ferrors.FromDomain(err)
	}
	return CancelReport{Key: key, Ok: true, StatusCode: resp.StatusCode, Status: resp.Status}, nil
}

func (c *client) DeleteProcessInstance(ctx context.Context, key string, opts ...foptions.FacadeOption) (DeleteReport, error) {
	resp, err := c.piApi.DeleteProcessInstance(ctx, key, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return DeleteReport{Key: key, Ok: false, StatusCode: resp.StatusCode, Status: resp.Status}, ferrors.FromDomain(err)
	}
	return DeleteReport{Key: key, Ok: true, StatusCode: resp.StatusCode, Status: resp.Status}, nil
}

func (c *client) GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...foptions.FacadeOption) (ProcessInstances, error) {
	children, err := c.piApi.GetDirectChildrenOfProcessInstance(ctx, key, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstances{}, ferrors.FromDomain(err)
	}
	return fromDomainProcessInstances(children), nil
}

func (c *client) FilterProcessInstanceWithOrphanParent(ctx context.Context, items []ProcessInstance, opts ...foptions.FacadeOption) ([]ProcessInstance, error) {
	in := toolx.MapSlice(items, toDomainProcessInstance)
	out, err := c.piApi.FilterProcessInstanceWithOrphanParent(ctx, in, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return nil, ferrors.FromDomain(err)
	}
	return toolx.MapSlice(out, fromDomainProcessInstance), nil
}

func (c *client) WaitForProcessInstanceState(ctx context.Context, key string, desired States, opts ...foptions.FacadeOption) (State, error) {
	got, _, err := c.piApi.WaitForProcessInstanceState(ctx, key, toolx.MapSlice(desired, func(s State) d.State { return d.State(s) }), foptions.MapFacadeOptionsToCallOptions(opts)...)
	return State(got), ferrors.FromDomain(err)
}
