// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	SearchTenants(ctx context.Context, filter d.TenantFilter, size int32, opts ...services.CallOption) ([]d.Tenant, error)
	GetTenant(ctx context.Context, tenantID string, opts ...services.CallOption) (d.Tenant, error)
}

type GenTenantClient interface {
	SearchTenantsWithResponse(ctx context.Context, body camundav810.SearchTenantsJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchTenantsResponse, error)
	GetTenantWithResponse(ctx context.Context, tenantId camundav810.TenantId, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetTenantResponse, error)
}

var _ API = (*Service)(nil)
var _ GenTenantClient = (*camundav810.ClientWithResponses)(nil)
