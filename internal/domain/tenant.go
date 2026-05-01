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

func FilterTenantsByNameContains(tenants []Tenant, text string) []Tenant {
	if text == "" {
		return slices.Clone(tenants)
	}

	out := make([]Tenant, 0, len(tenants))
	for _, tenant := range tenants {
		if strings.Contains(tenant.Name, text) {
			out = append(out, tenant)
		}
	}
	return out
}
