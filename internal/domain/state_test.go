// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestObservableProcessInstanceCreationStates documents the broad run-confirmation state set.
func TestObservableProcessInstanceCreationStates(t *testing.T) {
	got := ObservableProcessInstanceCreationStates()

	require.Equal(t, States{StateActive, StateCompleted, StateCanceled, StateTerminated}, got)
	require.True(t, got.Contains(StateActive))
	require.True(t, got.Contains(StateCompleted))
	require.True(t, got.Contains(StateCanceled))
	require.True(t, got.Contains(StateTerminated))
	require.False(t, got.Contains(StateAll))
	require.False(t, got.Contains(StateAbsent))
	require.False(t, got.Contains(StateUnknown))
}

// TestObservableProcessInstanceCreationStates_ReturnsCopy protects callers from mutating shared confirmation semantics.
func TestObservableProcessInstanceCreationStates_ReturnsCopy(t *testing.T) {
	got := ObservableProcessInstanceCreationStates()
	got[0] = StateUnknown

	require.Equal(t, States{StateActive, StateCompleted, StateCanceled, StateTerminated}, ObservableProcessInstanceCreationStates())
}
