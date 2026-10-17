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

	require.Equal(t, States{StateActive, StateSuspended, StateCompleted, StateCanceled, StateTerminated}, got)
	require.True(t, got.Contains(StateActive))
	require.True(t, got.Contains(StateSuspended))
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

	require.Equal(t, States{StateActive, StateSuspended, StateCompleted, StateCanceled, StateTerminated}, ObservableProcessInstanceCreationStates())
}

// TestStateSuspendedIsNonterminal preserves existing domain terminal and case-insensitive membership semantics.
func TestStateSuspendedIsNonterminal(t *testing.T) {
	t.Parallel()
	require.Equal(t, "SUSPENDED", StateSuspended.String())
	for _, state := range []State{StateSuspended, "suspended", "SuSpEnDeD"} {
		require.True(t, state.EqualsIgnoreCase(StateSuspended))
		require.True(t, States{StateActive, StateSuspended}.Contains(state))
		require.False(t, state.In(StateActive, StateCompleted, StateCanceled, StateTerminated, StateAbsent))
		require.False(t, state.IsTerminal())
	}
	for _, state := range []State{StateActive, StateAll, StateUnknown, "FUTURE_STATE"} {
		require.False(t, state.IsTerminal(), state.String())
	}
	for _, state := range []State{StateCompleted, StateCanceled, StateTerminated, StateAbsent, "completed", "canceled", "terminated", "absent"} {
		require.True(t, state.IsTerminal(), state.String())
	}
}
