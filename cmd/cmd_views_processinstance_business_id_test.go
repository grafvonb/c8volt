// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/stretchr/testify/require"
)

// TestProcessInstanceBusinessIDRow keeps simple identities compact and opaque values safely quoted.
func TestProcessInstanceBusinessIDRow(t *testing.T) {
	for _, tt := range []struct{ value, want string }{
		{"bubaId", "bid:bubaId"},
		{"order-42_v2.0@eu/a:b", "bid:order-42_v2.0@eu/a:b"},
		{"żółć123", "bid:żółć123"},
		{"", `bid:""`},
		{"two words", `bid:"two words"`},
		{"a,b", `bid:"a,b"`},
		{"a\\b", `bid:"a\\b"`},
		{"\x1b[31m", `bid:"\x1b[31m"`},
		{" new,\"ID\"\n\t|<> ", `bid:" new,\"ID\"\n\t|<> "`},
	} {
		t.Run(tt.value, func(t *testing.T) {
			item := process.ProcessInstance{Key: "101", BpmnProcessId: "order", ProcessVersion: 2, BusinessId: &tt.value}
			for _, line := range []string{oneLinePI(item), compactFlatRow(flatRowRunProcessInstanceWithTimezone(item, false))} {
				require.Contains(t, line, "order "+tt.want+" v2")
				require.NotContains(t, line, "\n")
				require.NotContains(t, line, "\x1b")
			}
		})
	}
	require.NotContains(t, oneLinePI(process.ProcessInstance{Key: "102"}), "bid:")
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
	require.Contains(t, formatProcessInstanceDryRunSelectedFinalState(summary.SelectedFinalState), `bid:old`)
	public := process.ProcessInstanceVariableUpdatePlan{Instances: []process.ProcessInstanceVariablePlan{{ProcessInstanceKey: "101", BusinessId: new("old")}}}
	update := processInstanceVariableUpdatePreviewFromFacadePlan(public)
	*public.Instances[0].BusinessId = "changed"
	require.Equal(t, "old", *update.ProcessInstances[0].BusinessId)
}
