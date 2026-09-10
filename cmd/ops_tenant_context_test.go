// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/stretchr/testify/require"
)

// TestOpsTenantContextEvidenceMatrix verifies target evidence is deduplicated
// by resource key and missing tenant metadata is never filled from selection.
func TestOpsTenantContextEvidenceMatrix(t *testing.T) {
	tests := []struct {
		name         string
		evidence     process.TenantEvidence
		wantResolved []string
		wantUnknown  int
		wantCross    bool
	}{
		{
			name:         "single",
			evidence:     process.TenantEvidence{Targets: []process.TenantEvidenceTarget{{Key: "1", TenantID: "tenant-a"}}},
			wantResolved: []string{"tenant-a"},
		},
		{
			name: "multiple",
			evidence: process.TenantEvidence{Targets: []process.TenantEvidenceTarget{
				{Key: "2", TenantID: "tenant-b"},
				{Key: "1", TenantID: "tenant-a"},
			}},
			wantResolved: []string{"tenant-a", "tenant-b"},
			wantCross:    true,
		},
		{
			name:         "actual default tenant",
			evidence:     process.TenantEvidence{Targets: []process.TenantEvidenceTarget{{Key: "1", TenantID: config.DefaultTenant}}},
			wantResolved: []string{config.DefaultTenant},
		},
		{
			name:         "unknown only",
			evidence:     process.TenantEvidence{Targets: []process.TenantEvidenceTarget{{Key: "1"}}},
			wantResolved: []string{},
			wantUnknown:  1,
		},
		{
			name: "known plus unknown",
			evidence: process.TenantEvidence{Targets: []process.TenantEvidenceTarget{
				{Key: "1", TenantID: "tenant-a"},
				{Key: "2"},
			}},
			wantResolved: []string{"tenant-a"},
			wantUnknown:  1,
		},
		{
			name: "multiple plus unknown",
			evidence: process.TenantEvidence{Targets: []process.TenantEvidenceTarget{
				{Key: "1", TenantID: "tenant-a"},
				{Key: "2", TenantID: "tenant-b"},
				{Key: "3"},
			}},
			wantResolved: []string{"tenant-a", "tenant-b"},
			wantUnknown:  1,
			wantCross:    true,
		},
		{
			name: "duplicate targets keep first observation",
			evidence: process.TenantEvidence{Targets: []process.TenantEvidenceTarget{
				{Key: "1", TenantID: "tenant-a"},
				{Key: "1"},
				{Key: "2", TenantID: "tenant-a"},
				{Key: "", TenantID: "tenant-ignored"},
			}},
			wantResolved: []string{"tenant-a"},
		},
		{
			name:         "empty",
			evidence:     process.TenantEvidence{},
			wantResolved: []string{},
		},
		{
			name: "aggregate negative unknown normalizes",
			evidence: process.TenantEvidence{
				ResolvedTenantIDs:  []string{"tenant-a", "tenant-a"},
				UnknownTargetCount: -2,
			},
			wantResolved: []string{"tenant-a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := opsTenantContextWithEvidence(newDiscoveryTenantContext("tenant-configured"), tt.evidence)

			require.Equal(t, tt.wantResolved, ctx.ResolvedTenantIDs)
			require.Equal(t, tt.wantUnknown, ctx.UnknownTargetCount)
			require.Equal(t, tt.wantCross, ctx.CrossTenant)
			if tt.name == "unknown only" {
				require.NotContains(t, ctx.ResolvedTenantIDs, "tenant-configured")
			}
		})
	}
}

// TestHandleOpsTenantScopeProgressEventAttachesEvidenceUnderProtectedPolicy
// verifies output suppression does not discard validated report evidence.
func TestHandleOpsTenantScopeProgressEventAttachesEvidenceUnderProtectedPolicy(t *testing.T) {
	resetTenantContextRenderFlags(t)
	cmd, buf := newTenantContextRenderTestCommand()
	initializeTenantContextHumanRenderStages(cmd)
	attachTenantContext(cmd, newDiscoveryTenantContext("tenant-configured"))

	handleOpsTenantScopeProgressEvent(cmd, ops.ProgressEvent{
		Kind: ops.ProgressEventKindTenantScope,
		TenantScope: &ops.TenantScopeProgress{Evidence: process.TenantEvidence{Targets: []process.TenantEvidenceTarget{
			{Key: "1", TenantID: "tenant-a"},
			{Key: "2"},
		}}},
	}, ops.ProgressChannel{Mode: ops.ProgressModeJSON})

	require.Empty(t, buf.String())
	attached, ok := attachedTenantContext(cmd)
	require.True(t, ok)
	require.Equal(t, []string{"tenant-a"}, attached.ResolvedTenantIDs)
	require.Equal(t, 1, attached.UnknownTargetCount)
	state, staged := tenantContextHumanRenderStages(cmd)
	require.True(t, staged)
	require.False(t, state.selectionRendered)
	require.False(t, state.affectedRendered)
}

// TestHandleOpsTenantScopeProgressEventTreatsEmptyEvidenceAsComplete verifies a
// successful empty payload marks the affected stage without inventing output.
func TestHandleOpsTenantScopeProgressEventTreatsEmptyEvidenceAsComplete(t *testing.T) {
	resetTenantContextRenderFlags(t)
	cmd, buf := newTenantContextRenderTestCommand()
	initializeTenantContextHumanRenderStages(cmd)
	base := newDiscoveryTenantContext("tenant-configured")
	attachTenantContext(cmd, base)
	channel := ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true}

	printOpsTenantSelectionContext(cmd, base, channel)
	handleOpsTenantScopeProgressEvent(cmd, ops.ProgressEvent{
		Kind:        ops.ProgressEventKindTenantScope,
		TenantScope: &ops.TenantScopeProgress{},
	}, channel)
	renderAttachedTenantContext(cmd)

	require.Equal(t, "selection scope: tenant-configured only\n", buf.String())
	state, ok := tenantContextHumanRenderStages(cmd)
	require.True(t, ok)
	require.True(t, state.selectionRendered)
	require.True(t, state.affectedRendered)
}

// TestInitializeOpsTenantContextHumanReportingStartsFreshInvocation verifies
// reused command objects do not carry stage suppression into the next run.
func TestInitializeOpsTenantContextHumanReportingStartsFreshInvocation(t *testing.T) {
	resetTenantContextRenderFlags(t)
	cmd, buf := newTenantContextRenderTestCommand()
	cfg := &config.Config{App: config.App{Tenant: "tenant-a"}}

	initializeOpsTenantContextHumanReporting(cmd, cfg, false)
	handleOpsTenantScopeProgressEvent(cmd, ops.ProgressEvent{
		Kind: ops.ProgressEventKindTenantScope,
		TenantScope: &ops.TenantScopeProgress{Evidence: process.TenantEvidence{
			ResolvedTenantIDs: []string{"tenant-a"},
		}},
	}, ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true})
	initializeOpsTenantContextHumanReporting(cmd, cfg, false)

	require.Equal(t, "selection scope: tenant-a only\naffected tenants: tenant-a\nselection scope: tenant-a only\n", buf.String())
	state, ok := tenantContextHumanRenderStages(cmd)
	require.True(t, ok)
	require.True(t, state.selectionRendered)
	require.False(t, state.affectedRendered)
}
