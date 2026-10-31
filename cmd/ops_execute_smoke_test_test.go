// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
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
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

const (
	opsSmokeTestTerminalConfigEnv  = "C8VOLT_SMOKE_TERMINAL_CONFIG"
	opsSmokeTestTerminalStderrEnv  = "C8VOLT_SMOKE_TERMINAL_STDERR"
	opsSmokeTestTerminalCaptureEnv = "C8VOLT_SMOKE_TERMINAL_CAPTURE"
)

// TestOpsExecuteSmokeTestTenantContextUsesCreationSemantics verifies smoke
// tests report a default creation target and merge returned resource evidence.
func TestOpsExecuteSmokeTestTenantContextUsesCreationSemantics(t *testing.T) {
	cmd := &cobra.Command{}
	result := ops.SmokeTestResult{
		Deployment: ops.SmokeTestDeploymentResult{
			TenantEvidence: process.TenantEvidence{
				Targets: []process.TenantEvidenceTarget{{Key: "pd-smoke", TenantID: "tenant-b"}},
			},
		},
		Run: ops.SmokeTestRunResult{
			TenantEvidence: process.TenantEvidence{
				Targets: []process.TenantEvidenceTarget{{Key: "pi-smoke", TenantID: "tenant-b"}},
			},
		},
	}

	got := attachOpsExecuteSmokeTestResultTenantContext(cmd, &config.Config{}, result)

	require.NotNil(t, got.Report.TenantContext)
	require.Equal(t, tenant.ContextModeCreation, got.Report.TenantContext.Mode)
	require.Equal(t, tenant.ContextFilterNotApplicable, got.Report.TenantContext.Filter)
	require.Equal(t, config.DefaultTenant, got.Report.TenantContext.TargetTenantID)
	require.Equal(t, []string{"tenant-b"}, got.Report.TenantContext.ResolvedTenantIDs)
	require.Equal(t, config.DefaultTenant, got.Report.TenantID)
}

// TestSmokeTestCleanupSummaryPrefersDeploymentWideEvidence verifies new results
// report the complete definition collection while nil summaries retain legacy wording.
func TestSmokeTestCleanupSummaryPrefersDeploymentWideEvidence(t *testing.T) {
	collection := ops.SmokeTestCleanupResult{
		ProcessInstanceCleanup: ops.SmokeTestProcessInstanceCleanupResult{
			SubmittedKeys: []string{"pi-1"},
		},
		DefinitionSummary: &ops.SmokeTestDefinitionCleanupSummary{
			Status:    ops.WorkflowStepStatusConfirmed,
			Total:     3,
			Confirmed: 3,
		},
	}
	require.Equal(t, "removed 1 process instance and 3 process definitions", smokeTestCleanupSummary(collection, ops.WorkflowStepStatusConfirmed))

	legacy := ops.SmokeTestCleanupResult{
		ProcessInstanceCleanup: ops.SmokeTestProcessInstanceCleanupResult{
			SubmittedKeys: []string{"pi-1"},
		},
		ProcessDefinitionCleanup: ops.SmokeTestProcessDefinitionCleanupResult{
			Status: ops.WorkflowStepStatusConfirmed,
		},
	}
	require.Equal(t, "removed 1 process instance and fixture process definition", smokeTestCleanupSummary(legacy, ops.WorkflowStepStatusConfirmed))
}

// TestSmokeTestCleanupSummaryReportsMixedDefinitionStates verifies compact
// human output does not collapse retained or failed resources into success.
func TestSmokeTestCleanupSummaryReportsMixedDefinitionStates(t *testing.T) {
	cleanup := ops.SmokeTestCleanupResult{
		ProcessInstanceCleanup: ops.SmokeTestProcessInstanceCleanupResult{
			SubmittedKeys: []string{"pi-1"},
		},
		DefinitionSummary: &ops.SmokeTestDefinitionCleanupSummary{
			Status:    ops.WorkflowStepStatusFailed,
			Total:     3,
			Confirmed: 1,
			Retained:  1,
			Failed:    1,
		},
	}
	require.Equal(t, "failed (1 process instance; process definitions: 1 confirmed, 1 retained, 1 failed)", smokeTestCleanupSummary(cleanup, ops.WorkflowStepStatusFailed))
}

// TestOpsExecuteSmokeTestHelpDocumentsCommand verifies help describes deployment-wide cleanup and supported controls.
func TestOpsExecuteSmokeTestHelpDocumentsCommand(t *testing.T) {
	output := executeRootForProcessInstanceTest(t, "ops", "execute", "--help")

	assertHelpOutputContainsAll(t, output,
		"Run predefined operational playbooks",
		"retention-policy",
		"smoke-test",
	)

	commandOutput := executeRootForProcessInstanceTest(t, "ops", "execute", "smoke-test", "--help")

	assertHelpOutputContainsAll(t, commandOutput,
		"Verify a configured Camunda environment",
		"each deployed fixture definition that has no unrelated instances",
		"--count int",
		"-n, --count int",
		"--workers int",
		"--no-worker-limit",
		"--fail-fast",
		"--no-cleanup",
		"--dry-run",
		"--no-wait",
		"--report-file string",
		"--report-format string",
		"--all-tenants is not supported because",
		"--keys-only is not supported because the workflow returns a structured audit result",
		"./c8volt ops execute smoke-test --dry-run",
		"./c8volt ops execute smoke-test --report-file smoke-test.md",
		"./c8volt ops execute smoke-test --count 5 --report-file smoke-test.md",
	)
}

// TestOpsExecuteSmokeTestDefaultStageMilestonesAndPhaseIsolation verifies
// smoke-test stages get semantic aggregate milestones while nested lower-level
// phases are ignored by the command reporter.
func TestOpsExecuteSmokeTestDefaultStageMilestonesAndPhaseIsolation(t *testing.T) {
	resetSemanticProgressModeFlags(t)
	now := time.Date(2026, 9, 1, 7, 5, 0, 0, time.UTC)
	opsSmokeTestSemanticProgressNow = func() time.Time { return now }
	t.Cleanup(func() { opsSmokeTestSemanticProgressNow = time.Now })

	cmd, stderr := newSemanticProgressStderrCommand()
	request := ops.SmokeTestRequest{}
	progress := configureOpsExecuteSmokeTestProgress(cmd, &request)
	defer progress.Close()

	reportOpsSmokeTestCompletionEvent(request.Progress, "deploying smoke-test fixture", "smoke-definition-1", "process definition(s)", 2, ops.CompletionDispositionConfirmed, "")
	reportOpsSmokeTestCompletionEvent(request.Progress, "delete", "nested-root", "process-instance tree(s)", 1, ops.CompletionDispositionConfirmed, "")
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	reportOpsSmokeTestCompletionEvent(request.Progress, "deploying smoke-test fixture", "smoke-definition-2", "process definition(s)", 2, ops.CompletionDispositionConfirmed, "")
	reportOpsSmokeTestCompletionEvent(request.Progress, "cleaning up smoke-test process definitions", "retained-definition", "process definition(s)", 3, ops.CompletionDispositionSkipped, "cleanup disabled (--no-cleanup)")
	reportOpsSmokeTestCompletionEvent(request.Progress, "cleaning up smoke-test process instances", "smoke-root", "process-instance tree(s)", 1, ops.CompletionDispositionFailed, "cleanup rejected")
	progress.Close()

	output := stderr.String()
	require.Contains(t, output, "deploying smoke-test fixture, 2/2 process definition(s)")
	require.Contains(t, output, "smoke-root failed: cleanup rejected (cleaning up smoke-test process instances, 1/1 process-instance tree(s), 1 failed)")
	require.NotContains(t, output, "nested-root")
	require.NotContains(t, output, "delete process-instance")
}

// TestOpsExecuteSmokeTestSemanticProgressModeGate verifies smoke-test stage
// completion progress is silent for machine modes and only quiet failures are
// durable.
func TestOpsExecuteSmokeTestSemanticProgressModeGate(t *testing.T) {
	assertOpsCompletionProgressModeGate(t, opsCompletionProgressModeGateCase{
		Configure: func(cmd *cobra.Command) (func(ops.ProgressEvent), func()) {
			request := ops.SmokeTestRequest{}
			progress := configureOpsExecuteSmokeTestProgress(cmd, &request)
			return request.Progress, progress.Close
		},
		Event: func(disposition ops.CompletionDisposition, detail string) ops.ProgressEvent {
			return ops.ProgressEvent{
				Kind: ops.ProgressEventKindCompletion,
				Completion: &ops.CompletionProgress{
					Phase:         "starting process instances",
					CoreResource:  "process instance(s)",
					Total:         1,
					Identity:      "pi-1",
					Disposition:   disposition,
					FailureDetail: detail,
				},
			}
		},
		QuietWarning: "pi-1 failed: request rejected (starting process instances, 1/1 process instance(s), 1 failed)",
	})
}

// reportOpsSmokeTestCompletionEvent sends one high-level smoke-test completion
// fact through the configured smoke-test command progress callback.
func reportOpsSmokeTestCompletionEvent(progress func(ops.ProgressEvent), phase string, identity string, resource string, total int, disposition ops.CompletionDisposition, detail string) {
	progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:         phase,
			CoreResource:  resource,
			Total:         total,
			Identity:      identity,
			Disposition:   disposition,
			FailureDetail: detail,
		},
	})
}

func TestOpsExecuteSmokeTestInvalidLocalFlags(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "zero count",
			args: []string{"ops", "execute", "smoke-test", "--count", "0"},
			want: "invalid value for --count: 0, expected positive integer",
		},
		{
			name: "negative count",
			args: []string{"ops", "execute", "smoke-test", "-n", "-1"},
			want: "invalid value for --count: -1, expected positive integer",
		},
		{
			name: "non integer count",
			args: []string{"ops", "execute", "smoke-test", "--count", "not-a-number"},
			want: "invalid argument \"not-a-number\" for \"-n, --count\" flag",
		},
		{
			name: "invalid report format",
			args: []string{"ops", "execute", "smoke-test", "--report-file", "smoke-test.md", "--report-format", "yaml"},
			want: `unsupported ops workflow report format "yaml"`,
		},
		{
			name: "report format without report file",
			args: []string{"ops", "execute", "smoke-test", "--report-format", "json"},
			want: "--report-format requires --report-file",
		},
		{
			name: "unsupported keys-only output",
			args: []string{"--keys-only", "ops", "execute", "smoke-test", "--dry-run"},
			want: "--keys-only is not supported for ops execute smoke-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, "TestOpsExecuteSmokeTestInvalidLocalFlagsHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":     cfgPath,
				"C8VOLT_TEST_SMOKE_ARGS": marshalSmokeTestArgsForEnv(t, tt.args),
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

func TestOpsExecuteSmokeTestInvalidLocalFlagsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_SMOKE_ARGS")), &args); err != nil {
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

// TestOpsExecuteSmokeTestAllTenantsRejectsBeforePlanningOrRequest proves both
// dry-run and normal smoke-test execution reject all-tenants before reports,
// prompts, activity, planning, or remote preflight can run.
func TestOpsExecuteSmokeTestAllTenantsRejectsBeforePlanningOrRequest(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "dry-run",
			args: []string{"--all-tenants", "ops", "execute", "smoke-test", "--dry-run"},
		},
		{
			name: "normal",
			args: []string{"ops", "execute", "smoke-test", "--all-tenants"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			srv := newOpsExecuteSmokeTestDryRunServer(t, "8.8", &requests)
			t.Cleanup(srv.Close)
			reportPath := filepath.Join(t.TempDir(), "smoke-test.md")
			args := append([]string{"--config", writeTestConfigForVersion(t, srv.URL, "8.8")}, tt.args...)
			args = append(args, "--report-file", reportPath)

			output, err := testx.RunCmdSubprocess(t, "TestOpsExecuteSmokeTestAllTenantsRejectsBeforePlanningOrRequestHelper", map[string]string{
				"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, args),
			})
			assertAllTenantsConcreteDestinationSubprocessFailure(t, output, err, "ops execute smoke-test")
			require.NoFileExists(t, reportPath)
			require.Empty(t, requests.Snapshot())
			require.NotContains(t, string(output), "dry run: execute smoke test")
			require.NotContains(t, string(output), "validating smoke-test plan")
			require.NotContains(t, string(output), "running smoke-test workflow")
			require.NotContains(t, string(output), "creation target:")
		})
	}
}

// Helper-process entrypoint for all-tenants smoke-test rejection.
func TestOpsExecuteSmokeTestAllTenantsRejectsBeforePlanningOrRequestHelper(t *testing.T) {
	executeRootHelperFromArgsEnv(t, "C8VOLT_TEST_ROOT_ARGS")
}

// TestOpsExecuteSmokeTestDryRunHumanOutputPlansWithoutMutation preserves the existing command contract with advisory-aware fixtures.
func TestOpsExecuteSmokeTestDryRunHumanOutputPlansWithoutMutation(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestDryRunServer(t, "8.8", &requests)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "smoke-test",
		"--dry-run",
		"--count", "2",
		"--no-cleanup",
	)

	require.Contains(t, output, "dry run: execute smoke test")
	require.Contains(t, output, "fixture: embedded/processdefinitions/C88_MultipleSubProcessesParent.bpmn")
	require.Contains(t, output, "smoke-test preview: would deploy the fixture, start 2 process instances, and walk their process-instance families")
	require.Contains(t, output, "cleanup: skipped (--no-cleanup)")
	require.NotContains(t, output, "connectivity: confirmed")
	require.NotContains(t, output, "deployment: planned -")
	require.NotContains(t, output, "run: planned -")
	require.NotContains(t, output, "cleanup: planned")
	require.Contains(t, output, "outcome: planned; no changes applied")
	require.Equal(t, []string{"GET /v2/topology", "GET /v2/topology"}, requests.Snapshot())
}

// TestOpsExecuteSmokeTestDryRunJSONOutputReportsV810Fixture protects the dry-run envelope and verifies V810 reports the native fixture identity.
func TestOpsExecuteSmokeTestDryRunJSONOutputReportsV810Fixture(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestDryRunServer(t, "8.10", &requests)
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.10"),
		"--json",
		"ops", "execute", "smoke-test",
		"--dry-run",
	)

	require.Empty(t, strings.TrimSpace(stderr))
	require.NotContains(t, stdout, "dry run:")
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "ops execute smoke-test", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "planned", payload["outcome"])
	request := requireJSONObject(t, payload["request"])
	require.Equal(t, true, request["dryRun"])
	require.Equal(t, float64(1), request["count"])
	plan := requireJSONObject(t, payload["plan"])
	require.Equal(t, "planned", plan["status"])
	require.Equal(t, "8.10", plan["camundaVersion"])
	fixture := requireJSONObject(t, plan["fixture"])
	require.Equal(t, "embedded/processdefinitions/C810_MultipleSubProcessesParent.bpmn", fixture["file"])
	steps := plan["plannedSteps"].([]any)
	require.Len(t, steps, 7)
	connectivity := requireJSONObject(t, steps[0])
	require.Equal(t, "connectivity", connectivity["name"])
	require.Equal(t, "confirmed", connectivity["status"])
	require.Equal(t, []string{"GET /v2/topology", "GET /v2/topology"}, requests.Snapshot())
}

// TestOpsExecuteSmokeTestDryRunWritesMarkdownReport preserves the existing command contract with advisory-aware fixtures.
func TestOpsExecuteSmokeTestDryRunWritesMarkdownReport(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestDryRunServer(t, "8.8", &requests)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "smoke-test.md")

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "smoke-test",
		"--dry-run",
		"--report-file", reportPath,
	)

	require.Contains(t, output, "outcome: planned; no changes applied")
	require.Contains(t, output, "report: written "+reportPath)
	require.Less(t, strings.Index(output, "report: written "+reportPath), strings.Index(output, "outcome: planned; no changes applied"))
	report := readReportFile(t, reportPath)
	require.Contains(t, report, "# Execute Smoke Test Audit Report")
	require.Contains(t, report, "- Command: ops execute smoke-test")
	require.Contains(t, report, "- Dry Run: true")
	require.Contains(t, report, "- Camunda Version: 8.8")
	require.Contains(t, report, "- File: embedded/processdefinitions/C88_MultipleSubProcessesParent.bpmn")
	require.Contains(t, report, "- Requested Count: 1")
	require.Contains(t, report, "- Outcome: planned")
	require.Equal(t, []string{"GET /v2/topology", "GET /v2/topology"}, requests.Snapshot())
}

// TestOpsExecuteSmokeTestDryRunExistingReportFailsBeforePreflight preserves the existing command contract with advisory-aware fixtures.
func TestOpsExecuteSmokeTestDryRunExistingReportFailsBeforePreflight(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestDryRunServer(t, "8.8", &requests)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "smoke-test.md")
	const existingReport = "existing report"
	require.NoError(t, os.WriteFile(reportPath, []byte(existingReport), 0o600))

	output, err := testx.RunCmdSubprocess(t, "TestOpsExecuteSmokeTestInvalidLocalFlagsHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.8"),
		"C8VOLT_TEST_SMOKE_ARGS": marshalSmokeTestArgsForEnv(t, []string{
			"ops", "execute", "smoke-test",
			"--dry-run",
			"--report-file", reportPath,
		}),
	})
	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "report file already exists: "+reportPath)
	require.Equal(t, existingReport, readReportFile(t, reportPath))
	require.Empty(t, requests.Snapshot())
}

// TestOpsExecuteSmokeTestWritesMarkdownAuditReport verifies every deployed definition and cleanup outcome reaches Markdown.
func TestOpsExecuteSmokeTestWritesMarkdownAuditReport(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, "8.8", &requests, nil)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "smoke-test.md")

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "smoke-test",
		"--no-cleanup",
		"--report-file", reportPath,
	)

	require.Contains(t, output, "outcome: passed_cleanup_skipped")
	require.Contains(t, output, "report: written "+reportPath)
	require.Less(t, strings.Index(output, "report: written "+reportPath), strings.Index(output, "outcome: passed_cleanup_skipped"))
	report := readReportFile(t, reportPath)
	require.Contains(t, report, "# Execute Smoke Test Audit Report")
	require.Contains(t, report, "- Schema Version: ops.smoke-test.v1")
	require.Contains(t, report, "- Command: ops execute smoke-test")
	require.Contains(t, report, "- Dry Run: false")
	require.Contains(t, report, "- Camunda Version: 8.8")
	require.Contains(t, report, "- File: embedded/processdefinitions/C88_MultipleSubProcessesParent.bpmn")
	require.Contains(t, report, "- Process Definition Key: pd-88")
	require.Contains(t, report, "- Requested Count: 1")
	require.Contains(t, report, "- Created Count: 1")
	require.Contains(t, report, "Created Process Instance Keys:")
	require.Contains(t, report, "101")
	require.Contains(t, report, "## Walk")
	require.Contains(t, report, "root 101")
	require.Contains(t, report, "- No Cleanup: true")
	require.Contains(t, report, "- Retained Process Definition Key: pd-88")
	require.Contains(t, report, "### Deployed Process Definitions")
	require.Contains(t, report, "| pd-88-child | C88_SimpleUserTask | 4 | <default> | processdefinitions/C88_SimpleUserTask.bpmn |")
	require.Contains(t, report, "| pd-88-middle | C88_SimpleParent | 4 | <default> | processdefinitions/C88_SimpleParent.bpmn |")
	require.Contains(t, report, "| pd-88 | C88_MultipleSubProcessesParent | 4 | <default> | processdefinitions/C88_MultipleSubProcessesParent.bpmn |")
	require.Contains(t, report, "### Process Definition Cleanup Summary")
	require.Contains(t, report, "- Retained: 3")
	require.Contains(t, report, "### Process Definition Cleanup Results")
	require.Contains(t, report, "| pd-88-child | C88_SimpleUserTask | <default> | retained | cleanup disabled (--no-cleanup) |")
	require.Contains(t, report, "- Outcome: passed_cleanup_skipped")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/v2/resources/pd-88/deletion")
}

// TestOpsExecuteSmokeTestWritesJSONAuditReportWithInferredFormat preserves the existing command contract with advisory-aware fixtures.
func TestOpsExecuteSmokeTestWritesJSONAuditReportWithInferredFormat(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, "8.8", &requests, nil)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "smoke-test.json")

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "smoke-test",
		"--no-cleanup",
		"--report-file", reportPath,
	)

	require.Contains(t, output, "report: written "+reportPath)
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "ops.smoke-test.v1", report["schemaVersion"])
	require.Equal(t, "ops execute smoke-test", report["commandName"])
	require.Equal(t, "8.8", report["camundaVersion"])
	require.Equal(t, "<default>", report["tenantId"])
	require.Equal(t, "passed_cleanup_skipped", report["outcome"])
	fixture := requireJSONObject(t, report["fixture"])
	require.Equal(t, "embedded/processdefinitions/C88_MultipleSubProcessesParent.bpmn", fixture["file"])
	deployment := requireJSONObject(t, report["deployment"])
	require.Equal(t, "confirmed", deployment["status"])
	require.Equal(t, "pd-88", deployment["processDefinitionKey"])
	run := requireJSONObject(t, report["run"])
	require.Equal(t, float64(1), run["requestedCount"])
	require.Equal(t, float64(1), run["createdCount"])
	require.Equal(t, []any{"101"}, run["processInstanceKeys"])
	walk := requireJSONObject(t, report["walk"])
	require.Equal(t, "confirmed", walk["status"])
	require.Len(t, walk["items"], 1)
	cleanup := requireJSONObject(t, report["cleanup"])
	require.Equal(t, true, cleanup["noCleanup"])
	require.Equal(t, []any{"101"}, cleanup["retainedProcessInstanceKeys"])
	require.Equal(t, "pd-88", cleanup["retainedProcessDefinitionKey"])
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/v2/resources/pd-88/deletion")
}

// TestOpsExecuteSmokeTestExistingReportPreservation preserves the existing command contract with advisory-aware fixtures.
func TestOpsExecuteSmokeTestExistingReportPreservation(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "dry-run",
			args: []string{"ops", "execute", "smoke-test", "--dry-run"},
		},
		{
			name: "unconfirmed",
			args: []string{"ops", "execute", "smoke-test"},
		},
		{
			name: "no-cleanup",
			args: []string{"ops", "execute", "smoke-test", "--no-cleanup"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			srv := newOpsExecuteSmokeTestDryRunServer(t, "8.8", &requests)
			t.Cleanup(srv.Close)
			reportPath := filepath.Join(t.TempDir(), "smoke-test.md")
			const existingReport = "existing report"
			require.NoError(t, os.WriteFile(reportPath, []byte(existingReport), 0o600))
			args := append([]string{}, tt.args...)
			args = append(args, "--report-file", reportPath)

			output, err := testx.RunCmdSubprocess(t, "TestOpsExecuteSmokeTestInvalidLocalFlagsHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":     writeTestConfigForVersion(t, srv.URL, "8.8"),
				"C8VOLT_TEST_SMOKE_ARGS": marshalSmokeTestArgsForEnv(t, args),
			})
			require.Error(t, err)

			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.Error, exitErr.ExitCode())
			require.Contains(t, string(output), "report file already exists: "+reportPath)
			require.NotContains(t, string(output), "write audit report")
			require.Equal(t, existingReport, readReportFile(t, reportPath))
			require.Empty(t, requests.Snapshot())
		})
	}
}

// TestOpsExecuteSmokeTestAutomationJSONStdoutIsMachineOnly preserves the existing command contract with advisory-aware fixtures.
func TestOpsExecuteSmokeTestAutomationJSONStdoutIsMachineOnly(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, "8.8", &requests, nil)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "smoke-test.json")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"--json",
		"ops", "execute", "smoke-test",
		"--automation",
		"--no-cleanup",
		"--report-file", reportPath,
	)

	require.Empty(t, strings.TrimSpace(stderr))
	require.NotContains(t, stdout, "execute smoke test")
	require.NotContains(t, stdout, "report: written")
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "ops execute smoke-test", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "passed_cleanup_skipped", payload["outcome"])
	request := requireJSONObject(t, payload["request"])
	require.Equal(t, true, request["automation"])
	require.Equal(t, true, request["noCleanup"])
	require.NotEmpty(t, readReportFile(t, reportPath))
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/v2/resources/pd-88/deletion")
}

// TestOpsExecuteSmokeTestMachineOutputModesRemainExact verifies quiet JSON
// emits exactly one envelope through EOF without stderr contamination.
func TestOpsExecuteSmokeTestMachineOutputModesRemainExact(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, "8.8", &requests, nil)
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"--quiet", "--json",
		"ops", "execute", "smoke-test", "--automation", "--no-cleanup",
	)

	require.Empty(t, stderr)
	envelope := requireSingleJSONObjectDocument(t, stdout)
	require.Equal(t, "ops execute smoke-test", envelope["command"])
}

// TestOpsExecuteSmokeTestConfirmationTerminal verifies the destructive prompt
// uses real terminal stdin and the effective configured or inherited stderr.
func TestOpsExecuteSmokeTestConfirmationTerminal(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		runOpsExecuteSmokeTestConfirmationTerminalHelper(t)
		os.Exit(0)
	}

	for _, stderrMode := range []string{"configured", "inherited"} {
		t.Run(stderrMode, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			srv := newOpsExecuteSmokeTestRunWalkServer(t, "8.9", &requests, nil)
			t.Cleanup(srv.Close)
			capturePath := filepath.Join(t.TempDir(), "configured-stderr.txt")
			prompt := "smoke test: deploy fixture, start 1 process instance(s), walk process-instance families, then clean up created resources.\nDo you want to proceed? [y/N]: "

			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
				ScopeTestName: "TestOpsExecuteSmokeTestConfirmationTerminal",
				Env: map[string]string{
					opsSmokeTestTerminalConfigEnv:  writeTestConfigForVersion(t, srv.URL, "8.9"),
					opsSmokeTestTerminalStderrEnv:  stderrMode,
					opsSmokeTestTerminalCaptureEnv: capturePath,
				},
				Exchanges: []testx.CmdTerminalExchange{{Prompt: prompt, Response: "y"}},
				Timeout:   5 * time.Second,
			})

			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			require.NoError(t, result.Err, result.Stderr)
			require.NotContains(t, result.Stdout, "Do you want to proceed?")
			require.Equal(t, 1, strings.Count(result.Stderr, prompt))
			requestLog := strings.Join(requests.Snapshot(), "\n")
			require.Equal(t, 3, strings.Count(requestLog, "/v2/resources/"))
			require.Equal(t, 1, strings.Count(requestLog, "/v2/process-instances/101/deletion"))
			if stderrMode == "configured" {
				captured, err := os.ReadFile(capturePath)
				require.NoError(t, err)
				require.Equal(t, result.Stderr, string(captured))
			}
		})
	}
}

// runOpsExecuteSmokeTestConfirmationTerminalHelper executes the real smoke-test
// command with terminal stdin and either configured or inherited stderr.
func runOpsExecuteSmokeTestConfirmationTerminalHelper(t *testing.T) {
	require.Equal(t, "TestOpsExecuteSmokeTestConfirmationTerminal", os.Getenv(testx.CmdSubprocessNameEnv))
	require.True(t, term.IsTerminal(int(os.Stdin.Fd())))

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetOut(os.Stdout)
	if os.Getenv(opsSmokeTestTerminalStderrEnv) == "configured" {
		capture, err := os.Create(os.Getenv(opsSmokeTestTerminalCaptureEnv))
		require.NoError(t, err)
		defer capture.Close()
		root.SetErr(io.MultiWriter(os.Stderr, capture))
	} else {
		root.SetErr(nil)
	}
	root.SetArgs([]string{
		"--config", os.Getenv(opsSmokeTestTerminalConfigEnv),
		"--no-indicator",
		"ops", "execute", "smoke-test",
		"--no-wait",
	})
	_, _ = root.ExecuteC()
}

// TestOpsExecuteSmokeTestDeploysFixtureAndRendersDeploymentOutput verifies the
// final smoke-test summary remains while duplicate legacy progress is hidden.
func TestOpsExecuteSmokeTestDeploysFixtureAndRendersDeploymentOutput(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestDeploymentServer(t, "8.9", &requests)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"ops", "execute", "smoke-test",
		"--no-wait",
	)

	require.Contains(t, output, "execute smoke test")
	require.Contains(t, output, "fixture: embedded/processdefinitions/C89_MultipleSubProcessesParent.bpmn")
	require.Contains(t, output, "deployment: confirmed")
	require.Contains(t, output, "created process instances: 1/1")
	require.Contains(t, output, "walk: confirmed (process instances: 1)")
	require.Contains(t, output, "cleanup: submitted 1 process instance and 3 process definitions (--no-wait)")
	require.NotContains(t, output, "cleanup confirmation:")
	require.NotContains(t, output, "deploy: fixture embedded/processdefinitions/C89_MultipleSubProcessesParent.bpmn")
	require.NotContains(t, output, "deploy: confirmed process definition pd-89")
	require.NotContains(t, output, "start: 1 process instance")
	require.NotContains(t, output, "start: created 1/1")
	require.NotContains(t, output, "walk: 1 process-instance family")
	require.NotContains(t, output, "walk: confirmed 1 process-instance family")
	require.NotContains(t, output, "cleanup: deleting created resources")
	require.NotContains(t, output, "pi delete done")
	require.NotContains(t, output, "delete request sent")
	require.NotContains(t, output, "pd delete done")
	require.NotContains(t, output, "created keys: 101")
	require.NotContains(t, output, "walk 101:")
	require.Contains(t, output, "outcome: passed")
	require.Equal(t, []string{
		"GET /v2/topology",
		"GET /v2/topology",
		"POST /v2/deployments",
		"GET /v2/process-definitions/pd-89-child",
		"GET /v2/process-definitions/pd-89-middle",
		"GET /v2/process-definitions/pd-89",
		"POST /v2/process-instances",
		"GET /v2/process-instances/101",
		"GET /v2/process-instances/101",
		"GET /v2/process-instances/101",
		"POST /v2/process-instances/search",
		"GET /v2/process-instances/101",
		"GET /v2/process-instances/101",
		"POST /v2/process-instances/search",
		"GET /v2/process-instances/101",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/101/deletion",
		"POST /v2/process-instances/search",
		"GET /v2/process-definitions/pd-89-child",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/search",
		"POST /v2/resources/pd-89-child/deletion",
		"POST /v2/process-instances/search",
		"GET /v2/process-definitions/pd-89-middle",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/search",
		"POST /v2/resources/pd-89-middle/deletion",
		"POST /v2/process-instances/search",
		"GET /v2/process-definitions/pd-89",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/search",
		"POST /v2/resources/pd-89/deletion",
	}, requests.Snapshot())
}

// TestOpsExecuteSmokeTestCreatesAndWalksRequestedInstances verifies count
// aliases drive the final summary without reintroducing legacy progress logs.
func TestOpsExecuteSmokeTestCreatesAndWalksRequestedInstances(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "count flag",
			args: []string{"--count", "2"},
		},
		{
			name: "count shorthand",
			args: []string{"-n", "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			var createBodies testx.SafeSlice[string]
			srv := newOpsExecuteSmokeTestRunWalkServer(t, "8.8", &requests, &createBodies)
			t.Cleanup(srv.Close)

			args := append([]string{
				"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
				"ops", "execute", "smoke-test",
				"--workers", "1",
				"--no-cleanup",
			}, tt.args...)
			output := executeRootForProcessInstanceTest(t, args...)

			require.Contains(t, output, "cleanup: skipped (--no-cleanup)")
			require.Contains(t, output, "created process instances: 2/2")
			require.Contains(t, output, "walk: confirmed (process instances: 2)")
			require.Contains(t, output, "cleanup: skipped (--no-cleanup)")
			require.Contains(t, output, "outcome: passed_cleanup_skipped; use --verbose to list retained resources")
			require.NotContains(t, output, "start: 2 process instances")
			require.NotContains(t, output, "walk: 2 process-instance families")
			require.NotContains(t, output, "created keys: 101, 102")
			require.NotContains(t, output, "walk 101:")
			require.Len(t, createBodies.Snapshot(), 2)
			for _, body := range createBodies.Snapshot() {
				require.Contains(t, body, `"processDefinitionKey":"pd-88"`)
				require.NotContains(t, body, `"processDefinitionId"`)
			}
		})
	}
}

// TestOpsExecuteSmokeTestVerboseProgressRendersStageCounters verifies smoke-test progress is routed to stderr while stdout keeps final output.
func TestOpsExecuteSmokeTestVerboseProgressRendersStageCounters(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, "8.8", &requests, nil)
	t.Cleanup(srv.Close)

	_, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"--verbose",
		"ops", "execute", "smoke-test",
		"--workers", "1",
		"--no-cleanup",
		"--count", "2",
	)

	require.Contains(t, stderr, "deploying smoke-test fixture 1/1 deployment(s)")
	require.Contains(t, stderr, "starting process instances 2/2 process instance(s)")
	require.Contains(t, stderr, "walking process-instance families 2/2 process instance(s)")
}

// TestOpsExecuteSmokeTestNoCleanupHumanOutputReportsRetainedResources verifies verbose output accounts for all retained definitions.
func TestOpsExecuteSmokeTestNoCleanupHumanOutputReportsRetainedResources(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, "8.8", &requests, nil)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "smoke-test",
		"--no-cleanup",
		"--verbose",
	)

	require.Contains(t, output, "cleanup: skipped (--no-cleanup)")
	require.Contains(t, output, "retained process instances: 101")
	require.Contains(t, output, "deployed process definition: pd-88-child (C88_SimpleUserTask, version 4, tenant <default>)")
	require.Contains(t, output, "process definition pd-88-child (C88_SimpleUserTask, version 4, tenant <default>): retained - cleanup disabled (--no-cleanup)")
	require.Contains(t, output, "process definition pd-88-middle (C88_SimpleParent, version 4, tenant <default>): retained - cleanup disabled (--no-cleanup)")
	require.Contains(t, output, "process definition pd-88 (C88_MultipleSubProcessesParent, version 4, tenant <default>): retained - cleanup disabled (--no-cleanup)")
	require.Contains(t, output, "pd-88-child skipped: cleanup disabled (--no-cleanup) (cleaning up smoke-test process definitions, 1/3 process definition(s))")
	require.Contains(t, output, "pd-88 skipped: cleanup disabled (--no-cleanup) (cleaning up smoke-test process definitions, 3/3 process definition(s))")
	require.Contains(t, output, "outcome: passed_cleanup_skipped")
	requestLog := strings.Join(requests.Snapshot(), "\n")
	require.NotContains(t, requestLog, "DELETE /v1/process-instances/")
	require.NotContains(t, requestLog, "/v2/resources/pd-88/deletion")
}

// TestOpsExecuteSmokeTestNoCleanupJSONOutputReportsRetainedResources verifies the shared envelope carries the complete scope.
func TestOpsExecuteSmokeTestNoCleanupJSONOutputReportsRetainedResources(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, "8.8", &requests, nil)
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"--json",
		"ops", "execute", "smoke-test",
		"--no-cleanup",
	)

	require.Empty(t, strings.TrimSpace(stderr))
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "passed_cleanup_skipped", payload["outcome"])
	cleanup := requireJSONObject(t, payload["cleanup"])
	require.Equal(t, true, cleanup["noCleanup"])
	require.Equal(t, []any{"101"}, cleanup["retainedProcessInstanceKeys"])
	require.Equal(t, "pd-88", cleanup["retainedProcessDefinitionKey"])
	require.Equal(t, "C88_MultipleSubProcessesParent", cleanup["retainedBpmnProcessId"])
	deployment := requireJSONObject(t, payload["deployment"])
	require.Equal(t, true, deployment["definitionScopeComplete"])
	require.Len(t, deployment["processDefinitions"].([]any), 3)
	definitionSummary := requireJSONObject(t, cleanup["definitionSummary"])
	require.Equal(t, "skipped", definitionSummary["status"])
	require.Equal(t, float64(3), definitionSummary["total"])
	require.Equal(t, float64(3), definitionSummary["retained"])
	definitionResults := cleanup["definitionResults"].([]any)
	require.Len(t, definitionResults, 3)
	wantDefinitionKeys := []string{"pd-88-child", "pd-88-middle", "pd-88"}
	for i, wantKey := range wantDefinitionKeys {
		item := requireJSONObject(t, definitionResults[i])
		require.Equal(t, wantKey, requireJSONObject(t, item["definition"])["processDefinitionKey"])
		require.Equal(t, "retained", item["disposition"])
	}
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/v2/resources/pd-88/deletion")
}

// TestOpsExecuteSmokeTestMapsWorkerControlsToJSONRequest preserves the existing command contract with advisory-aware fixtures.
func TestOpsExecuteSmokeTestMapsWorkerControlsToJSONRequest(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestDryRunServer(t, "8.9", &requests)
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"--json",
		"ops", "execute", "smoke-test",
		"--dry-run",
		"--count", "4",
		"--workers", "3",
		"--fail-fast",
		"--no-worker-limit",
	)

	require.Empty(t, strings.TrimSpace(stderr))
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	payload := requireJSONObject(t, envelope["payload"])
	request := requireJSONObject(t, payload["request"])
	require.Equal(t, float64(4), request["count"])
	require.Equal(t, float64(3), request["workers"])
	require.Equal(t, true, request["failFast"])
	require.Equal(t, true, request["noWorkerLimit"])
	run := requireJSONObject(t, payload["run"])
	require.Equal(t, "planned", run["status"])
	require.Equal(t, float64(4), run["requestedCount"])
}

// TestOpsExecuteSmokeTestUsesImplicitConfirmationForCleanup verifies no-wait output counts every accepted definition request.
func TestOpsExecuteSmokeTestUsesImplicitConfirmationForCleanup(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, "8.9", &requests, nil)
	t.Cleanup(srv.Close)
	var prompts []string
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
		require.False(t, autoConfirm)
		prompts = append(prompts, prompt)
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"ops", "execute", "smoke-test",
		"--no-wait",
	)

	require.Contains(t, output, "cleanup: submitted 1 process instance and 3 process definitions (--no-wait)")
	require.NotContains(t, output, "cleanup confirmation:")
	require.Len(t, prompts, 1)
	require.Contains(t, prompts[0], "smoke test: deploy fixture, start 1 process instance(s), walk process-instance families, then clean up created resources")
}

// TestOpsExecuteSmokeTestAutomationNoCleanupDoesNotPrompt preserves the existing command contract with advisory-aware fixtures.
func TestOpsExecuteSmokeTestAutomationNoCleanupDoesNotPrompt(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, "8.8", &requests, nil)
	t.Cleanup(srv.Close)
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(_ io.Writer, _ bool, _ string) error {
		t.Fatal("unexpected confirmation prompt for --automation --no-cleanup")
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "smoke-test",
		"--automation",
		"--no-cleanup",
	)

	require.Contains(t, output, "outcome: passed_cleanup_skipped")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/v2/resources/pd-88/deletion")
}

// TestOpsExecuteSmokeTestDirtyClusterSkipsProcessDefinitionCleanup preserves the existing command contract with advisory-aware fixtures.
func TestOpsExecuteSmokeTestDirtyClusterSkipsProcessDefinitionCleanup(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServerWithCleanupBlocker(t, "8.8", &requests, nil, "999")
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsExecuteSmokeTestInvalidLocalFlagsHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.8"),
		"C8VOLT_TEST_SMOKE_ARGS": marshalSmokeTestArgsForEnv(t, []string{
			"ops", "execute", "smoke-test",
			"--auto-confirm",
			"--no-wait",
		}),
	})

	require.NoError(t, err)
	require.Contains(t, string(output), "process-definition cleanup blockers: 1")
	require.Contains(t, string(output), "outcome: passed_cleanup_skipped")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/v2/resources/pd-88/deletion")
}

// TestOpsExecuteSmokeTestBlockedDependencyReportsMixedEvidence verifies eligible siblings proceed while verbose and audit output explain the retained dependency.
func TestOpsExecuteSmokeTestBlockedDependencyReportsMixedEvidence(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServerWithBlockedDefinition(t, "8.9", &requests, nil, "pd-89-child", "999")
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "smoke-test.md")

	outputBytes, err := testx.RunCmdSubprocess(t, "TestOpsExecuteSmokeTestInvalidLocalFlagsHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_SMOKE_ARGS": marshalSmokeTestArgsForEnv(t, []string{
			"--verbose",
			"ops", "execute", "smoke-test",
			"--auto-confirm",
			"--no-wait",
			"--report-file", reportPath,
			"--report-format", "markdown",
		}),
	})
	require.NoError(t, err)
	output := string(outputBytes)

	require.Contains(t, output, "cleanup: skipped (1 process instance; process definitions: 2 submitted, 1 retained)")
	require.Contains(t, output, "process definition pd-89-child (C89_SimpleUserTask, version 4, tenant <default>): retained - unrelated process instances: 999")
	require.Contains(t, output, "process definition pd-89-middle (C89_SimpleParent, version 4, tenant <default>): submitted")
	require.Contains(t, output, "process definition pd-89 (C89_MultipleSubProcessesParent, version 4, tenant <default>): submitted")
	requestLog := strings.Join(requests.Snapshot(), "\n")
	require.NotContains(t, requestLog, "/v2/resources/pd-89-child/deletion")
	require.Contains(t, requestLog, "/v2/resources/pd-89-middle/deletion")
	require.Contains(t, requestLog, "/v2/resources/pd-89/deletion")
	report, err := os.ReadFile(reportPath)
	require.NoError(t, err)
	require.Contains(t, string(report), "| pd-89-child | C89_SimpleUserTask | <default> | retained | unrelated process instances: 999 |")
	require.Contains(t, string(report), "- Submitted: 2")
	require.Contains(t, string(report), "- Retained: 1")
}

// newOpsExecuteSmokeTestDryRunServer keeps fixture setup aligned with the invocation's advisory version check.
func newOpsExecuteSmokeTestDryRunServer(t *testing.T, version string, requests *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/topology":
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, version+".0", 1, 1)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

// newOpsExecuteSmokeTestDeploymentServer keeps fixture setup aligned with the invocation's advisory version check.
func newOpsExecuteSmokeTestDeploymentServer(t *testing.T, version string, requests *testx.SafeSlice[string]) *httptest.Server {
	return newOpsExecuteSmokeTestRunWalkServer(t, version, requests, nil)
}

// newOpsExecuteSmokeTestRunWalkServer keeps fixture setup aligned with the invocation's advisory version check.
func newOpsExecuteSmokeTestRunWalkServer(t *testing.T, version string, requests *testx.SafeSlice[string], createBodies *testx.SafeSlice[string]) *httptest.Server {
	return newOpsExecuteSmokeTestRunWalkServerWithCleanupBlocker(t, version, requests, createBodies, "")
}

// newOpsExecuteSmokeTestRunWalkServerWithCleanupBlocker preserves the legacy fixture that blocks every deployed definition.
func newOpsExecuteSmokeTestRunWalkServerWithCleanupBlocker(t *testing.T, version string, requests *testx.SafeSlice[string], createBodies *testx.SafeSlice[string], cleanupBlocker string) *httptest.Server {
	return newOpsExecuteSmokeTestRunWalkServerWithBlockedDefinition(t, version, requests, createBodies, "", cleanupBlocker)
}

// newOpsExecuteSmokeTestRunWalkServerWithBlockedDefinition can retain one exact deployed definition while allowing sibling cleanup.
func newOpsExecuteSmokeTestRunWalkServerWithBlockedDefinition(t *testing.T, version string, requests *testx.SafeSlice[string], createBodies *testx.SafeSlice[string], blockedDefinitionKey string, cleanupBlocker string) *httptest.Server {
	t.Helper()

	var created int
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/topology":
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, version+".0", 1, 1)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			requests.Append(r.Method + " " + r.URL.Path)
			require.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			prefix := "C88"
			keyPrefix := "pd-88"
			if strings.Contains(string(body), "C89_MultipleSubProcessesParent") {
				prefix = "C89"
				keyPrefix = "pd-89"
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{
				"deploymentKey": "deployment-1",
				"tenantId": "<default>",
				"deployments": [
					{
						"processDefinition": {
							"processDefinitionId": "%[1]s_SimpleUserTask",
							"processDefinitionKey": "%[2]s-child",
							"processDefinitionVersion": 4,
							"resourceName": "processdefinitions/%[1]s_SimpleUserTask.bpmn",
							"tenantId": "<default>"
						}
					},
					{
						"processDefinition": {
							"processDefinitionId": "%[1]s_SimpleParent",
							"processDefinitionKey": "%[2]s-middle",
							"processDefinitionVersion": 4,
							"resourceName": "processdefinitions/%[1]s_SimpleParent.bpmn",
							"tenantId": "<default>"
						}
					},
					{
						"processDefinition": {
							"processDefinitionId": "%[1]s_MultipleSubProcessesParent",
							"processDefinitionKey": "%[2]s",
							"processDefinitionVersion": 4,
							"resourceName": "processdefinitions/%[1]s_MultipleSubProcessesParent.bpmn",
							"tenantId": "<default>"
						}
					}
				]
			}`, prefix, keyPrefix)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-definitions/pd-"):
			requests.Append(r.Method + " " + r.URL.Path)
			definitions := map[string]string{
				"pd-88":        "C88_MultipleSubProcessesParent",
				"pd-88-child":  "C88_SimpleUserTask",
				"pd-88-middle": "C88_SimpleParent",
				"pd-89":        "C89_MultipleSubProcessesParent",
				"pd-89-child":  "C89_SimpleUserTask",
				"pd-89-middle": "C89_SimpleParent",
			}
			processDefinitionKey := strings.TrimPrefix(r.URL.Path, "/v2/process-definitions/")
			processDefinitionID, ok := definitions[processDefinitionKey]
			require.Truef(t, ok, "unexpected process definition lookup: %s", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{
				"processDefinitionId": %q,
				"processDefinitionKey": %q
			}`, processDefinitionID, processDefinitionKey)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances":
			requests.Append(r.Method + " " + r.URL.Path)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			if createBodies != nil {
				createBodies.Append(string(body))
			}
			created++
			key := fmt.Sprintf("%d", 100+created)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(processInstanceCreationJSON(key)))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			requests.Append(r.Method + " " + r.URL.Path)
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(processInstanceJSON(key)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			requests.Append(r.Method + " " + r.URL.Path)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			w.Header().Set("Content-Type", "application/json")
			definitionMatches := blockedDefinitionKey == "" || strings.Contains(string(body), blockedDefinitionKey)
			if cleanupBlocker != "" && definitionMatches && strings.Contains(string(body), "processDefinitionKey") && !strings.Contains(string(body), `"state"`) {
				_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[%s],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, processInstanceJSON(cleanupBlocker))))
				return
			}
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/v1/process-instances/"):
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"deleted"}`))
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v2/process-instances/") && strings.HasSuffix(r.URL.Path, "/deletion"):
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"deleted"}`))
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v2/resources/") && strings.HasSuffix(r.URL.Path, "/deletion"):
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"batchOperation":{"batchOperationKey":"batch-1"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func processInstanceCreationJSON(key string) string {
	return fmt.Sprintf(`{
		"processInstanceKey": %q,
		"processDefinitionId": "C88_MultipleSubProcessesParent",
		"processDefinitionKey": "pd-88",
		"processDefinitionVersion": 4,
		"tenantId": "<default>"
	}`, key)
}

func processInstanceJSON(key string) string {
	return fmt.Sprintf(`{
		"hasIncident": false,
		"processDefinitionId": "C88_MultipleSubProcessesParent",
		"processDefinitionKey": "pd-88",
		"processDefinitionName": "C88_MultipleSubProcessesParent",
		"processDefinitionVersion": 4,
		"processInstanceKey": %q,
		"startDate": "2026-05-17T08:00:00Z",
		"state": "ACTIVE",
		"tenantId": "<default>"
	}`, key)
}

func marshalSmokeTestArgsForEnv(t *testing.T, args []string) string {
	t.Helper()

	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}
