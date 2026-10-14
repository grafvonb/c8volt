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

// TestValidatePhysicalTenantRoles accepts distinct explicit routes, one common
// logical tenant, and a separately configured default-route equivalence control.
func TestValidatePhysicalTenantRoles(t *testing.T) {
	t.Parallel()
	roles := validTenantRolesWithPhysical()
	require.NoError(t, ValidateTenantRoles(roles, validTenantProfilesWithPhysical(), "8.10"))
	require.NoError(t, ValidateRequiredTenantCoverage(roles, TenantCoveragePhysical))
	require.Equal(t, []string{
		"physical-a", "physical-b", "physical-default", "tenant-a", "tenant-a-restricted", "tenant-b", "tenant-both",
	}, roles.ProfileNames())
}

// TestPhysicalTenantRolesRejectInvalidRoutesAndDeclarations prevents URL
// similarity, duplicate identities, and missing controls from becoming proof.
func TestPhysicalTenantRolesRejectInvalidRoutesAndDeclarations(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*TenantRoles, map[string]TenantProfileMetadata){
		"duplicate physical IDs": func(roles *TenantRoles, _ map[string]TenantProfileMetadata) {
			roles.Physical.PhysicalTenantBID = roles.Physical.PhysicalTenantAID
		},
		"foreign cluster declaration": func(roles *TenantRoles, _ map[string]TenantProfileMetadata) {
			roles.Physical.ClusterID = "other-cluster"
		},
		"wrong explicit route": func(_ *TenantRoles, profiles map[string]TenantProfileMetadata) {
			profile := profiles["physical-b"]
			profile.CamundaBaseURL = "https://gateway.example.test/physical-tenants/physical-a/v2"
			profiles["physical-b"] = profile
		},
		"default profile uses explicit route": func(_ *TenantRoles, profiles map[string]TenantProfileMetadata) {
			profile := profiles["physical-default"]
			profile.CamundaBaseURL = "https://gateway.example.test/physical-tenants/physical-a/v2"
			profiles["physical-default"] = profile
		},
		"default identity outside pair": func(roles *TenantRoles, _ map[string]TenantProfileMetadata) {
			roles.Physical.DefaultRoutePhysicalTenantID = "physical-c"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			roles := validTenantRolesWithPhysical()
			profiles := validTenantProfilesWithPhysical()
			mutate(&roles, profiles)
			require.Error(t, ValidateTenantRoles(roles, profiles, "8.10"))
		})
	}

	roles := TenantRoles{SchemaVersion: SchemaVersion, Logical: validTenantRolesWithPhysical().Logical}
	require.ErrorContains(t, ValidateRequiredTenantCoverage(roles, TenantCoveragePhysical), "physical tenant role group is required")
}

// TestPhysicalRouteIdentity distinguishes explicit physical routing from the
// default route without using host similarity as cluster evidence.
func TestPhysicalRouteIdentity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		baseURL string
		want    string
		wantErr bool
	}{
		{name: "explicit", baseURL: "https://gateway.example.test/physical-tenants/tenant-a/v2", want: "tenant-a"},
		{name: "prefixed explicit", baseURL: "https://gateway.example.test/camunda/physical-tenants/tenant-b/v2", want: "tenant-b"},
		{name: "default", baseURL: "https://gateway.example.test/v2"},
		{name: "wrong API version", baseURL: "https://gateway.example.test/physical-tenants/tenant-a/v1", wantErr: true},
		{name: "credential bearing", baseURL: "https://user:secret@gateway.example.test/v2", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PhysicalRouteIdentity(tc.baseURL)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
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

// validTenantRolesWithPhysical supplies a complete declared cluster with an
// explicit default route mapped to physical tenant A.
func validTenantRolesWithPhysical() TenantRoles {
	return TenantRoles{
		SchemaVersion: SchemaVersion,
		Logical: &LogicalTenantRoles{
			ClusterID: "cluster", PhysicalTenantID: "default", TenantAID: "tenant-a", TenantBID: "tenant-b",
			TenantAProfile: "tenant-a", TenantBProfile: "tenant-b", BothTenantsProfile: "tenant-both", RestrictedAProfile: "tenant-a-restricted",
		},
		Physical: &PhysicalTenantRoles{
			ClusterID: "cluster", PhysicalTenantAID: "physical-a", PhysicalTenantBID: "physical-b", LogicalTenantID: "shared-logical",
			TenantAProfile: "physical-a", TenantBProfile: "physical-b", DefaultRouteProfile: "physical-default", DefaultRoutePhysicalTenantID: "physical-a",
		},
	}
}

// validTenantProfilesWithPhysical extends the logical fixtures with normalized
// explicit and default v2 routes.
func validTenantProfilesWithPhysical() map[string]TenantProfileMetadata {
	profiles := validTenantProfiles()
	profiles["physical-a"] = TenantProfileMetadata{Name: "physical-a", ExpectedMinor: "8.10", LogicalTenantID: "shared-logical", CamundaBaseURL: "https://gateway.example.test/physical-tenants/physical-a/v2"}
	profiles["physical-b"] = TenantProfileMetadata{Name: "physical-b", ExpectedMinor: "8.10", LogicalTenantID: "shared-logical", CamundaBaseURL: "https://gateway.example.test/physical-tenants/physical-b/v2"}
	profiles["physical-default"] = TenantProfileMetadata{Name: "physical-default", ExpectedMinor: "8.10", LogicalTenantID: "shared-logical", CamundaBaseURL: "https://gateway.example.test/v2"}
	return profiles
}
