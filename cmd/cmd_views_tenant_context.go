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
		renderHumanLine(cmd, "Resource tenant: %s", ctx.ResolvedTenantIDs[0])
	default:
		renderHumanLine(cmd, "Resource tenants: %s", strings.Join(ctx.ResolvedTenantIDs, ", "))
	}

	for _, warning := range ctx.Warnings {
		if warning.Code == tenant.ContextWarningUnfilteredSelection {
			continue
		}
		renderTenantContextWarningLine(cmd, warning.Message)
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
		return "Create in tenant: " + ctx.TargetTenantID
	case ctx.Filter == tenant.ContextFilterNamed:
		return "Tenant filter: " + ctx.ConfiguredTenantID
	case ctx.Filter == tenant.ContextFilterNone:
		return "Tenant filter: none — resources from multiple tenants may be affected"
	case ctx.Filter == tenant.ContextFilterNotApplied:
		return "Tenant filter: not applied for explicit resource keys"
	default:
		return ""
	}
}

// renderTenantContextWarningLine preserves the feature's required WARNING:
// prefix while still routing through the warning channel for log-backed output.
func renderTenantContextWarningLine(cmd *cobra.Command, msg string) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return
	}
	renderHumanLogLine(cmd, true, "%s", msg)
}
