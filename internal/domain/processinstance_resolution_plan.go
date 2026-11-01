// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "github.com/grafvonb/c8volt/typex"

// ProcessInstancesIncidentPlan records completed per-instance reads and their exact incident observations.
// Resolution submits only active owned incidents; job and variable observations authorize no extra action.
type ProcessInstancesIncidentPlan struct {
	Keys          typex.Keys
	CompletedKeys typex.Keys
	Incidents     []ProcessInstanceIncidentDetail
}

// SelectedProcessInstanceResolutionPlan retains selection, family scope and reviewed incident work.
type SelectedProcessInstanceResolutionPlan struct {
	Scope     ProcessInstanceSelectedMutationPlan
	Incidents ProcessInstancesIncidentPlan
}
