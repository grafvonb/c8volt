// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	SearchTenants(ctx context.Context, size int32, opts ...services.CallOption) ([]d.Tenant, error)
	GetTenant(ctx context.Context, tenantID string, opts ...services.CallOption) (d.Tenant, error)
}

type GenTenantClient interface {
	SearchTenantsWithResponse(ctx context.Context, body camundav89.SearchTenantsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchTenantsResponse, error)
	GetTenantWithResponse(ctx context.Context, tenantId camundav89.TenantId, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetTenantResponse, error)
}

var _ API = (*Service)(nil)
var _ GenTenantClient = (*camundav89.ClientWithResponses)(nil)
