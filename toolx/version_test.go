// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSupportedCamundaVersionsIncludeV810 verifies V810 is discoverable as a
// selectable compatibility line before it is promoted to the runtime factory set.
func TestSupportedCamundaVersionsIncludeV810(t *testing.T) {
	t.Parallel()

	require.Equal(t, []CamundaVersion{V87, V88, V89, V810}, SupportedCamundaVersions())
	require.Equal(t, "8.7, 8.8, 8.9, 8.10", SupportedCamundaVersionsString())
}

// TestNormalizeCamundaVersionAcceptsV810Aliases verifies the single canonical
// V810 identity accepts only the stable operator-facing aliases.
func TestNormalizeCamundaVersionAcceptsV810Aliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{name: "canonical", input: "8.10"},
		{name: "numeric", input: "810"},
		{name: "compact with prefix", input: "v810"},
		{name: "dotted with prefix", input: "v8.10"},
		{name: "trim and case", input: " V8.10 "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeCamundaVersion(tt.input)

			require.NoError(t, err)
			require.Equal(t, V810, got)
			require.Equal(t, "8.10", got.String())
		})
	}
}

// TestNormalizeCamundaVersionRejectsV810SourceTags verifies prerelease or patch
// source identifiers do not become separate c8volt configuration identities.
func TestNormalizeCamundaVersionRejectsV810SourceTags(t *testing.T) {
	t.Parallel()

	tests := []string{
		"8.10-alpha4",
		"8.10.0-alpha4",
		"8.10.0",
		"v8.10.0-alpha4",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			_, err := NormalizeCamundaVersion(input)

			require.ErrorIs(t, err, ErrUnknownCamundaVersion)
		})
	}
}

// TestCurrentCamundaVersionIsV89 verifies configuration without an explicit
// version selects the current stable runtime.
func TestCurrentCamundaVersionIsV89(t *testing.T) {
	t.Parallel()

	require.Equal(t, V89, CurrentCamundaVersion)
}

// TestImplementedCamundaVersionsIncludeCompleteV810Runtime verifies V810 is
// advertised only after all native service factories and top-level wiring exist.
func TestImplementedCamundaVersionsIncludeCompleteV810Runtime(t *testing.T) {
	t.Parallel()

	require.Equal(t, []CamundaVersion{V87, V88, V89, V810}, ImplementedCamundaVersions())
	require.Equal(t, "8.7, 8.8, 8.9, 8.10", ImplementedCamundaVersionsString())
}
