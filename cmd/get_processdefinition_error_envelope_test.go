// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
			humanMessage string
		}{
			{
				name:         "by-key",
				humanMessage: "get process definition: unavailable",
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
				humanMessage: "validate process definition selector: unavailable",
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
				humanMessage: "search process definitions: unavailable",
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
					if serveVersionCheckTopology(w, r, "8.8") {
						return
					}

					requests.Add(1)
					require.Equal(t, tt.method, r.Method)
					require.Equal(t, tt.requestPath, r.URL.Path)
					http.Error(w, "boom", http.StatusServiceUnavailable)
				}))
				t.Cleanup(server.Close)
				cfgPath := testx.WriteTestConfig(t, server.URL)

				var humanArgs []string
				if tt.humanMessage != "" {
					humanArgs = tt.args
				}
				assertProcessDefinitionErrorMatrix(t, cfgPath, tt.args, humanArgs, exitcode.Unavailable, "failed", "unavailable", tt.message(server.URL), tt.humanMessage, &requests, tt.requestCount)
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
	if path := os.Getenv("C8VOLT_TEST_PD_LOG_DEST"); path != "" {
		destination, err := os.Create(path)
		require.NoError(t, err)
		root := Root()
		if os.Getenv("C8VOLT_TEST_PD_LOG_ROUTE") == "configured" {
			command, _, err := root.Find([]string{"get", "process-definition"})
			require.NoError(t, err)
			command.SetErr(destination)
		} else {
			root.SetErr(destination)
		}
		// Execute installs process stderr; execute the same Cobra tree directly
		// when exercising a caller-owned configured or inherited destination.
		root.SetOut(os.Stdout)
		root.SetArgs(args[1:])
		require.NoError(t, root.Execute())
		return
	}
	os.Args = args
	Execute()
}

// TestGetProcessDefinitionLifecycleSearchDiagnostics traces ordinary search and
// selector HTTP failures through configured loggers and the unchanged JSON envelope.
func TestGetProcessDefinitionLifecycleSearchDiagnostics(t *testing.T) {
	for _, selector := range []bool{false, true} {
		for _, format := range []string{"plain", "json"} {
			for _, route := range []string{"configured", "inherited"} {
				for _, mode := range []string{"normal", "verbose", "quiet", "json", "debug-flag", "debug-config"} {
					t.Run(fmt.Sprintf("selector=%v/%s/%s/%s", selector, format, route, mode), func(t *testing.T) {
						var requests atomic.Int32
						server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							if serveVersionCheckTopology(w, r, "8.10") {
								return
							}

							requests.Add(1)
							require.Equal(t, "POST /v2/process-definitions/search", r.Method+" "+r.URL.Path)
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusServiceUnavailable)
							_, _ = w.Write([]byte(`{"message":"repeated-backend-body POST https://search-secret.invalid/8675309","access_token":"search-token-secret"}`))
						}))
						t.Cleanup(server.Close)
						level := "info"
						if mode == "debug-config" {
							level = "debug"
						}
						cfg := writeRawTestConfig(t, fmt.Sprintf("app:\n  camunda_version: \"8.10\"\nauth:\n  mode: none\napis:\n  camunda_api:\n    base_url: %s\nlog:\n  level: %s\n  format: %s\n", server.URL, level, format))
						args := []string{"get", "pd"}
						message := "search process definitions: unavailable"
						if selector {
							args = append(args, "--bpmn-process-id", "invoice")
							message = "validate process definition selector: unavailable"
						}
						if mode == "quiet" || mode == "verbose" {
							args = append(args, "--"+mode)
						}
						if mode == "debug-flag" {
							args = append(args, "--debug")
						}
						capture := filepath.Join(t.TempDir(), "diagnostics")
						stdout, stderr, runErr := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, processDefinitionErrorEnvelopeHelper, "", map[string]string{
							"C8VOLT_TEST_CONFIG": cfg, "C8VOLT_TEST_OUTPUT_MODE": mode, "C8VOLT_TEST_COMMAND_ARGS": marshalRootArgsForEnv(t, args), "C8VOLT_TEST_PD_LOG_DEST": capture, "C8VOLT_TEST_PD_LOG_ROUTE": route,
						}, "")
						var exitErr *exec.ExitError
						require.ErrorAs(t, runErr, &exitErr)
						require.Equal(t, exitcode.Unavailable, exitErr.ExitCode())
						require.EqualValues(t, 1, requests.Load())
						require.Empty(t, stderr)
						diagnosticBytes, err := os.ReadFile(capture)
						require.NoError(t, err)
						diagnostics := string(diagnosticBytes)
						if mode == "json" {
							decoder := json.NewDecoder(strings.NewReader(stdout))
							var envelope map[string]any
							require.NoError(t, decoder.Decode(&envelope))
							require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
							require.Equal(t, "unavailable", envelope["class"])
							require.Equal(t, "failed", envelope["outcome"])
							require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "repeated-backend-body")
							require.NotContains(t, envelope, "payload", "failed discovery has no authoritative partial result")
							require.Empty(t, diagnostics)
							return
						}
						require.Empty(t, stdout)
						require.Equal(t, 1, strings.Count(diagnostics, message))
						require.NotContains(t, diagnostics, "search-token-secret")
						if strings.HasPrefix(mode, "debug-") {
							require.Contains(t, diagnostics, "api #")
							require.Contains(t, diagnostics, "/v2/process-definitions/search")
						} else {
							for _, secret := range []string{"POST", server.URL, "search-secret.invalid", "8675309", "repeated-backend-body"} {
								require.NotContains(t, diagnostics, secret)
							}
						}
						if format == "plain" {
							require.Contains(t, diagnostics, " ERROR "+message+"\n")
							require.Equal(t, 1, strings.Count(diagnostics, " ERROR "))
						} else {
							decoder := json.NewDecoder(strings.NewReader(diagnostics))
							errors := 0
							for {
								var record map[string]any
								err := decoder.Decode(&record)
								if err == io.EOF {
									break
								}
								require.NoError(t, err)
								if record["level"] == "ERROR" {
									errors++
									require.Equal(t, message, record["msg"])
								}
							}
							require.Equal(t, 1, errors)
						}
					})
				}
			}
		}
	}
}
