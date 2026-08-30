// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// TenantContextMode identifies which tenant semantics apply to an operation.
type TenantContextMode string

const (
	// TenantContextModeConfiguration describes configured discovery scope without resource evidence.
	TenantContextModeConfiguration TenantContextMode = "configuration"
	// TenantContextModeDiscovery describes search or selector scope before mutation.
	TenantContextModeDiscovery TenantContextMode = "discovery"
	// TenantContextModeCreation describes the tenant target for newly created resources.
	TenantContextModeCreation TenantContextMode = "creation"
	// TenantContextModeExplicitKeys describes backend-authorized direct resource key operations.
	TenantContextModeExplicitKeys TenantContextMode = "explicit_keys"
)

// String returns the serialized tenant-context mode value.
func (m TenantContextMode) String() string {
	return string(m)
}

// TenantContextFilter identifies whether and how the configured tenant applies.
type TenantContextFilter string

const (
	// TenantContextFilterNamed means discovery is limited by a configured tenant ID.
	TenantContextFilterNamed TenantContextFilter = "named"
	// TenantContextFilterNone means discovery has no tenant filter.
	TenantContextFilterNone TenantContextFilter = "none"
	// TenantContextFilterNotApplied means explicit keys are not locally tenant-filtered.
	TenantContextFilterNotApplied TenantContextFilter = "not_applied"
	// TenantContextFilterNotApplicable means the operation does not use a discovery filter.
	TenantContextFilterNotApplicable TenantContextFilter = "not_applicable"
)

// TenantContextWarningCode identifies a stable tenant-context warning category.
type TenantContextWarningCode string

const (
	// TenantContextWarningUnfilteredSelection warns that discovery used no tenant filter.
	TenantContextWarningUnfilteredSelection TenantContextWarningCode = "unfiltered_selection"
	// TenantContextWarningMultipleTenants warns that known targets span multiple tenants.
	TenantContextWarningMultipleTenants TenantContextWarningCode = "multiple_tenants"
	// TenantContextWarningUnknownTargetTenants warns that some target tenant metadata is unavailable.
	TenantContextWarningUnknownTargetTenants TenantContextWarningCode = "unknown_target_tenants"
)

// TenantContextWarning carries a stable warning code and human-readable message.
type TenantContextWarning struct {
	Code    TenantContextWarningCode `json:"code" yaml:"code"`
	Message string                   `json:"message" yaml:"message"`
}

// TenantContextInput contains the mutable facts used to construct a TenantContext.
type TenantContextInput struct {
	Filter             TenantContextFilter
	ConfiguredTenantID string
	TargetTenantID     string
	ResolvedTenantIDs  []string
	UnknownTargetCount int
}

// TenantContext describes effective tenant semantics and frozen target evidence.
type TenantContext struct {
	Mode               TenantContextMode      `json:"mode" yaml:"mode"`
	Filter             TenantContextFilter    `json:"filter" yaml:"filter"`
	ConfiguredTenantID string                 `json:"configuredTenantId,omitempty" yaml:"configuredTenantId,omitempty"`
	TargetTenantID     string                 `json:"targetTenantId,omitempty" yaml:"targetTenantId,omitempty"`
	ResolvedTenantIDs  []string               `json:"resolvedTenantIds" yaml:"resolvedTenantIds"`
	UnknownTargetCount int                    `json:"unknownTargetCount" yaml:"unknownTargetCount"`
	CrossTenant        bool                   `json:"crossTenant" yaml:"crossTenant"`
	Warnings           []TenantContextWarning `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

// NewTenantContext validates tenant semantics, normalizes evidence, and derives warnings.
func NewTenantContext(mode TenantContextMode, in TenantContextInput) (TenantContext, error) {
	if err := validateTenantContext(mode, in); err != nil {
		return TenantContext{}, err
	}

	resolved := normalizeTenantIDs(in.ResolvedTenantIDs)
	ctx := TenantContext{
		Mode:               mode,
		Filter:             in.Filter,
		ConfiguredTenantID: in.ConfiguredTenantID,
		TargetTenantID:     in.TargetTenantID,
		ResolvedTenantIDs:  resolved,
		UnknownTargetCount: in.UnknownTargetCount,
		CrossTenant:        len(resolved) > 1,
	}
	ctx.Warnings = tenantContextWarnings(ctx)
	return ctx, nil
}

// NewConfigurationTenantContext constructs a configuration-mode context from the effective tenant.
func NewConfigurationTenantContext(configuredTenantID string) (TenantContext, error) {
	filter := TenantContextFilterNone
	if configuredTenantID != "" {
		filter = TenantContextFilterNamed
	}
	return NewTenantContext(TenantContextModeConfiguration, TenantContextInput{
		Filter:             filter,
		ConfiguredTenantID: configuredTenantID,
	})
}

// NewDiscoveryTenantContext constructs a discovery-mode context from the effective tenant.
func NewDiscoveryTenantContext(configuredTenantID string) (TenantContext, error) {
	filter := TenantContextFilterNone
	if configuredTenantID != "" {
		filter = TenantContextFilterNamed
	}
	return NewTenantContext(TenantContextModeDiscovery, TenantContextInput{
		Filter:             filter,
		ConfiguredTenantID: configuredTenantID,
	})
}

// NewCreationTenantContext constructs a creation-mode context for the requested target tenant.
func NewCreationTenantContext(targetTenantID string) (TenantContext, error) {
	return NewTenantContext(TenantContextModeCreation, TenantContextInput{
		Filter:         TenantContextFilterNotApplicable,
		TargetTenantID: targetTenantID,
	})
}

// NewExplicitKeysTenantContext constructs an explicit-keys context retaining optional configuration evidence.
func NewExplicitKeysTenantContext(configuredTenantID string) (TenantContext, error) {
	return NewTenantContext(TenantContextModeExplicitKeys, TenantContextInput{
		Filter:             TenantContextFilterNotApplied,
		ConfiguredTenantID: configuredTenantID,
	})
}

// WithTenantEvidence returns a copy of ctx with normalized resolved and unknown target evidence.
func WithTenantEvidence(ctx TenantContext, resolvedTenantIDs []string, unknownTargetCount int) (TenantContext, error) {
	return NewTenantContext(ctx.Mode, TenantContextInput{
		Filter:             ctx.Filter,
		ConfiguredTenantID: ctx.ConfiguredTenantID,
		TargetTenantID:     ctx.TargetTenantID,
		ResolvedTenantIDs:  resolvedTenantIDs,
		UnknownTargetCount: unknownTargetCount,
	})
}

// validateTenantContext enforces the mode/filter/identifier matrix before serialization.
func validateTenantContext(mode TenantContextMode, in TenantContextInput) error {
	if in.UnknownTargetCount < 0 {
		return fmt.Errorf("%w: tenant context unknown target count must not be negative", ErrValidation)
	}

	switch mode {
	case TenantContextModeConfiguration, TenantContextModeDiscovery:
		if in.TargetTenantID != "" {
			return fmt.Errorf("%w: tenant context %s mode does not accept target tenant", ErrValidation, mode)
		}
		switch in.Filter {
		case TenantContextFilterNamed:
			if in.ConfiguredTenantID == "" {
				return fmt.Errorf("%w: tenant context named filter requires configured tenant", ErrValidation)
			}
		case TenantContextFilterNone:
			if in.ConfiguredTenantID != "" {
				return fmt.Errorf("%w: tenant context none filter must omit configured tenant", ErrValidation)
			}
		default:
			return fmt.Errorf("%w: tenant context %s mode does not accept %q filter", ErrValidation, mode, in.Filter)
		}
	case TenantContextModeCreation:
		if in.Filter != TenantContextFilterNotApplicable {
			return fmt.Errorf("%w: tenant context creation mode requires not_applicable filter", ErrValidation)
		}
		if in.ConfiguredTenantID != "" {
			return fmt.Errorf("%w: tenant context creation mode must omit configured tenant", ErrValidation)
		}
		if in.TargetTenantID == "" {
			return fmt.Errorf("%w: tenant context creation mode requires target tenant", ErrValidation)
		}
	case TenantContextModeExplicitKeys:
		if in.Filter != TenantContextFilterNotApplied {
			return fmt.Errorf("%w: tenant context explicit_keys mode requires not_applied filter", ErrValidation)
		}
		if in.TargetTenantID != "" {
			return fmt.Errorf("%w: tenant context explicit_keys mode must omit target tenant", ErrValidation)
		}
	default:
		return fmt.Errorf("%w: unknown tenant context mode %q", ErrValidation, mode)
	}
	return nil
}

// normalizeTenantIDs drops unavailable tenant metadata and returns a sorted distinct copy.
func normalizeTenantIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != "" {
			out = append(out, id)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// tenantContextWarnings derives stable warning order from normalized tenant evidence.
func tenantContextWarnings(ctx TenantContext) []TenantContextWarning {
	warnings := make([]TenantContextWarning, 0, 3)
	if ctx.Mode == TenantContextModeDiscovery && ctx.Filter == TenantContextFilterNone {
		warnings = append(warnings, TenantContextWarning{
			Code:    TenantContextWarningUnfilteredSelection,
			Message: "selection scope: unfiltered across accessible tenants",
		})
	}
	if ctx.CrossTenant {
		warnings = append(warnings, TenantContextWarning{
			Code:    TenantContextWarningMultipleTenants,
			Message: "WARNING: resources from multiple tenants will be affected: " + strings.Join(ctx.ResolvedTenantIDs, ", "),
		})
	}
	if ctx.UnknownTargetCount > 0 {
		target := "targets"
		if ctx.UnknownTargetCount == 1 {
			target = "target"
		}
		warnings = append(warnings, TenantContextWarning{
			Code:    TenantContextWarningUnknownTargetTenants,
			Message: "WARNING: tenant metadata is unknown for " + strconv.Itoa(ctx.UnknownTargetCount) + " " + target,
		})
	}
	if len(warnings) == 0 {
		return nil
	}
	return warnings
}
