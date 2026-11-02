// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

// BusinessIDAssignmentPlan holds complete explicit scope; previews do not guarantee eligibility.
type BusinessIDAssignmentPlan struct {
	// Required exact frozen assignment
	RequestedBusinessID *string `json:"requestedBusinessId"`
	// Complete validated keys, normalized and deduplicated in stable order
	RequestedKeys []string `json:"requestedKeys"`
	// One item per distinct target; never expanded to parent/family
	Items []BusinessIDAssignmentItem `json:"items"`
	// Authoritative complete preparation or incomplete diagnostic evidence.
	Ready bool `json:"ready"`
}

// BusinessIDAssignmentItem contains exact-target preview facts, never inferred family facts.
type BusinessIDAssignmentItem struct {
	Key string `json:"key"`
	// Optional prior fact; absence is not synthesized
	PreviouslyObservedBusinessID *string `json:"previouslyObservedBusinessId,omitempty"`
	State                        State   `json:"state,omitempty"`
}

// BusinessIDAssignmentResult separates submission evidence from later exact-key observation.
// Status is not_submitted, accepted, confirmed, rejected or unknown; confirmation is skipped, confirmed or unconfirmed.
type BusinessIDAssignmentResult struct {
	Key                          string  `json:"key"`
	RequestedBusinessID          *string `json:"requestedBusinessId"`
	PreviouslyObservedBusinessID *string `json:"previouslyObservedBusinessId,omitempty"`
	Status                       string  `json:"status"`
	Confirmation                 string  `json:"confirmation"`
	ObservedBusinessID           *string `json:"observedBusinessId,omitempty"`
	HasObservation               bool    `json:"hasObservation"`
	HTTPStatus                   *int    `json:"httpStatus,omitempty"`
	Error                        string  `json:"error,omitempty"`
}

// BusinessIDAssignmentResults preserves input order, including targets never submitted.
type BusinessIDAssignmentResults struct {
	Items []BusinessIDAssignmentResult `json:"items"`
}
