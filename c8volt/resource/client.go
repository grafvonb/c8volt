package resource

import (
	"context"
	"errors"
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
	err := c.api.Delete(ctx, key, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return DeleteReport{Key: key, Ok: false}, ferrors.FromDomain(err)
	}
	return DeleteReport{Key: key, Ok: true}, nil
}

func (c *client) DeleteProcessDefinitions(ctx context.Context, filter process.ProcessDefinitionFilter, opts ...foptions.FacadeOption) (DeleteReports, error) {
	pds, err := c.papi.SearchProcessDefinitions(ctx, filter, opts...)
	if err != nil {
		return DeleteReports{}, ferrors.FromDomain(err)
	}
	var errs []error
	var reps DeleteReports
	for _, pd := range pds.Items {
		r, err := c.DeleteProcessDefinition(ctx, pd.Key, opts...)
		if err != nil {
			errs = append(errs, err)
		}
		reps.Items = append(reps.Items, r)
	}
	return reps, errors.Join(errs...)
}
