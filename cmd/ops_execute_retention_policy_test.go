// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const opsRetentionPolicySeedKey = "2251799813685249"

func TestOpsExecuteRetentionPolicyHelpDocumentsCommand(t *testing.T) {
	output := executeRootForProcessInstanceTest(t, "ops", "execute", "--help")

	assertHelpOutputContainsAll(t, output,
		"Discover predefined operational playbooks",
		"retention-policy",
	)

	commandOutput := executeRootForProcessInstanceTest(t, "ops", "execute", "retention-policy", "--help")

	assertHelpOutputContainsAll(t, commandOutput,
		"Execute process-instance retention cleanup",
		"--retention-days int",
		"./c8volt ops execute retention-policy --retention-days 90 --dry-run",
	)
}

func TestOpsExecuteRetentionPolicyInvalidRetentionDays(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")
	tests := []struct {
		name   string
		helper string
		args   []string
		want   string
	}{
		{
			name:   "missing",
			helper: "TestOpsExecuteRetentionPolicyInvalidRetentionDaysHelper",
			args:   []string{"ops", "execute", "retention-policy"},
			want:   "ops execute retention-policy requires --retention-days",
		},
		{
			name:   "negative",
			helper: "TestOpsExecuteRetentionPolicyInvalidRetentionDaysHelper",
			args:   []string{"ops", "execute", "retention-policy", "--retention-days", "-1"},
			want:   "invalid value for --retention-days: -1, expected non-negative integer",
		},
		{
			name:   "non integer",
			helper: "TestOpsExecuteRetentionPolicyInvalidRetentionDaysHelper",
			args:   []string{"ops", "execute", "retention-policy", "--retention-days", "not-a-number"},
			want:   "invalid argument \"not-a-number\" for \"--retention-days\" flag",
		},
		{
			name:   "explicit key",
			helper: "TestOpsExecuteRetentionPolicyInvalidRetentionDaysHelper",
			args:   []string{"ops", "execute", "retention-policy", "--retention-days", "90", "--key", "2251799813685249"},
			want:   "retention policy discovers eligible process instances and does not accept explicit process-instance keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, tt.helper, map[string]string{
				"C8VOLT_TEST_CONFIG":         cfgPath,
				"C8VOLT_TEST_RETENTION_ARGS": marshalRetentionArgsForEnv(t, tt.args),
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

func TestOpsExecuteRetentionPolicyInvalidRetentionDaysHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_RETENTION_ARGS")), &args); err != nil {
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

func TestOpsExecuteRetentionPolicyDryRunAppliesCompatibleFilters(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() { relativeDayNow = prevNow })

	var requests testx.SafeSlice[string]
	srv := newOpsRetentionPolicySearchServer(t, &requests)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "retention-policy",
		"--retention-days", "90",
		"--dry-run",
		"--bpmn-process-id", "order-process",
		"--state", "completed",
		"--batch-size", "25",
		"--limit", "1",
		"--no-incidents-only",
	)

	require.Contains(t, output, "retention discovery: planned")
	require.Contains(t, output, "selection filters:")
	snapshot := requests.Snapshot()
	require.Len(t, snapshot, 1)
	request := decodeCapturedPISearchRequest(t, strings.TrimPrefix(snapshot[0], "POST /v2/process-instances/search "))
	filter := request["filter"].(map[string]any)
	page := request["page"].(map[string]any)
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Equal(t, "COMPLETED", filter["state"])
	require.Equal(t, false, filter["hasIncident"])
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-02-13T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")
	require.Equal(t, float64(25), page["limit"])
}

func TestOpsExecuteRetentionPolicyDryRunNoTargetsReportsNoOp(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsRetentionPolicySearchServer(t, &requests)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "retention-policy",
		"--retention-days", "90",
		"--dry-run",
	)

	require.Contains(t, output, "retention discovery: planned")
	require.Contains(t, output, "retention seeds: 0")
	require.Contains(t, output, "no retention cleanup targets found")
	require.Contains(t, output, "delete plan: skipped")
	require.Contains(t, output, "deletion: skipped; no deletion request submitted")
	require.Contains(t, output, "outcome: planned; no changes applied")
	snapshot := requests.Snapshot()
	require.Len(t, snapshot, 1)
	require.True(t, strings.HasPrefix(snapshot[0], "POST /v2/process-instances/search "))
}

func TestOpsExecuteRetentionPolicyDryRunJSONOutputIsStructured(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() { relativeDayNow = prevNow })

	var requests testx.SafeSlice[string]
	srv := newOpsRetentionPolicySearchServer(t, &requests)
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"--json",
		"ops", "execute", "retention-policy",
		"--retention-days", "90",
		"--dry-run",
	)

	require.Empty(t, strings.TrimSpace(stderr))
	require.NotContains(t, stdout, "retention discovery:")
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "ops execute retention-policy", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "planned", payload["outcome"])

	request := requireJSONObject(t, payload["request"])
	require.Equal(t, true, request["dryRun"])
	require.Equal(t, float64(90), request["retentionDays"])
	require.Equal(t, "2026-02-13", request["derivedEndDateBoundary"])

	discovery := requireJSONObject(t, payload["discovery"])
	require.Equal(t, "planned", discovery["status"])
	require.Equal(t, float64(0), discovery["count"])
	require.Equal(t, float64(90), discovery["retentionDays"])
	require.Equal(t, "2026-02-13", discovery["derivedEndDateBoundary"])

	deletePlan := requireJSONObject(t, payload["deletePlan"])
	require.Equal(t, "skipped", deletePlan["status"])
	deletion := requireJSONObject(t, payload["deletion"])
	require.Equal(t, "skipped", deletion["status"])
	require.NotContains(t, deletion, "submitted")
	report := requireJSONObject(t, payload["report"])
	require.Equal(t, "planned", report["outcome"])
}

func TestOpsExecuteRetentionPolicyDryRunDoesNotPromptOrMutate(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsRetentionPolicyServerWithSeed(t, &requests, &deleted)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "execute", "retention-policy",
		"--retention-days", "90",
		"--dry-run",
	)

	require.Contains(t, output, "delete plan: planned")
	require.Contains(t, output, "deletion: skipped; no deletion request submitted")
	require.Contains(t, output, "outcome: planned; no changes applied")
	require.Empty(t, deleted.Snapshot())
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/deletion")
}

func TestOpsExecuteRetentionPolicyDryRunDiscoveryOutput(t *testing.T) {
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	err := renderOpsExecuteRetentionPolicyResult(cmd, ops.RetentionPolicyResult{
		Request: ops.RetentionPolicyRequest{
			RetentionDays:          90,
			DerivedEndDateBoundary: "2026-02-13",
		},
		Discovery: ops.RetentionDiscoveryResult{
			Status:        ops.WorkflowStepStatusPlanned,
			RetentionDays: 90,
			Filters: process.ProcessInstanceFilter{
				BpmnProcessId: "invoice",
				EndDateBefore: "2026-02-13",
			},
			SeedKeys: []string{"seed-1", "seed-2"},
			Count:    2,
		},
		DeletePlan: ops.RetentionDeletePlan{
			Status: ops.WorkflowStepStatusSkipped,
		},
		Deletion: ops.RetentionDeletionResult{
			Status: ops.WorkflowStepStatusSkipped,
		},
		Outcome: ops.RetentionPolicyOutcomePlanned,
	})

	require.NoError(t, err)
	require.Contains(t, out.String(), "retention policy: planned")
	require.Contains(t, out.String(), "retention days: 90")
	require.Contains(t, out.String(), "retention boundary: endDate <= 2026-02-13")
	require.Contains(t, out.String(), "selection filters: {bpmnProcessId=\"invoice\", endDateBefore=\"2026-02-13\"}")
	require.Contains(t, out.String(), "retention discovery: planned")
	require.Contains(t, out.String(), "retention seeds: 2")
	require.Contains(t, out.String(), "delete plan: skipped")
	require.Contains(t, out.String(), "deletion: skipped; no deletion request submitted")
	require.Contains(t, out.String(), "outcome: planned; no changes applied")
}

func TestOpsExecuteRetentionPolicyDryRunPlanRendering(t *testing.T) {
	prevVerbose := flagVerbose
	flagVerbose = true
	t.Cleanup(func() { flagVerbose = prevVerbose })

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	err := renderOpsExecuteRetentionPolicyResult(cmd, ops.RetentionPolicyResult{
		Request: ops.RetentionPolicyRequest{
			RetentionDays:          90,
			DerivedEndDateBoundary: "2026-02-13",
		},
		Discovery: ops.RetentionDiscoveryResult{
			Status:  ops.WorkflowStepStatusPlanned,
			Count:   2,
			Filters: process.ProcessInstanceFilter{EndDateBefore: "2026-02-13"},
		},
		DeletePlan: ops.RetentionDeletePlan{
			Status:                ops.WorkflowStepStatusPlanned,
			SeedKeys:              []string{"child-1", "child-2"},
			ResolvedRootKeys:      []string{"root-1"},
			AffectedKeys:          []string{"root-1", "child-1", "child-2"},
			DuplicateKeys:         []string{"root-1"},
			NonFinalAffectedItems: []process.ProcessInstance{{Key: "child-2", State: process.StateActive}},
			MissingAncestors:      []process.MissingAncestor{{Key: "missing-parent", StartKey: "child-2"}},
			TraversalWarnings:     []string{"one or more parent process instances were not found"},
			RequiresConfirmation:  true,
		},
		Deletion: ops.RetentionDeletionResult{
			Status: ops.WorkflowStepStatusBlocked,
		},
		Outcome: ops.RetentionPolicyOutcomeFailed,
	})

	require.NoError(t, err)
	got := out.String()
	require.Contains(t, got, "delete plan: planned (seeds: 2, roots: 1, affected: 3)")
	require.Contains(t, got, "duplicate roots: 1")
	require.Contains(t, got, "process instances not in final state: 1 (use --force to cancel before delete)")
	require.Contains(t, got, "missing ancestors: 1")
	require.Contains(t, got, "traversal warning: one or more parent process instances were not found")
	require.Contains(t, got, "confirmation required: true")
	require.Contains(t, got, "retention seed keys: child-1, child-2")
	require.Contains(t, got, "resolved root keys: root-1")
	require.Contains(t, got, "affected process-instance keys: root-1, child-1, child-2")
	require.Contains(t, got, "deletion: blocked")
}

func marshalRetentionArgsForEnv(t *testing.T, args []string) string {
	t.Helper()

	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}

func newOpsRetentionPolicySearchServer(t *testing.T, requests *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func newOpsRetentionPolicyServerWithSeed(t *testing.T, requests *testx.SafeSlice[string], deleted *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(string(body), "parentProcessInstanceKey") {
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[` + opsRetentionPolicyProcessInstanceJSON(opsRetentionPolicySeedKey) + `],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/"+opsRetentionPolicySeedKey:
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(opsRetentionPolicyProcessInstanceJSON(opsRetentionPolicySeedKey)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/"+opsRetentionPolicySeedKey+"/deletion":
			if deleted != nil {
				deleted.Append(r.URL.Path)
			}
			requests.Append(r.Method + " " + r.URL.Path)
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func opsRetentionPolicyProcessInstanceJSON(key string) string {
	return `{"processInstanceKey":"` + key + `","processDefinitionId":"retention-process","processDefinitionKey":"9001","processDefinitionName":"retention-process","processDefinitionVersion":3,"startDate":"2026-01-11T12:00:00Z","endDate":"2026-02-12T12:00:00Z","state":"COMPLETED","tenantId":"tenant"}`
}
