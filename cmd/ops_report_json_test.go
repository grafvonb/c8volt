// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
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
