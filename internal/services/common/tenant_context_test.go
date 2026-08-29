// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
