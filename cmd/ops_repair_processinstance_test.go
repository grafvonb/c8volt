// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestOpsRepairProcessInstanceHelpDocumentsSelectionShape verifies the target-specific key and incident selector contract.
func TestOpsRepairProcessInstanceHelpDocumentsSelectionShape(t *testing.T) {
	resetOpsRepairProcessInstanceFlagState()
	t.Cleanup(resetOpsRepairProcessInstanceFlagState)

	output := executeRootForProcessInstanceTest(t, "ops", "repair", "process-instance", "--help")

	assertHelpOutputContainsAll(t, output,
		"Repair incidents selected by process instances",
		"Aliases:",
		"pi",
		"--key strings",
		"--incidents-only",
		"--direct-incidents-only",
		"--batch-size int32",
		"--limit int32",
		"--retries int32",
		"--job-timeout string",
		"--vars string",
		"--vars-file string",
		"--dry-run",
		"--no-wait",
		"--workers int",
		"printf '%s\\n' \"$PI_KEY_A\" \"$PI_KEY_B\" | ./c8volt ops repair process-instance -",
	)
}

// TestOpsRepairProcessInstanceVarsFileDryRunShowsVariableScopes verifies file-backed variables are planned per process-instance scope.
func TestOpsRepairProcessInstanceVarsFileDryRunShowsVariableScopes(t *testing.T) {
	resetOpsRepairProcessInstanceFlagState()
	t.Cleanup(resetOpsRepairProcessInstanceFlagState)

	varsFile := t.TempDir() + "/repair-vars.json"
	require.NoError(t, os.WriteFile(varsFile, []byte(`{"customerTier":"gold"}`), 0o600))

	var requests testx.SafeSlice[string]
	srv := newOpsRepairProcessInstanceServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairProcessInstanceCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":             writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_PI_ARGS": marshalOpsRepairProcessInstanceArgsForEnv(t, []string{"ops", "repair", "process-instance", "--key", "2251799813685251", "--vars-file", varsFile, "--dry-run", "--verbose"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "variable scopes: 1")
	require.Contains(t, string(output), "variable scope 2251799813685251: names=customerTier status=planned dependents=2251799813685249")
	require.Contains(t, string(output), "process-instance 2251799813685251 incident 2251799813685249: vars=planned")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "PUT /v2/element-instances/")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/resolution")
}

// TestOpsRepairProcessInstanceExplicitKeyNoWaitRepairsDiscoveredIncidents verifies keyed PI repair routes to incident repair.
func TestOpsRepairProcessInstanceExplicitKeyNoWaitRepairsDiscoveredIncidents(t *testing.T) {
	resetOpsRepairProcessInstanceFlagState()
	t.Cleanup(resetOpsRepairProcessInstanceFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairProcessInstanceServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairProcessInstanceCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":             writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_PI_ARGS": marshalOpsRepairProcessInstanceArgsForEnv(t, []string{"ops", "repair", "process-instance", "--key", "2251799813685251", "--no-wait"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "repair process-instance incidents")
	require.Contains(t, string(output), "frozen process instances: 1")
	require.Contains(t, string(output), "frozen incidents: 1 deduped")
	require.Contains(t, string(output), "outcome: repaired")
	gotRequests := strings.Join(requests.Snapshot(), "\n")
	require.Contains(t, gotRequests, "GET /v2/process-instances/2251799813685251")
	require.Contains(t, gotRequests, "POST /v2/process-instances/2251799813685251/incidents/search")
	require.Contains(t, gotRequests, "PATCH /v2/jobs/2251799813685252")
	require.Contains(t, gotRequests, "POST /v2/incidents/2251799813685249/resolution")
}

// TestOpsRepairProcessInstanceStdinDryRunFreezesDiscoveredIncidents verifies stdin PI keys discover incidents without mutation in dry-run.
func TestOpsRepairProcessInstanceStdinDryRunFreezesDiscoveredIncidents(t *testing.T) {
	resetOpsRepairProcessInstanceFlagState()
	t.Cleanup(resetOpsRepairProcessInstanceFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairProcessInstanceServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocessWithStdin(t, "TestOpsRepairProcessInstanceCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":             writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_PI_ARGS": marshalOpsRepairProcessInstanceArgsForEnv(t, []string{"ops", "repair", "process-instance", "-", "--dry-run"}),
	}, "2251799813685253\n")

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "dry run: repair process-instance incidents")
	require.Contains(t, string(output), "frozen process instances: 1")
	require.Contains(t, string(output), "frozen incidents: 1 deduped")
	require.Contains(t, string(output), "related jobs: 0 applicable, 1 not applicable")
	gotRequests := strings.Join(requests.Snapshot(), "\n")
	require.NotContains(t, gotRequests, "PATCH /v2/jobs/")
	require.NotContains(t, gotRequests, "/resolution")
}

// TestOpsRepairProcessInstanceRejectsInvalidSelection verifies local validation catches ambiguous or unsafe inputs.
func TestOpsRepairProcessInstanceRejectsInvalidSelection(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "invalid key",
			args: []string{"ops", "repair", "process-instance", "--key", "bad-key"},
			want: `process-instance key "bad-key" is not a valid key`,
		},
		{
			name: "keyed plus filter",
			args: []string{"ops", "repair", "process-instance", "--key", "2251799813685251", "--incidents-only"},
			want: "--key cannot be combined with process-instance search filters",
		},
		{
			name: "search without incident selector",
			args: []string{"ops", "repair", "process-instance", "--state", "active", "--dry-run"},
			want: "process-instance search repair requires --incidents-only or --direct-incidents-only",
		},
		{
			name: "invalid limit",
			args: []string{"ops", "repair", "process-instance", "--incidents-only", "--limit", "0"},
			want: "--limit must be positive integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, "TestOpsRepairProcessInstanceCommandHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":             writeTestConfigForVersion(t, "http://127.0.0.1:9", "8.9"),
				"C8VOLT_TEST_OPS_REPAIR_PI_ARGS": marshalOpsRepairProcessInstanceArgsForEnv(t, tt.args),
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

// TestOpsRepairProcessInstanceCommandHelper runs process-instance repair commands in a subprocess for exit-code assertions.
func TestOpsRepairProcessInstanceCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	cfgPath := os.Getenv("C8VOLT_TEST_CONFIG")
	args := unmarshalOpsRepairProcessInstanceArgsFromEnv(t)
	root := Root()
	resetCommandTreeFlags(root)
	resetOpsRepairProcessInstanceFlagState()
	root.SetArgs(append([]string{"--config", cfgPath}, args...))
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		handleBootstrapError(root, err)
	}
}

// newOpsRepairProcessInstanceServer provides the minimal v2 endpoints used by process-instance repair tests.
func newOpsRepairProcessInstanceServer(t *testing.T, requests *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()
	return testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/2251799813685251":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(opsRepairProcessInstanceJSON("2251799813685251")))
		case "/v2/process-instances/2251799813685253":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(opsRepairProcessInstanceJSON("2251799813685253")))
		case "/v2/process-instances/2251799813685251/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[` + opsRepairIncidentJSON("2251799813685249", "2251799813685251", "2251799813685252", "ACTIVE") + `],"page":{"totalItems":1}}`))
		case "/v2/process-instances/2251799813685253/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[` + opsRepairIncidentJSON("2251799813685250", "2251799813685253", "", "ACTIVE") + `],"page":{"totalItems":1}}`))
		case "/v2/jobs/2251799813685252":
			require.Equal(t, http.MethodPatch, r.Method)
			w.WriteHeader(http.StatusNoContent)
		case "/v2/incidents/2251799813685249/resolution":
			require.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

// opsRepairProcessInstanceJSON returns a compact process-instance API response for repair tests.
func opsRepairProcessInstanceJSON(key string) string {
	return `{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"2251799813685200","processDefinitionName":"demo","processDefinitionVersion":1,"processInstanceKey":"` + key + `","rootProcessInstanceKey":"` + key + `","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"<default>"}`
}

// marshalOpsRepairProcessInstanceArgsForEnv serializes subprocess command arguments.
func marshalOpsRepairProcessInstanceArgsForEnv(t *testing.T, args []string) string {
	t.Helper()
	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}

// unmarshalOpsRepairProcessInstanceArgsFromEnv deserializes subprocess command arguments.
func unmarshalOpsRepairProcessInstanceArgsFromEnv(t *testing.T) []string {
	t.Helper()
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_OPS_REPAIR_PI_ARGS")), &args))
	return args
}

// resetOpsRepairProcessInstanceFlagState restores shared command globals that process-instance repair uses.
func resetOpsRepairProcessInstanceFlagState() {
	flagOpsRepairProcessInstanceKeys = nil
	flagOpsRepairProcessInstanceRetries = 1
	flagOpsRepairProcessInstanceJobTimeoutRaw = ""
	flagOpsRepairProcessInstanceVars = ""
	flagOpsRepairProcessInstanceVarsFile = ""
	flagGetPIBpmnProcessID = ""
	flagGetPIProcessVersion = 0
	flagGetPIProcessVersionTag = ""
	flagGetPIProcessDefinitionKey = ""
	flagGetPIStartDateAfter = ""
	flagGetPIStartDateBefore = ""
	flagGetPIEndDateAfter = ""
	flagGetPIEndDateBefore = ""
	flagGetPIStartAfterDays = -1
	flagGetPIStartBeforeDays = -1
	flagGetPIEndAfterDays = -1
	flagGetPIEndBeforeDays = -1
	flagGetPIState = "all"
	flagGetPIParentKey = ""
	flagGetPISize = consts.MaxPISearchSize
	flagGetPILimit = 0
	flagGetPIRootsOnly = false
	flagGetPIChildrenOnly = false
	flagGetPIOrphanChildrenOnly = false
	flagGetPIIncidentsOnly = false
	flagGetPIDirectIncidentsOnly = false
	flagGetPINoIncidentsOnly = false
	flagGetPIIncidentState = "active"
	flagGetPIIncidentErrorType = ""
	flagGetPIIncidentErrorMessage = ""
	flagGetPIIncidentMessageLimit = 0
	flagGetPIWithIncidents = false
	flagGetPIWithVars = false
	flagGetPIVarValueLimit = 0
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
