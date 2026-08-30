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

	targetTenants               map[string]string
	aggregateTenantIDs          map[string]struct{}
	aggregateUnknownTargetCount int
	aggregateTargetCount        int
}

// TenantEvidenceAccumulator tracks keyed targets and aggregate-only evidence.
type TenantEvidenceAccumulator struct {
	targetTenants               map[string]string
	aggregateTenantIDs          map[string]struct{}
	aggregateUnknownTargetCount int
	aggregateTargetCount        int
}

// NewTenantEvidenceAccumulator creates an empty tenant evidence accumulator.
func NewTenantEvidenceAccumulator() *TenantEvidenceAccumulator {
	return &TenantEvidenceAccumulator{
		targetTenants:      make(map[string]string),
		aggregateTenantIDs: make(map[string]struct{}),
	}
}

// Add records the first tenant observation for a target key and ignores duplicate keys.
func (a *TenantEvidenceAccumulator) Add(targetKey, tenantID string) {
	if a == nil || targetKey == "" {
		return
	}
	a.ensureMaps()
	if _, ok := a.targetTenants[targetKey]; ok {
		return
	}
	a.targetTenants[targetKey] = tenantID
}

// AddAggregate records evidence whose individual target keys are unavailable.
// Aggregate counts are additive because they cannot be deduplicated by key.
func (a *TenantEvidenceAccumulator) AddAggregate(resolvedTenantIDs []string, targetCount int, unknownTargetCount int) {
	if a == nil {
		return
	}
	a.ensureMaps()
	for _, tenantID := range resolvedTenantIDs {
		if tenantID != "" {
			a.aggregateTenantIDs[tenantID] = struct{}{}
		}
	}
	if targetCount > 0 {
		a.aggregateTargetCount += targetCount
	}
	if unknownTargetCount > 0 {
		a.aggregateUnknownTargetCount += unknownTargetCount
	}
}

// Merge adds another snapshot while preserving keyed-target deduplication and aggregate totals.
func (a *TenantEvidenceAccumulator) Merge(snapshot TenantEvidenceSnapshot) {
	if a == nil {
		return
	}
	for targetKey, tenantID := range snapshot.targetTenants {
		a.Add(targetKey, tenantID)
	}
	a.ensureMaps()
	for tenantID := range snapshot.aggregateTenantIDs {
		a.aggregateTenantIDs[tenantID] = struct{}{}
	}
	a.aggregateTargetCount += snapshot.aggregateTargetCount
	a.aggregateUnknownTargetCount += snapshot.aggregateUnknownTargetCount
}

// Snapshot returns sorted known tenants and unknown counts without exposing mutable accumulator state.
func (a *TenantEvidenceAccumulator) Snapshot() TenantEvidenceSnapshot {
	if a == nil {
		return TenantEvidenceSnapshot{
			ResolvedTenantIDs: []string{},
		}
	}

	knownTenants := make([]string, 0, len(a.targetTenants)+len(a.aggregateTenantIDs))
	unknownTargetCount := 0
	for _, tenantID := range a.targetTenants {
		if tenantID == "" {
			unknownTargetCount++
			continue
		}
		knownTenants = append(knownTenants, tenantID)
	}
	for tenantID := range a.aggregateTenantIDs {
		knownTenants = append(knownTenants, tenantID)
	}
	slices.Sort(knownTenants)
	knownTenants = slices.Compact(knownTenants)

	return TenantEvidenceSnapshot{
		ResolvedTenantIDs:           knownTenants,
		UnknownTargetCount:          unknownTargetCount + a.aggregateUnknownTargetCount,
		TargetCount:                 len(a.targetTenants) + a.aggregateTargetCount,
		targetTenants:               maps.Clone(a.targetTenants),
		aggregateTenantIDs:          maps.Clone(a.aggregateTenantIDs),
		aggregateUnknownTargetCount: a.aggregateUnknownTargetCount,
		aggregateTargetCount:        a.aggregateTargetCount,
	}
}

// ensureMaps allows zero-value accumulators to behave like values created by the constructor.
func (a *TenantEvidenceAccumulator) ensureMaps() {
	if a.targetTenants == nil {
		a.targetTenants = make(map[string]string)
	}
	if a.aggregateTenantIDs == nil {
		a.aggregateTenantIDs = make(map[string]struct{})
	}
}
