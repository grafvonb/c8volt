package process

import (
	"context"
	"log/slog"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
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

func (c *client) SearchProcessDefinitions(ctx context.Context, filter ProcessDefinitionFilter, opts ...options.FacadeOption) (ProcessDefinitions, error) {
	pds, err := c.pdApi.SearchProcessDefinitions(ctx, toDomainProcessDefinitionFilter(filter), pdsvc.MaxResultSize, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinitions{}, ferr.FromDomain(err)
	}
	return fromDomainProcessDefinitions(pds), nil
}

func (c *client) SearchProcessDefinitionsLatest(ctx context.Context, filter ProcessDefinitionFilter, opts ...options.FacadeOption) (ProcessDefinitions, error) {
	pds, err := c.pdApi.SearchProcessDefinitionsLatest(ctx, toDomainProcessDefinitionFilter(filter), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinitions{}, ferr.FromDomain(err)
	}
	return fromDomainProcessDefinitions(pds), nil
}

func (c *client) GetProcessDefinition(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessDefinition, error) {
	pd, err := c.pdApi.GetProcessDefinition(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinition{}, ferr.FromDomain(err)
	}
	return fromDomainProcessDefinition(pd), nil
}

func (c *client) GetProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstance, error) {
	pi, err := c.piApi.GetProcessInstance(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstance{}, ferr.FromDomain(err)
	}
	return fromDomainProcessInstance(pi), nil
}

func (c *client) CreateProcessInstance(ctx context.Context, data ProcessInstanceData, opts ...options.FacadeOption) (ProcessInstance, error) {
	pic, err := c.piApi.CreateProcessInstance(ctx, toProcessInstanceData(data), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstance{}, ferr.FromDomain(err)
	}
	return fromDomainProcessInstanceCreation(pic), nil
}

func (c *client) CreateProcessInstances(ctx context.Context, datas []ProcessInstanceData, opts ...options.FacadeOption) ([]ProcessInstance, error) {
	pis := make([]ProcessInstance, 0, len(datas))
	for _, data := range datas {
		pic, err := c.piApi.CreateProcessInstance(ctx, toProcessInstanceData(data), options.MapFacadeOptionsToCallOptions(opts)...)
		if err != nil {
			return nil, ferr.FromDomain(err)
		}
		pis = append(pis, fromDomainProcessInstanceCreation(pic))
	}
	return pis, nil
}

func (c *client) SearchProcessInstances(ctx context.Context, filter ProcessInstanceFilter, size int32, opts ...options.FacadeOption) (ProcessInstances, error) {
	pis, err := c.piApi.SearchForProcessInstances(ctx, toDomainProcessInstanceFilter(filter), size, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstances{}, ferr.FromDomain(err)
	}
	return fromDomainProcessInstances(pis), nil
}

func (c *client) CancelProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (CancelReport, ProcessInstances, error) {
	resp, pis, err := c.piApi.CancelProcessInstance(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return CancelReport{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, ProcessInstances{}, ferr.FromDomain(err)
	}
	return CancelReport{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, fromDomainProcessInstances(pis), nil
}

func (c *client) DeleteProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (DeleteReport, error) {
	resp, err := c.piApi.DeleteProcessInstance(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return DeleteReport{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, ferr.FromDomain(err)
	}
	return DeleteReport{Key: key, Ok: resp.Ok, StatusCode: resp.StatusCode, Status: resp.Status}, nil
}

func (c *client) GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstances, error) {
	children, err := c.piApi.GetDirectChildrenOfProcessInstance(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessInstances{}, ferr.FromDomain(err)
	}
	return fromDomainProcessInstances(children), nil
}

func (c *client) FilterProcessInstanceWithOrphanParent(ctx context.Context, items []ProcessInstance, opts ...options.FacadeOption) ([]ProcessInstance, error) {
	in := toolx.MapSlice(items, toDomainProcessInstance)
	out, err := c.piApi.FilterProcessInstanceWithOrphanParent(ctx, in, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return nil, ferr.FromDomain(err)
	}
	return toolx.MapSlice(out, fromDomainProcessInstance), nil
}

func (c *client) WaitForProcessInstanceState(ctx context.Context, key string, desired States, opts ...options.FacadeOption) (StateReport, ProcessInstance, error) {
	got, pi, err := c.piApi.WaitForProcessInstanceState(ctx, key, toolx.MapSlice(desired, func(s State) d.State { return d.State(s) }), options.MapFacadeOptionsToCallOptions(opts)...)
	pgot, _ := ParseState(got.State.String())
	if err != nil {
		return StateReport{State: pgot, Status: got.Status}, ProcessInstance{}, ferr.FromDomain(err)
	}
	return StateReport{State: pgot, Status: got.Status, Key: pi.Key}, fromDomainProcessInstance(pi), nil
}

func MapStateResponseToReport(in d.StateResponse) StateReport {
	return StateReport{
		State:  State(in.State.String()),
		Status: in.Status,
	}
}

func MapStateResponsesToReports(in d.StateResponses) StateReports {
	out := StateReports{
		Items: make([]StateReport, len(in.Items)),
	}
	for i, r := range in.Items {
		out.Items[i] = MapStateResponseToReport(r)
	}
	return out
}
