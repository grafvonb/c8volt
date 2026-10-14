// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLoadDefaultPolicyFromArbitraryWorkingDirectory proves embedded catalogs
// do not depend on repository-relative process state.
func TestLoadDefaultPolicyFromArbitraryWorkingDirectory(t *testing.T) {
	original, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { require.NoError(t, os.Chdir(original)) })

	catalog, gaps, err := LoadDefaultPolicy()
	require.NoError(t, err)
	require.Len(t, catalog.Families, 37)
	require.Len(t, gaps.Gaps, 8)
	require.Contains(t, scenarioIDs(catalog.Requirements), "TestGetFamilyUserTaskVariables")
	require.Contains(t, scenarioIDs(catalog.Requirements), "TestVolumeGetFamilyUserTaskVariables")
	for _, family := range []string{
		"common-inventory", "common-manifest", "common-profile", "common-smoke", "common-examples",
		"batch",
		"tenant-logical", "tenant-physical",
		"real-state-jobs", "real-state-incidents", "real-state-listeners",
		"real-state-bpmn-error", "real-state-retention", "real-state-destructive",
		"real-state-cascade",
	} {
		require.Contains(t, familyIDs(catalog.Families), family)
	}
}

// TestValidatePolicyRejectsDuplicateStaleAndWholeFamilyCoverage proves catalog
// drift cannot silently remove required release work.
func TestValidatePolicyRejectsDuplicateStaleAndWholeFamilyCoverage(t *testing.T) {
	t.Parallel()
	catalog, gaps := testPolicy()
	require.NoError(t, ValidatePolicy(catalog, gaps))

	t.Run("duplicate scenario", func(t *testing.T) {
		broken := catalog
		broken.Requirements = append(append([]CoverageRequirement(nil), catalog.Requirements...), catalog.Requirements[0])
		require.ErrorContains(t, ValidatePolicy(broken, gaps), "duplicate scenario identity")
	})
	t.Run("whole family removed", func(t *testing.T) {
		broken := catalog
		broken.Requirements = append([]CoverageRequirement(nil), catalog.Requirements[1:]...)
		require.ErrorContains(t, ValidatePolicy(broken, gaps), "minimum is 1")
	})
	t.Run("stale gap", func(t *testing.T) {
		broken := gaps
		broken.Gaps = append([]PrerequisiteGap(nil), gaps.Gaps...)
		broken.Gaps[0].ScenarioIDs = []string{"removed-scenario"}
		require.ErrorContains(t, ValidatePolicy(catalog, broken), "unknown scenario")
	})
	t.Run("required waiver", func(t *testing.T) {
		broken := catalog
		broken.Requirements = append([]CoverageRequirement(nil), catalog.Requirements...)
		broken.Requirements[0].AcceptedGapID = "optional-gap"
		require.ErrorContains(t, ValidatePolicy(broken, gaps), "cannot be waived")
	})
}

// scenarioIDs extracts stable identities for catalog assertions.
func scenarioIDs(requirements []CoverageRequirement) []string {
	result := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		result = append(result, requirement.ScenarioID)
	}
	return result
}

// familyIDs extracts family identities for catalog assertions.
func familyIDs(families []CoverageFamily) []string {
	result := make([]string, 0, len(families))
	for _, family := range families {
		result = append(result, family.ID)
	}
	return result
}
