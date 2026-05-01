// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package tenant

import (
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromDomainTenant(x d.Tenant) Tenant {
	return Tenant{
		TenantId:    x.TenantId,
		Name:        x.Name,
		Description: x.Description,
	}
}

func fromDomainTenants(xs []d.Tenant) Tenants {
	items := toolx.MapSlice(xs, fromDomainTenant)
	return Tenants{
		Total: int32(len(items)),
		Items: items,
	}
}
