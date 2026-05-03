// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"slices"
	"strings"
)

type Tenant struct {
	TenantId    string `json:"tenantId,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type TenantFilter struct {
	NameContains string
}

// FilterTenantsByNameOrIDContains applies the CLI literal contains filter across tenant name and ID.
func FilterTenantsByNameOrIDContains(tenants []Tenant, text string) []Tenant {
	if text == "" {
		return slices.Clone(tenants)
	}

	needle := strings.ToLower(text)
	out := make([]Tenant, 0, len(tenants))
	for _, tenant := range tenants {
		name := strings.ToLower(tenant.Name)
		tenantID := strings.ToLower(tenant.TenantId)
		if strings.Contains(name, needle) || strings.Contains(tenantID, needle) {
			out = append(out, tenant)
		}
	}
	return out
}
