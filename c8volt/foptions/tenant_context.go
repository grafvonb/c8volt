// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package foptions

// TenantContextMode identifies which tenant semantics apply to an operation in
// facade progress callbacks.
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

// TenantContextFilter identifies whether and how the configured tenant applies
// in facade progress callbacks.
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

// TenantContextWarningCode identifies a stable tenant-context warning category
// in facade progress callbacks.
type TenantContextWarningCode string

const (
	// TenantContextWarningUnfilteredSelection warns that discovery used no tenant filter.
	TenantContextWarningUnfilteredSelection TenantContextWarningCode = "unfiltered_selection"
	// TenantContextWarningMultipleTenants warns that known targets span multiple tenants.
	TenantContextWarningMultipleTenants TenantContextWarningCode = "multiple_tenants"
	// TenantContextWarningUnknownTargetTenants warns that some target tenant metadata is unavailable.
	TenantContextWarningUnknownTargetTenants TenantContextWarningCode = "unknown_target_tenants"
)

// TenantContextWarning carries a stable warning code and human-readable message
// in facade progress callbacks.
type TenantContextWarning struct {
	Code    TenantContextWarningCode `json:"code" yaml:"code"`
	Message string                   `json:"message" yaml:"message"`
}

// TenantContext describes effective tenant semantics and frozen target
// evidence in facade progress callbacks without importing facade packages.
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
