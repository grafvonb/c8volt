// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestNewOpsETASampleWindowRequiresMinimumSamplesAndExactTotal verifies ETA is withheld until the timing sample is trustworthy.
func TestNewOpsETASampleWindowRequiresMinimumSamplesAndExactTotal(t *testing.T) {
	startedAt := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)

	subThreshold := NewOpsETASampleWindow("loading runtime elements", startedAt, startedAt.Add(500*time.Millisecond), 3, 10, OpsDefaultETAMinimumSamples)
	require.True(t, subThreshold.MinimumSamplesMet)
	require.Zero(t, subThreshold.Elapsed)
	require.Nil(t, subThreshold.Rate)
	require.Nil(t, subThreshold.Remaining)

	tooFew := NewOpsETASampleWindow("loading runtime elements", startedAt, startedAt.Add(2*time.Second), 2, 10, OpsDefaultETAMinimumSamples)
	require.False(t, tooFew.MinimumSamplesMet)
	require.NotNil(t, tooFew.Rate)
	require.Nil(t, tooFew.Remaining)

	unknownTotal := NewOpsETASampleWindow("loading runtime elements", startedAt, startedAt.Add(3*time.Second), 3, 0, OpsDefaultETAMinimumSamples)
	require.True(t, unknownTotal.MinimumSamplesMet)
	require.NotNil(t, unknownTotal.Rate)
	require.Nil(t, unknownTotal.Remaining)

	exactTotal := NewOpsETASampleWindow("loading runtime elements", startedAt, startedAt.Add(3*time.Second), 3, 12, OpsDefaultETAMinimumSamples)
	require.True(t, exactTotal.MinimumSamplesMet)
	require.Equal(t, 3*time.Second, exactTotal.Elapsed)
	require.NotNil(t, exactTotal.Rate)
	require.InDelta(t, 1.0, *exactTotal.Rate, 0.001)
	require.NotNil(t, exactTotal.Remaining)
	require.Equal(t, 9*time.Second, exactTotal.Remaining.Round(time.Second))
}

// TestNewOpsETASampleWindowOmitsRemainingWhenComplete verifies completed frozen scopes do not show stale ETA.
func TestNewOpsETASampleWindowOmitsRemainingWhenComplete(t *testing.T) {
	startedAt := time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC)

	got := NewOpsETASampleWindow("loading runtime elements", startedAt, startedAt.Add(4*time.Second), 4, 4, OpsDefaultETAMinimumSamples)

	require.True(t, got.MinimumSamplesMet)
	require.NotNil(t, got.Rate)
	require.Nil(t, got.Remaining)
}
