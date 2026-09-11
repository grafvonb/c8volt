// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const clusterErrorEnvelopeHelper = "TestCommandErrorEnvelopeClusterHelper"

// TestCommandErrorEnvelopeCluster verifies cluster runtime failures retain
// their command context, classification, streams, and exit policy.
func TestCommandErrorEnvelopeCluster(t *testing.T) {
	tests := []struct {
		name         string
		commandPath  string
		args         []string
		requestPath  string
		status       int
		body         string
		class        string
		exitCode     int
		requestCount int32
		message      func(string) string
	}{
		{
			name:         "topology-unavailable",
			commandPath:  "get cluster topology",
			args:         []string{"get", "cluster", "topology"},
			requestPath:  "/v2/topology",
			status:       http.StatusServiceUnavailable,
			body:         "boom",
			class:        "unavailable",
			exitCode:     exitcode.Unavailable,
			requestCount: 4,
			message: func(baseURL string) string {
				return fmt.Sprintf("get cluster topology: service unavailable: service unavailable: 503 GET %s/v2/topology (boom)", baseURL)
			},
		},
		{
			name:         "version-unavailable",
			commandPath:  "get cluster version",
			args:         []string{"get", "cluster", "version"},
			requestPath:  "/v2/topology",
			status:       http.StatusServiceUnavailable,
			body:         "boom",
			class:        "unavailable",
			exitCode:     exitcode.Unavailable,
			requestCount: 4,
			message: func(baseURL string) string {
				return fmt.Sprintf("get cluster version: service unavailable: service unavailable: 503 GET %s/v2/topology (boom)", baseURL)
			},
		},
		{
			name:         "license-unavailable",
			commandPath:  "get cluster license",
			args:         []string{"get", "cluster", "license"},
			requestPath:  "/v2/license",
			status:       http.StatusServiceUnavailable,
			body:         "boom",
			class:        "unavailable",
			exitCode:     exitcode.Unavailable,
			requestCount: 4,
			message: func(baseURL string) string {
				return fmt.Sprintf("get cluster license: service unavailable: service unavailable: 503 GET %s/v2/license (boom)", baseURL)
			},
		},
		{
			name:         "license-malformed-response",
			commandPath:  "get cluster license",
			args:         []string{"get", "cluster", "license"},
			requestPath:  "/v2/license",
			status:       http.StatusOK,
			class:        "malformed_response",
			exitCode:     exitcode.Error,
			requestCount: 1,
			message: func(string) string {
				return "get cluster license: malformed response: malformed response: 200 OK but empty payload; body="
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, tt.requestPath, r.URL.Path)
				w.WriteHeader(tt.status)
				if tt.body != "" {
					_, _ = w.Write([]byte(tt.body))
				}
			}))
			t.Cleanup(server.Close)
			cfgPath := testx.WriteTestConfig(t, server.URL)

			for _, outputMode := range []string{"json", "human"} {
				for _, suppressExit := range []bool{false, true} {
					name := outputMode
					if suppressExit {
						name += "/no-error-codes"
					}
					t.Run(name, func(t *testing.T) {
						before := requests.Load()
						wantExitCode := tt.exitCode
						if suppressExit {
							wantExitCode = 0
						}
						stdout, stderr := runCommandErrorEnvelopeSubprocess(t, clusterErrorEnvelopeHelper, "", map[string]string{
							"C8VOLT_TEST_CONFIG":       cfgPath,
							"C8VOLT_TEST_OUTPUT_MODE":  outputMode,
							"C8VOLT_TEST_NO_ERR_CODES": boolEnv(suppressExit),
							"C8VOLT_TEST_COMMAND_ARGS": marshalRootArgsForEnv(t, tt.args),
						}, "", wantExitCode)
						message := tt.message(server.URL)
						if outputMode == "json" {
							require.NotContains(t, stderr, message, "JSON must not duplicate the final failure diagnostic")
							require.NotContains(t, stderr, " ERROR ", "JSON may retain retry context but not a human error result")
							assertCommandErrorEnvelope(t, stdout, "", commandErrorEnvelopeExpectation{
								Outcome: "failed",
								Class:   tt.class,
								Command: tt.commandPath,
								Message: message,
							})
						} else {
							assertHumanCommandError(t, stdout, stderr, message)
						}
						require.Equal(t, before+tt.requestCount, requests.Load(), "rendering must not add backend requests beyond established retries")
					})
				}
			}
		})
	}
}

// TestCommandErrorEnvelopeClusterHelper executes the selected real cluster
// command in the exact helper subprocess.
func TestCommandErrorEnvelopeClusterHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, clusterErrorEnvelopeHelper, os.Getenv(testx.CmdSubprocessNameEnv))

	var commandArgs []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_COMMAND_ARGS")), &commandArgs))
	args := []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--backoff-max-retries", "0"}
	if os.Getenv("C8VOLT_TEST_OUTPUT_MODE") == "json" {
		args = append(args, "--json")
	}
	if os.Getenv("C8VOLT_TEST_NO_ERR_CODES") == "1" {
		args = append(args, "--no-err-codes")
	}
	os.Args = append(args, commandArgs...)
	Execute()
}
