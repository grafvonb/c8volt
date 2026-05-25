// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestUpdateJobCommand_RetriesConfirmedHumanOutput(t *testing.T) {
	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponse("2251799813711967", 1),
		jobSearchResponse("2251799813711967", 3),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--retries", "3", "--auto-confirm")

	require.Equal(t, []string{"POST /v2/jobs/search", "PATCH /v2/jobs/2251799813711967", "POST /v2/jobs/search"}, requests)
	require.Len(t, patchBodies, 1)
	requirePatchRetries(t, patchBodies[0], float64(3))
	require.Contains(t, output, "updated job 2251799813711967: confirmed retries=3")
}

func TestUpdateJobCommand_RetriesConfirmedJSONOutput(t *testing.T) {
	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponse("2251799813711967", 1),
		jobSearchResponse("2251799813711967", 3),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "--json", "update", "job", "--key", "2251799813711967", "--retries", "3", "--auto-confirm")

	require.Equal(t, []string{"POST /v2/jobs/search", "PATCH /v2/jobs/2251799813711967", "POST /v2/jobs/search"}, requests)
	require.Len(t, patchBodies, 1)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "confirmed", payload["status"])
	require.Equal(t, true, payload["mutationAccepted"])
	require.Equal(t, "confirmed", payload["confirmationStatus"])
	require.Equal(t, float64(3), payload["submittedRetries"])
	require.Equal(t, float64(3), payload["confirmedRetries"])
	plan := requireJSONObject(t, payload["plan"])
	require.Equal(t, true, plan["mutationSubmitted"])
	require.Equal(t, "changed", plan["retryStatus"])
}

func TestUpdateJobCommand_RetriesDryRunLoadsCurrentJobAndSkipsMutation(t *testing.T) {
	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--retries", "3", "--dry-run")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, patchBodies)
	require.Contains(t, output, "dry run: update job 2251799813711967: retries: 1 -> 3; no changes applied")
}

func TestUpdateJobCommand_RetriesNoOpSkipsPromptAndMutation(t *testing.T) {
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt for retry no-op")
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponse("2251799813711967", 3),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--retries", "3")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, patchBodies)
	require.Contains(t, output, "plan: update job 2251799813711967: nothing to update; no confirmation required")
}

func TestUpdateJobCommand_RetriesNoOpDryRunReportsNoChangesApplied(t *testing.T) {
	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponse("2251799813711967", 3),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--retries", "3", "--dry-run")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, patchBodies)
	require.Contains(t, output, "dry run: update job 2251799813711967: nothing to update; no changes applied")
}

func TestUpdateJobCommand_MaterialInteractiveRetriesUpdateRequiresConfirmation(t *testing.T) {
	prevConfirm := confirmCmdOrAbortFn
	var prompt string
	confirmCmdOrAbortFn = func(autoConfirm bool, got string) error {
		require.False(t, autoConfirm)
		prompt = got
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponse("2251799813711967", 1),
		jobSearchResponse("2251799813711967", 3),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--retries", "3")

	require.Contains(t, prompt, "You are about to update job 2251799813711967")
	require.Equal(t, []string{"POST /v2/jobs/search", "PATCH /v2/jobs/2251799813711967", "POST /v2/jobs/search"}, requests)
	require.Len(t, patchBodies, 1)
	require.Contains(t, output, "plan: update job 2251799813711967: retries: 1 -> 3")
	require.NotContains(t, output, "pending confirmation")
	require.Contains(t, output, "updated job 2251799813711967: confirmed retries=3")
}

func TestUpdateJobCommand_JSONDryRunRetriesPlanPayload(t *testing.T) {
	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "--json", "update", "job", "--key", "2251799813711967", "--retries", "3", "--dry-run")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, patchBodies)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "2251799813711967", payload["key"])
	require.Equal(t, false, payload["mutationSubmitted"])
	require.Equal(t, true, payload["dryRun"])
	require.Equal(t, "update", payload["mode"])
	require.Equal(t, true, payload["materialChange"])
	require.Equal(t, "changed", payload["retryStatus"])
	require.Equal(t, float64(3), payload["requestedRetries"])
	current := requireJSONObject(t, payload["current"])
	require.Equal(t, float64(1), current["retries"])
}

func TestUpdateJobTimeoutSubmittedViewIncludesSubmittedTimeoutOnly(t *testing.T) {
	timeoutMillis := int64(300000)
	cmd, output := newJobViewTestCommand()

	err := jobUpdateResultView(cmd, job.UpdateResult{
		Key:                "2251799813711967",
		Status:             "submitted",
		MutationAccepted:   true,
		ConfirmationStatus: "skipped",
		SubmittedTimeoutMS: &timeoutMillis,
	})

	require.NoError(t, err)
	require.Equal(t, "updated job 2251799813711967: submitted timeout=300000ms\n", output.String())
}

func TestUpdateJobRetriesAndTimeoutViewShowsRetriesConfirmedAndTimeoutSubmitted(t *testing.T) {
	retries := int32(3)
	timeoutMillis := int64(300000)
	cmd, output := newJobViewTestCommand()

	err := jobUpdateResultView(cmd, job.UpdateResult{
		Key:                "2251799813711967",
		Status:             "confirmed",
		MutationAccepted:   true,
		ConfirmationStatus: "confirmed",
		SubmittedRetries:   &retries,
		SubmittedTimeoutMS: &timeoutMillis,
		ConfirmedRetries:   &retries,
	})

	require.NoError(t, err)
	require.Equal(t, "updated job 2251799813711967: confirmed retries=3; timeout=300000ms submitted\n", output.String())
	require.NotContains(t, output.String(), "deadline")
}

func TestUpdateJobCommand_TimeoutSubmittedHumanOutputWithoutConfirmationPolling(t *testing.T) {
	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponseWithState("2251799813711967", 1, "CREATED"),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--timeout", "5m", "--auto-confirm")

	require.Equal(t, []string{"POST /v2/jobs/search", "PATCH /v2/jobs/2251799813711967"}, requests)
	require.Len(t, patchBodies, 1)
	requirePatchTimeout(t, patchBodies[0], float64(300000))
	require.NotContains(t, output, "confirmed")
	require.Contains(t, output, "updated job 2251799813711967: submitted timeout=300000ms")
}

func TestUpdateJobCommand_RetriesAndTimeoutConfirmsRetriesOnly(t *testing.T) {
	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponseWithState("2251799813711967", 1, "CREATED"),
		jobSearchResponse("2251799813711967", 3),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--retries", "3", "--timeout", "5m", "--auto-confirm")

	require.Equal(t, []string{"POST /v2/jobs/search", "PATCH /v2/jobs/2251799813711967", "POST /v2/jobs/search"}, requests)
	require.Len(t, patchBodies, 1)
	requirePatchRetries(t, patchBodies[0], float64(3))
	requirePatchTimeout(t, patchBodies[0], float64(300000))
	require.Contains(t, output, "updated job 2251799813711967: confirmed retries=3; timeout=300000ms submitted")
	require.NotContains(t, output, "confirmed deadline")
}

func TestUpdateJobCommand_TimeoutDryRunReportsSubmissionIntent(t *testing.T) {
	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponseWithState("2251799813711967", 1, "CREATED"),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--timeout", "5m", "--dry-run")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, patchBodies)
	require.Contains(t, output, "dry run: update job 2251799813711967: timeout: set to 5m; no changes applied")
	require.NotContains(t, output, "deadline")
}

func TestUpdateJobCommand_JSONDryRunRetriesAndTimeoutPlanPayload(t *testing.T) {
	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponseWithState("2251799813711967", 1, "CREATED"),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "--json", "update", "job", "--key", "2251799813711967", "--retries", "3", "--timeout", "5m", "--dry-run")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, patchBodies)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, true, payload["dryRun"])
	require.Equal(t, true, payload["materialChange"])
	require.Equal(t, false, payload["mutationSubmitted"])
	require.Equal(t, "changed", payload["retryStatus"])
	require.Equal(t, float64(3), payload["requestedRetries"])
	require.Equal(t, "5m", payload["requestedTimeout"])
	require.Equal(t, float64(300000), payload["timeoutMillis"])
	items := payload["items"].([]any)
	require.Len(t, items, 2)
	timeoutItem := requireJSONObject(t, items[1])
	require.Equal(t, "timeout", timeoutItem["name"])
	require.Equal(t, "5m", timeoutItem["after"])
	require.Equal(t, "submit", timeoutItem["status"])
	require.Empty(t, timeoutItem["before"])
}

func TestUpdateJobCommand_NoWaitSkipsRetryConfirmation(t *testing.T) {
	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--retries", "3", "--no-wait", "--auto-confirm")

	require.Equal(t, []string{"POST /v2/jobs/search", "PATCH /v2/jobs/2251799813711967"}, requests)
	require.Len(t, patchBodies, 1)
	requirePatchRetries(t, patchBodies[0], float64(3))
	require.Contains(t, output, "updated job 2251799813711967: submitted retries=3")
	require.NotContains(t, output, "confirmed retries")
}

func TestUpdateJobCommand_NoWaitJSONSubmittedResult(t *testing.T) {
	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "--json", "update", "job", "--key", "2251799813711967", "--retries", "3", "--no-wait", "--auto-confirm")

	require.Equal(t, []string{"POST /v2/jobs/search", "PATCH /v2/jobs/2251799813711967"}, requests)
	require.Len(t, patchBodies, 1)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "submitted", payload["status"])
	require.Equal(t, true, payload["mutationAccepted"])
	require.Equal(t, "skipped", payload["confirmationStatus"])
	require.Equal(t, float64(3), payload["submittedRetries"])
	require.NotContains(t, payload, "confirmedRetries")
	plan := requireJSONObject(t, payload["plan"])
	require.Equal(t, true, plan["mutationSubmitted"])
	require.Equal(t, "changed", plan["retryStatus"])
}

func TestUpdateJobCommand_NoWaitStillRequiresInteractiveConfirmationForMaterialUpdates(t *testing.T) {
	prevConfirm := confirmCmdOrAbortFn
	var prompt string
	confirmCmdOrAbortFn = func(autoConfirm bool, got string) error {
		require.False(t, autoConfirm)
		prompt = got
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponse("2251799813711967", 1),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--retries", "3", "--no-wait")

	require.Contains(t, prompt, "You are about to update job 2251799813711967")
	require.Equal(t, []string{"POST /v2/jobs/search", "PATCH /v2/jobs/2251799813711967"}, requests)
	require.Len(t, patchBodies, 1)
	require.Contains(t, output, "plan: update job 2251799813711967: retries: 1 -> 3")
	require.NotContains(t, output, "pending confirmation")
	require.Contains(t, output, "updated job 2251799813711967: submitted retries=3")
	require.NotContains(t, output, "confirmed retries")
}

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

func TestUpdateJobCommand_UnsupportedV87FailsBeforeMutation(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, err := testx.RunCmdSubprocess(t, "TestUpdateJobCommand_UnsupportedV87FailsBeforeMutationHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "get job")
	require.Contains(t, string(output), "Camunda 8.8")
	require.NotContains(t, string(output), "updated job")
}

func TestUpdateJobCommand_UnsupportedV87FailsBeforeMutationHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "update", "job", "--key", "2251799813711967", "--retries", "3", "--auto-confirm"}

	Execute()
}

func TestUpdateJobCommand_RejectsJSONVerboseBeforeLookupOrMutation(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)

	flagViewAsJson = true
	flagVerbose = true
	flagDryRun = true

	err := validateUpdateJobJSONGuardrails(updateJobCmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--json cannot be combined with --verbose for update job")
}

func TestParseUpdateJobRequestParsesTechnicalFailure(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())
	require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
	require.NoError(t, updateJobCmd.Flags().Set("retries", "2"))
	require.NoError(t, updateJobCmd.Flags().Set("retry-backoff", "5m"))
	t.Cleanup(func() {
		require.NoError(t, updateJobCmd.Flags().Set("fail", "false"))
		require.NoError(t, updateJobCmd.Flags().Set("retries", "0"))
		require.NoError(t, updateJobCmd.Flags().Set("retry-backoff", ""))
	})

	flagUpdateJobKey = "2251799813711967"
	flagUpdateJobFail = true
	flagUpdateJobRetries = 2
	flagUpdateJobRetryBackoffRaw = "5m"
	flagUpdateJobMessage = "worker unavailable"

	request, err := parseUpdateJobRequest(updateJobCmd)

	require.NoError(t, err)
	require.NotNil(t, request.WorkerOutcome)
	require.Equal(t, job.WorkerOutcomeTechnicalFailure, request.WorkerOutcome.Mode)
	require.Equal(t, int32(2), *request.WorkerOutcome.Retries)
	require.Equal(t, int64(300000), *request.WorkerOutcome.RetryBackoffMillis)
	require.Equal(t, "worker unavailable", request.WorkerOutcome.Message)
	require.Nil(t, request.Retries)
	require.Nil(t, request.TimeoutMillis)
}

func TestParseUpdateJobRequestParsesBPMNError(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())
	require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
	require.NoError(t, updateJobCmd.Flags().Set("vars", `{"approved":false}`))
	t.Cleanup(func() {
		require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", ""))
		require.NoError(t, updateJobCmd.Flags().Set("vars", ""))
	})

	flagUpdateJobKey = "2251799813711967"
	flagUpdateJobBPMNError = "PAYMENT_DECLINED"
	flagUpdateJobMessage = "card declined"
	flagUpdateJobVariables = `{"approved":false}`

	request, err := parseUpdateJobRequest(updateJobCmd)

	require.NoError(t, err)
	require.NotNil(t, request.WorkerOutcome)
	require.Equal(t, job.WorkerOutcomeBPMNError, request.WorkerOutcome.Mode)
	require.Equal(t, "PAYMENT_DECLINED", request.WorkerOutcome.ErrorCode)
	require.Equal(t, "card declined", request.WorkerOutcome.Message)
	require.Equal(t, map[string]any{"approved": false}, request.WorkerOutcome.Variables)
	require.Nil(t, request.Retries)
	require.Nil(t, request.TimeoutMillis)
}

func TestParseUpdateJobRequestParsesCompletion(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())
	require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
	require.NoError(t, updateJobCmd.Flags().Set("vars", `{"approved":true}`))
	t.Cleanup(func() {
		require.NoError(t, updateJobCmd.Flags().Set("complete", "false"))
		require.NoError(t, updateJobCmd.Flags().Set("vars", ""))
	})

	flagUpdateJobKey = "2251799813711967"
	flagUpdateJobComplete = true
	flagUpdateJobVariables = `{"approved":true}`

	request, err := parseUpdateJobRequest(updateJobCmd)

	require.NoError(t, err)
	require.NotNil(t, request.WorkerOutcome)
	require.Equal(t, job.WorkerOutcomeCompletion, request.WorkerOutcome.Mode)
	require.Equal(t, map[string]any{"approved": true}, request.WorkerOutcome.Variables)
	require.Nil(t, request.Retries)
	require.Nil(t, request.TimeoutMillis)
}

func TestParseUpdateJobRequestRejectsTechnicalFailureValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		set     func(t *testing.T)
		message string
	}{
		{
			name: "missing retries",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
			},
			message: "technical job failure requires --retries",
		},
		{
			name: "negative retries",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("retries", "-1"))
				flagUpdateJobRetries = -1
			},
			message: "invalid value for --retries",
		},
		{
			name: "invalid retry backoff",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("retries", "1"))
				require.NoError(t, updateJobCmd.Flags().Set("retry-backoff", "0s"))
				flagUpdateJobRetries = 1
				flagUpdateJobRetryBackoffRaw = "0s"
			},
			message: "invalid value for --retry-backoff",
		},
		{
			name: "timeout conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("retries", "1"))
				require.NoError(t, updateJobCmd.Flags().Set("timeout", "5m"))
				flagUpdateJobRetries = 1
			},
			message: "--fail cannot be combined with --timeout",
		},
		{
			name: "bpmn conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("retries", "1"))
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				flagUpdateJobRetries = 1
			},
			message: "--throw-bpmn-error cannot be combined with --fail",
		},
		{
			name: "complete conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("retries", "1"))
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				flagUpdateJobRetries = 1
			},
			message: "--fail cannot be combined with --complete",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetUpdateJobFlagState()
			t.Cleanup(resetUpdateJobFlagState)
			resetCommandTreeFlags(Root())
			flagUpdateJobKey = "2251799813711967"
			tt.set(t)

			_, err := parseUpdateJobRequest(updateJobCmd)

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.message)
		})
	}
}

func TestParseUpdateJobRequestRejectsBPMNErrorValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		set     func(t *testing.T)
		message string
	}{
		{
			name: "empty code",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", ""))
			},
			message: "BPMN error requires a non-empty --throw-bpmn-error",
		},
		{
			name: "fail conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
			},
			message: "--throw-bpmn-error cannot be combined with --fail",
		},
		{
			name: "complete conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
			},
			message: "--throw-bpmn-error cannot be combined with --complete",
		},
		{
			name: "retries conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				require.NoError(t, updateJobCmd.Flags().Set("retries", "1"))
			},
			message: "--throw-bpmn-error cannot be combined with --retries",
		},
		{
			name: "timeout conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				require.NoError(t, updateJobCmd.Flags().Set("timeout", "5m"))
			},
			message: "--throw-bpmn-error cannot be combined with --timeout",
		},
		{
			name: "retry backoff conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				require.NoError(t, updateJobCmd.Flags().Set("retry-backoff", "5m"))
			},
			message: "--throw-bpmn-error cannot be combined with --retry-backoff",
		},
		{
			name: "invalid variables",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				require.NoError(t, updateJobCmd.Flags().Set("vars", "{"))
				flagUpdateJobVariables = "{"
			},
			message: "--vars must be a valid JSON object",
		},
		{
			name: "non-object variables",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
				require.NoError(t, updateJobCmd.Flags().Set("vars", `["bad"]`))
				flagUpdateJobVariables = `["bad"]`
			},
			message: "--vars must be a JSON object",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetUpdateJobFlagState()
			t.Cleanup(resetUpdateJobFlagState)
			resetCommandTreeFlags(Root())
			flagUpdateJobKey = "2251799813711967"
			flagUpdateJobBPMNError = "PAYMENT_DECLINED"
			tt.set(t)

			_, err := parseUpdateJobRequest(updateJobCmd)

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.message)
		})
	}
}

func TestParseUpdateJobRequestRejectsCompletionValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		set     func(t *testing.T)
		message string
	}{
		{
			name: "fail conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("fail", "true"))
			},
			message: "--fail cannot be combined with --complete",
		},
		{
			name: "bpmn conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("throw-bpmn-error", "PAYMENT_DECLINED"))
			},
			message: "--throw-bpmn-error cannot be combined with --complete",
		},
		{
			name: "retries conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("retries", "1"))
			},
			message: "--complete cannot be combined with --retries",
		},
		{
			name: "timeout conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("timeout", "5m"))
			},
			message: "--complete cannot be combined with --timeout",
		},
		{
			name: "retry backoff conflict",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("retry-backoff", "5m"))
			},
			message: "--complete cannot be combined with --retry-backoff",
		},
		{
			name: "invalid variables",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("vars", "{"))
				flagUpdateJobVariables = "{"
			},
			message: "--vars must be a valid JSON object",
		},
		{
			name: "non-object variables",
			set: func(t *testing.T) {
				require.NoError(t, updateJobCmd.Flags().Set("complete", "true"))
				require.NoError(t, updateJobCmd.Flags().Set("vars", `["bad"]`))
				flagUpdateJobVariables = `["bad"]`
			},
			message: "--vars must be a JSON object",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetUpdateJobFlagState()
			t.Cleanup(resetUpdateJobFlagState)
			resetCommandTreeFlags(Root())
			flagUpdateJobKey = "2251799813711967"
			tt.set(t)

			_, err := parseUpdateJobRequest(updateJobCmd)

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.message)
		})
	}
}

func newJobUpdateServer(t *testing.T, requests *[]string, patchBodies *[]map[string]any, searchResponses []string, updateStatus int) *httptest.Server {
	t.Helper()
	searchIndex := 0
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requests = append(*requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/jobs/search":
			require.Less(t, searchIndex, len(searchResponses))
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			require.NotEmpty(t, filter["jobKey"])
			_, _ = w.Write([]byte(searchResponses[searchIndex]))
			searchIndex++
		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/v2/jobs/"):
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			*patchBodies = append(*patchBodies, body)
			w.WriteHeader(updateStatus)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
}

func newJobFailServer(t *testing.T, requests *[]string, failBodies *[]map[string]any, searchResponses []string, failStatus int) *httptest.Server {
	t.Helper()
	searchIndex := 0
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requests = append(*requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/jobs/search":
			require.Less(t, searchIndex, len(searchResponses))
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			require.NotEmpty(t, filter["jobKey"])
			_, _ = w.Write([]byte(searchResponses[searchIndex]))
			searchIndex++
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/failure"):
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			*failBodies = append(*failBodies, body)
			w.WriteHeader(failStatus)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
}

func newJobBPMNErrorServer(t *testing.T, requests *[]string, errorBodies *[]map[string]any, searchResponses []string, errorStatus int) *httptest.Server {
	t.Helper()
	searchIndex := 0
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requests = append(*requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/jobs/search":
			require.Less(t, searchIndex, len(searchResponses))
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			require.NotEmpty(t, filter["jobKey"])
			_, _ = w.Write([]byte(searchResponses[searchIndex]))
			searchIndex++
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/error"):
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			*errorBodies = append(*errorBodies, body)
			w.WriteHeader(errorStatus)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
}

func newJobCompleteServer(t *testing.T, requests *[]string, completeBodies *[]map[string]any, searchResponses []string, completeStatus int) *httptest.Server {
	t.Helper()
	searchIndex := 0
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requests = append(*requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/jobs/search":
			require.Less(t, searchIndex, len(searchResponses))
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			require.NotEmpty(t, filter["jobKey"])
			_, _ = w.Write([]byte(searchResponses[searchIndex]))
			searchIndex++
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/completion"):
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			*completeBodies = append(*completeBodies, body)
			w.WriteHeader(completeStatus)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
}

func jobSearchResponse(key string, retries int32) string {
	return jobSearchResponseWithState(key, retries, "FAILED")
}

// jobSearchResponseWithState builds a get job response fixture with an explicit state.
func jobSearchResponseWithState(key string, retries int32, state string) string {
	return `{"items":[{"jobKey":"` + key + `","state":"` + state + `","retries":` + strconvFormatInt32(retries) + `,"processInstanceKey":"2251799813711000","elementInstanceKey":"2251799813711001","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`
}

func requirePatchRetries(t *testing.T, body map[string]any, want float64) {
	t.Helper()
	changeset := requireJSONObject(t, body["changeset"])
	require.Equal(t, want, changeset["retries"])
}

func requirePatchTimeout(t *testing.T, body map[string]any, want float64) {
	t.Helper()
	changeset := requireJSONObject(t, body["changeset"])
	require.Equal(t, want, changeset["timeout"])
}

func requireFailRetries(t *testing.T, body map[string]any, want float64) {
	t.Helper()
	require.Equal(t, want, body["retries"])
}

func requireFailRetryBackoff(t *testing.T, body map[string]any, want float64) {
	t.Helper()
	require.Equal(t, want, body["retryBackOff"])
}

func requireBPMNErrorCode(t *testing.T, body map[string]any, want string) {
	t.Helper()
	require.Equal(t, want, body["errorCode"])
}

func newJobViewTestCommand() (*cobra.Command, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	return cmd, buf
}

func strconvFormatInt32(value int32) string {
	return strconv.Itoa(int(value))
}

func TestParseUpdateJobRequestParsesTimeoutMillis(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())
	require.NoError(t, updateJobCmd.Flags().Set("timeout", "5m"))
	t.Cleanup(func() { require.NoError(t, updateJobCmd.Flags().Set("timeout", "")) })

	flagUpdateJobKey = "2251799813711967"
	flagUpdateJobTimeoutRaw = "5m"

	request, err := parseUpdateJobRequest(updateJobCmd)

	require.NoError(t, err)
	require.NotNil(t, request.TimeoutMillis)
	require.Equal(t, int64(300000), *request.TimeoutMillis)
	require.False(t, request.ConfirmRetries)
}

// TestParseUpdateJobRequestPreservesRetryUpdateMode protects the legacy retry path while worker outcome flags are added.
func TestParseUpdateJobRequestPreservesRetryUpdateMode(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())
	require.NoError(t, updateJobCmd.Flags().Set("retries", "3"))
	t.Cleanup(func() { require.NoError(t, updateJobCmd.Flags().Set("retries", "0")) })

	flagUpdateJobKey = "2251799813711967"
	flagUpdateJobRetries = 3
	flagNoWait = false

	request, err := parseUpdateJobRequest(updateJobCmd)

	require.NoError(t, err)
	require.Equal(t, "2251799813711967", request.Key)
	require.NotNil(t, request.Retries)
	require.Equal(t, int32(3), *request.Retries)
	require.True(t, request.ConfirmRetries)
	require.Nil(t, request.TimeoutMillis)
}

// TestParseUpdateJobRequestPreservesTimeoutUpdateMode protects timeout conversion without enabling retry confirmation.
func TestParseUpdateJobRequestPreservesTimeoutUpdateMode(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())
	require.NoError(t, updateJobCmd.Flags().Set("timeout", "5m"))
	t.Cleanup(func() { require.NoError(t, updateJobCmd.Flags().Set("timeout", "")) })

	flagUpdateJobKey = "2251799813711967"
	flagUpdateJobTimeoutRaw = "5m"

	request, err := parseUpdateJobRequest(updateJobCmd)

	require.NoError(t, err)
	require.Equal(t, "2251799813711967", request.Key)
	require.Nil(t, request.Retries)
	require.NotNil(t, request.TimeoutMillis)
	require.Equal(t, int64(300000), *request.TimeoutMillis)
	require.False(t, request.ConfirmRetries)
}

// TestUpdateJobPlanPreconditionRejectsTimeoutForNonActiveJob verifies timeout updates stop before mutation when the job is not active.
func TestUpdateJobPlanPreconditionRejectsTimeoutForNonActiveJob(t *testing.T) {
	timeoutMillis := int64(20000)
	plan := job.UpdatePlan{
		Key: "2251799814014237",
		Current: job.Job{
			Key:   "2251799814014237",
			State: "RETRIES_UPDATED",
		},
	}
	request := job.UpdateRequest{Key: "2251799814014237", TimeoutMillis: &timeoutMillis}

	err := validateUpdateJobPlanPreconditions(plan, request)

	require.Error(t, err)
	require.Contains(t, err.Error(), "local precondition failed")
	require.Contains(t, err.Error(), "job timeout can be updated only for active jobs")
	require.Contains(t, err.Error(), "job 2251799814014237 is RETRIES_UPDATED")
}

// TestUpdateJobPlanPreconditionAllowsTimeoutForCreatedJob verifies timeout updates remain valid for active get job state.
func TestUpdateJobPlanPreconditionAllowsTimeoutForCreatedJob(t *testing.T) {
	timeoutMillis := int64(20000)
	plan := job.UpdatePlan{
		Key: "2251799813711967",
		Current: job.Job{
			Key:   "2251799813711967",
			State: "CREATED",
		},
	}
	request := job.UpdateRequest{Key: "2251799813711967", TimeoutMillis: &timeoutMillis}

	err := validateUpdateJobPlanPreconditions(plan, request)

	require.NoError(t, err)
}

// TestUpdateJobPlanPreconditionAllowsRetryOnlyForNonActiveJob verifies retry updates are not blocked by timeout-only state checks.
func TestUpdateJobPlanPreconditionAllowsRetryOnlyForNonActiveJob(t *testing.T) {
	retries := int32(2)
	plan := job.UpdatePlan{
		Key: "2251799814014237",
		Current: job.Job{
			Key:   "2251799814014237",
			State: "FAILED",
		},
	}
	request := job.UpdateRequest{Key: "2251799814014237", Retries: &retries}

	err := validateUpdateJobPlanPreconditions(plan, request)

	require.NoError(t, err)
}

func TestUpdateJobCommand_RejectsJSONMutationWithoutAutoConfirmOrAutomationBeforeLookupOrMutation(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)

	flagViewAsJson = true

	err := validateUpdateJobJSONGuardrails(updateJobCmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--json update job requires --dry-run, --auto-confirm, or --automation")
}

func TestUpdateJobCommand_RejectsJSONNoWaitWithoutAutoConfirmOrAutomationBeforeLookupOrMutation(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)

	flagViewAsJson = true
	flagNoWait = true

	err := validateUpdateJobJSONGuardrails(updateJobCmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--json update job requires --dry-run, --auto-confirm, or --automation")
}

func TestUpdateJobCommand_AllowsJSONDryRunWithoutAutoConfirm(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)

	flagViewAsJson = true
	flagDryRun = true

	err := validateUpdateJobJSONGuardrails(updateJobCmd)

	require.NoError(t, err)
}

func TestParseUpdateJobRequestRequiresUpdateFlag(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())

	flagUpdateJobKey = "2251799813711967"

	_, err := parseUpdateJobRequest(updateJobCmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "update job requires --retries, --timeout, or both")
}

func resetUpdateJobFlagState() {
	flagViewAsJson = false
	flagVerbose = false
	flagDryRun = false
	flagNoWait = false
	flagCmdAutoConfirm = false
	flagCmdAutomation = false
	flagUpdateJobKey = ""
	flagUpdateJobRetries = 0
	flagUpdateJobTimeoutRaw = ""
	flagUpdateJobFail = false
	flagUpdateJobRetryBackoffRaw = ""
	flagUpdateJobMessage = ""
	flagUpdateJobBPMNError = ""
	flagUpdateJobComplete = false
	flagUpdateJobVariables = ""
}
