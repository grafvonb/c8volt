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

func TestCancelProcessInstanceSearchScaffold_UsesTempConfigAndCapturesSearchRequest(t *testing.T) {
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

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceSearchScaffoldHelper", cfgPath)

	body := decodeSingleRequestJSON(t, requests)
	filter := body["filter"].(map[string]any)
	startDate := filter["startDate"].(map[string]any)
	endDate := filter["endDate"].(map[string]any)

	require.Equal(t, exitcode.Error, code)
	require.Equal(t, "ACTIVE", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Equal(t, "2026-01-01T00:00:00Z", startDate["$gte"])
	require.Equal(t, "2026-01-31T23:59:59.999999999Z", endDate["$lte"])
	require.Equal(t, true, endDate["$exists"])
	require.Contains(t, output, "no process instance keys provided or found to cancel")
}

func TestCancelProcessInstanceCommand_SearchSelectionUsesDateFiltersAndCancelsMatches(t *testing.T) {
	var requests []string
	var cancelled []string

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
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813711967":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/2251799813711967/cancellation":
			cancelled = append(cancelled, r.URL.Path)
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeCancelProcessInstanceSuccessHelper(t, "TestCancelProcessInstanceCommand_SearchSelectionUsesDateFiltersAndCancelsMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 2)
	require.Equal(t, []string{"/v2/process-instances/2251799813711967/cancellation"}, cancelled)
	body := decodeSingleRequestJSON(t, requests[:1])
	filter := body["filter"].(map[string]any)
	startDate := filter["startDate"].(map[string]any)
	endDate := filter["endDate"].(map[string]any)

	require.Equal(t, "ACTIVE", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Equal(t, "2026-01-01T00:00:00Z", startDate["$gte"])
	require.Equal(t, "2026-01-31T23:59:59.999999999Z", endDate["$lte"])
	require.Equal(t, true, endDate["$exists"])

	var descendantSearch map[string]any
	require.NoError(t, json.Unmarshal([]byte(requests[1]), &descendantSearch))
	descFilter := descendantSearch["filter"].(map[string]any)
	require.Equal(t, "2251799813711967", descFilter["parentProcessInstanceKey"])
	require.NotContains(t, output, "no process instance keys provided or found to cancel")
}

func TestCancelProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatches(t *testing.T) {
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

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatchesHelper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Len(t, requests, 1)
	require.Contains(t, output, "no process instance keys provided or found to cancel")
}

func TestCancelProcessInstanceCommand_RejectsInvalidSearchState(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsInvalidSearchStateHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "invalid value for --state")
}

func TestCancelProcessInstanceCommand_RejectsInvalidDateFilter(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsInvalidDateFilterHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), `invalid value for --start-date-after: "2026-02-30", expected YYYY-MM-DD`)
}

func TestCancelProcessInstanceCommand_RejectsInvalidDateRange(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsInvalidDateRangeHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), `invalid range for --end-date-after and --end-date-before: "2026-02-01" is later than "2026-01-31"`)
}

func TestCancelProcessInstanceCommand_RejectsKeyAndDateFilters(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsKeyAndDateFiltersHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "date filters are only supported for list/search usage and cannot be combined with --key")
}

func TestCancelProcessInstanceCommand_RejectsDateFiltersOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, code := executeCancelProcessInstanceFailureHelper(t, "TestCancelProcessInstanceCommand_RejectsDateFiltersOnV87Helper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "process-instance date filters require Camunda 8.8")
}

func executeCancelProcessInstanceFailureHelper(t *testing.T, helperName string, cfgPath string) (string, int) {
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

func executeCancelProcessInstanceSuccessHelper(t *testing.T, helperName string, cfgPath string) (string, error) {
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

func TestCancelProcessInstanceSearchScaffoldHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31"}

	Execute()
}

func TestCancelProcessInstanceCommand_SearchSelectionUsesDateFiltersAndCancelsMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31", "--auto-confirm", "--no-state-check", "--no-wait"}

	Execute()
}

func TestCancelProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31"}

	Execute()
}

func TestCancelProcessInstanceCommand_RejectsInvalidSearchStateHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "broken", "--bpmn-process-id", "order-process"}

	Execute()
}

func TestCancelProcessInstanceCommand_RejectsInvalidDateFilterHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--start-date-after", "2026-02-30"}

	Execute()
}

func TestCancelProcessInstanceCommand_RejectsInvalidDateRangeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--end-date-after", "2026-02-01", "--end-date-before", "2026-01-31"}

	Execute()
}

func TestCancelProcessInstanceCommand_RejectsKeyAndDateFiltersHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--key", "2251799813711967", "--start-date-after", "2026-01-01"}

	Execute()
}

func TestCancelProcessInstanceCommand_RejectsDateFiltersOnV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "active", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01"}

	Execute()
}
