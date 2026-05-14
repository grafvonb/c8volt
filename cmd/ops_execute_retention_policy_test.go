// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestOpsExecuteRetentionPolicyHelpDocumentsCommand(t *testing.T) {
	output := executeRootForProcessInstanceTest(t, "ops", "execute", "--help")

	assertHelpOutputContainsAll(t, output,
		"Discover predefined operational playbooks",
		"retention-policy",
	)

	commandOutput := executeRootForProcessInstanceTest(t, "ops", "execute", "retention-policy", "--help")

	assertHelpOutputContainsAll(t, commandOutput,
		"Execute process-instance retention cleanup",
		"--retention-days int",
		"./c8volt ops execute retention-policy --retention-days 90 --dry-run",
	)
}

func TestOpsExecuteRetentionPolicyInvalidRetentionDays(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")
	tests := []struct {
		name   string
		helper string
		args   []string
		want   string
	}{
		{
			name:   "missing",
			helper: "TestOpsExecuteRetentionPolicyInvalidRetentionDaysHelper",
			args:   []string{"ops", "execute", "retention-policy"},
			want:   "ops execute retention-policy requires --retention-days",
		},
		{
			name:   "negative",
			helper: "TestOpsExecuteRetentionPolicyInvalidRetentionDaysHelper",
			args:   []string{"ops", "execute", "retention-policy", "--retention-days", "-1"},
			want:   "invalid value for --retention-days: -1, expected non-negative integer",
		},
		{
			name:   "non integer",
			helper: "TestOpsExecuteRetentionPolicyInvalidRetentionDaysHelper",
			args:   []string{"ops", "execute", "retention-policy", "--retention-days", "not-a-number"},
			want:   "invalid argument \"not-a-number\" for \"--retention-days\" flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, tt.helper, map[string]string{
				"C8VOLT_TEST_CONFIG":         cfgPath,
				"C8VOLT_TEST_RETENTION_ARGS": marshalRetentionArgsForEnv(t, tt.args),
			})
			require.Error(t, err)

			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
			require.Contains(t, string(output), "invalid input")
			require.Contains(t, string(output), tt.want)
			require.NotContains(t, string(output), "Usage:")
		})
	}
}

func TestOpsExecuteRetentionPolicyInvalidRetentionDaysHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_RETENTION_ARGS")), &args); err != nil {
		t.Fatalf("invalid helper args: %v", err)
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs(append([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG")}, args...))
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		handleBootstrapError(root, err)
	}
}

func marshalRetentionArgsForEnv(t *testing.T, args []string) string {
	t.Helper()

	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}
