// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProductionFixturePrefixMapsSupportedVersions protects the production fixture family selected for each supported compatibility line.
func TestProductionFixturePrefixMapsSupportedVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version CamundaVersion
		want    string
	}{
		{
			name:    "v87",
			version: V87,
			want:    "C87_",
		},
		{
			name:    "v88",
			version: V88,
			want:    "C88_",
		},
		{
			name:    "v89",
			version: V89,
			want:    "C89_",
		},
		{
			name:    "v810",
			version: V810,
			want:    "C810_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ProductionFixturePrefix(tt.version)

			require.True(t, ok)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestProductionFixturePrefixRejectsUnknownVersions ensures future versions do not inherit a known fixture family silently.
func TestProductionFixturePrefixRejectsUnknownVersions(t *testing.T) {
	t.Parallel()

	got, ok := ProductionFixturePrefix(CamundaVersion("8.11"))

	require.False(t, ok)
	require.Empty(t, got)
}

func TestV810FilePrefixRemainsSeparateFromFixtureCompatibility(t *testing.T) {
	t.Parallel()

	require.Equal(t, "unknown", V810.FilePrefix())
}
