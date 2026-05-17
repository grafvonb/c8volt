// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package resource

import (
	"context"
	"log/slog"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	rsvc "github.com/grafvonb/c8volt/internal/services/resource"
	types "github.com/grafvonb/c8volt/typex"
)

type client struct {
	api   rsvc.API
	pdApi pdsvc.API
	piApi pisvc.API
	log   *slog.Logger
}

func New(api rsvc.API, pdApi pdsvc.API, piApi pisvc.API, log *slog.Logger) API {
	return &client{api: api, pdApi: pdApi, piApi: piApi, log: log}
}

func (c *client) GetResource(ctx context.Context, key string, opts ...options.FacadeOption) (Resource, error) {
	resource, err := c.api.Get(ctx, key, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Resource{}, ferr.FromDomain(err)
	}
	return fromResource(resource), nil
}

func (c *client) DeployProcessDefinition(ctx context.Context, units []DeploymentUnitData, opts ...options.FacadeOption) ([]ProcessDefinitionDeployment, error) {
	pdd, err := c.api.Deploy(ctx, toDeploymentUnitDatas(units), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return nil, ferr.FromDomain(err)
	}
	return fromProcessDefinitionDeployment(pdd), nil
}

func (c *client) DeleteProcessDefinition(ctx context.Context, key string, opts ...options.FacadeOption) (DeleteReport, error) {
	resp, err := pdsvc.DeleteProcessDefinition(ctx, c.api, c.pdApi, c.piApi, c.log, key, options.MapFacadeOptionsToCallOptions(opts)...)
	out := fromResourceDeleteResponse(key, resp, resp.Ok && err == nil)
	if err != nil {
		return out, ferr.FromDomain(err)
	}
	return out, nil
}

func (c *client) PreviewDeleteProcessDefinitions(ctx context.Context, keys types.Keys, opts ...options.FacadeOption) (DeleteProcessDefinitionPlan, error) {
	plan, err := pdsvc.PreviewDeleteProcessDefinitions(ctx, c.pdApi, c.piApi, c.log, keys, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return DeleteProcessDefinitionPlan{}, ferr.FromDomain(err)
	}
	return fromDomainDeleteProcessDefinitionPlan(plan), nil
}
