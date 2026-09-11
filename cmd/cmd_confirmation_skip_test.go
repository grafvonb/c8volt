// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

const confirmationSkipScenarioEnv = "C8VOLT_CONFIRMATION_SKIP_SCENARIO"

// TestConfirmationSkipPolicies proves guarded selector recovery never reads unavailable input or emits either recovery prompt.
func TestConfirmationSkipPolicies(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		runConfirmationSkipPolicyHelper(t)
		os.Exit(0)
	}

	scenarios := []string{
		"auto-confirm-visible",
		"json-near-matches",
		"keys-only-visible",
		"supported-automation-near-matches",
		"unsupported-automation-visible",
		"non-terminal-near-matches",
	}
	for _, scenario := range scenarios {
		t.Run(scenario, func(t *testing.T) {
			stdout, stderr, err := runConfirmationSkipPolicySubprocess(t, scenario, time.Second)

			require.NoError(t, err)
			require.Equal(t, "skipped=true\n", stdout)
			require.Empty(t, stderr)
			require.NotContains(t, stdout+stderr, "List visible process definitions?")
			require.NotContains(t, stdout+stderr, "List matching process definitions?")
		})
	}
}

// runConfirmationSkipPolicySubprocess holds stdin open so an accidental recovery read is terminated by the deadline.
func runConfirmationSkipPolicySubprocess(t *testing.T, scenario string, timeout time.Duration) (string, string, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^"+regexp.QuoteMeta("TestConfirmationSkipPolicies")+"$")
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	defer stdin.Close()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		testx.CmdSubprocessNameEnv+"=TestConfirmationSkipPolicies",
		confirmationSkipScenarioEnv+"="+scenario,
	)
	err = cmd.Run()
	if ctx.Err() != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("confirmation skip subprocess timed out: %w", ctx.Err())
	}
	return stdout.String(), stderr.String(), err
}

// runConfirmationSkipPolicyHelper exercises caller eligibility before a seam that would block on stdin if reached.
func runConfirmationSkipPolicyHelper(t *testing.T) {
	require.Equal(t, "TestConfirmationSkipPolicies", os.Getenv(testx.CmdSubprocessNameEnv))
	resetProcessInstanceCommandGlobals()
	flagCmdAutoConfirm = false
	flagViewAsJson = false
	flagViewKeysOnly = false
	processDefinitionSelectorInteractiveTerminalFn = func() bool { return true }
	confirmProcessDefinitionSelectorListVisibleFn = func(promptWriter io.Writer, _ bool, prompt string) error {
		_, _ = fmt.Fprint(promptWriter, formatConfirmationPrompt(prompt, "[Y/n]"))
		_, err := bufio.NewReader(os.Stdin).ReadString('\n')
		return err
	}

	scenario := os.Getenv(confirmationSkipScenarioEnv)
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.Flags().Bool("automation", false, "")

	switch {
	case strings.HasPrefix(scenario, "auto-confirm"):
		flagCmdAutoConfirm = true
	case strings.HasPrefix(scenario, "json"):
		flagViewAsJson = true
	case strings.HasPrefix(scenario, "keys-only"):
		flagViewKeysOnly = true
	case strings.HasPrefix(scenario, "supported-automation"):
		require.NoError(t, cmd.Flags().Set("automation", "true"))
		setAutomationSupport(cmd, AutomationSupportFull, "safe for unattended execution")
		require.NoError(t, requireAutomationSupport(cmd))
	case strings.HasPrefix(scenario, "unsupported-automation"):
		require.NoError(t, cmd.Flags().Set("automation", "true"))
		setAutomationSupport(cmd, AutomationSupportUnsupported, "interactive only")
		require.Error(t, requireAutomationSupport(cmd))
	case strings.HasPrefix(scenario, "non-terminal"):
		processDefinitionSelectorInteractiveTerminalFn = func() bool { return false }
	default:
		t.Fatalf("unknown confirmation skip scenario %q", scenario)
	}

	result := processDefinitionSelectorValidationResult{MissingBpmnProcessIDs: []string{"missing"}}
	if strings.Contains(scenario, "near-matches") {
		result.NearMatchesByBpmnProcessID = map[string]process.ProcessDefinitions{
			"missing": {Items: []process.ProcessDefinition{{Key: "pd-near"}}},
		}
	}
	cli := stubProcessAPI{
		searchProcessDefinitions: func(context.Context, process.ProcessDefinitionFilter, ...foptions.FacadeOption) (process.ProcessDefinitions, error) {
			return process.ProcessDefinitions{Items: []process.ProcessDefinition{{Key: "pd-visible"}}}, nil
		},
	}
	if processDefinitionSelectorPromptAllowed(cmd) {
		_ = processDefinitionSelectorRecovery(cmd, cli, result)
	}
	fmt.Fprintln(os.Stdout, "skipped=true")
}

// TestConfirmationRedirectedStdoutSkip uses terminal stdin to prove redirected stdout still suppresses selector recovery.
func TestConfirmationRedirectedStdoutSkip(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		runConfirmationRedirectedStdoutSkipHelper(t)
		os.Exit(0)
	}

	result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
		ScopeTestName: "TestConfirmationRedirectedStdoutSkip",
		Timeout:       2 * time.Second,
	})
	if !result.Supported {
		t.Skip(result.UnsupportedReason)
	}
	require.True(t, result.Supported, result.UnsupportedReason)
	require.NoError(t, result.Err)
	require.Equal(t, "terminal-stdin=true\nskipped=true\n", result.Stdout)
	require.Empty(t, result.Stderr)
}

// runConfirmationRedirectedStdoutSkipHelper verifies the real stdin/stdout terminal eligibility check without reading stdin.
func runConfirmationRedirectedStdoutSkipHelper(t *testing.T) {
	require.Equal(t, "TestConfirmationRedirectedStdoutSkip", os.Getenv(testx.CmdSubprocessNameEnv))
	require.True(t, term.IsTerminal(int(os.Stdin.Fd())))
	require.False(t, term.IsTerminal(int(os.Stdout.Fd())))
	fmt.Fprintf(os.Stdout, "terminal-stdin=%t\n", term.IsTerminal(int(os.Stdin.Fd())))

	resetProcessInstanceCommandGlobals()
	processDefinitionSelectorInteractiveTerminalFn = processDefinitionSelectorInteractiveTerminal
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.Flags().Bool("automation", false, "")
	require.False(t, processDefinitionSelectorPromptAllowed(cmd))
	fmt.Fprintln(os.Stdout, "skipped=true")
}
