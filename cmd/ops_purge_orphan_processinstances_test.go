// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
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

const (
	opsOrphanChildKey   = "2251799813685250"
	opsOrphanParentKey  = "2251799813685249"
	opsOrphanProcessKey = "2251799813685248"
)

func TestOpsPurgeOrphanProcessInstancesHelpDocumentsSafeAutomationPreview(t *testing.T) {
	output := executeRootForProcessInstanceTest(t, "ops", "purge", "orphan-process-instances", "--help")

	assertHelpOutputContainsAll(t, output,
		"Purge orphan child process instances",
		"./c8volt ops purge orphan-process-instances --automation --json --dry-run",
	)
	assertHelpOutputOmitsAll(t, output,
		"./c8volt ops purge orphan-process-instances --automation --json\n",
	)
}

func TestOpsPurgeOrphanProcessInstancesDryRunHidesCandidateKeysWithoutDelete(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServer(t, &requests, true)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "purge", "orphan-process-instances",
		"--dry-run",
		"--state", "active",
	)

	require.Contains(t, output, "dry run: purge orphan process-instances")
	require.Contains(t, output, "candidate orphan process instances: 1")
	require.NotContains(t, output, "candidate keys:")
	require.Contains(t, output, "delete plan: planned")
	require.NotContains(t, output, "one or more parent process instances were not found")
	require.NotContains(t, output, "no deletion request submitted")
	require.Contains(t, output, "outcome: planned; no changes applied; use --verbose to list process-instance keys")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/deletion")
}

func TestOpsPurgeOrphanProcessInstancesDryRunVerboseReportsCandidateKeys(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServer(t, &requests, true)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"--verbose",
		"ops", "purge", "orphan-process-instances",
		"--dry-run",
		"--state", "active",
	)

	require.Contains(t, output, "candidate orphan process instances: 1")
	require.Contains(t, output, "candidate keys: "+opsOrphanChildKey)
	require.NotContains(t, output, "one or more parent process instances were not found")
	require.NotContains(t, output, "use --verbose to list process-instance keys")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/deletion")
}

func TestOpsPurgeOrphanProcessInstancesDryRunNoTargetsReportsNoOp(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServer(t, &requests, false)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "purge", "orphan-process-instances",
		"--dry-run",
	)

	require.Contains(t, output, "candidate orphan process instances: 0")
	require.Contains(t, output, "delete plan: skipped")
	require.Contains(t, output, "outcome: planned; no changes applied")
	require.NotContains(t, output, "use --verbose to list process-instance keys")
	snapshot := requests.Snapshot()
	require.Len(t, snapshot, 1)
	require.True(t, strings.HasPrefix(snapshot[0], "POST /v2/process-instances/search "))
}

func TestOpsPurgeOrphanProcessInstancesDryRunAppliesCompatibleFilters(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServer(t, &requests, false)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "purge", "orphan-process-instances",
		"--dry-run",
		"--bpmn-process-id", "order-process",
		"--state", "active",
		"--batch-size", "25",
		"--limit", "1",
	)

	require.Contains(t, output, "candidate orphan process instances: 0")
	request := decodeCapturedPISearchRequest(t, strings.TrimPrefix(requests.Snapshot()[0], "POST /v2/process-instances/search "))
	filter := request["filter"].(map[string]any)
	page := request["page"].(map[string]any)
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Equal(t, "ACTIVE", filter["state"])
	require.Equal(t, float64(25), page["limit"])
	require.Contains(t, filter, "parentProcessInstanceKey")
}

func TestOpsPurgeOrphanProcessInstancesAutoConfirmDeletesCandidateKeys(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"ops", "purge", "orphan-process-instances",
		"--auto-confirm",
		"--no-wait",
	)

	require.Contains(t, output, "purge orphan process-instances")
	require.Contains(t, output, "candidate orphan process instances: 1")
	require.Contains(t, output, "delete plan: planned")
	require.NotContains(t, output, "one or more parent process instances were not found")
	require.Contains(t, output, "deletion: submitted (requests: 1)")
	require.Contains(t, output, "deletion confirmation: skipped (--no-wait)")
	require.Contains(t, output, "outcome: deleted")
	require.Contains(t, output, "elapsed:")
	require.Equal(t, []string{"/v2/process-instances/" + opsOrphanChildKey + "/deletion"}, deleted.Snapshot())
	require.NotContains(t, strings.Join(deleted.Snapshot(), "\n"), opsOrphanParentKey)
}

func TestOpsPurgeOrphanProcessInstancesAutoConfirmNoTargetsSkipsDelete(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, false, "TERMINATED")
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"ops", "purge", "orphan-process-instances",
		"--auto-confirm",
		"--no-wait",
	)

	require.Contains(t, output, "candidate orphan process instances: 0")
	require.Contains(t, output, "delete plan: skipped")
	require.Contains(t, output, "outcome: planned; no targets deleted")
	require.Empty(t, deleted.Snapshot())
}

func TestOpsPurgeOrphanProcessInstancesAutomationDeletesWithoutAutoConfirm(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"--automation",
		"ops", "purge", "orphan-process-instances",
		"--no-wait",
	)

	require.Contains(t, output, "deletion: submitted (requests: 1)")
	require.Contains(t, output, "deletion confirmation: skipped (--no-wait)")
	require.Contains(t, output, "outcome: deleted")
	require.Equal(t, []string{"/v2/process-instances/" + opsOrphanChildKey + "/deletion"}, deleted.Snapshot())
}

func TestOpsPurgeOrphanProcessInstancesAutomationJSONUsesEnvelope(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"--automation",
		"--json",
		"ops", "purge", "orphan-process-instances",
		"--no-wait",
	)

	require.NotContains(t, stderr, "purge orphan process-instances\n")
	require.NotContains(t, stderr, "report: written")
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "ops purge orphan-process-instances", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "deleted", payload["outcome"])
	require.Equal(t, true, payload["deleteRequested"])
	require.NotContains(t, stdout, "purge orphan process-instances\n")
	require.Equal(t, []string{"/v2/process-instances/" + opsOrphanChildKey + "/deletion"}, deleted.Snapshot())
}

func TestOpsPurgeOrphanProcessInstancesWritesMarkdownReport(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServer(t, &requests, true)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "orphan-purge.md")

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "purge", "orphan-process-instances",
		"--dry-run",
		"--report-file", reportPath,
	)

	require.Contains(t, output, "outcome: planned; no changes applied")
	require.Contains(t, output, "report: written "+reportPath)
	report := readReportFile(t, reportPath)
	require.Contains(t, report, "# Orphan Process Instance Purge Audit Report")
	require.Contains(t, report, "- Command: ops purge orphan-process-instances")
	require.Contains(t, report, "- Dry Run: true")
	require.Contains(t, report, "- No Wait: false")
	require.Contains(t, report, "- Outcome: planned")
	require.Contains(t, report, "- Camunda Version: 8.8")
	require.Contains(t, report, "- Profile: default")
	require.Contains(t, report, "  - "+opsOrphanChildKey)
}

func TestOpsPurgeOrphanProcessInstancesWritesJSONReport(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "orphan-purge.json")
	require.NoError(t, os.WriteFile(reportPath, []byte("old report"), 0o600))

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"ops", "purge", "orphan-process-instances",
		"--auto-confirm",
		"--no-wait",
		"--report-file", reportPath,
		"--report-format", "json",
	)

	require.Contains(t, output, "outcome: deleted")
	require.Contains(t, output, "report: written "+reportPath)
	require.NotContains(t, readReportFile(t, reportPath), "old report")
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "ops.orphan-process-instances.v1", report["schemaVersion"])
	require.Equal(t, "ops purge orphan-process-instances", report["commandName"])
	require.Equal(t, "deleted", report["outcome"])
	require.Equal(t, true, report["deleteRequested"])
	require.Equal(t, true, report["noWait"])
	require.NotContains(t, report, "dryRun")
	require.Equal(t, "8.9", report["camundaVersion"])
	discovery := requireJSONObject(t, report["discovery"])
	require.Equal(t, float64(1), discovery["count"])
	keys := discovery["keys"].([]any)
	require.Equal(t, opsOrphanChildKey, keys[0])
	deletion := requireJSONObject(t, report["deletion"])
	require.Equal(t, "submitted", deletion["status"])
	require.Equal(t, true, deletion["noWait"])
}

func TestOpsPurgeOrphanProcessInstancesExistingReportFailsBeforePreflight(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "orphan-purge.md")
	const existingReport = "existing report"
	require.NoError(t, os.WriteFile(reportPath, []byte(existingReport), 0o600))

	output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeOrphanProcessInstancesAbortPreservesExistingReportHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_REPORT": reportPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "report file already exists: "+reportPath)
	require.NotContains(t, string(output), "aborted by user")
	require.NotContains(t, string(output), "write audit report")
	require.Equal(t, existingReport, readReportFile(t, reportPath))
	require.Empty(t, requests.Snapshot())
	require.Empty(t, deleted.Snapshot())
}

func TestOpsPurgeOrphanProcessInstancesWritesReportAfterPostDiscoveryFailure(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "ACTIVE")
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "orphan-purge-failed.json")

	output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeOrphanProcessInstancesWritesReportAfterPostDiscoveryFailureHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_REPORT": reportPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "refusing to delete orphan process-instance scope")
	require.Empty(t, deleted.Snapshot())
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "failed", report["outcome"])
	discovery := requireJSONObject(t, report["discovery"])
	require.Equal(t, float64(1), discovery["count"])
	deletion := requireJSONObject(t, report["deletion"])
	require.Equal(t, "blocked", deletion["status"])
	require.NotEmpty(t, report["errors"])
}

func TestOpsPurgeOrphanProcessInstancesWritesReportAfterPostDiscoveryFailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"ops", "purge", "orphan-process-instances",
		"--auto-confirm",
		"--report-file", os.Getenv("C8VOLT_TEST_REPORT"),
		"--report-format", "json",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestOpsPurgeOrphanProcessInstancesAbortPreservesExistingReportHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	confirmCmdOrAbortFn = func(bool, string) error {
		return localPreconditionError(ErrCmdAborted)
	}
	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"ops", "purge", "orphan-process-instances",
		"--report-file", os.Getenv("C8VOLT_TEST_REPORT"),
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func readReportFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func newOpsOrphanPurgeServer(t *testing.T, requests *testx.SafeSlice[string], withOrphan bool) *httptest.Server {
	return newOpsOrphanPurgeServerWithState(t, requests, nil, withOrphan, "ACTIVE")
}

func newOpsOrphanPurgeServerWithState(t *testing.T, requests *testx.SafeSlice[string], deleted *testx.SafeSlice[string], withOrphan bool, orphanState string) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(string(body), opsOrphanChildKey) || !withOrphan {
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[` + opsOrphanProcessInstanceJSON(opsOrphanChildKey, opsOrphanParentKey, orphanState) + `],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/"+opsOrphanChildKey:
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(opsOrphanProcessInstanceJSON(opsOrphanChildKey, opsOrphanParentKey, orphanState)))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/"+opsOrphanParentKey:
			requests.Append(r.Method + " " + r.URL.Path)
			http.NotFound(w, r)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/"+opsOrphanChildKey+"/deletion":
			if deleted != nil {
				deleted.Append(r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func opsOrphanProcessInstanceJSON(key string, parentKey string, state string) string {
	parent := ""
	if parentKey != "" {
		parent = `,"parentProcessInstanceKey":"` + parentKey + `","rootProcessInstanceKey":"` + opsOrphanProcessKey + `"`
	}
	return `{"processInstanceKey":"` + key + `","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"startDate":"2026-05-11T12:00:00Z","state":"` + state + `","tenantId":"tenant"` + parent + `}`
}
