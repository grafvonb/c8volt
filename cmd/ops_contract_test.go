// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestOpsWorkflowStepStatusesMatchSharedContract protects the status vocabulary promised to future ops reports.
func TestOpsWorkflowStepStatusesMatchSharedContract(t *testing.T) {
	statuses := opsWorkflowStepStatuses()

	require.Equal(t, []OpsWorkflowStepStatus{
		OpsWorkflowStepStatusPlanned,
		OpsWorkflowStepStatusSkipped,
		OpsWorkflowStepStatusNotApplicable,
		OpsWorkflowStepStatusSubmitted,
		OpsWorkflowStepStatusConfirmed,
		OpsWorkflowStepStatusConfirmationFailed,
		OpsWorkflowStepStatusBlocked,
		OpsWorkflowStepStatusFailed,
	}, statuses)
	require.Equal(t, []string{
		"planned",
		"skipped",
		"not_applicable",
		"submitted",
		"confirmed",
		"confirmation_failed",
		"blocked",
		"failed",
	}, opsWorkflowStatusStrings(statuses))

	for _, status := range statuses {
		require.True(t, status.IsValid(), "expected %q to be a valid ops workflow status", status)
		require.Equal(t, string(status), status.String())
	}
	require.False(t, OpsWorkflowStepStatus("mutation_failed").IsValid())
}

func TestOpsWorkflowElapsedSuffixUsesApproximateDuration(t *testing.T) {
	require.Empty(t, opsWorkflowElapsedSuffix(""))
	require.Equal(t, "; elapsed: <1s", opsWorkflowElapsedSuffix((250 * time.Millisecond).String()))
	require.Equal(t, "; elapsed: 1m31s", opsWorkflowElapsedSuffix((90*time.Second + 600*time.Millisecond).String()))
	require.Equal(t, "; elapsed: about five minutes", opsWorkflowElapsedSuffix("about five minutes"))
}

// TestOpsAnalyseSlowProcessInstancesMetadataRecordsReadOnlyContract protects the analysis command's ops contract.
func TestOpsAnalyseSlowProcessInstancesMetadataRecordsReadOnlyContract(t *testing.T) {
	capability := commandCapabilityForCommand(opsAnalyseSlowProcessInstancesCmd)

	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, opsAnalyseCmd.Aliases, "analyze")
	require.Contains(t, capability.Aliases, "slow-pi")
	require.Contains(t, capability.Aliases, "spi")
	require.Contains(t, capability.AutomationNotes, "read-only analysis")
	require.Contains(t, capability.AutomationNotes, "key pipelines")
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "one-line", Supported: true})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "keys-only", Supported: true, Notes: "stdout remains one process-instance key per line with no progress or preflight text"})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "json", Supported: true, MachinePreferred: true, Notes: "stdout remains one JSON document; preflight and frozen-scope metadata are exposed as result fields"})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-listeners",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "include runtime listener jobs under matching element timeline rows",
	})
}

// TestOpsJSONReportContractPlacesTenantContextAtRoot verifies tenant context is
// a shared report object and does not reshape workflow-specific fields.
func TestOpsJSONReportContractPlacesTenantContextAtRoot(t *testing.T) {
	ctx := withTenantContextEvidence(newDiscoveryTenantContext(""), []string{"tenant-b", "tenant-a"}, 1)
	data, err := renderOpsPurgeOrphanProcessInstancesJSONReport(ops.OrphanPurgeReport{
		SchemaVersion: "ops.orphan-purge.v1",
		CommandName:   "ops purge orphan-process-instances",
		TenantContext: &ctx,
		DeletionPlan: ops.DeletionPlan{
			Status: ops.WorkflowStepStatusPlanned,
		},
		Outcome: ops.OrphanPurgeOutcomePlanned,
	})

	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, "ops.orphan-purge.v1", got["schemaVersion"])
	require.Equal(t, "ops purge orphan-process-instances", got["commandName"])
	tenantContext := requireJSONObject(t, got["tenantContext"])
	require.Equal(t, string(tenant.ContextModeDiscovery), tenantContext["mode"])
	require.Equal(t, string(tenant.ContextFilterNone), tenantContext["filter"])
	require.Equal(t, []any{"tenant-a", "tenant-b"}, tenantContext["resolvedTenantIds"])
	require.Equal(t, float64(1), tenantContext["unknownTargetCount"])
	require.Equal(t, true, tenantContext["crossTenant"])
	requireJSONItems(t, tenantContext["warnings"], 3)
	deletionPlan := requireJSONObject(t, got["deletionPlan"])
	require.Equal(t, string(ops.WorkflowStepStatusPlanned), deletionPlan["status"])
	require.NotContains(t, deletionPlan, "tenantContext")
}

// TestOpsMutationCommandsPreserveTenantProtectedModes executes every affected
// handler and proves staged tenant reporting follows the command output mode.
func TestOpsMutationCommandsPreserveTenantProtectedModes(t *testing.T) {
	tests := []struct {
		name        string
		commandPath string
		keysOnly    bool
		run         func(*testing.T, bool, ...string) (string, string, bool)
	}{
		{name: "all process definitions purge", commandPath: "ops purge all-process-definitions", run: runOpsAllProcessDefinitionsModeContract},
		{name: "orphan process instances purge", commandPath: "ops purge orphan-process-instances", keysOnly: true, run: runOpsOrphanProcessInstancesModeContract},
		{name: "incident process instances purge", commandPath: "ops purge process-instances-with-incidents", run: runOpsIncidentProcessInstancesModeContract},
		{name: "retention policy", commandPath: "ops execute retention-policy", run: runOpsRetentionPolicyModeContract},
		{name: "incident repair", commandPath: "ops repair incident", run: runOpsRepairIncidentModeContract},
		{name: "process instance repair", commandPath: "ops repair process-instance", run: runOpsRepairProcessInstanceModeContract},
	}
	modes := []struct {
		name             string
		flags            []string
		execute          bool
		protected        bool
		structuredOutput bool
	}{
		{name: "one-line", protected: false},
		{name: "verbose", flags: []string{"--verbose"}, protected: false},
		{name: "debug", flags: []string{"--debug"}, protected: false},
		{name: "json", flags: []string{"--json"}, protected: true, structuredOutput: true},
		{name: "quiet", flags: []string{"--quiet"}, protected: true},
		{name: "automation", flags: []string{"--automation"}, execute: true, protected: true},
		{name: "keys-only", flags: []string{"--keys-only"}, protected: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, mode := range modes {
				if mode.name == "keys-only" && !tc.keysOnly {
					continue
				}
				t.Run(mode.name, func(t *testing.T) {
					stdout, stderr, mutated := tc.run(t, mode.execute, mode.flags...)
					if mode.execute {
						require.True(t, mutated, "automation must retain implicit confirmation and execute the mutation")
					}
					if mode.protected {
						if !mode.structuredOutput {
							assertNoOpsTenantChatter(t, stdout)
						}
						assertNoOpsTenantChatter(t, stderr)
					} else {
						require.Contains(t, stderr, "selection scope:")
						require.Contains(t, stderr, "affected tenants:")
						require.NotContains(t, stdout, "selection scope:")
						require.NotContains(t, stdout, "affected tenants:")
					}
					if mode.structuredOutput {
						var envelope map[string]any
						require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), stdout)
						require.Equal(t, tc.commandPath, envelope["command"])
						require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
					}
					if mode.name == "keys-only" {
						lines := strings.Split(strings.TrimSpace(stdout), "\n")
						require.NotEmpty(t, lines)
						for _, line := range lines {
							require.Regexp(t, `^[0-9]+$`, line)
						}
					}
				})
			}
		})
	}
}

// TestOpsMutationCommandOutputModeContracts keeps keys-only unsupported for
// workflows whose result is an audit report rather than a key stream.
func TestOpsMutationCommandOutputModeContracts(t *testing.T) {
	commands := []struct {
		cmd      *cobra.Command
		keysOnly bool
	}{
		{cmd: opsPurgeAllProcessDefinitionsCmd},
		{cmd: opsPurgeOrphanProcessInstancesCmd, keysOnly: true},
		{cmd: opsPurgeProcessInstancesWithIncidentsCmd},
		{cmd: opsExecuteRetentionPolicyCmd},
		{cmd: opsRepairIncidentCmd},
		{cmd: opsRepairProcessInstanceCmd},
	}
	for _, item := range commands {
		capability := commandCapabilityForCommand(item.cmd)
		require.Contains(t, capability.OutputModes, OutputModeContract{Name: RenderModeOneLine.String(), Supported: true})
		require.Contains(t, capability.OutputModes, OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true})
		if item.keysOnly {
			require.Contains(t, capability.OutputModes, OutputModeContract{Name: RenderModeKeysOnly.String(), Supported: true})
		} else {
			require.NotContains(t, capability.OutputModes, OutputModeContract{Name: RenderModeKeysOnly.String(), Supported: true})
		}
	}
}

// assertNoOpsTenantChatter rejects every staged tenant line from protected streams.
func assertNoOpsTenantChatter(t *testing.T, output string) {
	t.Helper()
	for _, text := range []string{
		"selection scope:",
		"affected tenants:",
		"target(s) have no tenant metadata",
		"overrides the configured tenant filter",
	} {
		require.NotContains(t, output, text)
	}
}

// opsModeContractArgs places root mode flags before the actual command path.
func opsModeContractArgs(configPath string, modeFlags, commandArgs []string) []string {
	args := []string{"--config", configPath}
	args = append(args, modeFlags...)
	return append(args, commandArgs...)
}

// runOpsAllProcessDefinitionsModeContract exercises APD discovery and deletion through the real handler.
func runOpsAllProcessDefinitionsModeContract(t *testing.T, execute bool, modeFlags ...string) (string, string, bool) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
	t.Cleanup(srv.Close)
	commandArgs := []string{"ops", "purge", "all-process-definitions"}
	if execute {
		commandArgs = append(commandArgs, "--no-wait")
	} else {
		commandArgs = append(commandArgs, "--dry-run")
	}
	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, opsModeContractArgs(writeTestConfigForVersion(t, srv.URL, "8.9"), modeFlags, commandArgs)...)
	return stdout, stderr, len(deleted.Snapshot()) > 0
}

// runOpsOrphanProcessInstancesModeContract exercises orphan planning and deletion through the real handler.
func runOpsOrphanProcessInstancesModeContract(t *testing.T, execute bool, modeFlags ...string) (string, string, bool) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(srv.Close)
	commandArgs := []string{"ops", "purge", "orphan-process-instances"}
	if execute {
		commandArgs = append(commandArgs, "--no-wait")
	} else {
		commandArgs = append(commandArgs, "--dry-run")
	}
	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, opsModeContractArgs(writeTestConfigForVersion(t, srv.URL, "8.9"), modeFlags, commandArgs)...)
	return stdout, stderr, len(deleted.Snapshot()) > 0
}

// runOpsIncidentProcessInstancesModeContract exercises incident purge planning and deletion through the real handler.
func runOpsIncidentProcessInstancesModeContract(t *testing.T, execute bool, modeFlags ...string) (string, string, bool) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsIncidentPurgeServer(t, &requests, &deleted, false)
	t.Cleanup(srv.Close)
	commandArgs := []string{"ops", "purge", "process-instances-with-incidents"}
	if execute {
		commandArgs = append(commandArgs, "--no-wait")
	} else {
		commandArgs = append(commandArgs, "--dry-run")
	}
	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, opsModeContractArgs(writeTestConfigForVersion(t, srv.URL, "8.9"), modeFlags, commandArgs)...)
	return stdout, stderr, len(deleted.Snapshot()) > 0
}

// runOpsRetentionPolicyModeContract exercises retention planning and deletion through the real handler.
func runOpsRetentionPolicyModeContract(t *testing.T, execute bool, modeFlags ...string) (string, string, bool) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsRetentionPolicyServerWithSeed(t, &requests, &deleted)
	t.Cleanup(srv.Close)
	commandArgs := []string{"ops", "execute", "retention-policy", "--retention-days", "90"}
	if execute {
		commandArgs = append(commandArgs, "--no-wait", "--no-state-check")
	} else {
		commandArgs = append(commandArgs, "--dry-run")
	}
	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, opsModeContractArgs(writeTestConfigForVersion(t, srv.URL, "8.8"), modeFlags, commandArgs)...)
	return stdout, stderr, len(deleted.Snapshot()) > 0
}

// runOpsRepairIncidentModeContract exercises incident repair planning and mutation through the real handler.
func runOpsRepairIncidentModeContract(t *testing.T, execute bool, modeFlags ...string) (string, string, bool) {
	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)
	commandArgs := []string{"ops", "repair", "incident", "--state", "active", "--limit", "2", "--batch-size", "1"}
	if execute {
		commandArgs = append(commandArgs, "--no-wait")
	} else {
		commandArgs = append(commandArgs, "--dry-run")
	}
	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, opsModeContractArgs(writeTestConfigForVersion(t, srv.URL, "8.9"), modeFlags, commandArgs)...)
	return stdout, stderr, strings.Contains(strings.Join(requests.Snapshot(), "\n"), "POST /v2/incidents/")
}

// runOpsRepairProcessInstanceModeContract exercises process-instance-selected repair through the real handler.
func runOpsRepairProcessInstanceModeContract(t *testing.T, execute bool, modeFlags ...string) (string, string, bool) {
	var requests testx.SafeSlice[string]
	srv := newOpsRepairProcessInstanceServer(t, &requests)
	t.Cleanup(srv.Close)
	commandArgs := []string{"ops", "repair", "process-instance", "--state", "active", "--limit", "1", "--batch-size", "1"}
	if execute {
		commandArgs = append(commandArgs, "--no-wait")
	} else {
		commandArgs = append(commandArgs, "--dry-run")
	}
	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, opsModeContractArgs(writeTestConfigForVersion(t, srv.URL, "8.9"), modeFlags, commandArgs)...)
	return stdout, stderr, strings.Contains(strings.Join(requests.Snapshot(), "\n"), "POST /v2/incidents/")
}

// opsWorkflowStatusStrings keeps test assertions focused on the stable serialized tokens.
func opsWorkflowStatusStrings(statuses []OpsWorkflowStepStatus) []string {
	out := make([]string, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, status.String())
	}
	return out
}
