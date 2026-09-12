// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "github.com/grafvonb/c8volt/typex"

type TraversalOutcome string

const (
	TraversalOutcomeComplete   TraversalOutcome = "complete"
	TraversalOutcomePartial    TraversalOutcome = "partial"
	TraversalOutcomeUnresolved TraversalOutcome = "unresolved"
)

type MissingAncestor struct {
	Key      string
	StartKey string
}

// TenantEvidenceTarget records one affected target's tenant observation for
// cross-page deduplication.
type TenantEvidenceTarget struct {
	Key      string
	TenantID string
}

// TenantEvidence captures already-resolved tenant metadata for affected targets.
type TenantEvidence struct {
	ResolvedTenantIDs  []string
	UnknownTargetCount int
	TargetCount        int
	Targets            []TenantEvidenceTarget
}

type DryRunPIKeyExpansion struct {
	Roots                      typex.Keys
	Collected                  typex.Keys
	TenantEvidence             TenantEvidence
	DuplicateRoots             typex.Keys
	SelectedFinalState         []ProcessInstance
	RequiresCancelBeforeDelete []ProcessInstance
	MissingAncestors           []MissingAncestor
	Warning                    string
	Outcome                    TraversalOutcome
}

func (r DryRunPIKeyExpansion) HasActionableResults() bool {
	return len(r.Roots) > 0 || len(r.Collected) > 0
}
