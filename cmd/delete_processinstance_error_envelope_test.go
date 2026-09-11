// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"net/http"
	"os"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const deleteValidationErrorEnvelopeHelper = "TestCommandErrorEnvelopeDeleteValidationHelper"

// TestCommandErrorEnvelopeDeleteValidation verifies execution-time search
// validation uses the shared JSON contract and preserves human behavior.
func TestCommandErrorEnvelopeDeleteValidation(t *testing.T) {
	var requests atomic.Int32
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	t.Cleanup(server.Close)
	cfgPath := testx.WriteTestConfig(t, server.URL)

	const message = "invalid input: invalid flag value: --limit must be positive integer"
	for _, outputMode := range []string{"json", "human"} {
		for _, suppressExit := range []bool{false, true} {
			name := outputMode
			if suppressExit {
				name += "/no-error-codes"
			}
			t.Run(name, func(t *testing.T) {
				wantExitCode := exitcode.InvalidArgs
				if suppressExit {
					wantExitCode = 0
				}
				stdout, stderr := runCommandErrorEnvelopeSubprocess(t, deleteValidationErrorEnvelopeHelper, "", map[string]string{
					"C8VOLT_TEST_CONFIG":      cfgPath,
					"C8VOLT_TEST_OUTPUT_MODE": outputMode,
					"C8VOLT_TEST_NO_ERR_CODES": func() string {
						if suppressExit {
							return "1"
						}
						return "0"
					}(),
				}, "", wantExitCode)
				if outputMode == "json" {
					assertCommandErrorEnvelope(t, stdout, stderr, commandErrorEnvelopeExpectation{
						Outcome: "invalid",
						Class:   "invalid_input",
						Command: "delete process-instance",
						Message: message,
					})
				} else {
					assertHumanCommandError(t, stdout, stderr, message)
				}
				require.Zero(t, requests.Load(), "validation must not start discovery or mutation")
			})
		}
	}
}

// TestCommandErrorEnvelopeDeleteValidationHelper executes the real command
// path in the isolated subprocess selected by the exact helper scope.
func TestCommandErrorEnvelopeDeleteValidationHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, deleteValidationErrorEnvelopeHelper, os.Getenv(testx.CmdSubprocessNameEnv))

	args := []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG")}
	if os.Getenv("C8VOLT_TEST_OUTPUT_MODE") == "json" {
		args = append(args, "--json")
	}
	if os.Getenv("C8VOLT_TEST_NO_ERR_CODES") == "1" {
		args = append(args, "--no-err-codes")
	}
	args = append(args, "delete", "process-instance", "--state", "completed", "--limit", "0")
	os.Args = args
	Execute()
}
