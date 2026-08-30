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

// TestUpdateJobCommand_RetriesConfirmedHumanOutput verifies the update job command wiring behavior covered by this scenario.
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

// TestUpdateJobCommand_RetriesConfirmedRendersExplicitKeyTenantContext verifies
// auto-confirmed direct job updates show tenant evidence before mutation output.
func TestUpdateJobCommand_RetriesConfirmedRendersExplicitKeyTenantContext(t *testing.T) {
	var requests []string
	var patchBodies []map[string]any
	srv := newJobUpdateServer(t, &requests, &patchBodies, []string{
		jobSearchResponseWithTenant("2251799813711967", 1, "FAILED", tenantAdminKeysReturnedTenant),
		jobSearchResponseWithTenant("2251799813711967", 3, "FAILED", tenantAdminKeysReturnedTenant),
	}, http.StatusNoContent)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForJobTest(t, "--config", cfgPath, "--tenant", tenantAdminKeysSelectedTenant, "update", "job", "--key", "2251799813711967", "--retries", "3", "--auto-confirm")

	require.Equal(t, []string{"POST /v2/jobs/search", "PATCH /v2/jobs/2251799813711967", "POST /v2/jobs/search"}, requests)
	require.Len(t, patchBodies, 1)
	require.Contains(t, output, "selection scope: explicit resource keys; tenant filter not applied\n")
	require.Contains(t, output, "affected tenants: "+tenantAdminKeysReturnedTenant+"\n")
	require.NotContains(t, output, "selection scope: "+tenantAdminKeysSelectedTenant)
	require.Less(t,
		strings.Index(output, "selection scope: explicit resource keys; tenant filter not applied"),
		strings.Index(output, "updated job 2251799813711967"),
	)
}

// TestUpdateJobCommand_RetriesConfirmedJSONOutput verifies the update job command wiring behavior covered by this scenario.
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

// TestUpdateJobCommand_MaterialInteractiveRetriesUpdateRequiresConfirmation verifies the update job command wiring behavior covered by this scenario.
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

// TestUpdateJobCommand_V810PromptedRetriesUpdatePreservesConfirmationFlow verifies V810 job updates keep the command prompt and confirmation contract.
func TestUpdateJobCommand_V810PromptedRetriesUpdatePreservesConfirmationFlow(t *testing.T) {
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
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.10")

	output := executeRootForJobTest(t, "--config", cfgPath, "update", "job", "--key", "2251799813711967", "--retries", "3")

	require.Equal(t, "You are about to update job 2251799813711967. Do you want to proceed?", prompt)
	require.Equal(t, []string{"POST /v2/jobs/search", "PATCH /v2/jobs/2251799813711967", "POST /v2/jobs/search"}, requests)
	require.Len(t, patchBodies, 1)
	requirePatchRetries(t, patchBodies[0], float64(3))
	require.Contains(t, output, "plan: update job 2251799813711967: retries: 1 -> 3")
	require.Contains(t, output, "updated job 2251799813711967: confirmed retries=3")
	require.NotContains(t, output, `"outcome"`)
}

// TestUpdateJobTimeoutSubmittedViewIncludesSubmittedTimeoutOnly verifies the update job command wiring behavior covered by this scenario.
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

// TestUpdateJobRetriesAndTimeoutViewShowsRetriesConfirmedAndTimeoutSubmitted verifies the update job command wiring behavior covered by this scenario.
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

// TestUpdateJobCommand_TimeoutSubmittedHumanOutputWithoutConfirmationPolling verifies the update job command wiring behavior covered by this scenario.
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

// TestUpdateJobCommand_RetriesAndTimeoutConfirmsRetriesOnly verifies the update job command wiring behavior covered by this scenario.
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

// TestUpdateJobCommand_NoWaitSkipsRetryConfirmation verifies the update job command wiring behavior covered by this scenario.
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

// TestUpdateJobCommand_NoWaitJSONSubmittedResult verifies the update job command wiring behavior covered by this scenario.
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

// TestUpdateJobCommand_NoWaitStillRequiresInteractiveConfirmationForMaterialUpdates verifies the update job command wiring behavior covered by this scenario.
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

// TestUpdateJobCommand_UnsupportedV87FailsBeforeMutation verifies the update job command wiring behavior covered by this scenario.
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

// TestUpdateJobCommand_UnsupportedV87FailsBeforeMutationHelper verifies the update job command wiring behavior covered by this scenario.
func TestUpdateJobCommand_UnsupportedV87FailsBeforeMutationHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "update", "job", "--key", "2251799813711967", "--retries", "3", "--auto-confirm"}

	Execute()
}

// newJobUpdateServer returns a fake update endpoint that records retry and timeout mutation requests.
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

// newJobFailServer returns a fake worker-failure endpoint that records submitted failure bodies.
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

// newJobBPMNErrorServer returns a fake BPMN error endpoint that records submitted error bodies.
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

// newJobCompleteServer returns a fake completion endpoint that records submitted variable payloads.
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

// jobSearchResponse builds a failed get job response fixture with the requested retry count.
func jobSearchResponse(key string, retries int32) string {
	return jobSearchResponseWithState(key, retries, "FAILED")
}

// jobSearchResponseWithState builds a get job response fixture with an explicit state.
func jobSearchResponseWithState(key string, retries int32, state string) string {
	return jobSearchResponseWithTenant(key, retries, state, "tenant-a")
}

// jobSearchResponseWithTenant builds a get job response fixture with explicit tenant evidence.
func jobSearchResponseWithTenant(key string, retries int32, state string, tenantID string) string {
	return `{"items":[{"jobKey":"` + key + `","state":"` + state + `","retries":` + strconvFormatInt32(retries) + `,"processInstanceKey":"2251799813711000","elementInstanceKey":"2251799813711001","tenantId":"` + tenantID + `"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`
}

// requirePatchRetries asserts the retry changeset sent to the job update endpoint.
func requirePatchRetries(t *testing.T, body map[string]any, want float64) {
	t.Helper()
	changeset := requireJSONObject(t, body["changeset"])
	require.Equal(t, want, changeset["retries"])
}

// requirePatchTimeout asserts the timeout changeset sent to the job update endpoint.
func requirePatchTimeout(t *testing.T, body map[string]any, want float64) {
	t.Helper()
	changeset := requireJSONObject(t, body["changeset"])
	require.Equal(t, want, changeset["timeout"])
}

// requireFailRetries asserts the retries sent to the technical-failure endpoint.
func requireFailRetries(t *testing.T, body map[string]any, want float64) {
	t.Helper()
	require.Equal(t, want, body["retries"])
}

// requireFailRetryBackoff asserts the retry backoff sent to the technical-failure endpoint.
func requireFailRetryBackoff(t *testing.T, body map[string]any, want float64) {
	t.Helper()
	require.Equal(t, want, body["retryBackOff"])
}

// requireBPMNErrorCode asserts the BPMN error code sent to the error endpoint.
func requireBPMNErrorCode(t *testing.T, body map[string]any, want string) {
	t.Helper()
	require.Equal(t, want, body["errorCode"])
}

// newJobViewTestCommand returns a Cobra command wired to a buffer for renderer assertions.
func newJobViewTestCommand() (*cobra.Command, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	return cmd, buf
}

// strconvFormatInt32 keeps generated JSON fixtures explicit about integer formatting.
func strconvFormatInt32(value int32) string {
	return strconv.Itoa(int(value))
}

// resetUpdateJobFlagState restores update-job package flags shared by command tests.
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
