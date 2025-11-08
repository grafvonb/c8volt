package resource

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	rsvc "github.com/grafvonb/c8volt/internal/services/resource"
)

type client struct {
	api  rsvc.API
	papi process.API
	log  *slog.Logger
}

func New(api rsvc.API, papi process.API, log *slog.Logger) API {
	return &client{api: api, papi: papi, log: log}
}

func (c *client) DeployProcessDefinition(ctx context.Context, tenantId string, units []DeploymentUnitData, opts ...foptions.FacadeOption) ([]ProcessDefinitionDeployment, error) {
	pdd, err := c.api.Deploy(ctx, tenantId, toDeploymentUnitDatas(units), foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return nil, ferrors.FromDomain(err)
	}
	return fromProcessDefinitionDeployment(pdd), nil
}

func (c *client) DeleteProcessDefinition(ctx context.Context, key string, opts ...foptions.FacadeOption) (DeleteReport, error) {
	cCfg := foptions.ApplyFacadeOptions(opts)
	if !cCfg.NoStateCheck {
		filter := process.ProcessInstanceFilter{ProcessDefinitionKey: key, State: process.StateActive}
		pis, err := c.papi.SearchProcessInstances(ctx, filter, 1, opts...)
		if err != nil {
			return DeleteReport{Key: key, Ok: false}, ferrors.FromDomain(err)
		}
		if len(pis.Items) > 0 {
			return DeleteReport{Key: key, Ok: false}, fmt.Errorf("cannot delete process definition %s with active process instances; cancel or terminate them first", key)
		}
	}
	err := c.api.Delete(ctx, key, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return DeleteReport{Key: key, Ok: false}, ferrors.FromDomain(err)
	}
	return DeleteReport{Key: key, Ok: true}, nil
}
