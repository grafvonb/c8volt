// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"testing"

	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

func TestCompactOpsExecutionOptionsSuppressesDetailUnlessVerbose(t *testing.T) {
	t.Parallel()

	compact := services.ApplyCallOptions(compactOpsExecutionOptions())
	require.True(t, compact.SuppressWorkflowDetailLogs)
	require.True(t, compact.SuppressProcessInstanceDetailLogs)

	verbose := services.ApplyCallOptions(compactOpsExecutionOptions(services.WithVerbose()))
	require.True(t, verbose.Verbose)
	require.False(t, verbose.SuppressWorkflowDetailLogs)
	require.False(t, verbose.SuppressProcessInstanceDetailLogs)
}
