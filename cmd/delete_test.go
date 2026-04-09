package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/stretchr/testify/require"
)

// Verifies search-mode deletion builds the expected date-filtered search request before failing on empty matches.
func TestDeleteProcessInstanceSearchScaffold_UsesTempConfigAndCapturesSearchRequest(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-instances/search", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceSearchScaffoldHelper", cfgPath)

	body := decodeSingleRequestJSON(t, requests)
	filter := body["filter"].(map[string]any)
	startDate := filter["startDate"].(map[string]any)
	endDate := filter["endDate"].(map[string]any)

	require.Equal(t, exitcode.Error, code)
	require.Equal(t, "COMPLETED", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Equal(t, "2026-01-01T00:00:00Z", startDate["$gte"])
	require.Equal(t, "2026-01-31T23:59:59.999999999Z", endDate["$lte"])
	require.Equal(t, true, endDate["$exists"])
	require.Contains(t, output, "no process instance keys provided or found to delete")
}

// Verifies reversed date ranges are rejected when the after-bound is later than the before-bound.
func TestDeleteProcessInstanceCommand_RejectsInvalidDateFilter(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsInvalidDateFilterHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), `invalid range for --end-date-after and --end-date-before: "2026-02-01" is later than "2026-01-31"`)
}

// Verifies invalid date literals for date flags are rejected with a clear YYYY-MM-DD validation error.
func TestDeleteProcessInstanceCommand_RejectsInvalidDateValue(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsInvalidDateValueHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), `invalid value for --start-date-after: "2026-02-30", expected YYYY-MM-DD`)
}

// Verifies date filters cannot be combined with direct key lookup mode.
func TestDeleteProcessInstanceCommand_RejectsKeyAndDateFilters(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsKeyAndDateFiltersHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "date filters are only supported for list/search usage and cannot be combined with --key")
}

// Verifies process-instance date filters are rejected for Camunda 8.7 where the capability is unsupported.
func TestDeleteProcessInstanceCommand_RejectsDateFiltersOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsDateFiltersOnV87Helper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "process-instance date filters require Camunda 8.8")
}

// Verifies date-filtered search selection deletes matched instances and preserves descendant lookup behavior.
func TestDeleteProcessInstanceCommand_SearchSelectionUsesDateFiltersAndDeletesMatches(t *testing.T) {
	var requests []string
	var deleted []string

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			var searchBody map[string]any
			require.NoError(t, json.Unmarshal(body, &searchBody))

			filter, _ := searchBody["filter"].(map[string]any)
			parentKey, _ := filter["parentProcessInstanceKey"].(string)

			w.Header().Set("Content-Type", "application/json")
			if parentKey == "2251799813711967" {
				_, _ = w.Write([]byte(`{"items":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-01-03T18:00:00Z","endDate":"2026-01-12T08:30:00Z","state":"COMPLETED","tenantId":"tenant"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813711967":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-01-03T18:00:00Z","endDate":"2026-01-12T08:30:00Z","state":"COMPLETED","tenantId":"tenant"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/process-instances/2251799813711967":
			deleted = append(deleted, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceCommand_SearchSelectionUsesDateFiltersAndDeletesMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(requests), 2)
	require.Equal(t, []string{"/v1/process-instances/2251799813711967"}, deleted)
	body := decodeSingleRequestJSON(t, requests[:1])
	filter := body["filter"].(map[string]any)
	startDate := filter["startDate"].(map[string]any)
	endDate := filter["endDate"].(map[string]any)

	require.Equal(t, "COMPLETED", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Equal(t, "2026-01-01T00:00:00Z", startDate["$gte"])
	require.Equal(t, "2026-01-31T23:59:59.999999999Z", endDate["$lte"])
	require.Equal(t, true, endDate["$exists"])

	var descendantSearch map[string]any
	require.NoError(t, json.Unmarshal([]byte(requests[len(requests)-1]), &descendantSearch))
	descFilter := descendantSearch["filter"].(map[string]any)
	require.Equal(t, "2251799813711967", descFilter["parentProcessInstanceKey"])
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

// Verifies delete exits with an error when a date-filtered search returns no process instances.
func TestDeleteProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatches(t *testing.T) {
	var requests []string

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-instances/search", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatchesHelper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Len(t, requests, 1)
	require.Contains(t, output, "no process instance keys provided or found to delete")
}

// Verifies delete process-definition requires either --key or --bpmn-process-id as a target selector.
func TestDeleteProcessDefinitionCommand_RequiresTargetSelector(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	cmd := exec.Command(os.Args[0], "-test.run=TestDeleteProcessDefinitionCommand_RequiresTargetSelectorHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "either --key or --bpmn-process-id must be provided")
}

func executeDeleteProcessInstanceFailureHelper(t *testing.T, helperName string, cfgPath string) (string, int) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run="+helperName)
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	return string(output), exitErr.ExitCode()
}

func executeDeleteProcessInstanceSuccessHelper(t *testing.T, helperName string, cfgPath string) (string, error) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run="+helperName)
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	if strings.Contains(string(output), "PASS") {
		return "", nil
	}
	return string(output), nil
}

// Helper-process entrypoint for the search scaffold failure test.
func TestDeleteProcessInstanceSearchScaffoldHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31", "--auto-confirm"}

	Execute()
}

// Helper-process entrypoint for invalid date range validation.
func TestDeleteProcessInstanceCommand_RejectsInvalidDateFilterHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--end-date-after", "2026-02-01", "--end-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for invalid date format validation.
func TestDeleteProcessInstanceCommand_RejectsInvalidDateValueHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--start-date-after", "2026-02-30"}

	Execute()
}

// Helper-process entrypoint for key-and-date-filter exclusivity validation.
func TestDeleteProcessInstanceCommand_RejectsKeyAndDateFiltersHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--key", "2251799813711967", "--start-date-after", "2026-01-01"}

	Execute()
}

// Helper-process entrypoint for version capability validation on Camunda 8.7.
func TestDeleteProcessInstanceCommand_RejectsDateFiltersOnV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--auto-confirm"}

	Execute()
}

// Helper-process entrypoint for the successful search-select-and-delete flow test.
func TestDeleteProcessInstanceCommand_SearchSelectionUsesDateFiltersAndDeletesMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31", "--auto-confirm", "--no-state-check", "--no-wait"}

	Execute()
}

// Helper-process entrypoint for the no-matches failure test.
func TestDeleteProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for delete process-definition target-selector validation.
func TestDeleteProcessDefinitionCommand_RequiresTargetSelectorHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-definition"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}
