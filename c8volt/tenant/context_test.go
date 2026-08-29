// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package tenant

import (
	"encoding/json"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Verifies tenant-context JSON and YAML tags use the common public contract field names.
func TestContext_JSONAndYAMLTagsMatchContract(t *testing.T) {
	t.Parallel()

	value := Context{
		Mode:               ContextModeExplicitKeys,
		Filter:             ContextFilterNotApplied,
		ConfiguredTenantID: "tenant-a",
		ResolvedTenantIDs:  []string{"tenant-b"},
		UnknownTargetCount: 1,
		Warnings: []ContextWarning{
			{
				Code:    ContextWarningUnknownTargetTenants,
				Message: "WARNING: tenant metadata is unknown for 1 target",
			},
		},
	}

	rawJSON, err := json.Marshal(value)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"mode": "explicit_keys",
		"filter": "not_applied",
		"configuredTenantId": "tenant-a",
		"resolvedTenantIds": ["tenant-b"],
		"unknownTargetCount": 1,
		"crossTenant": false,
		"warnings": [
			{
				"code": "unknown_target_tenants",
				"message": "WARNING: tenant metadata is unknown for 1 target"
			}
		]
	}`, string(rawJSON))

	rawYAML, err := yaml.Marshal(value)
	require.NoError(t, err)
	var gotYAML map[string]any
	require.NoError(t, yaml.Unmarshal(rawYAML, &gotYAML))

	assert.Contains(t, gotYAML, "mode")
	assert.Contains(t, gotYAML, "filter")
	assert.Contains(t, gotYAML, "configuredTenantId")
	assert.Contains(t, gotYAML, "resolvedTenantIds")
	assert.Contains(t, gotYAML, "unknownTargetCount")
	assert.Contains(t, gotYAML, "crossTenant")
	assert.Contains(t, gotYAML, "warnings")
	assert.NotContains(t, gotYAML, "ConfiguredTenantID")
	assert.NotContains(t, gotYAML, "TargetTenantID")
}

// Verifies public conversion from domain context copies all slices so callers cannot mutate domain evidence.
func TestFromDomainTenantContext_CopiesSlices(t *testing.T) {
	t.Parallel()

	domainWarnings := []d.TenantContextWarning{
		{
			Code:    d.TenantContextWarningMultipleTenants,
			Message: "WARNING: resources from multiple tenants will be affected: tenant-a, tenant-b",
		},
	}
	domainValue := d.TenantContext{
		Mode:               d.TenantContextModeDiscovery,
		Filter:             d.TenantContextFilterNone,
		ResolvedTenantIDs:  []string{"tenant-a", "tenant-b"},
		UnknownTargetCount: 1,
		CrossTenant:        true,
		Warnings:           domainWarnings,
	}

	got := fromDomainTenantContext(domainValue)
	domainValue.ResolvedTenantIDs[0] = "changed"
	domainWarnings[0].Message = "changed"

	assert.Equal(t, Context{
		Mode:               ContextModeDiscovery,
		Filter:             ContextFilterNone,
		ResolvedTenantIDs:  []string{"tenant-a", "tenant-b"},
		UnknownTargetCount: 1,
		CrossTenant:        true,
		Warnings: []ContextWarning{
			{
				Code:    ContextWarningMultipleTenants,
				Message: "WARNING: resources from multiple tenants will be affected: tenant-a, tenant-b",
			},
		},
	}, got)
}

// Verifies conversion back to the domain model validates combinations and copies mutable slices.
func TestToDomainTenantContext_ValidatesAndCopiesSlices(t *testing.T) {
	t.Parallel()

	publicValue := Context{
		Mode:               ContextModeDiscovery,
		Filter:             ContextFilterNamed,
		ConfiguredTenantID: "tenant-a",
		ResolvedTenantIDs:  []string{"tenant-b"},
		Warnings: []ContextWarning{
			{
				Code:    ContextWarningMultipleTenants,
				Message: "stale caller warning is recomputed by the domain model",
			},
		},
	}

	got, err := toDomainTenantContext(publicValue)
	require.NoError(t, err)
	publicValue.ResolvedTenantIDs[0] = "changed"

	assert.Equal(t, d.TenantContext{
		Mode:               d.TenantContextModeDiscovery,
		Filter:             d.TenantContextFilterNamed,
		ConfiguredTenantID: "tenant-a",
		ResolvedTenantIDs:  []string{"tenant-b"},
	}, got)

	_, err = toDomainTenantContext(Context{
		Mode:   ContextModeCreation,
		Filter: ContextFilterNotApplicable,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, d.ErrValidation)
}
