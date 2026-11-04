// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestOpsHelpDocumentsGroupingCommand verifies the root ops surface is discoverable without concrete playbooks.
func TestOpsHelpDocumentsGroupingCommand(t *testing.T) {
	assertCommandHelpOutput(t, []string{"ops"}, []string{"repair", "retention", "--dry-run"}, nil)
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

	require.Contains(t, output, "Run playbooks")
	require.Contains(t, output, "Usage:")
}

// TestOpsCommandReturnsHelpForGroupingInvocation covers the no-argument grouping behavior used by Cobra docs.
func TestOpsCommandReturnsHelpForGroupingInvocation(t *testing.T) {
	output := executeRootForTest(t, "ops")

	require.Contains(t, output, "Run playbooks")
	require.Contains(t, output, "Usage:")
	require.Contains(t, output, "c8volt ops")
}

// TestOpsAnalyseHelpDocumentsGroupingCommand verifies the analysis family is discoverable under both spellings.
func TestOpsAnalyseHelpDocumentsGroupingCommand(t *testing.T) {
	for _, spelling := range []string{"analyse", "analyze"} {
		assertCommandHelpOutput(t, []string{"ops", spelling}, []string{"runtime performance", "slow-process-instances"}, nil)
	}
}

// TestOpsAnalyseSlowProcessInstancesHelpDocumentsScaffold protects the initial slow-analysis command surface.
func TestOpsAnalyseSlowProcessInstancesHelpDocumentsScaffold(t *testing.T) {
	assertCommandHelpOutput(t, []string{"ops", "analyse", "slow-process-instances"}, []string{"--dur-longer", "--dur-element-longer", "--with-listeners", "--limit", "--key", "--bpmn-process-id"}, nil)
}

// TestOpsExecuteHelpDocumentsGroupingCommand verifies execute is only a discoverable parent for future playbooks.
func TestOpsExecuteHelpDocumentsGroupingCommand(t *testing.T) {
	assertCommandHelpOutput(t, []string{"ops", "execute"}, []string{"retention-policy", "smoke-test"}, nil)
}

// TestOpsExecuteCommandReturnsHelpForGroupingInvocation covers no-argument grouping behavior.
func TestOpsExecuteCommandReturnsHelpForGroupingInvocation(t *testing.T) {
	output := executeRootForTest(t, "ops", "execute")

	require.Contains(t, output, "Run predefined operational playbooks")
	require.Contains(t, output, "Usage:")
	require.Contains(t, output, "c8volt ops execute")
}

// TestOpsRepairHelpDocumentsGroupingCommand verifies repair is only a discoverable parent for target remediation workflows.
func TestOpsRepairHelpDocumentsGroupingCommand(t *testing.T) {
	assertCommandHelpOutput(t, []string{"ops", "repair"}, []string{"incident", "process-instance"}, nil)
}

// TestOpsRepairCommandReturnsHelpForGroupingInvocation covers no-argument grouping behavior.
func TestOpsRepairCommandReturnsHelpForGroupingInvocation(t *testing.T) {
	output := executeRootForTest(t, "ops", "repair")

	require.Contains(t, output, "Repair incidents and affected process instances")
	require.Contains(t, output, "Usage:")
	require.Contains(t, output, "c8volt ops repair")
}

// TestOpsRepairCommandDefinesNoTopLevelKeyFlag prevents ambiguous repair target semantics at the grouping level.
func TestOpsRepairCommandDefinesNoTopLevelKeyFlag(t *testing.T) {
	require.Nil(t, opsRepairCmd.Flags().Lookup("key"))
	require.Nil(t, opsRepairCmd.PersistentFlags().Lookup("key"))
}

// TestOpsMutationHelpDocumentsTenantContextOrder verifies every affected workflow explains pre-mutation reporting and auto-confirm semantics.
func TestOpsMutationHelpDocumentsTenantContextOrder(t *testing.T) {
	for _, command := range []*cobra.Command{opsPurgeAllProcessDefinitionsCmd, opsPurgeOrphanProcessInstancesCmd, opsPurgeProcessInstancesWithIncidentsCmd, opsExecuteRetentionPolicyCmd, opsRepairIncidentCmd, opsRepairProcessInstanceCmd} {
		require.Contains(t, command.Long, "--dry-run")
		require.Contains(t, command.Long, "--auto-confirm for unattended")
		require.Contains(t, command.Long, "--report-file")
	}
}
