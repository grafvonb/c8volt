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
				Message: "affected tenants: tenant-a, tenant-b",
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
		require.Equal(t, "affected tenants: tenant-a, tenant-b", ctx.Warnings[0].Message)
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
				Message: "tenant metadata is unknown for 1 target",
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
	require.Equal(t, "tenant metadata is unknown for 1 target", publicEvent.Preflight.TenantContext.Warnings[0].Message)

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

// TestProgressConversions_MapCompletionFact verifies completion facts cross the
// ops facade boundary without adding command wording or losing lifecycle data.
func TestProgressConversions_MapCompletionFact(t *testing.T) {
	t.Parallel()

	affected := 0
	publicEvent := fromDomainProgressEvent(d.OpsProgressEvent{
		Kind: d.OpsProgressEventKindCompletion,
		Completion: &d.OpsCompletionProgress{
			Phase:            "deleting process-instance trees",
			CoreResource:     "process-instance root tree(s)",
			Total:            5,
			Identity:         "2251799813685251",
			Disposition:      d.OpsCompletionDispositionFailed,
			FailureDetail:    "context deadline exceeded",
			AffectedResource: "affected process instance(s)",
			AffectedCount:    &affected,
		},
	})

	require.Equal(t, ProgressEventKindCompletion, publicEvent.Kind)
	require.Equal(t, &CompletionProgress{
		Phase:            "deleting process-instance trees",
		CoreResource:     "process-instance root tree(s)",
		Total:            5,
		Identity:         "2251799813685251",
		Disposition:      CompletionDispositionFailed,
		FailureDetail:    "context deadline exceeded",
		AffectedResource: "affected process instance(s)",
		AffectedCount:    &affected,
	}, publicEvent.Completion)

	domainEvent := toDomainProgressEvent(publicEvent)
	require.Equal(t, d.OpsProgressEventKindCompletion, domainEvent.Kind)
	require.Equal(t, &d.OpsCompletionProgress{
		Phase:            "deleting process-instance trees",
		CoreResource:     "process-instance root tree(s)",
		Total:            5,
		Identity:         "2251799813685251",
		Disposition:      d.OpsCompletionDispositionFailed,
		FailureDetail:    "context deadline exceeded",
		AffectedResource: "affected process instance(s)",
		AffectedCount:    &affected,
	}, domainEvent.Completion)
}

// TestProgressConversions_MapCompletionDispositionLifecycle verifies ops
// facade conversions preserve generic lifecycle state without command verbs.
func TestProgressConversions_MapCompletionDispositionLifecycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		domain     d.OpsCompletionDisposition
		public     CompletionDisposition
		publicName string
	}{
		{
			name:       "accepted no-wait work is submitted",
			domain:     d.OpsCompletionDispositionSubmitted,
			public:     CompletionDispositionSubmitted,
			publicName: "submitted",
		},
		{
			name:       "waited work is confirmed",
			domain:     d.OpsCompletionDispositionConfirmed,
			public:     CompletionDispositionConfirmed,
			publicName: "confirmed",
		},
		{
			name:       "failed work stays failed",
			domain:     d.OpsCompletionDispositionFailed,
			public:     CompletionDispositionFailed,
			publicName: "failed",
		},
	}

	renderedVerbs := []string{"cancelled", "canceled", "deleted", "deployed", "repaired", "started", "satisfied"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			publicEvent := fromDomainProgressEvent(d.OpsProgressEvent{
				Kind: d.OpsProgressEventKindCompletion,
				Completion: &d.OpsCompletionProgress{
					Phase:       "delete",
					Identity:    "2251799813685251",
					Disposition: tt.domain,
				},
			})
			require.NotNil(t, publicEvent.Completion)
			require.Equal(t, tt.public, publicEvent.Completion.Disposition)
			require.Equal(t, tt.publicName, string(publicEvent.Completion.Disposition))
			for _, renderedVerb := range renderedVerbs {
				require.NotEqual(t, renderedVerb, string(publicEvent.Completion.Disposition))
			}

			roundTrip := toDomainProgressEvent(publicEvent)
			require.NotNil(t, roundTrip.Completion)
			require.Equal(t, tt.domain, roundTrip.Completion.Disposition)
		})
	}
}

// TestProgressConversions_PreserveUnknownCompletionAffectedCount verifies nil
// affected counts remain unavailable across both ops conversion directions.
func TestProgressConversions_PreserveUnknownCompletionAffectedCount(t *testing.T) {
	t.Parallel()

	domainEvent := toDomainProgressEvent(ProgressEvent{
		Kind: ProgressEventKindCompletion,
		Completion: &CompletionProgress{
			Phase:       "submitting process-instance cancellation",
			Identity:    "2251799813685252",
			Disposition: CompletionDispositionSubmitted,
		},
	})

	require.Equal(t, d.OpsProgressEventKindCompletion, domainEvent.Kind)
	require.NotNil(t, domainEvent.Completion)
	require.Equal(t, d.OpsCompletionDispositionSubmitted, domainEvent.Completion.Disposition)
	require.Nil(t, domainEvent.Completion.AffectedCount)

	publicEvent := fromDomainProgressEvent(domainEvent)
	require.Equal(t, ProgressEventKindCompletion, publicEvent.Kind)
	require.NotNil(t, publicEvent.Completion)
	require.Nil(t, publicEvent.Completion.AffectedCount)
}
