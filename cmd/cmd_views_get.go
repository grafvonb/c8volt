// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/spf13/cobra"
)

// listTenantsView renders tenant discovery output through the shared list, keys-only, and JSON modes.
func listTenantsView(cmd *cobra.Command, resp tenant.Tenants) error {
	return listOrJSONFlat(cmd, resp, resp.Items, pickMode(), flatRowTenant, func(it tenant.Tenant) string { return it.TenantId })
}

// tenantView renders a single tenant through the same mode contract as other get commands.
func tenantView(cmd *cobra.Command, item tenant.Tenant) error {
	return itemView(cmd, item, pickMode(), oneLineTenant, func(it tenant.Tenant) string { return it.TenantId })
}

// oneLineTenant formats compact tenant rows.
func oneLineTenant(it tenant.Tenant) string {
	return compactFlatRow(flatRowTenant(it))
}

// flatRowTenant omits the description column when absent so sparse tenant lists stay compact.
func flatRowTenant(it tenant.Tenant) flatRow {
	if it.Description == "" {
		return flatRow{it.TenantId, it.Name}
	}
	return flatRow{it.TenantId, it.Name, it.Description}
}
