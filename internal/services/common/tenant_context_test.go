// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verifies tenant evidence is counted by unique affected target and known tenant IDs are sorted.
func TestTenantEvidenceAccumulator_SnapshotCountsUniqueTargetsAndSortsTenants(t *testing.T) {
	t.Parallel()

	acc := NewTenantEvidenceAccumulator()
	acc.Add("2251799813685249", "tenant-b")
	acc.Add("2251799813685250", "tenant-a")
	acc.Add("2251799813685251", "tenant-a")

	got := acc.Snapshot()

	assert.Equal(t, 3, got.TargetCount)
	assert.Equal(t, []string{"tenant-a", "tenant-b"}, got.ResolvedTenantIDs)
	assert.Equal(t, 0, got.UnknownTargetCount)
}

// Verifies the first observation for a target wins so duplicate plan entries cannot inflate evidence.
func TestTenantEvidenceAccumulator_AddSuppressesDuplicateTargets(t *testing.T) {
	t.Parallel()

	acc := NewTenantEvidenceAccumulator()
	acc.Add("2251799813685249", "tenant-a")
	acc.Add("2251799813685249", "tenant-b")
	acc.Add("2251799813685249", "")

	got := acc.Snapshot()

	assert.Equal(t, 1, got.TargetCount)
	assert.Equal(t, []string{"tenant-a"}, got.ResolvedTenantIDs)
	assert.Equal(t, 0, got.UnknownTargetCount)
}

// Verifies empty tenant metadata is tracked once for each unique affected target.
func TestTenantEvidenceAccumulator_SnapshotCountsUnknownTargets(t *testing.T) {
	t.Parallel()

	acc := NewTenantEvidenceAccumulator()
	acc.Add("2251799813685249", "")
	acc.Add("2251799813685250", "")
	acc.Add("2251799813685249", "tenant-a")

	got := acc.Snapshot()

	assert.Equal(t, 2, got.TargetCount)
	assert.Empty(t, got.ResolvedTenantIDs)
	assert.Equal(t, 2, got.UnknownTargetCount)
}

// Verifies merged snapshots preserve per-key de-duplication across pages or workflow steps.
func TestTenantEvidenceAccumulator_MergeDeduplicatesKnownAndUnknownTargets(t *testing.T) {
	t.Parallel()

	pageOne := NewTenantEvidenceAccumulator()
	pageOne.Add("2251799813685249", "tenant-b")
	pageOne.Add("2251799813685250", "")

	pageTwo := NewTenantEvidenceAccumulator()
	pageTwo.Add("2251799813685249", "tenant-a")
	pageTwo.Add("2251799813685251", "tenant-a")
	pageTwo.Add("2251799813685252", "")

	merged := NewTenantEvidenceAccumulator()
	merged.Merge(pageOne.Snapshot())
	merged.Merge(pageTwo.Snapshot())

	got := merged.Snapshot()

	assert.Equal(t, 4, got.TargetCount)
	assert.Equal(t, []string{"tenant-a", "tenant-b"}, got.ResolvedTenantIDs)
	assert.Equal(t, 2, got.UnknownTargetCount)
}

// Verifies aggregate-only evidence contributes tenant IDs and counts without
// being represented as synthetic keyed targets.
func TestTenantEvidenceAccumulator_AddAggregateKeepsIdentitySeparateFromCount(t *testing.T) {
	t.Parallel()

	page := NewTenantEvidenceAccumulator()
	page.Add("pd-1", "tenant-b")
	page.AddAggregate([]string{"tenant-a", "tenant-a", ""}, 3, 1)

	merged := NewTenantEvidenceAccumulator()
	merged.Merge(page.Snapshot())
	got := merged.Snapshot()

	assert.Equal(t, 4, got.TargetCount)
	assert.Equal(t, []string{"tenant-a", "tenant-b"}, got.ResolvedTenantIDs)
	assert.Equal(t, 1, got.UnknownTargetCount)
}

// Verifies snapshots are immutable copies so callers cannot corrupt accumulated evidence.
func TestTenantEvidenceAccumulator_SnapshotReturnsCopies(t *testing.T) {
	t.Parallel()

	acc := NewTenantEvidenceAccumulator()
	acc.Add("2251799813685249", "tenant-b")
	acc.Add("2251799813685250", "tenant-a")

	got := acc.Snapshot()
	got.ResolvedTenantIDs[0] = "changed"

	next := acc.Snapshot()

	assert.Equal(t, []string{"tenant-a", "tenant-b"}, next.ResolvedTenantIDs)
}

// TestEffectiveTenant_UsesNormalizedVersionSpecificTenantSemantics verifies
// service request builders consume the effective tenant directly instead of
// rendering empty discovery as a default tenant for Camunda 8.8 and newer.
func TestEffectiveTenant_UsesNormalizedVersionSpecificTenantSemantics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		version    toolx.CamundaVersion
		wantTenant string
	}{
		{name: "v87 omitted tenant becomes default tenant", version: toolx.V87, wantTenant: config.DefaultTenant},
		{name: "v88 omitted tenant stays unfiltered", version: toolx.V88},
		{name: "v89 omitted tenant stays unfiltered", version: toolx.V89},
		{name: "v810 omitted tenant stays unfiltered", version: toolx.V810},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.New()
			cfg.App.CamundaVersion = tt.version
			require.NoError(t, cfg.Normalize())

			assert.Equal(t, tt.wantTenant, EffectiveTenant(cfg))
		})
	}
}
