// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromTenantResult(x camundav88.TenantResult) d.Tenant {
	return d.Tenant{
		TenantId:    string(x.TenantId),
		Name:        x.Name,
		Description: toolx.Deref(x.Description, ""),
	}
}
