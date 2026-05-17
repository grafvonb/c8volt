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

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestOpsPurgeProcessInstancesWithIncidentsHelpDocumentsCommandShape verifies the registered command, alias, and safe examples.
func TestOpsPurgeProcessInstancesWithIncidentsHelpDocumentsCommandShape(t *testing.T) {
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)

	output := executeRootForProcessInstanceTest(t, "ops", "purge", "process-instances-with-incidents", "--help")

	assertHelpOutputContainsAll(t, output,
		"Purge process instances selected by incidents",
		"Aliases:",
		"pi-with-incidents",
		"--key strings",
		"--state string",
		"--error-type string",
		"--error-message string",
		"--bpmn-process-id string",
		"--pd-key string",
		"--pi-key string",
		"--root-key string",
		"--flow-node-id string",
		"--fni-key string",
		"--creation-time-after string",
		"--creation-time-before string",
		"--batch-size int32",
		"--limit int32",
		"--dry-run",
		"--workers int",
		"--no-worker-limit",
		"--fail-fast",
		"--no-wait",
		"--force",
		"--report-file string",
		"--report-format string",
		"./c8volt ops purge process-instances-with-incidents --automation --json --dry-run",
	)
	assertHelpOutputOmitsAll(t, output,
		"incident-pis",
		"./c8volt ops purge process-instances-with-incidents --automation --json\n",
		"--pi-keys-only",
		"--total",
		"--error-message-limit",
		"--with-no-error-message",
	)

	aliasOutput := executeRootForProcessInstanceTest(t, "ops", "purge", "pi-with-incidents", "--help")
	require.Contains(t, aliasOutput, "Purge process instances selected by incidents")
}

// TestOpsPurgeProcessInstancesWithIncidentsRejectsIncidentDisplayOnlyFlags keeps display flags out of the purge surface.
func TestOpsPurgeProcessInstancesWithIncidentsRejectsIncidentDisplayOnlyFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "pi keys only", args: []string{"--pi-keys-only"}, want: "unknown flag: --pi-keys-only"},
		{name: "total", args: []string{"--total"}, want: "unknown flag: --total"},
		{name: "message limit", args: []string{"--error-message-limit", "20"}, want: "unknown flag: --error-message-limit"},
		{name: "omit message", args: []string{"--with-no-error-message"}, want: "unknown flag: --with-no-error-message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeOpsPurgeProcessInstancesWithIncidentsExpectError(t, tt.args...)
			require.Error(t, err)
			require.Contains(t, output, tt.want)
		})
	}
}

// TestOpsPurgeProcessInstancesWithIncidentsInvalidFlagsUseInvalidInput verifies registered static validation failures.
func TestOpsPurgeProcessInstancesWithIncidentsInvalidFlagsUseInvalidInput(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:9", "8.8")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "invalid state",
			args: []string{"ops", "purge", "process-instances-with-incidents", "--state", "open"},
			want: `invalid value for --state: "open"`,
		},
		{
			name: "zero explicit limit",
			args: []string{"ops", "purge", "process-instances-with-incidents", "--limit", "0"},
			want: "--limit must be positive integer",
		},
		{
			name: "batch size too large",
			args: []string{"ops", "purge", "process-instances-with-incidents", "--batch-size", "1001"},
			want: "invalid value for --batch-size: 1001, expected positive integer up to 1000",
		},
		{
			name: "invalid worker count",
			args: []string{"ops", "purge", "process-instances-with-incidents", "--workers", "0"},
			want: "--workers must be positive integer",
		},
		{
			name: "invalid incident key",
			args: []string{"ops", "purge", "process-instances-with-incidents", "--key", "not-a-key"},
			want: `incident key "not-a-key" is not a valid key`,
		},
		{
			name: "invalid report format",
			args: []string{"ops", "purge", "process-instances-with-incidents", "--report-file", "incident-purge.txt", "--report-format", "yaml"},
			want: `unsupported ops workflow report format "yaml"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeProcessInstancesWithIncidentsInvalidFlagsHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":              cfgPath,
				"C8VOLT_TEST_INCIDENT_PURGE_ARGS": marshalOpsPurgeProcessInstancesWithIncidentsArgsForEnv(t, tt.args),
			})
			require.Error(t, err)

			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
			require.Contains(t, string(output), "invalid input")
			require.Contains(t, string(output), tt.want)
			require.NotContains(t, string(output), "Usage:")
		})
	}
}

// TestOpsPurgeProcessInstancesWithIncidentsDryRunDiscoveryOutput verifies dry-run discovery freezes incident candidates without delete requests.
func TestOpsPurgeProcessInstancesWithIncidentsDryRunDiscoveryOutput(t *testing.T) {
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	require.NoError(t, renderOpsPurgeProcessInstancesWithIncidentsResult(cmd, sampleIncidentPurgeDryRunPlanResult()))
	output := buf.String()

	require.Contains(t, output, "dry run: purge process-instances with incidents")
	require.Contains(t, output, `selection filters: {state=active}`)
	require.Contains(t, output, "candidate incidents: 3")
	require.Contains(t, output, "candidate process instances: 1")
	require.Contains(t, output, "duplicate candidate process instances: 1")
	require.Contains(t, output, "skipped incidents: 1")
	require.Contains(t, output, "delete plan: planned (candidate process instances: 1, roots: 1, affected process instances: 1)")
	require.Contains(t, output, "outcome: planned; no changes applied; use --verbose to list process-instance keys")
}

// TestOpsPurgeProcessInstancesWithIncidentsDryRunJSONDiscoveryData verifies machine output carries complete discovery fields.
func TestOpsPurgeProcessInstancesWithIncidentsDryRunJSONDiscoveryData(t *testing.T) {
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)

	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "process-instances-with-incidents"}
	cmd.SetOut(&buf)
	require.NoError(t, renderSucceededResult(cmd, sampleIncidentPurgeDryRunPlanResult()))
	output := buf.String()

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, "succeeded", envelope["outcome"])
	payload := requireJSONObject(t, envelope["payload"])
	discovery := requireJSONObject(t, payload["discovery"])
	require.Equal(t, float64(3), discovery["incidentCount"])
	require.Equal(t, float64(1), discovery["candidateProcessInstanceCount"])
	require.Len(t, discovery["incidentKeys"], 3)
	require.Len(t, discovery["candidateProcessInstanceKeys"], 1)
	require.Len(t, discovery["duplicateCandidateProcessInstanceKeys"], 1)
	require.Len(t, discovery["skippedIncidents"], 1)
	require.Len(t, discovery["notices"], 2)
	deletePlan := requireJSONObject(t, payload["deletePlan"])
	require.Equal(t, "planned", deletePlan["status"])
	require.Len(t, deletePlan["resolvedRootKeys"], 1)
	require.Len(t, deletePlan["affectedKeys"], 1)
}

// TestOpsPurgeProcessInstancesWithIncidentsDryRunPlanRendering verifies compact and verbose plan output once planning is available.
func TestOpsPurgeProcessInstancesWithIncidentsDryRunPlanRendering(t *testing.T) {
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)

	result := ops.IncidentPurgeResult{
		Request: ops.IncidentPurgeRequest{DryRun: true},
		Discovery: ops.IncidentDiscoveryResult{
			Status:                        ops.WorkflowStepStatusPlanned,
			IncidentCount:                 2,
			CandidateProcessInstanceCount: 2,
			IncidentKeys:                  typex.Keys{"inc-1", "inc-2"},
			CandidateProcessInstanceKeys:  typex.Keys{"child-1", "child-2"},
		},
		DeletePlan: ops.IncidentPurgeDeletePlan{
			Status:                       ops.WorkflowStepStatusPlanned,
			CandidateProcessInstanceKeys: typex.Keys{"child-1", "child-2"},
			ResolvedRootKeys:             typex.Keys{"root-1"},
			AffectedKeys:                 typex.Keys{"root-1", "child-1", "child-2"},
			DuplicateResolvedRootKeys:    typex.Keys{"root-1"},
		},
		Deletion: ops.IncidentPurgeDeletionResult{
			Status: ops.WorkflowStepStatusSkipped,
		},
		Outcome: ops.IncidentPurgeOutcomePlanned,
	}

	var compact bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&compact)
	require.NoError(t, renderOpsPurgeProcessInstancesWithIncidentsResult(cmd, result))
	require.Contains(t, compact.String(), "delete plan: planned (candidate process instances: 2, roots: 1, affected process instances: 3)")
	require.NotContains(t, compact.String(), "resolved root keys:")
	require.NotContains(t, compact.String(), "affected process-instance keys:")

	flagVerbose = true
	var verbose bytes.Buffer
	cmd = &cobra.Command{}
	cmd.SetOut(&verbose)
	require.NoError(t, renderOpsPurgeProcessInstancesWithIncidentsResult(cmd, result))
	require.Contains(t, verbose.String(), "resolved root keys: root-1")
	require.Contains(t, verbose.String(), "affected process-instance keys: root-1, child-1, child-2")
	require.Contains(t, verbose.String(), "duplicate resolved root keys: root-1")
}

// TestOpsPurgeProcessInstancesWithIncidentsVerboseListsAllKeyGroups verifies compact output hides keys unless verbose is enabled.
func TestOpsPurgeProcessInstancesWithIncidentsVerboseListsAllKeyGroups(t *testing.T) {
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)

	result := sampleIncidentPurgeDryRunPlanResult()
	result.DeletePlan.ResolvedRootKeys = typex.Keys{"2251799813711972"}
	result.DeletePlan.AffectedKeys = typex.Keys{"2251799813711972", "2251799813711973"}

	var compact bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&compact)
	require.NoError(t, renderOpsPurgeProcessInstancesWithIncidentsResult(cmd, result))
	require.NotContains(t, compact.String(), "incident keys:")
	require.NotContains(t, compact.String(), "candidate process-instance keys:")
	require.NotContains(t, compact.String(), "resolved root keys:")
	require.NotContains(t, compact.String(), "affected process-instance keys:")
	require.NotContains(t, compact.String(), "skipped incident keys:")

	flagVerbose = true
	var verbose bytes.Buffer
	cmd = &cobra.Command{}
	cmd.SetOut(&verbose)
	require.NoError(t, renderOpsPurgeProcessInstancesWithIncidentsResult(cmd, result))
	require.Contains(t, verbose.String(), "incident keys: 2251799813685253, 2251799813685254, 2251799813685255")
	require.Contains(t, verbose.String(), "candidate process-instance keys: 2251799813711972")
	require.Contains(t, verbose.String(), "duplicate candidate process-instance keys: 2251799813711972")
	require.Contains(t, verbose.String(), "resolved root keys: 2251799813711972")
	require.Contains(t, verbose.String(), "affected process-instance keys: 2251799813711972, 2251799813711973")
	require.Contains(t, verbose.String(), "skipped incident keys: 2251799813685255 (missing process-instance key)")
}

// TestOpsPurgeProcessInstancesWithIncidentsJSONOutputsStayMachineReadable verifies dry-run and automation JSON emit only envelopes.
func TestOpsPurgeProcessInstancesWithIncidentsJSONOutputsStayMachineReadable(t *testing.T) {
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsIncidentPurgeServer(t, &requests, &deleted, false)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	dryRunStdout, dryRunStderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsPurgeProcessInstancesWithIncidentsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
		"C8VOLT_TEST_INCIDENT_PURGE_ARGS": marshalOpsPurgeProcessInstancesWithIncidentsArgsForEnv(t, []string{
			"--json",
			"ops", "purge", "process-instances-with-incidents",
			"--dry-run",
		}),
	})
	require.NoError(t, err, dryRunStderr)
	require.Empty(t, dryRunStderr)
	require.NotContains(t, dryRunStdout, "dry run: purge process-instances with incidents")
	var dryRunEnvelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(dryRunStdout), &dryRunEnvelope), dryRunStdout)
	require.Equal(t, string(OutcomeSucceeded), dryRunEnvelope["outcome"])
	require.Equal(t, "ops purge process-instances-with-incidents", dryRunEnvelope["command"])
	dryRunPayload := requireJSONObject(t, dryRunEnvelope["payload"])
	require.Equal(t, "planned", dryRunPayload["outcome"])
	require.Equal(t, true, requireJSONObject(t, dryRunPayload["request"])["dryRun"])
	require.Equal(t, "planned", requireJSONObject(t, dryRunPayload["deletePlan"])["status"])
	require.Equal(t, "skipped", requireJSONObject(t, dryRunPayload["deletion"])["status"])
	require.Empty(t, deleted.Snapshot())

	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	autoStdout, autoStderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsPurgeProcessInstancesWithIncidentsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
		"C8VOLT_TEST_INCIDENT_PURGE_ARGS": marshalOpsPurgeProcessInstancesWithIncidentsArgsForEnv(t, []string{
			"--automation",
			"--json",
			"ops", "purge", "process-instances-with-incidents",
			"--no-wait",
			"--force",
		}),
	})
	require.NoError(t, err, autoStderr)
	require.NotContains(t, autoStderr, "purge process-instances with incidents")
	require.NotContains(t, autoStdout, "purge process-instances with incidents")
	var autoEnvelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(autoStdout), &autoEnvelope), autoStdout)
	require.Equal(t, string(OutcomeSucceeded), autoEnvelope["outcome"])
	autoPayload := requireJSONObject(t, autoEnvelope["payload"])
	require.Equal(t, "deleted", autoPayload["outcome"])
	require.Equal(t, "submitted", requireJSONObject(t, autoPayload["deletion"])["status"])
	require.Equal(t, true, requireJSONObject(t, autoPayload["request"])["automation"])
	require.Equal(t, []string{"/v2/process-instances/" + opsIncidentPurgeRootKey + "/deletion"}, deleted.Snapshot())
}

// TestOpsPurgeProcessInstancesWithIncidentsWritesMarkdownReport verifies the dry-run report includes complete audit sections.
func TestOpsPurgeProcessInstancesWithIncidentsWritesMarkdownReport(t *testing.T) {
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsIncidentPurgeServer(t, &requests, &deleted, false)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "incident-purge.md")

	outputBytes, err := testx.RunCmdSubprocess(t, "TestOpsPurgeProcessInstancesWithIncidentsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_INCIDENT_PURGE_ARGS": marshalOpsPurgeProcessInstancesWithIncidentsArgsForEnv(t, []string{
			"ops", "purge", "process-instances-with-incidents",
			"--dry-run",
			"--report-file", reportPath,
		}),
	})
	require.NoError(t, err, string(outputBytes))
	output := string(outputBytes)

	require.Contains(t, output, "outcome: planned; no changes applied")
	require.Contains(t, output, "report: written "+reportPath)
	require.Empty(t, deleted.Snapshot())
	report := readReportFile(t, reportPath)
	require.Contains(t, report, "# Incident Process Instance Purge Audit Report")
	require.Contains(t, report, "- Command: ops purge process-instances-with-incidents")
	require.Contains(t, report, "- Dry Run: true")
	require.Contains(t, report, "- Camunda Version: 8.9")
	require.Contains(t, report, "- Profile: default")
	require.Contains(t, report, "- Outcome: planned")
	require.Contains(t, report, "## Discovery")
	require.Contains(t, report, "- Incident Keys:")
	require.Contains(t, report, "  - 2251799813685299")
	require.Contains(t, report, "## Delete Plan")
	require.Contains(t, report, "  - "+opsIncidentPurgeRootKey)
}

// TestOpsPurgeProcessInstancesWithIncidentsWritesJSONReport verifies confirmed runs overwrite only after deletion submission.
func TestOpsPurgeProcessInstancesWithIncidentsWritesJSONReport(t *testing.T) {
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsIncidentPurgeServer(t, &requests, &deleted, false)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "incident-purge.json")
	require.NoError(t, os.WriteFile(reportPath, []byte("old report"), 0o600))

	outputBytes, err := testx.RunCmdSubprocess(t, "TestOpsPurgeProcessInstancesWithIncidentsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_INCIDENT_PURGE_ARGS": marshalOpsPurgeProcessInstancesWithIncidentsArgsForEnv(t, []string{
			"ops", "purge", "process-instances-with-incidents",
			"--auto-confirm",
			"--no-wait",
			"--report-file", reportPath,
			"--report-format", "json",
		}),
	})
	require.NoError(t, err, string(outputBytes))
	output := string(outputBytes)

	require.Contains(t, output, "outcome: deleted")
	require.Contains(t, output, "elapsed:")
	require.Contains(t, output, "report: written "+reportPath)
	require.NotContains(t, readReportFile(t, reportPath), "old report")
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "ops.process-instances-with-incidents.v1", report["schemaVersion"])
	require.Equal(t, "ops purge process-instances-with-incidents", report["commandName"])
	require.Equal(t, "deleted", report["outcome"])
	require.Equal(t, true, report["noWait"])
	require.Equal(t, "8.9", report["camundaVersion"])
	discovery := requireJSONObject(t, report["discovery"])
	require.Equal(t, float64(1), discovery["incidentCount"])
	require.Equal(t, float64(1), discovery["candidateProcessInstanceCount"])
	deletePlan := requireJSONObject(t, report["deletePlan"])
	require.Len(t, deletePlan["resolvedRootKeys"], 1)
	deletion := requireJSONObject(t, report["deletion"])
	require.Equal(t, "submitted", deletion["status"])
	require.Equal(t, true, deletion["submitted"])
	require.Equal(t, []string{"/v2/process-instances/" + opsIncidentPurgeRootKey + "/deletion"}, deleted.Snapshot())
}

// TestOpsPurgeProcessInstancesWithIncidentsExistingReportPreservation verifies non-submitted paths never clobber reports.
func TestOpsPurgeProcessInstancesWithIncidentsExistingReportPreservation(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		withActiveChild bool
		want            string
		wantRequests    bool
	}{
		{
			name: "dry run",
			args: []string{"ops", "purge", "process-instances-with-incidents", "--dry-run"},
			want: "report file already exists:",
		},
		{
			name: "unconfirmed",
			args: []string{"ops", "purge", "process-instances-with-incidents"},
			want: "report file already exists:",
		},
		{
			name:            "locally blocked",
			args:            []string{"ops", "purge", "process-instances-with-incidents", "--auto-confirm"},
			withActiveChild: true,
			want:            "write audit report: report file already exists:",
			wantRequests:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			var deleted testx.SafeSlice[string]
			srv := newOpsIncidentPurgeServer(t, &requests, &deleted, tt.withActiveChild)
			t.Cleanup(srv.Close)
			reportPath := filepath.Join(t.TempDir(), "incident-purge.md")
			const existingReport = "existing report"
			require.NoError(t, os.WriteFile(reportPath, []byte(existingReport), 0o600))
			args := append([]string{}, tt.args...)
			args = append(args, "--report-file", reportPath)

			output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeProcessInstancesWithIncidentsCommandHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
				"C8VOLT_TEST_INCIDENT_PURGE_ARGS": marshalOpsPurgeProcessInstancesWithIncidentsArgsForEnv(t, args),
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

// TestOpsPurgeProcessInstancesWithIncidentsConfirmedDeletionUsesFrozenPlanRoots verifies the prompt path executes the confirmed frozen candidate set.
func TestOpsPurgeProcessInstancesWithIncidentsConfirmedDeletionUsesFrozenPlanRoots(t *testing.T) {
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)
	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsIncidentPurgeServer(t, &requests, &deleted, false)
	t.Cleanup(srv.Close)
	var prompt string
	promptPath := filepath.Join(t.TempDir(), "prompt.txt")
	outputBytes, err := testx.RunCmdSubprocess(t, "TestOpsPurgeProcessInstancesWithIncidentsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":                     writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_INCIDENT_PURGE_PROMPT_FILE": promptPath,
		"C8VOLT_TEST_INCIDENT_PURGE_ARGS": marshalOpsPurgeProcessInstancesWithIncidentsArgsForEnv(t, []string{
			"ops", "purge", "process-instances-with-incidents",
			"--no-wait",
		}),
	})
	require.NoError(t, err, string(outputBytes))
	prompt = readReportFile(t, promptPath)
	output := string(outputBytes)

	require.Contains(t, prompt, "Incident purge matched 1 candidate incident(s)")
	require.Contains(t, output, "deletion: submitted (requests: 1)")
	require.Contains(t, output, "deletion confirmation: skipped (--no-wait)")
	require.Contains(t, output, "outcome: deleted")
	require.Equal(t, []string{"/v2/process-instances/" + opsIncidentPurgeRootKey + "/deletion"}, deleted.Snapshot())
	require.Equal(t, 1, countOpsIncidentPurgeRequests(requests.Snapshot(), "POST /v2/incidents/search "))
}

// TestOpsPurgeProcessInstancesWithIncidentsAutomationJSONExecutesWithoutAutoConfirm verifies automation mode confirms the supported purge path.
func TestOpsPurgeProcessInstancesWithIncidentsAutomationJSONExecutesWithoutAutoConfirm(t *testing.T) {
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)

	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsIncidentPurgeServer(t, &requests, &deleted, false)
	t.Cleanup(srv.Close)

	stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsPurgeProcessInstancesWithIncidentsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_INCIDENT_PURGE_ARGS": marshalOpsPurgeProcessInstancesWithIncidentsArgsForEnv(t, []string{
			"--automation",
			"--json",
			"ops", "purge", "process-instances-with-incidents",
			"--workers", "2",
			"--fail-fast",
			"--no-worker-limit",
			"--no-wait",
			"--force",
		}),
	})
	require.NoError(t, err, stderr)

	require.NotContains(t, stderr, "purge process-instances with incidents")
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), stdout)
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "ops purge process-instances-with-incidents", envelope["command"])
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
	require.Equal(t, []string{"/v2/process-instances/" + opsIncidentPurgeRootKey + "/deletion"}, deleted.Snapshot())
}

// TestOpsPurgeProcessInstancesWithIncidentsBlocksNonFinalScopeBeforeMutation verifies post-planning blockers keep local-precondition exit behavior.
func TestOpsPurgeProcessInstancesWithIncidentsBlocksNonFinalScopeBeforeMutation(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsIncidentPurgeServer(t, &requests, &deleted, true)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeProcessInstancesWithIncidentsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_INCIDENT_PURGE_ARGS": marshalOpsPurgeProcessInstancesWithIncidentsArgsForEnv(t, []string{
			"ops", "purge", "process-instances-with-incidents",
		}),
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "local precondition failed")
	require.Contains(t, string(output), "refusing to delete incident purge process-instance scope")
	require.Contains(t, string(output), "affected process instance(s) are not in a final state")
	require.Empty(t, deleted.Snapshot())
}

// TestOpsPurgeProcessInstancesWithIncidentsInvalidFlagsHelper runs command validation in a subprocess for exit-code assertions.
func TestOpsPurgeProcessInstancesWithIncidentsInvalidFlagsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_INCIDENT_PURGE_ARGS")), &args); err != nil {
		t.Fatalf("invalid helper args: %v", err)
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	if promptPath := os.Getenv("C8VOLT_TEST_INCIDENT_PURGE_PROMPT_FILE"); promptPath != "" {
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

// TestOpsPurgeProcessInstancesWithIncidentsCommandHelper runs incident purge command subprocess cases.
func TestOpsPurgeProcessInstancesWithIncidentsCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_INCIDENT_PURGE_ARGS")), &args); err != nil {
		t.Fatalf("invalid helper args: %v", err)
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	if promptPath := os.Getenv("C8VOLT_TEST_INCIDENT_PURGE_PROMPT_FILE"); promptPath != "" {
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

// executeOpsPurgeProcessInstancesWithIncidentsExpectError runs the command without exiting on Cobra parse errors.
func executeOpsPurgeProcessInstancesWithIncidentsExpectError(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := Root()
	buf := &bytes.Buffer{}
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)
	root.SetOut(buf)
	root.SetErr(buf)
	fullArgs := append([]string{"ops", "purge", "process-instances-with-incidents"}, args...)
	root.SetArgs(fullArgs)
	_, err := root.ExecuteC()
	if buf.Len() == 0 && err != nil {
		return err.Error(), err
	}
	return buf.String(), err
}

// marshalOpsPurgeProcessInstancesWithIncidentsArgsForEnv preserves argument boundaries for subprocess helpers.
func marshalOpsPurgeProcessInstancesWithIncidentsArgsForEnv(t *testing.T, args []string) string {
	t.Helper()

	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}

// resetOpsPurgeProcessInstancesWithIncidentsFlagState restores incident-purge globals between command tests.
func resetOpsPurgeProcessInstancesWithIncidentsFlagState() {
	flagOpsPurgeIncidentKeys = nil
	flagOpsPurgeIncidentState = "active"
	flagOpsPurgeIncidentErrorType = ""
	flagOpsPurgeIncidentErrorMessage = ""
	flagOpsPurgeIncidentBpmnProcessID = ""
	flagOpsPurgeIncidentPDKey = ""
	flagOpsPurgeIncidentPIKey = ""
	flagOpsPurgeIncidentRootKey = ""
	flagOpsPurgeIncidentFlowNodeID = ""
	flagOpsPurgeIncidentFNIKey = ""
	flagOpsPurgeIncidentCreationTimeAfter = ""
	flagOpsPurgeIncidentCreationTimeBefore = ""
	flagOpsPurgeIncidentBatchSize = consts.MaxPISearchSize
	flagOpsPurgeIncidentLimit = 0
	flagOpsPurgeIncidentReportFile = ""
	flagOpsPurgeIncidentReportFormat = ""
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

func sampleIncidentPurgeDryRunPlanResult() ops.IncidentPurgeResult {
	return ops.IncidentPurgeResult{
		Request: ops.IncidentPurgeRequest{
			DryRun: true,
		},
		Discovery: ops.IncidentDiscoveryResult{
			Status:                                ops.WorkflowStepStatusPlanned,
			Filters:                               incidentPurgeActiveFilter(),
			IncidentKeys:                          typex.Keys{"2251799813685253", "2251799813685254", "2251799813685255"},
			CandidateProcessInstanceKeys:          typex.Keys{"2251799813711972"},
			DuplicateCandidateProcessInstanceKeys: typex.Keys{"2251799813711972"},
			SkippedIncidents: []ops.IncidentPurgeSkippedIncident{
				{Incident: incident.ProcessInstanceIncidentDetail{IncidentKey: "2251799813685255"}, Reason: "missing process-instance key"},
			},
			IncidentCount:                 3,
			CandidateProcessInstanceCount: 1,
			Notices: []ops.IncidentPurgeWorkflowNotice{
				{Code: "duplicate_candidate_process_instances", Severity: "info", Message: "duplicate candidate process instances detected"},
				{Code: "skipped_incidents", Severity: "warning", Message: "some candidate incidents could not produce process-instance keys"},
			},
		},
		DeletePlan: ops.IncidentPurgeDeletePlan{
			Status:                       ops.WorkflowStepStatusPlanned,
			CandidateProcessInstanceKeys: typex.Keys{"2251799813711972"},
			ResolvedRootKeys:             typex.Keys{"2251799813711972"},
			AffectedKeys:                 typex.Keys{"2251799813711972"},
		},
		Deletion: ops.IncidentPurgeDeletionResult{
			Status: ops.WorkflowStepStatusSkipped,
		},
		Outcome: ops.IncidentPurgeOutcomePlanned,
	}
}

func incidentPurgeActiveFilter() incident.Filter {
	return incident.Filter{State: "active"}
}

const (
	opsIncidentPurgeRootKey  = "2251799813685301"
	opsIncidentPurgeChildKey = "2251799813685302"
)

func newOpsIncidentPurgeServer(t *testing.T, requests *testx.SafeSlice[string], deleted *testx.SafeSlice[string], withActiveChild bool) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/incidents/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[` + opsIncidentPurgeIncidentJSON("2251799813685299", opsIncidentPurgeRootKey) + `],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/"+opsIncidentPurgeRootKey:
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(opsIncidentPurgeProcessInstanceJSON(opsIncidentPurgeRootKey, "", "COMPLETED")))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/"+opsIncidentPurgeChildKey:
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(opsIncidentPurgeProcessInstanceJSON(opsIncidentPurgeChildKey, opsIncidentPurgeRootKey, "ACTIVE")))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			payload := string(body)
			requests.Append(r.Method + " " + r.URL.Path + " " + payload)
			w.Header().Set("Content-Type", "application/json")
			if withActiveChild && strings.Contains(payload, opsIncidentPurgeRootKey) && strings.Contains(payload, "parentProcessInstanceKey") {
				_, _ = w.Write([]byte(`{"items":[` + opsIncidentPurgeProcessInstanceJSON(opsIncidentPurgeChildKey, opsIncidentPurgeRootKey, "ACTIVE") + `],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/"+opsIncidentPurgeRootKey+"/deletion":
			if deleted != nil {
				deleted.Append(r.URL.Path)
			}
			requests.Append(r.Method + " " + r.URL.Path)
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func opsIncidentPurgeIncidentJSON(key string, piKey string) string {
	return `{"incidentKey":"` + key + `","processInstanceKey":"` + piKey + `","tenantId":"tenant","state":"ACTIVE","errorType":"JOB_NO_RETRIES","errorMessage":"no retries left"}`
}

func opsIncidentPurgeProcessInstanceJSON(key string, parentKey string, state string) string {
	parent := ""
	if parentKey != "" {
		parent = `,"parentProcessInstanceKey":"` + parentKey + `","rootProcessInstanceKey":"` + parentKey + `"`
	}
	return `{"processInstanceKey":"` + key + `","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"startDate":"2026-05-16T12:00:00Z","state":"` + state + `","tenantId":"tenant"` + parent + `}`
}

func countOpsIncidentPurgeRequests(items []string, prefix string) int {
	count := 0
	for _, item := range items {
		if strings.HasPrefix(item, prefix) {
			count++
		}
	}
	return count
}
