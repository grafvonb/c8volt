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
