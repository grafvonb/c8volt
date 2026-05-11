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

// TestOpsExecuteHelpDocumentsGroupingCommand verifies execute is only a discoverable parent for future playbooks.
func TestOpsExecuteHelpDocumentsGroupingCommand(t *testing.T) {
	output := executeRootForTest(t, "ops", "execute", "--help")

	assertHelpOutputContainsAll(t, output,
		"Discover predefined operational playbooks",
		"reserved for future playbooks that discover",
		"target sets and execute existing c8volt resource actions",
		"./c8volt ops execute --help",
		"./c8volt capabilities --json",
	)
	assertHelpOutputOmitsAll(t, output,
		"orphan-cleanup",
		"retention-policy",
		"smoke-test",
	)
}

// TestOpsExecuteCommandReturnsHelpForGroupingInvocation covers no-argument grouping behavior.
func TestOpsExecuteCommandReturnsHelpForGroupingInvocation(t *testing.T) {
	output := executeRootForTest(t, "ops", "execute")

	require.Contains(t, output, "Discover predefined operational playbooks")
	require.Contains(t, output, "Usage:")
	require.Contains(t, output, "c8volt ops execute")
}

// TestOpsRepairHelpDocumentsGroupingCommand verifies repair is only a discoverable parent for future remediation workflows.
func TestOpsRepairHelpDocumentsGroupingCommand(t *testing.T) {
	output := executeRootForTest(t, "ops", "repair", "--help")

	assertHelpOutputContainsAll(t, output,
		"Discover repair and remediation workflows",
		"reserved for future workflows that repair",
		"Target-specific subcommands will define their own target semantics",
		"./c8volt ops repair --help",
		"./c8volt capabilities --json",
	)
	assertHelpOutputOmitsAll(t, output,
		"--key string",
		"--key strings",
		"repair incident",
		"repair process-instance",
	)
}

// TestOpsRepairCommandReturnsHelpForGroupingInvocation covers no-argument grouping behavior.
func TestOpsRepairCommandReturnsHelpForGroupingInvocation(t *testing.T) {
	output := executeRootForTest(t, "ops", "repair")

	require.Contains(t, output, "Discover repair and remediation workflows")
	require.Contains(t, output, "Usage:")
	require.Contains(t, output, "c8volt ops repair")
}

// TestOpsRepairCommandDefinesNoTopLevelKeyFlag prevents ambiguous repair target semantics at the grouping level.
func TestOpsRepairCommandDefinesNoTopLevelKeyFlag(t *testing.T) {
	require.Nil(t, opsRepairCmd.Flags().Lookup("key"))
	require.Nil(t, opsRepairCmd.PersistentFlags().Lookup("key"))
}
