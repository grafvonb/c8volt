// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// TestCanonicalAndObservedMinorParsing keeps strict request syntax separate
// from full server-version parsing.
func TestCanonicalAndObservedMinorParsing(t *testing.T) {
	t.Parallel()
	for _, minor := range []string{"8.8", "8.9", "8.10"} {
		got, err := CanonicalMinor(minor)
		require.NoError(t, err)
		require.Equal(t, minor, got)
	}
	for _, value := range []string{"v8.10", "8.10.3", "8.10-SNAPSHOT", "18.10", " 8.10 "} {
		_, err := CanonicalMinor(value)
		require.Error(t, err, value)
	}
	got, err := ObservedMinor("v8.10.3")
	require.NoError(t, err)
	require.Equal(t, "8.10", got)
	for _, value := range []string{"8.10-SNAPSHOT", "8.10.3-alpha", "18.10.3", "8"} {
		_, err := ObservedMinor(value)
		require.Error(t, err, value)
	}
}
