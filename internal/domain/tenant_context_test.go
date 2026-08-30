// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verifies every supported tenant-context mode accepts only its documented filter and identifier shape.
func TestNewTenantContext_ValidModeFilterCombinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mode TenantContextMode
		in   TenantContextInput
		want TenantContext
	}{
		{
			name: "configuration named",
			mode: TenantContextModeConfiguration,
			in: TenantContextInput{
				Filter:             TenantContextFilterNamed,
				ConfiguredTenantID: "tenant-a",
			},
			want: TenantContext{
				Mode:               TenantContextModeConfiguration,
				Filter:             TenantContextFilterNamed,
				ConfiguredTenantID: "tenant-a",
				ResolvedTenantIDs:  []string{},
			},
		},
		{
			name: "configuration none",
			mode: TenantContextModeConfiguration,
			in:   TenantContextInput{Filter: TenantContextFilterNone},
			want: TenantContext{
				Mode:              TenantContextModeConfiguration,
				Filter:            TenantContextFilterNone,
				ResolvedTenantIDs: []string{},
			},
		},
		{
			name: "discovery named",
			mode: TenantContextModeDiscovery,
			in: TenantContextInput{
				Filter:             TenantContextFilterNamed,
				ConfiguredTenantID: "tenant-a",
			},
			want: TenantContext{
				Mode:               TenantContextModeDiscovery,
				Filter:             TenantContextFilterNamed,
				ConfiguredTenantID: "tenant-a",
				ResolvedTenantIDs:  []string{},
			},
		},
		{
			name: "discovery none",
			mode: TenantContextModeDiscovery,
			in:   TenantContextInput{Filter: TenantContextFilterNone},
			want: TenantContext{
				Mode:              TenantContextModeDiscovery,
				Filter:            TenantContextFilterNone,
				ResolvedTenantIDs: []string{},
				Warnings: []TenantContextWarning{
					{
						Code:    TenantContextWarningUnfilteredSelection,
						Message: "selection scope: unfiltered across accessible tenants",
					},
				},
			},
		},
		{
			name: TenantContextModeCreation.String(),
			mode: TenantContextModeCreation,
			in: TenantContextInput{
				Filter:         TenantContextFilterNotApplicable,
				TargetTenantID: "<default>",
			},
			want: TenantContext{
				Mode:              TenantContextModeCreation,
				Filter:            TenantContextFilterNotApplicable,
				TargetTenantID:    "<default>",
				ResolvedTenantIDs: []string{},
			},
		},
		{
			name: "explicit keys with configured tenant",
			mode: TenantContextModeExplicitKeys,
			in: TenantContextInput{
				Filter:             TenantContextFilterNotApplied,
				ConfiguredTenantID: "tenant-a",
			},
			want: TenantContext{
				Mode:               TenantContextModeExplicitKeys,
				Filter:             TenantContextFilterNotApplied,
				ConfiguredTenantID: "tenant-a",
				ResolvedTenantIDs:  []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewTenantContext(tt.mode, tt.in)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Verifies invalid filter combinations and required identifiers fail before commands can serialize them.
func TestNewTenantContext_RejectsInvalidModeFilterAndIdentifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mode TenantContextMode
		in   TenantContextInput
	}{
		{name: "unknown mode", mode: TenantContextMode("bogus"), in: TenantContextInput{Filter: TenantContextFilterNamed, ConfiguredTenantID: "tenant-a"}},
		{name: "configuration invalid filter", mode: TenantContextModeConfiguration, in: TenantContextInput{Filter: TenantContextFilterNotApplied}},
		{name: "discovery named missing tenant", mode: TenantContextModeDiscovery, in: TenantContextInput{Filter: TenantContextFilterNamed}},
		{name: "discovery none with configured tenant", mode: TenantContextModeDiscovery, in: TenantContextInput{Filter: TenantContextFilterNone, ConfiguredTenantID: "tenant-a"}},
		{name: "creation missing target", mode: TenantContextModeCreation, in: TenantContextInput{Filter: TenantContextFilterNotApplicable}},
		{name: "creation with configured tenant", mode: TenantContextModeCreation, in: TenantContextInput{Filter: TenantContextFilterNotApplicable, ConfiguredTenantID: "tenant-a", TargetTenantID: "<default>"}},
		{name: "explicit keys with target tenant", mode: TenantContextModeExplicitKeys, in: TenantContextInput{Filter: TenantContextFilterNotApplied, TargetTenantID: "tenant-a"}},
		{name: "negative unknown count", mode: TenantContextModeExplicitKeys, in: TenantContextInput{Filter: TenantContextFilterNotApplied, UnknownTargetCount: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewTenantContext(tt.mode, tt.in)

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrValidation)
		})
	}
}

// Verifies resolved tenant evidence is immutable, sorted, deduplicated, and reflected in derived warnings.
func TestNewTenantContext_CopiesAndNormalizesEvidenceSlices(t *testing.T) {
	t.Parallel()

	resolved := []string{"tenant-b", "", "tenant-a", "tenant-b"}
	got, err := NewTenantContext(TenantContextModeExplicitKeys, TenantContextInput{
		Filter:             TenantContextFilterNotApplied,
		ConfiguredTenantID: "tenant-z",
		ResolvedTenantIDs:  resolved,
		UnknownTargetCount: 2,
	})

	require.NoError(t, err)
	resolved[0] = "changed"

	assert.Equal(t, []string{"tenant-a", "tenant-b"}, got.ResolvedTenantIDs)
	assert.True(t, got.CrossTenant)
	assert.Equal(t, []TenantContextWarning{
		{
			Code:    TenantContextWarningMultipleTenants,
			Message: "resources from multiple tenants will be affected: tenant-a, tenant-b",
		},
		{
			Code:    TenantContextWarningUnknownTargetTenants,
			Message: "tenant metadata is unknown for 2 targets",
		},
	}, got.Warnings)
}

// Verifies domain JSON tags match the shared structured contract before public facade conversion is applied.
func TestTenantContext_JSONTagsMatchContract(t *testing.T) {
	t.Parallel()

	got, err := NewTenantContext(TenantContextModeDiscovery, TenantContextInput{
		Filter:             TenantContextFilterNamed,
		ConfiguredTenantID: "tenant-a",
		ResolvedTenantIDs: []string{
			"tenant-a",
		},
	})
	require.NoError(t, err)

	raw, err := json.Marshal(got)
	require.NoError(t, err)

	require.JSONEq(t, `{
		"mode": "discovery",
		"filter": "named",
		"configuredTenantId": "tenant-a",
		"resolvedTenantIds": ["tenant-a"],
		"unknownTargetCount": 0,
		"crossTenant": false
	}`, string(raw))
}
