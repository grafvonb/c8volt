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
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const (
	opsOrphanChildKey   = "2251799813685250"
	opsOrphanParentKey  = "2251799813685249"
	opsOrphanProcessKey = "2251799813685248"
)

func TestOpsPurgeOrphanProcessInstancesDryRunReportsDiscoveredKeysWithoutDelete(t *testing.T) {
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
	require.Contains(t, output, "discovered orphan process instances: 1")
	require.Contains(t, output, "discovered keys: "+opsOrphanChildKey)
	require.Contains(t, output, "delete plan: planned")
	require.Contains(t, output, "no deletion request submitted")
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

	require.Contains(t, output, "discovered orphan process instances: 0")
	require.Contains(t, output, "delete plan: skipped")
	require.Contains(t, output, "outcome: planned; no changes applied")
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

	require.Contains(t, output, "discovered orphan process instances: 0")
	request := decodeCapturedPISearchRequest(t, strings.TrimPrefix(requests.Snapshot()[0], "POST /v2/process-instances/search "))
	filter := request["filter"].(map[string]any)
	page := request["page"].(map[string]any)
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Equal(t, "ACTIVE", filter["state"])
	require.Equal(t, float64(25), page["limit"])
	require.Contains(t, filter, "parentProcessInstanceKey")
}

func TestOpsPurgeOrphanProcessInstancesAutoConfirmDeletesDiscoveredKeys(t *testing.T) {
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
	require.Contains(t, output, "discovered orphan process instances: 1")
	require.Contains(t, output, "delete plan: planned")
	require.Contains(t, output, "deletion: submitted (requests: 1)")
	require.Contains(t, output, "outcome: deleted")
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

	require.Contains(t, output, "discovered orphan process instances: 0")
	require.Contains(t, output, "delete plan: skipped")
	require.Contains(t, output, "outcome: planned; no targets deleted")
	require.Empty(t, deleted.Snapshot())
}

func TestOpsPurgeOrphanProcessInstancesAutomationWithoutAutoConfirmFailsBeforeMutation(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeOrphanProcessInstancesAutomationWithoutAutoConfirmFailsBeforeMutationHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "requires --auto-confirm")
	require.Empty(t, deleted.Snapshot())
}

func TestOpsPurgeOrphanProcessInstancesAutomationJSONAutoConfirmUsesEnvelope(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"--automation",
		"--json",
		"ops", "purge", "orphan-process-instances",
		"--auto-confirm",
		"--no-wait",
	)

	require.Empty(t, stderr)
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

func TestOpsPurgeOrphanProcessInstancesAutomationGuardRequiresAutoConfirmForTargets(t *testing.T) {
	err := validateOpsPurgeAutomationConfirmation(ops.OrphanPurgeRequest{
		CommandName: opsPurgeOrphanProcessInstancesCommandName,
		Automation:  true,
	}, 1)

	require.Error(t, err)
	require.Contains(t, err.Error(), "requires --auto-confirm")
}

func TestOpsPurgeOrphanProcessInstancesAutomationGuardAllowsDryRunAndNoTargets(t *testing.T) {
	require.NoError(t, validateOpsPurgeAutomationConfirmation(ops.OrphanPurgeRequest{
		CommandName: opsPurgeOrphanProcessInstancesCommandName,
		Automation:  true,
		DryRun:      true,
	}, 1))
	require.NoError(t, validateOpsPurgeAutomationConfirmation(ops.OrphanPurgeRequest{
		CommandName: opsPurgeOrphanProcessInstancesCommandName,
		Automation:  true,
	}, 0))
	require.NoError(t, validateOpsPurgeAutomationConfirmation(ops.OrphanPurgeRequest{
		CommandName: opsPurgeOrphanProcessInstancesCommandName,
		Automation:  true,
		AutoConfirm: true,
	}, 1))
}

func TestOpsPurgeOrphanProcessInstancesAutomationWithoutAutoConfirmFailsBeforeMutationHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"--automation",
		"ops", "purge", "orphan-process-instances",
		"--no-wait",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
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
