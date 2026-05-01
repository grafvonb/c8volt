// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package tenant

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v87 "github.com/grafvonb/c8volt/internal/services/tenant/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/tenant/v88"
	v89 "github.com/grafvonb/c8volt/internal/services/tenant/v89"
)

var MaxResultSize int32 = 1000

type API interface {
	SearchTenants(ctx context.Context, filter d.TenantFilter, size int32, opts ...services.CallOption) ([]d.Tenant, error)
	GetTenant(ctx context.Context, tenantID string, opts ...services.CallOption) (d.Tenant, error)
}

var _ API = (*v87.Service)(nil)
var _ API = (*v88.Service)(nil)
var _ API = (*v89.Service)(nil)
var _ API = (v87.API)(nil)
var _ API = (v88.API)(nil)
var _ API = (v89.API)(nil)
