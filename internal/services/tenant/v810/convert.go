// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// fromTenantResult converts the generated v8.10 tenant payload into the shared domain model.
func fromTenantResult(x camundav810.TenantResult) d.Tenant {
	return d.Tenant{
		TenantId:    string(x.TenantId),
		Name:        x.Name,
		Description: toolx.Deref(x.Description, ""),
	}
}
