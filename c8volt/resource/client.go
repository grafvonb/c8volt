package resource

import (
	"context"
	"fmt"
	"log/slog"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
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

func (c *client) DeployProcessDefinition(ctx context.Context, tenantId string, units []DeploymentUnitData, opts ...options.FacadeOption) ([]ProcessDefinitionDeployment, error) {
	pdd, err := c.api.Deploy(ctx, tenantId, toDeploymentUnitDatas(units), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return nil, ferr.FromDomain(err)
	}
	return fromProcessDefinitionDeployment(pdd), nil
}

func (c *client) DeleteProcessDefinition(ctx context.Context, key string, opts ...options.FacadeOption) (DeleteReport, error) {
	cCfg := options.ApplyFacadeOptions(opts)
	if !cCfg.NoStateCheck {
		filter := process.ProcessInstanceFilter{ProcessDefinitionKey: key, State: process.StateActive}
		pis, err := c.papi.SearchProcessInstances(ctx, filter, 1, opts...)
		if err != nil {
			return DeleteReport{Key: key, Ok: false}, ferr.FromDomain(err)
		}
		if len(pis.Items) > 0 {
			if cCfg.Force {
				c.log.Info(fmt.Sprintf("cancelling active process instance(s) for process definition %s before deletion", key))
				pis, err = c.papi.SearchProcessInstances(ctx, filter, consts.MaxPISearchSize, opts...)
				if err != nil {
					return DeleteReport{Key: key, Ok: false}, ferr.FromDomain(err)
				}
				var keys []string
				for _, pi := range pis.Items {
					keys = append(keys, pi.Key)
				}
				_, err := c.papi.CancelProcessInstances(ctx, keys, len(keys), opts...)
				if err != nil {
					return DeleteReport{Key: key, Ok: false}, fmt.Errorf("cancelling active process instances for process definition %s before deletion failed: %w", key, err)
				}
			} else {
				return DeleteReport{Key: key, Ok: false}, fmt.Errorf("cannot delete process definition %s with active process instances; user --force to cancel them automatically", key)
			}
		}
	}
	err := c.api.Delete(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return DeleteReport{Key: key, Ok: false}, ferr.FromDomain(err)
	}
	return DeleteReport{Key: key, Ok: true}, nil
}
