// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package tenant

import (
	"context"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	tsvc "github.com/grafvonb/c8volt/internal/services/tenant"
)

type client struct {
	api tsvc.API
	log *slog.Logger
}

func New(api tsvc.API, log *slog.Logger) API {
	return &client{api: api, log: log}
}

func (c *client) SearchTenants(ctx context.Context, opts ...foptions.FacadeOption) (Tenants, error) {
	tenants, err := c.api.SearchTenants(ctx, tsvc.MaxResultSize, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Tenants{}, ferrors.FromDomain(err)
	}
	d.SortTenantsByNameAscThenTenantIDAsc(tenants)
	return fromDomainTenants(tenants), nil
}

func (c *client) GetTenant(ctx context.Context, tenantID string, opts ...foptions.FacadeOption) (Tenant, error) {
	tenant, err := c.api.GetTenant(ctx, tenantID, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Tenant{}, ferrors.FromDomain(err)
	}
	return fromDomainTenant(tenant), nil
}
