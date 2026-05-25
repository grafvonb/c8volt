// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalEnumValueReturnsCanonicalMatch(t *testing.T) {
	type testEnum string
	valid := []testEnum{"BPMN_ELEMENT", "TASK_LISTENER"}

	got, ok := CanonicalEnumValue(" bpmn_element ", valid)

	require.True(t, ok)
	require.Equal(t, testEnum("BPMN_ELEMENT"), got)
}

func TestCanonicalEnumValueRejectsUnknownAndEmpty(t *testing.T) {
	got, ok := CanonicalEnumString("", []string{"FAILED"})
	require.False(t, ok)
	require.Empty(t, got)

	got, ok = CanonicalEnumString("open", []string{"FAILED"})
	require.False(t, ok)
	require.Empty(t, got)
}
