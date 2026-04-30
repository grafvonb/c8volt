// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package oauth2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenTargetLabel(t *testing.T) {
	require.Equal(t, "<default>", tokenTargetLabel(""))
	require.Equal(t, "<default>", tokenTargetLabel("  "))
	require.Equal(t, "camunda_api", tokenTargetLabel("camunda_api"))
}
