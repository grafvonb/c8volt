// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package tenant

// ContextMode identifies which tenant semantics apply to an operation.
type ContextMode string

const (
	// ContextModeConfiguration describes configured discovery scope without resource evidence.
	ContextModeConfiguration ContextMode = "configuration"
	// ContextModeDiscovery describes search or selector scope before mutation.
	ContextModeDiscovery ContextMode = "discovery"
	// ContextModeCreation describes the tenant target for newly created resources.
	ContextModeCreation ContextMode = "creation"
	// ContextModeExplicitKeys describes backend-authorized direct resource key operations.
	ContextModeExplicitKeys ContextMode = "explicit_keys"
)

// ContextFilter identifies whether and how the configured tenant applies.
type ContextFilter string

const (
	// ContextFilterNamed means discovery is limited by a configured tenant ID.
	ContextFilterNamed ContextFilter = "named"
	// ContextFilterNone means discovery has no tenant filter.
	ContextFilterNone ContextFilter = "none"
	// ContextFilterNotApplied means explicit keys are not locally tenant-filtered.
	ContextFilterNotApplied ContextFilter = "not_applied"
	// ContextFilterNotApplicable means the operation does not use a discovery filter.
	ContextFilterNotApplicable ContextFilter = "not_applicable"
)

// ContextWarningCode identifies a stable tenant-context warning category.
type ContextWarningCode string

const (
	// ContextWarningUnfilteredSelection warns that discovery used no tenant filter.
	ContextWarningUnfilteredSelection ContextWarningCode = "unfiltered_selection"
	// ContextWarningMultipleTenants warns that known targets span multiple tenants.
	ContextWarningMultipleTenants ContextWarningCode = "multiple_tenants"
	// ContextWarningUnknownTargetTenants warns that some target tenant metadata is unavailable.
	ContextWarningUnknownTargetTenants ContextWarningCode = "unknown_target_tenants"
)

// ContextWarning carries a stable warning code and human-readable message.
type ContextWarning struct {
	Code    ContextWarningCode `json:"code" yaml:"code"`
	Message string             `json:"message" yaml:"message"`
}

// Context describes effective tenant semantics and frozen target evidence.
type Context struct {
	Mode               ContextMode      `json:"mode" yaml:"mode"`
	Filter             ContextFilter    `json:"filter" yaml:"filter"`
	ConfiguredTenantID string           `json:"configuredTenantId,omitempty" yaml:"configuredTenantId,omitempty"`
	TargetTenantID     string           `json:"targetTenantId,omitempty" yaml:"targetTenantId,omitempty"`
	ResolvedTenantIDs  []string         `json:"resolvedTenantIds" yaml:"resolvedTenantIds"`
	UnknownTargetCount int              `json:"unknownTargetCount" yaml:"unknownTargetCount"`
	CrossTenant        bool             `json:"crossTenant" yaml:"crossTenant"`
	Warnings           []ContextWarning `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}
