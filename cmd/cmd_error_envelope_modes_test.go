// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const commandErrorEnvelopeModesHelper = "TestCommandErrorEnvelopeModesHelper"

// TestCommandErrorEnvelopeModes verifies corrected failures preserve output
// precedence, quiet and automation behavior, and exit-code suppression.
func TestCommandErrorEnvelopeModes(t *testing.T) {
	var requests atomic.Int32
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/license", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	cfgPath := testx.WriteTestConfig(t, server.URL)

	tests := []struct {
		name        string
		modeArgs    []string
		commandArgs []string
		want        commandErrorEnvelopeExpectation
		wantJSON    bool
		exitCode    int
		requests    int32
	}{
		{
			name:        "validation/json-and-quiet",
			modeArgs:    []string{"--json", "--quiet"},
			commandArgs: []string{"delete", "process-instance", "--state", "completed", "--limit", "0"},
			want: commandErrorEnvelopeExpectation{
				Outcome: "invalid",
				Class:   "invalid_input",
				Command: "delete process-instance",
				Message: "invalid input: invalid flag value: --limit must be positive integer",
			},
			wantJSON: true,
			exitCode: exitcode.InvalidArgs,
		},
		{
			name:        "validation/json-precedes-keys-only",
			modeArgs:    []string{"--json", "--keys-only"},
			commandArgs: []string{"delete", "process-instance", "--state", "completed", "--limit", "0"},
			want: commandErrorEnvelopeExpectation{
				Outcome: "invalid",
				Class:   "invalid_input",
				Command: "delete process-instance",
				Message: "invalid input: invalid flag value: --limit must be positive integer",
			},
			wantJSON: true,
			exitCode: exitcode.InvalidArgs,
		},
		{
			name:        "validation/supported-automation-with-json",
			modeArgs:    []string{"--automation", "--json"},
			commandArgs: []string{"delete", "process-instance", "--state", "completed", "--limit", "0"},
			want: commandErrorEnvelopeExpectation{
				Outcome: "invalid",
				Class:   "invalid_input",
				Command: "delete process-instance",
				Message: "invalid input: invalid flag value: --limit must be positive integer",
			},
			wantJSON: true,
			exitCode: exitcode.InvalidArgs,
		},
		{
			name:        "validation/keys-only",
			modeArgs:    []string{"--keys-only"},
			commandArgs: []string{"delete", "process-instance", "--state", "completed", "--limit", "0"},
			want: commandErrorEnvelopeExpectation{
				Message: "invalid input: invalid flag value: --limit must be positive integer",
			},
			exitCode: exitcode.InvalidArgs,
		},
		{
			name:        "validation/quiet",
			modeArgs:    []string{"--quiet"},
			commandArgs: []string{"delete", "process-instance", "--state", "completed", "--limit", "0"},
			want: commandErrorEnvelopeExpectation{
				Message: "invalid input: invalid flag value: --limit must be positive integer",
			},
			exitCode: exitcode.InvalidArgs,
		},
		{
			name:        "runtime/json-and-quiet",
			modeArgs:    []string{"--json", "--quiet"},
			commandArgs: []string{"get", "cluster", "license"},
			want: commandErrorEnvelopeExpectation{
				Outcome: "failed",
				Class:   "malformed_response",
				Command: "get cluster license",
				Message: "get cluster license: malformed response: malformed response: 200 OK but empty payload; body=",
			},
			wantJSON: true,
			exitCode: exitcode.Error,
			requests: 1,
		},
		{
			name:        "runtime/quiet",
			modeArgs:    []string{"--quiet"},
			commandArgs: []string{"get", "cluster", "license"},
			want: commandErrorEnvelopeExpectation{
				Message: "get cluster license: malformed response: malformed response: 200 OK but empty payload; body=",
			},
			exitCode: exitcode.Error,
			requests: 1,
		},
		{
			name:        "unsupported-automation/guard-precedes-stdin-validation",
			modeArgs:    []string{"--automation", "--json"},
			commandArgs: []string{"expect", "process-instance", "--state", "active", "-"},
			want: commandErrorEnvelopeExpectation{
				Outcome: "failed",
				Class:   "unsupported",
				Command: "expect process-instance",
				Message: "unsupported capability: expect process-instance does not support --automation: automation mode is not supported for wait commands; remove --automation or inspect `c8volt capabilities --json` for supported commands",
			},
			wantJSON: true,
			exitCode: exitcode.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, suppressExit := range []bool{false, true} {
				name := "classified-exit"
				wantExitCode := tt.exitCode
				if suppressExit {
					name = "suppressed-exit"
					wantExitCode = 0
				}
				t.Run(name, func(t *testing.T) {
					before := requests.Load()
					stdout, stderr := runCommandErrorEnvelopeSubprocess(t, commandErrorEnvelopeModesHelper, "", map[string]string{
						"C8VOLT_TEST_CONFIG":       cfgPath,
						"C8VOLT_TEST_MODE_ARGS":    marshalRootArgsForEnv(t, tt.modeArgs),
						"C8VOLT_TEST_COMMAND_ARGS": marshalRootArgsForEnv(t, tt.commandArgs),
						"C8VOLT_TEST_NO_ERR_CODES": boolEnv(suppressExit),
					}, "filter: state=ACTIVE\n", wantExitCode)

					if tt.wantJSON {
						assertCommandErrorEnvelope(t, stdout, stderr, tt.want)
					} else {
						assertHumanCommandError(t, stdout, stderr, tt.want.Message)
					}
					require.Equal(t, before+tt.requests, requests.Load(), "mode handling must not add or bypass command work")
				})
			}
		})
	}
}

// TestCommandErrorEnvelopeModesHelper executes the selected real command with
// root output flags under exact helper-process scoping.
func TestCommandErrorEnvelopeModesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, commandErrorEnvelopeModesHelper, os.Getenv(testx.CmdSubprocessNameEnv))

	var modeArgs []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_MODE_ARGS")), &modeArgs))
	var commandArgs []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_COMMAND_ARGS")), &commandArgs))

	args := []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--backoff-max-retries", "0"}
	args = append(args, modeArgs...)
	if os.Getenv("C8VOLT_TEST_NO_ERR_CODES") == "1" {
		args = append(args, "--no-err-codes")
	}
	os.Args = append(args, commandArgs...)
	Execute()
}
