// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// fromTenantResult converts the generated v8.9 tenant payload into the shared domain model.
func fromTenantResult(x camundav89.TenantResult) d.Tenant {
	return d.Tenant{
		TenantId:    string(x.TenantId),
		Name:        x.Name,
		Description: toolx.Deref(x.Description, ""),
	}
}
