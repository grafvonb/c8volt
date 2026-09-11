// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/stretchr/testify/require"
)

// TestUpdateJobCommand_RetriesDryRunLoadsCurrentJobAndSkipsMutation verifies the update job planning and dry-run behavior covered by this scenario.
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

// TestUpdateJobCommand_RetriesDryRunRendersExplicitKeyTenantContext verifies
// direct job updates report backend-authorized tenant evidence from the loaded job.
func TestUpdateJobCommand_RetriesDryRunRendersExplicitKeyTenantContext(t *testing.T) {
	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponseWithTenant("2251799813711967", 1, "FAILED", tenantAdminKeysReturnedTenant),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "--tenant", tenantAdminKeysSelectedTenant, "update", "job", "--key", "2251799813711967", "--retries", "3", "--dry-run")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, patchBodies)
	require.Contains(t, output, "selection scope: explicit resource keys; tenant filter not applied\n")
	require.Contains(t, output, "affected tenants: "+tenantAdminKeysReturnedTenant+"\n")
	require.NotContains(t, output, "selection scope: "+tenantAdminKeysSelectedTenant)
	require.Less(t,
		strings.Index(output, "selection scope: explicit resource keys; tenant filter not applied"),
		strings.Index(output, "dry run: update job"),
	)
}

// TestUpdateJobCommand_JSONDryRunIncludesExplicitKeyTenantContext verifies the
// shared JSON envelope carries explicit-key tenant evidence without reshaping the plan payload.
func TestUpdateJobCommand_JSONDryRunIncludesExplicitKeyTenantContext(t *testing.T) {
	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponseWithTenant("2251799813711967", 1, "FAILED", tenantAdminKeysReturnedTenant),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "--tenant", tenantAdminKeysSelectedTenant, "--json", "update", "job", "--key", "2251799813711967", "--retries", "3", "--dry-run")

	require.Equal(t, []string{"POST /v2/jobs/search"}, requests)
	require.Empty(t, patchBodies)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	tenantContext := requireJSONObject(t, envelope["tenantContext"])
	require.Equal(t, "explicit_keys", tenantContext["mode"])
	require.Equal(t, "not_applied", tenantContext["filter"])
	require.Equal(t, tenantAdminKeysSelectedTenant, tenantContext["configuredTenantId"])
	require.Equal(t, []any{tenantAdminKeysReturnedTenant}, tenantContext["resolvedTenantIds"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "2251799813711967", payload["key"])
	require.Equal(t, false, payload["mutationSubmitted"])
}

// TestUpdateJobCommand_RetriesNoOpSkipsPromptAndMutation verifies the update job planning and dry-run behavior covered by this scenario.
func TestUpdateJobCommand_RetriesNoOpSkipsPromptAndMutation(t *testing.T) {
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(_ io.Writer, _ bool, _ string) error {
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

// TestUpdateJobCommand_RetriesNoOpDryRunReportsNoChangesApplied verifies the update job planning and dry-run behavior covered by this scenario.
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

// TestUpdateJobCommand_JSONDryRunRetriesPlanPayload verifies the update job planning and dry-run behavior covered by this scenario.
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

// TestUpdateJobCommand_TimeoutDryRunReportsSubmissionIntent verifies the update job planning and dry-run behavior covered by this scenario.
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

// TestUpdateJobCommand_JSONDryRunRetriesAndTimeoutPlanPayload verifies the update job planning and dry-run behavior covered by this scenario.
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
