package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestBypassRootBootstrap_TreatsCompletionCommandsAsSharedUtilitySeam(t *testing.T) {
	require.True(t, bypassRootBootstrap(&cobra.Command{Use: "__complete"}))
	require.True(t, bypassRootBootstrap(&cobra.Command{Use: "__completeNoDesc"}))
	require.True(t, bypassRootBootstrap(&cobra.Command{Use: "completion"}))
	require.False(t, bypassRootBootstrap(&cobra.Command{Use: "get"}))
}

func TestCompletionCommandsBypassBootstrapWithoutConfig(t *testing.T) {
	output := executeCompletionForTest(t, "walk", "process-instance", "--mode", "")

	require.Contains(t, output, "parent")
	require.Contains(t, output, "children")
	require.Contains(t, output, "family")
	require.NotContains(t, output, "configuration is invalid")
	require.NotContains(t, output, "Usage:")
}

func TestRootCompletion_TopLevelSuggestionsStayReadable(t *testing.T) {
	output := executeCompletionForTest(t, "")

	require.Contains(t, output, "get\tGet resources\n")
	require.Contains(t, output, "walk\tTraverse (walk) the parent/child graph of resource type\n")
	require.NotContains(t, output, "\ncompletion\n")
	require.NotContains(t, output, "__complete")
	require.NotContains(t, output, "Usage:")
	require.NotContains(t, output, "Get resources such as process definitions or process instances.")
}

func TestRootCompletion_PartialTopLevelSuggestionsStayReadable(t *testing.T) {
	output := executeCompletionForTest(t, "g")

	require.Contains(t, output, "get\tGet resources\n")
	require.NotContains(t, output, "Usage:")
	require.NotContains(t, output, "Get resources such as process definitions or process instances.")
}

func TestNestedCompletion_SubcommandsStayUserFacing(t *testing.T) {
	output := executeCompletionForTest(t, "get", "")

	require.Contains(t, output, "process-definition\tGet deployed process definitions\n")
	require.Contains(t, output, "process-instance\tGet process instances\n")
	require.NotContains(t, output, "\ncompletion\n")
	require.NotContains(t, output, "__complete")
	require.NotContains(t, output, "Usage:")
	require.NotContains(t, output, "Get resources such as process definitions or process instances.")
}

func TestCompletionSuggestionsWithoutDescriptionsStayClean(t *testing.T) {
	output := executeCompletionForTest(t, "walk", "process-instance", "--mode", "")

	require.Contains(t, output, "parent\n")
	require.Contains(t, output, "children\n")
	require.Contains(t, output, "family\n")
	require.NotContains(t, output, "\t")
	require.NotContains(t, output, "Usage:")
}

func TestCompletionCommand_ZshUsesDescriptionBearingCompletionRequests(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")
	cmd := exec.Command(os.Args[0], "-test.run=TestCompletionCommand_ZshUsesDescriptionBearingCompletionRequestsHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err)

	require.Contains(t, string(output), "__complete")
	require.NotContains(t, string(output), "__completeNoDesc")
}

func TestRunProcessInstanceCommand_RejectsMutuallyExclusiveDefinitionFlags(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	cmd := exec.Command(os.Args[0], "-test.run=TestRunProcessInstanceCommand_RejectsMutuallyExclusiveDefinitionFlagsHelper")
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
	require.Contains(t, string(output), "flags --pd-key and --bpmn-process-id are mutually exclusive")
}

func TestDeployProcessDefinitionCommand_RejectsRepeatedStdinFile(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	cmd := exec.Command(os.Args[0], "-test.run=TestDeployProcessDefinitionCommand_RejectsRepeatedStdinFileHelper")
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
	require.Contains(t, string(output), "only one '-' (stdin) allowed")
}

func TestCancelProcessInstanceCommand_RejectsInvalidSearchState(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	cmd := exec.Command(os.Args[0], "-test.run=TestCancelProcessInstanceCommand_RejectsInvalidSearchStateHelper")
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
	require.Contains(t, string(output), "invalid value for --state")
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

func TestWalkProcessInstanceCommand_RejectsInvalidMode(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	cmd := exec.Command(os.Args[0], "-test.run=TestWalkProcessInstanceCommand_RejectsInvalidModeHelper")
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
	require.Contains(t, string(output), "invalid --mode")
}

func TestEmbedExportCommand_RequiresSelection(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	cmd := exec.Command(os.Args[0], "-test.run=TestEmbedExportCommand_RequiresSelectionHelper")
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
	require.Contains(t, string(output), "either --all or at least one --file is required")
}

func TestConfigShowCommand_UsesSharedFailureModelForInvalidEffectiveConfig(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestConfigShowCommand_UsesSharedFailureModelForInvalidEffectiveConfigHelper")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "local precondition failed")
	require.Contains(t, string(output), "configuration is invalid")
}

func TestRunProcessInstanceCommand_ConflictUsesConflictExitCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-instances", r.URL.Path)
		http.Error(w, "already exists", http.StatusConflict)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	cmd := exec.Command(os.Args[0], "-test.run=TestRunProcessInstanceCommand_ConflictUsesConflictExitCodeHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Conflict, exitErr.ExitCode())
	require.Contains(t, string(output), "conflict")
	require.Contains(t, string(output), "running process instance(s)")
}

func TestCompletionCommand_ZshUsesDescriptionBearingCompletionRequestsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "completion", "zsh"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestRunProcessInstanceCommand_RejectsMutuallyExclusiveDefinitionFlagsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "run", "process-instance", "--pd-key", "2251799813685255", "--bpmn-process-id", "order-process"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestDeployProcessDefinitionCommand_RejectsRepeatedStdinFileHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "deploy", "process-definition", "--file", "-", "--file", "-"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestCancelProcessInstanceCommand_RejectsInvalidSearchStateHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "cancel", "process-instance", "--state", "broken", "--bpmn-process-id", "order-process"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
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

func TestWalkProcessInstanceCommand_RejectsInvalidModeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "walk", "process-instance", "--key", "2251799813685255", "--mode", "broken"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestEmbedExportCommand_RequiresSelectionHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "embed", "export"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestConfigShowCommand_UsesSharedFailureModelForInvalidEffectiveConfigHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevValidate := flagShowConfigValidate
	prevTemplate := flagShowConfigTemplate
	t.Cleanup(func() {
		flagShowConfigValidate = prevValidate
		flagShowConfigTemplate = prevTemplate
	})

	cfg := config.New()
	flagShowConfigValidate = true
	flagShowConfigTemplate = false

	configShowCmd.SetContext(cfg.ToContext(context.Background()))
	configShowCmd.SetOut(os.Stdout)
	configShowCmd.SetErr(os.Stderr)
	configShowCmd.Run(configShowCmd, nil)
}

func TestRunProcessInstanceCommand_ConflictUsesConflictExitCodeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "run", "process-instance", "--bpmn-process-id", "order-process", "--no-wait"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}
