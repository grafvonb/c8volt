// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestUpdateJobCommand_TechnicalFailureDryRunLoadsCurrentJobAndSkipsMutation verifies the update job worker outcome behavior covered by this scenario.
func TestUpdateJobCommand_TechnicalFailureDryRunLoadsCurrentJobAndSkipsMutation(t *testing.T) {
	var requests []string
	var failBodies []map[string]any
	srv := newJobFailServer(t, &requests, &failBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--fail", "--retries", "0", "--message", "worker unavailable", "--dry-run")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, failBodies)
	require.Contains(t, output, "dry run: update job 2251799813711967: technical failure: submit; retries: 0; message: worker unavailable; no changes applied")
}

// TestUpdateJobCommand_TechnicalFailureDryRunRendersExplicitKeyTenantContext
// verifies worker outcome plans reuse the current job tenant as direct-key evidence.
func TestUpdateJobCommand_TechnicalFailureDryRunRendersExplicitKeyTenantContext(t *testing.T) {
	var requests []string
	var failBodies []map[string]any
	srv := newJobFailServer(t, &requests, &failBodies, []string{
		jobSearchResponseWithTenant("2251799813711967", 1, "FAILED", tenantAdminKeysReturnedTenant),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "--tenant", tenantAdminKeysSelectedTenant, "update", "job", "--key", "2251799813711967", "--fail", "--retries", "0", "--message", "worker unavailable", "--dry-run")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, failBodies)
	require.Contains(t, output, "Tenant filter: not applied for explicit resource keys\n")
	require.Contains(t, output, "Resource tenant: "+tenantAdminKeysReturnedTenant+"\n")
	require.NotContains(t, output, "Tenant filter: "+tenantAdminKeysSelectedTenant)
	require.Less(t,
		strings.Index(output, "Tenant filter: not applied for explicit resource keys"),
		strings.Index(output, "dry run: update job"),
	)
}

// TestUpdateJobCommand_TechnicalFailureSubmittedHumanOutput verifies the update job worker outcome behavior covered by this scenario.
func TestUpdateJobCommand_TechnicalFailureSubmittedHumanOutput(t *testing.T) {
	var requests []string
	var failBodies []map[string]any
	srv := newJobFailServer(t, &requests, &failBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--fail", "--retries", "2", "--retry-backoff", "5m", "--message", "worker unavailable", "--auto-confirm")

	require.Equal(t, []string{"POST /v2/jobs/search", "POST /v2/jobs/2251799813711967/failure"}, requests)
	require.Len(t, failBodies, 1)
	requireFailRetries(t, failBodies[0], float64(2))
	requireFailRetryBackoff(t, failBodies[0], float64(300000))
	require.Equal(t, "worker unavailable", failBodies[0]["errorMessage"])
	require.Contains(t, output, "updated job 2251799813711967: submitted technical failure retries=2 retryBackoff=300000ms")
}

// TestUpdateJobCommand_JSONDryRunTechnicalFailurePlanPayload verifies the update job worker outcome behavior covered by this scenario.
func TestUpdateJobCommand_JSONDryRunTechnicalFailurePlanPayload(t *testing.T) {
	var requests []string
	var failBodies []map[string]any
	srv := newJobFailServer(t, &requests, &failBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "--json", "update", "job", "--key", "2251799813711967", "--fail", "--retries", "2", "--retry-backoff", "5m", "--message", "worker unavailable", "--dry-run")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, failBodies)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "technical_failure", payload["mode"])
	require.Equal(t, true, payload["dryRun"])
	require.Equal(t, true, payload["materialChange"])
	require.Equal(t, false, payload["mutationSubmitted"])
	require.Equal(t, float64(2), payload["requestedRetries"])
	require.Equal(t, "worker unavailable", payload["message"])
	require.Equal(t, "5m", payload["retryBackoff"])
	require.Equal(t, float64(300000), payload["retryBackoffMs"])
}

// TestUpdateJobCommand_BPMNErrorDryRunLoadsCurrentJobAndSkipsMutation verifies the update job worker outcome behavior covered by this scenario.
func TestUpdateJobCommand_BPMNErrorDryRunLoadsCurrentJobAndSkipsMutation(t *testing.T) {
	var requests []string
	var errorBodies []map[string]any
	srv := newJobBPMNErrorServer(t, &requests, &errorBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--throw-bpmn-error", "PAYMENT_DECLINED", "--message", "card declined", "--vars", `{"approved":false}`, "--dry-run")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, errorBodies)
	require.Contains(t, output, "dry run: update job 2251799813711967: BPMN error: submit; error code: PAYMENT_DECLINED; message: card declined; variables: submit; no changes applied")
}

// TestUpdateJobCommand_BPMNErrorSubmittedHumanOutput verifies the update job worker outcome behavior covered by this scenario.
func TestUpdateJobCommand_BPMNErrorSubmittedHumanOutput(t *testing.T) {
	var requests []string
	var errorBodies []map[string]any
	srv := newJobBPMNErrorServer(t, &requests, &errorBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--throw-bpmn-error", "PAYMENT_DECLINED", "--message", "card declined", "--vars", `{"approved":false}`, "--auto-confirm")

	require.Equal(t, []string{"POST /v2/jobs/search", "POST /v2/jobs/2251799813711967/error"}, requests)
	require.Len(t, errorBodies, 1)
	requireBPMNErrorCode(t, errorBodies[0], "PAYMENT_DECLINED")
	require.Equal(t, "card declined", errorBodies[0]["errorMessage"])
	require.Equal(t, map[string]any{"approved": false}, errorBodies[0]["variables"])
	require.Contains(t, output, "updated job 2251799813711967: submitted BPMN error errorCode=PAYMENT_DECLINED")
}

// TestUpdateJobCommand_JSONDryRunBPMNErrorPlanPayload verifies the update job worker outcome behavior covered by this scenario.
func TestUpdateJobCommand_JSONDryRunBPMNErrorPlanPayload(t *testing.T) {
	var requests []string
	var errorBodies []map[string]any
	srv := newJobBPMNErrorServer(t, &requests, &errorBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "--json", "update", "job", "--key", "2251799813711967", "--throw-bpmn-error", "PAYMENT_DECLINED", "--message", "card declined", "--vars", `{"approved":false}`, "--dry-run")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, errorBodies)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "bpmn_error", payload["mode"])
	require.Equal(t, true, payload["dryRun"])
	require.Equal(t, true, payload["materialChange"])
	require.Equal(t, false, payload["mutationSubmitted"])
	require.Equal(t, "PAYMENT_DECLINED", payload["errorCode"])
	require.Equal(t, "card declined", payload["message"])
	require.Equal(t, map[string]any{"approved": false}, payload["variables"])
}

// TestUpdateJobCommand_CompletionDryRunLoadsCurrentJobAndSkipsMutation verifies the update job worker outcome behavior covered by this scenario.
func TestUpdateJobCommand_CompletionDryRunLoadsCurrentJobAndSkipsMutation(t *testing.T) {
	var requests []string
	var completeBodies []map[string]any
	srv := newJobCompleteServer(t, &requests, &completeBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--complete", "--vars", `{"approved":true}`, "--dry-run")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, completeBodies)
	require.Contains(t, output, "dry run: update job 2251799813711967: completion: submit; variables: submit; no changes applied")
}

// TestUpdateJobCommand_CompletionSubmittedHumanOutput verifies the update job worker outcome behavior covered by this scenario.
func TestUpdateJobCommand_CompletionSubmittedHumanOutput(t *testing.T) {
	var requests []string
	var completeBodies []map[string]any
	srv := newJobCompleteServer(t, &requests, &completeBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--complete", "--vars", `{"approved":true}`, "--auto-confirm")

	require.Equal(t, []string{"POST /v2/jobs/search", "POST /v2/jobs/2251799813711967/completion"}, requests)
	require.Len(t, completeBodies, 1)
	require.Equal(t, map[string]any{"approved": true}, completeBodies[0]["variables"])
	require.Contains(t, output, "updated job 2251799813711967: submitted completion")
}

// TestUpdateJobCommand_CompletionSubmittedWithoutVariables verifies the update job worker outcome behavior covered by this scenario.
func TestUpdateJobCommand_CompletionSubmittedWithoutVariables(t *testing.T) {
	var requests []string
	var completeBodies []map[string]any
	srv := newJobCompleteServer(t, &requests, &completeBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--complete", "--auto-confirm")

	require.Equal(t, []string{"POST /v2/jobs/search", "POST /v2/jobs/2251799813711967/completion"}, requests)
	require.Len(t, completeBodies, 1)
	require.Equal(t, map[string]any{}, completeBodies[0]["variables"])
	require.Contains(t, output, "updated job 2251799813711967: submitted completion")
}

// TestUpdateJobCommand_JSONDryRunCompletionPlanPayload verifies the update job worker outcome behavior covered by this scenario.
func TestUpdateJobCommand_JSONDryRunCompletionPlanPayload(t *testing.T) {
	var requests []string
	var completeBodies []map[string]any
	srv := newJobCompleteServer(t, &requests, &completeBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "--json", "update", "job", "--key", "2251799813711967", "--complete", "--vars", `{"approved":true}`, "--dry-run")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, completeBodies)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "completion", payload["mode"])
	require.Equal(t, true, payload["dryRun"])
	require.Equal(t, true, payload["materialChange"])
	require.Equal(t, false, payload["mutationSubmitted"])
	require.Equal(t, map[string]any{"approved": true}, payload["variables"])
}

// TestUpdateJobCommand_WorkerOutcomeUnsupportedV87FailsBeforeMutation verifies the update job worker outcome behavior covered by this scenario.
func TestUpdateJobCommand_WorkerOutcomeUnsupportedV87FailsBeforeMutation(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	for _, mode := range []string{"fail", "bpmn-error", "completion"} {
		t.Run(mode, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, "TestUpdateJobCommand_WorkerOutcomeUnsupportedV87FailsBeforeMutationHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":             cfgPath,
				"C8VOLT_TEST_JOB_WORKER_OUTCOME": mode,
			})
			require.Error(t, err)

			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.Error, exitErr.ExitCode())
			require.Contains(t, string(output), "Camunda 8.8")
			require.NotContains(t, string(output), "updated job")
		})
	}
}

// TestUpdateJobCommand_WorkerOutcomeUnsupportedV87FailsBeforeMutationHelper verifies the update job worker outcome behavior covered by this scenario.
func TestUpdateJobCommand_WorkerOutcomeUnsupportedV87FailsBeforeMutationHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	base := []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "update", "job", "--key", "2251799813711967"}
	switch os.Getenv("C8VOLT_TEST_JOB_WORKER_OUTCOME") {
	case "fail":
		os.Args = append(base, "--fail", "--retries", "0", "--message", "worker unavailable", "--auto-confirm")
	case "bpmn-error":
		os.Args = append(base, "--throw-bpmn-error", "PAYMENT_DECLINED", "--message", "card declined", "--auto-confirm")
	default:
		os.Args = append(base, "--complete", "--auto-confirm")
	}

	Execute()
}
