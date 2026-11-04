// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"github.com/grafvonb/c8volt/c8volt/process"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// toDomainCleanupFrozenScope copies reviewed evidence mechanically across the facade boundary.
func toDomainCleanupFrozenScope(x CleanupFrozenScope) d.CleanupFrozenScope {
	return d.CleanupFrozenScope{Selection: toDomainRepairSelection(x.Selection), Tenant: x.Tenant, Seeds: append([]string(nil), x.Seeds...), Ancestries: toolx.MapSlice(x.Ancestries, toDomainCleanupFamily), Families: toolx.MapSlice(x.Families, toDomainCleanupFamily), ExpansionOutcome: d.TraversalOutcome(x.ExpansionOutcome)}
}

// toDomainRetentionFrozenPlan copies reviewed evidence mechanically across the facade boundary.
func toDomainRetentionFrozenPlan(x RetentionFrozenPlan) d.RetentionFrozenPlan {
	return d.RetentionFrozenPlan{Scope: toDomainCleanupFrozenScope(x.Scope), OriginalSelection: toDomainProcessInstanceFilter(x.OriginalSelection), RetentionDays: x.RetentionDays, DerivedEndDateBoundary: x.DerivedEndDateBoundary}
}

// toDomainOrphanPurgeFrozenPlan copies reviewed evidence mechanically across the facade boundary.
func toDomainOrphanPurgeFrozenPlan(x OrphanPurgeFrozenPlan) d.OrphanPurgeFrozenPlan {
	return d.OrphanPurgeFrozenPlan{Scope: toDomainCleanupFrozenScope(x.Scope), OriginalSelection: toDomainProcessInstanceFilter(x.OriginalSelection)}
}

// toDomainCleanupFamily isolates member observations and ordered keys.
func toDomainCleanupFamily(x process.ProcessInstanceMutationFamily) d.ProcessInstanceMutationFamily {
	return d.ProcessInstanceMutationFamily{RootKey: x.RootKey, Keys: append([]string(nil), x.Keys...), Items: toolx.MapSlice(x.Items, toDomainRepairProcessInstance)}
}

// fromDomainCleanupFrozenScope copies reviewed evidence mechanically across the facade boundary.
func fromDomainCleanupFrozenScope(x d.CleanupFrozenScope) CleanupFrozenScope {
	return CleanupFrozenScope{Selection: fromDomainRepairSelection(x.Selection), Tenant: x.Tenant, Seeds: append([]string(nil), x.Seeds...), Ancestries: toolx.MapSlice(x.Ancestries, fromDomainCleanupFamily), Families: toolx.MapSlice(x.Families, fromDomainCleanupFamily), ExpansionOutcome: process.TraversalOutcome(x.ExpansionOutcome)}
}

// fromDomainRetentionFrozenPlan copies reviewed evidence mechanically across the facade boundary.
func fromDomainRetentionFrozenPlan(x d.RetentionFrozenPlan) RetentionFrozenPlan {
	return RetentionFrozenPlan{Scope: fromDomainCleanupFrozenScope(x.Scope), OriginalSelection: fromDomainProcessInstanceFilter(x.OriginalSelection), RetentionDays: x.RetentionDays, DerivedEndDateBoundary: x.DerivedEndDateBoundary}
}

// fromDomainOrphanPurgeFrozenPlan copies reviewed evidence mechanically across the facade boundary.
func fromDomainOrphanPurgeFrozenPlan(x d.OrphanPurgeFrozenPlan) OrphanPurgeFrozenPlan {
	return OrphanPurgeFrozenPlan{Scope: fromDomainCleanupFrozenScope(x.Scope), OriginalSelection: fromDomainProcessInstanceFilter(x.OriginalSelection)}
}

// fromDomainCleanupFamily isolates member observations and ordered keys.
func fromDomainCleanupFamily(x d.ProcessInstanceMutationFamily) process.ProcessInstanceMutationFamily {
	return process.ProcessInstanceMutationFamily{RootKey: x.RootKey, Keys: append([]string(nil), x.Keys...), Items: toolx.MapSlice(x.Items, fromDomainProcessInstance)}
}
