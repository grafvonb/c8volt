// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"testing"

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
				Message: "WARNING: resources from multiple tenants will be affected: tenant-a, tenant-b",
			},
			{
				Code:    tenant.ContextWarningUnknownTargetTenants,
				Message: "WARNING: tenant metadata is unknown for 1 target",
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
		"WARNING: resources from multiple tenants will be affected: tenant-a, tenant-b\n"+
		"WARNING: tenant metadata is unknown for 2 targets\n", buf.String())
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

			renderTenantContext(cmd, withTenantContextEvidence(newDiscoveryTenantContext(""), []string{"tenant-a", "tenant-b"}, 1))

			require.Empty(t, buf.String())
		})
	}
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
