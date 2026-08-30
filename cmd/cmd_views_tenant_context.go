// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"

	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/spf13/cobra"
)

// renderTenantContext writes operation-specific tenant semantics in human mode
// while preserving JSON, keys-only, and quiet contracts.
func renderTenantContext(cmd *cobra.Command, ctx tenant.Context) {
	if !shouldRenderTenantContextHuman(cmd, ctx) {
		return
	}
	if tenantContextHumanRendered(cmd) {
		return
	}
	markTenantContextHumanRendered(cmd)

	if line := tenantContextPrimaryHumanLine(ctx); line != "" {
		renderHumanLine(cmd, "%s", line)
	}

	switch len(ctx.ResolvedTenantIDs) {
	case 0:
	case 1:
		renderHumanLine(cmd, "affected tenants: %s", ctx.ResolvedTenantIDs[0])
	default:
		renderHumanLine(cmd, "affected tenants: %s", strings.Join(ctx.ResolvedTenantIDs, ", "))
	}

	for _, warning := range ctx.Warnings {
		if warning.Code == tenant.ContextWarningUnfilteredSelection {
			continue
		}
		renderHumanWarningLine(cmd, "%s", warning.Message)
	}
}

// renderAttachedTenantContext emits the command-scoped tenant context for
// command views that own a visible preflight, confirmation, or result surface.
func renderAttachedTenantContext(cmd *cobra.Command) {
	if ctx, ok := attachedTenantContext(cmd); ok {
		renderTenantContext(cmd, *ctx)
	}
}

// shouldRenderTenantContextHuman keeps tenant context out of protected output
// modes and absent contexts.
func shouldRenderTenantContextHuman(_ *cobra.Command, ctx tenant.Context) bool {
	if tenantContextIsZero(ctx) || flagQuiet {
		return false
	}
	return pickMode() == RenderModeOneLine
}

// tenantContextPrimaryHumanLine centralizes the semantic line so command
// workflows can choose the correct output channel without changing wording.
func tenantContextPrimaryHumanLine(ctx tenant.Context) string {
	switch {
	case ctx.Mode == tenant.ContextModeCreation:
		if ctx.TargetTenantID == "<default>" {
			return "creation target: default tenant"
		}
		return "creation target: " + ctx.TargetTenantID
	case ctx.Filter == tenant.ContextFilterNamed:
		return "selection scope: " + ctx.ConfiguredTenantID + " only"
	case ctx.Filter == tenant.ContextFilterNone:
		return "selection scope: unfiltered across accessible tenants"
	case ctx.Filter == tenant.ContextFilterNotApplied:
		return "selection scope: explicit resource keys; tenant filter not applied"
	default:
		return ""
	}
}
