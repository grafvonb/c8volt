// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os/exec"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const cancelDeleteRelativeDayNow = "2026-04-10T12:00:00Z"

// TestCancelCommand_CommandLocalBackoffTimeoutEnvOverridesProfileAndConfig verifies command-local timeout precedence.
func TestCancelCommand_CommandLocalBackoffTimeoutEnvOverridesProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_BACKOFF_TIMEOUT", "27s")

	cfg := resolveCommandConfigForTest(t, cancelCmd, writeBackoffPrecedenceConfig(t), nil)

	require.Equal(t, 27*time.Second, cfg.App.Backoff.Timeout)
}

// TestCancelHelp_DocumentsConfirmationAndNoWaitSemantics verifies cancel help explains confirmation and wait behavior.
func TestCancelHelp_DocumentsConfirmationAndNoWaitSemantics(t *testing.T) {
	assertCommandHelpOutput(t, []string{"cancel"}, []string{"--dry-run", "process-instance"}, nil)
	assertCommandHelpOutput(t, []string{"cancel", "process-instance"}, []string{"affected instance families", "Empty searches finish without confirmation or mutation", "--force", "--dry-run"}, nil)
}

// executeCancelProcessInstanceFailureHelper runs a cancel helper subprocess expected to fail.
func executeCancelProcessInstanceFailureHelper(t *testing.T, helperName string, cfgPath string) (string, int) {
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

// executeCancelProcessInstanceSuccessHelper runs a cancel helper subprocess expected to succeed.
func executeCancelProcessInstanceSuccessHelper(t *testing.T, helperName string, cfgPath string) (string, error) {
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
