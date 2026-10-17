// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStateAcceptsCaseInsensitiveCanonicalValues(t *testing.T) {
	got, ok := ParseState(" COMPLETED ")

	require.True(t, ok)
	require.Equal(t, StateCompleted, got)
}

func TestParseStateKeepsCancelledAlias(t *testing.T) {
	got, ok := ParseState("cancelled")

	require.True(t, ok)
	require.Equal(t, StateCanceled, got)
}

// TestParseStateLifecycleVocabulary protects canonical parsing and existing fallback semantics.
func TestParseStateLifecycleVocabulary(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		input string
		want  State
		valid bool
	}{
		{"SUSPENDED", StateSuspended, true},
		{"suspended", StateSuspended, true},
		{"SuSpEnDeD", StateSuspended, true},
		{" \t suspended \n", StateSuspended, true},
		{"canceled", StateCanceled, true},
		{" CaNcElLeD ", StateCanceled, true},
		{"terminated", StateTerminated, true},
		{"absent", StateAbsent, true},
		{"active", StateActive, true},
		{"completed", StateCompleted, true},
		{"all", StateAll, true},
		{"UNKNOWN", StateUnknown, false},
		{"FUTURE_STATE", StateUnknown, false},
		{"", StateUnknown, false},
	} {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := ParseState(tt.input)
			require.Equal(t, tt.valid, ok)
			require.Equal(t, tt.want, got)
		})
	}
	require.Contains(t, ValidStateStrings(), "SUSPENDED")
	states, err := ParseStates([]string{" suspended ", "CaNcElLeD", "terminated", "absent"})
	require.NoError(t, err)
	require.Equal(t, States{StateSuspended, StateCanceled, StateTerminated, StateAbsent}, states)
	_, err = ParseStates([]string{"suspended", "FUTURE_STATE"})
	require.Error(t, err)
}

// TestStateTerminalClassification preserves the public ABSENT distinction and nonterminal suspension.
func TestStateTerminalClassification(t *testing.T) {
	t.Parallel()
	for _, state := range []State{StateSuspended, "suspended", StateActive, StateAll, StateAbsent, StateUnknown, "FUTURE_STATE"} {
		require.False(t, state.IsTerminal(), state.String())
	}
	for _, state := range []State{StateCompleted, StateCanceled, StateTerminated, "completed", "canceled", "terminated"} {
		require.True(t, state.IsTerminal(), state.String())
	}
}
