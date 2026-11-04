// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package resource

import (
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestPlanTenantEvidenceDelegatesWithoutCaching preserves manually assembled plans,
// kind-qualified target identity, first observation, aggregate-only counts and ownership.
func TestPlanTenantEvidenceDelegatesWithoutCaching(t *testing.T) {
	t.Parallel()
	nested := process.TenantEvidence{Targets: []process.TenantEvidenceTarget{{Key: "same", TenantID: "tenant-a"}, {Key: "child"}}}
	plan := DeleteProcessDefinitionPlan{Items: []DeleteProcessDefinitionPlanItem{
		{Key: "same", TenantId: "tenant-b", CancellationPlan: process.DryRunPIKeyExpansion{TenantEvidence: nested}},
		{Key: "same", TenantId: "ignored", CancellationPlan: process.DryRunPIKeyExpansion{TenantEvidence: nested}},
		{CancellationPlan: process.DryRunPIKeyExpansion{TenantEvidence: process.TenantEvidence{ResolvedTenantIDs: []string{"tenant-c", "tenant-a"}, TargetCount: 3, UnknownTargetCount: 1}}},
	}}
	got := plan.TenantEvidence()
	require.Equal(t, process.TenantEvidence{ResolvedTenantIDs: []string{"tenant-a", "tenant-b", "tenant-c"}, TargetCount: 6, UnknownTargetCount: 2, Targets: []process.TenantEvidenceTarget{{Key: "same", TenantID: "tenant-b"}, {Key: "same", TenantID: "tenant-a"}, {Key: "child"}}}, got)
	got.Targets[0].TenantID = "changed"
	got.ResolvedTenantIDs[0] = "changed"
	require.Equal(t, "tenant-b", plan.Items[0].TenantId)
	plan.Items[0].TenantId = "updated"
	require.Contains(t, plan.TenantEvidence().ResolvedTenantIDs, "updated")
	require.NotContains(t, plan.TenantEvidence().ResolvedTenantIDs, "tenant-b")
	empty := (DeleteProcessDefinitionPlan{}).TenantEvidence()
	require.Equal(t, process.TenantEvidence{ResolvedTenantIDs: []string{}}, empty)
}
