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

const processDefinitionErrorEnvelopeHelper = "TestCommandErrorEnvelopeProcessDefinitionHelper"

// TestCommandErrorEnvelopeProcessDefinition verifies all reachable corrected
// retrieval, selector, search, and XML validation branches use one result.
func TestCommandErrorEnvelopeProcessDefinition(t *testing.T) {
	t.Run("runtime-failures", func(t *testing.T) {
		tests := []struct {
			name         string
			args         []string
			method       string
			requestPath  string
			requestCount int32
			message      func(string) string
		}{
			{
				name:         "by-key",
				args:         []string{"get", "process-definition", "--key", "123"},
				method:       http.MethodGet,
				requestPath:  "/v2/process-definitions/123",
				requestCount: 4,
				message: func(baseURL string) string {
					return fmt.Sprintf("get process definition: service unavailable: service unavailable: 503 GET %s/v2/process-definitions/123 (boom)", baseURL)
				},
			},
			{
				name:         "selector",
				args:         []string{"get", "process-definition", "--bpmn-process-id", "order"},
				method:       http.MethodPost,
				requestPath:  "/v2/process-definitions/search",
				requestCount: 1,
				message: func(baseURL string) string {
					return fmt.Sprintf("validate process definition selector \"order\": service unavailable: service unavailable: 503 POST %s/v2/process-definitions/search (boom)", baseURL)
				},
			},
			{
				name:         "paged-search",
				args:         []string{"get", "process-definition"},
				method:       http.MethodPost,
				requestPath:  "/v2/process-definitions/search",
				requestCount: 1,
				message: func(baseURL string) string {
					return fmt.Sprintf("search process definitions: service unavailable: service unavailable: 503 POST %s/v2/process-definitions/search (boom)", baseURL)
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var requests atomic.Int32
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					require.Equal(t, tt.method, r.Method)
					require.Equal(t, tt.requestPath, r.URL.Path)
					http.Error(w, "boom", http.StatusServiceUnavailable)
				}))
				t.Cleanup(server.Close)
				cfgPath := testx.WriteTestConfig(t, server.URL)

				assertProcessDefinitionErrorMatrix(t, cfgPath, tt.args, nil, exitcode.Unavailable, "failed", "unavailable", tt.message(server.URL), "", &requests, tt.requestCount)
			})
		}
	})

	t.Run("xml-validation", func(t *testing.T) {
		tests := []struct {
			name         string
			args         []string
			message      string
			humanArgs    []string
			humanMessage string
		}{
			{
				name:    "missing-key-precedes-json-incompatibility",
				args:    []string{"get", "process-definition", "--xml"},
				message: "invalid input: missing dependent flags: xml output requires --key to select a single process definition",
			},
			{
				name:         "json-incompatible-with-key",
				args:         []string{"get", "process-definition", "--key", "123", "--xml"},
				message:      "invalid input: forbidden flag combination: xml output only supports --key; incompatible with --json",
				humanArgs:    []string{"get", "process-definition", "--key", "123", "--xml", "--latest"},
				humanMessage: "invalid input: forbidden flag combination: xml output only supports --key; incompatible with --latest",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var requests atomic.Int32
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
					requests.Add(1)
				}))
				t.Cleanup(server.Close)
				cfgPath := testx.WriteTestConfig(t, server.URL)

				assertProcessDefinitionErrorMatrix(t, cfgPath, tt.args, tt.humanArgs, exitcode.InvalidArgs, "invalid", "invalid_input", tt.message, tt.humanMessage, &requests, 0)
			})
		}
	})
}

// assertProcessDefinitionErrorMatrix checks JSON and human streams with both
// exit policies while proving validation and rendering do not add requests.
func assertProcessDefinitionErrorMatrix(t *testing.T, cfgPath string, commandArgs []string, humanArgs []string, exitCode int, outcome string, class string, message string, humanMessage string, requests *atomic.Int32, wantRequests int32) {
	t.Helper()
	for _, outputMode := range []string{"json", "human"} {
		for _, suppressExit := range []bool{false, true} {
			name := outputMode
			if suppressExit {
				name += "/no-error-codes"
			}
			t.Run(name, func(t *testing.T) {
				args := commandArgs
				wantMessage := message
				if outputMode == "human" && humanArgs != nil {
					args = humanArgs
					wantMessage = humanMessage
				}
				before := requests.Load()
				wantExitCode := exitCode
				if suppressExit {
					wantExitCode = 0
				}
				stdout, stderr := runCommandErrorEnvelopeSubprocess(t, processDefinitionErrorEnvelopeHelper, "", map[string]string{
					"C8VOLT_TEST_CONFIG":       cfgPath,
					"C8VOLT_TEST_OUTPUT_MODE":  outputMode,
					"C8VOLT_TEST_NO_ERR_CODES": boolEnv(suppressExit),
					"C8VOLT_TEST_COMMAND_ARGS": marshalRootArgsForEnv(t, args),
				}, "", wantExitCode)
				if outputMode == "json" {
					require.NotContains(t, stderr, message, "JSON must not duplicate the final failure diagnostic")
					require.NotContains(t, stderr, " ERROR ", "JSON may retain retry context but not a human error result")
					assertCommandErrorEnvelope(t, stdout, "", commandErrorEnvelopeExpectation{
						Outcome: outcome,
						Class:   class,
						Command: "get process-definition",
						Message: wantMessage,
					})
				} else {
					assertHumanCommandError(t, stdout, stderr, wantMessage)
				}
				require.Equal(t, before+wantRequests, requests.Load())
			})
		}
	}
}

// TestCommandErrorEnvelopeProcessDefinitionHelper executes the selected real
// process-definition path under exact helper-process scoping.
func TestCommandErrorEnvelopeProcessDefinitionHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, processDefinitionErrorEnvelopeHelper, os.Getenv(testx.CmdSubprocessNameEnv))

	var commandArgs []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_COMMAND_ARGS")), &commandArgs))
	args := []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--backoff-max-retries", "0"}
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
