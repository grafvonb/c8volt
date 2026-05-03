// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verifies tenant filtering treats pattern-looking input as literal text.
func TestFilterTenantsByNameOrIDContains_TreatsTextLiterally(t *testing.T) {
	tenants := []Tenant{
		{TenantId: "tenant-a", Name: "demo.*"},
		{TenantId: "tenant-b", Name: "demo-1"},
		{TenantId: "tenant-c", Name: "prod"},
	}

	got := FilterTenantsByNameOrIDContains(tenants, ".*")

	require.Len(t, got, 1)
	assert.Equal(t, "tenant-a", got[0].TenantId)
}

// Verifies tenant filtering matches either tenant ID or name without case sensitivity.
func TestFilterTenantsByNameOrIDContains_MatchesNameOrTenantIDCaseInsensitively(t *testing.T) {
	tenants := []Tenant{
		{TenantId: "dev01", Name: "Development Stage 01"},
		{TenantId: "tenant-a", Name: "Alpha"},
		{TenantId: "tenant-b", Name: "Beta"},
	}

	got := FilterTenantsByNameOrIDContains(tenants, "DEV")

	require.Len(t, got, 1)
	assert.Equal(t, "dev01", got[0].TenantId)

	got = FilterTenantsByNameOrIDContains(tenants, "alp")

	require.Len(t, got, 1)
	assert.Equal(t, "tenant-a", got[0].TenantId)
}

// Verifies empty tenant filter text returns an equal copy instead of aliasing the input slice.
func TestFilterTenantsByNameOrIDContains_EmptyTextReturnsCopy(t *testing.T) {
	tenants := []Tenant{{TenantId: "tenant-a", Name: "Alpha"}}

	got := FilterTenantsByNameOrIDContains(tenants, "")

	require.Equal(t, tenants, got)
	require.NotSame(t, &tenants[0], &got[0])
}
