// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpsHelpDocumentsGroupingCommand(t *testing.T) {
	output := executeRootForTest(t, "ops", "--help")

	assertHelpOutputContainsAll(t, output,
		"Discover high-level operational workflows",
		"groups operational playbooks for execution, repair, and",
		"target-specific subcommands define concrete behavior",
		"./c8volt ops --help",
		"./c8volt ops execute --help",
		"./c8volt ops repair --help",
	)
	assertHelpOutputOmitsAll(t, output,
		"orphan-cleanup",
		"retention-policy",
		"smoke-test",
		"repair incident",
		"repair process-instance",
	)
}

func TestOpsCommandReturnsHelpForGroupingInvocation(t *testing.T) {
	output := executeRootForTest(t, "ops")

	require.Contains(t, output, "Discover high-level operational workflows")
	require.Contains(t, output, "Usage:")
	require.Contains(t, output, "c8volt ops")
}
