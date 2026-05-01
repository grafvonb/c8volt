// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package tenant

type Tenant struct {
	TenantId    string `json:"tenantId,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type Tenants struct {
	Total int32    `json:"total,omitempty"`
	Items []Tenant `json:"items,omitempty"`
}
