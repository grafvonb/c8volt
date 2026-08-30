// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

type tenantContextKey struct{}
type tenantContextHumanRenderedKey struct{}

// attachTenantContext stores the immutable tenant context for later shared
// envelope rendering.
func attachTenantContext(cmd *cobra.Command, ctx tenant.Context) {
	if cmd == nil || tenantContextIsZero(ctx) {
		return
	}
	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	cmd.SetContext(context.WithValue(parent, tenantContextKey{}, cloneTenantContext(ctx)))
}

// attachedTenantContext returns a copy so renderers cannot mutate the command
// context value by accident.
func attachedTenantContext(cmd *cobra.Command) (*tenant.Context, bool) {
	if cmd == nil || cmd.Context() == nil {
		return nil, false
	}
	ctx, ok := cmd.Context().Value(tenantContextKey{}).(tenant.Context)
	if !ok || tenantContextIsZero(ctx) {
		return nil, false
	}
	cloned := cloneTenantContext(ctx)
	return &cloned, true
}

// markTenantContextHumanRendered records that a command already emitted the
// current human tenant-context block so shared renderers do not duplicate it.
func markTenantContextHumanRendered(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	cmd.SetContext(context.WithValue(parent, tenantContextHumanRenderedKey{}, true))
}

// tenantContextHumanRendered reports whether this command already emitted
// human tenant-context output.
func tenantContextHumanRendered(cmd *cobra.Command) bool {
	if cmd == nil || cmd.Context() == nil {
		return false
	}
	rendered, _ := cmd.Context().Value(tenantContextHumanRenderedKey{}).(bool)
	return rendered
}

// attachConfigurationTenantContext records configured tenant semantics for
// configuration diagnostics.
func attachConfigurationTenantContext(cmd *cobra.Command, cfg *config.Config) tenant.Context {
	ctx := newConfigurationTenantContext(configuredTenantID(cfg))
	attachTenantContext(cmd, ctx)
	return ctx
}

// attachDiscoveryTenantContext records search or selector tenant semantics
// before a mutation plan is resolved.
func attachDiscoveryTenantContext(cmd *cobra.Command, cfg *config.Config) tenant.Context {
	ctx := newDiscoveryTenantContext(configuredTenantID(cfg))
	attachTenantContext(cmd, ctx)
	return ctx
}

// attachCreationTenantContext records the concrete tenant target used by
// create, deploy, and run operations.
func attachCreationTenantContext(cmd *cobra.Command, cfg *config.Config) tenant.Context {
	ctx := newCreationTenantContext(creationTenantID(cfg))
	attachTenantContext(cmd, ctx)
	return ctx
}

// attachExplicitKeysTenantContext records that direct keys bypass local tenant
// filtering and rely on backend authorization.
func attachExplicitKeysTenantContext(cmd *cobra.Command, cfg *config.Config) tenant.Context {
	ctx := newExplicitKeysTenantContext(configuredTenantID(cfg))
	attachTenantContext(cmd, ctx)
	return ctx
}

// newConfigurationTenantContext constructs configuration-mode context from the
// effective configured tenant without treating an empty value as default.
func newConfigurationTenantContext(configuredTenantID string) tenant.Context {
	filter := tenant.ContextFilterNone
	if configuredTenantID != "" {
		filter = tenant.ContextFilterNamed
	}
	return withTenantContextWarnings(tenant.Context{
		Mode:               tenant.ContextModeConfiguration,
		Filter:             filter,
		ConfiguredTenantID: configuredTenantID,
		ResolvedTenantIDs:  []string{},
	})
}

// newDiscoveryTenantContext constructs search/selector context from the
// effective configured tenant.
func newDiscoveryTenantContext(configuredTenantID string) tenant.Context {
	filter := tenant.ContextFilterNone
	if configuredTenantID != "" {
		filter = tenant.ContextFilterNamed
	}
	return withTenantContextWarnings(tenant.Context{
		Mode:               tenant.ContextModeDiscovery,
		Filter:             filter,
		ConfiguredTenantID: configuredTenantID,
		ResolvedTenantIDs:  []string{},
	})
}

// newCreationTenantContext constructs creation target context without applying
// discovery filter semantics.
func newCreationTenantContext(targetTenantID string) tenant.Context {
	return withTenantContextWarnings(tenant.Context{
		Mode:              tenant.ContextModeCreation,
		Filter:            tenant.ContextFilterNotApplicable,
		TargetTenantID:    targetTenantID,
		ResolvedTenantIDs: []string{},
	})
}

// newExplicitKeysTenantContext constructs direct-key context while preserving
// the configured tenant as non-filtering evidence.
func newExplicitKeysTenantContext(configuredTenantID string) tenant.Context {
	return withTenantContextWarnings(tenant.Context{
		Mode:               tenant.ContextModeExplicitKeys,
		Filter:             tenant.ContextFilterNotApplied,
		ConfiguredTenantID: configuredTenantID,
		ResolvedTenantIDs:  []string{},
	})
}

// withTenantContextEvidence returns a normalized copy with resolved tenant
// evidence and derived warnings.
func withTenantContextEvidence(ctx tenant.Context, resolvedTenantIDs []string, unknownTargetCount int) tenant.Context {
	ctx = cloneTenantContext(ctx)
	ctx.ResolvedTenantIDs = normalizeTenantContextIDs(resolvedTenantIDs)
	if unknownTargetCount < 0 {
		unknownTargetCount = 0
	}
	ctx.UnknownTargetCount = unknownTargetCount
	ctx.CrossTenant = len(ctx.ResolvedTenantIDs) > 1
	return withTenantContextWarnings(ctx)
}

// withTenantContextWarnings derives the stable command contract warning order.
func withTenantContextWarnings(ctx tenant.Context) tenant.Context {
	ctx.Warnings = nil
	if ctx.Mode == tenant.ContextModeDiscovery && ctx.Filter == tenant.ContextFilterNone {
		ctx.Warnings = append(ctx.Warnings, tenant.ContextWarning{
			Code:    tenant.ContextWarningUnfilteredSelection,
			Message: "selection scope: unfiltered across accessible tenants",
		})
	}
	if ctx.CrossTenant {
		ctx.Warnings = append(ctx.Warnings, tenant.ContextWarning{
			Code:    tenant.ContextWarningMultipleTenants,
			Message: "WARNING: resources from multiple tenants will be affected: " + strings.Join(ctx.ResolvedTenantIDs, ", "),
		})
	}
	if ctx.UnknownTargetCount > 0 {
		target := "targets"
		if ctx.UnknownTargetCount == 1 {
			target = "target"
		}
		ctx.Warnings = append(ctx.Warnings, tenant.ContextWarning{
			Code:    tenant.ContextWarningUnknownTargetTenants,
			Message: "WARNING: tenant metadata is unknown for " + strconv.Itoa(ctx.UnknownTargetCount) + " " + target,
		})
	}
	return ctx
}

// cloneTenantContext copies mutable slices held inside a public tenant context.
func cloneTenantContext(ctx tenant.Context) tenant.Context {
	ctx.ResolvedTenantIDs = slices.Clone(ctx.ResolvedTenantIDs)
	ctx.Warnings = slices.Clone(ctx.Warnings)
	return ctx
}

// normalizeTenantContextIDs drops unavailable tenant IDs and returns sorted
// distinct tenant evidence.
func normalizeTenantContextIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		out = append(out, id)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// configuredTenantID safely reads the effective tenant used for discovery.
func configuredTenantID(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	return cfg.App.Tenant
}

// creationTenantID safely reads the target tenant used for newly created work.
func creationTenantID(cfg *config.Config) string {
	if cfg == nil {
		return config.DefaultTenant
	}
	return cfg.App.TargetTenant()
}

// tenantContextIsZero detects the absence of an attached context.
func tenantContextIsZero(ctx tenant.Context) bool {
	return ctx.Mode == "" && ctx.Filter == ""
}
