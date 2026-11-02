// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestBusinessIDSelectionRequestCopy prevents caller changes from rewriting a native task selection and retains omitted versus empty input.
func TestBusinessIDSelectionRequestCopy(t *testing.T) {
	t.Parallel()
	for _, bid := range []*string{nil, new(""), new(" OLD,零* ")} {
		source := SearchRequest{BusinessId: bid}
		copied := toDomainSearchRequest(source)
		encoded, err := json.Marshal(source)
		require.NoError(t, err)
		var decoded SearchRequest
		require.NoError(t, json.Unmarshal(encoded, &decoded))
		require.Equal(t, source, decoded)
		if bid == nil {
			require.Nil(t, copied.BusinessId)
			require.NotContains(t, string(encoded), "businessId")
			continue
		}
		want := *bid
		*bid = "new PI identity"
		require.Equal(t, want, *copied.BusinessId)
	}
}
