// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStateAcceptsCaseInsensitiveCanonicalValues(t *testing.T) {
	got, ok := ParseState(" COMPLETED ")

	require.True(t, ok)
	require.Equal(t, StateCompleted, got)
}

func TestParseStateKeepsCancelledAlias(t *testing.T) {
	got, ok := ParseState("cancelled")

	require.True(t, ok)
	require.Equal(t, StateCanceled, got)
}
