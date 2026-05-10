// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpsHelpDocumentsGroupingCommand verifies the root ops surface is discoverable without concrete playbooks.
func TestOpsHelpDocumentsGroupingCommand(t *testing.T) {
	output := executeRootForTest(t, "ops", "--help")

	assertHelpOutputContainsAll(t, output,
		"Discover high-level operational workflows",
		"groups operational playbooks for execution, repair, and",
		"target-specific subcommands will define concrete behavior",
		"./c8volt ops --help",
		"./c8volt capabilities --json",
	)
	assertHelpOutputOmitsAll(t, output,
		"orphan-cleanup",
		"retention-policy",
		"smoke-test",
		"repair incident",
		"repair process-instance",
	)
}

// TestOpsHelpSkipsRuntimeConfigurationValidation proves help remains available without usable Camunda config.
func TestOpsHelpSkipsRuntimeConfigurationValidation(t *testing.T) {
	prevWD, err := os.Getwd()
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("C8VOLT_APP_CAMUNDA_VERSION", "not-a-supported-version")

	output := executeRootForTest(t, "ops", "--help")

	require.Contains(t, output, "Discover high-level operational workflows")
	require.Contains(t, output, "Usage:")
}

// TestOpsCommandReturnsHelpForGroupingInvocation covers the no-argument grouping behavior used by Cobra docs.
func TestOpsCommandReturnsHelpForGroupingInvocation(t *testing.T) {
	output := executeRootForTest(t, "ops")

	require.Contains(t, output, "Discover high-level operational workflows")
	require.Contains(t, output, "Usage:")
	require.Contains(t, output, "c8volt ops")
}
