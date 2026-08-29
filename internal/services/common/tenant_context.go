// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"maps"
	"slices"
)

// TenantEvidenceSnapshot captures deterministic tenant evidence for already-resolved targets.
type TenantEvidenceSnapshot struct {
	ResolvedTenantIDs  []string
	UnknownTargetCount int
	TargetCount        int

	targetTenants map[string]string
}

// TenantEvidenceAccumulator tracks tenant evidence by unique affected target key.
type TenantEvidenceAccumulator struct {
	targetTenants map[string]string
}

// NewTenantEvidenceAccumulator creates an empty tenant evidence accumulator.
func NewTenantEvidenceAccumulator() *TenantEvidenceAccumulator {
	return &TenantEvidenceAccumulator{
		targetTenants: make(map[string]string),
	}
}

// Add records the first tenant observation for a target key and ignores duplicate keys.
func (a *TenantEvidenceAccumulator) Add(targetKey, tenantID string) {
	if targetKey == "" {
		return
	}
	a.ensureMap()
	if _, ok := a.targetTenants[targetKey]; ok {
		return
	}
	a.targetTenants[targetKey] = tenantID
}

// Merge adds another snapshot's per-target evidence while preserving duplicate suppression.
func (a *TenantEvidenceAccumulator) Merge(snapshot TenantEvidenceSnapshot) {
	for targetKey, tenantID := range snapshot.targetTenants {
		a.Add(targetKey, tenantID)
	}
}

// Snapshot returns sorted known tenants and unknown counts without exposing mutable accumulator state.
func (a *TenantEvidenceAccumulator) Snapshot() TenantEvidenceSnapshot {
	if a == nil || len(a.targetTenants) == 0 {
		return TenantEvidenceSnapshot{
			ResolvedTenantIDs: []string{},
		}
	}

	knownTenants := make([]string, 0, len(a.targetTenants))
	unknownTargetCount := 0
	for _, tenantID := range a.targetTenants {
		if tenantID == "" {
			unknownTargetCount++
			continue
		}
		knownTenants = append(knownTenants, tenantID)
	}
	slices.Sort(knownTenants)
	knownTenants = slices.Compact(knownTenants)

	return TenantEvidenceSnapshot{
		ResolvedTenantIDs:  knownTenants,
		UnknownTargetCount: unknownTargetCount,
		TargetCount:        len(a.targetTenants),
		targetTenants:      maps.Clone(a.targetTenants),
	}
}

// ensureMap allows zero-value accumulators to behave like values created by the constructor.
func (a *TenantEvidenceAccumulator) ensureMap() {
	if a.targetTenants == nil {
		a.targetTenants = make(map[string]string)
	}
}
