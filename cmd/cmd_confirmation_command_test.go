// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const (
	confirmationCommandScenarioEnv = "C8VOLT_CONFIRMATION_COMMAND_SCENARIO"
	confirmationCommandCaptureEnv  = "C8VOLT_CONFIRMATION_COMMAND_CAPTURE"
)

// TestConfirmationCommand verifies a mutation command honors configured and inherited stderr without changing decisions.
func TestConfirmationCommand(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		runConfirmationCommandHelper(t)
		os.Exit(0)
	}

	tests := []struct {
		name             string
		scenario         string
		response         string
		wantExitCode     int
		wantMutation     bool
		wantResultOutput bool
	}{
		{
			name:             "configured command stderr accepts",
			scenario:         "configured-accept",
			response:         "y",
			wantMutation:     true,
			wantResultOutput: true,
		},
		{
			name:         "inherited stderr declines",
			scenario:     "inherited-decline",
			response:     "no",
			wantExitCode: exitcode.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configuredCapturePath := filepath.Join(t.TempDir(), "configured-stderr.txt")
			var requests testx.SafeSlice[string]
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Append(r.Method + " " + r.URL.Path)
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/v2/resources/2251799813685255/deletion", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"resourceKey":"2251799813685255","batchOperation":{"batchOperationKey":"batch-2251799813685255","batchOperationType":"DELETE_PROCESS_DEFINITION"}}`))
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
				ScopeTestName: "TestConfirmationCommand",
				Env: map[string]string{
					"C8VOLT_TEST_CONFIG":           cfgPath,
					confirmationCommandScenarioEnv: tt.scenario,
					confirmationCommandCaptureEnv:  configuredCapturePath,
				},
				Exchanges: []testx.CmdTerminalExchange{{
					Prompt:   confirmTerminalPrompt,
					Response: tt.response,
				}},
				Timeout: 3 * time.Second,
			})

			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			require.True(t, result.Supported, result.UnsupportedReason)
			if tt.wantExitCode == 0 {
				require.NoError(t, result.Err, result.Stderr)
			} else {
				var exitErr *exec.ExitError
				require.ErrorAs(t, result.Err, &exitErr)
				require.Equal(t, tt.wantExitCode, exitErr.ExitCode())
			}
			require.Equal(t, tt.wantMutation, len(requests.Snapshot()) == 1)
			require.Contains(t, result.Stderr, confirmTerminalPrompt)
			require.Equal(t, 1, strings.Count(result.Stderr, confirmTerminalPrompt))
			require.NotContains(t, result.Stdout, confirmTerminalPrompt)
			if tt.wantResultOutput {
				require.Contains(t, result.Stdout, `"outcome": "accepted"`)
				require.Contains(t, result.Stdout, "2251799813685255")
				configuredCapture, err := os.ReadFile(configuredCapturePath)
				require.NoError(t, err)
				require.Contains(t, string(configuredCapture), confirmTerminalPrompt)
			} else {
				require.NotContains(t, result.Stdout, `"outcome": "accepted"`)
			}
		})
	}
}

// runConfirmationCommandHelper executes the real delete command with either a leaf-configured or inherited stderr writer.
func runConfirmationCommandHelper(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetOut(os.Stdout)
	deleteProcessDefinitionCmd.SetErr(nil)

	if os.Getenv(confirmationCommandScenarioEnv) == "configured-accept" {
		configuredCapture, err := os.Create(os.Getenv(confirmationCommandCaptureEnv))
		require.NoError(t, err)
		defer configuredCapture.Close()
		root.SetErr(io.MultiWriter(os.Stderr, configuredCapture))
	} else {
		root.SetErr(nil)
	}
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"--json",
		"delete", "process-definition",
		"--key", "2251799813685255",
		"--no-state-check",
		"--no-wait",
	})
	_ = root.Execute()
}
