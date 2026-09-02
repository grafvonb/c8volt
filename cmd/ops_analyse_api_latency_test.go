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
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestOpsAnalyseAPILatencyHelpDocumentsReadOnlySurface verifies the diagnostic command is discoverable.
func TestOpsAnalyseAPILatencyHelpDocumentsReadOnlySurface(t *testing.T) {
	output := executeRootForTest(t, "ops", "analyse", "api-latency", "--help")

	assertHelpOutputContainsAll(t, output,
		"Analyse API latency without changing cluster state",
		"The command is read-only.",
		"--count is the total primary sample-cycle budget",
		"--workers is the maximum closed-loop worker count",
		"JSON output uses the shared command envelope",
		"Keys-only output is not meaningful",
		"-n, --count int",
		"-w, --workers int",
		"--report-file string",
		"--report-format string",
		"./c8volt ops analyse api-latency --count 20 --workers 4",
		"./c8volt ops analyse api-latency -n 6 -w 2",
	)
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
	require.Contains(t, output, "request: count 1; workers 1; stages 1; derived requests <= 2")
	require.Contains(t, output, "topology: brokers 1; partitions 1")
	require.Contains(t, output, "stage 1: workers 1; primary 3/3; derived 2/2; errors 0; timeouts 0; unavailable 0")
	require.Contains(t, output, "finding: no_abnormal_evidence; no abnormal evidence; confidence low")
	require.Contains(t, output, "limitation: read-only evidence cannot prove write-path health")
	require.Contains(t, output, "limitation: read-only evidence cannot prove exporter health or end-to-end process execution health")
	require.Contains(t, output, "outcome: completed")
	require.NotContains(t, output, "GET /v2")
	require.NotContains(t, output, "process-definition-key")
	require.NotContains(t, output, "process-instance-key")
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

func requireOpsAnalyseAPILatencySearchBody(t *testing.T, r *http.Request) {
	t.Helper()

	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	page := requireJSONObject(t, payload["page"])
	require.Equal(t, float64(1), page["limit"])
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
