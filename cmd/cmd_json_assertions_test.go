// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func requireJSONObject(t *testing.T, value any) map[string]any {
	t.Helper()

	got, ok := value.(map[string]any)
	require.True(t, ok, "expected JSON object")
	return got
}

func requireJSONItems(t *testing.T, value any, wantLen int) []any {
	t.Helper()

	items, ok := value.([]any)
	require.True(t, ok, "expected JSON array")
	require.Len(t, items, wantLen)
	return items
}
