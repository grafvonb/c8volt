// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"errors"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestNormalizeProcessInstanceSearchFilter checks normalization, all-member
// capability validation, caller ownership and preservation of capability errors.
func TestNormalizeProcessInstanceSearchFilter(t *testing.T) {
	t.Parallel()
	original := d.ProcessInstanceFilter{States: d.States{" active ", "suspended", d.StateCanceled, d.StateTerminated, d.StateCompleted}}
	var observed []d.States
	got, err := NormalizeProcessInstanceSearchFilter(original, func(states d.States) error {
		observed = append(observed, states)
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []d.States{{d.StateActive, d.StateSuspended, d.StateTerminated, d.StateCompleted}, {d.StateSuspended}}, observed)
	require.Equal(t, observed[0], got.States)
	require.Contains(t, got.String(), "states=ACTIVE, SUSPENDED, TERMINATED, COMPLETED")
	got.States[0] = d.StateCompleted
	require.Equal(t, d.State(" active "), original.States[0])
	failure := errors.New("capability failed")
	_, err = NormalizeProcessInstanceSearchFilter(original, func(d.States) error { return failure })
	require.ErrorIs(t, err, failure)
	// An empty plural slice is still an ordinary scalar query, including ALL's
	// existing empty-state representation and the scalar canceled alias.
	for _, state := range []d.State{"", d.StateActive, d.StateCanceled} {
		got, err := NormalizeProcessInstanceSearchFilter(d.ProcessInstanceFilter{State: state}, func(states d.States) error {
			require.Equal(t, d.States{state}, states)
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, state, got.State)
		require.Empty(t, got.States)
	}
}
