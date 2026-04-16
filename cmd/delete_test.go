package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestDeleteCommand_CommandLocalBackoffTimeoutFlagOverridesEnvProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_BACKOFF_TIMEOUT", "24s")

	cfg := resolveCommandConfigForTest(t, deleteCmd, writeBackoffPrecedenceConfig(t), func(cmd *cobra.Command) {
		require.NoError(t, cmd.PersistentFlags().Set("backoff-timeout", "46s"))
	})

	require.Equal(t, 46*time.Second, cfg.App.Backoff.Timeout)
}

// Verifies search-mode deletion builds the expected date-filtered search request and no-ops cleanly on empty matches.
func TestDeleteProcessInstanceSearchScaffold_UsesTempConfigAndCapturesSearchRequest(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceSearchScaffoldHelper", cfgPath)

	filter := decodeCapturedPISearchFilter(t, requests)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Equal(t, "COMPLETED", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-01-31T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

// Verifies reversed date ranges are rejected when the after-bound is later than the before-bound.
func TestDeleteProcessInstanceCommand_RejectsInvalidDateFilter(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsInvalidDateFilterHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, `invalid range for --end-date-after and --end-date-before: "2026-02-01" is later than "2026-01-31"`)
}

// Verifies invalid date literals for date flags are rejected with a clear YYYY-MM-DD validation error.
func TestDeleteProcessInstanceCommand_RejectsInvalidDateValue(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsInvalidDateValueHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, `invalid value for --start-date-after: "2026-02-30", expected YYYY-MM-DD`)
}

// Verifies date filters cannot be combined with direct key lookup mode.
func TestDeleteProcessInstanceCommand_RejectsKeyAndDateFilters(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsKeyAndDateFiltersHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, "date filters are only supported for list/search usage and cannot be combined with --key")
}

// Verifies relative-day filters cannot be combined with direct key lookup mode.
func TestDeleteProcessInstanceCommand_RejectsKeyAndRelativeDayFilters(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsKeyAndRelativeDayFiltersHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, "date filters are only supported for list/search usage and cannot be combined with --key")
}

// Verifies process-instance date filters are rejected for Camunda 8.7 where the capability is unsupported.
func TestDeleteProcessInstanceCommand_RejectsDateFiltersOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsDateFiltersOnV87Helper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "process-instance date filters require Camunda 8.8")
}

// Verifies relative-day process-instance filters are also rejected for Camunda 8.7.
func TestDeleteProcessInstanceCommand_RejectsRelativeDayFiltersOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsRelativeDayFiltersOnV87Helper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "process-instance date filters require Camunda 8.8")
}

// Verifies date-filtered search selection deletes matched instances and preserves descendant lookup behavior.
func TestDeleteProcessInstanceCommand_SearchSelectionUsesDateFiltersAndDeletesMatches(t *testing.T) {
	var requests []string
	var deleted safeSlice[string]

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
			deleted.Append(r.URL.Path)
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
	require.Equal(t, []string{"/v1/process-instances/2251799813711967"}, deleted.Snapshot())
	filter := decodeCapturedPISearchFilter(t, requests[:1])

	require.Equal(t, "COMPLETED", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-01-31T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")

	descendantSearch := decodeCapturedPISearchRequest(t, requests[len(requests)-1])
	descFilter := descendantSearch["filter"].(map[string]any)
	require.Equal(t, "2251799813711967", descFilter["parentProcessInstanceKey"])
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

// Verifies relative-day search selection derives canonical end-date bounds before deleting matches.
func TestDeleteProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndDeletesMatches(t *testing.T) {
	var requests []string
	var deleted safeSlice[string]

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
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-02-10T18:00:00Z","endDate":"2026-03-15T08:30:00Z","state":"COMPLETED","tenantId":"tenant"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813711967":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-02-10T18:00:00Z","endDate":"2026-03-15T08:30:00Z","state":"COMPLETED","tenantId":"tenant"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/process-instances/2251799813711967":
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndDeletesMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(requests), 2)
	require.Equal(t, []string{"/v1/process-instances/2251799813711967"}, deleted.Snapshot())
	filter := decodeCapturedPISearchFilter(t, requests[:1])

	require.Equal(t, "COMPLETED", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-02-09T00:00:00Z")
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-04-03T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")

	descendantSearch := decodeCapturedPISearchRequest(t, requests[len(requests)-1])
	descFilter := descendantSearch["filter"].(map[string]any)
	require.Equal(t, "2251799813711967", descFilter["parentProcessInstanceKey"])
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

// Verifies delete no-ops successfully when a date-filtered search returns no process instances.
func TestDeleteProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatches(t *testing.T) {
	var requests []string

	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

// Verifies a relative-day-only filter is sufficient to trigger search mode.
func TestDeleteProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficient(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficientHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.NotContains(t, output, "either at least one --key is required, or sufficient filtering options")
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

// Verifies paged delete search prompts between pages and continues when confirmations are accepted.
func TestDeleteProcessInstanceCommand_SearchPagingPromptFlow(t *testing.T) {
	var requests []string
	var deleted safeSlice[string]
	searchPage := 0

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"401","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"402","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"403","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodGet && (r.URL.Path == "/v2/process-instances/401" || r.URL.Path == "/v2/process-instances/402" || r.URL.Path == "/v2/process-instances/403"):
			key := r.URL.Path[len("/v2/process-instances/"):]
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"` + key + `","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`))
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/401" || r.URL.Path == "/v1/process-instances/402" || r.URL.Path == "/v1/process-instances/403"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	var prompts []string
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		prompts = append(prompts, prompt)
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--count", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests)
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 0, pages[0]["from"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.ElementsMatch(t, []string{
		"/v1/process-instances/401",
		"/v1/process-instances/402",
		"/v1/process-instances/403",
	}, deleted.Snapshot())
	require.Len(t, prompts, 2)
	require.Contains(t, prompts[0], "You are about to delete 2 process instance(s)")
	require.Contains(t, prompts[1], "Processed 2 process instance(s) on this page (2 requested so far, 2 including dependencies). More matching process instances remain. Continue?")
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: prompt")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
}

// Verifies v8.7 paged delete prompts include dependency-inclusive totals across continuation pages.
func TestDeleteProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotals(t *testing.T) {
	var requests []string
	var deleted safeSlice[string]
	searchPage := 0

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if parentKey, ok := filter["parentKey"]; ok && parentKey != nil {
					parent := int64(parentKey.(float64))
					if parent != 901 && parent != 902 {
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(`{"items":[]}`))
						return
					}
					childKey := "1901"
					if parent == 902 {
						childKey = "1902"
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[{"key":` + childKey + `,"parentKey":` + fmt.Sprintf("%d", parent) + `,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"}]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"key":901,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"},{"key":902,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"}],"total":3}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"key":901,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"},{"key":902,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"},{"key":903,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"}],"total":3}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodGet && (r.URL.Path == "/v1/process-instances/901" || r.URL.Path == "/v1/process-instances/902" || r.URL.Path == "/v1/process-instances/903" || r.URL.Path == "/v1/process-instances/1901" || r.URL.Path == "/v1/process-instances/1902"):
			key := r.URL.Path[len("/v1/process-instances/"):]
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"key":` + key + `,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"}`))
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/901" || r.URL.Path == "/v1/process-instances/902" || r.URL.Path == "/v1/process-instances/903" || r.URL.Path == "/v1/process-instances/1901" || r.URL.Path == "/v1/process-instances/1902"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")
	var prompts []string
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		prompts = append(prompts, prompt)
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	_ = executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--count", "2",
	)

	sizes := decodeCapturedTopLevelPISearchSizes(t, requests)
	require.Equal(t, []float64{2, 4}, sizes)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/1901",
		"/v1/process-instances/1902",
		"/v1/process-instances/901",
		"/v1/process-instances/902",
		"/v1/process-instances/903",
	}, deleted.Snapshot())
	require.Len(t, prompts, 2)
	require.Contains(t, prompts[1], "Processed 2 process instance(s) on this page (2 requested so far, 4 including dependencies). More matching process instances remain. Continue?")
}

// Verifies paged delete search auto-continues without continuation prompts when --auto-confirm is set.
func TestDeleteProcessInstanceCommand_SearchPagingAutoConfirmFlow(t *testing.T) {
	var requests []string
	var deleted safeSlice[string]
	searchPage := 0

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"501","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"502","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"503","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodGet && (r.URL.Path == "/v2/process-instances/501" || r.URL.Path == "/v2/process-instances/502" || r.URL.Path == "/v2/process-instances/503"):
			key := r.URL.Path[len("/v2/process-instances/"):]
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"` + key + `","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`))
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/501" || r.URL.Path == "/v1/process-instances/502" || r.URL.Path == "/v1/process-instances/503"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	promptCalls := 0
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		promptCalls++
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"--auto-confirm",
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--count", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests)
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/501",
		"/v1/process-instances/502",
		"/v1/process-instances/503",
	}, deleted.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
}

// Verifies paged delete reports a partial-completion summary when continuation is aborted.
func TestDeleteProcessInstanceCommand_SearchPagingPartialCompletionSummary(t *testing.T) {
	var requests []string
	var deleted safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"511","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"512","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
		case r.Method == http.MethodGet && (r.URL.Path == "/v2/process-instances/511" || r.URL.Path == "/v2/process-instances/512"):
			key := r.URL.Path[len("/v2/process-instances/"):]
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"` + key + `","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`))
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/511" || r.URL.Path == "/v1/process-instances/512"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	callCount := 0
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		callCount++
		if callCount == 1 {
			return nil
		}
		return ErrCmdAborted
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--count", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests)
	require.Len(t, pages, 1)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/511",
		"/v1/process-instances/512",
	}, deleted.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: partial-complete")
	require.Contains(t, output, "detail: stopped after 2 processed process instance(s); remaining matches were left untouched")
}

// Verifies paged delete emits a warning-stop summary when overflow state is indeterminate.
func TestDeleteProcessInstanceCommand_SearchPagingWarningStopSummary(t *testing.T) {
	var requests []string
	var deleted safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"521","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"522","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{}}`))
		case r.Method == http.MethodGet && (r.URL.Path == "/v2/process-instances/521" || r.URL.Path == "/v2/process-instances/522"):
			key := r.URL.Path[len("/v2/process-instances/"):]
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"` + key + `","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`))
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/521" || r.URL.Path == "/v1/process-instances/522"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error { return nil }
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--count", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests)
	require.Len(t, pages, 1)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/521",
		"/v1/process-instances/522",
	}, deleted.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: unknown, next step: warning-stop")
	require.Contains(t, output, "warning: stopped after 2 processed process instance(s) because more matching process instances may remain")
}

// Verifies direct --key deletion bypasses top-level search pagination logic.
func TestDeleteProcessInstanceCommand_DirectKeyBypassesTopLevelSearchPaging(t *testing.T) {
	var requests []string
	var deleted safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/601":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"601","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/process-instances/601":
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error { return nil }
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	_ = executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"delete", "process-instance",
		"--key", "601",
		"--no-wait",
		"--count", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests)
	require.Empty(t, pages)
	require.Equal(t, []string{"/v1/process-instances/601"}, deleted.Snapshot())
}

// Verifies delete process-definition requires either --key or --bpmn-process-id as a target selector.
func TestDeleteProcessDefinitionCommand_RequiresTargetSelector(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestDeleteProcessDefinitionCommand_RequiresTargetSelectorHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":  cfgPath,
		testRelativeDayNowEnv: cancelDeleteRelativeDayNow,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "either --key or --bpmn-process-id must be provided")
}

// Runs a delete helper subprocess expected to fail and returns combined output with the exit code.
func executeDeleteProcessInstanceFailureHelper(t *testing.T, helperName string, cfgPath string) (string, int) {
	t.Helper()

	output, err := testx.RunCmdSubprocess(t, helperName, map[string]string{
		"C8VOLT_TEST_CONFIG":  cfgPath,
		testRelativeDayNowEnv: cancelDeleteRelativeDayNow,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	return string(output), exitErr.ExitCode()
}

// Runs a delete helper subprocess and returns combined output with the underlying execution error.
func executeDeleteProcessInstanceSuccessHelper(t *testing.T, helperName string, cfgPath string) (string, error) {
	t.Helper()

	output, err := testx.RunCmdSubprocess(t, helperName, map[string]string{
		"C8VOLT_TEST_CONFIG":  cfgPath,
		testRelativeDayNowEnv: cancelDeleteRelativeDayNow,
	})
	out := string(output)
	if err != nil {
		return out, err
	}
	return out, nil
}

// Helper-process entrypoint for the search scaffold failure test.
func TestDeleteProcessInstanceSearchScaffoldHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

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
	applyRelativeDayNowOverrideFromEnv(t)

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
	applyRelativeDayNowOverrideFromEnv(t)

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
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--key", "2251799813711967", "--start-date-after", "2026-01-01"}

	Execute()
}

// Helper-process entrypoint for key-and-relative-day-filter exclusivity validation.
func TestDeleteProcessInstanceCommand_RejectsKeyAndRelativeDayFiltersHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--key", "2251799813711967", "--end-date-newer-days", "7"}

	Execute()
}

// Helper-process entrypoint for version capability validation on Camunda 8.7.
func TestDeleteProcessInstanceCommand_RejectsDateFiltersOnV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--auto-confirm"}

	Execute()
}

// Helper-process entrypoint for relative-day version capability validation on Camunda 8.7.
func TestDeleteProcessInstanceCommand_RejectsRelativeDayFiltersOnV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--end-date-newer-days", "7", "--auto-confirm"}

	Execute()
}

// Helper-process entrypoint for the successful search-select-and-delete flow test.
func TestDeleteProcessInstanceCommand_SearchSelectionUsesDateFiltersAndDeletesMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31", "--auto-confirm", "--no-state-check", "--no-wait"}

	Execute()
}

// Helper-process entrypoint for the successful relative-day search-select-and-delete flow test.
func TestDeleteProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndDeletesMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--end-date-older-days", "7", "--end-date-newer-days", "60", "--auto-confirm", "--no-state-check", "--no-wait"}

	Execute()
}

// Helper-process entrypoint for the no-matches failure test.
func TestDeleteProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for relative-day-only sufficiency validation.
func TestDeleteProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficientHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--end-date-older-days", "72"}

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
