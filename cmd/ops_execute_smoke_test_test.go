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
	require.Contains(t, output, "camunda version: 8.8")
	require.Contains(t, output, "fixture: embedded/processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn (available)")
	require.Contains(t, output, "connectivity: confirmed - cluster topology retrieved")
	require.Contains(t, output, "deployment: planned - deploy selected fixture")
	require.Contains(t, output, "run: planned - start 2 process instance(s)")
	require.Contains(t, output, "cleanup: skipped - retain created resources because --no-cleanup is set")
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

func TestOpsExecuteSmokeTestDeploysFixtureAndRendersDeploymentOutput(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsExecuteSmokeTestDeploymentServer(t, &requests)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "smoke-test",
	)

	require.Contains(t, output, "execute smoke test")
	require.Contains(t, output, "camunda version: 8.8")
	require.Contains(t, output, "fixture: embedded/processdefinitions/C88_MultipleSubProcessesParentProcess.bpmn (available)")
	require.Contains(t, output, "deployment: confirmed - deploy selected fixture")
	require.Contains(t, output, "deployment result: confirmed")
	require.Contains(t, output, "process definition: pd-88 (C88_MultipleSubProcessesParentProcess, version 4)")
	require.Contains(t, output, "tenant: <default>")
	require.Contains(t, output, "run result: confirmed")
	require.Contains(t, output, "created process instances: 1/1")
	require.Contains(t, output, "created keys: pi-1")
	require.Contains(t, output, "walk result: confirmed")
	require.Contains(t, output, "walk pi-1: confirmed, root pi-1, family 1")
	require.Contains(t, output, "outcome: passed_cleanup_skipped")
	require.Equal(t, []string{
		"GET /v2/topology",
		"POST /v2/deployments",
		"GET /v2/process-definitions/pd-88",
		"POST /v2/process-instances",
		"GET /v2/process-instances/pi-1",
		"GET /v2/process-instances/pi-1",
		"GET /v2/process-instances/pi-1",
		"POST /v2/process-instances/search",
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
			}, tt.args...)
			output := executeRootForProcessInstanceTest(t, args...)

			require.Contains(t, output, "run: confirmed - start 2 process instance(s)")
			require.Contains(t, output, "created process instances: 2/2")
			require.Contains(t, output, "created keys: pi-1, pi-2")
			require.Contains(t, output, "walk result: confirmed")
			require.Contains(t, output, "walk pi-1: confirmed, root pi-1, family 1")
			require.Contains(t, output, "walk pi-2: confirmed, root pi-2, family 1")
			require.Len(t, createBodies.Snapshot(), 2)
			for _, body := range createBodies.Snapshot() {
				require.Contains(t, body, `"processDefinitionKey":"pd-88"`)
				require.NotContains(t, body, `"processDefinitionId"`)
			}
		})
	}
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
			key := fmt.Sprintf("pi-%d", created)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(processInstanceCreationJSON(key)))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			requests.Append(r.Method + " " + r.URL.Path)
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(processInstanceJSON(key)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
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
