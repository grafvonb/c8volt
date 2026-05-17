// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
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
	require.Contains(t, output, "delete preview: skipped (no matching process definitions)")
	require.Contains(t, output, "outcome: planned; no changes applied")
	require.NotContains(t, output, "candidate process-definition keys:")

	flagVerbose = true
	var verbose bytes.Buffer
	cmd = &cobra.Command{}
	cmd.SetOut(&verbose)
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult()))
	require.Contains(t, verbose.String(), "candidate process-definition keys: 2251799813685255")
	require.Contains(t, verbose.String(), "candidate process-definition details: 2251799813685255 (bpmnProcessId=invoice, version=3, versionTag=stable)")
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

	require.Contains(t, output, "delete preview: 2 process definition(s) would be deleted; 3 process instance(s) affected")
	require.NotContains(t, output, "\nprocess definitions:\n")
	require.Contains(t, output, "invoice [v1: 3, v2/stable: 0]")
	require.Contains(t, output, "active-instance blocker: 3 active process instances require --force before deletion")
	require.Contains(t, output, "outcome: planned; no changes applied")
	require.NotContains(t, output, "candidate process-definition keys:")
	require.NotContains(t, output, "affected process-instance keys:")

	flagVerbose = true
	var verbose bytes.Buffer
	cmd = &cobra.Command{}
	cmd.SetOut(&verbose)
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunPlanResult()))
	require.Contains(t, verbose.String(), "candidate process-definition keys: pd-a, pd-b")
	require.NotContains(t, verbose.String(), "affected process-instance keys:")
	require.NotContains(t, verbose.String(), "blocked process-instance keys:")
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

// TestOpsPurgeAllProcessDefinitionsJSONOutputIsDeterministic verifies dry-run machine output is stable and complete.
func TestOpsPurgeAllProcessDefinitionsJSONOutputIsDeterministic(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	render := func() string {
		var buf bytes.Buffer
		cmd := &cobra.Command{Use: "all-process-definitions"}
		cmd.SetOut(&buf)
		setContractSupport(cmd, ContractSupportFull)
		flagViewAsJson = true
		require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunPlanResult()))
		return buf.String()
	}

	first := render()
	second := render()
	require.Equal(t, first, second)
	require.NotContains(t, first, "dry run: purge all process definitions")

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(first), &envelope), first)
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "planned", payload["outcome"])
	require.Equal(t, "planned", requireJSONObject(t, payload["discovery"])["status"])
	require.Equal(t, "planned", requireJSONObject(t, payload["deletePlan"])["status"])
	require.Equal(t, "skipped", requireJSONObject(t, payload["deletion"])["status"])
}

// TestOpsPurgeAllProcessDefinitionsConfirmedDeletionUsesFrozenCandidates verifies prompted deletion submits only the planned scope.
func TestOpsPurgeAllProcessDefinitionsConfirmedDeletionUsesFrozenCandidates(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
	t.Cleanup(srv.Close)
	promptPath := filepath.Join(t.TempDir(), "prompt.txt")

	outputBytes, err := testx.RunCmdSubprocess(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_PROMPT": promptPath,
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"ops", "purge", "all-process-definitions",
			"--no-wait",
		}),
	})
	require.NoError(t, err, string(outputBytes))
	output := string(outputBytes)

	require.Contains(t, readReportFile(t, promptPath), "All process-definitions purge matched 2 candidate process definition(s)")
	require.Contains(t, output, "deletion: submitted (submitted process-definition deletes: 2)")
	require.Contains(t, output, "deletion confirmation: skipped (--no-wait)")
	require.Contains(t, output, "outcome: deleted")
	require.Contains(t, output, "elapsed:")
	require.ElementsMatch(t, []string{
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyA + "/deletion",
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyB + "/deletion",
	}, deleted.Snapshot())
	require.Equal(t, 1, countOpsPurgeAllProcessDefinitionsRequests(requests.Snapshot(), "POST /v2/process-definitions/search "))
}

// TestOpsPurgeAllProcessDefinitionsAutomationJSONExecutesWithoutAutoConfirm verifies automation confirms the supported destructive path.
func TestOpsPurgeAllProcessDefinitionsAutomationJSONExecutesWithoutAutoConfirm(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
	t.Cleanup(srv.Close)

	stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"--automation",
			"--json",
			"ops", "purge", "all-process-definitions",
			"--workers", "2",
			"--fail-fast",
			"--no-worker-limit",
			"--no-wait",
			"--force",
		}),
	})
	require.NoError(t, err, stderr)

	require.NotContains(t, stderr, "purge all process definitions")
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), stdout)
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "ops purge all-process-definitions", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "deleted", payload["outcome"])
	request := requireJSONObject(t, payload["request"])
	require.Equal(t, true, request["automation"])
	require.NotContains(t, request, "autoConfirm")
	require.Equal(t, float64(2), request["workers"])
	require.Equal(t, true, request["failFast"])
	require.Equal(t, true, request["noWorkerLimit"])
	require.Equal(t, true, request["noWait"])
	require.Equal(t, true, request["force"])
	deletion := requireJSONObject(t, payload["deletion"])
	require.Equal(t, "submitted", deletion["status"])
	require.Equal(t, true, deletion["submitted"])
	require.Equal(t, true, deletion["noWait"])
	require.Len(t, deletion["submittedProcessDefinitionKeys"], 2)
	require.ElementsMatch(t, []string{
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyA + "/deletion",
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyB + "/deletion",
	}, deleted.Snapshot())
}

// TestOpsPurgeAllProcessDefinitionsBlocksActiveInstancesBeforeMutation verifies post-planning blockers keep local-precondition exit behavior.
func TestOpsPurgeAllProcessDefinitionsBlocksActiveInstancesBeforeMutation(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 3)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"ops", "purge", "all-process-definitions",
		}),
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "local precondition failed")
	require.Contains(t, string(output), "refusing to delete all-process-definitions purge scope")
	require.Contains(t, string(output), "active process instance")
	require.Empty(t, deleted.Snapshot())
}

// TestOpsPurgeAllProcessDefinitionsDeletionOutput verifies compact execution rendering.
func TestOpsPurgeAllProcessDefinitionsDeletionOutput(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	result := sampleAllProcessDefinitionsPurgeDeletedResult()
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, result))
	output := buf.String()

	require.Contains(t, output, "purge all process definitions")
	require.Contains(t, output, "deletion: submitted (submitted process-definition deletes: 2)")
	require.Contains(t, output, "deletion confirmation: skipped (--no-wait)")
	require.Contains(t, output, "outcome: deleted")
}

// TestOpsPurgeAllProcessDefinitionsWritesMarkdownReport verifies the dry-run report includes complete audit sections.
func TestOpsPurgeAllProcessDefinitionsWritesMarkdownReport(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "all-pd-purge.md")

	outputBytes, err := testx.RunCmdSubprocess(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"ops", "purge", "all-process-definitions",
			"--dry-run",
			"--report-file", reportPath,
		}),
	})
	require.NoError(t, err, string(outputBytes))
	output := string(outputBytes)

	require.Contains(t, output, "outcome: planned; no changes applied")
	require.Contains(t, output, "report: written "+reportPath)
	require.Less(t, strings.Index(output, "report: written "+reportPath), strings.Index(output, "outcome: planned; no changes applied"))
	require.Empty(t, deleted.Snapshot())
	report := readReportFile(t, reportPath)
	require.Contains(t, report, "# All Process Definitions Purge Audit Report")
	require.Contains(t, report, "- Command: ops purge all-process-definitions")
	require.Contains(t, report, "- Dry Run: true")
	require.Contains(t, report, "- Camunda Version: 8.9")
	require.Contains(t, report, "- Profile: default")
	require.Contains(t, report, "- Outcome: planned")
	require.Contains(t, report, "## Discovery")
	require.Contains(t, report, "- Candidate Process-Definition Keys:")
	require.Contains(t, report, "  - "+opsAllProcessDefinitionsPurgePDKeyA)
	require.Contains(t, report, "## Delete Plan")
	require.Contains(t, report, "- Affected Process Instances: 0")
}

// TestOpsPurgeAllProcessDefinitionsWritesJSONReport verifies confirmed runs overwrite only after deletion submission.
func TestOpsPurgeAllProcessDefinitionsWritesJSONReport(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "all-pd-purge.json")
	require.NoError(t, os.WriteFile(reportPath, []byte("old report"), 0o600))

	outputBytes, err := testx.RunCmdSubprocess(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"ops", "purge", "all-process-definitions",
			"--auto-confirm",
			"--no-wait",
			"--report-file", reportPath,
			"--report-format", "json",
		}),
	})
	require.NoError(t, err, string(outputBytes))
	output := string(outputBytes)

	require.Contains(t, output, "outcome: deleted")
	require.Contains(t, output, "report: written "+reportPath)
	require.Less(t, strings.Index(output, "report: written "+reportPath), strings.Index(output, "outcome: deleted"))
	require.NotContains(t, readReportFile(t, reportPath), "old report")
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "ops.all-process-definitions.v1", report["schemaVersion"])
	require.Equal(t, "ops purge all-process-definitions", report["commandName"])
	require.Equal(t, "deleted", report["outcome"])
	require.Equal(t, true, report["noWait"])
	require.Equal(t, "8.9", report["camundaVersion"])
	discovery := requireJSONObject(t, report["discovery"])
	require.Equal(t, float64(2), discovery["candidateProcessDefinitionCount"])
	require.Len(t, discovery["candidateProcessDefinitionKeys"], 2)
	deletePlan := requireJSONObject(t, report["deletePlan"])
	require.Len(t, deletePlan["candidateProcessDefinitionKeys"], 2)
	deletion := requireJSONObject(t, report["deletion"])
	require.Equal(t, "submitted", deletion["status"])
	require.Equal(t, true, deletion["submitted"])
	require.ElementsMatch(t, []string{
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyA + "/deletion",
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyB + "/deletion",
	}, deleted.Snapshot())
}

// TestOpsPurgeAllProcessDefinitionsExistingReportPreservation verifies non-submitted paths never clobber reports.
func TestOpsPurgeAllProcessDefinitionsExistingReportPreservation(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		activeCount  int64
		want         string
		wantRequests bool
	}{
		{
			name: "dry run",
			args: []string{"ops", "purge", "all-process-definitions", "--dry-run"},
			want: "report file already exists:",
		},
		{
			name: "unconfirmed",
			args: []string{"ops", "purge", "all-process-definitions"},
			want: "report file already exists:",
		},
		{
			name:         "locally blocked",
			args:         []string{"ops", "purge", "all-process-definitions", "--auto-confirm"},
			activeCount:  3,
			want:         "write audit report: report file already exists:",
			wantRequests: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			var deleted testx.SafeSlice[string]
			srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, tt.activeCount)
			t.Cleanup(srv.Close)
			reportPath := filepath.Join(t.TempDir(), "all-pd-purge.md")
			const existingReport = "existing report"
			require.NoError(t, os.WriteFile(reportPath, []byte(existingReport), 0o600))
			args := append([]string{}, tt.args...)
			args = append(args, "--report-file", reportPath)

			output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":            writeTestConfigForVersion(t, srv.URL, "8.9"),
				"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, args),
			})
			require.Error(t, err)

			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.Error, exitErr.ExitCode())
			require.Contains(t, string(output), tt.want)
			require.Equal(t, existingReport, readReportFile(t, reportPath))
			require.Empty(t, deleted.Snapshot())
			if tt.wantRequests {
				require.NotEmpty(t, requests.Snapshot())
			} else {
				require.Empty(t, requests.Snapshot())
			}
		})
	}
}

// TestOpsPurgeAllProcessDefinitionsCommandHelper runs all-process-definitions purge command subprocess cases.
func TestOpsPurgeAllProcessDefinitionsCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_ALL_PD_PURGE_ARGS")), &args); err != nil {
		t.Fatalf("invalid helper args: %v", err)
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetOpsPurgeAllProcessDefinitionsFlagState()
	if promptPath := os.Getenv("C8VOLT_TEST_ALL_PD_PURGE_PROMPT"); promptPath != "" {
		prevConfirm := confirmCmdOrAbortFn
		defer func() { confirmCmdOrAbortFn = prevConfirm }()
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			if autoConfirm {
				return fmt.Errorf("unexpected auto-confirm prompt")
			}
			return os.WriteFile(promptPath, []byte(prompt), 0o600)
		}
	}
	root.SetArgs(append([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG")}, args...))
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		handleBootstrapError(root, err)
	}
	os.Exit(0)
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

// sampleAllProcessDefinitionsPurgeDeletedResult returns a successful no-wait deletion result for command rendering tests.
func sampleAllProcessDefinitionsPurgeDeletedResult() ops.AllProcessDefinitionsPurgeResult {
	result := sampleAllProcessDefinitionsPurgeDryRunPlanResult()
	result.Request.DryRun = false
	result.Outcome = ops.AllProcessDefinitionsPurgeOutcomeDeleted
	result.DeletePlan.RequiresForce = false
	result.Deletion = ops.AllProcessDefinitionsPurgeDeletionResult{
		Status:                         ops.WorkflowStepStatusSubmitted,
		SubmittedProcessDefinitionKeys: typex.Keys{"pd-a", "pd-b"},
		Items: []resource.DeleteReport{
			{Key: "pd-a", Ok: true, StatusCode: http.StatusOK, Status: "200 OK"},
			{Key: "pd-b", Ok: true, StatusCode: http.StatusOK, Status: "200 OK"},
		},
		Submitted: true,
		NoWait:    true,
	}
	return result
}

// sampleAllProcessDefinitionsPurgeDryRunPlanResult returns a planned purge result for command rendering tests.
func sampleAllProcessDefinitionsPurgeDryRunPlanResult() ops.AllProcessDefinitionsPurgeResult {
	result := sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult()
	result.Discovery.CandidateProcessDefinitionKeys = typex.Keys{"pd-a", "pd-b"}
	result.Discovery.CandidateProcessDefinitions = []process.ProcessDefinition{
		{Key: "pd-a", BpmnProcessId: "invoice", ProcessVersion: 1},
		{Key: "pd-b", BpmnProcessId: "invoice", ProcessVersion: 2, ProcessVersionTag: "stable"},
	}
	result.Discovery.CandidateProcessDefinitionCount = 2
	result.DeletePlan = ops.AllProcessDefinitionsPurgeDeletePlan{
		Status:                         ops.WorkflowStepStatusPlanned,
		CandidateProcessDefinitionKeys: typex.Keys{"pd-a", "pd-b"},
		Items: []resource.DeleteProcessDefinitionPlanItem{
			{
				Key:                        "pd-a",
				ActiveProcessInstanceCount: 3,
				ActiveProcessInstanceKeys:  []string{"pi-a", "pi-b", "pi-c"},
				CancellationPlan: process.DryRunPIKeyExpansion{
					Collected: typex.Keys{"pi-a", "pi-b", "pi-c"},
					RequiresCancelBeforeDelete: []process.ProcessInstance{
						{Key: "pi-a"},
						{Key: "pi-b"},
						{Key: "pi-c"},
					},
				},
			},
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

// marshalOpsPurgeAllProcessDefinitionsArgsForEnv preserves argument boundaries for subprocess helpers.
func marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t *testing.T, args []string) string {
	t.Helper()

	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
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
	flagViewKeysOnly = false
	flagVerbose = false
}

const (
	opsAllProcessDefinitionsPurgePDKeyA = "2251799813685255"
	opsAllProcessDefinitionsPurgePDKeyB = "2251799813685256"
)

func newOpsPurgeAllProcessDefinitionsServer(t *testing.T, requests *testx.SafeSlice[string], deleted *testx.SafeSlice[string], activeCount int64) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			_, _ = w.Write([]byte(`{"items":[` +
				opsAllProcessDefinitionsPurgeDefinitionJSON(opsAllProcessDefinitionsPurgePDKeyA, "invoice", 2) + `,` +
				opsAllProcessDefinitionsPurgeDefinitionJSON(opsAllProcessDefinitionsPurgePDKeyA, "invoice", 2) + `,` +
				opsAllProcessDefinitionsPurgeDefinitionJSON(opsAllProcessDefinitionsPurgePDKeyB, "payment", 1) +
				`],"page":{"totalItems":3,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-definitions/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-definitions/")
			requests.Append(r.Method + " " + r.URL.Path)
			_, _ = w.Write([]byte(opsAllProcessDefinitionsPurgeDefinitionJSON(key, "invoice", 1)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			payload := string(body)
			requests.Append(r.Method + " " + r.URL.Path + " " + payload)
			total := int64(0)
			if strings.Contains(payload, "ACTIVE") {
				total = activeCount
			}
			_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[],"page":{"totalItems":%d,"hasMoreTotalItems":false}}`, total)))
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v2/resources/") && strings.HasSuffix(r.URL.Path, "/deletion"):
			if deleted != nil {
				deleted.Append(r.URL.Path)
			}
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/resources/"), "/deletion")
			_, _ = w.Write([]byte(`{"resourceKey":"` + key + `","batchOperation":{"batchOperationKey":"batch-` + key + `","batchOperationType":"DELETE_PROCESS_DEFINITION"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func opsAllProcessDefinitionsPurgeDefinitionJSON(key string, id string, version int32) string {
	return fmt.Sprintf(`{"processDefinitionKey":"%s","processDefinitionId":"%s","name":"%s","version":%d,"tenantId":"tenant","versionTag":"stable"}`, key, id, id, version)
}

func countOpsPurgeAllProcessDefinitionsRequests(items []string, prefix string) int {
	count := 0
	for _, item := range items {
		if strings.HasPrefix(item, prefix) {
			count++
		}
	}
	return count
}
