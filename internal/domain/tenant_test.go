// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
