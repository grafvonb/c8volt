// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBusinessIDSelectionFilterRoundTrip preserves optional presence, exact descriptions, and independent pointers in both report directions.
func TestBusinessIDSelectionFilterRoundTrip(t *testing.T) {
	t.Parallel()
	for _, bid := range []*string{nil, new(""), new(" ID,零*\n ")} {
		source := ProcessInstanceFilter{BusinessId: bid}
		internal := toDomainProcessInstanceFilter(source)
		public := fromDomainProcessInstanceFilter(internal)
		require.Equal(t, source, public)
		require.Equal(t, source.String(), internal.String())
		encoded, err := json.Marshal(public)
		require.NoError(t, err)
		var decoded ProcessInstanceFilter
		require.NoError(t, json.Unmarshal(encoded, &decoded))
		require.Equal(t, source, decoded)
		if bid == nil {
			require.NotContains(t, string(encoded), "businessId")
			continue
		}
		want := *bid
		*bid = "changed source"
		require.Equal(t, want, *internal.BusinessId)
		*internal.BusinessId = "changed internal"
		require.Equal(t, want, *public.BusinessId)
	}
}
