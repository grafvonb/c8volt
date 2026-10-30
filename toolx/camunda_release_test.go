// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCamundaRelease validates numeric release lines and bounded, single-line display text.
func TestCamundaRelease(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input        string
		major, minor uint64
	}{
		{"8.9", 8, 9}, {"8.10", 8, 10}, {"  v8.10.0-alpha4  ", 8, 10},
		{"V9.0.12-rc.1+build.2", 9, 0}, {"08.009.0001", 8, 9}, {"0.0", 0, 0},
		{"8.9+" + strings.Repeat("a", 124), 8, 9},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseCamundaRelease(tc.input)
			require.NoError(t, err)
			require.Equal(t, CamundaRelease{Major: tc.major, Minor: tc.minor, Display: strings.TrimSpace(tc.input)}, got)
		})
	}
	for _, input := range []string{
		"", " ", "8", "810", "-8.9", "8.-9", "8.9.-1", "8.9.1.2", "8..9", "8.9.",
		"8.9-", "8.9+", "8.9-rc..1", "8.9+build..1", "8.9-rc_1", "8.9+build/1", "8.9++a",
		"8.9\n", "\t8.9", "8.9\x00", "8.9\x7f", "8.9+é", "\u00a08.9", "8. 9", "8.9.garbage",
		"18446744073709551616.9", "8.18446744073709551616", "8.9.18446744073709551616",
		"8.9+" + strings.Repeat("a", 125),
	} {
		t.Run(input, func(t *testing.T) {
			got, err := ParseCamundaRelease(input)
			require.ErrorIs(t, err, ErrInvalidCamundaRelease)
			require.Zero(t, got)
		})
	}
}
