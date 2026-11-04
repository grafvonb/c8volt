// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/typex"
)

// CleanupFrozenScope retains discovery, eligibility and ordered family observations for cleanup.
// Ancestries are root-first paths for retention; orphan seeds have directly missing parents.
type CleanupFrozenScope struct {
	Selection        process.ProcessInstanceSelection
	Tenant           string
	Seeds            typex.Keys
	Ancestries       []process.ProcessInstanceMutationFamily
	Families         []process.ProcessInstanceMutationFamily
	ExpansionOutcome process.TraversalOutcome
}

// RetentionFrozenPlan binds the fixed retention boundary and original filters to reviewed scope.
type RetentionFrozenPlan struct {
	Scope                  CleanupFrozenScope
	OriginalSelection      process.ProcessInstanceFilter
	RetentionDays          int
	DerivedEndDateBoundary string
}

// OrphanPurgeFrozenPlan retains inspected candidates separately from orphan deletion targets.
type OrphanPurgeFrozenPlan struct {
	Scope             CleanupFrozenScope
	OriginalSelection process.ProcessInstanceFilter
}
