// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
			ReportFile:    "repair-preview.json",
			ReportFormat:  "json",
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
	require.Contains(t, output, "report: written repair-preview.json")
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
			ReportFile:    "repair-preview.md",
			ReportFormat:  "markdown",
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
	require.Contains(t, output, `"reportFile": "repair-preview.md"`)
	require.Contains(t, output, `"reportFormat": "markdown"`)
	require.Contains(t, output, `"incidentFilters": {`)
	require.Contains(t, output, `"incidentKeys": [`)
}

// TestRenderOpsRepairMarkdownReport verifies repair Markdown is derived from the structured audit model.
func TestRenderOpsRepairMarkdownReport(t *testing.T) {
	report := newOpsRepairRenderTestReport()

	data, err := renderOpsRepairMarkdownReport(report, nil)

	require.NoError(t, err)
	output := string(data)
	require.Contains(t, output, "# Repair Audit Report")
	require.Contains(t, output, "- Schema Version: ops.repair.v1")
	require.Contains(t, output, "- Command: ops repair incident")
	require.Contains(t, output, "- Dry Run: true")
	require.Contains(t, output, "- Outcome: planned")
	require.Contains(t, output, "## Frozen Targets")
	require.Contains(t, output, "  - 2251799813685249")
	require.Contains(t, output, "## Variable Updates")
	require.Contains(t, output, "  - scope=2251799813685251 status=planned names=approved dependents=2251799813685249")
	require.Contains(t, output, "## Incident Steps")
	require.Contains(t, output, "  - incident=2251799813685249 processInstance=2251799813685251 job=2251799813685252 vars=planned retry=planned timeout=skipped resolution=planned confirmation=skipped")
}

// TestRenderOpsRepairJSONReport verifies repair JSON reports keep the complete structured model.
func TestRenderOpsRepairJSONReport(t *testing.T) {
	report := newOpsRepairRenderTestReport()
	report.Outcome = ops.RepairOutcomePartiallyFailed
	report.Errors = []string{"resolution failed"}

	data, err := renderOpsRepairJSONReport(report)

	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, "ops.repair.v1", got["schemaVersion"])
	require.Equal(t, "ops repair incident", got["commandName"])
	require.Equal(t, "partially_failed", got["outcome"])
	require.Equal(t, true, got["dryRun"])
	require.Len(t, requireJSONObject(t, got["frozenSet"])["incidentKeys"], 1)
	require.Len(t, got["plan"], 1)
	require.Len(t, got["errors"], 1)
}

// newOpsRepairIncidentRenderTestCommand captures renderer output without building the full root command tree.
func newOpsRepairIncidentRenderTestCommand() (*cobra.Command, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cmd := &cobra.Command{Use: "incident"}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd, buf
}

// newOpsRepairRenderTestReport builds a representative report with every major repair section populated.
func newOpsRepairRenderTestReport() ops.RepairAuditReport {
	retries := int32(1)
	started := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	return ops.RepairAuditReport{
		SchemaVersion:   "ops.repair.v1",
		CommandName:     "ops repair incident",
		StartedAt:       started,
		FinishedAt:      started.Add(time.Second),
		Duration:        time.Second.String(),
		DryRun:          true,
		C8voltVersion:   "test",
		CamundaVersion:  "8.9",
		ProfileIdentity: "default",
		Request: ops.RepairRequest{
			CommandName:         "ops repair incident",
			Target:              ops.RepairTargetIncident,
			DiscoveryMode:       ops.RepairDiscoveryModeKeyed,
			InputKeys:           typex.Keys{"2251799813685249"},
			DryRun:              true,
			RequestedRetries:    &retries,
			ReportFile:          "repair.md",
			ReportFormat:        "markdown",
			Variables:           map[string]any{"approved": true},
			RequestedJobTimeout: 0,
		},
		FrozenSet: ops.RepairFrozenSet{
			Status:              ops.WorkflowStepStatusConfirmed,
			Target:              ops.RepairTargetIncident,
			DiscoveryMode:       ops.RepairDiscoveryModeKeyed,
			InputKeys:           typex.Keys{"2251799813685249"},
			IncidentKeys:        typex.Keys{"2251799813685249"},
			ProcessInstanceKeys: typex.Keys{"2251799813685251"},
			JobKeys:             typex.Keys{"2251799813685252"},
			VariableScopes:      typex.Keys{"2251799813685251"},
		},
		Plan: []ops.RepairPlanItem{{
			IncidentKey:            "2251799813685249",
			ProcessInstanceKey:     "2251799813685251",
			JobKey:                 "2251799813685252",
			VariableScopeKey:       "2251799813685251",
			RequestedVariableNames: []string{"approved"},
			RequestedRetries:       &retries,
			VariableUpdateStatus:   ops.WorkflowStepStatusPlanned,
			RetryUpdateStatus:      ops.WorkflowStepStatusPlanned,
			TimeoutUpdateStatus:    ops.WorkflowStepStatusSkipped,
			ResolutionStatus:       ops.WorkflowStepStatusPlanned,
			ConfirmationStatus:     ops.WorkflowStepStatusSkipped,
		}},
		VariableUpdates: []ops.RepairVariableScopeUpdate{{
			ScopeKey:              "2251799813685251",
			VariableNames:         []string{"approved"},
			DependentIncidentKeys: typex.Keys{"2251799813685249"},
			Status:                ops.WorkflowStepStatusPlanned,
		}},
		JobApplicability: []ops.RepairJobApplicability{{
			IncidentKey:      "2251799813685249",
			JobKey:           "2251799813685252",
			RetryStatus:      ops.WorkflowStepStatusPlanned,
			TimeoutStatus:    ops.WorkflowStepStatusSkipped,
			RequestedRetries: &retries,
		}},
		Remaining: ops.RepairRemainingIncidentSummary{
			Status:  ops.WorkflowStepStatusSkipped,
			Checked: false,
		},
		Outcome: ops.RepairOutcomePlanned,
	}
}
