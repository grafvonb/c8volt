// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package tenant

import (
	"context"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	tsvc "github.com/grafvonb/c8volt/internal/services/tenant"
)

type client struct {
	api tsvc.API
	log *slog.Logger
}

// New creates the tenant facade backed by the version-specific tenant service API.
func New(api tsvc.API, log *slog.Logger) API {
	return &client{api: api, log: log}
}

// SearchTenants converts facade filters and options into service calls and returns public tenant models.
func (c *client) SearchTenants(ctx context.Context, filter TenantFilter, opts ...foptions.FacadeOption) (Tenants, error) {
	tenants, err := c.api.SearchTenants(ctx, toDomainTenantFilter(filter), tsvc.MaxResultSize, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Tenants{}, ferrors.FromDomain(err)
	}
	return fromDomainTenants(tenants), nil
}

// GetTenant fetches one tenant by ID and maps service errors into facade error classes.
func (c *client) GetTenant(ctx context.Context, tenantID string, opts ...foptions.FacadeOption) (Tenant, error) {
	tenant, err := c.api.GetTenant(ctx, tenantID, foptions.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return Tenant{}, ferrors.FromDomain(err)
	}
	return fromDomainTenant(tenant), nil
}
