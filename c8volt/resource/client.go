package resource

import (
	"context"
	"errors"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/options"
	"github.com/grafvonb/c8volt/c8volt/process"
	rsvc "github.com/grafvonb/c8volt/internal/services/resource"
	"github.com/grafvonb/c8volt/toolx"
)

type API interface {
	DeployProcessDefinition(ctx context.Context, tenantId string, units []DeploymentUnitData, opts ...options.FacadeOption) (ProcessDefinitionDeployment, error)
	DeleteProcessDefinitionByKey(ctx context.Context, key string, opts ...options.FacadeOption) error
	DeleteProcessDefinitionVersionsByBpmnProcessId(ctx context.Context, bpmnProcessId string, opts ...options.FacadeOption) error
}

type client struct {
	api  rsvc.API
	papi process.API
	log  *slog.Logger
}

func New(api rsvc.API, papi process.API, log *slog.Logger) API {
	return &client{api: api, papi: papi, log: log}
}

func (c *client) DeployProcessDefinition(ctx context.Context, tenantId string, units []DeploymentUnitData, opts ...options.FacadeOption) (ProcessDefinitionDeployment, error) {
	pdd, err := c.api.Deploy(ctx, tenantId, toDeploymentUnitDatas(units), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ProcessDefinitionDeployment{}, ferrors.FromDomain(err)
	}
	c.log.Debug(toolx.ToJSONString(pdd))
	return fromProcessDefinitionDeployment(pdd), nil
}

func (c *client) DeleteProcessDefinitionByKey(ctx context.Context, key string, opts ...options.FacadeOption) error {
	err := c.api.Delete(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return ferrors.FromDomain(err)
	}
	return nil
}

func (c *client) DeleteProcessDefinitionVersionsByBpmnProcessId(ctx context.Context, bpmnProcessId string, opts ...options.FacadeOption) error {
	pds, err := c.papi.GetProcessDefinitionsByBpmnProcessId(ctx, bpmnProcessId, opts...)
	if err != nil {
		return ferrors.FromDomain(err)
	}
	var errs []error
	for _, pd := range pds.Items {
		err = c.DeleteProcessDefinitionByKey(ctx, pd.Key, opts...)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
