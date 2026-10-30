// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// isolateRequestRateEnvironment prevents inherited profiles, credentials and
// configuration from redirecting any command fixture away from its owned server.
func isolateRequestRateEnvironment(t *testing.T) {
	t.Helper()
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "C8VOLT_") || key == "GO_WANT_HELPER_PROCESS" {
			t.Setenv(key, "")
			require.NoError(t, os.Unsetenv(key))
		}
	}
	t.Setenv("GOFLAGS", "")
}

// TestRequestRateMutationCompatibility reuses the established acceptance fixtures
// across pacing/reporting settings: scope, prompts, ordering, partial results,
// redaction and exits must retain their existing acceptance assertions.
func TestRequestRateMutationCompatibility(t *testing.T) {
	isolateRequestRateEnvironment(t)
	for _, mode := range []struct{ rate, interval string }{{"0", "2ms"}, {"100", "2ms"}, {"100", "0s"}} {
		t.Run("rate="+mode.rate+"/interval="+mode.interval, func(t *testing.T) {
			t.Setenv("C8VOLT_HTTP_REQUEST_RATE", mode.rate)
			t.Setenv("C8VOLT_HTTP_REQUEST_RATE_REPORT_INTERVAL", mode.interval)
			for _, tc := range []struct {
				name string
				run  func(*testing.T)
			}{
				{"definition diagnostics", TestDeleteProcessDefinitionUnfinishedDiagnosticLogging},
				{"purge diagnostics", TestOpsPurgeAllProcessDefinitionsUnfinishedDiagnosticLogging},
				{"definition cleanup", TestDeleteProcessDefinitionSuspendedForceExecution},
				{"purge cleanup and sparse pages", TestOpsPurgeAllProcessDefinitionsSuspendedForceExecution},
				{"definition controls", TestDeleteProcessDefinitionUnfinishedEmptyWorkloadControls},
				{"purge empty terminal", TestOpsPurgeAllProcessDefinitionsEmptyTerminal},
				{"cancel empty", TestCancelProcessInstanceEmptySelectorOutput},
				{"delete empty", TestDeleteProcessInstanceEmptySelectorOutput},
				{"cancel sparse", TestCancelProcessInstanceDryRun_SearchContinuesAfterEmptySelectedPage},
				{"delete sparse", TestDeleteProcessInstanceDryRun_SearchContinuesAfterEmptySelectedPage},
				{"cancel selection", TestCancelProcessInstanceCommand_SearchSelectionUsesDateFiltersAndCancelsMatches},
				{"delete selection", TestDeleteProcessInstanceCommand_SearchSelectionUsesDateFiltersAndDeletesMatches},
				{"cancel terminal", TestAPIDiagnosticsTerminalCancellation},
				{"empty terminal", TestConfirmationEmptySelectorResults},
				{"mutation terminal", TestProcessInstanceConfirmationTerminal},
			} {
				t.Run(tc.name, func(t *testing.T) {
					resetRequestRateCommandContexts(Root())
					t.Cleanup(func() { resetRequestRateCommandContexts(Root()) })
					tc.run(t)
				})
			}
		})
	}
}

// runRequestRateMutation captures the effective diagnostic writer independently
// of both process streams, detecting even a correctly formatted logger bypass.
func runRequestRateMutation(t *testing.T, args []string, destination string, cancelAfter time.Duration) (string, string, error) {
	t.Helper()
	raw, err := json.Marshal(args)
	require.NoError(t, err)
	capture := filepath.Join(t.TempDir(), "diagnostics")
	stdout, stderr, runErr := testx.RunCmdSubprocessSeparate(t, "TestRequestRateMutationProcess", map[string]string{
		"C8VOLT_RATE_ARGS": string(raw), "C8VOLT_RATE_CAPTURE": capture,
		"C8VOLT_RATE_DESTINATION": destination, "C8VOLT_RATE_CANCEL_AFTER": cancelAfter.String(),
	})
	diagnostic, err := os.ReadFile(capture)
	require.NoError(t, err)
	require.Empty(t, stderr, "all diagnostics must use the effective command writer")
	return stdout, string(diagnostic), runErr
}

// TestRequestRateMutationProcess executes only explicitly scoped subprocesses;
// cancellation exercises the invocation context rather than an HTTP fake error.
func TestRequestRateMutationProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_RATE_ARGS")), &args))
	capture, err := os.Create(os.Getenv("C8VOLT_RATE_CAPTURE"))
	require.NoError(t, err)
	cancelAfter, err := time.ParseDuration(os.Getenv("C8VOLT_RATE_CANCEL_AFTER"))
	require.NoError(t, err)
	destination := os.Getenv("C8VOLT_RATE_DESTINATION")
	isolateRequestRateEnvironment(t)
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	resetOpsPurgeAllProcessDefinitionsFlagState()
	root.SetOut(os.Stdout)
	root.SetErr(nil)
	path := []string{"delete", "process-definition"}
	for i, arg := range args {
		if arg == "cancel" || arg == "delete" {
			path = args[i : i+2]
			break
		}
		if arg == "ops" {
			path = args[i : i+3]
			break
		}
	}
	command, _, err := root.Find(path)
	require.NoError(t, err)
	if destination == "configured" {
		command.SetErr(capture)
	} else {
		root.SetErr(capture)
		command.SetErr(nil)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if cancelAfter > 0 {
		time.AfterFunc(cancelAfter, cancel)
	}
	root.SetArgs(args)
	require.NoError(t, root.ExecuteContext(ctx))
	os.Exit(0)
}

// decodeRequestRateEnvelope enforces exactly one result, with no trailing output.
func decodeRequestRateEnvelope(t *testing.T, stdout string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(stdout))
	var envelope map[string]any
	require.NoError(t, decoder.Decode(&envelope))
	require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
	return envelope
}

// TestRequestRateMutationLocalFailure traces unsent admission through service,
// facade, selected output, logger and exit, retaining an earlier accepted delete.
// Opt out of advisory traffic to isolate the fixed operational admission slots.
func TestRequestRateMutationLocalFailure(t *testing.T) {
	isolateRequestRateEnvironment(t)
	for _, operation := range []string{"cancel", "delete", "definition", "purge"} {
		for _, format := range []string{"plain-time", "json"} {
			for i, flags := range [][]string{nil, {"--verbose"}, {"--quiet"}, {"--json"}, {"--quiet", "--json"}} {
				t.Run(operation+"/"+format+"/"+strings.Join(flags, " "), func(t *testing.T) {
					var requests testx.SafeSlice[string]
					server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						requests.Append(r.Method + " " + r.URL.Path)
						w.Header().Set("Content-Type", "application/json")
						switch r.URL.Path {
						case "/v2/resources/123/deletion":
							io.WriteString(w, `{"resourceKey":"123","batchOperation":null}`)
						case "/v2/process-instances/123":
							io.WriteString(w, opsAllProcessDefinitionsPurgeProcessInstanceJSON("123", "123", "900", "ACTIVE"))
						case "/v2/process-definitions/123", "/v2/process-definitions/2251799813685123":
							io.WriteString(w, opsAllProcessDefinitionsPurgeDefinitionJSON("123", "invoice", 1))
						default:
							t.Errorf("unexpected dispatch: %s %s", r.Method, r.URL.Path)
							w.WriteHeader(500)
						}
					}))
					t.Cleanup(server.Close)
					config := definitionDeleteLogConfig(t, server.URL, format, false)
					file, err := os.OpenFile(config, os.O_APPEND|os.O_WRONLY, 0600)
					require.NoError(t, err)
					_, err = io.WriteString(file, "http:\n  timeout: 100ms\n")
					require.NoError(t, err)
					require.NoError(t, file.Close())
					args := []string{"--no-version-check", "--config", config, "--request-rate", "0.01"}
					switch operation {
					case "cancel", "delete":
						args = append(args, operation, "process-instance", "--key", "123,456,789")
					case "definition":
						args = append(args, "delete", "process-definition", "--key", "123,456,789", "--no-state-check")
					case "purge":
						args = append(args, "ops", "purge", "all-process-definitions", "--key", "2251799813685123")
					}
					args = append(args, "--auto-confirm", "--no-wait", "--workers", "1")
					args = append(args, flags...)
					destination := "inherited"
					if i%2 == 0 {
						destination = "configured"
					}
					stdout, stderr, runErr := runRequestRateMutation(t, args, destination, 0)
					var exitErr *exec.ExitError
					require.ErrorAs(t, runErr, &exitErr)
					require.Equal(t, exitcode.Timeout, exitErr.ExitCode(), stdout+stderr)
					require.Len(t, requests.Snapshot(), 1, "unsent work must neither dispatch nor retry")
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					var errorMessages []string
					for _, record := range records {
						require.Contains(t, []string{"ERROR", "WARN", "INFO"}, record[0])
						if slices.Contains(flags, "--quiet") {
							require.Equal(t, "ERROR", record[0])
						}
						if record[0] == "ERROR" {
							errorMessages = append(errorMessages, record[1])
						}
					}
					if strings.Contains(strings.Join(flags, " "), "--json") {
						envelope := decodeRequestRateEnvelope(t, stdout)
						require.Equal(t, "timeout", envelope["class"])
						require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "request not sent")
						require.Empty(t, errorMessages)
						if operation == "definition" {
							items := requireJSONObject(t, envelope["payload"])["items"].([]any)
							require.Len(t, items, 3)
							require.Equal(t, true, requireJSONObject(t, items[0])["ok"])
							require.NotEqual(t, true, requireJSONObject(t, items[1])["ok"])
						}
					} else {
						require.Empty(t, stdout)
						require.Len(t, errorMessages, 1)
						require.Contains(t, errorMessages[0], "request not sent")
						require.Contains(t, errorMessages[0], "request-rate wait")
						require.Less(t, len(errorMessages[0]), 240)
						if operation == "definition" {
							require.Contains(t, errorMessages[0], "accepted 1")
						}
					}
					for _, bad := range []string{server.URL, "POST", "Get ", "/v2/"} {
						require.NotContains(t, stderr, bad)
					}
				})
			}
		}
	}
}

// TestRequestRateMutationInterruption cancels a live invocation while its second
// mutation is queued; the first accepted deletion must survive in JSON evidence.
// Advisory opt-out isolates this existing operational cancellation budget.
func TestRequestRateMutationInterruption(t *testing.T) {
	isolateRequestRateEnvironment(t)
	for _, machine := range []bool{false, true} {
		t.Run(strconv.FormatBool(machine), func(t *testing.T) {
			var requests testx.SafeSlice[string]
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Append(r.URL.Path)
				require.Equal(t, "/v2/resources/123/deletion", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"resourceKey":"123","batchOperation":null}`)
			}))
			t.Cleanup(server.Close)
			configPath := definitionDeleteLogConfig(t, server.URL, "plain-time", false)
			appendRequestRateConfig(t, configPath, "10ms")
			args := []string{"--no-version-check", "--config", configPath, "--debug", "--request-rate", "1", "delete", "process-definition", "--key", "123,456", "--no-state-check", "--no-wait", "--auto-confirm", "--workers", "1"}
			if machine {
				args = append(args, "--quiet", "--json")
			}
			stdout, stderr, err := runRequestRateMutation(t, args, "configured", 300*time.Millisecond)
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr)
			require.Equal(t, exitcode.Error, exitErr.ExitCode())
			require.Equal(t, []string{"/v2/resources/123/deletion"}, requests.Snapshot())
			if machine {
				envelope := decodeRequestRateEnvelope(t, stdout)
				require.Equal(t, "local_precondition", envelope["class"])
				items := requireJSONObject(t, envelope["payload"])["items"].([]any)
				require.Equal(t, true, requireJSONObject(t, items[0])["ok"])
				require.Empty(t, stderr)
			} else {
				require.Empty(t, stdout)
				requireRequestRateSummaries(t, stderr, "plain-time")
				var failures []string
				for _, record := range bulkCompletionDiagnosticRecords(t, stderr, "plain-time") {
					if record[0] == "ERROR" {
						failures = append(failures, record[1])
					}
				}
				require.Len(t, failures, 1)
				require.Contains(t, failures[0], "accepted 1")
				require.Contains(t, failures[0], "request not sent")
				require.NotContains(t, stderr, server.URL)
			}
		})
	}
}

// TestRequestRateMutationSharedStarts compares the same real command workflows
// with reporting off/on at zero/positive rates, recording discovery, cleanup
// and confirmation starts. A held first response guarantees a reporting window.
func TestRequestRateMutationSharedStarts(t *testing.T) {
	isolateRequestRateEnvironment(t)
	for _, operation := range []string{"cancel", "delete", "definition", "purge"} {
		for _, noWait := range []bool{false, true} {
			t.Run(operation+"/no-wait="+strconv.FormatBool(noWait), func(t *testing.T) {
				var baseline []string
				for _, mode := range []struct{ rate, interval string }{{"0", "0s"}, {"40", "0s"}, {"0", "2ms"}, {"40", "2ms"}} {
					rate := mode.rate
					var calls testx.SafeSlice[string]
					var starts testx.SafeSlice[time.Time]
					var canceled, deleted, definitionDeleted atomic.Bool
					server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						starts.Append(time.Now())
						calls.Append(r.Method + " " + r.URL.Path)
						if len(calls.Snapshot()) == 1 && mode.interval != "0s" {
							time.Sleep(30 * time.Millisecond)
						}
						if serveVersionCheckTopology(w, r, "8.10") {
							return
						}
						w.Header().Set("Content-Type", "application/json")
						state := "ACTIVE"
						if canceled.Load() {
							state = "TERMINATED"
						}
						row := opsAllProcessDefinitionsPurgeProcessInstanceJSON("2251799813685321", "2251799813685321", "2251799813685123", state)
						switch r.URL.Path {
						case "/v2/process-definitions/search":
							fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":1}}`, opsAllProcessDefinitionsPurgeDefinitionJSON("2251799813685123", "invoice", 1))
						case "/v2/process-definitions/2251799813685123":
							if definitionDeleted.Load() {
								w.WriteHeader(404)
								io.WriteString(w, `{"detail":"gone"}`)
								return
							}
							io.WriteString(w, opsAllProcessDefinitionsPurgeDefinitionJSON("2251799813685123", "invoice", 1))
						case "/v2/process-instances/2251799813685321":
							if deleted.Load() {
								w.WriteHeader(404)
								io.WriteString(w, `{"detail":"gone"}`)
								return
							}
							io.WriteString(w, row)
						case "/v2/process-instances/search":
							var body struct {
								Filter map[string]any `json:"filter"`
								Page   struct {
									Limit int `json:"limit"`
								} `json:"page"`
							}
							require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
							if _, children := body.Filter["parentProcessInstanceKey"]; children {
								io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
								return
							}
							if body.Page.Limit == 1 {
								count := 0
								if !canceled.Load() && body.Filter["state"] != "SUSPENDED" {
									count = 1
								}
								fmt.Fprintf(w, `{"items":[],"page":{"totalItems":%d}}`, count)
								return
							}
							if canceled.Load() && (operation == "definition" || operation == "purge") {
								io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
								return
							}
							fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":1}}`, row)
						case "/v2/process-instances/2251799813685321/cancellation":
							canceled.Store(true)
							w.WriteHeader(202)
							io.WriteString(w, `{}`)
						case "/v2/process-instances/2251799813685321/deletion":
							if !canceled.Load() {
								w.WriteHeader(http.StatusConflict)
								io.WriteString(w, `{"detail":"instance is active"}`)
								return
							}
							deleted.Store(true)
							io.WriteString(w, `{}`)
						case "/v2/resources/2251799813685123/deletion":
							definitionDeleted.Store(true)
							io.WriteString(w, `{"resourceKey":"2251799813685123","batchOperation":null}`)
						case "/v2/batch-operations/search":
							io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
						default:
							t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
							w.WriteHeader(500)
						}
					}))
					t.Cleanup(server.Close)
					configPath := definitionDeleteLogConfig(t, server.URL, "plain-time", false)
					appendRequestRateConfig(t, configPath, mode.interval)
					args := []string{"--config", configPath, "--request-rate", rate, "--debug"}
					switch operation {
					case "cancel", "delete":
						args = append(args, operation, "process-instance", "--state", "active")
					case "definition":
						args = append(args, "delete", "process-definition", "--key", "2251799813685123")
					case "purge":
						args = append(args, "ops", "purge", "all-process-definitions")
					}
					args = append(args, "--force", "--auto-confirm", "--json", "--workers", "2")
					if noWait {
						args = append(args, "--no-wait")
					}
					stdout, stderr, err := runRequestRateMutation(t, args, "inherited", 0)
					require.NoError(t, err, stdout+stderr)
					if mode.interval != "0s" {
						requireRequestRateSummaries(t, stderr, "plain-time")
					} else {
						require.NotContains(t, stderr, "HTTP request rate:")
					}
					envelope := decodeRequestRateEnvelope(t, stdout)
					require.Contains(t, []string{"succeeded", "accepted"}, envelope["outcome"])
					require.NotContains(t, stderr, "Proceed")
					got := calls.Snapshot()
					require.True(t, canceled.Load(), "the selected root must be canceled")
					if operation != "cancel" {
						require.True(t, deleted.Load())
					}
					if operation == "definition" || operation == "purge" {
						require.True(t, definitionDeleted.Load())
						require.Less(t, indexOfString(got, "POST /v2/process-instances/2251799813685321/cancellation"), indexOfString(got, "POST /v2/process-instances/2251799813685321/deletion"))
						require.Less(t, indexOfString(got, "POST /v2/process-instances/2251799813685321/deletion"), indexOfString(got, "POST /v2/resources/2251799813685123/deletion"))
					}
					if baseline == nil {
						baseline = got
					} else {
						require.Equal(t, baseline, got, "request controls must not add requests or reorder workflow phases")
					}
					if rate != "0" {
						times := starts.Snapshot()
						require.Greater(t, len(times), 3)
						// Server arrivals include scheduling and local network jitter; 8ms is
						// allowed here, while transport tests assert the tighter delegate bound.
						for i, start := range times {
							require.LessOrEqual(t, float64(i+1), 1+40*(start.Sub(times[0])+8*time.Millisecond).Seconds())
						}
					}
				}
			})
		}
	}
}

// resetRequestRateCommandContexts gives reused singleton command fixtures fresh
// invocation state, including tenant-summary emission state, between suites.
func resetRequestRateCommandContexts(command *cobra.Command) {
	command.SetContext(nil)
	for _, child := range command.Commands() {
		resetRequestRateCommandContexts(child)
	}
}

// TestRequestRateCancellationConfirmation distinguishes a locally unsent read
// from the cancellation already accepted before its short confirmation deadline.
// Advisory opt-out keeps these operational admission slots deterministic.
func TestRequestRateCancellationConfirmation(t *testing.T) {
	isolateRequestRateEnvironment(t)
	for _, flags := range [][]string{nil, {"--verbose"}, {"--quiet"}, {"--json"}, {"--debug", "--log-format", "json"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			var canceled atomic.Bool
			var calls testx.SafeSlice[string]
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Append(r.Method + " " + r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/process-instances/123":
					state := "ACTIVE"
					if canceled.Load() {
						state = "TERMINATED"
					}
					io.WriteString(w, opsAllProcessDefinitionsPurgeProcessInstanceJSON("123", "123", "900", state))
				case "/v2/process-instances/search":
					io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
				case "/v2/process-instances/123/cancellation":
					canceled.Store(true)
					w.WriteHeader(202)
					io.WriteString(w, `{}`)
				default:
					t.Errorf("unexpected request: %s", r.URL.Path)
					w.WriteHeader(500)
				}
			}))
			t.Cleanup(server.Close)
			cfg := writeRawTestConfig(t, fmt.Sprintf(`app:
  camunda_version: "8.10"
  backoff:
    strategy: fixed
    initial_delay: 1ms
    max_retries: 0
    timeout: 5ms
auth:
  mode: none
apis:
  camunda_api:
    base_url: %q
`, server.URL))
			args := append([]string{"--no-version-check", "--config", cfg, "--request-rate", "20", "cancel", "process-instance", "--key", "123", "--auto-confirm"}, flags...)
			stdout, stderr, err := runRequestRateMutation(t, args, "configured", 0)
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr)
			require.Equal(t, exitcode.Timeout, exitErr.ExitCode(), stdout+stderr)
			require.True(t, canceled.Load())
			require.Equal(t, 1, strings.Count(strings.Join(calls.Snapshot(), "\n"), "/cancellation"))
			format := "plain-time"
			if slices.Contains(flags, "--log-format") {
				format = "json"
			}
			var failures []string
			for _, record := range bulkCompletionDiagnosticRecords(t, stderr, format) {
				if record[0] == "ERROR" {
					failures = append(failures, record[1])
				}
			}
			if slices.Contains(flags, "--json") {
				envelope := decodeRequestRateEnvelope(t, stdout)
				require.Equal(t, "timeout", envelope["class"])
				require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "request not sent")
				require.Empty(t, failures)
			} else {
				require.Empty(t, stdout)
				require.Len(t, failures, 1)
				require.Contains(t, failures[0], "cancellation submitted, outcome unconfirmed")
				require.Contains(t, failures[0], "request not sent")
			}
			require.NotContains(t, stderr, server.URL, "raw transport chains must not bypass DEBUG redaction")
		})
	}
}

// TestRequestRateReportingPartialFailure retains accepted JSON evidence and timeout exit
// while the configured DEBUG reporter measures only the single dispatched mutation.
// Advisory opt-out isolates the deliberately exhausted operational request budget.
func TestRequestRateReportingPartialFailure(t *testing.T) {
	isolateRequestRateEnvironment(t)
	var calls atomic.Int32
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.Equal(t, "/v2/resources/123/deletion", r.URL.Path)
		time.Sleep(30 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"resourceKey":"123","batchOperation":null}`)
	}))
	defer server.Close()
	path := definitionDeleteLogConfig(t, server.URL, "json", false)
	appendRequestRateConfig(t, path, "2ms")
	args := []string{"--no-version-check", "--config", path, "--debug", "--json", "--timeout", "100ms", "--request-rate", "0.01", "delete", "process-definition", "--key", "123,456,789", "--no-state-check", "--no-wait", "--auto-confirm", "--workers", "1"}
	stdout, stderr, err := runRequestRateMutation(t, args, "configured", 0)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, exitcode.Timeout, exitErr.ExitCode())
	require.Equal(t, int32(1), calls.Load())
	envelope := decodeRequestRateEnvelope(t, stdout)
	require.Equal(t, "timeout", envelope["class"])
	items := requireJSONObject(t, envelope["payload"])["items"].([]any)
	require.Len(t, items, 3)
	require.Equal(t, true, requireJSONObject(t, items[0])["ok"])
	requireRequestRateSummaries(t, stderr, "json")
	require.Equal(t, 1, strings.Count(stderr, "1 requests in"), "unsent attempts must not be counted")
	require.NotContains(t, stderr, "2 requests in")
	require.NotContains(t, stderr, "3 requests in")
}
