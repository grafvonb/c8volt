// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
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
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const (
	apiLatencyProtectedAccessToken         = "api-latency-access-token-secret"
	apiLatencyProtectedClientSecret        = "api-latency-client-secret-value"
	apiLatencyProtectedVariable            = "variable-password-value"
	apiLatencyProtectedPayload             = "business-payload-value"
	apiLatencyProtectedAuthorizationHeader = "Authorization: Bearer " + apiLatencyProtectedAccessToken
	apiLatencyProtectedRawBody             = "raw-response-body-secret"
)

// TestOpsAPILatencyCommandErrorMapsOutcomes verifies completed diagnostics exit successfully while incomplete outcomes use the existing error path.
func TestOpsAPILatencyCommandErrorMapsOutcomes(t *testing.T) {
	for _, outcome := range []ops.APILatencyOutcome{
		ops.APILatencyOutcomePlanned,
		ops.APILatencyOutcomeCompleted,
		ops.APILatencyOutcomeCompletedRetained,
	} {
		require.NoError(t, opsAPILatencyCommandError("ops analyse api-latency", ops.APILatencyResult{Outcome: outcome}, context.Canceled))
	}

	err := opsAPILatencyCommandError("ops analyse api-latency", ops.APILatencyResult{Outcome: ops.APILatencyOutcomeInterrupted}, context.Canceled)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ops analyse api-latency")
	require.Contains(t, err.Error(), "context canceled")

	err = opsAPILatencyCommandError("ops execute api-latency-test", ops.APILatencyResult{Outcome: ops.APILatencyOutcomePartial}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "ops execute api-latency-test")
	require.Contains(t, err.Error(), "partial")
}

// TestOpsAnalyseAPILatencyHelpDocumentsReadOnlySurface verifies the diagnostic command is discoverable.
func TestOpsAnalyseAPILatencyHelpDocumentsReadOnlySurface(t *testing.T) {
	output := executeRootForTest(t, "ops", "analyse", "api-latency", "--help")

	assertHelpOutputContainsAll(t, output,
		"Analyse API latency without changing cluster state",
		"The command is read-only.",
		"read paths under bounded load",
		"--count sets the total sample budget",
		"--workers sets the maximum concurrency",
		"Use --verbose for stage details",
		"--report-file to save a Markdown or JSON report",
		"-n, --count int",
		"-w, --workers int",
		"--report-file string",
		"--report-format string",
		"./c8volt ops analyse api-latency --count 20 --workers 4",
		"./c8volt ops analyse api-latency -n 6 -w 2",
		"./c8volt --verbose ops analyse api-latency",
	)
	require.NotContains(t, output, "Keys-only output")
	require.NotContains(t, output, "shared command envelope")
}

// TestCommandContractOpsAnalyseAPILatency captures the read-only API latency machine contract.
func TestCommandContractOpsAnalyseAPILatency(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(opsAnalyseAPILatencyCmd)

	require.Equal(t, "ops analyse api-latency", capability.Path)
	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AllTenantsSupportAccepted, capability.AllTenantsSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.AutomationNotes, "read-only bounded diagnostics")
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "one-line", Supported: true})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "json", Supported: true, MachinePreferred: true, Notes: "stdout remains one JSON document; progress is suppressed"})
	require.Contains(t, capability.OutputModes, OutputModeContract{Name: "keys-only", Supported: false, Notes: "latency diagnostics do not produce key lists"})
	require.Contains(t, capability.Flags, FlagContract{Name: "count", Shorthand: "n", Type: "int", Description: "primary sample cycles to measure across all read-only stages"})
	require.Contains(t, capability.Flags, FlagContract{Name: "workers", Shorthand: "w", Type: "int", Description: "maximum closed-loop workers and final stage width"})
	require.Contains(t, capability.Flags, FlagContract{Name: "report-file", Type: "string", Description: "write an API latency report to the given path"})
	require.Contains(t, capability.Flags, FlagContract{Name: "report-format", Type: "string", Description: "API latency report format: markdown, json (default inferred from report-file extension)"})
}

// TestOpsAnalyseAPILatencyDefaultsAndValidation pins local budget validation before remote work.
func TestOpsAnalyseAPILatencyDefaultsAndValidation(t *testing.T) {
	cmd := resetOpsAnalyseAPILatencyTestFlags(t)

	flagOpsAnalyseAPILatencyCount = opsAnalyseAPILatencyDefaultCount
	flagOpsAnalyseAPILatencyWorkers = opsAnalyseAPILatencyDefaultWorkers
	request, err := buildOpsAnalyseAPILatencyRequest(cmd, testAPILatencyConfig())
	require.NoError(t, err)
	require.Equal(t, 20, request.Count)
	require.Equal(t, 4, request.Workers)
	require.Equal(t, ops.APILatencyModeReadOnly, request.Mode)

	flagOpsAnalyseAPILatencyCount = 2
	flagOpsAnalyseAPILatencyWorkers = 4
	require.ErrorContains(t, validateOpsAnalyseAPILatencyFlags(cmd), "--workers must be no greater than --count")

	flagOpsAnalyseAPILatencyCount = 4
	flagOpsAnalyseAPILatencyWorkers = 3
	require.ErrorContains(t, validateOpsAnalyseAPILatencyFlags(cmd), "count 4 is too small for worker stages 1, 2, 3")

	flagOpsAnalyseAPILatencyCount = 3
	flagOpsAnalyseAPILatencyWorkers = 1
	flagViewKeysOnly = true
	require.ErrorContains(t, validateOpsAnalyseAPILatencyFlags(cmd), "--keys-only is not supported")
}

// TestOpsAnalyseAPILatencyInvalidBudgetSkipsRemote proves Cobra validation exits before client work.
func TestOpsAnalyseAPILatencyInvalidBudgetSkipsRemote(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		http.Error(w, `{"message":"unexpected request"}`, http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsAnalyseAPILatencyInvalidBudgetSkipsRemoteHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
	})
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "count 4 is too small for worker stages 1, 2, 4")
	require.Empty(t, requests.Snapshot())
}

func TestOpsAnalyseAPILatencyInvalidBudgetSkipsRemoteHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"ops", "analyse", "api-latency",
		"--count", "4",
		"--workers", "4",
	})
	_, err := root.ExecuteC()
	require.NoError(t, err)
}

// TestOpsAnalyseAPILatencyRootArgsHelper runs root arguments in a subprocess for report error assertions.
func TestOpsAnalyseAPILatencyRootArgsHelper(t *testing.T) {
	executeRootHelperFromArgsEnv(t, "C8VOLT_TEST_ROOT_ARGS")
}

// TestOpsAnalyseAPILatencyRootArgsExitHelper runs root arguments and exits after success so subprocess stdout stays command-only.
func TestOpsAnalyseAPILatencyRootArgsExitHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_ROOT_ARGS")), &args))
	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs(nil)
	if os.Getenv("C8VOLT_TEST_CANCEL_CONTEXT") == "1" {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		root.SetContext(ctx)
	}
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = append([]string{"c8volt"}, args...)
	Execute()
	os.Exit(0)
}

// TestOpsAnalyseAPILatencyOutputSafetyExcludesProtectedContextAndRawBodies verifies command output and reports stay sanitized.
func TestOpsAnalyseAPILatencyOutputSafetyExcludesProtectedContextAndRawBodies(t *testing.T) {
	var requests testx.SafeSlice[string]
	var authHeaders testx.SafeSlice[string]
	tokenSrv := newAPILatencyOAuthTokenServer(t)
	t.Cleanup(tokenSrv.Close)
	srv := newOpsAnalyseAPILatencyOutputSafetyServer(t, &requests, &authHeaders)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "api-latency.json")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeAPILatencyOAuthConfig(t, srv.URL, tokenSrv.URL, "8.9"),
		"ops", "analyse", "api-latency",
		"--count", "1",
		"--workers", "1",
		"--report-file", reportPath,
		"--report-format", "json",
	)

	require.Empty(t, stdout)
	require.Contains(t, stderr, "finding: backpressure observed")
	report := readReportFile(t, reportPath)
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(report), &payload))
	require.Equal(t, "completed", payload["outcome"])
	requireNoAPILatencyProtectedMarkers(t, stderr+"\n"+report)
	require.NotContains(t, report, strings.Repeat("raw-upstream-detail-", 16))
	require.Contains(t, requests.Snapshot(), "POST /v2/process-definitions/search")
	require.NotEmpty(t, authHeaders.Snapshot())
	for _, header := range authHeaders.Snapshot() {
		require.Equal(t, "Bearer "+apiLatencyProtectedAccessToken, header)
	}
}

// TestOpsAnalyseAPILatencyQuietFailureKeepsErrorWithoutProgress verifies quiet mode suppresses progress but not failures.
func TestOpsAnalyseAPILatencyQuietFailureKeepsErrorWithoutProgress(t *testing.T) {
	output, err := testx.RunCmdSubprocess(t, "TestOpsAnalyseAPILatencyRootArgsHelper", map[string]string{
		"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, []string{
			"--quiet",
			"ops", "analyse", "api-latency",
			"--count", "4",
			"--workers", "4",
		}),
	})

	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	text := string(output)
	require.Contains(t, text, "count 4 is too small for worker stages 1, 2, 4")
	require.NotContains(t, text, "measuring read-only API latency")
}

// TestOpsAnalyseAPILatencyCompletedAbnormalSubprocessExitsSuccessfully verifies abnormal evidence is still a successful completed diagnostic.
func TestOpsAnalyseAPILatencyCompletedAbnormalSubprocessExitsSuccessfully(t *testing.T) {
	var requests testx.SafeSlice[string]
	var authHeaders testx.SafeSlice[string]
	tokenSrv := newAPILatencyOAuthTokenServer(t)
	t.Cleanup(tokenSrv.Close)
	srv := newOpsAnalyseAPILatencyOutputSafetyServer(t, &requests, &authHeaders)
	t.Cleanup(srv.Close)

	stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsAnalyseAPILatencyRootArgsExitHelper", map[string]string{
		"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, []string{
			"--config", writeAPILatencyOAuthConfig(t, srv.URL, tokenSrv.URL, "8.9"),
			"--json",
			"ops", "analyse", "api-latency",
			"--count", "1",
			"--workers", "1",
		}),
	})

	require.NoError(t, err)
	require.Empty(t, strings.TrimSpace(stderr))
	envelope := requireSingleJSONObjectDocument(t, stdout)
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "completed", payload["outcome"])
	findings := requireJSONItems(t, payload["findings"], 1)
	require.Equal(t, "backpressure_observed", requireJSONObject(t, findings[0])["code"])
	require.NotEmpty(t, authHeaders.Snapshot())
	requireNoAPILatencyProtectedMarkers(t, stdout+"\n"+stderr)
	require.Contains(t, requests.Snapshot(), "POST /v2/process-definitions/search")
}

// TestOpsAnalyseAPILatencyInvalidSubprocessUsesJSONErrorEnvelope verifies invalid local input exits nonzero with the shared envelope.
func TestOpsAnalyseAPILatencyInvalidSubprocessUsesJSONErrorEnvelope(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsAnalyseAPILatencyReadOnlyServer(t, &requests)
	t.Cleanup(srv.Close)

	stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsAnalyseAPILatencyRootArgsHelper", map[string]string{
		"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, []string{
			"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
			"--json",
			"ops", "analyse", "api-latency",
			"--count", "4",
			"--workers", "4",
		}),
	})

	requireAPILatencySubprocessExitCode(t, err, exitcode.InvalidArgs)
	require.Empty(t, strings.TrimSpace(stderr))
	envelope := requireSingleJSONObjectDocument(t, stdout)
	require.Equal(t, string(OutcomeInvalid), envelope["outcome"])
	require.Equal(t, "ops analyse api-latency", envelope["command"])
	require.Nil(t, envelope["payload"])
	detail := requireJSONObject(t, envelope["detail"])
	require.Equal(t, "invalid_input", detail["class"])
	require.Contains(t, detail["message"], "count 4 is too small for worker stages 1, 2, 4")
	require.Empty(t, requests.Snapshot())
}

// TestOpsAnalyseAPILatencyIncompleteSubprocessUsesJSONErrorEnvelope verifies interrupted measurement exits nonzero without a partial stdout payload.
func TestOpsAnalyseAPILatencyIncompleteSubprocessUsesJSONErrorEnvelope(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsAnalyseAPILatencyReadOnlyServer(t, &requests)
	t.Cleanup(srv.Close)

	stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsAnalyseAPILatencyRootArgsExitHelper", map[string]string{
		"C8VOLT_TEST_CANCEL_CONTEXT": "1",
		"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, []string{
			"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
			"--json",
			"ops", "analyse", "api-latency",
			"--count", "1",
			"--workers", "1",
		}),
	})

	requireAPILatencySubprocessExitCode(t, err, exitcode.Error)
	require.Empty(t, strings.TrimSpace(stderr))
	envelope := requireSingleJSONObjectDocument(t, stdout)
	require.Equal(t, string(OutcomeFailed), envelope["outcome"])
	require.Equal(t, "ops analyse api-latency", envelope["command"])
	require.Nil(t, envelope["payload"])
	detail := requireJSONObject(t, envelope["detail"])
	require.Contains(t, detail["message"], "ops analyse api-latency")
	require.Contains(t, detail["message"], "context canceled")
	require.Empty(t, requests.Snapshot())
}

// TestOpsAnalyseAPILatencyReportFailureSubprocessUsesJSONErrorEnvelope verifies report write failures keep the shared machine error shape.
func TestOpsAnalyseAPILatencyReportFailureSubprocessUsesJSONErrorEnvelope(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsAnalyseAPILatencyReadOnlyServer(t, &requests)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "missing", "api-latency.md")

	stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsAnalyseAPILatencyRootArgsHelper", map[string]string{
		"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, []string{
			"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
			"--json",
			"ops", "analyse", "api-latency",
			"--count", "1",
			"--workers", "1",
			"--report-file", reportPath,
		}),
	})

	requireAPILatencySubprocessExitCode(t, err, exitcode.Error)
	require.Empty(t, strings.TrimSpace(stderr))
	envelope := requireSingleJSONObjectDocument(t, stdout)
	require.Equal(t, string(OutcomeFailed), envelope["outcome"])
	require.Equal(t, "ops analyse api-latency", envelope["command"])
	require.Nil(t, envelope["payload"])
	detail := requireJSONObject(t, envelope["detail"])
	require.Contains(t, detail["message"], "write ops analyse api-latency report")
	require.NoFileExists(t, reportPath)
	requireOpsAnalyseAPILatencyReadOnlyRequests(t, requests.Snapshot())
}

// TestOpsAnalyseAPILatencyReadOnlyCommandRendersHuman verifies terminal output and zero mutation against a fixture server.
func TestOpsAnalyseAPILatencyReadOnlyCommandRendersHuman(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsAnalyseAPILatencyReadOnlyServer(t, &requests)
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"ops", "analyse", "api-latency",
		"--count", "1",
		"--workers", "1",
	)

	require.Empty(t, stdout)
	assertOpsAnalyseAPILatencyHumanOutput(t, stderr)
	requireOpsAnalyseAPILatencyReadOnlyRequests(t, requests.Snapshot())
}

// TestOpsAnalyseAPILatencyJSONUsesSingleEnvelope keeps machine output one document with no progress leakage.
func TestOpsAnalyseAPILatencyJSONUsesSingleEnvelope(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsAnalyseAPILatencyReadOnlyServer(t, &requests)
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"--json",
		"ops", "analyse", "api-latency",
		"-n", "1",
		"-w", "1",
	)

	require.Empty(t, stderr)
	envelope := requireSingleJSONObjectDocument(t, stdout)
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "ops analyse api-latency", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, ops.APILatencySchemaVersion, payload["schemaVersion"])
	require.Equal(t, "completed", payload["outcome"])
	stages := requireJSONItems(t, payload["stages"], 1)
	stage := requireJSONObject(t, stages[0])
	require.Equal(t, "completed", stage["status"])
	require.NotContains(t, stdout, "measuring read-only API latency")
	requireOpsAnalyseAPILatencyReadOnlyRequests(t, requests.Snapshot())
}

// TestOpsAnalyseAPILatencyJSONRecordsConfiguredTenant verifies safe tenant context is preserved in output.
func TestOpsAnalyseAPILatencyJSONRecordsConfiguredTenant(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsAnalyseAPILatencyReadOnlyServer(t, &requests)
	t.Cleanup(srv.Close)
	cfgPath := writeRawTestConfig(t, `
app:
  camunda_version: "8.9"
  tenant: "tenant-a"
auth:
  mode: none
apis:
  camunda_api:
    base_url: "`+srv.URL+`"
http:
  timeout: "10s"
`)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--json",
		"ops", "analyse", "api-latency",
		"-n", "1",
		"-w", "1",
	)

	require.Empty(t, stderr)
	payload := requireJSONObject(t, requireSingleJSONObjectDocument(t, stdout)["payload"])
	contextPayload := requireJSONObject(t, payload["context"])
	require.Equal(t, "tenant-a", contextPayload["tenant"])
	requestPayload := requireJSONObject(t, payload["request"])
	require.Equal(t, "tenant-a", requestPayload["tenantId"])
	requireOpsAnalyseAPILatencyReadOnlyRequests(t, requests.Snapshot())
}

// TestAttachOpsAPILatencyResultContextPreservesSafeEvidence verifies command-owned context enrichment stays bounded.
func TestAttachOpsAPILatencyResultContextPreservesSafeEvidence(t *testing.T) {
	cfg := testAPILatencyConfig()
	cfg.ActiveProfile = "support"
	cfg.App.Tenant = "tenant-a"
	cfg.App.CamundaVersion = "8.9"
	result := ops.APILatencyResult{
		Request: ops.APILatencyRequest{
			CommandName: "ops analyse api-latency",
			Mode:        ops.APILatencyModeReadOnly,
		},
		Plan: ops.APILatencyPlan{
			Mode:        ops.APILatencyModeReadOnly,
			Notices:     []string{"direct keyed reads reuse keys returned by measured searches when available and supported"},
			Limitations: []string{"read-only evidence cannot prove write-path health"},
		},
		Notices:     []string{"direct keyed reads reuse keys returned by measured searches when available and supported"},
		Limitations: []string{"a bounded sample cannot prove overall cluster health or capacity"},
	}

	got := attachOpsAPILatencyResultContext(cfg, result, opsAnalyseAPILatencyCommandName)

	require.Equal(t, ops.APILatencySchemaVersion, got.SchemaVersion)
	require.Equal(t, ops.APILatencySchemaVersion, got.Context.SchemaVersion)
	require.Equal(t, "ops analyse api-latency", got.Context.CommandName)
	require.Equal(t, CurrentBuildInfo().Version, got.Context.C8voltVersion)
	require.Equal(t, "8.9", got.Context.CamundaVersion)
	require.Equal(t, "support", got.Context.Profile)
	require.Equal(t, "tenant-a", got.Context.Tenant)
	require.Equal(t, []string{"direct keyed reads reuse keys returned by measured searches when available and supported"}, got.Notices)
	require.Equal(t, []string{
		"a bounded sample cannot prove overall cluster health or capacity",
		"read-only evidence cannot prove write-path health",
	}, got.Limitations)

	active := ops.APILatencyResult{
		Request: ops.APILatencyRequest{Mode: ops.APILatencyModeActive},
		Plan: ops.APILatencyPlan{
			Mode:        ops.APILatencyModeActive,
			Notices:     []string{"primary samples bound process-instance creates; derived reads and visibility checks are disclosed separately"},
			Limitations: []string{"active evidence is bounded by the previewed worker, primary-sample, and visibility-attempt ceilings"},
		},
	}

	got = attachOpsAPILatencyResultContext(cfg, active, opsExecuteAPILatencyCommandName)

	require.Equal(t, "ops execute api-latency-test", got.Context.CommandName)
	require.Equal(t, []string{"primary samples bound process-instance creates; derived reads and visibility checks are disclosed separately"}, got.Notices)
	require.Equal(t, []string{"active evidence is bounded by the previewed worker, primary-sample, and visibility-attempt ceilings"}, got.Limitations)
}

// TestOpsAnalyseAPILatencyWritesInferredMarkdownReport verifies read-only reports use shared path inference and permissions.
func TestOpsAnalyseAPILatencyWritesInferredMarkdownReport(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsAnalyseAPILatencyReadOnlyServer(t, &requests)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "api-latency")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"ops", "analyse", "api-latency",
		"--count", "1",
		"--workers", "1",
		"--report-file", reportPath,
	)

	require.Empty(t, stdout)
	require.Contains(t, stderr, "report: written "+reportPath)
	require.Less(t, strings.Index(stderr, "report: written "+reportPath), strings.Index(stderr, "outcome: completed"))
	info, err := os.Stat(reportPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	report := readReportFile(t, reportPath)
	require.Contains(t, report, "# Analyse API Latency Report")
	require.Contains(t, report, "- Command: ops analyse api-latency")
	require.Contains(t, report, "- Mode: read_only")
	require.Contains(t, report, "- Primary Sample Limit: 1")
	require.Contains(t, report, "- Stage 1: primary 3/3; derived 2/2; errors 0; timeouts 0; unavailable 0")
	require.Contains(t, report, "- no_abnormal_evidence: no abnormal evidence; confidence low")
	require.Contains(t, report, "- Outcome: completed")
	require.NotContains(t, report, "Ownership")
	require.NotContains(t, report, "Authorization")
	requireOpsAnalyseAPILatencyReadOnlyRequests(t, requests.Snapshot())
}

// TestOpsAnalyseAPILatencyWritesRawJSONReportWithExplicitOverride verifies report JSON is not the stdout envelope.
func TestOpsAnalyseAPILatencyWritesRawJSONReportWithExplicitOverride(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsAnalyseAPILatencyReadOnlyServer(t, &requests)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "api-latency.md")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"--json",
		"ops", "analyse", "api-latency",
		"--count", "1",
		"--workers", "1",
		"--report-file", reportPath,
		"--report-format", "json",
	)

	require.Empty(t, stderr)
	envelope := requireSingleJSONObjectDocument(t, stdout)
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, ops.APILatencySchemaVersion, report["schemaVersion"])
	require.Equal(t, "completed", report["outcome"])
	require.NotContains(t, report, "payload")
	require.NotContains(t, report, "command")
	require.Equal(t, "ops analyse api-latency", requireJSONObject(t, report["context"])["commandName"])
	require.Equal(t, "json", requireJSONObject(t, report["request"])["reportFormat"])
	requireOpsAnalyseAPILatencyReadOnlyRequests(t, requests.Snapshot())
}

// TestOpsAnalyseAPILatencyReportValidationAndWriteFailures verifies dependent flags, preservation, and destination errors.
func TestOpsAnalyseAPILatencyReportValidationAndWriteFailures(t *testing.T) {
	cmd := resetOpsAnalyseAPILatencyTestFlags(t)
	flagOpsAnalyseAPILatencyReportFormat = "json"
	require.ErrorContains(t, validateOpsAnalyseAPILatencyFlags(cmd), "--report-format requires --report-file")

	var requests testx.SafeSlice[string]
	srv := newOpsAnalyseAPILatencyReadOnlyServer(t, &requests)
	t.Cleanup(srv.Close)
	existingPath := filepath.Join(t.TempDir(), "api-latency.md")
	const existingReport = "existing report"
	require.NoError(t, os.WriteFile(existingPath, []byte(existingReport), 0o600))

	output, err := testx.RunCmdSubprocess(t, "TestOpsAnalyseAPILatencyRootArgsHelper", map[string]string{
		"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, []string{
			"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
			"ops", "analyse", "api-latency",
			"--count", "1",
			"--workers", "1",
			"--report-file", existingPath,
		}),
	})
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "report file already exists: "+existingPath)
	require.Equal(t, existingReport, readReportFile(t, existingPath))
	require.Empty(t, requests.Snapshot())

	missingParentPath := filepath.Join(t.TempDir(), "missing", "api-latency.md")
	output, err = testx.RunCmdSubprocess(t, "TestOpsAnalyseAPILatencyRootArgsHelper", map[string]string{
		"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, []string{
			"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
			"ops", "analyse", "api-latency",
			"--count", "1",
			"--workers", "1",
			"--report-file", missingParentPath,
		}),
	})
	require.Error(t, err)
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "write ops analyse api-latency report")
	require.NoFileExists(t, missingParentPath)
}

// TestRenderOpsAnalyseAPILatencyStableHumanAndJSON pins read-only renderer ordering, safe context, and active-field omission.
func TestRenderOpsAnalyseAPILatencyStableHumanAndJSON(t *testing.T) {
	resetOpsAnalyseAPILatencyTestFlags(t)
	p50 := 8 * time.Millisecond
	p95 := 15 * time.Millisecond
	maxLatency := 22 * time.Millisecond
	nextP95 := 45 * time.Millisecond
	throughput := 27.5
	p50Delta := 4 * time.Millisecond
	throughputDelta := -3.25
	throughputDeltaPercent := -11.8
	result := ops.APILatencyResult{
		SchemaVersion: ops.APILatencySchemaVersion,
		Context: ops.APILatencyRunContext{
			CommandName:    "ops analyse api-latency",
			SchemaVersion:  ops.APILatencySchemaVersion,
			C8voltVersion:  "dev-test",
			CamundaVersion: "8.9",
			Profile:        "support",
			Tenant:         "tenant-a",
			Duration:       "125ms",
		},
		Request: ops.APILatencyRequest{
			CommandName: "ops analyse api-latency",
			Mode:        ops.APILatencyModeReadOnly,
			Count:       7,
			Workers:     4,
			TenantID:    "tenant-a",
			OutputMode:  "one-line",
		},
		Plan: ops.APILatencyPlan{
			Mode:                    ops.APILatencyModeReadOnly,
			PrimarySampleLimit:      7,
			PrimarySampleAllocation: 7,
			DerivedRequestLimit:     14,
			Stages: []ops.APILatencyStagePlan{
				{Index: 1, WorkerCount: 1, PrimarySamples: 1, DerivedRequestLimit: 2},
				{Index: 2, WorkerCount: 2, PrimarySamples: 2, DerivedRequestLimit: 4},
				{Index: 3, WorkerCount: 4, PrimarySamples: 4, DerivedRequestLimit: 8},
			},
		},
		Topology: ops.APILatencyTopologyEvidence{
			BrokerCount:          2,
			PartitionCount:       3,
			UnhealthyPartitions:  []int{2},
			LeaderlessPartitions: []int{3},
			HealthKnown:          true,
		},
		Stages: []ops.APILatencyStageResult{
			{
				Plan:                 ops.APILatencyStagePlan{Index: 1, WorkerCount: 1, PrimarySamples: 1, DerivedRequestLimit: 2},
				Status:               ops.APILatencyStageStatusCompleted,
				ActualMaxConcurrency: 1,
				PrimaryAttempts:      3,
				DerivedAttempts:      2,
				Categories: []ops.APILatencyCategorySummary{
					{Category: ops.APILatencyCategoryTopologyRead, Attempts: 1, Successes: 1, ThroughputPerSecond: &throughput, P50: &p50, P95: &p95, Max: &maxLatency},
					{Category: ops.APILatencyCategoryProcessDefinitionSearch, Attempts: 1, Successes: 0, Timeouts: 1},
				},
				Classifications: []ops.APILatencyClassificationCount{
					{Classification: ops.APILatencyClassificationSuccess, Count: 1},
					{Classification: ops.APILatencyClassificationTimeout, Count: 1},
				},
			},
			{
				Plan:                 ops.APILatencyStagePlan{Index: 2, WorkerCount: 2, PrimarySamples: 2, DerivedRequestLimit: 4},
				Status:               ops.APILatencyStageStatusCompleted,
				ActualMaxConcurrency: 2,
				PrimaryAttempts:      6,
				DerivedAttempts:      4,
				Categories: []ops.APILatencyCategorySummary{
					{Category: ops.APILatencyCategoryTopologyRead, Attempts: 2, Successes: 2, ThroughputPerSecond: &throughput, P50: &p50, P95: &nextP95, Max: &nextP95},
					{Category: ops.APILatencyCategoryProcessInstanceRead, Attempts: 2, Successes: 0, Unavailable: 2},
				},
				Classifications: []ops.APILatencyClassificationCount{
					{Classification: ops.APILatencyClassificationSuccess, Count: 2},
					{Classification: ops.APILatencyClassificationUnavailable, Count: 2},
				},
				Comparison: &ops.APILatencyStageComparison{Categories: []ops.APILatencyCategoryComparison{{
					Category:               ops.APILatencyCategoryTopologyRead,
					P50Delta:               &p50Delta,
					ThroughputDelta:        &throughputDelta,
					ThroughputDeltaPercent: &throughputDeltaPercent,
					ComparisonSampleCount:  2,
				}}},
			},
		},
		Findings: []ops.APILatencyFinding{
			{Code: "timeout_evidence", LikelyArea: "gateway/connectivity/authentication", Confidence: ops.APILatencyFindingConfidenceHigh, NextInvestigation: "inspect gateway timeout logs"},
			{Code: "no_abnormal_evidence", LikelyArea: "no abnormal evidence", Confidence: ops.APILatencyFindingConfidenceLow, NextInvestigation: "rerun with active test if write symptoms continue"},
		},
		Notices:     []string{"logical samples include existing client retry behavior"},
		Limitations: []string{"read-only evidence cannot prove write-path health"},
		Outcome:     ops.APILatencyOutcomeCompleted,
	}

	humanCmd := &cobra.Command{}
	var humanOut bytes.Buffer
	humanCmd.SetOut(&humanOut)
	started := time.Now()
	require.NoError(t, renderOpsAPILatencyResult(humanCmd, result))
	require.Less(t, time.Since(started), 5*time.Second)
	human := humanOut.String()
	require.Contains(t, human, "analyse api latency")
	require.Contains(t, human, "context: c8volt dev-test; profile support; tenant tenant-a; camunda 8.9")
	require.Contains(t, human, "scope: read only; 7 measurement cycles; load increased from 1 to 4 workers")
	require.Contains(t, human, "topology: brokers 2; partitions 3; unhealthy 2; leaderless 3")
	require.Contains(t, human, "result: API measurements completed; request errors 0; timeouts 1")
	require.Contains(t, human, "latency: topology read was slowest; 95% completed within 45ms at 2 workers")
	require.Contains(t, human, "load effect: median topology read was unchanged from 1 to 2 workers")
	require.Less(t, strings.Index(human, "finding: timeout evidence"), strings.Index(human, "finding: no abnormal evidence found"))
	require.Contains(t, human, "next: inspect gateway timeout logs")
	require.Contains(t, human, "outcome: completed with 1 finding; elapsed 125ms")
	require.NotContains(t, human, "request: count")
	require.NotContains(t, human, "stage 1:")
	require.NotContains(t, human, "p50")
	require.NotContains(t, human, "p95")
	require.NotContains(t, human, "notice:")
	require.NotContains(t, human, "limitation:")
	require.NotContains(t, human, "ownership:")
	require.NotContains(t, human, "visibility:")
	require.NotContains(t, human, "cleanup:")
	require.NotContains(t, human, "GET /v2")
	require.NotContains(t, human, "Authorization")

	flagVerbose = true
	verboseCmd := &cobra.Command{}
	var verboseOut bytes.Buffer
	verboseCmd.SetOut(&verboseOut)
	require.NoError(t, renderOpsAPILatencyResult(verboseCmd, result))
	require.Contains(t, verboseOut.String(), "request: count 7; workers 1,2,4; stages 3; derived requests <= 14")
	require.Contains(t, verboseOut.String(), "stage 1: workers 1; primary 3/3; derived 2/2")
	require.Contains(t, verboseOut.String(), "finding detail: likely area gateway/connectivity/authentication; confidence high")
	require.Contains(t, verboseOut.String(), "notice: logical samples include existing client retry behavior")
	require.Contains(t, verboseOut.String(), "limitation: read-only evidence cannot prove write-path health")
	flagVerbose = false

	root := Root()
	resetCommandTreeFlags(root)
	jsonCmd, _, err := root.Find([]string{"ops", "analyse", "api-latency"})
	require.NoError(t, err)
	var jsonOut bytes.Buffer
	jsonCmd.SetOut(&jsonOut)
	flagViewAsJson = true
	started = time.Now()
	require.NoError(t, renderOpsAPILatencyResult(jsonCmd, result))
	require.Less(t, time.Since(started), 5*time.Second)
	envelope := requireSingleJSONObjectDocument(t, jsonOut.String())
	require.Equal(t, "succeeded", envelope["outcome"])
	require.Equal(t, "ops analyse api-latency", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, ops.APILatencySchemaVersion, payload["schemaVersion"])
	require.NotContains(t, payload, "ownership")
	require.NotContains(t, payload, "visibility")
	require.NotContains(t, payload, "cleanup")
	contextPayload := requireJSONObject(t, payload["context"])
	require.Equal(t, "ops analyse api-latency", contextPayload["commandName"])
	require.Equal(t, "dev-test", contextPayload["c8voltVersion"])
	require.Equal(t, "tenant-a", contextPayload["tenant"])
	plan := requireJSONObject(t, payload["plan"])
	plannedStages := requireJSONItems(t, plan["stages"], 3)
	require.Equal(t, float64(1), requireJSONObject(t, plannedStages[0])["workerCount"])
	require.Equal(t, float64(2), requireJSONObject(t, plannedStages[1])["workerCount"])
	require.Equal(t, float64(4), requireJSONObject(t, plannedStages[2])["workerCount"])
	stages := requireJSONItems(t, payload["stages"], 2)
	firstStage := requireJSONObject(t, stages[0])
	require.Equal(t, "completed", firstStage["status"])
	classifications := requireJSONItems(t, firstStage["classifications"], 2)
	require.Equal(t, "success", requireJSONObject(t, classifications[0])["classification"])
	require.Equal(t, "timeout", requireJSONObject(t, classifications[1])["classification"])
	categories := requireJSONItems(t, firstStage["categories"], 2)
	require.Equal(t, "topology_read", requireJSONObject(t, categories[0])["category"])
	require.Equal(t, "process_definition_search", requireJSONObject(t, categories[1])["category"])
	findings := requireJSONItems(t, payload["findings"], 2)
	require.Equal(t, "timeout_evidence", requireJSONObject(t, findings[0])["code"])
	require.Equal(t, "no_abnormal_evidence", requireJSONObject(t, findings[1])["code"])
}

// TestOpsAnalyseAPILatencyProgressModeGate verifies aggregate progress stays out of protected modes.
func TestOpsAnalyseAPILatencyProgressModeGate(t *testing.T) {
	for _, tc := range []struct {
		name       string
		setup      func()
		wantStderr string
	}{
		{name: "human", setup: func() {}, wantStderr: ""},
		{name: "json", setup: func() { flagViewAsJson = true }, wantStderr: ""},
		{name: "quiet", setup: func() { flagQuiet = true }, wantStderr: ""},
		{name: "automation", setup: func() { flagCmdAutomation = true }, wantStderr: ""},
		{name: "verbose", setup: func() { flagVerbose = true }, wantStderr: "measuring read-only API latency, 1/3 sample-cycle(s)"},
		{name: "debug", setup: func() { flagDebug = true }, wantStderr: "measuring read-only API latency, 1/3 sample-cycle(s)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetOpsAnalyseAPILatencyTestFlags(t)
			tc.setup()
			cmd, stderr := newSemanticProgressStderrCommand()
			request := ops.APILatencyRequest{}
			progress := configureOpsAPILatencyProgress(cmd, &request)
			defer progress.Close()

			request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindFrozenScope, FrozenScope: &ops.FrozenScopeProgress{
				Phase:        "measuring read-only API latency",
				CoreResource: "sample-cycle(s)",
				Done:         1,
				Total:        3,
			}})

			if tc.wantStderr == "" {
				require.Empty(t, stderr.String())
				return
			}
			require.Contains(t, stderr.String(), tc.wantStderr)
		})
	}
}

// TestOpsAnalyseAPILatencyProgressUsesActivityForHuman keeps default progress transient and off output streams.
func TestOpsAnalyseAPILatencyProgressUsesActivityForHuman(t *testing.T) {
	resetOpsAnalyseAPILatencyTestFlags(t)
	cmd := &cobra.Command{}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	sink := &activitysink.Sink{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))
	request := ops.APILatencyRequest{}

	progress := configureOpsAPILatencyProgress(cmd, &request)
	defer progress.Close()
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindFrozenScope, FrozenScope: &ops.FrozenScopeProgress{
		Phase:        "measuring read-only API latency",
		CoreResource: "sample-cycle(s)",
		Done:         2,
		Total:        3,
	}})

	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
	require.Equal(t, []activitysink.Update{{
		Message:    "measuring read-only API latency, 2/3 sample-cycle(s)",
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.PriorityUpdates())
}

func assertOpsAnalyseAPILatencyHumanOutput(t *testing.T, output string) {
	t.Helper()

	require.Contains(t, output, "analyse api latency")
	require.Contains(t, output, "scope: read only; 1 measurement cycle; load 1 worker")
	require.Contains(t, output, "topology: brokers 1; partitions 1")
	require.Contains(t, output, "result: API measurements completed; request errors 0; timeouts 0")
	require.Contains(t, output, "latency:")
	require.Contains(t, output, "95% completed within")
	require.Contains(t, output, "finding: no abnormal evidence found")
	require.Contains(t, output, "outcome: completed")
	require.NotContains(t, output, "request: count")
	require.NotContains(t, output, "stage 1:")
	require.NotContains(t, output, "p50")
	require.NotContains(t, output, "p95")
	require.NotContains(t, output, "limitation:")
	require.NotContains(t, output, "GET /v2")
	require.NotContains(t, output, "process-definition-key")
	require.NotContains(t, output, "process-instance-key")
}

func TestFormatOpsAPILatencyOperatorDurationUsesReadablePrecision(t *testing.T) {
	require.Equal(t, "14ms", formatOpsAPILatencyOperatorDuration(13*time.Millisecond+889*time.Microsecond))
	require.Equal(t, "3.1s", formatOpsAPILatencyOperatorDuration(3*time.Second+83*time.Millisecond))
}

func newOpsAnalyseAPILatencyReadOnlyServer(t *testing.T, requests *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/topology":
			_, _ = w.Write([]byte(singleBrokerClusterTopologyFixtureJSON()))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
			requireOpsAnalyseAPILatencySearchBody(t, r)
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionKey":"9001","processDefinitionId":"demo","name":"demo","version":3,"tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-definitions/9001":
			_, _ = w.Write([]byte(`{"processDefinitionKey":"9001","processDefinitionId":"demo","name":"demo","version":3,"tenantId":"tenant"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			requireOpsAnalyseAPILatencySearchBody(t, r)
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"1001","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/1001":
			_, _ = w.Write([]byte(`{"processInstanceKey":"1001","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		default:
			if r.Method == http.MethodPost || r.Method == http.MethodDelete || strings.Contains(r.URL.Path, "/cancellation") {
				t.Fatalf("unexpected mutation request: %s %s", r.Method, r.URL.Path)
			}
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

// newOpsAnalyseAPILatencyOutputSafetyServer injects protected markers into ignored or classified response data.
func newOpsAnalyseAPILatencyOutputSafetyServer(t *testing.T, requests *testx.SafeSlice[string], authHeaders *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		authHeaders.Append(r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/topology":
			_, _ = w.Write([]byte(singleBrokerClusterTopologyFixtureJSON()))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
			requireOpsAnalyseAPILatencySearchBody(t, r)
			http.Error(w, apiLatencyRawProtectedFailureBody(), http.StatusServiceUnavailable)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			requireOpsAnalyseAPILatencySearchBody(t, r)
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"1001","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant","variables":{"apiToken":"` + apiLatencyProtectedVariable + `"},"businessPayload":"` + apiLatencyProtectedPayload + `"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/1001":
			_, _ = w.Write([]byte(`{"processInstanceKey":"1001","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant","variables":{"secret":"` + apiLatencyProtectedVariable + `"},"businessPayload":"` + apiLatencyProtectedPayload + `"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func requireOpsAnalyseAPILatencySearchBody(t *testing.T, r *http.Request) {
	t.Helper()

	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	page := requireJSONObject(t, payload["page"])
	require.Equal(t, float64(1), page["limit"])
}

// newAPILatencyOAuthTokenServer proves the command can authenticate without leaking token material.
func newAPILatencyOAuthTokenServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "client_credentials", r.Form.Get("grant_type"))
		require.Equal(t, apiLatencyProtectedClientSecret, r.Form.Get("client_secret"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"` + apiLatencyProtectedAccessToken + `","token_type":"Bearer"}`))
	}))
}

// writeAPILatencyOAuthConfig builds an authenticated test profile containing protected config values.
func writeAPILatencyOAuthConfig(t *testing.T, apiURL string, tokenURL string, camundaVersion string) string {
	t.Helper()
	return writeRawTestConfig(t, `
app:
  camunda_version: "`+camundaVersion+`"
  tenant: "tenant-a"
auth:
  mode: oauth2
  oauth2:
    token_url: "`+tokenURL+`"
    client_id: api-latency-client
    client_secret: `+apiLatencyProtectedClientSecret+`
    scopes:
      camunda_api: api-latency
apis:
  camunda_api:
    base_url: "`+apiURL+`"
    require_scope: true
http:
  timeout: "10s"
`)
}

// apiLatencyRawProtectedFailureBody simulates an upstream body that must be reduced to safe classifications.
func apiLatencyRawProtectedFailureBody() string {
	return `{"error":"RESOURCE_EXHAUSTED","message":"` + apiLatencyProtectedAuthorizationHeader + ` ` + apiLatencyProtectedClientSecret + ` ` + apiLatencyProtectedVariable + ` ` + apiLatencyProtectedPayload + ` ` + apiLatencyProtectedRawBody + ` ` + strings.Repeat("raw-upstream-detail-", 64) + `"}`
}

// requireNoAPILatencyProtectedMarkers centralizes the API-latency output safety assertions.
func requireNoAPILatencyProtectedMarkers(t *testing.T, output string) {
	t.Helper()

	for _, marker := range []string{
		apiLatencyProtectedAccessToken,
		apiLatencyProtectedClientSecret,
		apiLatencyProtectedVariable,
		apiLatencyProtectedPayload,
		apiLatencyProtectedAuthorizationHeader,
		apiLatencyProtectedRawBody,
	} {
		require.NotContains(t, output, marker)
	}
	require.NotContains(t, output, "variables")
	require.NotContains(t, output, "businessPayload")
	require.NotContains(t, output, "client_secret")
}

// requireAPILatencySubprocessExitCode keeps subprocess exit assertions consistent across both API-latency leaves.
func requireAPILatencySubprocessExitCode(t *testing.T, err error, want int) {
	t.Helper()
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, want, exitErr.ExitCode())
}

func requireOpsAnalyseAPILatencyReadOnlyRequests(t *testing.T, requests []string) {
	t.Helper()

	require.Contains(t, requests, "GET /v2/topology")
	require.Contains(t, requests, "POST /v2/process-definitions/search")
	require.Contains(t, requests, "GET /v2/process-definitions/9001")
	require.Contains(t, requests, "POST /v2/process-instances/search")
	require.Contains(t, requests, "GET /v2/process-instances/1001")
	for _, request := range requests {
		require.NotContains(t, request, "/process-instances/1001/cancellation")
		require.NotContains(t, request, "/deployments")
		require.NotContains(t, request, "DELETE ")
	}
}

func resetOpsAnalyseAPILatencyTestFlags(t *testing.T) *cobra.Command {
	t.Helper()
	resetSemanticProgressModeFlags(t)
	prevCount := flagOpsAnalyseAPILatencyCount
	prevWorkers := flagOpsAnalyseAPILatencyWorkers
	prevReportFile := flagOpsAnalyseAPILatencyReportFile
	prevReportFormat := flagOpsAnalyseAPILatencyReportFormat
	t.Cleanup(func() {
		flagOpsAnalyseAPILatencyCount = prevCount
		flagOpsAnalyseAPILatencyWorkers = prevWorkers
		flagOpsAnalyseAPILatencyReportFile = prevReportFile
		flagOpsAnalyseAPILatencyReportFormat = prevReportFormat
	})
	flagOpsAnalyseAPILatencyCount = opsAnalyseAPILatencyDefaultCount
	flagOpsAnalyseAPILatencyWorkers = opsAnalyseAPILatencyDefaultWorkers
	flagOpsAnalyseAPILatencyReportFile = ""
	flagOpsAnalyseAPILatencyReportFormat = ""
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToContext(context.Background(), logging.New(logging.LoggerConfig{
		Format: "plain-time",
		Writer: io.Discard,
	})))
	return cmd
}

func testAPILatencyConfig() *config.Config {
	cfg := config.New()
	cfg.HTTP.Timeout = "10s"
	return cfg
}
