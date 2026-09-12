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
	output := executeRootForTest(t, "ops", "--help")

	assertHelpOutputContainsAll(t, output,
		"Run operational playbooks",
		"Run operational playbooks for analysis, retention, purge, repair, and cluster smoke testing",
		"Choose a subcommand for a specific workflow",
		"./c8volt ops --help",
		"./c8volt capabilities --json",
	)
	assertHelpOutputOmitsAll(t, output,
		"orphan-cleanup",
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

	require.Contains(t, output, "Run operational playbooks")
	require.Contains(t, output, "Usage:")
}

// TestOpsCommandReturnsHelpForGroupingInvocation covers the no-argument grouping behavior used by Cobra docs.
func TestOpsCommandReturnsHelpForGroupingInvocation(t *testing.T) {
	output := executeRootForTest(t, "ops")

	require.Contains(t, output, "Run operational playbooks")
	require.Contains(t, output, "Usage:")
	require.Contains(t, output, "c8volt ops")
}

// TestOpsAnalyseHelpDocumentsGroupingCommand verifies the analysis family is discoverable under both spellings.
func TestOpsAnalyseHelpDocumentsGroupingCommand(t *testing.T) {
	output := executeRootForTest(t, "ops", "analyse", "--help")

	assertHelpOutputContainsAll(t, output,
		"Discover read-only operational analyses",
		"inspection workflows that combine existing runtime resources",
		"./c8volt ops analyse --help",
		"./c8volt ops analyse slow-process-instances --help",
	)

	aliasOutput := executeRootForTest(t, "ops", "analyze", "--help")
	require.Contains(t, aliasOutput, "Discover read-only operational analyses")
}

// TestOpsAnalyseSlowProcessInstancesHelpDocumentsScaffold protects the initial slow-analysis command surface.
func TestOpsAnalyseSlowProcessInstancesHelpDocumentsScaffold(t *testing.T) {
	output := executeRootForTest(t, "ops", "analyse", "slow-process-instances", "--help")

	assertHelpOutputContainsAll(t, output,
		"Analyse process-instance and runtime-element durations",
		"without changing cluster state",
		"--key strings",
		"--bpmn-process-id string",
		"--pd-key string",
		"--state string",
		"--no-incidents-only",
		"--batch-size int32",
		"--limit int32",
		"--element-id string",
		"--dur-longer string",
		"--dur-element-longer string",
		"Durations use Go syntax",
		"Calendar units such as 1d are not supported",
		"./c8volt get process-instance --state active --keys-only | ./c8volt ops analyse slow-process-instances -",
	)
	assertHelpOutputOmitsAll(t, output, "--duration-after", "--incidents-only")
}

// TestOpsExecuteHelpDocumentsGroupingCommand verifies execute is only a discoverable parent for future playbooks.
func TestOpsExecuteHelpDocumentsGroupingCommand(t *testing.T) {
	output := executeRootForTest(t, "ops", "execute", "--help")

	assertHelpOutputContainsAll(t, output,
		"Run predefined operational playbooks",
		"Choose retention-policy",
		"./c8volt ops execute --help",
		"./c8volt ops execute retention-policy --retention-days 90 --dry-run",
		"./c8volt ops execute smoke-test --dry-run",
		"./c8volt capabilities --json",
	)
	assertHelpOutputOmitsAll(t, output,
		"orphan-cleanup",
	)
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
	output := executeRootForTest(t, "ops", "repair", "--help")

	assertHelpOutputContainsAll(t, output,
		"Repair incidents and affected process instances",
		"Choose a target command",
		"resolve incidents, and verify recovery",
		"./c8volt ops repair --help",
		"./c8volt capabilities --json",
	)
	assertHelpOutputOmitsAll(t, output,
		"--key string",
		"--key strings",
	)
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
	tests := []struct {
		name    string
		command *cobra.Command
	}{
		{name: "all process definitions purge", command: opsPurgeAllProcessDefinitionsCmd},
		{name: "orphan process instances purge", command: opsPurgeOrphanProcessInstancesCmd},
		{name: "incident process instances purge", command: opsPurgeProcessInstancesWithIncidentsCmd},
		{name: "retention policy", command: opsExecuteRetentionPolicyCmd},
		{name: "incident repair", command: opsRepairIncidentCmd},
		{name: "process instance repair", command: opsRepairProcessInstanceCmd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Contains(t, tt.command.Long, "--auto-confirm or --automation for unattended")
		})
	}
}
