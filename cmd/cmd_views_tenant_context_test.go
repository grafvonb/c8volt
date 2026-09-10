// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestNewTenantContexts_UseOperationSpecificSemantics verifies command-owned
// base contexts distinguish discovery, creation, and explicit-key meanings.
func TestNewTenantContexts_UseOperationSpecificSemantics(t *testing.T) {
	require.Equal(t, tenant.Context{
		Mode:              tenant.ContextModeDiscovery,
		Filter:            tenant.ContextFilterNone,
		ResolvedTenantIDs: []string{},
		Warnings: []tenant.ContextWarning{
			{
				Code:    tenant.ContextWarningUnfilteredSelection,
				Message: "selection scope: unfiltered across accessible tenants",
			},
		},
	}, newDiscoveryTenantContext(""))

	require.Equal(t, tenant.Context{
		Mode:               tenant.ContextModeDiscovery,
		Filter:             tenant.ContextFilterNamed,
		ConfiguredTenantID: "tenant-a",
		ResolvedTenantIDs:  []string{},
	}, newDiscoveryTenantContext("tenant-a"))

	require.Equal(t, tenant.Context{
		Mode:              tenant.ContextModeCreation,
		Filter:            tenant.ContextFilterNotApplicable,
		TargetTenantID:    "<default>",
		ResolvedTenantIDs: []string{},
	}, newCreationTenantContext("<default>"))

	require.Equal(t, tenant.Context{
		Mode:               tenant.ContextModeExplicitKeys,
		Filter:             tenant.ContextFilterNotApplied,
		ConfiguredTenantID: "tenant-a",
		ResolvedTenantIDs:  []string{},
	}, newExplicitKeysTenantContext("tenant-a"))
}

// TestWithTenantContextEvidence_NormalizesWarnings keeps resolved evidence
// sorted, deduplicated, and appended after the base semantic warning.
func TestWithTenantContextEvidence_NormalizesWarnings(t *testing.T) {
	got := withTenantContextEvidence(newDiscoveryTenantContext(""), []string{"tenant-b", "", "tenant-a", "tenant-b"}, 1)

	require.Equal(t, tenant.Context{
		Mode:               tenant.ContextModeDiscovery,
		Filter:             tenant.ContextFilterNone,
		ResolvedTenantIDs:  []string{"tenant-a", "tenant-b"},
		UnknownTargetCount: 1,
		CrossTenant:        true,
		Warnings: []tenant.ContextWarning{
			{
				Code:    tenant.ContextWarningUnfilteredSelection,
				Message: "selection scope: unfiltered across accessible tenants",
			},
			{
				Code:    tenant.ContextWarningMultipleTenants,
				Message: "affected tenants: tenant-a, tenant-b",
			},
			{
				Code:    tenant.ContextWarningUnknownTargetTenants,
				Message: "tenant metadata is unknown for 1 target",
			},
		},
	}, got)
}

// TestRenderTenantContextHumanExactWordingAndOrder verifies compact preflight
// output preserves the shared wording contract before a mutation.
func TestRenderTenantContextHumanExactWordingAndOrder(t *testing.T) {
	resetTenantContextRenderFlags(t)
	cmd, buf := newTenantContextRenderTestCommand()
	ctx := withTenantContextEvidence(newExplicitKeysTenantContext("tenant-a"), []string{"tenant-b", "tenant-a"}, 2)

	renderTenantContext(cmd, ctx)

	require.Equal(t, ""+
		"selection scope: explicit resource keys; tenant filter not applied\n"+
		"affected tenants: tenant-a, tenant-b\n"+
		"tenant metadata is unknown for 2 targets\n", buf.String())
}

// TestTenantContextHumanLinesClassifyTenantOverridesAndAffectedTenants verifies
// override provenance and multi-tenant evidence are classified once before
// renderers choose stdout, stderr, or logger-backed channels.
func TestTenantContextHumanLinesClassifyTenantOverridesAndAffectedTenants(t *testing.T) {
	tests := []struct {
		name       string
		provenance tenantOverrideProvenance
		ctx        tenant.Context
		want       []tenantContextHumanLine
	}{
		{
			name:       "absent flag has no override chatter",
			provenance: tenantOverrideProvenance{},
			ctx:        newDiscoveryTenantContext("tenant-a"),
			want: []tenantContextHumanLine{
				{Text: "selection scope: tenant-a only"},
			},
		},
		{
			name: "equal explicit flag stays silent",
			provenance: tenantOverrideProvenance{
				ConfiguredTenantID: "tenant-a",
				ExplicitTenantID:   "tenant-a",
				Explicit:           true,
			},
			ctx: newDiscoveryTenantContext("tenant-a"),
			want: []tenantContextHumanLine{
				{Text: "selection scope: tenant-a only"},
			},
		},
		{
			name: "named to empty warns about unfiltered selection",
			provenance: tenantOverrideProvenance{
				ConfiguredTenantID: "tenant-a",
				ExplicitTenantID:   "",
				Explicit:           true,
			},
			ctx: newDiscoveryTenantContext(""),
			want: []tenantContextHumanLine{
				{Text: "configured tenant: tenant-a"},
				{Text: `--tenant "" overrides the configured tenant filter; selection is unfiltered`, Warn: true},
				{Text: "selection scope: unfiltered across accessible tenants"},
			},
		},
		{
			name: "named to different is informational",
			provenance: tenantOverrideProvenance{
				ConfiguredTenantID: "tenant-a",
				ExplicitTenantID:   "tenant-b",
				Explicit:           true,
			},
			ctx: newDiscoveryTenantContext("tenant-b"),
			want: []tenantContextHumanLine{
				{Text: "configured tenant: tenant-a"},
				{Text: `--tenant "tenant-b" overrides configured tenant filter`},
				{Text: "selection scope: tenant-b only"},
			},
		},
		{
			name: "empty to named is informational",
			provenance: tenantOverrideProvenance{
				ConfiguredTenantID: "",
				ExplicitTenantID:   "tenant-a",
				Explicit:           true,
			},
			ctx: newDiscoveryTenantContext("tenant-a"),
			want: []tenantContextHumanLine{
				{Text: "configured tenant: none"},
				{Text: `--tenant "tenant-a" sets the tenant filter`},
				{Text: "selection scope: tenant-a only"},
			},
		},
		{
			name: "all tenants from named config warns before unfiltered scope",
			provenance: tenantOverrideProvenance{
				ConfiguredTenantID: "tenant-a",
				AllTenants:         true,
			},
			ctx: newDiscoveryTenantContext(""),
			want: []tenantContextHumanLine{
				{Text: "configured tenant: tenant-a"},
				{Text: "--all-tenants overrides the configured tenant filter; selection is unfiltered", Warn: true},
				{Text: "selection scope: unfiltered across accessible tenants"},
			},
		},
		{
			name: "all tenants from already empty config has no override chatter",
			provenance: tenantOverrideProvenance{
				AllTenants: true,
			},
			ctx: newDiscoveryTenantContext(""),
			want: []tenantContextHumanLine{
				{Text: "selection scope: unfiltered across accessible tenants"},
			},
		},
		{
			name: "multiple tenants use one warning-level affected summary",
			ctx:  withTenantContextEvidence(newExplicitKeysTenantContext("tenant-a"), []string{"tenant-b", "tenant-a"}, 1),
			want: []tenantContextHumanLine{
				{Text: "selection scope: explicit resource keys; tenant filter not applied"},
				{Text: "affected tenants: tenant-a, tenant-b", Warn: true},
				{Text: "tenant metadata is unknown for 1 target", Warn: true},
			},
		},
		{
			name: "single tenant stays informational",
			ctx:  withTenantContextEvidence(newExplicitKeysTenantContext("tenant-a"), []string{"tenant-b"}, 0),
			want: []tenantContextHumanLine{
				{Text: "selection scope: explicit resource keys; tenant filter not applied"},
				{Text: "affected tenants: tenant-b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _ := newTenantContextRenderTestCommand()
			cmd.SetContext(tt.provenance.ToContext(context.Background()))

			require.Equal(t, tt.want, tenantContextHumanLines(cmd, tt.ctx))
		})
	}
}

// TestRenderTenantContext_AllTenantsWarningRendersOnce verifies the broadening
// warning uses the shared render-once guard with the effective unfiltered scope.
func TestRenderTenantContext_AllTenantsWarningRendersOnce(t *testing.T) {
	resetTenantContextRenderFlags(t)
	cmd, buf := newTenantContextRenderTestCommand()
	cmd.SetContext(tenantOverrideProvenance{
		ConfiguredTenantID: "tenant-a",
		AllTenants:         true,
	}.ToContext(context.Background()))
	ctx := newDiscoveryTenantContext("")

	renderTenantContext(cmd, ctx)
	renderTenantContext(cmd, ctx)

	require.Equal(t, ""+
		"configured tenant: tenant-a\n"+
		"--all-tenants overrides the configured tenant filter; selection is unfiltered\n"+
		"selection scope: unfiltered across accessible tenants\n", buf.String())
}

// TestRenderTenantContextHumanModeLines verifies each operation mode receives
// its distinct label instead of reusing the default-tenant display.
func TestRenderTenantContextHumanModeLines(t *testing.T) {
	tests := []struct {
		name string
		ctx  tenant.Context
		want string
	}{
		{name: "named discovery", ctx: newDiscoveryTenantContext("tenant-a"), want: "selection scope: tenant-a only\n"},
		{name: "unfiltered discovery", ctx: newDiscoveryTenantContext(""), want: "selection scope: unfiltered across accessible tenants\n"},
		{name: "default creation", ctx: newCreationTenantContext("<default>"), want: "creation target: default tenant\n"},
		{name: "configuration none", ctx: newConfigurationTenantContext(""), want: "selection scope: unfiltered across accessible tenants\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTenantContextRenderFlags(t)
			cmd, buf := newTenantContextRenderTestCommand()

			renderTenantContext(cmd, tt.ctx)

			require.Equal(t, tt.want, buf.String())
		})
	}
}

// TestRenderTenantContextProtectedModesStaySilent keeps tenant diagnostics out
// of JSON, keys-only, and quiet stdout contracts.
func TestRenderTenantContextProtectedModesStaySilent(t *testing.T) {
	tests := []struct {
		name  string
		setup func()
	}{
		{name: "json", setup: func() { flagViewAsJson = true }},
		{name: "keys only", setup: func() { flagViewKeysOnly = true }},
		{name: "quiet", setup: func() { flagQuiet = true }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTenantContextRenderFlags(t)
			tt.setup()
			cmd, buf := newTenantContextRenderTestCommand()
			cmd.SetContext(tenantOverrideProvenance{
				ConfiguredTenantID: "tenant-a",
				AllTenants:         true,
			}.ToContext(context.Background()))

			renderTenantContext(cmd, withTenantContextEvidence(newDiscoveryTenantContext(""), []string{"tenant-a", "tenant-b"}, 1))

			require.Empty(t, buf.String())
		})
	}
}

// TestStagedTenantContextSelectionThenAffectedRendersOnce verifies separate
// callbacks cannot repeat either semantic stage or the final attached view.
func TestStagedTenantContextSelectionThenAffectedRendersOnce(t *testing.T) {
	resetTenantContextRenderFlags(t)
	cmd, buf := newTenantContextRenderTestCommand()
	initializeTenantContextHumanRenderStages(cmd)
	channel := ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true}
	selection := newDiscoveryTenantContext("tenant-a")
	affected := withTenantContextEvidence(selection, []string{"tenant-b", "tenant-a"}, 1)

	attachTenantContext(cmd, selection)
	printOpsTenantSelectionContext(cmd, selection, channel)
	attachTenantContext(cmd, affected)
	printOpsTenantAffectedContext(cmd, affected, channel)
	printOpsTenantAffectedContext(cmd, affected, channel)
	renderAttachedTenantContext(cmd)

	require.Equal(t, ""+
		"selection scope: tenant-a only\n"+
		"affected tenants: tenant-a, tenant-b\n"+
		"tenant metadata is unknown for 1 target\n", buf.String())
	state, ok := tenantContextHumanRenderStages(cmd)
	require.True(t, ok)
	require.True(t, state.selectionRendered)
	require.True(t, state.affectedRendered)
}

// TestStagedTenantContextEmptyAffectedScopeCompletesSilently verifies a
// validated empty scope is not mistaken for missing evidence at final output.
func TestStagedTenantContextEmptyAffectedScopeCompletesSilently(t *testing.T) {
	resetTenantContextRenderFlags(t)
	cmd, buf := newTenantContextRenderTestCommand()
	initializeTenantContextHumanRenderStages(cmd)
	channel := ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true}
	ctx := newDiscoveryTenantContext("tenant-a")
	attachTenantContext(cmd, ctx)

	printOpsTenantSelectionContext(cmd, ctx, channel)
	printOpsTenantAffectedContext(cmd, ctx, channel)
	renderAttachedTenantContext(cmd)

	require.Equal(t, "selection scope: tenant-a only\n", buf.String())
	state, ok := tenantContextHumanRenderStages(cmd)
	require.True(t, ok)
	require.True(t, state.selectionRendered)
	require.True(t, state.affectedRendered)
}

// TestRenderAttachedTenantContextCompletesOnlyMissingStages verifies final
// rendering fills a partial staged view while a complete view stays silent.
func TestRenderAttachedTenantContextCompletesOnlyMissingStages(t *testing.T) {
	resetTenantContextRenderFlags(t)
	channel := ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true}
	ctx := withTenantContextEvidence(newExplicitKeysTenantContext("tenant-a"), []string{"tenant-b"}, 0)

	t.Run("partial", func(t *testing.T) {
		cmd, buf := newTenantContextRenderTestCommand()
		initializeTenantContextHumanRenderStages(cmd)
		attachTenantContext(cmd, ctx)
		printOpsTenantSelectionContext(cmd, ctx, channel)

		renderAttachedTenantContext(cmd)

		require.Equal(t, ""+
			"selection scope: explicit resource keys; tenant filter not applied\n"+
			"affected tenants: tenant-b\n", buf.String())
	})

	t.Run("complete", func(t *testing.T) {
		cmd, buf := newTenantContextRenderTestCommand()
		initializeTenantContextHumanRenderStages(cmd)
		attachTenantContext(cmd, ctx)
		printOpsTenantSelectionContext(cmd, ctx, channel)
		printOpsTenantAffectedContext(cmd, ctx, channel)
		beforeFinal := buf.String()

		renderAttachedTenantContext(cmd)

		require.Equal(t, beforeFinal, buf.String())
	})
}

// TestStagedTenantContextProtectedChannelsDoNotMarkRendered verifies a
// suppressed stage remains available to a later permitted final renderer.
func TestStagedTenantContextProtectedChannelsDoNotMarkRendered(t *testing.T) {
	resetTenantContextRenderFlags(t)
	cmd, buf := newTenantContextRenderTestCommand()
	initializeTenantContextHumanRenderStages(cmd)
	ctx := withTenantContextEvidence(newDiscoveryTenantContext("tenant-a"), []string{"tenant-a"}, 0)
	attachTenantContext(cmd, ctx)

	printOpsTenantSelectionContext(cmd, ctx, ops.ProgressChannel{Mode: ops.ProgressModeJSON})
	printOpsTenantAffectedContext(cmd, ctx, ops.ProgressChannel{Mode: ops.ProgressModeQuiet})

	state, ok := tenantContextHumanRenderStages(cmd)
	require.True(t, ok)
	require.False(t, state.selectionRendered)
	require.False(t, state.affectedRendered)
	require.Empty(t, buf.String())

	renderAttachedTenantContext(cmd)
	require.Equal(t, "selection scope: tenant-a only\naffected tenants: tenant-a\n", buf.String())
}

// TestInitializeTenantContextHumanRenderStagesResetsCommandExecution verifies
// reused Cobra commands receive fresh suppression state for every execution.
func TestInitializeTenantContextHumanRenderStagesResetsCommandExecution(t *testing.T) {
	resetTenantContextRenderFlags(t)
	cmd, buf := newTenantContextRenderTestCommand()
	channel := ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true}
	ctx := newDiscoveryTenantContext("tenant-a")

	initializeTenantContextHumanRenderStages(cmd)
	printOpsTenantSelectionContext(cmd, ctx, channel)
	initializeTenantContextHumanRenderStages(cmd)
	printOpsTenantSelectionContext(cmd, ctx, channel)

	require.Equal(t, "selection scope: tenant-a only\nselection scope: tenant-a only\n", buf.String())
}

// TestRenderTenantContextLegacyCreationStillRendersFullContext verifies staged
// ops support does not change unrelated creation-mode rendering.
func TestRenderTenantContextLegacyCreationStillRendersFullContext(t *testing.T) {
	resetTenantContextRenderFlags(t)
	cmd, buf := newTenantContextRenderTestCommand()
	ctx := withTenantContextEvidence(newCreationTenantContext("<default>"), []string{"<default>"}, 0)

	renderTenantContext(cmd, ctx)
	renderTenantContext(cmd, ctx)

	require.Equal(t, "creation target: default tenant\naffected tenants: <default>\n", buf.String())
}

// newTenantContextRenderTestCommand captures tenant-context renderer output
// without constructing the global command tree.
func newTenantContextRenderTestCommand() (*cobra.Command, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cmd := &cobra.Command{Use: "tenant-context"}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd, buf
}

// resetTenantContextRenderFlags isolates global output-mode flags used by
// tenant-context renderer tests.
func resetTenantContextRenderFlags(t *testing.T) {
	t.Helper()
	prevJSON := flagViewAsJson
	prevKeysOnly := flagViewKeysOnly
	prevQuiet := flagQuiet
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
		flagViewKeysOnly = prevKeysOnly
		flagQuiet = prevQuiet
	})
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagQuiet = false
}
