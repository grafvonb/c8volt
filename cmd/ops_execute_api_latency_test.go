// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestOpsExecuteAPILatencyHelpDocumentsActiveSurface verifies the active diagnostic command is discoverable.
func TestOpsExecuteAPILatencyHelpDocumentsActiveSurface(t *testing.T) {
	output := executeRootForTest(t, "ops", "execute", "api-latency-test", "--help")

	assertHelpOutputContainsAll(t, output,
		"Execute a bounded active API latency test",
		"deploys the existing version-matched SimpleUserTask fixture",
		"requires one concrete tenant",
		"--dry-run validates and previews the active plan without mutation",
		"--no-cleanup explicitly retains run-owned resources",
		"JSON active execution requires --dry-run, --auto-confirm, or --automation",
		"-n, --count int",
		"-w, --workers int",
		"--dry-run",
		"--no-cleanup",
		"--report-file string",
		"--report-format string",
		"./c8volt ops execute api-latency-test --dry-run",
		"./c8volt ops execute api-latency-test --count 20 --workers 4 --auto-confirm",
	)
}

// TestCommandContractOpsExecuteAPILatency captures the active API latency machine contract.
func TestCommandContractOpsExecuteAPILatency(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetOpsExecuteAPILatencyTestFlags(t)

	capability := commandCapabilityForCommand(opsExecuteAPILatencyCmd)

	require.Equal(t, "ops execute api-latency-test", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AllTenantsSupportRejectedConcreteDestination, capability.AllTenantsSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "implicitly confirmed active API latency diagnostics")
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "one-line", Supported: true})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "json", Supported: true, MachinePreferred: true, Notes: "stdout remains one JSON document; active execution requires dry-run; auto-confirm; or automation"})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "keys-only", Supported: false, Notes: "latency diagnostics do not produce key lists"})
	require.Contains(t, capability.Flags, FlagContract{Name: "count", Shorthand: "n", Type: "int", Description: "primary process-instance create samples across active stages"})
	require.Contains(t, capability.Flags, FlagContract{Name: "workers", Shorthand: "w", Type: "int", Description: "maximum closed-loop workers and final active stage width"})
	require.Contains(t, capability.Flags, FlagContract{Name: "dry-run", Type: "bool", Description: "validate and preview the active API latency plan without mutation"})
	require.Contains(t, capability.Flags, FlagContract{Name: "no-cleanup", Type: "bool", Description: "retain active API latency resources after execution"})
	require.Contains(t, capability.Flags, FlagContract{Name: "report-file", Type: "string", Description: "write an API latency test report to the given path"})
	require.Contains(t, capability.Flags, FlagContract{Name: "report-format", Type: "string", Description: "API latency test report format: markdown, json (default inferred from report-file extension)"})
}

// TestOpsExecuteAPILatencyDefaultsAndValidation pins local flag and JSON guardrail behavior.
func TestOpsExecuteAPILatencyDefaultsAndValidation(t *testing.T) {
	cmd := resetOpsExecuteAPILatencyTestFlags(t)

	request, err := buildOpsExecuteAPILatencyRequest(cmd, testAPILatencyConfig())
	require.NoError(t, err)
	require.Equal(t, 20, request.Count)
	require.Equal(t, 4, request.Workers)
	require.Equal(t, ops.APILatencyModeActive, request.Mode)
	require.False(t, request.DryRun)
	require.False(t, request.NoCleanup)

	flagOpsExecuteAPILatencyCount = 2
	flagOpsExecuteAPILatencyWorkers = 4
	require.ErrorContains(t, validateOpsExecuteAPILatencyFlags(cmd), "--workers must be no greater than --count")

	flagOpsExecuteAPILatencyCount = 4
	flagOpsExecuteAPILatencyWorkers = 3
	require.ErrorContains(t, validateOpsExecuteAPILatencyFlags(cmd), "count 4 is too small for worker stages 1, 2, 3")

	flagOpsExecuteAPILatencyCount = 3
	flagOpsExecuteAPILatencyWorkers = 1
	flagViewKeysOnly = true
	require.ErrorContains(t, validateOpsExecuteAPILatencyFlags(cmd), "--keys-only is not supported")

	flagViewKeysOnly = false
	flagViewAsJson = true
	require.ErrorContains(t, validateOpsExecuteAPILatencyJSONGuardrails(cmd), "--json ops execute api-latency-test requires --dry-run, --auto-confirm, or --automation")

	flagOpsExecuteAPILatencyDryRun = true
	require.NoError(t, validateOpsExecuteAPILatencyJSONGuardrails(cmd))

	flagOpsExecuteAPILatencyReportFormat = "json"
	require.ErrorContains(t, validateOpsExecuteAPILatencyFlags(cmd), "--report-format requires --report-file")
}

// TestOpsExecuteAPILatencyDryRunJSONPlansWithoutMutation verifies dry-run dispatch performs preflight only.
func TestOpsExecuteAPILatencyDryRunJSONPlansWithoutMutation(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteAPILatencyDryRunServer(t, &requests, "8.9.0")
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"--json",
		"ops", "execute", "api-latency-test",
		"--dry-run",
		"--count", "7",
		"--workers", "4",
	)

	require.Empty(t, strings.TrimSpace(stderr))
	envelope := requireSingleJSONObjectDocument(t, stdout)
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "ops execute api-latency-test", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "planned", payload["outcome"])
	request := requireJSONObject(t, payload["request"])
	require.Equal(t, true, request["dryRun"])
	require.Equal(t, float64(7), request["count"])
	require.Equal(t, float64(4), request["workers"])
	plan := requireJSONObject(t, payload["plan"])
	require.Equal(t, "active", plan["mode"])
	require.NotEmpty(t, plan["runId"])
	require.Equal(t, float64(7), plan["primarySampleLimit"])
	require.Equal(t, float64(7), plan["primarySampleAllocation"])
	require.Equal(t, float64(1), plan["visibilityAttemptLimit"])
	fixture := requireJSONObject(t, plan["fixture"])
	require.Equal(t, "embedded/processdefinitions/C89_SimpleUserTask.bpmn", fixture["file"])
	cleanup := requireJSONObject(t, plan["cleanup"])
	require.Equal(t, true, cleanup["requested"])
	require.Equal(t, true, cleanup["supported"])
	require.Equal(t, []string{"GET /v2/topology"}, requests.Snapshot())
}

// TestOpsExecuteAPILatencyAllTenantsRejectsBeforePreflight protects the concrete-tenant mutation contract.
func TestOpsExecuteAPILatencyAllTenantsRejectsBeforePreflight(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteAPILatencyDryRunServer(t, &requests, "8.9.0")
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "api-latency.md")

	output, err := testx.RunCmdSubprocess(t, "TestOpsExecuteAPILatencyRootArgsHelper", map[string]string{
		"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, []string{
			"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
			"--all-tenants",
			"ops", "execute", "api-latency-test",
			"--dry-run",
			"--report-file", reportPath,
		}),
	})

	assertAllTenantsConcreteDestinationSubprocessFailure(t, output, err, "ops execute api-latency-test")
	require.Empty(t, requests.Snapshot())
	require.NoFileExists(t, reportPath)
}

// TestOpsExecuteAPILatencyJSONMutationRequiresImplicitConfirmation verifies machine stdout cannot be mixed with prompts.
func TestOpsExecuteAPILatencyJSONMutationRequiresImplicitConfirmation(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteAPILatencyDryRunServer(t, &requests, "8.9.0")
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsExecuteAPILatencyRootArgsHelper", map[string]string{
		"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, []string{
			"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
			"--json",
			"ops", "execute", "api-latency-test",
		}),
	})

	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "--json ops execute api-latency-test requires --dry-run, --auto-confirm, or --automation")
	require.Empty(t, requests.Snapshot())
}

// TestOpsExecuteAPILatencyReportPathPreflightRunsBeforeRemoteWork preserves shared report planning behavior.
func TestOpsExecuteAPILatencyReportPathPreflightRunsBeforeRemoteWork(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteAPILatencyDryRunServer(t, &requests, "8.9.0")
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "api-latency.md")
	const existingReport = "existing report"
	require.NoError(t, os.WriteFile(reportPath, []byte(existingReport), 0o600))

	output, err := testx.RunCmdSubprocess(t, "TestOpsExecuteAPILatencyRootArgsHelper", map[string]string{
		"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, []string{
			"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
			"ops", "execute", "api-latency-test",
			"--dry-run",
			"--report-file", reportPath,
		}),
	})

	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "report file already exists: "+reportPath)
	require.Equal(t, existingReport, readReportFile(t, reportPath))
	require.Empty(t, requests.Snapshot())
}

// TestOpsExecuteAPILatencyConfirmationIncludesNoCleanup verifies explicit retention still prompts outside automation.
func TestOpsExecuteAPILatencyConfirmationIncludesNoCleanup(t *testing.T) {
	cmd := resetOpsExecuteAPILatencyTestFlags(t)
	flagOpsExecuteAPILatencyNoCleanup = true
	request, err := buildOpsExecuteAPILatencyRequest(cmd, testAPILatencyConfig())
	require.NoError(t, err)

	prompt := opsExecuteAPILatencyConfirmationPrompt(request)

	require.Contains(t, prompt, "active API latency test: deploy fixture, create 20 process instance(s), measure read and visibility latency, then retain run-owned resources")
	require.Contains(t, prompt, "Do you want to proceed?")
}

// TestOpsExecuteAPILatencyAutomationNoCleanupUsesImplicitConfirmation verifies automation confirms retained active execution.
func TestOpsExecuteAPILatencyAutomationNoCleanupUsesImplicitConfirmation(t *testing.T) {
	var requests testx.SafeSlice[string]
	var createBodies testx.SafeSlice[string]
	srv := newOpsExecuteAPILatencyActiveServer(t, &requests, &createBodies)
	t.Cleanup(srv.Close)
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		require.True(t, autoConfirm)
		require.Contains(t, prompt, "then retain run-owned resources")
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"--json",
		"ops", "execute", "api-latency-test",
		"--automation",
		"--no-cleanup",
		"--count", "1",
		"--workers", "1",
	)

	require.Empty(t, strings.TrimSpace(stderr))
	payload := requireJSONObject(t, requireSingleJSONObjectDocument(t, stdout)["payload"])
	require.Equal(t, "completed_retained", payload["outcome"])
	ownership := requireJSONObject(t, payload["ownership"])
	require.Equal(t, "pd-88", ownership["processDefinitionKey"])
	require.Equal(t, []any{"101"}, ownership["processInstanceKeys"])
	require.Len(t, createBodies.Snapshot(), 1)
	require.Contains(t, createBodies.Snapshot()[0], `"processDefinitionKey":"pd-88"`)
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/deletion")
}

// TestOpsExecuteAPILatencyJSONAutomationOutputSafetyExcludesProtectedValues verifies active machine output and reports stay sanitized.
func TestOpsExecuteAPILatencyJSONAutomationOutputSafetyExcludesProtectedValues(t *testing.T) {
	var requests testx.SafeSlice[string]
	var createBodies testx.SafeSlice[string]
	var authHeaders testx.SafeSlice[string]
	tokenSrv := newAPILatencyOAuthTokenServer(t)
	t.Cleanup(tokenSrv.Close)
	srv := newOpsExecuteAPILatencyOutputSafetyServer(t, &requests, &createBodies, &authHeaders)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "api-latency.md")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeAPILatencyOAuthConfig(t, srv.URL, tokenSrv.URL, "8.8"),
		"--json",
		"ops", "execute", "api-latency-test",
		"--automation",
		"--no-cleanup",
		"--count", "1",
		"--workers", "1",
		"--report-file", reportPath,
	)

	require.Empty(t, strings.TrimSpace(stderr))
	envelope := requireSingleJSONObjectDocument(t, stdout)
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "completed_retained", payload["outcome"])
	report := readReportFile(t, reportPath)
	require.Contains(t, report, "# Execute API Latency Test Report")
	require.Contains(t, report, "- Outcome: completed_retained")
	requireNoAPILatencyProtectedMarkers(t, stdout+"\n"+stderr+"\n"+report)
	require.Len(t, createBodies.Snapshot(), 1)
	requireNoAPILatencyProtectedMarkers(t, createBodies.Snapshot()[0])
	require.NotEmpty(t, authHeaders.Snapshot())
	for _, header := range authHeaders.Snapshot() {
		require.Equal(t, "Bearer "+apiLatencyProtectedAccessToken, header)
	}
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/deletion")
}

// TestOpsExecuteAPILatencyOverwritesExistingReportAfterMutation verifies active reports use confirmed-mutation write mode.
func TestOpsExecuteAPILatencyOverwritesExistingReportAfterMutation(t *testing.T) {
	var requests testx.SafeSlice[string]
	var createBodies testx.SafeSlice[string]
	srv := newOpsExecuteAPILatencyActiveServer(t, &requests, &createBodies)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "api-latency.md")
	require.NoError(t, os.WriteFile(reportPath, []byte("old report"), 0o600))

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "api-latency-test",
		"--auto-confirm",
		"--no-cleanup",
		"--count", "1",
		"--workers", "1",
		"--report-file", reportPath,
		"--report-format", "json",
	)

	require.Empty(t, stdout)
	require.Contains(t, stderr, "report: written "+reportPath)
	require.Less(t, strings.Index(stderr, "report: written "+reportPath), strings.Index(stderr, "outcome: completed_retained"))
	info, err := os.Stat(reportPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	require.NotContains(t, readReportFile(t, reportPath), "old report")
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, ops.APILatencySchemaVersion, report["schemaVersion"])
	require.Equal(t, "completed_retained", report["outcome"])
	require.NotContains(t, report, "payload")
	require.Equal(t, true, requireJSONObject(t, report["request"])["noCleanup"])
	require.Equal(t, "json", requireJSONObject(t, report["request"])["reportFormat"])
	require.Equal(t, "pd-88", requireJSONObject(t, report["ownership"])["processDefinitionKey"])
	require.Equal(t, []any{"101"}, requireJSONObject(t, report["ownership"])["processInstanceKeys"])
	require.Len(t, createBodies.Snapshot(), 1)
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/deletion")
}

// TestOpsExecuteAPILatencyWritesPartialReportBeforeCleanupFailure verifies raw reports preserve partial active evidence.
func TestOpsExecuteAPILatencyWritesPartialReportBeforeCleanupFailure(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "api-latency.json")
	result := ops.APILatencyResult{
		SchemaVersion: ops.APILatencySchemaVersion,
		Context: ops.APILatencyRunContext{
			CommandName:    "ops execute api-latency-test",
			SchemaVersion:  ops.APILatencySchemaVersion,
			CamundaVersion: "8.9",
			Profile:        "support",
			Tenant:         "<default>",
			Duration:       "1s",
		},
		Request: ops.APILatencyRequest{
			CommandName:  "ops execute api-latency-test",
			Mode:         ops.APILatencyModeActive,
			Count:        1,
			Workers:      1,
			ReportFile:   reportPath,
			ReportFormat: "json",
		},
		Plan: ops.APILatencyPlan{
			Mode:                    ops.APILatencyModeActive,
			RunID:                   "0123456789abcdef0123456789abcdef",
			PrimarySampleLimit:      1,
			PrimarySampleAllocation: 1,
			DerivedRequestLimit:     2,
			Stages:                  []ops.APILatencyStagePlan{{Index: 1, WorkerCount: 1, PrimarySamples: 1, DerivedRequestLimit: 2}},
			Cleanup:                 &ops.APILatencyCleanupPlan{Requested: true, Supported: true},
		},
		Ownership: &ops.APILatencyOwnership{
			RunID:                "0123456789abcdef0123456789abcdef",
			DeploymentSubmitted:  true,
			ProcessDefinitionKey: "pd-89",
			ProcessInstanceKeys:  []string{"101"},
		},
		Cleanup: []ops.APILatencyCleanupRecord{
			{ResourceType: ops.APILatencyCleanupResourceProcessInstance, Key: "101", Status: ops.APILatencyCleanupStatusFailed, Classification: ops.APILatencyClassificationRequestError, RecoveryCommand: "c8volt delete process-instance --key 101 --force --auto-confirm"},
			{ResourceType: ops.APILatencyCleanupResourceProcessDefinition, Key: "pd-89", Status: ops.APILatencyCleanupStatusFailed, Classification: ops.APILatencyClassificationRequestError, RecoveryCommand: "c8volt delete process-definition --key pd-89 --auto-confirm"},
		},
		Outcome: ops.APILatencyOutcomePartial,
	}

	require.NoError(t, writeOpsAPILatencyReport(result, testAPILatencyConfig(), opsExecuteAPILatencyReportWriteMode(result)))
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "partial", report["outcome"])
	require.Equal(t, "pd-89", requireJSONObject(t, report["ownership"])["processDefinitionKey"])
	cleanup := requireJSONItems(t, report["cleanup"], 2)
	require.Equal(t, "failed", requireJSONObject(t, cleanup[0])["status"])
	require.Contains(t, requireJSONObject(t, cleanup[0])["recoveryCommand"], "c8volt delete process-instance --key 101")
	require.Equal(t, "failed", requireJSONObject(t, cleanup[1])["status"])
	require.Contains(t, requireJSONObject(t, cleanup[1])["recoveryCommand"], "c8volt delete process-definition --key pd-89")
}

// TestOpsExecuteAPILatencyDryRunRendersActivePreview verifies human preview output is active-specific.
func TestOpsExecuteAPILatencyDryRunRendersActivePreview(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteAPILatencyDryRunServer(t, &requests, "8.9.0")
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"ops", "execute", "api-latency-test",
		"--dry-run",
		"--count", "7",
		"--workers", "4",
	)

	require.Empty(t, stdout)
	require.Contains(t, stderr, "execute api latency test")
	require.Contains(t, stderr, "request: count 7; primary allocation 7/7; workers 1,2,4; stages 3; derived requests <= 14")
	require.Contains(t, stderr, "run: ")
	require.Contains(t, stderr, "fixture: embedded/processdefinitions/C89_SimpleUserTask.bpmn (C89_SimpleUserTask)")
	require.Contains(t, stderr, "visibility: attempts <= 1")
	require.Contains(t, stderr, "cleanup: requested; supported")
	require.Contains(t, stderr, "stage 1: workers 1; primary 0/1; derived 0/2")
	require.Contains(t, stderr, "outcome: planned")
	require.NotContains(t, stderr, "analyse api latency")
	require.NotContains(t, stderr, "GET /v2")
	require.Equal(t, []string{"GET /v2/topology"}, requests.Snapshot())
}

// TestRenderOpsExecuteAPILatencyHumanRendersActiveResult verifies compact active result output.
func TestRenderOpsExecuteAPILatencyHumanRendersActiveResult(t *testing.T) {
	resetOpsExecuteAPILatencyTestFlags(t)
	p50 := 10 * time.Millisecond
	p95 := 20 * time.Millisecond
	maxLatency := 25 * time.Millisecond
	throughput := 40.0
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	result := ops.APILatencyResult{
		Request: ops.APILatencyRequest{
			Mode:  ops.APILatencyModeActive,
			Count: 1,
		},
		Plan: ops.APILatencyPlan{
			Mode:                    ops.APILatencyModeActive,
			RunID:                   "0123456789abcdef0123456789abcdef",
			PrimarySampleLimit:      1,
			PrimarySampleAllocation: 1,
			DerivedRequestLimit:     2,
			VisibilityAttemptLimit:  1,
			Stages: []ops.APILatencyStagePlan{{
				Index:               1,
				WorkerCount:         1,
				PrimarySamples:      1,
				DerivedRequestLimit: 2,
			}},
			Fixture: &ops.APILatencyFixturePlan{
				File:          "embedded/processdefinitions/C89_SimpleUserTask.bpmn",
				BpmnProcessID: "C89_SimpleUserTask",
			},
			Cleanup: &ops.APILatencyCleanupPlan{
				Requested: true,
				Supported: true,
			},
		},
		Stages: []ops.APILatencyStageResult{{
			Plan: ops.APILatencyStagePlan{
				Index:               1,
				WorkerCount:         1,
				PrimarySamples:      1,
				DerivedRequestLimit: 2,
			},
			Status:          ops.APILatencyStageStatusCompleted,
			PrimaryAttempts: 1,
			DerivedAttempts: 2,
			Categories: []ops.APILatencyCategorySummary{{
				Category:            ops.APILatencyCategoryProcessInstanceCreate,
				Attempts:            1,
				Successes:           1,
				ThroughputPerSecond: &throughput,
				P50:                 &p50,
				P95:                 &p95,
				Max:                 &maxLatency,
			}},
		}},
		Findings: []ops.APILatencyFinding{{
			Code:              "no_abnormal_evidence",
			LikelyArea:        "no abnormal evidence",
			Confidence:        ops.APILatencyFindingConfidenceLow,
			NextInvestigation: "compare with a read-only diagnostic if symptoms continue",
		}},
		Ownership: &ops.APILatencyOwnership{
			RunID:                "0123456789abcdef0123456789abcdef",
			FixtureName:          "embedded/processdefinitions/C89_SimpleUserTask.bpmn",
			BpmnProcessID:        "C89_SimpleUserTask",
			DeploymentSubmitted:  true,
			ProcessDefinitionKey: "pd-89",
			ProcessInstanceKeys:  []string{"101"},
		},
		Visibility: []ops.APILatencyVisibilityResult{{
			ProcessInstanceKey:  "101",
			Attempts:            1,
			AttemptLimit:        1,
			Visible:             true,
			Duration:            750 * time.Millisecond,
			FinalClassification: ops.APILatencyClassificationSuccess,
		}},
		Cleanup: []ops.APILatencyCleanupRecord{
			{ResourceType: ops.APILatencyCleanupResourceProcessInstance, Key: "101", Status: ops.APILatencyCleanupStatusDeleted, Classification: ops.APILatencyClassificationSuccess},
			{ResourceType: ops.APILatencyCleanupResourceProcessDefinition, Key: "pd-89", Status: ops.APILatencyCleanupStatusDeleted, Classification: ops.APILatencyClassificationSuccess},
		},
		Outcome: ops.APILatencyOutcomeCompleted,
	}

	require.NoError(t, renderOpsAPILatencyResult(cmd, result))
	output := out.String()
	require.Contains(t, output, "execute api latency test")
	require.Contains(t, output, "request: count 1; primary allocation 1/1; workers 1; stages 1; derived requests <= 2")
	require.Contains(t, output, "run: 0123456789abcdef0123456789abcdef")
	require.Contains(t, output, "fixture: embedded/processdefinitions/C89_SimpleUserTask.bpmn (C89_SimpleUserTask)")
	require.Contains(t, output, "stage 1: workers 1; primary 1/1; derived 2/2; errors 0; timeouts 0; unavailable 0")
	require.Contains(t, output, "p95 process_instance_create 20ms")
	require.Contains(t, output, "ownership: process definition recorded; process instances 1")
	require.Contains(t, output, "visibility: visible 1/1; attempts 1/1; max 750ms")
	require.Contains(t, output, "cleanup: deleted 2/2; retained 0; failed 0; unknown 0")
	require.Contains(t, output, "finding: no_abnormal_evidence; no abnormal evidence; confidence low")
	require.Contains(t, output, "outcome: completed")
	require.NotContains(t, output, "GET /v2")
	require.NotContains(t, output, "POST /v2")
	require.NotContains(t, output, "process-instance key 101")
	require.NotContains(t, output, "process-definition key pd-89")
}

// TestRenderOpsExecuteAPILatencyStableHumanAndJSON pins active renderer ordering, safe context, and active evidence fields.
func TestRenderOpsExecuteAPILatencyStableHumanAndJSON(t *testing.T) {
	resetOpsExecuteAPILatencyTestFlags(t)
	p50 := 11 * time.Millisecond
	p95 := 29 * time.Millisecond
	maxLatency := 31 * time.Millisecond
	visibilityP95 := 1400 * time.Millisecond
	throughput := 33.25
	result := ops.APILatencyResult{
		SchemaVersion: ops.APILatencySchemaVersion,
		Context: ops.APILatencyRunContext{
			CommandName:    "ops execute api-latency-test",
			SchemaVersion:  ops.APILatencySchemaVersion,
			C8voltVersion:  "dev-test",
			CamundaVersion: "8.9",
			Profile:        "support",
			Tenant:         "<default>",
			Duration:       "2s",
		},
		Request: ops.APILatencyRequest{
			CommandName: "ops execute api-latency-test",
			Mode:        ops.APILatencyModeActive,
			Count:       3,
			Workers:     2,
			TenantID:    "<default>",
			OutputMode:  "one-line",
		},
		Plan: ops.APILatencyPlan{
			Mode:                    ops.APILatencyModeActive,
			RunID:                   "0123456789abcdef0123456789abcdef",
			PrimarySampleLimit:      3,
			PrimarySampleAllocation: 3,
			DerivedRequestLimit:     9,
			VisibilityAttemptLimit:  2,
			Stages: []ops.APILatencyStagePlan{
				{Index: 1, WorkerCount: 1, PrimarySamples: 1, DerivedRequestLimit: 3},
				{Index: 2, WorkerCount: 2, PrimarySamples: 2, DerivedRequestLimit: 6},
			},
			Fixture: &ops.APILatencyFixturePlan{
				CamundaVersion: "8.9",
				File:           "embedded/processdefinitions/C89_SimpleUserTask.bpmn",
				BpmnProcessID:  "C89_SimpleUserTask",
				Available:      true,
			},
			Cleanup: &ops.APILatencyCleanupPlan{
				Requested:         true,
				Supported:         true,
				IndependentBudget: 5 * time.Minute,
			},
		},
		Topology: ops.APILatencyTopologyEvidence{BrokerCount: 1, PartitionCount: 1, HealthKnown: true},
		Stages: []ops.APILatencyStageResult{
			{
				Plan:                 ops.APILatencyStagePlan{Index: 1, WorkerCount: 1, PrimarySamples: 1, DerivedRequestLimit: 3},
				Status:               ops.APILatencyStageStatusCompleted,
				ActualMaxConcurrency: 1,
				PrimaryAttempts:      1,
				DerivedAttempts:      3,
				Categories: []ops.APILatencyCategorySummary{
					{Category: ops.APILatencyCategoryProcessInstanceCreate, Attempts: 1, Successes: 1, ThroughputPerSecond: &throughput, P50: &p50, P95: &p95, Max: &maxLatency},
					{Category: ops.APILatencyCategoryConcurrentRead, Attempts: 1, Successes: 1, ThroughputPerSecond: &throughput, P50: &p50, P95: &p95, Max: &maxLatency},
					{Category: ops.APILatencyCategorySearchVisibility, Attempts: 2, Successes: 1, Timeouts: 1, P95: &visibilityP95, Max: &visibilityP95},
				},
				Classifications: []ops.APILatencyClassificationCount{
					{Classification: ops.APILatencyClassificationSuccess, Count: 2},
					{Classification: ops.APILatencyClassificationBackpressure, Count: 1},
					{Classification: ops.APILatencyClassificationTimeout, Count: 1},
				},
			},
			{
				Plan:                 ops.APILatencyStagePlan{Index: 2, WorkerCount: 2, PrimarySamples: 2, DerivedRequestLimit: 6},
				Status:               ops.APILatencyStageStatusCompleted,
				ActualMaxConcurrency: 2,
				PrimaryAttempts:      2,
				DerivedAttempts:      6,
				Categories: []ops.APILatencyCategorySummary{
					{Category: ops.APILatencyCategoryProcessInstanceCreate, Attempts: 2, Successes: 2, ThroughputPerSecond: &throughput, P50: &p50, P95: &p95, Max: &maxLatency},
					{Category: ops.APILatencyCategorySearchVisibility, Attempts: 4, Successes: 4, P95: &visibilityP95, Max: &visibilityP95},
				},
				Classifications: []ops.APILatencyClassificationCount{
					{Classification: ops.APILatencyClassificationSuccess, Count: 6},
				},
			},
		},
		Findings: []ops.APILatencyFinding{
			{Code: "backpressure_evidence", LikelyArea: "cluster pressure", Confidence: ops.APILatencyFindingConfidenceHigh, NextInvestigation: "inspect broker resource usage"},
			{Code: "delayed_visibility", LikelyArea: "exporter visibility", Confidence: ops.APILatencyFindingConfidenceMedium, NextInvestigation: "compare exporter lag"},
		},
		Ownership: &ops.APILatencyOwnership{
			RunID:                "0123456789abcdef0123456789abcdef",
			FixtureName:          "embedded/processdefinitions/C89_SimpleUserTask.bpmn",
			BpmnProcessID:        "C89_SimpleUserTask",
			DeploymentSubmitted:  true,
			ProcessDefinitionKey: "pd-89",
			ProcessInstanceKeys:  []string{"101", "102", "103"},
		},
		Visibility: []ops.APILatencyVisibilityResult{
			{ProcessInstanceKey: "101", Attempts: 1, AttemptLimit: 2, Visible: true, Duration: 700 * time.Millisecond, FinalClassification: ops.APILatencyClassificationSuccess},
			{ProcessInstanceKey: "102", Attempts: 2, AttemptLimit: 2, Visible: false, Duration: 1500 * time.Millisecond, FinalClassification: ops.APILatencyClassificationTimeout},
		},
		Cleanup: []ops.APILatencyCleanupRecord{
			{ResourceType: ops.APILatencyCleanupResourceProcessInstance, Key: "101", Status: ops.APILatencyCleanupStatusDeleted, Classification: ops.APILatencyClassificationSuccess},
			{ResourceType: ops.APILatencyCleanupResourceProcessInstance, Key: "102", Status: ops.APILatencyCleanupStatusFailed, Classification: ops.APILatencyClassificationBackpressure, RecoveryCommand: "c8volt delete process-instance --key 102 --force --auto-confirm"},
			{ResourceType: ops.APILatencyCleanupResourceProcessDefinition, Key: "pd-89", Status: ops.APILatencyCleanupStatusUnknown, Classification: ops.APILatencyClassificationTimeout, RecoveryCommand: "c8volt delete process-definition --key pd-89 --auto-confirm"},
		},
		Notices:     []string{"fixture deployment and cleanup are reported outside primary samples"},
		Limitations: []string{"bounded evidence does not prove capacity"},
		Outcome:     ops.APILatencyOutcomePartial,
	}

	humanCmd := &cobra.Command{}
	var humanOut bytes.Buffer
	humanCmd.SetOut(&humanOut)
	started := time.Now()
	require.NoError(t, renderOpsAPILatencyResult(humanCmd, result))
	require.Less(t, time.Since(started), 5*time.Second)
	human := humanOut.String()
	require.Contains(t, human, "execute api latency test")
	require.Contains(t, human, "context: c8volt dev-test; profile support; tenant <default>; camunda 8.9")
	require.Contains(t, human, "request: count 3; primary allocation 3/3; workers 1,2; stages 2; derived requests <= 9")
	require.Contains(t, human, "run: 0123456789abcdef0123456789abcdef")
	require.Contains(t, human, "fixture: embedded/processdefinitions/C89_SimpleUserTask.bpmn (C89_SimpleUserTask)")
	require.Contains(t, human, "visibility: attempts <= 2")
	require.Contains(t, human, "cleanup: requested; supported")
	require.Contains(t, human, "stage 1: workers 1; primary 1/1; derived 3/3; errors 0; timeouts 1; unavailable 0; p95 process_instance_create 29ms; throughput 33.2/s")
	require.Contains(t, human, "stage 2: workers 2; primary 2/2; derived 6/6; errors 0; timeouts 0; unavailable 0; p95 process_instance_create 29ms; throughput 33.2/s; p50 delta -; throughput delta -")
	require.Less(t, strings.Index(human, "stage 1:"), strings.Index(human, "stage 2:"))
	require.Less(t, strings.Index(human, "finding: backpressure_evidence"), strings.Index(human, "finding: delayed_visibility"))
	require.Contains(t, human, "ownership: process definition recorded; process instances 3")
	require.Contains(t, human, "visibility: visible 1/2; attempts 3/4; max 1.5s")
	require.Contains(t, human, "cleanup: deleted 1/3; retained 0; failed 1; unknown 1")
	require.Contains(t, human, "cleanup resource: process_instance 102; failed; recovery: c8volt delete process-instance --key 102 --force --auto-confirm")
	require.Contains(t, human, "cleanup resource: process_definition pd-89; unknown; recovery: c8volt delete process-definition --key pd-89 --auto-confirm")
	require.Contains(t, human, "outcome: partial; elapsed 2s")
	require.NotContains(t, human, "GET /v2")
	require.NotContains(t, human, "Authorization")
	require.NotContains(t, human, "process-instance key 101")

	root := Root()
	resetCommandTreeFlags(root)
	jsonCmd, _, err := root.Find([]string{"ops", "execute", "api-latency-test"})
	require.NoError(t, err)
	var jsonOut bytes.Buffer
	jsonCmd.SetOut(&jsonOut)
	flagViewAsJson = true
	started = time.Now()
	require.NoError(t, renderOpsAPILatencyResult(jsonCmd, result))
	require.Less(t, time.Since(started), 5*time.Second)
	envelope := requireSingleJSONObjectDocument(t, jsonOut.String())
	require.Equal(t, "succeeded", envelope["outcome"])
	require.Equal(t, "ops execute api-latency-test", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, ops.APILatencySchemaVersion, payload["schemaVersion"])
	contextPayload := requireJSONObject(t, payload["context"])
	require.Equal(t, "ops execute api-latency-test", contextPayload["commandName"])
	require.Equal(t, "<default>", contextPayload["tenant"])
	plan := requireJSONObject(t, payload["plan"])
	require.Equal(t, "active", plan["mode"])
	require.Equal(t, float64(2), plan["visibilityAttemptLimit"])
	require.Equal(t, "embedded/processdefinitions/C89_SimpleUserTask.bpmn", requireJSONObject(t, plan["fixture"])["file"])
	require.Equal(t, true, requireJSONObject(t, plan["cleanup"])["requested"])
	stages := requireJSONItems(t, payload["stages"], 2)
	firstStage := requireJSONObject(t, stages[0])
	classifications := requireJSONItems(t, firstStage["classifications"], 3)
	require.Equal(t, "success", requireJSONObject(t, classifications[0])["classification"])
	require.Equal(t, "backpressure", requireJSONObject(t, classifications[1])["classification"])
	require.Equal(t, "timeout", requireJSONObject(t, classifications[2])["classification"])
	categories := requireJSONItems(t, firstStage["categories"], 3)
	require.Equal(t, "process_instance_create", requireJSONObject(t, categories[0])["category"])
	require.Equal(t, "concurrent_read", requireJSONObject(t, categories[1])["category"])
	require.Equal(t, "search_visibility", requireJSONObject(t, categories[2])["category"])
	findings := requireJSONItems(t, payload["findings"], 2)
	require.Equal(t, "backpressure_evidence", requireJSONObject(t, findings[0])["code"])
	require.Equal(t, "delayed_visibility", requireJSONObject(t, findings[1])["code"])
	ownership := requireJSONObject(t, payload["ownership"])
	require.Equal(t, "pd-89", ownership["processDefinitionKey"])
	require.Equal(t, []any{"101", "102", "103"}, ownership["processInstanceKeys"])
	visibility := requireJSONItems(t, payload["visibility"], 2)
	require.Equal(t, "101", requireJSONObject(t, visibility[0])["processInstanceKey"])
	require.Equal(t, "102", requireJSONObject(t, visibility[1])["processInstanceKey"])
	cleanup := requireJSONItems(t, payload["cleanup"], 3)
	require.Equal(t, "deleted", requireJSONObject(t, cleanup[0])["status"])
	require.Equal(t, "failed", requireJSONObject(t, cleanup[1])["status"])
	require.Equal(t, "unknown", requireJSONObject(t, cleanup[2])["status"])
}

// TestOpsExecuteAPILatencyInterruptContextScopesActiveExecution verifies signal cancellation is active only during the mutation window.
func TestOpsExecuteAPILatencyInterruptContextScopesActiveExecution(t *testing.T) {
	cmd := resetOpsExecuteAPILatencyTestFlags(t)
	parent := cmd.Context()
	var stopCalled bool
	prevContext := newOpsExecuteAPILatencyInterruptContext
	newOpsExecuteAPILatencyInterruptContext = func(parent context.Context) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		cancel()
		return ctx, func() { stopCalled = true }
	}
	t.Cleanup(func() { newOpsExecuteAPILatencyInterruptContext = prevContext })

	_, err := withOpsExecuteAPILatencyInterruptContext(cmd, ops.APILatencyRequest{Mode: ops.APILatencyModeActive}, func() (ops.APILatencyResult, error) {
		require.ErrorIs(t, cmd.Context().Err(), context.Canceled)
		return ops.APILatencyResult{}, cmd.Context().Err()
	})

	require.ErrorIs(t, err, context.Canceled)
	require.True(t, stopCalled)
	require.Same(t, parent, cmd.Context())
}

// TestRenderOpsExecuteAPILatencyHumanRendersRetainedAndRecoveryResources verifies compact output keeps exact recovery evidence.
func TestRenderOpsExecuteAPILatencyHumanRendersRetainedAndRecoveryResources(t *testing.T) {
	resetOpsExecuteAPILatencyTestFlags(t)
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	result := ops.APILatencyResult{
		Request: ops.APILatencyRequest{
			Mode:      ops.APILatencyModeActive,
			Count:     1,
			NoCleanup: true,
		},
		Plan: ops.APILatencyPlan{
			Mode:                    ops.APILatencyModeActive,
			RunID:                   "0123456789abcdef0123456789abcdef",
			PrimarySampleLimit:      1,
			PrimarySampleAllocation: 1,
			DerivedRequestLimit:     2,
			Stages: []ops.APILatencyStagePlan{{
				Index:               1,
				WorkerCount:         1,
				PrimarySamples:      1,
				DerivedRequestLimit: 2,
			}},
			Cleanup: &ops.APILatencyCleanupPlan{IntentionalRetention: true},
		},
		Ownership: &ops.APILatencyOwnership{
			RunID:                "0123456789abcdef0123456789abcdef",
			ProcessDefinitionKey: "pd-88",
			ProcessInstanceKeys:  []string{"101", "102"},
		},
		Cleanup: []ops.APILatencyCleanupRecord{
			{ResourceType: ops.APILatencyCleanupResourceProcessInstance, Key: "101", Status: ops.APILatencyCleanupStatusRetained, RecoveryCommand: "c8volt delete process-instance --key 101 --force --auto-confirm"},
			{ResourceType: ops.APILatencyCleanupResourceProcessInstance, Key: "102", Status: ops.APILatencyCleanupStatusUnknown, Classification: ops.APILatencyClassificationTimeout, RecoveryCommand: "c8volt delete process-instance --key 102 --force --auto-confirm"},
			{ResourceType: ops.APILatencyCleanupResourceProcessDefinition, Key: "pd-88", Status: ops.APILatencyCleanupStatusFailed, Classification: ops.APILatencyClassificationRequestError},
		},
		Outcome: ops.APILatencyOutcomePartial,
	}

	require.NoError(t, renderOpsAPILatencyResult(cmd, result))
	output := out.String()
	require.Contains(t, output, "cleanup: deleted 0/3; retained 1; failed 1; unknown 1")
	require.Contains(t, output, "cleanup resource: process_instance 101; retained; recovery: c8volt delete process-instance --key 101 --force --auto-confirm")
	require.Contains(t, output, "cleanup resource: process_instance 102; unknown; recovery: c8volt delete process-instance --key 102 --force --auto-confirm")
	require.Contains(t, output, "cleanup resource: process_definition pd-88; failed")
	require.Contains(t, output, "outcome: partial")
}

// TestOpsExecuteAPILatencyCleanupFailureUsesJSONErrorEnvelope verifies partial active failures keep the established machine error shape.
func TestOpsExecuteAPILatencyCleanupFailureUsesJSONErrorEnvelope(t *testing.T) {
	err := fmt.Errorf("ops execute api-latency-test: %w", context.Canceled)

	envelope := resultEnvelopeForError(opsExecuteAPILatencyCmd, err)
	require.Equal(t, OutcomeFailed, envelope.Outcome)
	require.Equal(t, "ops execute api-latency-test", envelope.Command)
	require.Nil(t, envelope.Payload)
	require.Contains(t, envelope.Detail.Message, "ops execute api-latency-test")
	require.Equal(t, exitcode.Error, ferrors.ResolveExitCode(false, err))
}

// TestOpsExecuteAPILatencyRequestedCleanupFailureSubprocessUsesJSONErrorEnvelope verifies exact cleanup failures exit nonzero without a partial stdout payload.
func TestOpsExecuteAPILatencyRequestedCleanupFailureSubprocessUsesJSONErrorEnvelope(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteAPILatencyCleanupFailureServer(t, &requests)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "api-latency.json")

	stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsExecuteAPILatencyRootArgsHelper", map[string]string{
		"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, []string{
			"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
			"--json",
			"ops", "execute", "api-latency-test",
			"--auto-confirm",
			"--count", "1",
			"--workers", "1",
			"--report-file", reportPath,
			"--report-format", "json",
		}),
	})

	requireAPILatencySubprocessExitCode(t, err, exitcode.Error)
	require.Empty(t, strings.TrimSpace(stderr))
	envelope := requireSingleJSONObjectDocument(t, stdout)
	require.Equal(t, string(OutcomeFailed), envelope["outcome"])
	require.Equal(t, "ops execute api-latency-test", envelope["command"])
	require.Nil(t, envelope["payload"])
	detail := requireJSONObject(t, envelope["detail"])
	require.Contains(t, detail["message"], "ops execute api-latency-test")
	require.Contains(t, detail["message"], "delete API latency process instance 101")
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "partial", report["outcome"])
	cleanup := requireJSONItems(t, report["cleanup"], 2)
	require.Equal(t, "failed", requireJSONObject(t, cleanup[0])["status"])
	require.Equal(t, "unknown", requireJSONObject(t, cleanup[1])["status"])
	require.Contains(t, requests.Snapshot(), "POST /v2/process-instances/101/deletion")
	require.NotContains(t, requests.Snapshot(), "POST /v2/resources/pd-89/deletion")
}

// TestOpsExecuteAPILatencyProgressModeGate verifies active aggregate progress stays out of protected modes.
func TestOpsExecuteAPILatencyProgressModeGate(t *testing.T) {
	for _, tc := range []struct {
		name       string
		setup      func()
		wantStderr string
	}{
		{name: "human", setup: func() {}, wantStderr: ""},
		{name: "json", setup: func() { flagViewAsJson = true }, wantStderr: ""},
		{name: "quiet", setup: func() { flagQuiet = true }, wantStderr: ""},
		{name: "automation", setup: func() { flagCmdAutomation = true }, wantStderr: ""},
		{name: "verbose", setup: func() { flagVerbose = true }, wantStderr: "running active API latency test, 1/2 process-instance create(s)"},
		{name: "debug", setup: func() { flagDebug = true }, wantStderr: "running active API latency test, 1/2 process-instance create(s)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetOpsExecuteAPILatencyTestFlags(t)
			tc.setup()
			cmd, stderr := newSemanticProgressStderrCommand()
			request := ops.APILatencyRequest{}
			progress := configureOpsAPILatencyProgress(cmd, &request)
			defer progress.Close()

			request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindFrozenScope, FrozenScope: &ops.FrozenScopeProgress{
				Phase:        "running active API latency test",
				CoreResource: "process-instance create(s)",
				Done:         1,
				Total:        2,
			}})

			if tc.wantStderr == "" {
				require.Empty(t, stderr.String())
				return
			}
			require.Contains(t, stderr.String(), tc.wantStderr)
			require.NotContains(t, stderr.String(), apiLatencyProtectedAuthorizationHeader)
		})
	}
}

// TestOpsExecuteAPILatencyRootArgsHelper runs root arguments in a subprocess for exit-code assertions.
func TestOpsExecuteAPILatencyRootArgsHelper(t *testing.T) {
	executeRootHelperFromArgsEnv(t, "C8VOLT_TEST_ROOT_ARGS")
}

func newOpsExecuteAPILatencyDryRunServer(t *testing.T, requests *testx.SafeSlice[string], gatewayVersion string) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/topology":
			_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, gatewayVersion, 1, 1)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func newOpsExecuteAPILatencyActiveServer(t *testing.T, requests *testx.SafeSlice[string], createBodies *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()

	var created int
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/topology":
			_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, "8.8.0", 1, 1)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			require.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), "C88_SimpleUserTask")
			_, _ = w.Write([]byte(`{
				"deploymentKey": "deployment-1",
				"tenantId": "<default>",
				"deployments": [{
					"processDefinition": {
						"processDefinitionId": "C88_SimpleUserTask",
						"processDefinitionKey": "pd-88",
						"processDefinitionVersion": 1,
						"resourceName": "processdefinitions/C88_SimpleUserTask.bpmn",
						"tenantId": "<default>"
					}
				}]
			}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			if createBodies != nil {
				createBodies.Append(string(body))
			}
			created++
			_, _ = w.Write([]byte(apiLatencyProcessInstanceCreationJSON(fmt.Sprintf("%d", 100+created))))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[%s],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, apiLatencyProcessInstanceJSON("101"))))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

// newOpsExecuteAPILatencyOutputSafetyServer returns active responses with ignored protected payload fields.
func newOpsExecuteAPILatencyOutputSafetyServer(t *testing.T, requests *testx.SafeSlice[string], createBodies *testx.SafeSlice[string], authHeaders *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		authHeaders.Append(r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/topology":
			_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, "8.8.0", 1, 1)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			require.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requireNoAPILatencyProtectedMarkers(t, string(body))
			require.Contains(t, string(body), "C88_SimpleUserTask")
			_, _ = w.Write([]byte(`{
				"deploymentKey": "deployment-1",
				"tenantId": "<default>",
				"deployments": [{
					"processDefinition": {
						"processDefinitionId": "C88_SimpleUserTask",
						"processDefinitionKey": "pd-88",
						"processDefinitionVersion": 1,
						"resourceName": "processdefinitions/C88_SimpleUserTask.bpmn",
						"tenantId": "<default>"
					}
				}]
			}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			createBodies.Append(string(body))
			_, _ = w.Write([]byte(apiLatencyProcessInstanceCreationJSON("101")))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			_, _ = w.Write([]byte(`{"items":[{"hasIncident":false,"processDefinitionId":"C88_SimpleUserTask","processDefinitionKey":"pd-88","processDefinitionName":"C88_SimpleUserTask","processDefinitionVersion":1,"processInstanceKey":"101","startDate":"2026-09-02T08:00:00Z","state":"ACTIVE","tenantId":"<default>","variables":{"token":"` + apiLatencyProtectedVariable + `"},"businessPayload":"` + apiLatencyProtectedPayload + `"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

// newOpsExecuteAPILatencyCleanupFailureServer returns a v8.9 active run that fails exact cleanup.
func newOpsExecuteAPILatencyCleanupFailureServer(t *testing.T, requests *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()

	var canceled atomic.Bool
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/topology":
			_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, "8.9.0", 1, 1)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			require.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), "C89_SimpleUserTask")
			_, _ = w.Write([]byte(`{
				"deploymentKey": "deployment-1",
				"tenantId": "<default>",
				"deployments": [{
					"processDefinition": {
						"processDefinitionId": "C89_SimpleUserTask",
						"processDefinitionKey": "pd-89",
						"processDefinitionVersion": 1,
						"resourceName": "processdefinitions/C89_SimpleUserTask.bpmn",
						"tenantId": "<default>"
					}
				}]
			}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances":
			_, _ = w.Write([]byte(apiLatencyProcessInstanceCreationJSONForDefinition("101", "pd-89", "C89_SimpleUserTask")))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			if strings.Contains(string(body), "parentProcessInstanceKey") {
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
				return
			}
			_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[%s],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, apiLatencyProcessInstanceJSONForDefinition("101", "pd-89", "C89_SimpleUserTask"))))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/101":
			state := "ACTIVE"
			if canceled.Load() {
				state = "CANCELED"
			}
			_, _ = w.Write([]byte(apiLatencyProcessInstanceJSONForDefinitionState("101", "pd-89", "C89_SimpleUserTask", state)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/101/cancellation":
			canceled.Store(true)
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/101/deletion":
			http.Error(w, `{"message":"delete rejected"}`, http.StatusInternalServerError)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/resources/pd-89/deletion":
			http.Error(w, `{"message":"resource delete rejected"}`, http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func apiLatencyProcessInstanceCreationJSON(key string) string {
	return apiLatencyProcessInstanceCreationJSONForDefinition(key, "pd-88", "C88_SimpleUserTask")
}

// apiLatencyProcessInstanceCreationJSONForDefinition builds create responses for version-specific active fixtures.
func apiLatencyProcessInstanceCreationJSONForDefinition(key string, processDefinitionKey string, bpmnProcessID string) string {
	return fmt.Sprintf(`{
		"processInstanceKey": %q,
		"processDefinitionId": %q,
		"processDefinitionKey": %q,
		"processDefinitionVersion": 1,
		"tenantId": "<default>"
	}`, key, bpmnProcessID, processDefinitionKey)
}

func apiLatencyProcessInstanceJSON(key string) string {
	return apiLatencyProcessInstanceJSONForDefinition(key, "pd-88", "C88_SimpleUserTask")
}

// apiLatencyProcessInstanceJSONForDefinition builds search responses for version-specific active fixtures.
func apiLatencyProcessInstanceJSONForDefinition(key string, processDefinitionKey string, bpmnProcessID string) string {
	return apiLatencyProcessInstanceJSONForDefinitionState(key, processDefinitionKey, bpmnProcessID, "ACTIVE")
}

// apiLatencyProcessInstanceJSONForDefinitionState builds a process-instance response with a caller-selected lifecycle state.
func apiLatencyProcessInstanceJSONForDefinitionState(key string, processDefinitionKey string, bpmnProcessID string, state string) string {
	return fmt.Sprintf(`{
		"hasIncident": false,
		"processDefinitionId": %q,
		"processDefinitionKey": %q,
		"processDefinitionName": %q,
		"processDefinitionVersion": 1,
		"processInstanceKey": %q,
		"startDate": "2026-09-02T08:00:00Z",
		"state": %q,
		"tenantId": "<default>"
	}`, bpmnProcessID, processDefinitionKey, bpmnProcessID, key, state)
}

func resetOpsExecuteAPILatencyTestFlags(t *testing.T) *cobra.Command {
	t.Helper()

	resetSemanticProgressModeFlags(t)
	prevCount := flagOpsExecuteAPILatencyCount
	prevWorkers := flagOpsExecuteAPILatencyWorkers
	prevDryRun := flagOpsExecuteAPILatencyDryRun
	prevNoCleanup := flagOpsExecuteAPILatencyNoCleanup
	prevReportFile := flagOpsExecuteAPILatencyReportFile
	prevReportFormat := flagOpsExecuteAPILatencyReportFormat
	t.Cleanup(func() {
		flagOpsExecuteAPILatencyCount = prevCount
		flagOpsExecuteAPILatencyWorkers = prevWorkers
		flagOpsExecuteAPILatencyDryRun = prevDryRun
		flagOpsExecuteAPILatencyNoCleanup = prevNoCleanup
		flagOpsExecuteAPILatencyReportFile = prevReportFile
		flagOpsExecuteAPILatencyReportFormat = prevReportFormat
	})
	flagOpsExecuteAPILatencyCount = opsExecuteAPILatencyDefaultCount
	flagOpsExecuteAPILatencyWorkers = opsExecuteAPILatencyDefaultWorkers
	flagOpsExecuteAPILatencyDryRun = false
	flagOpsExecuteAPILatencyNoCleanup = false
	flagOpsExecuteAPILatencyReportFile = ""
	flagOpsExecuteAPILatencyReportFormat = ""
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToContext(t.Context(), logging.New(logging.LoggerConfig{
		Format: "plain-time",
		Writer: io.Discard,
	})))
	return cmd
}
