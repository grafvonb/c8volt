// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strconv"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

// attachOpsDiscoveryTenantContext records search-based ops workflow semantics
// plus tenant evidence from the frozen service-owned plan.
func attachOpsDiscoveryTenantContext(cmd *cobra.Command, cfg *config.Config, evidence process.TenantEvidence) tenant.Context {
	ctx := opsTenantContextWithEvidence(newDiscoveryTenantContext(configuredTenantID(cfg)), evidence)
	attachTenantContext(cmd, ctx)
	return ctx
}

// attachOpsExplicitKeysTenantContext records direct-key ops workflow semantics
// plus tenant evidence already present in the frozen plan.
func attachOpsExplicitKeysTenantContext(cmd *cobra.Command, cfg *config.Config, evidence process.TenantEvidence) tenant.Context {
	ctx := opsTenantContextWithEvidence(newExplicitKeysTenantContext(configuredTenantID(cfg)), evidence)
	attachTenantContext(cmd, ctx)
	return ctx
}

// attachOpsCreationTenantContext records smoke-test creation target semantics
// and any tenant evidence returned by the created resources.
func attachOpsCreationTenantContext(cmd *cobra.Command, cfg *config.Config, evidence process.TenantEvidence) tenant.Context {
	ctx := opsTenantContextWithEvidence(newCreationTenantContext(creationTenantID(cfg)), evidence)
	attachTenantContext(cmd, ctx)
	return ctx
}

// printOpsTenantContext writes tenant semantics on the ops durable progress
// channel so preflight and confirmation boundaries do not contaminate stdout.
func printOpsTenantContext(cmd *cobra.Command, ctx tenant.Context, channel ops.ProgressChannel) {
	if cmd == nil || tenantContextIsZero(ctx) || !channel.DurableAllowed || !channel.StderrAllowed || tenantContextHumanRendered(cmd) {
		return
	}
	markTenantContextHumanRendered(cmd)
	if line := tenantContextPrimaryHumanLine(ctx); line != "" {
		printOpsDurableLine(cmd, line, false)
	}
	switch len(ctx.ResolvedTenantIDs) {
	case 0:
	case 1:
		printOpsDurableLine(cmd, "affected tenants: "+ctx.ResolvedTenantIDs[0], false)
	default:
		printOpsDurableLine(cmd, "affected tenants: "+strings.Join(ctx.ResolvedTenantIDs, ", "), false)
	}
	for _, warning := range ctx.Warnings {
		if warning.Code == tenant.ContextWarningUnfilteredSelection {
			continue
		}
		printOpsDurableLine(cmd, warning.Message, true)
	}
}

// opsTenantContextWithEvidence merges a command-owned base context with
// service-owned frozen evidence without introducing enrichment requests.
func opsTenantContextWithEvidence(base tenant.Context, evidence process.TenantEvidence) tenant.Context {
	ids, unknown := opsTenantEvidenceSummary(evidence)
	return withTenantContextEvidence(base, ids, unknown)
}

// opsTenantEvidenceSummary derives stable summary fields and deduplicates by
// target key when services supplied per-target observations.
func opsTenantEvidenceSummary(evidence process.TenantEvidence) ([]string, int) {
	if len(evidence.Targets) == 0 {
		return evidence.ResolvedTenantIDs, evidence.UnknownTargetCount
	}
	seen := make(map[string]struct{}, len(evidence.Targets))
	ids := make([]string, 0, len(evidence.Targets))
	unknown := 0
	for _, target := range evidence.Targets {
		if target.Key == "" {
			continue
		}
		if _, ok := seen[target.Key]; ok {
			continue
		}
		seen[target.Key] = struct{}{}
		if target.TenantID == "" {
			unknown++
			continue
		}
		ids = append(ids, target.TenantID)
	}
	return ids, unknown
}

// opsMergedTenantEvidence combines independent smoke-test creation steps for a
// single report-level context while preserving target-key deduplication.
func opsMergedTenantEvidence(items ...process.TenantEvidence) process.TenantEvidence {
	var targets []process.TenantEvidenceTarget
	var ids []string
	unknown := 0
	for _, item := range items {
		targets = append(targets, item.Targets...)
		if len(item.Targets) == 0 {
			ids = append(ids, item.ResolvedTenantIDs...)
			unknown += item.UnknownTargetCount
		}
	}
	if len(targets) > 0 {
		return process.TenantEvidence{Targets: targets}
	}
	return process.TenantEvidence{ResolvedTenantIDs: ids, UnknownTargetCount: unknown}
}

// opsLegacyTenantIDForContext keeps deprecated audit tenantId only when it is a
// truthful named filter or creation target.
func opsLegacyTenantIDForContext(ctx *tenant.Context) string {
	if ctx == nil {
		return ""
	}
	switch ctx.Mode {
	case tenant.ContextModeCreation:
		return ctx.TargetTenantID
	case tenant.ContextModeDiscovery, tenant.ContextModeConfiguration:
		if ctx.Filter == tenant.ContextFilterNamed {
			return ctx.ConfiguredTenantID
		}
	}
	return ""
}

// cloneTenantContextPtr returns an isolated optional context for report models.
func cloneTenantContextPtr(ctx tenant.Context) *tenant.Context {
	cloned := cloneTenantContext(ctx)
	return &cloned
}

// writeMarkdownTenantContext renders the common context in ops audit reports
// using the same wording as command preflight and confirmation output.
func writeMarkdownTenantContext(out *strings.Builder, ctx *tenant.Context) {
	if out == nil || ctx == nil || tenantContextIsZero(*ctx) {
		return
	}
	writeMarkdownReportField(out, "Tenant Context", tenantContextPrimaryHumanLine(*ctx))
	switch len(ctx.ResolvedTenantIDs) {
	case 0:
	case 1:
		writeMarkdownReportField(out, "Resource Tenant", ctx.ResolvedTenantIDs[0])
	default:
		writeMarkdownReportField(out, "Resource Tenants", strings.Join(ctx.ResolvedTenantIDs, ", "))
	}
	writeMarkdownReportField(out, "Unknown Target Tenants", strconv.Itoa(ctx.UnknownTargetCount))
	writeMarkdownReportField(out, "Cross Tenant", strconv.FormatBool(ctx.CrossTenant))
	warnings := make([]string, 0, len(ctx.Warnings))
	for _, warning := range ctx.Warnings {
		if warning.Code == tenant.ContextWarningUnfilteredSelection {
			continue
		}
		warnings = append(warnings, warning.Message)
	}
	writeMarkdownReportList(out, "Tenant Warnings", warnings)
}
