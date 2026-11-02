// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "github.com/grafvonb/c8volt/typex"

// ProcessInstanceSuspensionPlan freezes discovered scope without claiming mutation evidence.
// TargetKeys is exactly the ordered projection of Instances classified eligible.
// Each target appears once; parent/child identities remain distinct.
// Successful planning exposes non-nil arrays. Any discovery, malformed-response,
// explicit partial/unresolved-result, or context failure returns a zero plan plus
// error. The caller must check error before considering a no-op.
type ProcessInstanceSuspensionPlan struct {
	// Selection retains completed search provenance; nil preserves explicit-key behavior.
	Selection *ProcessInstanceSelection `json:"selection,omitempty"`
	// SelectionTenant binds search authorization to the receiving service.
	SelectionTenant string `json:"selectionTenant,omitempty"`
	// Exact distinguishes selected keys from descendant eligibility plans.
	Exact bool `json:"exact,omitempty"`

	// RequestedKeys: All normalized unique explicit seeds, in input order, including seeds that overlap earlier subtrees.
	RequestedKeys typex.Keys `json:"requestedKeys"`
	// Instances: Unique discovered members in first-seen traversal order.
	Instances []ProcessInstanceSuspensionPlanItem `json:"instances"`
	// TargetKeys: Frozen eligible mutation keys in the same order as their entries in Instances.
	TargetKeys typex.Keys `json:"targetKeys"`
}

// ProcessInstanceSuspensionPlanItem records one discovery observation and its eligibility.
type ProcessInstanceSuspensionPlanItem struct {
	// BusinessId is the optional identity observed during planning.
	BusinessId *string `json:"businessId,omitempty"`
	// Key: Discovered instance identity.
	Key string `json:"key"`
	// ObservedState: Planning observation; never post-submission confirmation evidence.
	ObservedState  State                                       `json:"observedState"`
	Classification ProcessInstanceSuspensionPlanClassification `json:"classification"`
}

// ProcessInstanceSuspensionPlanClassification separates eligibility from existing suspension.
type ProcessInstanceSuspensionPlanClassification string

const (
	// ProcessInstanceSuspensionPlanEligible identifies an observed ACTIVE target.
	ProcessInstanceSuspensionPlanEligible ProcessInstanceSuspensionPlanClassification = "eligible"
	// ProcessInstanceSuspensionPlanAlreadySuspended records existing suspension, not a new mutation.
	ProcessInstanceSuspensionPlanAlreadySuspended ProcessInstanceSuspensionPlanClassification = "already_suspended"
	// ProcessInstanceSuspensionPlanExcluded identifies every other observed state.
	ProcessInstanceSuspensionPlanExcluded ProcessInstanceSuspensionPlanClassification = "excluded"
)
