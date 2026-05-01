// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSortTenantsByNameAscThenTenantIDAsc(t *testing.T) {
	tenants := []Tenant{
		{TenantId: "tenant-c", Name: "Beta"},
		{TenantId: "tenant-b", Name: "Alpha"},
		{TenantId: "tenant-a", Name: "Alpha"},
	}

	SortTenantsByNameAscThenTenantIDAsc(tenants)

	require.Len(t, tenants, 3)
	assert.Equal(t, "tenant-a", tenants[0].TenantId)
	assert.Equal(t, "tenant-b", tenants[1].TenantId)
	assert.Equal(t, "tenant-c", tenants[2].TenantId)
}

func TestFilterTenantsByNameContains_TreatsTextLiterally(t *testing.T) {
	tenants := []Tenant{
		{TenantId: "tenant-a", Name: "demo.*"},
		{TenantId: "tenant-b", Name: "demo-1"},
		{TenantId: "tenant-c", Name: "prod"},
	}

	got := FilterTenantsByNameContains(tenants, ".*")

	require.Len(t, got, 1)
	assert.Equal(t, "tenant-a", got[0].TenantId)
}

func TestFilterTenantsByNameContains_EmptyTextReturnsCopy(t *testing.T) {
	tenants := []Tenant{{TenantId: "tenant-a", Name: "Alpha"}}

	got := FilterTenantsByNameContains(tenants, "")

	require.Equal(t, tenants, got)
	require.NotSame(t, &tenants[0], &got[0])
}
