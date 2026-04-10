package cmd

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/exitcode"
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

	require.Equal(t, "2026-04-03", filter.StartDateAfter)
	require.Equal(t, "2026-03-11", filter.StartDateBefore)
	require.Equal(t, "2026-03-27", filter.EndDateAfter)
	require.Equal(t, "2026-04-09", filter.EndDateBefore)
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

	resetProcessInstanceCommandGlobals()
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

func resetProcessInstanceCommandGlobals() {
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
	flagGetPIState = "all"
	flagGetPIParentKey = ""
	flagGetPISize = consts.MaxPISearchSize
	flagGetPIRootsOnly = false
	flagGetPIChildrenOnly = false
	flagGetPIOrphanChildrenOnly = false
	flagGetPIIncidentsOnly = false
	flagGetPINoIncidentsOnly = false
}

func executeProcessInstanceFailureHelper(t *testing.T, helperName string, cfgPath string) (string, int) {
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

// Helper-process entrypoint for invalid date format validation.
func TestGetProcessInstanceInvalidDateFormatHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

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

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "2251799813711967", "--start-date-after", "2026-01-01"}

	Execute()
}
