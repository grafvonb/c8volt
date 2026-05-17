// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestRenderOpsRepairIncidentDryRunSearchHumanOutput verifies dry-run search output exposes filters and frozen keys.
func TestRenderOpsRepairIncidentDryRunSearchHumanOutput(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	flagVerbose = true
	cmd, buf := newOpsRepairIncidentRenderTestCommand()

	err := renderOpsRepairIncidentResult(cmd, ops.RepairResult{
		Request: ops.RepairRequest{
			DryRun:        true,
			DiscoveryMode: ops.RepairDiscoveryModeSearch,
			IncidentSelection: incident.Filter{
				State:     "active",
				ErrorType: "io_mapping_error",
			},
		},
		FrozenSet: ops.RepairFrozenSet{
			DiscoveryMode:       ops.RepairDiscoveryModeSearch,
			IncidentKeys:        typex.Keys{"2251799813685249", "2251799813685250"},
			ProcessInstanceKeys: typex.Keys{"2251799813685251"},
			IncidentFilters: incident.Filter{
				State:     "active",
				ErrorType: "io_mapping_error",
			},
		},
		Plan: []ops.RepairPlanItem{
			{IncidentKey: "2251799813685249", JobKey: "2251799813685252", RetryUpdateStatus: ops.WorkflowStepStatusPlanned, TimeoutUpdateStatus: ops.WorkflowStepStatusSkipped, ResolutionStatus: ops.WorkflowStepStatusPlanned, ConfirmationStatus: ops.WorkflowStepStatusSkipped},
			{IncidentKey: "2251799813685250", RetryUpdateStatus: ops.WorkflowStepStatusNotApplicable, TimeoutUpdateStatus: ops.WorkflowStepStatusNotApplicable, ResolutionStatus: ops.WorkflowStepStatusPlanned, ConfirmationStatus: ops.WorkflowStepStatusSkipped},
		},
		Outcome: ops.RepairOutcomePlanned,
	})

	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "dry run: repair incidents")
	require.Contains(t, output, `discovery: search filters {state=active, errorType="io_mapping_error"}`)
	require.Contains(t, output, "frozen incidents: 2")
	require.Contains(t, output, "incident keys: 2251799813685249, 2251799813685250")
	require.Contains(t, output, "outcome: planned; no changes applied")
}

// TestRenderOpsRepairIncidentDryRunSearchJSON verifies the shared envelope preserves dry-run search fields.
func TestRenderOpsRepairIncidentDryRunSearchJSON(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	flagViewAsJson = true
	cmd, buf := newOpsRepairIncidentRenderTestCommand()
	setContractSupport(cmd, ContractSupportFull)

	err := renderOpsRepairIncidentResult(cmd, ops.RepairResult{
		Request: ops.RepairRequest{
			DryRun:        true,
			DiscoveryMode: ops.RepairDiscoveryModeSearch,
			IncidentSelection: incident.Filter{
				State: "active",
			},
		},
		FrozenSet: ops.RepairFrozenSet{
			DiscoveryMode: ops.RepairDiscoveryModeSearch,
			IncidentKeys:  typex.Keys{"2251799813685249"},
			IncidentFilters: incident.Filter{
				State: "active",
			},
		},
		Plan:    []ops.RepairPlanItem{{IncidentKey: "2251799813685249", ResolutionStatus: ops.WorkflowStepStatusPlanned}},
		Outcome: ops.RepairOutcomePlanned,
	})

	require.NoError(t, err)
	output := buf.String()
	require.True(t, strings.HasPrefix(output, "{"))
	require.Contains(t, output, `"outcome": "succeeded"`)
	require.Contains(t, output, `"discoveryMode": "search"`)
	require.Contains(t, output, `"incidentFilters": {`)
	require.Contains(t, output, `"incidentKeys": [`)
}

// newOpsRepairIncidentRenderTestCommand captures renderer output without building the full root command tree.
func newOpsRepairIncidentRenderTestCommand() (*cobra.Command, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cmd := &cobra.Command{Use: "incident"}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd, buf
}
