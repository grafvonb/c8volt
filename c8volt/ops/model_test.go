// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"encoding/json"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/tenant"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestAuditReports_TenantContextJSONContract verifies the public ops audit
// models serialize one optional nested tenant-context object and omit stale
// legacy tenantId values for unfiltered discovery.
func TestAuditReports_TenantContextJSONContract(t *testing.T) {
	t.Parallel()

	ctx := tenant.Context{
		Mode:              tenant.ContextModeDiscovery,
		Filter:            tenant.ContextFilterNone,
		ResolvedTenantIDs: []string{"tenant-a"},
		Warnings: []tenant.ContextWarning{
			{
				Code:    tenant.ContextWarningUnfilteredSelection,
				Message: "selection scope: unfiltered across accessible tenants",
			},
		},
	}

	tests := []struct {
		name   string
		report any
	}{
		{name: "retention", report: RetentionAuditReport{TenantContext: &ctx}},
		{name: "orphan purge", report: OrphanPurgeReport{TenantContext: &ctx}},
		{name: "incident purge", report: IncidentPurgeReport{TenantContext: &ctx}},
		{name: "all process definitions purge", report: AllProcessDefinitionsPurgeReport{TenantContext: &ctx}},
		{name: "repair", report: RepairAuditReport{TenantContext: &ctx}},
		{name: "smoke test", report: SmokeTestAuditReport{TenantContext: &ctx}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			raw, err := json.Marshal(tt.report)
			require.NoError(t, err)

			var got map[string]any
			require.NoError(t, json.Unmarshal(raw, &got))
			require.NotContains(t, got, "tenantId")
			tenantContext := got["tenantContext"].(map[string]any)
			require.Equal(t, "discovery", tenantContext["mode"])
			require.Equal(t, "none", tenantContext["filter"])
			require.Equal(t, []any{"tenant-a"}, tenantContext["resolvedTenantIds"])
			require.NotContains(t, tenantContext, "targetTenantId")
		})
	}
}

// TestAuditReports_KeepTruthfulLegacyTenantID verifies the deprecated tenantId
// field remains available when a report can still populate it truthfully.
func TestAuditReports_KeepTruthfulLegacyTenantID(t *testing.T) {
	t.Parallel()

	ctx := tenant.Context{
		Mode:               tenant.ContextModeDiscovery,
		Filter:             tenant.ContextFilterNamed,
		ConfiguredTenantID: "tenant-a",
		ResolvedTenantIDs:  []string{"tenant-a"},
	}

	raw, err := json.Marshal(RetentionAuditReport{
		TenantID:      "tenant-a",
		TenantContext: &ctx,
	})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Equal(t, "tenant-a", got["tenantId"])
	tenantContext := got["tenantContext"].(map[string]any)
	require.Equal(t, "discovery", tenantContext["mode"])
	require.Equal(t, "named", tenantContext["filter"])
	require.Equal(t, "tenant-a", tenantContext["configuredTenantId"])
	require.Equal(t, []any{"tenant-a"}, tenantContext["resolvedTenantIds"])
}

// TestFromDomainAuditReports_CopyTenantContext verifies every ops report
// converter uses the common nested object and copies mutable evidence slices.
func TestFromDomainAuditReports_CopyTenantContext(t *testing.T) {
	t.Parallel()

	domainCtx := d.TenantContext{
		Mode:               d.TenantContextModeDiscovery,
		Filter:             d.TenantContextFilterNamed,
		ConfiguredTenantID: "tenant-a",
		ResolvedTenantIDs:  []string{"tenant-a", "tenant-b"},
		CrossTenant:        true,
		Warnings: []d.TenantContextWarning{
			{
				Code:    d.TenantContextWarningMultipleTenants,
				Message: "WARNING: resources from multiple tenants will be affected: tenant-a, tenant-b",
			},
		},
	}

	got := []*tenant.Context{
		fromDomainRetentionAuditReport(d.RetentionAuditReport{TenantContext: &domainCtx}).TenantContext,
		fromDomainOrphanPurgeReport(d.OrphanPurgeReport{TenantContext: &domainCtx}).TenantContext,
		fromDomainIncidentPurgeReport(d.IncidentPurgeReport{TenantContext: &domainCtx}).TenantContext,
		fromDomainAllProcessDefinitionsPurgeReport(d.AllProcessDefinitionsPurgeReport{TenantContext: &domainCtx}).TenantContext,
		fromDomainRepairAuditReport(d.OpsRepairAuditReport{TenantContext: &domainCtx}).TenantContext,
		fromDomainSmokeTestAuditReport(d.SmokeTestAuditReport{TenantContext: &domainCtx}).TenantContext,
	}

	domainCtx.ResolvedTenantIDs[0] = "changed"
	domainCtx.Warnings[0].Message = "changed"

	for _, ctx := range got {
		require.NotNil(t, ctx)
		require.Equal(t, tenant.ContextModeDiscovery, ctx.Mode)
		require.Equal(t, tenant.ContextFilterNamed, ctx.Filter)
		require.Equal(t, "tenant-a", ctx.ConfiguredTenantID)
		require.Equal(t, []string{"tenant-a", "tenant-b"}, ctx.ResolvedTenantIDs)
		require.Equal(t, "WARNING: resources from multiple tenants will be affected: tenant-a, tenant-b", ctx.Warnings[0].Message)
	}
}

// TestProgressConversions_CopyTenantContext verifies progress callbacks carry
// tenant context through both facade conversion directions without aliasing.
func TestProgressConversions_CopyTenantContext(t *testing.T) {
	t.Parallel()

	domainCtx := d.TenantContext{
		Mode:              d.TenantContextModeExplicitKeys,
		Filter:            d.TenantContextFilterNotApplied,
		ResolvedTenantIDs: []string{"tenant-b"},
		Warnings: []d.TenantContextWarning{
			{
				Code:    d.TenantContextWarningUnknownTargetTenants,
				Message: "WARNING: tenant metadata is unknown for 1 target",
			},
		},
	}

	publicEvent := fromDomainProgressEvent(d.OpsProgressEvent{
		Kind: d.OpsProgressEventKindPreflight,
		Preflight: &d.OpsPreflightScope{
			Phase:         "preflight",
			TenantContext: &domainCtx,
		},
	})
	require.NotNil(t, publicEvent.Preflight.TenantContext)
	domainCtx.ResolvedTenantIDs[0] = "changed"
	domainCtx.Warnings[0].Message = "changed"
	require.Equal(t, []string{"tenant-b"}, publicEvent.Preflight.TenantContext.ResolvedTenantIDs)
	require.Equal(t, "WARNING: tenant metadata is unknown for 1 target", publicEvent.Preflight.TenantContext.Warnings[0].Message)

	publicCtx := tenant.Context{
		Mode:              tenant.ContextModeCreation,
		Filter:            tenant.ContextFilterNotApplicable,
		TargetTenantID:    "<default>",
		ResolvedTenantIDs: []string{"<default>"},
	}
	domainEvent := toDomainProgressEvent(ProgressEvent{
		Kind: ProgressEventKindPreflight,
		Preflight: &PreflightScope{
			Phase:         "preflight",
			TenantContext: &publicCtx,
		},
	})
	require.NotNil(t, domainEvent.Preflight.TenantContext)
	publicCtx.ResolvedTenantIDs[0] = "changed"
	require.Equal(t, []string{"<default>"}, domainEvent.Preflight.TenantContext.ResolvedTenantIDs)
}
