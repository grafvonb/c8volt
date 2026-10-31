// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later
package resource

import (
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestProcessDefinitionImpactBusinessID keeps independent observations and pointer ownership in impact data.
func TestProcessDefinitionImpactBusinessID(t *testing.T) {
	t.Parallel()
	value := "old"
	plan := fromDomainDeleteProcessDefinitionPlan(d.DeleteProcessDefinitionPlan{Items: []d.DeleteProcessDefinitionPlanItem{{CancellationPlan: d.DryRunPIKeyExpansion{
		SelectedFinalState:         []d.ProcessInstance{{Key: "101", BusinessId: &value}},
		RequiresCancelBeforeDelete: []d.ProcessInstance{{Key: "102"}, {Key: "103", BusinessId: new(string)}},
	}}}})
	got := plan.Items[0].CancellationPlan
	require.Equal(t, "old", *got.SelectedFinalState[0].BusinessId)
	require.NotSame(t, &value, got.SelectedFinalState[0].BusinessId)
	require.Nil(t, got.RequiresCancelBeforeDelete[0].BusinessId)
	require.Equal(t, "", *got.RequiresCancelBeforeDelete[1].BusinessId)
	value = "new"
	require.Equal(t, "old", *got.SelectedFinalState[0].BusinessId)
}
