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
	require.Contains(t, out.String(), "deletion: skipped")
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
