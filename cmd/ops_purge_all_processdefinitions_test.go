// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestOpsPurgeAllProcessDefinitionsHelpDocumentsCommandShape verifies the registered command, alias, and safe examples.
func TestOpsPurgeAllProcessDefinitionsHelpDocumentsCommandShape(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	output := executeRootForTest(t, "ops", "purge", "all-process-definitions", "--help")

	assertHelpOutputContainsAll(t, output,
		"Purge all selected process definitions",
		"Aliases:",
		"all-pds",
		"--key string",
		"--bpmn-process-id string",
		"--pd-version int32",
		"--pd-version-tag string",
		"--latest",
		"--dry-run",
		"--workers int",
		"--no-worker-limit",
		"--fail-fast",
		"--no-wait",
		"--force",
		"--report-file string",
		"--report-format string",
		"./c8volt ops purge all-process-definitions --dry-run",
		"./c8volt ops purge all-pds --bpmn-process-id invoice --latest --dry-run",
		"./c8volt ops purge all-process-definitions --automation --json --dry-run",
	)
	assertHelpOutputOmitsAll(t, output,
		"purge-definitions",
		"delete-all",
		"./c8volt ops purge all-process-definitions --automation --json\n",
		"--xml",
		"--stat",
	)

	aliasOutput := executeRootForTest(t, "ops", "purge", "all-pds", "--help")
	require.Contains(t, aliasOutput, "Purge all selected process definitions")
}

// TestOpsPurgeAllProcessDefinitionsRejectsDisplayOnlyPDFlags keeps get-pd display flags out of the purge surface.
func TestOpsPurgeAllProcessDefinitionsRejectsDisplayOnlyPDFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "xml", args: []string{"--xml"}, want: "unknown flag: --xml"},
		{name: "stat", args: []string{"--stat"}, want: "unknown flag: --stat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeOpsPurgeAllProcessDefinitionsExpectError(t, tt.args...)
			require.Error(t, err)
			require.Contains(t, output, tt.want)
		})
	}
}

// TestOpsPurgeAllProcessDefinitionsInvalidFlagsUseInvalidInput verifies local flag validation before remote work.
func TestOpsPurgeAllProcessDefinitionsInvalidFlagsUseInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "invalid key",
			args: []string{"--key", "not-a-key"},
			want: `process definition key "not-a-key" is not a valid key`,
		},
		{
			name: "zero explicit process definition version",
			args: []string{"--pd-version", "0"},
			want: "--pd-version must be positive integer",
		},
		{
			name: "negative process definition version",
			args: []string{"--pd-version", "-1"},
			want: "--pd-version must be positive integer",
		},
		{
			name: "invalid worker count",
			args: []string{"--workers", "0"},
			want: "--workers must be positive integer",
		},
		{
			name: "report format without file",
			args: []string{"--report-format", "json"},
			want: "--report-format requires --report-file",
		},
		{
			name: "unsupported report format",
			args: []string{"--report-file", "purge.txt", "--report-format", "yaml"},
			want: `unsupported ops workflow report format "yaml"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeOpsPurgeAllProcessDefinitionsExpectError(t, tt.args...)
			require.Error(t, err)
			require.Contains(t, output, tt.want)
		})
	}
}

// TestOpsPurgeAllProcessDefinitionsDryRunDiscoveryOutput verifies compact discovery output for dry-run previews.
func TestOpsPurgeAllProcessDefinitionsDryRunDiscoveryOutput(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult()))
	output := buf.String()

	require.Contains(t, output, "dry run: purge all process definitions")
	require.Contains(t, output, `selection filters: {bpmnProcessId="invoice", processVersion=3, processVersionTag="stable", latestOnly=true}`)
	require.Contains(t, output, "candidate process definitions: 1")
	require.Contains(t, output, "candidate scope: latest matching process definitions")
	require.Contains(t, output, "duplicate candidate process definitions: 1")
	require.Contains(t, output, "delete plan: skipped")
	require.Contains(t, output, "outcome: planned; no changes applied; use --verbose to list process-definition keys")
	require.NotContains(t, output, "candidate process-definition keys:")

	flagVerbose = true
	var verbose bytes.Buffer
	cmd = &cobra.Command{}
	cmd.SetOut(&verbose)
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult()))
	require.Contains(t, verbose.String(), "candidate process-definition keys: 2251799813685255")
	require.Contains(t, verbose.String(), "duplicate candidate process-definition keys: 2251799813685255")
}

// TestOpsPurgeAllProcessDefinitionsDryRunJSONDiscoveryData verifies machine output carries complete discovery fields.
func TestOpsPurgeAllProcessDefinitionsDryRunJSONDiscoveryData(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "all-process-definitions"}
	cmd.SetOut(&buf)
	setContractSupport(cmd, ContractSupportFull)
	flagViewAsJson = true
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult()))

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope), buf.String())
	require.Equal(t, "succeeded", envelope["outcome"])
	payload := requireJSONObject(t, envelope["payload"])
	discovery := requireJSONObject(t, payload["discovery"])
	require.Equal(t, "planned", discovery["status"])
	require.Equal(t, float64(1), discovery["candidateProcessDefinitionCount"])
	require.Equal(t, true, discovery["latestOnly"])
	require.Len(t, discovery["candidateProcessDefinitionKeys"], 1)
	require.Len(t, discovery["candidateProcessDefinitions"], 1)
	require.Len(t, discovery["duplicateCandidateProcessDefinitionKeys"], 1)
	require.Len(t, discovery["notices"], 2)
	require.Equal(t, "skipped", requireJSONObject(t, payload["deletePlan"])["status"])
	require.Equal(t, "skipped", requireJSONObject(t, payload["deletion"])["status"])
}

// TestOpsPurgeAllProcessDefinitionsDryRunPlanOutput verifies compact delete-plan rendering after discovery.
func TestOpsPurgeAllProcessDefinitionsDryRunPlanOutput(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunPlanResult()))
	output := buf.String()

	require.Contains(t, output, "delete plan: planned (candidate process definitions: 2, affected process instances: 3)")
	require.Contains(t, output, "active-instance blocker: 3 active process instances require --force before deletion")
	require.Contains(t, output, "outcome: planned; no changes applied; use --verbose to list process-definition keys")
	require.NotContains(t, output, "candidate process-definition keys:")

	flagVerbose = true
	var verbose bytes.Buffer
	cmd = &cobra.Command{}
	cmd.SetOut(&verbose)
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunPlanResult()))
	require.Contains(t, verbose.String(), "candidate process-definition keys: pd-a, pd-b")
}

// TestOpsPurgeAllProcessDefinitionsDryRunJSONPlanData verifies machine output carries complete delete-plan fields.
func TestOpsPurgeAllProcessDefinitionsDryRunJSONPlanData(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "all-process-definitions"}
	cmd.SetOut(&buf)
	setContractSupport(cmd, ContractSupportFull)
	flagViewAsJson = true
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunPlanResult()))

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope), buf.String())
	payload := requireJSONObject(t, envelope["payload"])
	plan := requireJSONObject(t, payload["deletePlan"])
	require.Equal(t, "planned", plan["status"])
	require.Len(t, plan["candidateProcessDefinitionKeys"], 2)
	require.Len(t, plan["duplicateCandidateProcessDefinitionKeys"], 1)
	require.Len(t, plan["items"], 2)
	require.Equal(t, float64(3), plan["affectedProcessInstanceCount"])
	require.Equal(t, float64(3), plan["activeProcessInstanceCount"])
	require.Equal(t, true, plan["requiresForce"])
	require.Equal(t, true, plan["requiresConfirmation"])
}

// executeOpsPurgeAllProcessDefinitionsExpectError runs all-process-definitions purge and returns Cobra parse/validation errors.
func executeOpsPurgeAllProcessDefinitionsExpectError(t *testing.T, args ...string) (string, error) {
	t.Helper()

	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	root := Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(append([]string{"ops", "purge", "all-process-definitions"}, args...))
	resetCommandTreeFlags(root)
	resetOpsPurgeAllProcessDefinitionsFlagState()

	_, err := root.ExecuteC()
	if err != nil {
		return buf.String() + err.Error(), err
	}
	return buf.String(), nil
}

// sampleAllProcessDefinitionsPurgeDryRunPlanResult returns a planned purge result for command rendering tests.
func sampleAllProcessDefinitionsPurgeDryRunPlanResult() ops.AllProcessDefinitionsPurgeResult {
	result := sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult()
	result.Discovery.CandidateProcessDefinitionKeys = typex.Keys{"pd-a", "pd-b"}
	result.Discovery.CandidateProcessDefinitionCount = 2
	result.DeletePlan = ops.AllProcessDefinitionsPurgeDeletePlan{
		Status:                         ops.WorkflowStepStatusPlanned,
		CandidateProcessDefinitionKeys: typex.Keys{"pd-a", "pd-b"},
		Items: []resource.DeleteProcessDefinitionPlanItem{
			{Key: "pd-a", ActiveProcessInstanceCount: 3, ActiveProcessInstanceKeys: []string{"pi-a", "pi-b", "pi-c"}},
			{Key: "pd-b"},
		},
		DuplicateCandidateProcessDefinitionKeys: typex.Keys{"pd-a"},
		AffectedProcessInstanceCount:            3,
		ActiveProcessInstanceCount:              3,
		RequiresConfirmation:                    true,
		RequiresForce:                           true,
	}
	return result
}

// sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult returns a discovery-only purge result for command rendering tests.
func sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult() ops.AllProcessDefinitionsPurgeResult {
	return ops.AllProcessDefinitionsPurgeResult{
		Request: ops.AllProcessDefinitionsPurgeRequest{
			CommandName: "ops purge all-process-definitions",
			DryRun:      true,
			Selection: ops.ProcessDefinitionSelection{
				BpmnProcessId:     "invoice",
				ProcessVersion:    3,
				ProcessVersionTag: "stable",
				LatestOnly:        true,
			},
		},
		Discovery: ops.ProcessDefinitionDiscoveryResult{
			Status:                         ops.WorkflowStepStatusPlanned,
			Filters:                        ops.ProcessDefinitionSelection{BpmnProcessId: "invoice", ProcessVersion: 3, ProcessVersionTag: "stable", LatestOnly: true},
			CandidateProcessDefinitionKeys: typex.Keys{"2251799813685255"},
			CandidateProcessDefinitions: []process.ProcessDefinition{{
				Key:               "2251799813685255",
				BpmnProcessId:     "invoice",
				ProcessVersion:    3,
				ProcessVersionTag: "stable",
			}},
			DuplicateCandidateProcessDefinitionKeys: typex.Keys{"2251799813685255"},
			CandidateProcessDefinitionCount:         1,
			LatestOnly:                              true,
			Notices: []ops.AllProcessDefinitionsPurgeNotice{
				{Code: "latest_only_scope", Severity: "info", Message: "candidate discovery was narrowed to latest matching process definitions"},
				{Code: "duplicate_candidate_process_definitions", Severity: "info", Message: "duplicate candidate process-definition keys detected"},
			},
		},
		DeletePlan: ops.AllProcessDefinitionsPurgeDeletePlan{Status: ops.WorkflowStepStatusSkipped},
		Deletion:   ops.AllProcessDefinitionsPurgeDeletionResult{Status: ops.WorkflowStepStatusSkipped},
		Outcome:    ops.AllProcessDefinitionsPurgeOutcomePlanned,
	}
}

// resetOpsPurgeAllProcessDefinitionsFlagState restores all-process-definitions purge globals between command tests.
func resetOpsPurgeAllProcessDefinitionsFlagState() {
	flagOpsPurgeAllPDKey = ""
	flagOpsPurgeAllPDBpmnProcessID = ""
	flagOpsPurgeAllPDProcessVersion = 0
	flagOpsPurgeAllPDProcessVersionTag = ""
	flagOpsPurgeAllPDLatest = false
	flagOpsPurgeAllPDReportFile = ""
	flagOpsPurgeAllPDReportFormat = ""
	flagDryRun = false
	flagWorkers = 0
	flagNoWorkerLimit = false
	flagFailFast = false
	flagNoWait = false
	flagForce = false
	flagCmdAutoConfirm = false
	flagViewAsJson = false
	flagVerbose = false
}
