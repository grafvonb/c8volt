// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
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

func apiLatencyProcessInstanceCreationJSON(key string) string {
	return fmt.Sprintf(`{
		"processInstanceKey": %q,
		"processDefinitionId": "C88_SimpleUserTask",
		"processDefinitionKey": "pd-88",
		"processDefinitionVersion": 1,
		"tenantId": "<default>"
	}`, key)
}

func apiLatencyProcessInstanceJSON(key string) string {
	return fmt.Sprintf(`{
		"hasIncident": false,
		"processDefinitionId": "C88_SimpleUserTask",
		"processDefinitionKey": "pd-88",
		"processDefinitionName": "C88_SimpleUserTask",
		"processDefinitionVersion": 1,
		"processInstanceKey": %q,
		"startDate": "2026-09-02T08:00:00Z",
		"state": "ACTIVE",
		"tenantId": "<default>"
	}`, key)
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
