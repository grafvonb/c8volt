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
	require.Contains(t, output, "dry run: update job 2251799813711967: timeout: submit 5m; no changes applied")
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
}
