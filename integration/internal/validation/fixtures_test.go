// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRealStateMinorSelection proves only the two release real-state minors are
// eligible and that malformed, patch, and legacy values cannot select them.
func TestRealStateMinorSelection(t *testing.T) {
	for _, minor := range []string{"8.9", "8.10"} {
		got, err := RealStateMinor(minor)
		require.NoError(t, err)
		require.Equal(t, minor, got)
	}
	for _, value := range []string{"8.8", "8.10.3", " 8.10 ", "8.1", "unknown", ""} {
		_, err := RealStateMinor(value)
		require.Error(t, err, value)
	}
}

// TestEmbeddedFixtureSelection keeps baseline fixture selection available while
// proving that missing or cross-version files never satisfy the requested minor.
func TestEmbeddedFixtureSelection(t *testing.T) {
	files := []string{
		"processdefinitions/C88_SimpleUserTask.bpmn",
		"processdefinitions/C89_SimpleUserTask.bpmn",
		"processdefinitions/C810_SimpleUserTask.bpmn",
	}
	for minor, want := range map[string]string{
		"8.8":  files[0],
		"8.9":  files[1],
		"8.10": files[2],
	} {
		got, err := SelectEmbeddedFixture(minor, "SimpleUserTask.bpmn", files)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	_, err := SelectEmbeddedFixture("8.10", "Missing.bpmn", files)
	require.ErrorContains(t, err, "no embedded Missing.bpmn fixture")
}

// TestRealStateCascadeProcessIDs proves 8.9 and 8.10 select their own checked-in
// parent/mid/leaf identities and that 8.8 cannot enter real-state cascade work.
func TestRealStateCascadeProcessIDs(t *testing.T) {
	for minor, want := range map[string]CascadeProcessIDs{
		"8.9":  {Parent: "C89_MultipleSubProcessesParent", Mid: "C89_SimpleParent", Leaf: "C89_SimpleUserTask"},
		"8.10": {Parent: "C810_MultipleSubProcessesParent", Mid: "C810_SimpleParent", Leaf: "C810_SimpleUserTask"},
	} {
		got, err := RealStateCascadeProcessIDs(minor)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	_, err := RealStateCascadeProcessIDs("8.8")
	require.ErrorContains(t, err, "does not support real-state")
}
