// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "github.com/grafvonb/c8volt/typex"

// ProcessInstanceResumptionPlan freezes discovered scope without claiming mutation evidence.
// TargetKeys is exactly the ordered projection of Instances classified eligible.
// Each target appears once; parent/child identities remain distinct.
// Successful planning exposes non-nil arrays. Any discovery, malformed-response,
// explicit partial/unresolved-result, or context failure returns a zero plan plus
// error. The caller must check error before considering a no-op.
type ProcessInstanceResumptionPlan struct {
	// RequestedKeys: All normalized unique explicit seeds, in input order, including seeds that overlap earlier subtrees.
	RequestedKeys typex.Keys `json:"requestedKeys"`
	// Instances: Unique discovered members in first-seen traversal order.
	Instances []ProcessInstanceResumptionPlanItem `json:"instances"`
	// TargetKeys: Frozen eligible mutation keys in the same order as their entries in Instances.
	TargetKeys typex.Keys `json:"targetKeys"`
}

// ProcessInstanceResumptionPlanItem records one discovery observation and its eligibility.
type ProcessInstanceResumptionPlanItem struct {
	// Key: Discovered instance identity.
	Key string `json:"key"`
	// ObservedState: Planning observation; never post-submission confirmation evidence.
	ObservedState  State                                       `json:"observedState"`
	Classification ProcessInstanceResumptionPlanClassification `json:"classification"`
}

// ProcessInstanceResumptionPlanClassification identifies eligibility observed during complete discovery.
type ProcessInstanceResumptionPlanClassification string

const (
	// ProcessInstanceResumptionPlanEligible identifies an observed SUSPENDED target.
	ProcessInstanceResumptionPlanEligible ProcessInstanceResumptionPlanClassification = "eligible"
	// ProcessInstanceResumptionPlanExcluded identifies every other observed state.
	ProcessInstanceResumptionPlanExcluded ProcessInstanceResumptionPlanClassification = "excluded"
)
