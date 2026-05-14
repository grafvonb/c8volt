// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const opsRetentionPolicySeedKey = "2251799813685249"

func TestOpsExecuteRetentionPolicyHelpDocumentsCommand(t *testing.T) {
	output := executeRootForProcessInstanceTest(t, "ops", "execute", "--help")

	assertHelpOutputContainsAll(t, output,
		"Discover predefined operational playbooks",
		"retention-policy",
	)

	commandOutput := executeRootForProcessInstanceTest(t, "ops", "execute", "retention-policy", "--help")

	assertHelpOutputContainsAll(t, commandOutput,
		"Execute process-instance retention cleanup",
		"--retention-days int",
		"./c8volt ops execute retention-policy --retention-days 90 --dry-run",
	)
}

func TestOpsExecuteRetentionPolicyInvalidRetentionDays(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")
	tests := []struct {
		name   string
		helper string
		args   []string
		want   string
	}{
		{
			name:   "missing",
			helper: "TestOpsExecuteRetentionPolicyInvalidRetentionDaysHelper",
			args:   []string{"ops", "execute", "retention-policy"},
			want:   "ops execute retention-policy requires --retention-days",
		},
		{
			name:   "negative",
			helper: "TestOpsExecuteRetentionPolicyInvalidRetentionDaysHelper",
			args:   []string{"ops", "execute", "retention-policy", "--retention-days", "-1"},
			want:   "invalid value for --retention-days: -1, expected non-negative integer",
		},
		{
			name:   "non integer",
			helper: "TestOpsExecuteRetentionPolicyInvalidRetentionDaysHelper",
			args:   []string{"ops", "execute", "retention-policy", "--retention-days", "not-a-number"},
			want:   "invalid argument \"not-a-number\" for \"--retention-days\" flag",
		},
		{
			name:   "explicit key",
			helper: "TestOpsExecuteRetentionPolicyInvalidRetentionDaysHelper",
			args:   []string{"ops", "execute", "retention-policy", "--retention-days", "90", "--key", "2251799813685249"},
			want:   "retention policy discovers eligible process instances and does not accept explicit process-instance keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, tt.helper, map[string]string{
				"C8VOLT_TEST_CONFIG":         cfgPath,
				"C8VOLT_TEST_RETENTION_ARGS": marshalRetentionArgsForEnv(t, tt.args),
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

func TestOpsExecuteRetentionPolicyInvalidRetentionDaysHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_RETENTION_ARGS")), &args); err != nil {
		t.Fatalf("invalid helper args: %v", err)
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs(append([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG")}, args...))
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		handleBootstrapError(root, err)
	}
}

func TestOpsExecuteRetentionPolicyDryRunAppliesCompatibleFilters(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() { relativeDayNow = prevNow })

	var requests testx.SafeSlice[string]
	srv := newOpsRetentionPolicySearchServer(t, &requests)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "retention-policy",
		"--retention-days", "90",
		"--dry-run",
		"--bpmn-process-id", "order-process",
		"--state", "completed",
		"--batch-size", "25",
		"--limit", "1",
		"--no-incidents-only",
	)

	require.Contains(t, output, "retention discovery: planned")
	require.Contains(t, output, "selection filters:")
	snapshot := requests.Snapshot()
	require.Len(t, snapshot, 1)
	request := decodeCapturedPISearchRequest(t, strings.TrimPrefix(snapshot[0], "POST /v2/process-instances/search "))
	filter := request["filter"].(map[string]any)
	page := request["page"].(map[string]any)
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Equal(t, "COMPLETED", filter["state"])
	require.Equal(t, false, filter["hasIncident"])
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-02-13T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")
	require.Equal(t, float64(25), page["limit"])
}

func TestOpsExecuteRetentionPolicyDryRunNoTargetsReportsNoOp(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsRetentionPolicySearchServer(t, &requests)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "retention-policy",
		"--retention-days", "90",
		"--dry-run",
	)

	require.Contains(t, output, "retention discovery: planned")
	require.Contains(t, output, "retention seeds: 0")
	require.Contains(t, output, "no retention cleanup targets found")
	require.Contains(t, output, "delete plan: skipped")
	require.Contains(t, output, "deletion: skipped; no deletion request submitted")
	require.Contains(t, output, "outcome: planned; no changes applied")
	snapshot := requests.Snapshot()
	require.Len(t, snapshot, 1)
	require.True(t, strings.HasPrefix(snapshot[0], "POST /v2/process-instances/search "))
}

func TestOpsExecuteRetentionPolicyDryRunJSONOutputIsStructured(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() { relativeDayNow = prevNow })

	var requests testx.SafeSlice[string]
	srv := newOpsRetentionPolicySearchServer(t, &requests)
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"--json",
		"ops", "execute", "retention-policy",
		"--retention-days", "90",
		"--dry-run",
	)

	require.Empty(t, strings.TrimSpace(stderr))
	require.NotContains(t, stdout, "retention discovery:")
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "ops execute retention-policy", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "planned", payload["outcome"])

	request := requireJSONObject(t, payload["request"])
	require.Equal(t, true, request["dryRun"])
	require.Equal(t, float64(90), request["retentionDays"])
	require.Equal(t, "2026-02-13", request["derivedEndDateBoundary"])

	discovery := requireJSONObject(t, payload["discovery"])
	require.Equal(t, "planned", discovery["status"])
	require.Equal(t, float64(0), discovery["count"])
	require.Equal(t, float64(90), discovery["retentionDays"])
	require.Equal(t, "2026-02-13", discovery["derivedEndDateBoundary"])

	deletePlan := requireJSONObject(t, payload["deletePlan"])
	require.Equal(t, "skipped", deletePlan["status"])
	deletion := requireJSONObject(t, payload["deletion"])
	require.Equal(t, "skipped", deletion["status"])
	require.NotContains(t, deletion, "submitted")
	report := requireJSONObject(t, payload["report"])
	require.Equal(t, "planned", report["outcome"])
}

func TestOpsExecuteRetentionPolicyDryRunDoesNotPromptOrMutate(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsRetentionPolicyServerWithSeed(t, &requests, &deleted)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "retention-policy",
		"--retention-days", "90",
		"--dry-run",
	)

	require.Contains(t, output, "delete plan: planned")
	require.Contains(t, output, "deletion: skipped; no deletion request submitted")
	require.Contains(t, output, "outcome: planned; no changes applied")
	require.Empty(t, deleted.Snapshot())
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/deletion")
}

func TestOpsExecuteRetentionPolicyConfirmedDeletionUsesFrozenPlanRoots(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsRetentionPolicyChangingSeedServer(t, &requests, &deleted)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "retention-policy",
		"--retention-days", "90",
		"--no-wait",
	)

	require.Contains(t, output, "retention discovery: planned")
	require.Contains(t, output, "delete plan: planned")
	require.Contains(t, output, "deletion: submitted (requests: 1)")
	require.Contains(t, output, "deletion confirmation: skipped (--no-wait)")
	require.Contains(t, output, "outcome: deleted")
	require.Equal(t, []string{"/v1/process-instances/" + opsRetentionPolicySeedKey}, deleted.Snapshot())
	require.NotContains(t, strings.Join(deleted.Snapshot(), "\n"), opsRetentionPolicyChangedSeedKey)
}

func TestOpsExecuteRetentionPolicyAutomationJSONExecutesWithoutAutoConfirm(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsRetentionPolicyServerWithSeed(t, &requests, &deleted)
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"--automation",
		"--json",
		"ops", "execute", "retention-policy",
		"--retention-days", "90",
		"--workers", "2",
		"--fail-fast",
		"--no-worker-limit",
		"--no-wait",
		"--no-state-check",
		"--force",
	)

	require.Empty(t, strings.TrimSpace(stderr))
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "ops execute retention-policy", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "deleted", payload["outcome"])
	request := requireJSONObject(t, payload["request"])
	require.Equal(t, true, request["automation"])
	require.NotContains(t, request, "autoConfirm")
	require.Equal(t, float64(2), request["workers"])
	require.Equal(t, true, request["failFast"])
	require.Equal(t, true, request["noWorkerLimit"])
	require.Equal(t, true, request["noWait"])
	require.Equal(t, true, request["noStateCheck"])
	require.Equal(t, true, request["force"])
	deletion := requireJSONObject(t, payload["deletion"])
	require.Equal(t, "submitted", deletion["status"])
	require.Equal(t, true, deletion["submitted"])
	require.Equal(t, true, deletion["noWait"])
	require.Equal(t, []string{"/v2/process-instances/" + opsRetentionPolicySeedKey + "/deletion"}, deleted.Snapshot())
}

func TestOpsExecuteRetentionPolicyWritesMarkdownReport(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsRetentionPolicyServerWithSeed(t, &requests, nil)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "retention-report.md")

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "retention-policy",
		"--retention-days", "90",
		"--dry-run",
		"--report-file", reportPath,
	)

	require.Contains(t, output, "outcome: planned; no changes applied")
	require.Contains(t, output, "report: written "+reportPath)
	report := readReportFile(t, reportPath)
	require.Contains(t, report, "# Retention Policy Audit Report")
	require.Contains(t, report, "- Command: ops execute retention-policy")
	require.Contains(t, report, "- Dry Run: true")
	require.Contains(t, report, "- Retention Days: 90")
	require.Contains(t, report, "- Outcome: planned")
	require.Contains(t, report, "- Camunda Version: 8.8")
	require.Contains(t, report, "- Profile: default")
	require.Contains(t, report, "- Tenant: <default>")
	require.Contains(t, report, "  - "+opsRetentionPolicySeedKey)
}

func TestOpsExecuteRetentionPolicyWritesJSONReport(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsRetentionPolicyServerWithSeed(t, &requests, &deleted)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "retention-report.json")
	require.NoError(t, os.WriteFile(reportPath, []byte("old report"), 0o600))

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"ops", "execute", "retention-policy",
		"--retention-days", "90",
		"--auto-confirm",
		"--no-wait",
		"--report-file", reportPath,
		"--report-format", "json",
	)

	require.Contains(t, output, "outcome: deleted")
	require.Contains(t, output, "report: written "+reportPath)
	require.NotContains(t, readReportFile(t, reportPath), "old report")
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "ops.retention-policy.v1", report["schemaVersion"])
	require.Equal(t, "ops execute retention-policy", report["commandName"])
	require.Equal(t, "deleted", report["outcome"])
	require.Equal(t, float64(90), report["retentionDays"])
	require.Equal(t, "8.9", report["camundaVersion"])
	require.Equal(t, "<default>", report["tenantId"])
	discovery := requireJSONObject(t, report["discovery"])
	require.Equal(t, float64(1), discovery["count"])
	keys := discovery["seedKeys"].([]any)
	require.Equal(t, opsRetentionPolicySeedKey, keys[0])
	deletion := requireJSONObject(t, report["deletion"])
	require.Equal(t, "submitted", deletion["status"])
	require.Equal(t, true, deletion["noWait"])
	require.Equal(t, []string{"/v1/process-instances/" + opsRetentionPolicySeedKey}, deleted.Snapshot())
}

func TestOpsExecuteRetentionPolicyExistingReportFailsBeforePreflight(t *testing.T) {
	tests := []struct {
		name  string
		state string
		args  []string
	}{
		{
			name:  "dry-run",
			state: "COMPLETED",
			args:  []string{"ops", "execute", "retention-policy", "--retention-days", "90", "--dry-run"},
		},
		{
			name:  "unconfirmed",
			state: "COMPLETED",
			args:  []string{"ops", "execute", "retention-policy", "--retention-days", "90"},
		},
		{
			name:  "locally blocked",
			state: "ACTIVE",
			args:  []string{"ops", "execute", "retention-policy", "--retention-days", "90"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			var deleted testx.SafeSlice[string]
			srv := newOpsRetentionPolicyServerWithSeedState(t, &requests, &deleted, tt.state)
			t.Cleanup(srv.Close)
			reportPath := filepath.Join(t.TempDir(), "retention-report.md")
			const existingReport = "existing report"
			require.NoError(t, os.WriteFile(reportPath, []byte(existingReport), 0o600))

			args := append([]string{}, tt.args...)
			args = append(args, "--report-file", reportPath)
			output, err := testx.RunCmdSubprocess(t, "TestOpsExecuteRetentionPolicyReportHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":         writeTestConfigForVersion(t, srv.URL, "8.8"),
				"C8VOLT_TEST_RETENTION_ARGS": marshalRetentionArgsForEnv(t, args),
			})
			require.Error(t, err)

			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.Error, exitErr.ExitCode())
			require.Contains(t, string(output), "report file already exists: "+reportPath)
			require.NotContains(t, string(output), "write audit report")
			require.Equal(t, existingReport, readReportFile(t, reportPath))
			require.Empty(t, requests.Snapshot())
			require.Empty(t, deleted.Snapshot())
		})
	}
}

func TestOpsExecuteRetentionPolicyWritesReportAfterPostDiscoveryFailure(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsRetentionPolicyServerWithSeedState(t, &requests, &deleted, "ACTIVE")
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "retention-failed.json")

	output, err := testx.RunCmdSubprocess(t, "TestOpsExecuteRetentionPolicyReportHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_RETENTION_ARGS": marshalRetentionArgsForEnv(t, []string{
			"ops", "execute", "retention-policy",
			"--retention-days", "90",
			"--auto-confirm",
			"--report-file", reportPath,
			"--report-format", "json",
		}),
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "refusing to delete retention process-instance scope")
	require.Empty(t, deleted.Snapshot())
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "failed", report["outcome"])
	discovery := requireJSONObject(t, report["discovery"])
	require.Equal(t, float64(1), discovery["count"])
	deletion := requireJSONObject(t, report["deletion"])
	require.Equal(t, "blocked", deletion["status"])
	require.NotEmpty(t, report["errors"])
}

func TestOpsExecuteRetentionPolicyReportHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_RETENTION_ARGS")), &args); err != nil {
		t.Fatalf("invalid helper args: %v", err)
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs(append([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG")}, args...))
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestOpsExecuteRetentionPolicyBlocksNonFinalScopeBeforeMutation(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsRetentionPolicyServerWithSeedState(t, &requests, &deleted, "ACTIVE")
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsExecuteRetentionPolicyBlocksNonFinalScopeBeforeMutationHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.8"),
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "local precondition failed")
	require.Contains(t, string(output), "refusing to delete retention process-instance scope")
	require.Empty(t, deleted.Snapshot())
}

func TestOpsExecuteRetentionPolicyBlocksNonFinalScopeBeforeMutationHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"ops", "execute", "retention-policy",
		"--retention-days", "90",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		handleBootstrapError(root, err)
	}
}

func TestOpsExecuteRetentionPolicyDryRunDiscoveryOutput(t *testing.T) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	err := renderOpsExecuteRetentionPolicyResult(cmd, ops.RetentionPolicyResult{
		Request: ops.RetentionPolicyRequest{
			RetentionDays:          90,
			DerivedEndDateBoundary: "2026-02-13",
		},
		Discovery: ops.RetentionDiscoveryResult{
			Status:        ops.WorkflowStepStatusPlanned,
			RetentionDays: 90,
			Filters: process.ProcessInstanceFilter{
				BpmnProcessId: "invoice",
				EndDateBefore: "2026-02-13",
			},
			SeedKeys: []string{"seed-1", "seed-2"},
			Count:    2,
		},
		DeletePlan: ops.RetentionDeletePlan{
			Status: ops.WorkflowStepStatusSkipped,
		},
		Deletion: ops.RetentionDeletionResult{
			Status: ops.WorkflowStepStatusSkipped,
		},
		Outcome: ops.RetentionPolicyOutcomePlanned,
	})

	require.NoError(t, err)
	require.Contains(t, out.String(), "retention policy: planned")
	require.Contains(t, out.String(), "retention days: 90")
	require.Contains(t, out.String(), "retention boundary: endDate <= 2026-02-13")
	require.Contains(t, out.String(), "selection filters: {bpmnProcessId=\"invoice\", endDateBefore=\"2026-02-13\"}")
	require.Contains(t, out.String(), "retention discovery: planned")
	require.Contains(t, out.String(), "retention seeds: 2")
	require.Contains(t, out.String(), "delete plan: skipped")
	require.Contains(t, out.String(), "deletion: skipped; no deletion request submitted")
	require.Contains(t, out.String(), "outcome: planned; no changes applied")
}

func TestOpsExecuteRetentionPolicyDryRunPlanRendering(t *testing.T) {
	prevVerbose := flagVerbose
	flagVerbose = true
	t.Cleanup(func() { flagVerbose = prevVerbose })

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	err := renderOpsExecuteRetentionPolicyResult(cmd, ops.RetentionPolicyResult{
		Request: ops.RetentionPolicyRequest{
			RetentionDays:          90,
			DerivedEndDateBoundary: "2026-02-13",
		},
		Discovery: ops.RetentionDiscoveryResult{
			Status:  ops.WorkflowStepStatusPlanned,
			Count:   2,
			Filters: process.ProcessInstanceFilter{EndDateBefore: "2026-02-13"},
		},
		DeletePlan: ops.RetentionDeletePlan{
			Status:                ops.WorkflowStepStatusPlanned,
			SeedKeys:              []string{"child-1", "child-2"},
			ResolvedRootKeys:      []string{"root-1"},
			AffectedKeys:          []string{"root-1", "child-1", "child-2"},
			DuplicateKeys:         []string{"root-1"},
			NonFinalAffectedItems: []process.ProcessInstance{{Key: "child-2", State: process.StateActive}},
			MissingAncestors:      []process.MissingAncestor{{Key: "missing-parent", StartKey: "child-2"}},
			TraversalWarnings:     []string{"one or more parent process instances were not found"},
			RequiresConfirmation:  true,
		},
		Deletion: ops.RetentionDeletionResult{
			Status: ops.WorkflowStepStatusBlocked,
		},
		Outcome: ops.RetentionPolicyOutcomeFailed,
	})

	require.NoError(t, err)
	got := out.String()
	require.Contains(t, got, "delete plan: planned (seeds: 2, roots: 1, affected: 3)")
	require.Contains(t, got, "duplicate roots: 1")
	require.Contains(t, got, "process instances not in final state: 1 (use --force to cancel before delete)")
	require.Contains(t, got, "missing ancestors: 1")
	require.Contains(t, got, "traversal warning: one or more parent process instances were not found")
	require.Contains(t, got, "confirmation required: true")
	require.Contains(t, got, "retention seed keys: child-1, child-2")
	require.Contains(t, got, "resolved root keys: root-1")
	require.Contains(t, got, "affected process-instance keys: root-1, child-1, child-2")
	require.Contains(t, got, "deletion: blocked")
}

func marshalRetentionArgsForEnv(t *testing.T, args []string) string {
	t.Helper()

	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}

func newOpsRetentionPolicySearchServer(t *testing.T, requests *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

const opsRetentionPolicyChangedSeedKey = "2251799813685251"

func newOpsRetentionPolicyServerWithSeed(t *testing.T, requests *testx.SafeSlice[string], deleted *testx.SafeSlice[string]) *httptest.Server {
	return newOpsRetentionPolicyServerWithSeedState(t, requests, deleted, "COMPLETED")
}

func newOpsRetentionPolicyServerWithSeedState(t *testing.T, requests *testx.SafeSlice[string], deleted *testx.SafeSlice[string], state string) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(string(body), "parentProcessInstanceKey") {
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[` + opsRetentionPolicyProcessInstanceJSON(opsRetentionPolicySeedKey, state) + `],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/"+opsRetentionPolicySeedKey:
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(opsRetentionPolicyProcessInstanceJSON(opsRetentionPolicySeedKey, state)))
		case (r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/"+opsRetentionPolicySeedKey+"/deletion") ||
			(r.Method == http.MethodDelete && r.URL.Path == "/v1/process-instances/"+opsRetentionPolicySeedKey):
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

func newOpsRetentionPolicyChangingSeedServer(t *testing.T, requests *testx.SafeSlice[string], deleted *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()

	discoverySearches := 0
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(string(body), "parentProcessInstanceKey") {
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
				return
			}
			discoverySearches++
			key := opsRetentionPolicySeedKey
			if discoverySearches > 1 {
				key = opsRetentionPolicyChangedSeedKey
			}
			_, _ = w.Write([]byte(`{"items":[` + opsRetentionPolicyProcessInstanceJSON(key, "COMPLETED") + `],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && (r.URL.Path == "/v2/process-instances/"+opsRetentionPolicySeedKey || r.URL.Path == "/v2/process-instances/"+opsRetentionPolicyChangedSeedKey):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(opsRetentionPolicyProcessInstanceJSON(key, "COMPLETED")))
		case (r.Method == http.MethodPost && (r.URL.Path == "/v2/process-instances/"+opsRetentionPolicySeedKey+"/deletion" || r.URL.Path == "/v2/process-instances/"+opsRetentionPolicyChangedSeedKey+"/deletion")) ||
			(r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/"+opsRetentionPolicySeedKey || r.URL.Path == "/v1/process-instances/"+opsRetentionPolicyChangedSeedKey)):
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

func opsRetentionPolicyProcessInstanceJSON(key string, state string) string {
	return `{"processInstanceKey":"` + key + `","processDefinitionId":"retention-process","processDefinitionKey":"9001","processDefinitionName":"retention-process","processDefinitionVersion":3,"startDate":"2026-01-11T12:00:00Z","endDate":"2026-02-12T12:00:00Z","state":"` + state + `","tenantId":"tenant"}`
}
