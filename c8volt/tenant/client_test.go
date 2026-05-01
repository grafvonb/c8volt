// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package tenant

import (
	"context"
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

type stubTenantAPI struct {
	searchTenants func(context.Context, int32, ...services.CallOption) ([]d.Tenant, error)
	getTenant     func(context.Context, string, ...services.CallOption) (d.Tenant, error)
}

func (s stubTenantAPI) SearchTenants(ctx context.Context, size int32, opts ...services.CallOption) ([]d.Tenant, error) {
	return s.searchTenants(ctx, size, opts...)
}

func (s stubTenantAPI) GetTenant(ctx context.Context, tenantID string, opts ...services.CallOption) (d.Tenant, error) {
	return s.getTenant(ctx, tenantID, opts...)
}

func TestClient_SearchTenants_ConvertsAndSortsDomainResults(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := stubTenantAPI{
		searchTenants: func(_ context.Context, size int32, opts ...services.CallOption) ([]d.Tenant, error) {
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
	got, err := cli.SearchTenants(ctx, TenantFilter{}, foptions.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, int32(3), got.Total)
	require.Len(t, got.Items, 3)
	assert.Equal(t, "tenant-a", got.Items[0].TenantId)
	assert.Equal(t, "tenant-b", got.Items[1].TenantId)
	assert.Equal(t, "tenant-c", got.Items[2].TenantId)
	assert.Equal(t, "duplicate name", got.Items[1].Description)
}

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

func TestClient_SearchTenants_FiltersByLiteralNameContainsBeforeSorting(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := stubTenantAPI{
		searchTenants: func(_ context.Context, size int32, opts ...services.CallOption) ([]d.Tenant, error) {
			assert.Equal(t, tsvc.MaxResultSize, size)
			return []d.Tenant{
				{TenantId: "tenant-d", Name: "demo.*"},
				{TenantId: "tenant-c", Name: "z demo"},
				{TenantId: "tenant-b", Name: "a demo"},
				{TenantId: "tenant-a", Name: "alpha"},
			}, nil
		},
	}

	cli := New(api, slog.Default())

	matching, err := cli.SearchTenants(ctx, TenantFilter{NameContains: "demo"})
	require.NoError(t, err)
	require.Len(t, matching.Items, 3)
	assert.Equal(t, int32(3), matching.Total)
	assert.Equal(t, "tenant-b", matching.Items[0].TenantId)
	assert.Equal(t, "tenant-d", matching.Items[1].TenantId)
	assert.Equal(t, "tenant-c", matching.Items[2].TenantId)

	literal, err := cli.SearchTenants(ctx, TenantFilter{NameContains: ".*"})
	require.NoError(t, err)
	require.Len(t, literal.Items, 1)
	assert.Equal(t, int32(1), literal.Total)
	assert.Equal(t, "tenant-d", literal.Items[0].TenantId)

	empty, err := cli.SearchTenants(ctx, TenantFilter{NameContains: "missing"})
	require.NoError(t, err)
	assert.Empty(t, empty.Items)
	assert.Equal(t, int32(0), empty.Total)
}

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

func TestClient_MapsDomainErrors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := stubTenantAPI{
		searchTenants: func(context.Context, int32, ...services.CallOption) ([]d.Tenant, error) {
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

func fmtWrappedDomainError(err error) error {
	return errors.Join(err)
}
