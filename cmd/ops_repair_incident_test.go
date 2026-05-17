// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestOpsRepairIncidentHelpDocumentsExplicitKeyShape(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	output := executeRootForProcessInstanceTest(t, "ops", "repair", "incident", "--help")

	assertHelpOutputContainsAll(t, output,
		"Repair incidents by key",
		"Aliases:",
		"inc",
		"--key strings",
		"--state string",
		"--error-type string",
		"--batch-size int32",
		"--limit int32",
		"--retries int32",
		"--job-timeout string",
		"--vars string",
		"--vars-file string",
		"--report-file string",
		"--report-format string",
		"--dry-run",
		"--no-wait",
		"--workers int",
		"--no-worker-limit",
		"--fail-fast",
		"printf '%s\\n' \"$INCIDENT_KEY_A\" \"$INCIDENT_KEY_B\" | ./c8volt ops repair incident -",
	)

	parentOutput := executeRootForProcessInstanceTest(t, "ops", "repair", "--help")
	require.NotContains(t, parentOutput, "--key strings")
	require.NotContains(t, parentOutput, "--retries int32")
}

// TestOpsRepairIncidentDryRunWritesJSONReport verifies dry-run reports write structured audit data without mutation.
func TestOpsRepairIncidentDryRunWritesJSONReport(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	reportFile := filepath.Join(t.TempDir(), "repair.json")
	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"--json", "ops", "repair", "incident", "--state", "active", "--limit", "2", "--dry-run", "--report-file", reportFile, "--report-format", "json", "--automation"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), `"reportFile": "`+reportFile+`"`)
	require.Contains(t, string(output), `"reportFormat": "json"`)
	require.Contains(t, string(output), `"jobKeys": [`)
	require.Contains(t, string(output), `"retryUpdateStatus": "not_applicable"`)
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportFile)), &report))
	require.Equal(t, "ops.repair.v1", report["schemaVersion"])
	require.Equal(t, "ops repair incident", report["commandName"])
	require.Equal(t, "planned", report["outcome"])
	require.Equal(t, true, report["dryRun"])
	require.Equal(t, "8.9", report["camundaVersion"])
	require.Len(t, requireJSONObject(t, report["frozenSet"])["incidentKeys"], 2)
	gotRequests := strings.Join(requests.Snapshot(), "\n")
	require.Contains(t, gotRequests, "POST /v2/incidents/search")
	require.NotContains(t, gotRequests, "PATCH /v2/jobs/")
	require.NotContains(t, gotRequests, "/resolution")
}

// TestOpsRepairIncidentWritesReportForFailureAfterDiscovery verifies post-discovery failures keep audit output.
func TestOpsRepairIncidentWritesReportForFailureAfterDiscovery(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	reportFile := filepath.Join(t.TempDir(), "repair-failed.json")
	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentFailingResolutionServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{
			"ops", "repair", "incident",
			"--key", "2251799813685249",
			"--no-wait",
			"--report-file", reportFile,
			"--report-format", "json",
		}),
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "ops repair incident:")
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportFile)), &report))
	require.Equal(t, "failed", report["outcome"])
	require.Len(t, report["errors"], 1)
	require.Len(t, requireJSONObject(t, report["frozenSet"])["incidentKeys"], 1)
	require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "POST /v2/incidents/2251799813685249/resolution")
}

// TestOpsRepairIncidentVarsDryRunShowsVariableScopes verifies repair variable flags reuse update-pi parsing and appear in dry-run output.
func TestOpsRepairIncidentVarsDryRunShowsVariableScopes(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "--key", "2251799813685249", "--vars", `{"approved":true}`, "--dry-run", "--verbose"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "variable scopes: 1")
	require.Contains(t, string(output), "variable scope 2251799813685251: names=approved status=planned dependents=2251799813685249")
	require.Contains(t, string(output), "incident 2251799813685249: vars=planned")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "PUT /v2/element-instances/")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/resolution")
}

// TestOpsRepairIncidentRejectsInvalidVars verifies malformed repair variable JSON fails before remote mutation.
func TestOpsRepairIncidentRejectsInvalidVars(t *testing.T) {
	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, "http://127.0.0.1:9", "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "--key", "2251799813685249", "--vars", "{"}),
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "--vars must be a valid JSON object")
}

func TestOpsRepairIncidentExplicitKeyNoWaitRepairsThroughServices(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "--key", "2251799813685249", "--no-wait"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "repair incidents")
	require.Contains(t, string(output), "candidate incidents: 1")
	require.Contains(t, string(output), "outcome: repaired")
	require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "GET /v2/incidents/2251799813685249")
	require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "PATCH /v2/jobs/2251799813685252")
	require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "POST /v2/incidents/2251799813685249/resolution")
}

func TestOpsRepairIncidentStdinDryRunUsesFixedKeys(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocessWithStdin(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "-", "--dry-run"}),
	}, "2251799813685250\n")

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "dry run: repair incidents")
	require.Contains(t, string(output), "candidate incidents: 1")
	require.Contains(t, string(output), "repair preview: 1 incident(s), 0 related job(s), 0 variable scope(s) would be updated")
	require.Contains(t, string(output), "incidents without related jobs: 1")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "PATCH /v2/jobs/")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/resolution")
}

// TestOpsRepairIncidentFilterDryRunDiscoversFixedTargets verifies search-mode repair plans filtered incidents without mutation.
func TestOpsRepairIncidentFilterDryRunDiscoversFixedTargets(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "--state", "active", "--error-type", "io_mapping_error", "--limit", "2", "--dry-run", "--verbose"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "dry run: repair incidents")
	require.Contains(t, string(output), `selection filters: {state=active, errorType="IO_MAPPING_ERROR"}`)
	require.Contains(t, string(output), "candidate incidents: 2")
	require.Contains(t, string(output), "incident keys: 2251799813685249, 2251799813685250")
	require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "POST /v2/incidents/search")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "GET /v2/incidents/")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "PATCH /v2/jobs/")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/resolution")
}

// TestOpsRepairIncidentSearchPreflightsBeforeMutation verifies search repair plans and fixes keys before mutation.
func TestOpsRepairIncidentSearchPreflightsBeforeMutation(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "--state", "active", "--limit", "2", "--no-wait"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "repair incidents")
	requireRequestBefore(t, requests.Snapshot(), "POST /v2/incidents/search", "GET /v2/incidents/2251799813685249")
	requireRequestBefore(t, requests.Snapshot(), "GET /v2/incidents/2251799813685249", "POST /v2/incidents/2251799813685249/resolution")
}

// TestOpsRepairIncidentRejectsKeyedSearchMode verifies mixed key and filter selection fails before remote mutation.
func TestOpsRepairIncidentRejectsKeyedSearchMode(t *testing.T) {
	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, "http://127.0.0.1:9", "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "--key", "2251799813685249", "--state", "active"}),
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "--key cannot be combined with search filters")
	require.NotContains(t, string(output), "Usage:")
}

// TestOpsRepairIncidentSearchValidation verifies local batch and limit guardrails are enforced before CLI initialization.
func TestOpsRepairIncidentSearchValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "invalid batch size",
			args: []string{"ops", "repair", "incident", "--batch-size", "0"},
			want: "invalid value for --batch-size",
		},
		{
			name: "invalid limit",
			args: []string{"ops", "repair", "incident", "--limit", "0"},
			want: "--limit must be positive integer",
		},
		{
			name: "report format requires report file",
			args: []string{"ops", "repair", "incident", "--report-format", "json", "--dry-run"},
			want: "--report-format requires --report-file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, "http://127.0.0.1:9", "8.9"),
				"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, tt.args),
			})

			require.Error(t, err)
			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
			require.Contains(t, string(output), tt.want)
		})
	}
}

func TestOpsRepairIncidentInvalidKeyFailsBeforeMutation(t *testing.T) {
	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, "http://127.0.0.1:9", "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "--key", "bad-key"}),
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), `incident key "bad-key" is not a valid key`)
	require.NotContains(t, string(output), "Usage:")
}

func TestOpsRepairIncidentCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	cfgPath := os.Getenv("C8VOLT_TEST_CONFIG")
	args := unmarshalOpsRepairIncidentArgsFromEnv(t)
	root := Root()
	resetCommandTreeFlags(root)
	resetOpsRepairIncidentFlagState()
	root.SetArgs(append([]string{"--config", cfgPath}, args...))
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		handleBootstrapError(root, err)
	}
}

func newOpsRepairIncidentServer(t *testing.T, requests *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()
	return testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[` + opsRepairIncidentJSON("2251799813685249", "2251799813685251", "2251799813685252", "ACTIVE") + `,` + opsRepairIncidentJSON("2251799813685250", "2251799813685253", "", "ACTIVE") + `],"page":{"totalItems":2}}`))
		case "/v2/incidents/2251799813685249":
			_, _ = w.Write([]byte(opsRepairIncidentJSON("2251799813685249", "2251799813685251", "2251799813685252", "ACTIVE")))
		case "/v2/incidents/2251799813685250":
			_, _ = w.Write([]byte(opsRepairIncidentJSON("2251799813685250", "2251799813685253", "", "ACTIVE")))
		case "/v2/jobs/2251799813685252":
			require.Equal(t, http.MethodPatch, r.Method)
			w.WriteHeader(http.StatusNoContent)
		case "/v2/incidents/2251799813685249/resolution", "/v2/incidents/2251799813685250/resolution":
			require.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

// newOpsRepairIncidentFailingResolutionServer returns discovery data but fails after mutation begins.
func newOpsRepairIncidentFailingResolutionServer(t *testing.T, requests *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()
	return testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/incidents/2251799813685249":
			_, _ = w.Write([]byte(opsRepairIncidentJSON("2251799813685249", "2251799813685251", "2251799813685252", "ACTIVE")))
		case "/v2/jobs/2251799813685252":
			require.Equal(t, http.MethodPatch, r.Method)
			w.WriteHeader(http.StatusNoContent)
		case "/v2/incidents/2251799813685249/resolution":
			require.Equal(t, http.MethodPost, r.Method)
			http.Error(w, "resolution failed", http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func opsRepairIncidentJSON(incidentKey string, processInstanceKey string, jobKey string, state string) string {
	job := ""
	if jobKey != "" {
		job = `,"jobKey":"` + jobKey + `"`
	}
	return `{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"2251799813685300","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"` + incidentKey + `","processDefinitionId":"demo","processDefinitionKey":"2251799813685200","processInstanceKey":"` + processInstanceKey + `","rootProcessInstanceKey":"` + processInstanceKey + `","state":"` + state + `","tenantId":"<default>"` + job + `}`
}

func marshalOpsRepairIncidentArgsForEnv(t *testing.T, args []string) string {
	t.Helper()
	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}

func unmarshalOpsRepairIncidentArgsFromEnv(t *testing.T) []string {
	t.Helper()
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_OPS_REPAIR_INC_ARGS")), &args))
	return args
}

func resetOpsRepairIncidentFlagState() {
	flagOpsRepairIncidentKeys = nil
	flagOpsRepairIncidentState = "active"
	flagOpsRepairIncidentErrorType = ""
	flagOpsRepairIncidentErrorMessage = ""
	flagOpsRepairIncidentPIKey = ""
	flagOpsRepairIncidentRootKey = ""
	flagOpsRepairIncidentPDKey = ""
	flagOpsRepairIncidentBpmnProcessID = ""
	flagOpsRepairIncidentFlowNodeID = ""
	flagOpsRepairIncidentFNIKey = ""
	flagOpsRepairIncidentCreationTimeAfter = ""
	flagOpsRepairIncidentCreationTimeBefore = ""
	flagOpsRepairIncidentBatchSize = consts.MaxPISearchSize
	flagOpsRepairIncidentLimit = 0
	flagOpsRepairIncidentRetries = 1
	flagOpsRepairIncidentJobTimeoutRaw = ""
	flagOpsRepairIncidentVars = ""
	flagOpsRepairIncidentVarsFile = ""
	flagOpsRepairIncidentReportFile = ""
	flagOpsRepairIncidentReportFormat = ""
	flagDryRun = false
	flagNoWait = false
	flagWorkers = 0
	flagNoWorkerLimit = false
	flagFailFast = false
	flagVerbose = false
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagCmdAutoConfirm = false
	flagCmdAutomation = false
}

// requireRequestBefore asserts that one captured request happened before another.
func requireRequestBefore(t *testing.T, requests []string, before string, after string) {
	t.Helper()
	beforeIndex := -1
	afterIndex := -1
	for i, request := range requests {
		if beforeIndex < 0 && strings.Contains(request, before) {
			beforeIndex = i
		}
		if afterIndex < 0 && strings.Contains(request, after) {
			afterIndex = i
		}
	}
	require.NotEqual(t, -1, beforeIndex, "missing request containing %q in %v", before, requests)
	require.NotEqual(t, -1, afterIndex, "missing request containing %q in %v", after, requests)
	require.Less(t, beforeIndex, afterIndex, "%q should happen before %q", before, after)
}
