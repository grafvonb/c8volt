// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFormatFlatRows_AlignsColumnsWithoutTruncating protects the shared
// flat-list contract: align from observed values, but preserve every character.
func TestFormatFlatRows_AlignsColumnsWithoutTruncating(t *testing.T) {
	got := formatFlatRows([]flatRow{
		{"1", "tenant", "Short", "v1"},
		{"22", "t", "MuchLongerProcess", "v12"},
	})

	require.Equal(t, []string{
		"1  tenant Short             v1",
		"22 t      MuchLongerProcess v12",
	}, got)
}

// TestFormatFlatRows_OmitsAllEmptyOptionalColumns verifies sparse optional
// columns do not create scan-hostile gaps or trailing padding.
func TestFormatFlatRows_OmitsAllEmptyOptionalColumns(t *testing.T) {
	got := formatFlatRows([]flatRow{
		{"1", "tenant", "", "ACTIVE", ""},
		{"22", "", "", "COMPLETED", ""},
	})

	require.Equal(t, []string{
		"1  tenant ACTIVE",
		"22        COMPLETED",
	}, got)
}

// TestCompactFlatRow_SkipsEmptyFields verifies single-row rendering does not
// inherit list alignment padding or empty optional columns.
func TestCompactFlatRow_SkipsEmptyFields(t *testing.T) {
	got := compactFlatRow(flatRow{"2251799813685249", "", "Process_18qgpch", "", "v7", "ACTIVE"})

	require.Equal(t, "2251799813685249 Process_18qgpch v7 ACTIVE", got)
}
