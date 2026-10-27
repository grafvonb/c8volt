// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// TestNormalizeProcessInstanceKeys preserves suspension validation and ordering
// while sharing exact identity normalization with resumption.
func TestNormalizeProcessInstanceKeys(t *testing.T) {
	t.Parallel()
	for _, normalize := range []func(typex.Keys) (typex.Keys, error){NormalizeProcessInstanceKeys, NormalizeProcessInstanceSuspensionKeys} {
		input := typex.Keys{" 2251799813685249\n", "2251799813685250", "2251799813685249", "0000000000000001"}
		got, err := normalize(input)
		require.NoError(t, err)
		require.Equal(t, typex.Keys{"2251799813685249", "2251799813685250", "0000000000000001"}, got)
		got[0] = "changed"
		require.Equal(t, " 2251799813685249\n", input[0])
		got, err = normalize(nil)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Empty(t, got)
		for _, invalid := range []string{"", " ", "123", "12345678901234567", "123456789012345x", "-123456789012345", "１２３４５６７８９０１２３４５６", "12345678 0123456"} {
			got, err := normalize(typex.Keys{"2251799813685249", invalid})
			require.ErrorIs(t, err, d.ErrValidation)
			require.Nil(t, got, "invalid suffix must not return a partial executable scope")
		}
	}
}
