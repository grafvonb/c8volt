// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package tenant

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	tsvc "github.com/grafvonb/c8volt/internal/services/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubTenantAPI provides tenant service behavior directly to the facade tests.
type stubTenantAPI struct {
	searchTenants func(context.Context, d.TenantFilter, int32, ...services.CallOption) ([]d.Tenant, error)
	getTenant     func(context.Context, string, ...services.CallOption) (d.Tenant, error)
}

// SearchTenants delegates to the configured test search function.
func (s stubTenantAPI) SearchTenants(ctx context.Context, filter d.TenantFilter, size int32, opts ...services.CallOption) ([]d.Tenant, error) {
	return s.searchTenants(ctx, filter, size, opts...)
}

// GetTenant delegates to the configured test keyed lookup function.
func (s stubTenantAPI) GetTenant(ctx context.Context, tenantID string, opts ...services.CallOption) (d.Tenant, error) {
	return s.getTenant(ctx, tenantID, opts...)
}

// Verifies the tenant facade forwards filters, page size, and options while preserving service order.
func TestClient_SearchTenants_ConvertsAndForwardsFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := stubTenantAPI{
		searchTenants: func(_ context.Context, filter d.TenantFilter, size int32, opts ...services.CallOption) ([]d.Tenant, error) {
			assert.Equal(t, d.TenantFilter{NameContains: "demo"}, filter)
			assert.Equal(t, tsvc.MaxResultSize, size)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return []d.Tenant{
				{TenantId: "tenant-c", Name: "Beta", Description: "second"},
				{TenantId: "tenant-b", Name: "Alpha", Description: "duplicate name"},
				{TenantId: "tenant-a", Name: "Alpha"},
			}, nil
		},
	}

	cli := New(api, slog.Default())
	got, err := cli.SearchTenants(ctx, TenantFilter{NameContains: "demo"}, foptions.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, int32(3), got.Total)
	require.Len(t, got.Items, 3)
	assert.Equal(t, "tenant-c", got.Items[0].TenantId)
	assert.Equal(t, "tenant-b", got.Items[1].TenantId)
	assert.Equal(t, "tenant-a", got.Items[2].TenantId)
	assert.Equal(t, "duplicate name", got.Items[1].Description)
}

// Verifies keyed tenant facade lookup converts the domain result into the public model.
func TestClient_GetTenant_ConvertsDomainResult(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := stubTenantAPI{
		getTenant: func(_ context.Context, tenantID string, opts ...services.CallOption) (d.Tenant, error) {
			assert.Equal(t, "tenant-a", tenantID)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.Tenant{TenantId: "tenant-a", Name: "Alpha", Description: "primary tenant"}, nil
		},
	}

	cli := New(api, slog.Default())
	got, err := cli.GetTenant(ctx, "tenant-a", foptions.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, Tenant{TenantId: "tenant-a", Name: "Alpha", Description: "primary tenant"}, got)
}

// Verifies tenant JSON payloads expose only discovery fields and omit sensitive membership data.
func TestClient_TenantJSONExposesOnlyPublicDiscoveryFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := stubTenantAPI{
		searchTenants: func(context.Context, d.TenantFilter, int32, ...services.CallOption) ([]d.Tenant, error) {
			return []d.Tenant{
				{TenantId: "tenant-a", Name: "Alpha", Description: "primary tenant"},
			}, nil
		},
		getTenant: func(context.Context, string, ...services.CallOption) (d.Tenant, error) {
			return d.Tenant{TenantId: "tenant-a", Name: "Alpha", Description: "primary tenant"}, nil
		},
	}

	cli := New(api, slog.Default())
	list, err := cli.SearchTenants(ctx, TenantFilter{})
	require.NoError(t, err)
	keyed, err := cli.GetTenant(ctx, "tenant-a")
	require.NoError(t, err)

	assertJSONKeys(t, list.Items[0], "tenantId", "name", "description")
	assertJSONKeys(t, keyed, "tenantId", "name", "description")
	assertJSONKeys(t, list, "total", "items")
}

// Verifies keyed tenant not-found errors are mapped into the facade not-found class.
func TestClient_GetTenant_MapsNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := stubTenantAPI{
		getTenant: func(_ context.Context, tenantID string, opts ...services.CallOption) (d.Tenant, error) {
			assert.Equal(t, "missing", tenantID)
			return d.Tenant{}, fmtWrappedDomainError(d.ErrNotFound)
		},
	}

	cli := New(api, slog.Default())
	_, err := cli.GetTenant(ctx, "missing")

	require.Error(t, err)
	assert.ErrorIs(t, err, ferrors.ErrNotFound)
}

// Verifies tenant facade methods map domain error classes consistently across list and keyed modes.
func TestClient_MapsDomainErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := stubTenantAPI{
		searchTenants: func(context.Context, d.TenantFilter, int32, ...services.CallOption) ([]d.Tenant, error) {
			return nil, fmtWrappedDomainError(d.ErrUnsupported)
		},
		getTenant: func(context.Context, string, ...services.CallOption) (d.Tenant, error) {
			return d.Tenant{}, fmtWrappedDomainError(d.ErrNotFound)
		},
	}

	cli := New(api, slog.Default())

	_, err := cli.SearchTenants(ctx, TenantFilter{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ferrors.ErrUnsupported)

	_, err = cli.GetTenant(ctx, "missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, ferrors.ErrNotFound)
}

// fmtWrappedDomainError keeps test errors wrapped enough to prove error-class mapping uses errors.Is.
func fmtWrappedDomainError(err error) error {
	return errors.Join(err)
}

// assertJSONKeys verifies exact public JSON keys while guarding against sensitive tenant fields.
func assertJSONKeys(t *testing.T, value any, want ...string) {
	t.Helper()

	raw, err := json.Marshal(value)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))

	require.Len(t, got, len(want))
	for _, key := range want {
		assert.Contains(t, got, key)
	}
	assert.NotContains(t, got, "secret")
	assert.NotContains(t, got, "credentials")
	assert.NotContains(t, got, "authorization")
	assert.NotContains(t, got, "members")
	assert.NotContains(t, got, "roles")
}
