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
		"--retries int32",
		"--job-timeout string",
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
	require.Contains(t, string(output), "frozen incidents: 1")
	require.Contains(t, string(output), "outcome: repaired")
	require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "GET /v2/incidents/2251799813685249")
	require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "PATCH /v2/jobs/2251799813685252")
	require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "POST /v2/incidents/2251799813685249/resolution")
}

func TestOpsRepairIncidentStdinDryRunFreezesKeys(t *testing.T) {
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
	require.Contains(t, string(output), "frozen incidents: 1")
	require.Contains(t, string(output), "related jobs: 0 applicable, 1 not applicable")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "PATCH /v2/jobs/")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/resolution")
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
		switch r.URL.Path {
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
	flagOpsRepairIncidentRetries = 1
	flagOpsRepairIncidentJobTimeoutRaw = ""
	flagDryRun = false
	flagNoWait = false
	flagWorkers = 0
	flagNoWorkerLimit = false
	flagFailFast = false
	flagVerbose = false
	flagCmdAutoConfirm = false
	flagCmdAutomation = false
}
