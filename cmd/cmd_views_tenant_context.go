// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/spf13/cobra"
)

// renderTenantContext writes operation-specific tenant semantics in human mode
// while preserving JSON, keys-only, and quiet contracts.
func renderTenantContext(cmd *cobra.Command, ctx tenant.Context) {
	if !shouldRenderTenantContextHuman(cmd, ctx) {
		return
	}
	if state, staged := tenantContextHumanRenderStages(cmd); staged {
		renderStagedTenantContext(cmd, ctx, state)
		return
	}
	if tenantContextHumanRendered(cmd) {
		return
	}
	markTenantContextHumanRendered(cmd)

	for _, line := range tenantContextHumanLines(cmd, ctx) {
		if line.Warn {
			renderHumanWarningLine(cmd, "%s", line.Text)
			continue
		}
		renderHumanLine(cmd, "%s", line.Text)
	}
}

// renderStagedTenantContext fills only applicable stages not already emitted
// on the ops durable channel, preserving final-render fallback behavior.
func renderStagedTenantContext(cmd *cobra.Command, ctx tenant.Context, state tenantContextHumanRenderState) {
	if !state.selectionRendered {
		lines := tenantContextSelectionHumanLines(cmd, ctx)
		renderTenantContextHumanLines(cmd, lines)
		if len(lines) > 0 {
			markTenantContextSelectionRendered(cmd)
		}
	}
	if !state.affectedRendered {
		lines := tenantContextAffectedHumanLines(ctx)
		renderTenantContextHumanLines(cmd, lines)
		if len(lines) > 0 {
			markTenantContextAffectedRendered(cmd)
		}
	}
}

// renderTenantContextHumanLines preserves the existing human warning styling
// for a semantically selected subset of tenant-context lines.
func renderTenantContextHumanLines(cmd *cobra.Command, lines []tenantContextHumanLine) {
	for _, line := range lines {
		if line.Warn {
			renderHumanWarningLine(cmd, "%s", line.Text)
			continue
		}
		renderHumanLine(cmd, "%s", line.Text)
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
