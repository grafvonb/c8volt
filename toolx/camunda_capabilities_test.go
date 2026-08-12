// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSupportsFullProcessDefinitionHistoryDeletion names the exact release
// capability required before process-definition history delete mutations.
func TestSupportsFullProcessDefinitionHistoryDeletion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version CamundaVersion
		want    bool
	}{
		{version: V87, want: false},
		{version: V88, want: false},
		{version: V89, want: true},
		{version: V810, want: true},
		{version: CamundaVersion("unknown"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.version.String(), func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, SupportsFullProcessDefinitionHistoryDeletion(tt.version))
		})
	}
}
