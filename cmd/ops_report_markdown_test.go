// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/config"
	"github.com/stretchr/testify/require"
)

// TestWriteMarkdownReportFieldUsesDashForEmptyValues verifies shared Markdown report fields stay compact and placeholder-safe.
func TestWriteMarkdownReportFieldUsesDashForEmptyValues(t *testing.T) {
	var out strings.Builder

	writeMarkdownReportField(&out, "Schema Version", "ops.example.v1")
	writeMarkdownReportField(&out, "Profile", "")

	require.Equal(t, "- Schema Version: ops.example.v1\n- Profile: -\n", out.String())
}

// TestWriteMarkdownReportListUsesNestedBullets verifies shared Markdown report lists keep empty and populated values stable.
func TestWriteMarkdownReportListUsesNestedBullets(t *testing.T) {
	var out strings.Builder

	writeMarkdownReportList(&out, "Errors", nil)
	writeMarkdownReportList(&out, "Keys", []string{"2251799813685249", "2251799813685250"})

	require.Equal(t, "- Errors: -\n- Keys:\n  - 2251799813685249\n  - 2251799813685250\n", out.String())
}

// TestFormatOpsPurgeReportTimeUsesUTCAndConfiguredOffset verifies Markdown report timestamps follow the shared report time policy.
func TestFormatOpsPurgeReportTimeUsesUTCAndConfiguredOffset(t *testing.T) {
	when := time.Date(2026, time.August, 10, 12, 34, 56, 0, time.FixedZone("CEST", 2*60*60))

	require.Empty(t, formatOpsPurgeReportTime(time.Time{}, nil))
	require.Equal(t, "2026-08-10T10:34:56.000", formatOpsPurgeReportTime(when, nil))
	require.Equal(t, "2026-08-10T10:34:56.000", formatOpsPurgeReportTime(when, &config.Config{}))
	require.Equal(t, "2026-08-10T10:34:56.000+00:00", formatOpsPurgeReportTime(when, &config.Config{App: config.App{ShowTimezoneOffset: true}}))
}

// TestWriteMarkdownTenantContextUsesSharedHumanContract verifies ops Markdown
// reports render the common context and warnings without a generic tenant label.
func TestWriteMarkdownTenantContextUsesSharedHumanContract(t *testing.T) {
	ctx := withTenantContextEvidence(newExplicitKeysTenantContext("tenant-a"), []string{"tenant-b", "tenant-a"}, 1)
	var out strings.Builder

	writeMarkdownTenantContext(&out, &ctx)

	got := out.String()
	require.Contains(t, got, "- Tenant Context: selection scope: explicit resource keys; tenant filter not applied")
	require.NotContains(t, got, "- Resource Tenants: tenant-a, tenant-b")
	require.Contains(t, got, "- Cross Tenant: true")
	require.Contains(t, got, "affected tenants: tenant-a, tenant-b")
	require.Contains(t, got, "tenant metadata is unknown for 1 target")
	require.Equal(t, 1, strings.Count(got, "affected tenants: tenant-a, tenant-b"))
	require.NotContains(t, got, string(tenant.ContextWarningUnfilteredSelection))
}

// TestOpsAuditReportMarkdownPreservesTenantEvidenceAfterEarlyHumanRendering
// proves audit Markdown ignores staged human suppression and keeps all facts.
func TestOpsAuditReportMarkdownPreservesTenantEvidenceAfterEarlyHumanRendering(t *testing.T) {
	resetTenantContextRenderFlags(t)
	cmd, output := newTenantContextRenderTestCommand()
	cfg := &config.Config{}
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
	require.Contains(t, output.String(), "selection scope: unfiltered across accessible tenants")
	require.Contains(t, output.String(), "affected tenants: tenant-a, tenant-b")

	report := enrichOpsExecuteRetentionPolicyReport(ops.RetentionAuditReport{
		SchemaVersion: "ops.retention-policy.v1",
		CommandName:   "ops execute retention-policy",
		DeletePlan:    ops.RetentionDeletePlan{TenantEvidence: evidence},
	}, cfg)
	data, err := renderOpsExecuteRetentionPolicyMarkdownReport(report, cfg)
	require.NoError(t, err)
	got := string(data)
	require.Contains(t, got, "- Tenant: -")
	require.Contains(t, got, "- Tenant Context: selection scope: unfiltered across accessible tenants")
	require.Contains(t, got, "- Unknown Target Tenants: 1")
	require.Contains(t, got, "- Cross Tenant: true")
	require.Contains(t, got, "  - affected tenants: tenant-a, tenant-b")
	require.Contains(t, got, "  - tenant metadata is unknown for 1 target")
	require.Equal(t, 1, strings.Count(got, "affected tenants: tenant-a, tenant-b"))
	require.Equal(t, 1, strings.Count(got, "tenant metadata is unknown for 1 target"))
	require.NotContains(t, got, "configuredTenantId")
	require.NotContains(t, got, "allTenants")
}
