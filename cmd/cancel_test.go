package cmd

import (
	"io"
	"net/http"
	"os"
	"os/exec"
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

func TestCancelProcessInstanceSearchScaffoldHelper(t *testing.T) {
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
