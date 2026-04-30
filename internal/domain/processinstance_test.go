// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProcessInstanceFilterString_RendersOptionalBooleansConsistently verifies nil booleans are omitted from debug output.
func TestProcessInstanceFilterString_RendersOptionalBooleansConsistently(t *testing.T) {
	require.Equal(t, "none", fmt.Sprintf("%+v", ProcessInstanceFilter{}))
	require.NotContains(t, fmt.Sprintf("%+v", ProcessInstanceFilter{}), "<nil>")

	hasParent := true
	hasIncident := false
	got := fmt.Sprintf("%+v", ProcessInstanceFilter{
		BpmnProcessId: "order",
		HasParent:     &hasParent,
		HasIncident:   &hasIncident,
	})

	require.Equal(t, `{bpmnProcessId="order", hasParent=true, hasIncident=false}`, got)
}
