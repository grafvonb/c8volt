// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type API interface {
	SearchTenants(ctx context.Context, size int32, opts ...services.CallOption) ([]d.Tenant, error)
	GetTenant(ctx context.Context, tenantID string, opts ...services.CallOption) (d.Tenant, error)
}

var _ API = (*Service)(nil)
