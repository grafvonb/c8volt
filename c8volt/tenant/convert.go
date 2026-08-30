// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package tenant

import (
	"slices"

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

// fromDomainTenantContext converts immutable domain tenant context into the public API model.
func fromDomainTenantContext(x d.TenantContext) Context {
	return Context{
		Mode:               ContextMode(x.Mode),
		Filter:             ContextFilter(x.Filter),
		ConfiguredTenantID: x.ConfiguredTenantID,
		TargetTenantID:     x.TargetTenantID,
		ResolvedTenantIDs:  slices.Clone(x.ResolvedTenantIDs),
		UnknownTargetCount: x.UnknownTargetCount,
		CrossTenant:        x.CrossTenant,
		Warnings: toolx.MapSlice(x.Warnings, func(w d.TenantContextWarning) ContextWarning {
			return ContextWarning{
				Code:    ContextWarningCode(w.Code),
				Message: w.Message,
			}
		}),
	}
}

// toDomainTenantContext validates a public tenant context before crossing into internal services.
func toDomainTenantContext(x Context) (d.TenantContext, error) {
	return d.NewTenantContext(d.TenantContextMode(x.Mode), d.TenantContextInput{
		Filter:             d.TenantContextFilter(x.Filter),
		ConfiguredTenantID: x.ConfiguredTenantID,
		TargetTenantID:     x.TargetTenantID,
		ResolvedTenantIDs:  slices.Clone(x.ResolvedTenantIDs),
		UnknownTargetCount: x.UnknownTargetCount,
	})
}
