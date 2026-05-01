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

func SortTenantsByNameAscThenTenantIDAsc(tenants []Tenant) {
	slices.SortFunc(tenants, func(a, b Tenant) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		if a.TenantId < b.TenantId {
			return -1
		}
		if a.TenantId > b.TenantId {
			return 1
		}
		return 0
	})
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
