// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	SearchTenants(ctx context.Context, size int32, opts ...services.CallOption) ([]d.Tenant, error)
	GetTenant(ctx context.Context, tenantID string, opts ...services.CallOption) (d.Tenant, error)
}

type GenTenantClient interface {
	SearchTenantsWithResponse(ctx context.Context, body camundav88.SearchTenantsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchTenantsResponse, error)
	GetTenantWithResponse(ctx context.Context, tenantId camundav88.TenantId, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetTenantResponse, error)
	GetAuthenticationWithResponse(ctx context.Context, reqEditors ...camundav88.RequestEditorFn) (*camundav88.GetAuthenticationResponse, error)
}

var _ API = (*Service)(nil)
var _ GenTenantClient = (*camundav88.ClientWithResponses)(nil)
