// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/config"
	"github.com/stretchr/testify/require"
)

// TestOpsWorkflowReportJSONSerializationUsesStableTokens verifies the workflow-neutral report shape serializes stable field and status names.
func TestOpsWorkflowReportJSONSerializationUsesStableTokens(t *testing.T) {
	report := OpsWorkflowReport{
		SchemaVersion:   "ops.workflow.v1",
		CommandName:     "ops example",
		Workflow:        "example",
		StartedAt:       time.Date(2026, time.August, 10, 10, 0, 0, 0, time.UTC),
		FinishedAt:      time.Date(2026, time.August, 10, 10, 1, 0, 0, time.UTC),
		Duration:        "1m0s",
		DryRun:          true,
		C8voltVersion:   "test-version",
		CamundaVersion:  "8.9",
		ProfileIdentity: "default",
		Steps: []OpsWorkflowReportStep{
			{Name: "discover", Target: "process instances", Status: OpsWorkflowStepStatusPlanned, Message: "selected 2"},
			{Name: "delete", Status: OpsWorkflowStepStatusSkipped},
		},
		Errors:  []string{"blocked"},
		Outcome: "planned",
	}

	data, err := json.Marshal(report)

	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, "ops.workflow.v1", got["schemaVersion"])
	require.Equal(t, "ops example", got["commandName"])
	require.Equal(t, "planned", got["outcome"])
	require.Equal(t, true, got["dryRun"])
	steps := got["steps"].([]any)
	require.Equal(t, "planned", steps[0].(map[string]any)["status"])
	require.Equal(t, "skipped", steps[1].(map[string]any)["status"])
	require.Len(t, got["errors"], 1)
}

// TestOpsAuditReportJSONIncludesTenantContextAndOmitsUnfilteredLegacyTenant
// verifies unfiltered ops reports are not mislabeled as default-tenant work.
func TestOpsAuditReportJSONIncludesTenantContextAndOmitsUnfilteredLegacyTenant(t *testing.T) {
	report := enrichOpsExecuteRetentionPolicyReport(ops.RetentionAuditReport{
		DeletePlan: ops.RetentionDeletePlan{
			TenantEvidence: process.TenantEvidence{
				Targets: []process.TenantEvidenceTarget{{Key: "root-a", TenantID: "tenant-a"}},
			},
		},
	}, &config.Config{})

	data, err := renderOpsExecuteRetentionPolicyJSONReport(report)

	require.NoError(t, err)
	var got struct {
		TenantID      string          `json:"tenantId"`
		TenantContext *tenant.Context `json:"tenantContext"`
	}
	require.NoError(t, json.Unmarshal(data, &got))
	require.Empty(t, got.TenantID)
	require.NotNil(t, got.TenantContext)
	require.Equal(t, tenant.ContextModeDiscovery, got.TenantContext.Mode)
	require.Equal(t, tenant.ContextFilterNone, got.TenantContext.Filter)
	require.Equal(t, []string{"tenant-a"}, got.TenantContext.ResolvedTenantIDs)
	require.Equal(t, tenant.ContextWarningUnfilteredSelection, got.TenantContext.Warnings[0].Code)
}

// TestOpsAuditReportJSONPreservesTenantEvidenceAfterEarlyHumanRendering proves
// staged human output does not prune the complete serialized tenant context.
func TestOpsAuditReportJSONPreservesTenantEvidenceAfterEarlyHumanRendering(t *testing.T) {
	resetTenantContextRenderFlags(t)
	cmd, output := newTenantContextRenderTestCommand()
	cfg := &config.Config{App: config.App{Tenant: "tenant-selected"}}
	evidence := process.TenantEvidence{Targets: []process.TenantEvidenceTarget{
		{Key: "root-b", TenantID: "tenant-b"},
		{Key: "root-unknown"},
		{Key: "root-a", TenantID: "tenant-a"},
	}}

	initializeOpsTenantContextHumanReporting(cmd, cfg, false)
	handleOpsTenantScopeProgressEvent(cmd, ops.ProgressEvent{
		Kind:        ops.ProgressEventKindTenantScope,
		TenantScope: &ops.TenantScopeProgress{Evidence: evidence},
	}, ops.ProgressChannel{DurableAllowed: true, StderrAllowed: true})
	require.Contains(t, output.String(), "selection scope: tenant-selected only")
	require.Contains(t, output.String(), "affected tenants: tenant-a, tenant-b")
	require.Contains(t, output.String(), "tenant metadata is unknown for 1 target")

	report := enrichOpsExecuteRetentionPolicyReport(ops.RetentionAuditReport{
		DeletePlan: ops.RetentionDeletePlan{TenantEvidence: evidence},
	}, cfg)
	// The derived context and command attachment remain independent from the
	// report's raw plan evidence after enrichment.
	evidence.Targets[0].TenantID = "mutated-service-evidence"
	attachTenantContext(cmd, newDiscoveryTenantContext("mutated-command-context"))

	data, err := renderOpsExecuteRetentionPolicyJSONReport(report)
	require.NoError(t, err)
	var got struct {
		TenantID      string          `json:"tenantId"`
		TenantContext *tenant.Context `json:"tenantContext"`
	}
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, "tenant-selected", got.TenantID)
	require.Equal(t, &tenant.Context{
		Mode:               tenant.ContextModeDiscovery,
		Filter:             tenant.ContextFilterNamed,
		ConfiguredTenantID: "tenant-selected",
		ResolvedTenantIDs:  []string{"tenant-a", "tenant-b"},
		UnknownTargetCount: 1,
		CrossTenant:        true,
		Warnings: []tenant.ContextWarning{
			{Code: tenant.ContextWarningMultipleTenants, Message: "affected tenants: tenant-a, tenant-b"},
			{Code: tenant.ContextWarningUnknownTargetTenants, Message: "tenant metadata is unknown for 1 target"},
		},
	}, got.TenantContext)
	require.Contains(t, string(data), "mutated-service-evidence")
	require.NotContains(t, string(data), "mutated-command-context")
	require.NotContains(t, string(data), "explicitTenantId")
	require.NotContains(t, string(data), "allTenants")
	require.Equal(t, 1, strings.Count(string(data), "unknown_target_tenants"))
}

// TestOpsRepairAuditReportJSONKeepsExplicitKeySemantics verifies direct-key
// reports retain complete evidence without claiming the configured tenant was
// used as a discovery filter or legacy tenant identifier.
func TestOpsRepairAuditReportJSONKeepsExplicitKeySemantics(t *testing.T) {
	report := enrichOpsRepairReport(ops.RepairAuditReport{
		Request: ops.RepairRequest{DiscoveryMode: ops.RepairDiscoveryModeKeyed},
		FrozenSet: ops.RepairFrozenSet{TenantEvidence: process.TenantEvidence{
			ResolvedTenantIDs:  []string{"tenant-b", "tenant-a"},
			UnknownTargetCount: 1,
		}},
	}, &config.Config{App: config.App{Tenant: "tenant-configured"}})

	data, err := renderOpsRepairJSONReport(report)
	require.NoError(t, err)
	var got struct {
		TenantID      string          `json:"tenantId"`
		TenantContext *tenant.Context `json:"tenantContext"`
	}
	require.NoError(t, json.Unmarshal(data, &got))
	require.Empty(t, got.TenantID)
	require.Equal(t, tenant.ContextModeExplicitKeys, got.TenantContext.Mode)
	require.Equal(t, tenant.ContextFilterNotApplied, got.TenantContext.Filter)
	require.Equal(t, "tenant-configured", got.TenantContext.ConfiguredTenantID)
	require.Equal(t, []string{"tenant-a", "tenant-b"}, got.TenantContext.ResolvedTenantIDs)
	require.Equal(t, 1, got.TenantContext.UnknownTargetCount)
	require.True(t, got.TenantContext.CrossTenant)
	require.ElementsMatch(t, []tenant.ContextWarningCode{
		tenant.ContextWarningMultipleTenants,
		tenant.ContextWarningUnknownTargetTenants,
	}, []tenant.ContextWarningCode{
		got.TenantContext.Warnings[0].Code,
		got.TenantContext.Warnings[1].Code,
	})
}
