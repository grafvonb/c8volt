// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package toolx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFormatActiveFields verifies the shared filter formatter keeps empty filters readable.
func TestFormatActiveFields(t *testing.T) {
	require.Equal(t, "none", FormatActiveFields(nil))
	require.Equal(t, `{state=ACTIVE, bpmnProcessId="order"}`, FormatActiveFields([]string{"state=ACTIVE", `bpmnProcessId="order"`}))
}

// TestAppendFilterFields verifies optional filter fields only render when active.
func TestAppendFilterFields(t *testing.T) {
	yes := true
	no := false
	var parts []string

	parts = AppendQuotedField(parts, "empty", "")
	parts = AppendQuotedField(parts, "name", "order")
	parts = AppendInt32Field(parts, "zero", 0)
	parts = AppendInt32Field(parts, "version", 2)
	parts = AppendBoolPtrField(parts, "unset", nil)
	parts = AppendBoolPtrField(parts, "hasParent", &yes)
	parts = AppendBoolPtrField(parts, "hasIncident", &no)
	parts = AppendTrueBoolField(parts, "latestFalse", false)
	parts = AppendTrueBoolField(parts, "latest", true)
	parts = AppendRawField(parts, "state", "ACTIVE")

	require.Equal(t, []string{
		`name="order"`,
		"version=2",
		"hasParent=true",
		"hasIncident=false",
		"latest=true",
		"state=ACTIVE",
	}, parts)
}
