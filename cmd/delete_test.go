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

	require.Equal(t, exitcode.Error, code)
	require.Equal(t, "COMPLETED", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Contains(t, output, "no process instance keys provided or found to delete")
}

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

func TestDeleteProcessInstanceSearchScaffoldHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--auto-confirm"}

	Execute()
}

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
