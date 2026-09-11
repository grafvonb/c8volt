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

const stdinErrorEnvelopeHelper = "TestCommandErrorEnvelopeStdinHelper"

// stdinErrorEnvelopeCaller describes one real command path that delegates
// piped-key validation to mergeAndValidateKeys.
type stdinErrorEnvelopeCaller struct {
	Name        string
	CommandPath string
	Args        []string
}

// stdinErrorEnvelopeCallers is the audited 12-command caller inventory.
var stdinErrorEnvelopeCallers = []stdinErrorEnvelopeCaller{
	{Name: "get-incident", CommandPath: "get incident", Args: []string{"get", "incident", "-"}},
	{Name: "get-process-instance", CommandPath: "get process-instance", Args: []string{"get", "process-instance", "-"}},
	{Name: "expect-process-instance", CommandPath: "expect process-instance", Args: []string{"expect", "process-instance", "--state", "active", "-"}},
	{Name: "update-process-instance", CommandPath: "update process-instance", Args: []string{"update", "process-instance", "--vars", `{"updated":true}`, "--dry-run", "-"}},
	{Name: "cancel-process-instance", CommandPath: "cancel process-instance", Args: []string{"cancel", "process-instance", "-"}},
	{Name: "delete-process-instance", CommandPath: "delete process-instance", Args: []string{"delete", "process-instance", "-"}},
	{Name: "resolve-process-instance", CommandPath: "resolve process-instance", Args: []string{"resolve", "process-instance", "-"}},
	{Name: "delete-process-definition", CommandPath: "delete process-definition", Args: []string{"delete", "process-definition", "-"}},
	{Name: "resolve-incident", CommandPath: "resolve incident", Args: []string{"resolve", "incident", "-"}},
	{Name: "ops-analyse-slow-process-instances", CommandPath: "ops analyse slow-process-instances", Args: []string{"ops", "analyse", "slow-process-instances", "-"}},
	{Name: "ops-repair-incident", CommandPath: "ops repair incident", Args: []string{"ops", "repair", "incident", "--dry-run", "-"}},
	{Name: "ops-repair-process-instance", CommandPath: "ops repair process-instance", Args: []string{"ops", "repair", "process-instance", "--dry-run", "-"}},
}

// TestCommandErrorEnvelopeStdin verifies both stdin validation branches through
// every current full-contract caller, including suppressed process exits.
func TestCommandErrorEnvelopeStdin(t *testing.T) {
	var requests atomic.Int32
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	t.Cleanup(server.Close)
	cfgPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

	failures := []struct {
		name          string
		stdin         string
		message       string
		absentMessage string
	}{
		{
			name:          "filter-output",
			stdin:         "filter: state=active\n",
			message:       "invalid input: invalid flag value: validating keys from stdin failed: use --keys-only flag to get only keys as input",
			absentMessage: "line \"filter: state=active\" at index 0 is not a valid key",
		},
		{
			name:    "malformed-key-after-valid-and-blank",
			stdin:   "2251799813685248\n\nbad-key\n",
			message: "invalid input: invalid flag value: validating keys from stdin failed: line \"bad-key\" at index 1 is not a valid key; have you forgotten to use --keys-only flag in case of c8volt commands?",
		},
	}

	for _, caller := range stdinErrorEnvelopeCallers {
		caller := caller
		t.Run(caller.Name, func(t *testing.T) {
			for _, failure := range failures {
				failure := failure
				t.Run(failure.name, func(t *testing.T) {
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
								stdout, stderr := runCommandErrorEnvelopeSubprocess(t, stdinErrorEnvelopeHelper, "", map[string]string{
									"C8VOLT_TEST_CONFIG":       cfgPath,
									"C8VOLT_TEST_STDIN_CALLER": caller.Name,
									"C8VOLT_TEST_OUTPUT_MODE":  outputMode,
									"C8VOLT_TEST_NO_ERR_CODES": boolEnv(suppressExit),
								}, failure.stdin, wantExitCode)
								if outputMode == "json" {
									assertCommandErrorEnvelope(t, stdout, stderr, commandErrorEnvelopeExpectation{
										Outcome: "invalid",
										Class:   "invalid_input",
										Command: caller.CommandPath,
										Message: failure.message,
									})
								} else {
									assertHumanCommandError(t, stdout, stderr, failure.message)
								}
								if failure.absentMessage != "" {
									require.NotContains(t, stdout+stderr, failure.absentMessage)
								}
								require.Zero(t, requests.Load(), "stdin validation must stop all subsequent requests")
							})
						}
					}
				})
			}
		})
	}
}

// TestCommandErrorEnvelopeStdinHelper selects one real stdin caller and lets
// the command's exit behavior terminate the scoped subprocess.
func TestCommandErrorEnvelopeStdinHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, stdinErrorEnvelopeHelper, os.Getenv(testx.CmdSubprocessNameEnv))

	callerName := os.Getenv("C8VOLT_TEST_STDIN_CALLER")
	var commandArgs []string
	for _, caller := range stdinErrorEnvelopeCallers {
		if caller.Name == callerName {
			commandArgs = caller.Args
			break
		}
	}
	require.NotEmpty(t, commandArgs, "unknown stdin caller %q", callerName)

	args := []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG")}
	if os.Getenv("C8VOLT_TEST_OUTPUT_MODE") == "json" {
		args = append(args, "--json")
	}
	if os.Getenv("C8VOLT_TEST_NO_ERR_CODES") == "1" {
		args = append(args, "--no-err-codes")
	}
	args = append(args, commandArgs...)
	os.Args = args
	Execute()
}

// boolEnv converts a test matrix boolean into the subprocess environment form.
func boolEnv(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
