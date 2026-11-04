// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package resource

import (
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestDeleteReportsMapServiceIdentity ensures response identity is authoritative,
// without consulting or reconstructing the caller's selection in the facade.
func TestDeleteReportsMapServiceIdentity(t *testing.T) {
	t.Parallel()
	got := fromResourceDeleteResponses([]d.ResourceDeleteResponse{{Key: "pd-b", Ok: true}, {Key: "pd-a"}})
	require.Equal(t, []DeleteReport{{Key: "pd-b", Ok: true}, {Key: "pd-a"}}, got.Items)
	require.NotNil(t, fromResourceDeleteResponses(nil).Items)
}
