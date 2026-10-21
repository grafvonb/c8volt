// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCopyJSONMap snapshots nested containers and preserves large numeric values without float conversion.
func TestCopyJSONMap(t *testing.T) {
	t.Parallel()
	original := map[string]any{"number": int64(9007199254740993), "nested": map[string]any{"array": []any{"original"}}}
	copied, err := CopyJSONMap(original)
	require.NoError(t, err)
	require.Equal(t, json.Number("9007199254740993"), copied["number"])
	original["nested"].(map[string]any)["array"].([]any)[0] = "changed"
	require.Equal(t, "original", copied["nested"].(map[string]any)["array"].([]any)[0])
	for _, input := range []map[string]any{nil, {}} {
		copied, err = CopyJSONMap(input)
		require.NoError(t, err)
		require.Equal(t, input, copied)
	}
	_, err = CopyJSONMap(map[string]any{"unsupported": make(chan int)})
	require.Error(t, err)
	cyclic := map[string]any{}
	cyclic["self"] = cyclic
	_, err = CopyJSONMap(cyclic)
	require.Error(t, err)
}
