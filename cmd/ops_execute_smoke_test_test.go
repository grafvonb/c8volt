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

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestOpsExecuteSmokeTestHelpDocumentsCommand(t *testing.T) {
	output := executeRootForProcessInstanceTest(t, "ops", "execute", "--help")

	assertHelpOutputContainsAll(t, output,
		"Discover predefined operational playbooks",
		"retention-policy",
		"smoke-test",
	)

	commandOutput := executeRootForProcessInstanceTest(t, "ops", "execute", "smoke-test", "--help")

	assertHelpOutputContainsAll(t, commandOutput,
		"Execute a cluster smoke test workflow",
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
		"./c8volt ops execute smoke-test --dry-run",
		"./c8volt ops execute smoke-test --count 10 --automation --json --report-file smoke-test.json --report-format json",
	)
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

func TestOpsExecuteSmokeTestDryRunHumanOutputPlansWithoutMutation(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestDryRunServer(t, &requests)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "smoke-test",
		"--dry-run",
		"--count", "2",
		"--no-cleanup",
	)

	require.Contains(t, output, "dry run: execute smoke test")
	require.Contains(t, output, "fixture: embedded/processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn")
	require.Contains(t, output, "workflow: would deploy the fixture, start 2 process instances, and walk their process-instance families")
	require.Contains(t, output, "cleanup: skipped (--no-cleanup)")
	require.NotContains(t, output, "connectivity: confirmed")
	require.NotContains(t, output, "deployment: planned -")
	require.NotContains(t, output, "run: planned -")
	require.NotContains(t, output, "cleanup: planned")
	require.Contains(t, output, "outcome: planned; no changes applied")
	require.Equal(t, []string{"GET /v2/topology"}, requests.Snapshot())
}

func TestOpsExecuteSmokeTestDryRunJSONOutputIsStructured(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestDryRunServer(t, &requests)
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
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
	require.Equal(t, "8.9", plan["camundaVersion"])
	fixture := requireJSONObject(t, plan["fixture"])
	require.Equal(t, "embedded/processdefinitions/C89_MultipleSubProcessesParentProcess.bpmn", fixture["file"])
	steps := plan["plannedSteps"].([]any)
	require.Len(t, steps, 7)
	connectivity := requireJSONObject(t, steps[0])
	require.Equal(t, "connectivity", connectivity["name"])
	require.Equal(t, "confirmed", connectivity["status"])
	require.Equal(t, []string{"GET /v2/topology"}, requests.Snapshot())
}

func TestOpsExecuteSmokeTestDryRunWritesMarkdownReport(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestDryRunServer(t, &requests)
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
	require.Contains(t, report, "# Smoke Test Audit Report")
	require.Contains(t, report, "- Command: ops execute smoke-test")
	require.Contains(t, report, "- Dry Run: true")
	require.Contains(t, report, "- Camunda Version: 8.8")
	require.Contains(t, report, "- File: embedded/processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn")
	require.Contains(t, report, "- Requested Count: 1")
	require.Contains(t, report, "- Outcome: planned")
	require.Equal(t, []string{"GET /v2/topology"}, requests.Snapshot())
}

func TestOpsExecuteSmokeTestDryRunExistingReportFailsBeforePreflight(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestDryRunServer(t, &requests)
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

func TestOpsExecuteSmokeTestWritesMarkdownAuditReport(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, &requests, nil)
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
	require.Contains(t, report, "# Smoke Test Audit Report")
	require.Contains(t, report, "- Schema Version: ops.smoke-test.v1")
	require.Contains(t, report, "- Command: ops execute smoke-test")
	require.Contains(t, report, "- Dry Run: false")
	require.Contains(t, report, "- Camunda Version: 8.8")
	require.Contains(t, report, "- File: embedded/processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn")
	require.Contains(t, report, "- Process Definition Key: pd-88")
	require.Contains(t, report, "- Requested Count: 1")
	require.Contains(t, report, "- Created Count: 1")
	require.Contains(t, report, "Created Process Instance Keys:")
	require.Contains(t, report, "101")
	require.Contains(t, report, "## Walk")
	require.Contains(t, report, "root 101")
	require.Contains(t, report, "- No Cleanup: true")
	require.Contains(t, report, "- Retained Process Definition Key: pd-88")
	require.Contains(t, report, "- Outcome: passed_cleanup_skipped")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/v2/resources/pd-88/deletion")
}

func TestOpsExecuteSmokeTestWritesJSONAuditReportWithInferredFormat(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, &requests, nil)
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
	require.Equal(t, "embedded/processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn", fixture["file"])
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
			srv := newOpsExecuteSmokeTestDryRunServer(t, &requests)
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

func TestOpsExecuteSmokeTestAutomationJSONStdoutIsMachineOnly(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, &requests, nil)
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

func TestOpsExecuteSmokeTestDeploysFixtureAndRendersDeploymentOutput(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestDeploymentServer(t, &requests)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "smoke-test",
		"--no-wait",
	)

	require.Contains(t, output, "execute smoke test")
	require.Contains(t, output, "deploy: fixture embedded/processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn")
	require.Contains(t, output, "deploy: confirmed process definition pd-88")
	require.Contains(t, output, "start: 1 process instance")
	require.Contains(t, output, "start: created 1/1")
	require.Contains(t, output, "walk: 1 process-instance family")
	require.Contains(t, output, "walk: confirmed 1 process-instance family")
	require.Contains(t, output, "cleanup: deleting created resources")
	require.Contains(t, output, "fixture: embedded/processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn")
	require.Contains(t, output, "deployment: confirmed")
	require.Contains(t, output, "created process instances: 1/1")
	require.Contains(t, output, "walk: confirmed (process instances: 1)")
	require.Contains(t, output, "cleanup: submitted 1 process instance and fixture process definition (--no-wait)")
	require.NotContains(t, output, "cleanup confirmation:")
	require.NotContains(t, output, "pi delete done")
	require.NotContains(t, output, "delete request sent")
	require.NotContains(t, output, "pd delete done")
	require.NotContains(t, output, "created keys: 101")
	require.NotContains(t, output, "walk 101:")
	require.Contains(t, output, "outcome: passed")
	require.Equal(t, []string{
		"GET /v2/topology",
		"POST /v2/deployments",
		"GET /v2/process-definitions/pd-88",
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
		"DELETE /v1/process-instances/101",
		"POST /v2/process-instances/search",
		"GET /v2/process-definitions/pd-88",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/search",
		"POST /v2/resources/pd-88/deletion",
	}, requests.Snapshot())
}

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
			srv := newOpsExecuteSmokeTestRunWalkServer(t, &requests, &createBodies)
			t.Cleanup(srv.Close)

			args := append([]string{
				"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
				"ops", "execute", "smoke-test",
				"--workers", "1",
				"--no-cleanup",
			}, tt.args...)
			output := executeRootForProcessInstanceTest(t, args...)

			require.Contains(t, output, "start: 2 process instances")
			require.Contains(t, output, "walk: 2 process-instance families")
			require.Contains(t, output, "cleanup: skipped (--no-cleanup)")
			require.Contains(t, output, "created process instances: 2/2")
			require.Contains(t, output, "walk: confirmed (process instances: 2)")
			require.Contains(t, output, "cleanup: skipped (--no-cleanup)")
			require.Contains(t, output, "outcome: passed_cleanup_skipped; use --verbose to list retained resources")
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

func TestOpsExecuteSmokeTestNoCleanupHumanOutputReportsRetainedResources(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, &requests, nil)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "smoke-test",
		"--no-cleanup",
		"--verbose",
	)

	require.Contains(t, output, "cleanup: skipped (--no-cleanup)")
	require.Contains(t, output, "retained process instances: 101")
	require.Contains(t, output, "retained process definition: pd-88 (C88_MultipleSubProcessesParentProcess)")
	require.Contains(t, output, "outcome: passed_cleanup_skipped")
	requestLog := strings.Join(requests.Snapshot(), "\n")
	require.NotContains(t, requestLog, "DELETE /v1/process-instances/")
	require.NotContains(t, requestLog, "/v2/resources/pd-88/deletion")
}

func TestOpsExecuteSmokeTestNoCleanupJSONOutputReportsRetainedResources(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, &requests, nil)
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
	require.Equal(t, "C88_MultipleSubProcessesParentProcess", cleanup["retainedBpmnProcessId"])
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/v2/resources/pd-88/deletion")
}

func TestOpsExecuteSmokeTestMapsWorkerControlsToJSONRequest(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestDryRunServer(t, &requests)
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

func TestOpsExecuteSmokeTestUsesImplicitConfirmationForCleanup(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, &requests, nil)
	t.Cleanup(srv.Close)
	var prompts []string
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		require.False(t, autoConfirm)
		prompts = append(prompts, prompt)
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "smoke-test",
		"--no-wait",
	)

	require.Contains(t, output, "cleanup: submitted 1 process instance and fixture process definition (--no-wait)")
	require.NotContains(t, output, "cleanup confirmation:")
	require.Len(t, prompts, 1)
	require.Contains(t, prompts[0], "clean up the created instances and eligible process definition")
}

func TestOpsExecuteSmokeTestAutomationNoCleanupDoesNotPrompt(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServer(t, &requests, nil)
	t.Cleanup(srv.Close)
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(bool, string) error {
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

func TestOpsExecuteSmokeTestUnsafeCleanupBlockerExitsWithError(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestRunWalkServerWithCleanupBlocker(t, &requests, nil, "999")
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsExecuteSmokeTestInvalidLocalFlagsHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.8"),
		"C8VOLT_TEST_SMOKE_ARGS": marshalSmokeTestArgsForEnv(t, []string{
			"ops", "execute", "smoke-test",
			"--auto-confirm",
			"--no-wait",
		}),
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "process-definition cleanup blocked")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/v2/resources/pd-88/deletion")
}

func newOpsExecuteSmokeTestDryRunServer(t *testing.T, requests *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/topology":
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, "8.8.0", 1, 1)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func newOpsExecuteSmokeTestDeploymentServer(t *testing.T, requests *testx.SafeSlice[string]) *httptest.Server {
	return newOpsExecuteSmokeTestRunWalkServer(t, requests, nil)
}

func newOpsExecuteSmokeTestRunWalkServer(t *testing.T, requests *testx.SafeSlice[string], createBodies *testx.SafeSlice[string]) *httptest.Server {
	return newOpsExecuteSmokeTestRunWalkServerWithCleanupBlocker(t, requests, createBodies, "")
}

func newOpsExecuteSmokeTestRunWalkServerWithCleanupBlocker(t *testing.T, requests *testx.SafeSlice[string], createBodies *testx.SafeSlice[string], cleanupBlocker string) *httptest.Server {
	t.Helper()

	var created int
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/topology":
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, "8.8.0", 1, 1)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			requests.Append(r.Method + " " + r.URL.Path)
			require.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"deploymentKey": "deployment-1",
				"tenantId": "<default>",
				"deployments": [
					{
						"processDefinition": {
							"processDefinitionId": "C88_MultipleSubProcessesParentProcess",
							"processDefinitionKey": "pd-88",
							"processDefinitionVersion": 4,
							"resourceName": "processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn",
							"tenantId": "<default>"
						}
					}
				]
			}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-definitions/pd-88":
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"processDefinitionId": "C88_MultipleSubProcessesParentProcess",
				"processDefinitionKey": "pd-88"
			}`))
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
			if cleanupBlocker != "" && strings.Contains(string(body), "processDefinitionKey") && !strings.Contains(string(body), `"state"`) {
				_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[%s],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, processInstanceJSON(cleanupBlocker))))
				return
			}
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/v1/process-instances/"):
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
		"processDefinitionId": "C88_MultipleSubProcessesParentProcess",
		"processDefinitionKey": "pd-88",
		"processDefinitionVersion": 4,
		"tenantId": "<default>"
	}`, key)
}

func processInstanceJSON(key string) string {
	return fmt.Sprintf(`{
		"hasIncident": false,
		"processDefinitionId": "C88_MultipleSubProcessesParentProcess",
		"processDefinitionKey": "pd-88",
		"processDefinitionName": "C88_MultipleSubProcessesParentProcess",
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
