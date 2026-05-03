// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package tenant

import (
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// fromDomainTenant converts the internal tenant model to the public facade payload.
func fromDomainTenant(x d.Tenant) Tenant {
	return Tenant{
		TenantId:    x.TenantId,
		Name:        x.Name,
		Description: x.Description,
	}
}

// fromDomainTenants wraps converted tenant items with the total used by command and JSON views.
func fromDomainTenants(xs []d.Tenant) Tenants {
	items := toolx.MapSlice(xs, fromDomainTenant)
	return Tenants{
		Total: int32(len(items)),
		Items: items,
	}
}

// toDomainTenantFilter converts facade filter input into the shared tenant service filter.
func toDomainTenantFilter(x TenantFilter) d.TenantFilter {
	return d.TenantFilter{NameContains: x.NameContains}
}
