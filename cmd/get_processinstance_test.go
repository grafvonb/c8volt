package cmd

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Verifies search-mode get process-instance sends the expected filter and pagination request shape.
func TestGetProcessInstanceSearchScaffold_UsesTempConfigAndCapturesSearchRequest(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--state", "active",
		"--count", "5",
	)

	filter := decodeCapturedPISearchFilter(t, requests)
	page := decodeCapturedPISearchPage(t, requests)

	require.Equal(t, "ACTIVE", filter["state"])
	require.EqualValues(t, 5, page["limit"])

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.NotContains(t, got, "error")
}

func TestGetProcessInstanceJSONWithAge_AddsMetaField(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--with-age",
	)

	require.NotEmpty(t, requests)
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	meta, ok := got["meta"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, meta["withAge"])
}

// Verifies get process-instance date filters map to expected API query fields and invalid combinations are rejected.
func TestGetProcessInstanceDateFilterScaffold(t *testing.T) {
	t.Run("start date command coverage", func(t *testing.T) {
		t.Run("lower bound only", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--start-date-after", "2026-01-01",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
			require.NotContains(t, filter["startDate"], "$lte")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})

		t.Run("inclusive range", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--start-date-after", "2026-01-01",
				"--start-date-before", "2026-01-31",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
			requireCapturedPISearchDateBound(t, filter, "startDate", "$lte", "2026-01-31T23:59:59.999999999Z")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})
	})

	t.Run("end date command coverage", func(t *testing.T) {
		t.Run("lower bound only", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--end-date-after", "2026-02-01",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-02-01T00:00:00Z")
			requireCapturedPISearchDateExists(t, filter, "endDate")
			require.NotContains(t, filter["endDate"], "$lte")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})

		t.Run("inclusive range composed with state filter", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--state", "completed",
				"--end-date-after", "2026-02-01",
				"--end-date-before", "2026-03-31",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			require.Equal(t, "COMPLETED", filter["state"])
			requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-02-01T00:00:00Z")
			requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-03-31T23:59:59.999999999Z")
			requireCapturedPISearchDateExists(t, filter, "endDate")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})
	})

	t.Run("invalid date command coverage", func(t *testing.T) {
		t.Run("invalid start-date format exits through shared invalid-input path", func(t *testing.T) {
			cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

			output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceInvalidDateFormatHelper", cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, `invalid value for --start-date-after: "2026-02-30", expected YYYY-MM-DD`)
		})

		t.Run("invalid start-date range exits through shared invalid-input path", func(t *testing.T) {
			cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

			output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceInvalidStartDateRangeHelper", cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, `invalid range for --start-date-after and --start-date-before: "2026-02-01" is later than "2026-01-31"`)
		})

		t.Run("date filters are rejected for direct key lookup", func(t *testing.T) {
			cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

			output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceDateFiltersWithKeyHelper", cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, "date filters are only supported for list/search usage and cannot be combined with --key")
		})
	})
}

func TestGetProcessInstanceRelativeDayFilterScaffold(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	t.Run("start-day range search request uses derived absolute bounds", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServer(t, &requests)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--start-date-older-days", "7",
			"--start-date-newer-days", "30",
		)

		filter := decodeCapturedPISearchFilter(t, requests)

		requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-03-11T00:00:00Z")
		requireCapturedPISearchDateBound(t, filter, "startDate", "$lte", "2026-04-03T23:59:59.999999999Z")

		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.NotContains(t, got, "error")
	})

	t.Run("end-day upper bound search request uses derived absolute bounds", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServer(t, &requests)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--state", "completed",
			"--end-date-newer-days", "14",
		)

		filter := decodeCapturedPISearchFilter(t, requests)

		require.Equal(t, "COMPLETED", filter["state"])
		requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-03-27T00:00:00Z")
		requireCapturedPISearchDateExists(t, filter, "endDate")

		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.NotContains(t, got, "error")
	})
}

func TestGetProcessInstanceRelativeDayValidation(t *testing.T) {
	t.Run("negative relative-day values exit through shared invalid-input path", func(t *testing.T) {
		cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

		output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceNegativeRelativeDayHelper", cfgPath)

		require.Equal(t, exitcode.InvalidArgs, code)
		require.Contains(t, output, "invalid input")
		require.Contains(t, output, "invalid value for --start-date-older-days: -2, expected non-negative integer")
	})

	t.Run("mixed absolute and relative start-date filters are rejected", func(t *testing.T) {
		cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

		output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceMixedAbsoluteAndRelativeDateFiltersHelper", cfgPath)

		require.Equal(t, exitcode.InvalidArgs, code)
		require.Contains(t, output, "invalid input")
		require.Contains(t, output, "start-date absolute and relative day filters cannot be combined")
	})

	t.Run("invalid derived relative-day ranges are rejected", func(t *testing.T) {
		cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

		output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceInvalidRelativeDayRangeHelper", cfgPath)

		require.Equal(t, exitcode.InvalidArgs, code)
		require.Contains(t, output, "invalid input")
		require.Contains(t, output, `invalid range for --start-date-newer-days and --start-date-older-days: "2026-04-03" is later than "2026-03-11"`)
	})

	t.Run("local-day derivation honors the configured day boundary override", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServer(t, &requests)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTestWithEnv(t,
			[]string{testRelativeDayNowEnv + "=2026-04-10T00:30:00+02:00"},
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--start-date-older-days", "0",
		)

		filter := decodeCapturedPISearchFilter(t, requests)

		requireCapturedPISearchDateBound(t, filter, "startDate", "$lte", "2026-04-10T23:59:59.999999999Z")

		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.NotContains(t, got, "error")
	})
}

func TestPopulatePISearchFilterOpts_DerivesRelativeDayBounds(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	flagGetPIStartAfterDays = 7
	flagGetPIStartBeforeDays = 30
	flagGetPIEndAfterDays = 14
	flagGetPIEndBeforeDays = 1

	filter := populatePISearchFilterOpts()

	require.Equal(t, "2026-03-11", filter.StartDateAfter)
	require.Equal(t, "2026-04-03", filter.StartDateBefore)
	require.Equal(t, "2026-04-09", filter.EndDateAfter)
	require.Equal(t, "2026-03-27", filter.EndDateBefore)
}

func TestValidatePISearchFlags_RejectsMixedAbsoluteAndRelativeInputs(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIStartDateAfter = "2026-04-03"
	flagGetPIStartBeforeDays = 7

	err := validatePISearchFlags()

	require.Error(t, err)
	require.Contains(t, err.Error(), "start-date absolute and relative day filters cannot be combined")
}

func TestHasPISearchFilterFlags_WithRelativeDaysOnly(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIStartAfterDays = 72

	require.True(t, hasPISearchFilterFlags())
}

func TestResolvePISearchSize(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	cmd := getProcessInstanceCmd
	resetPISearchCountFlag(t, cmd)

	t.Run("uses shared config default when count flag is unchanged", func(t *testing.T) {
		resetPISearchCountFlag(t, cmd)
		cfg := &config.Config{}
		cfg.App.ProcessInstancePageSize = 250

		require.Equal(t, int32(250), resolvePISearchSize(cmd, cfg))
	})

	t.Run("uses count override when the flag is changed", func(t *testing.T) {
		resetPISearchCountFlag(t, cmd)
		require.NoError(t, cmd.Flags().Set("count", "125"))
		cfg := &config.Config{}
		cfg.App.ProcessInstancePageSize = 250

		require.Equal(t, int32(125), resolvePISearchSize(cmd, cfg))
	})

	t.Run("falls back to repository default for invalid config values", func(t *testing.T) {
		resetProcessInstanceCommandGlobals()
		resetPISearchCountFlag(t, cmd)
		cfg := &config.Config{}
		cfg.App.ProcessInstancePageSize = 0

		require.Equal(t, int32(consts.MaxPISearchSize), resolvePISearchSize(cmd, cfg))
	})
}

func TestGetProcessInstancePagingFlow(t *testing.T) {
	t.Run("uses shared config default and prompts before the next page", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		prompts := []string{}
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			prompts = append(prompts, prompt)
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"get", "process-instance",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 1000, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Len(t, prompts, 1)
		require.Contains(t, prompts[0], "More matching process instances remain")
		require.Contains(t, output, "page size: 1000, current page: 2, total so far: 2, more matches: yes, next step: prompt")
		require.Contains(t, output, "page size: 1000, current page: 1, total so far: 3, more matches: no, next step: complete")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.Contains(t, output, "125")
	})

	t.Run("count override and auto-confirm fetch every page without prompt", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
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
			"--auto-confirm",
			"get", "process-instance",
			"--count", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
		require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
	})

	t.Run("declined continuation reports partial completion summary", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			return ErrCmdAborted
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"get", "process-instance",
			"--count", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: prompt")
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: partial-complete")
		require.Contains(t, output, "detail: stopped after 2 processed process instance(s); remaining matches were left untouched")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
	})

	t.Run("indeterminate overflow stops with warning summary", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"get", "process-instance",
			"--count", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: unknown, next step: warning-stop")
		require.Contains(t, output, "warning: stopped after 2 processed process instance(s) because more matching process instances may remain")
	})
}

func TestPIContinuationHelpers(t *testing.T) {
	t.Run("auto-confirm chooses auto-continue for overflow", func(t *testing.T) {
		page := process.ProcessInstancePage{
			Request:       process.ProcessInstancePageRequest{Size: 50},
			OverflowState: process.ProcessInstanceOverflowStateHasMore,
			Items:         []process.ProcessInstance{{Key: "1"}, {Key: "2"}},
		}

		summary := newPIProgressSummary(page, 2, true)

		require.Equal(t, processInstanceContinuationAutoContinue, summary.ContinuationState)
		require.Equal(t, 50, int(summary.PageSize))
		require.Equal(t, 2, summary.CurrentPageCount)
		require.Equal(t, 2, summary.CumulativeCount)
	})

	t.Run("indeterminate overflow stops with warning", func(t *testing.T) {
		page := process.ProcessInstancePage{
			Request:       process.ProcessInstancePageRequest{Size: 25},
			OverflowState: process.ProcessInstanceOverflowStateIndeterminate,
		}

		summary := newPIProgressSummary(page, 0, false)

		require.Equal(t, processInstanceContinuationWarningStop, summary.ContinuationState)
	})
}

func decodeSingleRequestJSON(t *testing.T, requests []string) map[string]any {
	t.Helper()

	require.Len(t, requests, 1)

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(requests[0]), &got))
	return got
}

func newIPv4Server(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Skipf("local listener unavailable in this environment: %v", err)
	}

	srv := httptest.NewUnstartedServer(handler)
	srv.Listener = listener
	srv.Start()
	t.Cleanup(srv.Close)

	return srv
}

func executeRootForProcessInstanceTest(t *testing.T, args ...string) string {
	t.Helper()

	prevConfirm := confirmCmdOrAbortFn
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm
	t.Cleanup(resetProcessInstanceCommandGlobals)

	root := Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	_, err := root.ExecuteC()
	require.NoError(t, err)

	return buf.String()
}

func executeRootForProcessInstanceTestWithEnv(t *testing.T, env []string, args ...string) string {
	t.Helper()

	prevNow := relativeDayNow
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	for _, kv := range env {
		key, value, ok := strings.Cut(kv, "=")
		require.True(t, ok)
		prevValue, hadValue := os.LookupEnv(key)
		require.NoError(t, os.Setenv(key, value))
		t.Cleanup(func() {
			if hadValue {
				require.NoError(t, os.Setenv(key, prevValue))
				return
			}
			require.NoError(t, os.Unsetenv(key))
		})
	}
	applyRelativeDayNowOverrideFromEnv(t)

	return executeRootForProcessInstanceTest(t, args...)
}

func resetProcessInstanceCommandGlobals() {
	flagCancelPIKeys = nil
	flagDeletePIKeys = nil
	flagGetPIKeys = nil
	flagGetPIBpmnProcessID = ""
	flagGetPIProcessVersion = 0
	flagGetPIProcessVersionTag = ""
	flagGetPIProcessDefinitionKey = ""
	flagGetPIStartDateAfter = ""
	flagGetPIStartDateBefore = ""
	flagGetPIEndDateAfter = ""
	flagGetPIEndDateBefore = ""
	flagGetPIStartAfterDays = -1
	flagGetPIStartBeforeDays = -1
	flagGetPIEndAfterDays = -1
	flagGetPIEndBeforeDays = -1
	flagGetPIWithAge = false
	flagGetPIState = "all"
	flagGetPIParentKey = ""
	flagGetPISize = consts.MaxPISearchSize
	flagGetPIRootsOnly = false
	flagGetPIChildrenOnly = false
	flagGetPIOrphanChildrenOnly = false
	flagGetPIIncidentsOnly = false
	flagGetPINoIncidentsOnly = false
	flagCmdAutoConfirm = false
	flagVerbose = false
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagNoWait = false
	flagForce = false
	flagNoStateCheck = false
	flagWorkers = 0
	flagNoWorkerLimit = false
	flagFailFast = false
	confirmCmdOrAbortFn = confirmCmdOrAbort
}

func resetPISearchCountFlag(t *testing.T, cmd *cobra.Command) {
	t.Helper()

	flag := cmd.Flags().Lookup("count")
	require.NotNil(t, flag)
	require.NoError(t, flag.Value.Set("1000"))
	flag.Changed = false
}

func executeProcessInstanceFailureHelper(t *testing.T, helperName string, cfgPath string) (string, int) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run="+helperName)
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
		testRelativeDayNowEnv+"="+cancelDeleteRelativeDayNow,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	return string(output), exitErr.ExitCode()
}

// Helper-process entrypoint for negative relative-day validation.
func TestGetProcessInstanceNegativeRelativeDayHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-older-days", "-2"}

	Execute()
}

// Helper-process entrypoint for mixed absolute-plus-relative start-date validation.
func TestGetProcessInstanceMixedAbsoluteAndRelativeDateFiltersHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-after", "2026-04-03", "--start-date-newer-days", "7"}

	Execute()
}

// Helper-process entrypoint for invalid relative-day range validation.
func TestGetProcessInstanceInvalidRelativeDayRangeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-older-days", "30", "--start-date-newer-days", "7"}

	Execute()
}

// Helper-process entrypoint for invalid date format validation.
func TestGetProcessInstanceInvalidDateFormatHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-after", "2026-02-30"}

	Execute()
}

// Helper-process entrypoint for invalid start-date range validation.
func TestGetProcessInstanceInvalidStartDateRangeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-after", "2026-02-01", "--start-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for key-and-date-filter exclusivity validation.
func TestGetProcessInstanceDateFiltersWithKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "2251799813711967", "--start-date-after", "2026-01-01"}

	Execute()
}
