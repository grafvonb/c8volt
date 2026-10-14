// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/integration/internal/validation"
	"github.com/stretchr/testify/require"
)

const envITTenantRoles = "C8VOLT_IT_TENANT_ROLES"

type logicalTenantEvidence struct {
	ClusterID        string              `json:"clusterId"`
	PhysicalTenantID string              `json:"physicalTenantId"`
	ObservedVersion  string              `json:"observedVersion"`
	TenantAID        string              `json:"tenantAId"`
	TenantBID        string              `json:"tenantBId"`
	Profiles         map[string]string   `json:"profiles"`
	TenantAKeys      []string            `json:"tenantAKeys"`
	TenantBKeys      []string            `json:"tenantBKeys"`
	Proofs           map[string][]string `json:"proofs"`
}

// physicalTenantEvidence records only sanitized observable routing proof and
// explicitly states the infrastructure properties it cannot certify.
type physicalTenantEvidence struct {
	ClusterID                    string              `json:"clusterId"`
	PhysicalTenantAID            string              `json:"physicalTenantAId"`
	PhysicalTenantBID            string              `json:"physicalTenantBId"`
	LogicalTenantID              string              `json:"logicalTenantId"`
	DefaultRoutePhysicalTenantID string              `json:"defaultRoutePhysicalTenantId"`
	ObservedVersions             map[string]string   `json:"observedVersions"`
	Profiles                     map[string]string   `json:"profiles"`
	TenantAKeys                  []string            `json:"tenantAKeys"`
	TenantBKeys                  []string            `json:"tenantBKeys"`
	Proofs                       map[string][]string `json:"proofs"`
	EvidenceLimits               []string            `json:"evidenceLimits"`
}

// TestLogicalTenantFamily proves filtering and backend authorization with
// distinct owned resources before making any negative tenant assertion.
func TestLogicalTenantFamily(t *testing.T) {
	registerReleaseScenario(t)
	roles, profiles := requireLogicalTenantRoles(t)
	logical := roles.Logical

	for _, role := range []string{"tenantA", "tenantB", "bothTenants", "restrictedA"} {
		ready, err := checkProfileReadiness(t, profiles[role])
		require.NoErrorf(t, err, "logical tenant role %s readiness", role)
		profiles[role] = ready
	}

	datasetA, _, err := seedVolumeProcessInstanceDataset(t, profiles["tenantA"], 2)
	require.NoError(t, err)
	datasetB, _, err := seedVolumeProcessInstanceDataset(t, profiles["tenantB"], 2)
	require.NoError(t, err)
	require.Equal(t, datasetA.PositiveBpmnProcessID, datasetB.PositiveBpmnProcessID)

	markerFilter := fmt.Sprintf("c8voltITRunId=%q", suite.marker)
	requireTenantVisibility(t, profiles["tenantA"], false, markerFilter, datasetA.allProcessInstanceKeys(), datasetB.allProcessInstanceKeys())
	requireTenantVisibility(t, profiles["tenantB"], false, markerFilter, datasetB.allProcessInstanceKeys(), datasetA.allProcessInstanceKeys())
	requireTenantVisibility(t, profiles["restrictedA"], false, markerFilter, datasetA.allProcessInstanceKeys(), datasetB.allProcessInstanceKeys())
	requireTenantVisibility(t, profiles["bothTenants"], true, markerFilter, append(datasetA.allProcessInstanceKeys(), datasetB.allProcessInstanceKeys()...), nil)

	searchMutation := runC8VoltForProfile(t, profiles["tenantA"].Name, "tenant-a-scoped-cancel",
		"--automation", "--json", "cancel", "pi", "--bpmn-process-id", datasetA.PositiveBpmnProcessID, "--state", "active")
	require.NoErrorf(t, searchMutation.Err, "tenant-A scoped cancellation: %s", searchMutation.Stderr)
	for _, key := range datasetA.PositiveProcessInstanceKeys {
		require.NoError(t, requireProcessInstanceStateOneOf(t, profiles["tenantA"], key, "CANCELED", "TERMINATED"))
	}
	for _, key := range datasetB.PositiveProcessInstanceKeys {
		require.NoError(t, requireProcessInstanceState(t, profiles["tenantB"], key, "ACTIVE"))
	}

	foreignKey := firstString(datasetB.PositiveProcessInstanceKeys)
	denied := runC8VoltForProfile(t, profiles["restrictedA"].Name, "tenant-restricted-a-foreign-key",
		"--automation", "--json", "cancel", "pi", "--key", foreignKey, "--dry-run")
	require.Error(t, denied.Err, "restricted-A principal must not plan a proven tenant-B key")
	require.NoError(t, requireProcessInstanceState(t, profiles["tenantB"], foreignKey, "ACTIVE"))

	explicit := runC8VoltForProfile(t, profiles["bothTenants"].Name, "tenant-dual-explicit-key",
		"--automation", "--json", "cancel", "pi", "--key", foreignKey, "--dry-run")
	require.NoErrorf(t, explicit.Err, "dual-authorized explicit key was falsely filtered: %s", explicit.Stderr)
	stdin := runC8VoltWithInput(t, "tenant-dual-stdin-key", foreignKey+"\n",
		argsForProfile(profiles["bothTenants"].Name, "--automation", "--json", "cancel", "pi", "-", "--dry-run")...)
	require.NoErrorf(t, stdin.Err, "dual-authorized stdin key was falsely filtered: %s", stdin.Stderr)

	allTenantMutation := runC8VoltForProfile(t, profiles["bothTenants"].Name, "tenant-all-cancel",
		"--all-tenants", "--automation", "--json", "cancel", "pi", "--bpmn-process-id", datasetB.PositiveBpmnProcessID, "--state", "active")
	require.NoErrorf(t, allTenantMutation.Err, "all-tenant cancellation: %s", allTenantMutation.Stderr)
	for _, key := range datasetB.PositiveProcessInstanceKeys {
		require.NoError(t, requireProcessInstanceStateOneOf(t, profiles["tenantB"], key, "CANCELED", "TERMINATED"))
	}

	conflict := runC8VoltForProfile(t, profiles["bothTenants"].Name, "tenant-conflicting-flags",
		"--tenant", logical.TenantAID, "--all-tenants", "get", "pi", "--limit", "1")
	require.Error(t, conflict.Err)
	destination := runC8VoltForProfile(t, profiles["bothTenants"].Name, "tenant-all-destination-rejected",
		"--all-tenants", "run", "pi", "--bpmn-process-id", datasetA.PositiveBpmnProcessID)
	require.Error(t, destination.Err)

	writeJSON(t, "tenant-logical-evidence.json", logicalTenantEvidence{
		ClusterID: logical.ClusterID, PhysicalTenantID: logical.PhysicalTenantID,
		ObservedVersion: profiles["bothTenants"].ActualVersion,
		TenantAID:       logical.TenantAID, TenantBID: logical.TenantBID,
		Profiles: map[string]string{
			"tenantA": profiles["tenantA"].Name, "tenantB": profiles["tenantB"].Name,
			"bothTenants": profiles["bothTenants"].Name, "restrictedA": profiles["restrictedA"].Name,
		},
		TenantAKeys: datasetA.allProcessInstanceKeys(), TenantBKeys: datasetB.allProcessInstanceKeys(),
		Proofs: map[string][]string{
			"read-only":       {"positive-owned-visibility", "opposite-tenant-absence", "all-tenant-discovery"},
			"live-post-state": {"tenant-a-scoped-cancel", "restricted-a-foreign-key-unchanged", "all-tenant-cancel"},
			"controlled":      {"dual-explicit-key", "dual-stdin-key", "conflicting-flags", "single-destination-restriction"},
		},
	})
}

// TestPhysicalTenantFamily proves routing and observable resource separation
// across two provisioned endpoints in one operator-declared cluster.
func TestPhysicalTenantFamily(t *testing.T) {
	registerReleaseScenario(t)
	roles, allProfiles := requireTenantRoles(t)
	physical := roles.Physical
	if physical == nil {
		t.Skip("physical tenant roles unavailable; add the physical role group to opt into physical coverage")
	}
	profiles := map[string]integrationProfile{
		"tenantA":      allProfiles[physical.TenantAProfile],
		"tenantB":      allProfiles[physical.TenantBProfile],
		"defaultRoute": allProfiles[physical.DefaultRouteProfile],
	}
	for _, role := range []string{"tenantA", "tenantB", "defaultRoute"} {
		ready, err := checkProfileReadiness(t, profiles[role])
		require.NoErrorf(t, err, "physical tenant role %s topology readiness", role)
		profiles[role] = ready
	}

	datasetA, _, err := seedVolumeProcessInstanceDataset(t, profiles["tenantA"], 2)
	require.NoError(t, err)
	datasetB, _, err := seedVolumeProcessInstanceDataset(t, profiles["tenantB"], 2)
	require.NoError(t, err)
	require.Equal(t, datasetA.PositiveBpmnProcessID, datasetB.PositiveBpmnProcessID)

	markerFilter := fmt.Sprintf("c8voltITRunId=%q", suite.marker)
	keysA := datasetA.allProcessInstanceKeys()
	keysB := datasetB.allProcessInstanceKeys()
	requireTenantVisibility(t, profiles["tenantA"], false, markerFilter, keysA, keysB)
	requireTenantVisibility(t, profiles["tenantB"], false, markerFilter, keysB, keysA)
	requireTenantVisibility(t, profiles["tenantA"], true, markerFilter, keysA, keysB)

	defaultPresent, defaultAbsent := keysA, keysB
	if physical.DefaultRoutePhysicalTenantID == physical.PhysicalTenantBID {
		defaultPresent, defaultAbsent = keysB, keysA
	}
	requireTenantVisibility(t, profiles["defaultRoute"], false, markerFilter, defaultPresent, defaultAbsent)

	mutation := runC8VoltForProfile(t, profiles["tenantA"].Name, "physical-tenant-a-scoped-cancel",
		"--automation", "--json", "cancel", "pi", "--bpmn-process-id", datasetA.PositiveBpmnProcessID, "--state", "active")
	require.NoErrorf(t, mutation.Err, "physical tenant A scoped cancellation: %s", mutation.Stderr)
	for _, key := range datasetA.PositiveProcessInstanceKeys {
		require.NoError(t, requireProcessInstanceStateOneOf(t, profiles["tenantA"], key, "CANCELED", "TERMINATED"))
	}
	for _, key := range datasetB.PositiveProcessInstanceKeys {
		require.NoError(t, requireProcessInstanceState(t, profiles["tenantB"], key, "ACTIVE"))
	}

	writeJSON(t, "tenant-physical-evidence.json", physicalTenantEvidence{
		ClusterID: physical.ClusterID, PhysicalTenantAID: physical.PhysicalTenantAID,
		PhysicalTenantBID: physical.PhysicalTenantBID, LogicalTenantID: physical.LogicalTenantID,
		DefaultRoutePhysicalTenantID: physical.DefaultRoutePhysicalTenantID,
		ObservedVersions: map[string]string{
			"tenantA": profiles["tenantA"].ActualVersion, "tenantB": profiles["tenantB"].ActualVersion,
			"defaultRoute": profiles["defaultRoute"].ActualVersion,
		},
		Profiles: map[string]string{
			"tenantA": profiles["tenantA"].Name, "tenantB": profiles["tenantB"].Name,
			"defaultRoute": profiles["defaultRoute"].Name,
		},
		TenantAKeys: keysA, TenantBKeys: keysB,
		Proofs: map[string][]string{
			"read-only":       {"topology-readiness", "positive-local-visibility", "foreign-fixture-absence", "all-tenants-endpoint-confinement", "default-route-equivalence"},
			"live-post-state": {"tenant-a-scoped-cancel", "tenant-b-control-unchanged"},
		},
		EvidenceLimits: []string{"observable API routing and resource separation only", "no storage, backup, provisioning, or compute-isolation claim"},
	})
}

// requireLogicalTenantRoles loads the credential-free role file and verifies
// every referenced profile against default-local tenant/version metadata.
func requireLogicalTenantRoles(t *testing.T) (validation.TenantRoles, map[string]integrationProfile) {
	t.Helper()
	roles, byName := requireTenantRoles(t)
	logical := roles.Logical
	return roles, map[string]integrationProfile{
		"tenantA": byName[logical.TenantAProfile], "tenantB": byName[logical.TenantBProfile],
		"bothTenants": byName[logical.BothTenantsProfile], "restrictedA": byName[logical.RestrictedAProfile],
	}
}

// requireTenantRoles loads and validates every declared logical and physical
// profile against the default local configuration without retaining secrets.
func requireTenantRoles(t *testing.T) (validation.TenantRoles, map[string]integrationProfile) {
	t.Helper()
	path := strings.TrimSpace(os.Getenv(envITTenantRoles))
	if path == "" {
		if required := strings.TrimSpace(os.Getenv("C8VOLT_IT_REQUIRE_TENANTS")); required != "" && required != string(validation.TenantCoverageNone) {
			t.Fatalf("%s is required when C8VOLT_IT_REQUIRE_TENANTS=%s", envITTenantRoles, required)
		}
		t.Skipf("logical tenant roles unavailable; set %s to opt into tenant coverage", envITTenantRoles)
	}
	roles, err := validation.LoadTenantRoles(path)
	require.NoError(t, err)
	require.NotNil(t, roles.Logical)
	configFile, err := readDefaultLocalConfig()
	require.NoError(t, err)
	resolved, err := profilesFromDefaultConfig(configFile, roles.ProfileNames())
	require.NoError(t, err)
	metadata := make(map[string]validation.TenantProfileMetadata, len(resolved))
	byName := make(map[string]integrationProfile, len(resolved))
	for _, profile := range resolved {
		metadata[profile.Name] = validation.TenantProfileMetadata{
			Name: profile.Name, ExpectedMinor: profile.ExpectedVersion,
			LogicalTenantID: profile.Tenant, CamundaBaseURL: profile.CamundaBaseURL,
		}
		byName[profile.Name] = profile
	}
	requestedMinor := strings.TrimSpace(os.Getenv("C8VOLT_IT_RELEASE_VERSION"))
	require.NoError(t, validation.ValidateTenantRoles(roles, metadata, requestedMinor))
	requiredCoverage := validation.TenantCoverage(strings.TrimSpace(os.Getenv("C8VOLT_IT_REQUIRE_TENANTS")))
	if requiredCoverage == "" {
		requiredCoverage = validation.TenantCoverageNone
	}
	require.NoError(t, validation.ValidateRequiredTenantCoverage(roles, requiredCoverage))
	return roles, byName
}

// requireTenantVisibility checks owned keys first, then absence of foreign
// keys, so an empty or unauthorized result can never masquerade as isolation.
func requireTenantVisibility(t *testing.T, profile integrationProfile, allTenants bool, markerFilter string, present []string, absent []string) {
	t.Helper()
	args := []string{"--keys-only", "get", "pi", "--state", "active", "--var", markerFilter}
	if allTenants {
		args = append([]string{"--all-tenants"}, args...)
	}
	result := runC8VoltForProfile(t, profile.Name, "tenant-visibility-"+profile.Name, args...)
	require.NoErrorf(t, result.Err, "tenant visibility for %s: %s", profile.Name, result.Stderr)
	keys := splitNonEmptyLines(result.Stdout)
	for _, key := range present {
		require.Containsf(t, keys, key, "profile %s must positively observe owned key %s", profile.Name, key)
	}
	for _, key := range absent {
		require.NotContainsf(t, keys, key, "profile %s must not observe foreign key %s", profile.Name, key)
	}
}
