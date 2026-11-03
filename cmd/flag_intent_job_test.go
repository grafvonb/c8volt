// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestFlagIntentJobSurface proves HTTP timeout identity and canonical metadata/completion.
func TestFlagIntentJobSurface(t *testing.T) {
	require.Same(t, rootCmd.PersistentFlags().Lookup("timeout"), updateJobCmd.InheritedFlags().Lookup("timeout"))
	capability := commandCapabilityForCommand(updateJobCmd)
	for _, name := range []string{"set-retries", "set-timeout", "set-vars", "retries", "timeout", "throw-bpmn-error", "set-error-code", "set-error-message", "set-retry-backoff"} {
		found := false
		for _, flag := range capability.Flags {
			found = found || flag.Name == name
		}
		require.True(t, found, name)
	}
	require.Nil(t, updateJobCmd.Flags().Lookup("vars"))
	require.Nil(t, updateJobCmd.Flags().Lookup("set-vars-file"))
	output := executeCompletionNoDescForTest(t, "update", "job", "--")
	for _, name := range []string{"set-retries", "set-timeout", "set-vars", "retries", "timeout", "throw-bpmn-error", "set-error-code", "set-error-message", "set-retry-backoff"} {
		require.Contains(t, output, "--"+name+"\n")
	}
	for _, old := range []string{"vars", "message", "retry-backoff"} {
		require.Nil(t, updateJobCmd.Flags().Lookup(old))
		require.NotContains(t, output, "--"+old+"\n")
	}
}

// TestFlagIntentJobRejection checks semantic and removed-token failures before any remote bootstrap.
func TestFlagIntentJobRejection(t *testing.T) {
	var calls atomic.Int64
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }))
	defer server.Close()
	cases := []struct {
		flags    []string
		guidance string
	}{
		{[]string{"--key", "1", "--set-error-code", "CODE", "--throw-bpmn-error=false"}, "--set-error-code"},
		{[]string{"--key", "1", "--throw-bpmn-error=123"}, "--set-error-code"},
		{[]string{"--key", "1", "--throw-bpmn-error", "123", "--set-error-code", "CODE"}, "--set-error-code"},
		{[]string{"--key", "1", "--throw-bpmn-error=true", "--set-error-code", "CODE"}, "--set-error-code"},
		{[]string{"--key", "1", "--throw-bpmn-error=false", "--set-error-code", "CODE"}, "--set-error-code"},
		{[]string{"--key", "1", "--throw-bpmn-error=", "--set-error-code", "CODE"}, "--set-error-code"},
		{[]string{"--key", "1", "--throw-bpmn-error=PRIVATE", "--set-error-code", "CODE"}, "--set-error-code"},
		{[]string{"--key", "1", "--throw-bpmn-error", "PRIVATE"}, "--set-error-code"},
		{[]string{"--key", "1", "--throw-bpmn-error", "true"}, "--set-error-code"},
		{[]string{"--key", "1", "--throw-bpmn-error", "false"}, "--set-error-code"},
		{[]string{"--key", "1", "--throw-bpmn-error"}, "--set-error-code"},
		{[]string{"--key", "1", "--throw-bpmn-error", "--set-error-code", ""}, "--set-error-code"},
		{[]string{"--key", "1", "--throw-bpmn-error", "--set-error-code", "  "}, "--set-error-code"},
		{[]string{"--key", "1", "--set-error-code", "PRIVATE"}, "--set-error-code"},
		{[]string{"--key", "1", "--fail", "--set-retries", "0", "--message", "PRIVATE"}, "--set-error-message"},
		{[]string{"--key", "1", "--fail", "--set-retries", "0", "--message", "PRIVATE", "--set-error-message", "PRIVATE"}, "--set-error-message"},
		{[]string{"--key", "1", "--fail", "--set-retries", "0", "--set-error-message", "PRIVATE", "--message", "PRIVATE"}, "--set-error-message"},
		{[]string{"--key", "1", "--fail", "--set-retries", "0", "--retry-backoff", "PRIVATE"}, "--set-retry-backoff"},
		{[]string{"--key", "1", "--fail", "--set-retries", "0", "--retry-backoff", "PRIVATE", "--set-retry-backoff", "PRIVATE"}, "--set-retry-backoff"},
		{[]string{"--key", "1", "--fail", "--set-retries", "0", "--set-retry-backoff", "PRIVATE", "--retry-backoff", "PRIVATE"}, "--set-retry-backoff"},
		{[]string{"--key", "1", "--retries", "0"}, "--set-retries"},
		{[]string{"--key", "1", "--fail", "--retries", "0"}, "--set-retries"},
		{[]string{"--key", "1", "--retries", "0", "--set-retries", "0"}, "--set-retries"},
		{[]string{"--key", "1", "--set-retries", "0", "--retries", "0"}, "--set-retries"},
		{[]string{"--key", "1", "--timeout", "30s"}, "--set-timeout"},
		{[]string{"--complete", "--retries", "0", "--set-retries", "0"}, "--set-retries"},
		{[]string{"--complete", "--key", "1", "--retries", "0"}, "--retries"},
		{[]string{"--complete", "--vars", "PRIVATE"}, "--set-vars"},
		{[]string{"--complete", "--vars", "PRIVATE", "--set-vars", "PRIVATE"}, "--set-vars"},
		{[]string{"--complete", "--set-vars", "PRIVATE", "--vars", "PRIVATE"}, "--set-vars"},
	}
	for _, tc := range cases {
		for _, format := range []string{"plain-time", "json"} {
			for _, mode := range []string{"normal", "verbose", "quiet", "json"} {
				t.Run(strings.Join(tc.flags, " ")+"/"+format+"/"+mode, func(t *testing.T) {
					cfg := completionLogConfig(t, server.URL, format, "warn")
					args := append([]string{"update", "job"}, tc.flags...)
					args = append(args, "--config", cfg)
					if mode != "normal" {
						args = append(args, "--"+mode)
					}
					encoded, err := json.Marshal(args)
					require.NoError(t, err)
					stdout, stderr := runCommandErrorEnvelopeSubprocess(t, "TestFlagIntentVariablesHelper", t.TempDir(), map[string]string{"C8VOLT_TEST_ARGS": string(encoded)}, "", exitcode.InvalidArgs)
					require.NotContains(t, stdout+stderr, "PRIVATE")
					if mode == "json" {
						decoder := json.NewDecoder(strings.NewReader(stdout))
						var result map[string]any
						require.NoError(t, decoder.Decode(&result))
						require.ErrorIs(t, decoder.Decode(&result), io.EOF)
						require.Contains(t, stdout, tc.guidance)
						require.Contains(t, stdout, `"invalid_input"`)
						require.Empty(t, stderr)
					} else {
						require.Empty(t, stdout)
						if format == "json" {
							decoder := json.NewDecoder(strings.NewReader(stderr))
							var record map[string]any
							require.NoError(t, decoder.Decode(&record))
							require.Equal(t, "ERROR", record["level"])
							require.Contains(t, record["msg"], tc.guidance)
							require.ErrorIs(t, decoder.Decode(&record), io.EOF)
						} else {
							require.Regexp(t, `^\S+ ERROR invalid input: [^\n]+\n$`, stderr)
							require.Contains(t, stderr, tc.guidance)
						}
					}
				})
			}
		}
	}
	require.Zero(t, calls.Load())
}

// TestFlagIntentJobHTTPTimeout resolves real inherited flags while stopping before remote services.
func TestFlagIntentJobHTTPTimeout(t *testing.T) {
	for _, tc := range []struct {
		name, file, env, want string
		flags                 []string
	}{
		{name: "default", want: "30s"},
		{name: "file", file: "http:\n  timeout: 41s\n", want: "41s"},
		{name: "profile", file: "http:\n  timeout: 41s\nprofiles:\n  selected:\n    http:\n      timeout: 42s\n", flags: []string{"--profile", "selected"}, want: "42s"},
		{name: "environment", file: "http:\n  timeout: 41s\n", env: "43s", want: "43s"},
		{name: "explicit", file: "http:\n  timeout: 41s\n", env: "43s", flags: []string{"--timeout", "44s"}, want: "44s"},
	} {
		for _, before := range []bool{true, false} {
			t.Run(tc.name+map[bool]string{true: "/before", false: "/after"}[before], func(t *testing.T) {
				resetCommandTreeFlags(rootCmd)
				resetUpdateJobFlagState()
				t.Cleanup(func() { resetCommandTreeFlags(rootCmd); resetUpdateJobFlagState() })
				t.Setenv("C8VOLT_HTTP_TIMEOUT", tc.env)
				if tc.env == "" {
					require.NoError(t, os.Unsetenv("C8VOLT_HTTP_TIMEOUT"))
				}
				path := filepath.Join(t.TempDir(), "config.yaml")
				require.NoError(t, os.WriteFile(path, []byte(tc.file), 0600))
				oldRun, oldPre := updateJobCmd.Run, rootCmd.PersistentPreRunE
				t.Cleanup(func() { updateJobCmd.Run = oldRun; rootCmd.PersistentPreRunE = oldPre })
				rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
					require.Same(t, rootCmd.PersistentFlags().Lookup("timeout"), cmd.Flags().Lookup("timeout"))
					_, _, cfg, err := resolveLocalCommandConfig(cmd, false, false)
					require.NoError(t, err)
					require.Equal(t, tc.want, cfg.HTTP.Timeout)
					return nil
				}
				updateJobCmd.Run = func(cmd *cobra.Command, args []string) {
					request, _, err := parseUpdateJobCommandRequest(cmd, args...)
					require.NoError(t, err)
					require.Equal(t, int64(300000), *request.TimeoutMillis)
					require.Nil(t, request.Retries)
				}
				args := []string{"--config", path}
				if before {
					args = append(args, tc.flags...)
				}
				args = append(args, "update", "job", "--key", "1", "--set-timeout", "5m")
				if !before {
					args = append(args, tc.flags...)
				}
				rootCmd.SetArgs(args)
				require.NoError(t, rootCmd.Execute())
			})
		}
	}
}

// TestFlagIntentJobHTTPWithMutation keeps HTTP controls valid in all existing
// mutation modes and proves an explicit submitted zero never becomes omission.
func TestFlagIntentJobHTTPWithMutation(t *testing.T) {
	for _, flags := range [][]string{
		{"--key", "1", "--set-retries", "0"},
		{"--key", "1", "--fail", "--set-retries", "0"},
		{"--key", "1", "--throw-bpmn-error", "--set-error-code", "CODE", "--set-vars", `{"x":false}`},
		{"--key", "1", "--complete", "--set-vars", `{"x":false}`},
		{"--key", "1,2", "--complete", "--set-vars", `{"x":false}`},
		{"--complete", "--retries", "0", "--set-vars", `{"x":false}`},
	} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			resetCommandTreeFlags(rootCmd)
			resetUpdateJobFlagState()
			t.Cleanup(func() { resetCommandTreeFlags(rootCmd); resetUpdateJobFlagState() })
			require.NoError(t, updateJobCmd.ParseFlags(append(flags, "--timeout", "17s")))
			request, batch, err := parseUpdateJobCommandRequest(updateJobCmd)
			require.NoError(t, err)
			require.Nil(t, request.TimeoutMillis)
			if request.Retries != nil {
				require.Zero(t, *request.Retries)
			}
			if request.WorkerOutcome != nil && request.WorkerOutcome.Retries != nil {
				require.Zero(t, *request.WorkerOutcome.Retries)
			}
			if batch != nil && batch.Selection.Filter.Retries != nil {
				require.Zero(t, *batch.Selection.Filter.Retries)
			}
		})
	}
}

// TestFlagIntentJobBothTimeoutPayload captures the job timeout independently of
// inherited HTTP execution settings through the existing facade and fake adapter.
func TestFlagIntentJobBothTimeoutPayload(t *testing.T) {
	var requests []string
	var bodies []map[string]any
	server := newJobUpdateServer(t, "8.8", &requests, &bodies, []string{jobSearchResponseWithState("2251799813711967", 1, "CREATED")}, http.StatusNoContent)
	defer server.Close()
	stdout, stderr := executeRootForJobTestWithSeparateOutputs(t, "--config", writeTestConfigForVersion(t, server.URL, "8.8"), "--timeout", "17s", "update", "job", "--key", "2251799813711967", "--set-timeout", "5m", "--auto-confirm", "--json")
	require.Empty(t, stderr)
	require.Len(t, bodies, 1)
	requirePatchTimeout(t, bodies[0], 300000)
	decoder := json.NewDecoder(strings.NewReader(stdout))
	var envelope map[string]any
	require.NoError(t, decoder.Decode(&envelope))
	require.ErrorIs(t, decoder.Decode(&envelope), io.EOF)
	require.Equal(t, []string{"POST /v2/jobs/search", "PATCH /v2/jobs/2251799813711967"}, requests)
}

// TestFlagIntentJobActionTokens uses registered arity so opaque values and the
// terminator cannot be mistaken for attached action syntax.
func TestFlagIntentJobActionTokens(t *testing.T) {
	for _, tc := range []struct {
		args     []string
		rejected bool
	}{
		{[]string{"--throw-bpmn-error=true", "--set-error-code", "CODE"}, true},
		{[]string{"--throw-bpmn-error=false"}, true},
		{[]string{"--throw-bpmn-error="}, true},
		{[]string{"--set-error-message", "--throw-bpmn-error=true"}, false},
		{[]string{"--set-error-code=--throw-bpmn-error=false"}, false},
		{[]string{"--", "--throw-bpmn-error=true"}, false},
		{[]string{"--throw-bpmn-error", "--set-error-code", "true"}, false},
	} {
		_, err := migrationReportingInput(updateJobCmd, tc.args)
		require.Equal(t, tc.rejected, err != nil, tc.args)
	}
}

// TestFlagIntentJobOpaqueErrorPayload verifies Boolean-looking codes and exact
// messages through command dispatch, facade conversion and native HTTP bodies.
func TestFlagIntentJobOpaqueErrorPayload(t *testing.T) {
	for _, code := range []string{"true", "false", " spaced code "} {
		t.Run(code, func(t *testing.T) {
			var requests []string
			var bodies []map[string]any
			server := newJobBPMNErrorServer(t, "8.8", &requests, &bodies, []string{jobSearchResponse("2251799813711967", 1)}, http.StatusNoContent)
			t.Cleanup(server.Close)
			stdout, stderr := executeRootForJobTestWithSeparateOutputs(t, "--config", writeTestConfigForVersion(t, server.URL, "8.8"), "update", "job", "--key", "2251799813711967", "--throw-bpmn-error", "--set-error-code", code, "--set-error-message", "--throw-bpmn-error=false", "--set-vars", `{"number":123,"ok":false}`, "--auto-confirm", "--json")
			require.Empty(t, stderr)
			decoder := json.NewDecoder(strings.NewReader(stdout))
			var envelope map[string]any
			require.NoError(t, decoder.Decode(&envelope))
			require.ErrorIs(t, decoder.Decode(&envelope), io.EOF)
			require.Len(t, bodies, 1)
			requireBPMNErrorCode(t, bodies[0], code)
			require.Equal(t, "--throw-bpmn-error=false", bodies[0]["errorMessage"])
			require.Equal(t, map[string]any{"number": float64(123), "ok": false}, bodies[0]["variables"])
			require.Equal(t, []string{"POST /v2/jobs/search", "POST /v2/jobs/2251799813711967/error"}, requests)
		})
	}
}

// TestFlagIntentJobWorkerSummaryLogging proves renamed worker inputs retain INFO
// routing and suppression through real command execution and configured writers.
func TestFlagIntentJobWorkerSummaryLogging(t *testing.T) {
	for _, outcome := range scalarJobOutcomes[1:] {
		for _, format := range []string{"plain-time", "json"} {
			for _, destination := range []string{"configured", "inherited"} {
				for _, mode := range []string{"success", "dry-run", "verbose", "warn", "quiet"} {
					t.Run(outcome.name+"/"+format+"/"+destination+"/"+mode, func(t *testing.T) {
						url, calls := scalarJobOutcomeFixture(t, "8.8", "CREATED", outcome.path, http.StatusNoContent, false)
						cfg := writeTestConfigForVersion(t, url, "8.8")
						data, err := os.ReadFile(cfg)
						require.NoError(t, err)
						level := "info"
						if mode == "warn" {
							level = "warn"
						}
						require.NoError(t, os.WriteFile(cfg, append(data, []byte("\nlog:\n  level: "+level+"\n  format: "+format+"\n")...), 0600))
						args := append([]string{"--config", cfg, "update", "job", "--key", scalarJobKey}, outcome.flags...)
						args = append(args, "--set-error-message", "worker message", "--auto-confirm")
						if mode == "dry-run" || mode == "verbose" || mode == "quiet" {
							args = append(args, "--"+mode)
						}
						stdout, stderr, err := runBulkCompletionTerminal(t, args, destination, true, false, nil)
						require.NoError(t, err, stderr)
						require.Empty(t, stdout)
						records := bulkCompletionDiagnosticRecords(t, stderr, format)
						if mode == "warn" || mode == "quiet" {
							require.Empty(t, records)
							require.Empty(t, stderr)
						} else {
							require.NotEmpty(t, records)
							for _, record := range records {
								require.Equal(t, "INFO", record[0])
							}
						}
						expected := []string{"POST /v2/jobs/search"}
						if mode != "dry-run" {
							expected = append(expected, "POST /v2/jobs/"+scalarJobKey+"/"+outcome.path)
						}
						require.Equal(t, expected, calls.Snapshot())
					})
				}
			}
		}
	}
}
