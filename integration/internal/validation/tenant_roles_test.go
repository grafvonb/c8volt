// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLoadAndValidateTenantRoles accepts the four distinct credential-free
// logical principals required for positive and negative tenant assertions.
func TestLoadAndValidateTenantRoles(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "roles.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
  "schemaVersion": 1,
  "logical": {
    "clusterId": "local-cluster",
    "physicalTenantId": "default",
    "tenantAId": "tenant-a",
    "tenantBId": "tenant-b",
    "tenantAProfile": "tenant-a",
    "tenantBProfile": "tenant-b",
    "bothTenantsProfile": "tenant-both",
    "restrictedAProfile": "tenant-a-restricted"
  }
}`), 0o644))

	roles, err := LoadTenantRoles(path)
	require.NoError(t, err)
	require.NoError(t, ValidateTenantRoles(roles, validTenantProfiles(), "8.10"))
	require.Equal(t, []string{"tenant-a", "tenant-a-restricted", "tenant-b", "tenant-both"}, roles.LogicalProfileNames())
}

// TestTenantRolesRejectInvalidAssignments covers duplicate IDs, missing
// profiles, conflicting roles, tenant mismatch, version mismatch, and secrets.
func TestTenantRolesRejectInvalidAssignments(t *testing.T) {
	t.Parallel()
	base := TenantRoles{SchemaVersion: SchemaVersion, Logical: &LogicalTenantRoles{
		ClusterID: "cluster", PhysicalTenantID: "default", TenantAID: "tenant-a", TenantBID: "tenant-b",
		TenantAProfile: "tenant-a", TenantBProfile: "tenant-b", BothTenantsProfile: "tenant-both", RestrictedAProfile: "tenant-a-restricted",
	}}
	cases := map[string]func(*TenantRoles, map[string]TenantProfileMetadata){
		"duplicate tenant IDs": func(roles *TenantRoles, _ map[string]TenantProfileMetadata) {
			roles.Logical.TenantBID = roles.Logical.TenantAID
		},
		"missing profile": func(_ *TenantRoles, profiles map[string]TenantProfileMetadata) { delete(profiles, "tenant-b") },
		"conflicting roles": func(roles *TenantRoles, _ map[string]TenantProfileMetadata) {
			roles.Logical.RestrictedAProfile = roles.Logical.TenantAProfile
		},
		"tenant mismatch": func(_ *TenantRoles, profiles map[string]TenantProfileMetadata) {
			p := profiles["tenant-b"]
			p.LogicalTenantID = "tenant-a"
			profiles["tenant-b"] = p
		},
		"version mismatch": func(_ *TenantRoles, profiles map[string]TenantProfileMetadata) {
			p := profiles["tenant-both"]
			p.ExpectedMinor = "8.9"
			profiles["tenant-both"] = p
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			roles := base
			logical := *base.Logical
			roles.Logical = &logical
			profiles := validTenantProfiles()
			mutate(&roles, profiles)
			require.Error(t, ValidateTenantRoles(roles, profiles, "8.10"))
		})
	}

	path := filepath.Join(t.TempDir(), "roles-with-secret.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"schemaVersion":1,"logical":{},"clientSecret":"nope"}`), 0o644))
	_, err := LoadTenantRoles(path)
	require.ErrorContains(t, err, "unknown field")
}

// validTenantProfiles supplies non-secret profile metadata for validation tests.
func validTenantProfiles() map[string]TenantProfileMetadata {
	return map[string]TenantProfileMetadata{
		"tenant-a":            {Name: "tenant-a", ExpectedMinor: "8.10", LogicalTenantID: "tenant-a"},
		"tenant-b":            {Name: "tenant-b", ExpectedMinor: "8.10", LogicalTenantID: "tenant-b"},
		"tenant-both":         {Name: "tenant-both", ExpectedMinor: "8.10", LogicalTenantID: "tenant-a"},
		"tenant-a-restricted": {Name: "tenant-a-restricted", ExpectedMinor: "8.10", LogicalTenantID: "tenant-a"},
	}
}
