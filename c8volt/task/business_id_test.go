// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"encoding/json"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestUserTaskBusinessID preserves optional task identity through ordinary and effective-variable conversions without aliasing.
func TestUserTaskBusinessID(t *testing.T) {
	t.Parallel()
	for _, id := range []*string{nil, toolx.Ptr(""), toolx.Ptr("old,\"ID\"\n")} {
		original := toolx.CopyPtr(id)
		source := d.UserTask{Key: "1", ProcessInstanceKey: "2", BusinessId: id}
		got := fromDomainUserTask(source)
		enriched := fromDomainVariableEnrichedUserTask(d.VariableEnrichedUserTask{Item: source, Variables: []d.ProcessInstanceVariable{}})
		require.Equal(t, original, got.BusinessId)
		require.Equal(t, original, enriched.Item.BusinessId)
		require.NotNil(t, enriched.Variables)
		roundtrip := toDomainUserTask(got)
		require.Equal(t, original, roundtrip.BusinessId)
		raw, err := json.Marshal(got)
		require.NoError(t, err)
		var fields map[string]any
		require.NoError(t, json.Unmarshal(raw, &fields))
		if id == nil {
			require.NotContains(t, fields, "businessId")
		} else {
			require.Equal(t, *original, fields["businessId"])
			*id = "changed"
			require.Equal(t, original, got.BusinessId)
			require.Equal(t, original, enriched.Item.BusinessId)
			*got.BusinessId = "public changed"
			require.Equal(t, original, roundtrip.BusinessId)
		}
	}
}
