// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/tenant"
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

// opsWorkflowStatusStrings keeps test assertions focused on the stable serialized tokens.
func opsWorkflowStatusStrings(statuses []OpsWorkflowStepStatus) []string {
	out := make([]string, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, status.String())
	}
	return out
}
