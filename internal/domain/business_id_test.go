// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

// TestValidateBusinessID distinguishes omission and exact opaque values using UTF-16 length.
func TestValidateBusinessID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value *string
		valid bool
	}{
		{"omitted", nil, true}, {"empty", new(""), false},
		{"space", new(" "), true}, {"opaque", new("  order,AbC  "), true},
		{"malformed", new(string([]byte{0xff})), false},
		{"one", new("a"), true}, {"BMP256", new(strings.Repeat("é", 256)), true},
		{"BMP257", new(strings.Repeat("é", 257)), false},
		{"supplementary128", new(strings.Repeat("😀", 128)), true},
		{"supplementary129", new(strings.Repeat("😀", 129)), false},
		{"mixed256", new(strings.Repeat("a", 254) + "😀"), true},
		{"mixed257", new(strings.Repeat("a", 255) + "😀"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateBusinessID(tc.value)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrValidation)
			}
		})
	}
}

// TestValidateBusinessIDAssignment preserves optional omission and exact opaque values while rejecting blank assignments.
func TestValidateBusinessIDAssignment(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value *string
		valid bool
	}{
		{"nil", nil, true}, {"empty", new(""), false}, {"blank", new(" \t\n\u2003"), false},
		{"invalid", new(string([]byte{0xff})), false}, {"padded", new("  AbC,001  "), true}, {"numeric", new("00123"), true},
		{"BMP256", new(strings.Repeat("é", 256)), true}, {"BMP257", new(strings.Repeat("é", 257)), false},
		{"supplementary256", new(strings.Repeat("😀", 128)), true}, {"supplementary257", new(strings.Repeat("😀", 128) + "a"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var before string
			if tc.value != nil {
				before = *tc.value
			}
			err := ValidateBusinessIDAssignment(tc.value)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrValidation)
			}
			if tc.value != nil {
				require.Equal(t, before, *tc.value)
			}
		})
	}
}
