// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package tenant

import (
	"context"

	"github.com/grafvonb/c8volt/c8volt/foptions"
)

type API interface {
	SearchTenants(ctx context.Context, filter TenantFilter, opts ...foptions.FacadeOption) (Tenants, error)
	GetTenant(ctx context.Context, tenantID string, opts ...foptions.FacadeOption) (Tenant, error)
}

var _ API = (*client)(nil)
