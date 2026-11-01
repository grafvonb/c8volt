// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "github.com/grafvonb/c8volt/typex"

// ProcessInstanceSelectionCardinality declares the proof required by a selection consumer.
type ProcessInstanceSelectionCardinality string

const (
	// ProcessInstanceSelectionCollection accepts exhausted or explicitly bounded distinct selections.
	ProcessInstanceSelectionCollection ProcessInstanceSelectionCardinality = "collection"
	// ProcessInstanceSelectionUniqueStart requires exhaustion with at most one distinct key.
	ProcessInstanceSelectionUniqueStart ProcessInstanceSelectionCardinality = "unique_start"
)

// ProcessInstanceSelectionRequest retains search controls separately from cardinality.
type ProcessInstanceSelectionRequest struct {
	SearchRequest ProcessInstanceSearchRequest
	Cardinality   ProcessInstanceSelectionCardinality
}

// ProcessInstanceSelection preserves copied provenance and partial evidence; an error always prevents execution.
type ProcessInstanceSelection struct {
	Request        ProcessInstanceSelectionRequest
	Origin         string
	Items          []ProcessInstance
	Keys           typex.Keys
	Completion     ProcessInstanceSearchCompletion
	Pages          int32
	TenantEvidence TenantEvidence
	IgnoreTenant   bool
}

// ProcessInstanceSelectedMutationRequest identifies the reviewed collection operation.
type ProcessInstanceSelectedMutationRequest struct {
	Operation string
	Selection ProcessInstanceSelectionRequest
	Workers   int
}

// ProcessInstanceMutationFamily retains ordered members and observed parent relationships for one root.
type ProcessInstanceMutationFamily struct {
	RootKey string
	Keys    typex.Keys
	Items   []ProcessInstance
}

// ProcessInstanceSelectedMutationPlan is invocation-local evidence, not an authorization token.
// Readiness is derived from discovery, complete expansion and consistent family membership.
type ProcessInstanceSelectedMutationPlan struct {
	Request   ProcessInstanceSelectedMutationRequest
	Selection ProcessInstanceSelection
	Expansion DryRunPIKeyExpansion
	Families  []ProcessInstanceMutationFamily
	Tenant    string
}

// ProcessInstanceDeletionResult retains root outcomes and every attempted frozen member, including partial families.
type ProcessInstanceDeletionResult struct {
	MutationSubmitted bool
	Roots             []Reporter
	Members           []Reporter
}
