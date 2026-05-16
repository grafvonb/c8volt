// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDetermineNoOfWorkersDefaultUsesIOMultiplier verifies automatic worker
// selection is higher than CPU count for IO-heavy bulk calls while still capped.
func TestDetermineNoOfWorkersDefaultUsesIOMultiplier(t *testing.T) {
	old := runtime.GOMAXPROCS(4)
	t.Cleanup(func() { runtime.GOMAXPROCS(old) })

	require.Equal(t, 8, DetermineNoOfWorkers(100, 0, false))
	require.Equal(t, 3, DetermineNoOfWorkers(3, 0, false))
}

// TestDetermineNoOfWorkersDefaultHardLimit prevents large machines from
// creating an excessive default request burst against Camunda.
func TestDetermineNoOfWorkersDefaultHardLimit(t *testing.T) {
	old := runtime.GOMAXPROCS(64)
	t.Cleanup(func() { runtime.GOMAXPROCS(old) })

	require.Equal(t, defaultWorkerHardLimit, DetermineNoOfWorkers(1000, 0, false))
}

// TestDetermineNoOfWorkersExplicitWorkersOverrideDefaultCap keeps --workers as
// the user-controlled escape hatch while still clamping to the amount of work.
func TestDetermineNoOfWorkersExplicitWorkersOverrideDefaultCap(t *testing.T) {
	old := runtime.GOMAXPROCS(2)
	t.Cleanup(func() { runtime.GOMAXPROCS(old) })

	require.Equal(t, 20, DetermineNoOfWorkers(100, 20, false))
	require.Equal(t, 7, DetermineNoOfWorkers(7, 20, false))
}

// TestDetermineNoOfWorkersNoWorkerLimitPreservesLegacyUncappedDefault verifies
// --no-worker-limit still lets automatic selection use all queued jobs.
func TestDetermineNoOfWorkersNoWorkerLimitPreservesLegacyUncappedDefault(t *testing.T) {
	old := runtime.GOMAXPROCS(2)
	t.Cleanup(func() { runtime.GOMAXPROCS(old) })

	require.Equal(t, 100, DetermineNoOfWorkers(100, 0, true))
}
