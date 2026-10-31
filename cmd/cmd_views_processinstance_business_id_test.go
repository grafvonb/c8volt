// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strconv"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/stretchr/testify/require"
)

// TestProcessInstanceBusinessIDRow keeps opaque observations on one safely quoted human line.
func TestProcessInstanceBusinessIDRow(t *testing.T) {
	for _, value := range []string{"", " new,\"ID\"\n\t|<> "} {
		line := oneLinePI(process.ProcessInstance{Key: "101", BusinessId: &value})
		require.Contains(t, line, "business-id:"+strconv.Quote(value))
		require.NotContains(t, line, "\n")
	}
	require.NotContains(t, oneLinePI(process.ProcessInstance{Key: "102"}), "business-id:")
}

// TestProcessInstanceBusinessIDPreviewCopies freezes bespoke final/blocker entries and preserves empty observations through aggregation.
func TestProcessInstanceBusinessIDPreviewCopies(t *testing.T) {
	old, empty := "old", ""
	source := process.DryRunPIKeyExpansion{SelectedFinalState: []process.ProcessInstance{{Key: "101", State: process.StateCompleted, BusinessId: &old}}, RequiresCancelBeforeDelete: []process.ProcessInstance{{Key: "102", State: process.StateActive, BusinessId: &empty}, {Key: "103", State: process.StateActive}}}
	preview := newProcessInstanceDryRunPreview("delete", nil, source)
	summary := newProcessInstanceDryRunSummary("delete", []processInstanceDryRunPreview{preview})
	plan := aggregateDeleteSearchPlan([]processInstanceDryRunPreview{preview}, process.TenantEvidence{})
	old = "new"
	empty = "new"
	require.Equal(t, "old", *preview.SelectedFinalState[0].BusinessId)
	require.Equal(t, "", *preview.RequiresCancelBeforeDelete[0].BusinessId)
	*preview.SelectedFinalState[0].BusinessId = "changed"
	*preview.RequiresCancelBeforeDelete[0].BusinessId = "changed"
	require.Equal(t, "old", *summary.SelectedFinalState[0].BusinessId)
	require.Equal(t, "old", *summary.Previews[0].SelectedFinalState[0].BusinessId)
	require.Equal(t, "old", *plan.SelectedFinalState[0].BusinessId)
	require.Equal(t, "", *plan.RequiresCancelBeforeDelete[0].BusinessId)
	require.Nil(t, plan.RequiresCancelBeforeDelete[1].BusinessId)
	require.Contains(t, formatProcessInstanceDryRunSelectedFinalState(summary.SelectedFinalState), `business-id:"old"`)
	public := process.ProcessInstanceVariableUpdatePlan{Instances: []process.ProcessInstanceVariablePlan{{ProcessInstanceKey: "101", BusinessId: new("old")}}}
	update := processInstanceVariableUpdatePreviewFromFacadePlan(public)
	*public.Instances[0].BusinessId = "changed"
	require.Equal(t, "old", *update.ProcessInstances[0].BusinessId)
}
