package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestExpectCommand_CommandLocalBackoffTimeoutEnvOverridesProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_BACKOFF_TIMEOUT", "19s")

	cfg := resolveCommandConfigForTest(t, expectCmd, writeBackoffPrecedenceConfig(t), nil)

	require.Equal(t, 19*time.Second, cfg.App.Backoff.Timeout)
}

func TestExpectHelp_DocumentsWaitVerificationUsage(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"expect"}, []string{
		"Wait for verification targets to reach the expected state",
		"after a state-changing operation",
		"wait contract",
		"./c8volt expect process-instance --key 2251799813711967 --state absent",
	}, nil)
	require.Contains(t, output, "process-instance")

	output = assertCommandHelpOutput(t, []string{"expect", "process-instance"}, []string{
		"Use this read-only command after `run`, `cancel`, or `delete`",
		"waits until each keyed process instance reaches one of the requested states",
		"Use --json when another tool needs the final wait report",
		"`--automation` remains unsupported",
		"./c8volt run pi --bpmn-process-id order-process --no-wait --json",
	}, nil)
	require.Contains(t, output, "--state")
}

// Verifies expect process-instance rejects unsupported state values through invalid-input handling.
func TestExpectProcessInstanceCommand_RejectsInvalidStates(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	cmd := exec.Command(os.Args[0], "-test.run=TestExpectProcessInstanceCommand_RejectsInvalidStatesHelper")
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
	require.Contains(t, string(output), "error parsing states")
}

func TestExpectProcessInstanceCommand_JSONInvalidStateUsesEnvelope(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestExpectProcessInstanceCommand_JSONInvalidStateUsesEnvelopeHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())

	var got map[string]any
	require.NoError(t, json.Unmarshal(output, &got))
	require.Equal(t, string(OutcomeInvalid), got["outcome"])
	require.Equal(t, "expect process-instance", got["command"])
}

func TestExpectProcessInstanceCommand_RejectsAutomationMode(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestExpectProcessInstanceCommand_RejectsAutomationModeHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "expect process-instance does not support --automation")
}

// Helper-process entrypoint for invalid expect-state validation.
func TestExpectProcessInstanceCommand_RejectsInvalidStatesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "expect", "process-instance", "--key", "2251799813685255", "--state", "broken"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestExpectProcessInstanceCommand_JSONInvalidStateUsesEnvelopeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--json", "expect", "process-instance", "--key", "2251799813685255", "--state", "broken"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestExpectProcessInstanceCommand_RejectsAutomationModeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--automation", "expect", "process-instance", "--key", "2251799813685255", "--state", "completed"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}
