// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestV810BaselineMatchesPinnedProvenance verifies the operator-facing
// baseline disclosure is sourced from the active generated-client provenance.
func TestV810BaselineMatchesPinnedProvenance(t *testing.T) {
	data, err := os.ReadFile("../internal/clients/camunda/v810/camunda/provenance.json")
	require.NoError(t, err)

	var provenance struct {
		Repository string `json:"repository"`
		Tag        string `json:"tag"`
		Commit     string `json:"commit"`
		SourceSpec string `json:"sourceSpec"`
	}
	require.NoError(t, json.Unmarshal(data, &provenance))

	baseline := V810Baseline()
	require.Equal(t, V810, baseline.Version)
	require.Equal(t, "prerelease", baseline.Status)
	require.Equal(t, provenance.Tag, baseline.Tag)
	require.Equal(t, provenance.Commit, baseline.Commit)
	require.Equal(t, provenance.Repository, baseline.Repository)
	require.Equal(t, provenance.SourceSpec, baseline.SourceSpec)
}

// TestV810BaselineReplacementKeepsSingleCompatibilityIdentity verifies later
// 8.10 baseline tags update provenance metadata without creating aliases.
func TestV810BaselineReplacementKeepsSingleCompatibilityIdentity(t *testing.T) {
	originalBaseline := v810Baseline
	t.Cleanup(func() {
		v810Baseline = originalBaseline
	})

	for _, tt := range []struct {
		name   string
		tag    string
		status string
	}{
		{name: "newer alpha", tag: "8.10.0-alpha5", status: "prerelease"},
		{name: "release candidate", tag: "8.10.0-rc1", status: "release-candidate"},
		{name: "final release", tag: "8.10.0", status: "final"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v810Baseline = CamundaBaseline{
				Version:    V810,
				Status:     tt.status,
				Tag:        tt.tag,
				Commit:     "0123456789abcdef0123456789abcdef01234567",
				Repository: originalBaseline.Repository,
				SourceSpec: originalBaseline.SourceSpec,
			}

			baseline := V810Baseline()
			_, err := NormalizeCamundaVersion(baseline.Tag)

			require.Equal(t, V810, baseline.Version)
			require.Equal(t, "8.10", baseline.Version.String())
			require.ErrorIs(t, err, ErrUnknownCamundaVersion)
			require.Equal(t, "8.7, 8.8, 8.9, 8.10", SupportedCamundaVersionsString())
			require.Equal(t, SupportedCamundaVersionsString(), ImplementedCamundaVersionsString())
		})
	}
}
