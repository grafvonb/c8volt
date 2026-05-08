// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

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
	require.Contains(t, output, "plan: update job 2251799813711967: nothing to update; pending confirmation")
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
	require.Contains(t, output, "plan: update job 2251799813711967: retries: 1 -> 3; pending confirmation")
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
	return `{"items":[{"jobKey":"` + key + `","state":"FAILED","retries":` + strconvFormatInt32(retries) + `,"processInstanceKey":"2251799813711000","elementInstanceKey":"2251799813711001","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`
}

func requirePatchRetries(t *testing.T, body map[string]any, want float64) {
	t.Helper()
	changeset := requireJSONObject(t, body["changeset"])
	require.Equal(t, want, changeset["retries"])
}

func strconvFormatInt32(value int32) string {
	return strconv.Itoa(int(value))
}

func TestUpdateJobCommand_RejectsJSONMutationWithoutAutoConfirmOrAutomationBeforeLookupOrMutation(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)

	flagViewAsJson = true

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
