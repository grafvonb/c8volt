// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/spf13/cobra"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"testing/iotest"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// bulkCompletionFixture records discovery, rechecks and writes independently, and
// refuses any endpoint outside this command's completion workflow.
type bulkCompletionFixture struct {
	mu         sync.Mutex
	calls      []string
	scenario   string
	discovery  int
	reads      int
	lookupKeys []string
	writes     []string
}

// newBulkCompletionFixture supplies listener rows and explicit tenant/missing-key
// cases through actual HTTP adapters, recording the preparation-before-write boundary.
func newBulkCompletionFixture(t *testing.T, version string, scenario string) (*bulkCompletionFixture, *httptest.Server) {
	t.Helper()
	fixture := &bulkCompletionFixture{scenario: scenario}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, version) {
			return
		}

		fixture.mu.Lock()
		defer fixture.mu.Unlock()
		fixture.calls = append(fixture.calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" && r.URL.Path == "/v2/jobs/search" {
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			key, lookup := filter["jobKey"].(string)
			if lookup {
				fixture.reads++
				fixture.lookupKeys = append(fixture.lookupKeys, key)
				require.NotContains(t, filter, "tenantId")
				state := "CREATED"
				if scenario == "completed" {
					state = "COMPLETED"
				}
				if scenario == "partial" && key == "2251799813711968" {
					state = "CANCELED"
				}
				if scenario == "explicit missing" && key == "2" {
					io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
					return
				}
				row := bulkCompletionRow(key, state)
				if strings.HasPrefix(scenario, "explicit") && key == "2" {
					row = strings.ReplaceAll(row, `"tenant-a"`, `"tenant-b"`)
				}
				fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":1}}`, row)
				return
			}
			fixture.discovery++
			require.Empty(t, fixture.writes, "all preparation must finish before mutations")
			require.Equal(t, "tenant-a", filter["tenantId"])
			if scenario == "predicates" {
				for name, want := range map[string]any{"state": "CREATED", "type": "cleanup", "processInstanceKey": "2251799813711000", "elementInstanceKey": "2251799813711001", "elementId": "task", "worker": "worker", "retries": float64(0), "kind": "TASK_LISTENER", "listenerEventType": "CANCELING"} {
					require.Equal(t, want, filter[name], name)
				}
			}
			if scenario == "empty" {
				io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
				return
			}
			if scenario == "incomplete" && fixture.discovery == 2 {
				w.WriteHeader(400)
				io.WriteString(w, `{"title":"discovery failed"}`)
				return
			}
			if scenario == "incomplete" {
				fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":2}}`, bulkCompletionRow("2251799813711967", "CREATED"))
				return
			}
			if scenario == "partial" {
				fmt.Fprintf(w, `{"items":[%s,%s],"page":{"totalItems":2}}`, bulkCompletionRow("2251799813711967", "CREATED"), bulkCompletionRow("2251799813711968", "CREATED"))
				return
			}
			fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":1}}`, bulkCompletionRow("2251799813711967", "CREATED"))
			return
		}
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/completion") {
			if scenario == "explicit" {
				require.GreaterOrEqual(t, fixture.reads, 3)
				require.Equal(t, []string{"1", "2"}, fixture.lookupKeys[:2], "all explicit preparation reads must precede mutation")
			}
			fixture.writes = append(fixture.writes, r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Errorf("unexpected mutation or endpoint: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(400)
	}))
	t.Cleanup(server.Close)
	return fixture, server
}

// bulkCompletionRow contains only the observed listener context used by preview and recheck.
func bulkCompletionRow(key, state string) string {
	return fmt.Sprintf(`{"jobKey":%q,"state":%q,"tenantId":"tenant-a","type":"cleanup","kind":"TASK_LISTENER","listenerEventType":"CANCELING","retries":0,"processInstanceKey":"2251799813711000","elementInstanceKey":"2251799813711001","elementId":"task","worker":"worker"}`, key, state)
}

// TestUpdateJobBulkCompletionModes verifies basic safe output and request counts end to end.
func TestUpdateJobBulkCompletionModes(t *testing.T) {
	for _, tc := range []struct {
		name, scenario  string
		flags           []string
		status, outcome string
		reads, writes   int
	}{
		{"preview", "normal", []string{"--dry-run", "--json"}, "preview", "succeeded", 0, 0},
		{"preview quiet", "normal", []string{"--dry-run", "--quiet", "--json"}, "preview", "succeeded", 0, 0},
		{"execute", "normal", []string{"--auto-confirm", "--json"}, "finished", "succeeded", 1, 1},
		{"no wait", "normal", []string{"--auto-confirm", "--json", "--no-wait"}, "finished", "accepted", 1, 1},
		{"automation", "normal", []string{"--automation", "--json"}, "finished", "succeeded", 1, 1},
		{"completed noop", "completed", []string{"--auto-confirm", "--json", "--no-wait"}, "finished", "succeeded", 1, 0},
		{"empty", "empty", []string{"--auto-confirm", "--json", "--no-wait"}, "finished", "succeeded", 0, 0},
		{"empty quiet", "empty", []string{"--auto-confirm", "--quiet", "--json"}, "finished", "succeeded", 0, 0},
		{"empty preview", "empty", []string{"--dry-run", "--json"}, "preview", "succeeded", 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture, server := newBulkCompletionFixture(t, "8.8", tc.scenario)
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.8"), "--tenant", "tenant-a", "update", "job", "--type", "cleanup", "--complete", "--workers", "1"}
			args = append(args, tc.flags...)
			stdout, stderr := executeRootForJobTestWithSeparateOutputs(t, args...)
			envelope := decodeBulkCompletionEnvelope(t, stdout)
			require.Equal(t, tc.outcome, envelope["outcome"])
			payload := requireJSONObject(t, envelope["payload"])
			require.Equal(t, tc.status, payload["status"])
			require.Equal(t, tc.reads, fixture.reads)
			require.Len(t, fixture.writes, tc.writes)
			require.Equal(t, 1, fixture.discovery)
			require.NotContains(t, stderr, "Do you want to proceed?")
			if tc.status == "preview" {
				require.Contains(t, payload, "candidates")
				require.NotContains(t, payload, "outcomes")
			} else {
				require.Contains(t, payload, "outcomes")
				require.NotContains(t, payload, "candidates")
			}
			counts := requireJSONObject(t, payload["counts"])
			require.Len(t, counts, 5)
			if tc.scenario == "empty" {
				require.Equal(t, float64(0), counts["selected"])
				if tc.status == "preview" {
					require.Equal(t, []any{}, payload["candidates"])
				} else {
					require.Equal(t, []any{}, payload["outcomes"])
				}
			}
		})
	}
}

// decodeBulkCompletionEnvelope requires exactly one result without trailing text or another envelope.
func decodeBulkCompletionEnvelope(t *testing.T, stdout string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(stdout))
	var envelope map[string]any
	require.NoError(t, decoder.Decode(&envelope), stdout)
	var extra any
	require.ErrorIs(t, decoder.Decode(&extra), io.EOF)
	require.Equal(t, "update job", envelope["command"])
	return envelope
}

// TestUpdateJobBulkCompletionHuman verifies compact and verbose previews and quiet streams.
func TestUpdateJobBulkCompletionHuman(t *testing.T) {
	for _, flags := range [][]string{{"--dry-run"}, {"--dry-run", "--verbose"}, {"--dry-run", "--quiet"}, {"--auto-confirm"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			fixture, server := newBulkCompletionFixture(t, "8.8", "normal")
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.8"), "--tenant", "tenant-a", "update", "job", "--type", "cleanup", "--complete"}
			stdout, stderr := executeRootForJobTestWithSeparateOutputs(t, append(args, flags...)...)
			if !strings.Contains(strings.Join(flags, " "), "json") {
				stdout, stderr = assertBulkCompletionHumanLogs(t, stdout, stderr)
			}
			require.Empty(t, stderr)
			if strings.Contains(strings.Join(flags, " "), "quiet") {
				require.Empty(t, stdout)
				require.Empty(t, fixture.writes)
				return
			}
			require.Contains(t, stdout, "selected job(s): 1\n")
			require.Contains(t, stdout, "selection scope: tenant-a only\n")
			require.NotContains(t, stdout, "POST")
			if flags[0] == "--dry-run" {
				require.Contains(t, stdout, jobCompletionListenerConsequence)
				require.Empty(t, fixture.writes)
			} else {
				require.Contains(t, stdout, "accepted 1; observed completed 0; unsatisfied 0; not started 0")
				require.Len(t, fixture.writes, 1)
			}
			if len(flags) > 1 && flags[1] == "--verbose" {
				require.Contains(t, stdout, "2251799813711967")
				require.Contains(t, stdout, "pi:2251799813711000")
			} else {
				require.NotContains(t, stdout, "2251799813711967")
			}
		})
	}
}

// TestUpdateJobBulkCompletionFailures preserves diagnostic candidates or complete ordered partial outcomes on nonzero exit.
func TestUpdateJobBulkCompletionFailures(t *testing.T) {
	for _, scenario := range []string{"incomplete", "partial", "variables"} {
		t.Run(scenario, func(t *testing.T) {
			fixture, server := newBulkCompletionFixture(t, "8.8", scenario)
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.8"), "--tenant", "tenant-a", "--json", "update", "job", "--type", "cleanup", "--complete", "--auto-confirm", "--workers", "1", "--batch-size", "1"}
			if scenario == "variables" {
				args = append(args, "--vars", `{"x":1}`)
			}
			encoded, _ := json.Marshal(args)
			stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestUpdateJobBulkCompletionHelper", map[string]string{"C8VOLT_BATCH_ARGS": string(encoded)})
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr)
			envelope := decodeBulkCompletionEnvelope(t, stdout)
			require.Contains(t, []string{"failed", "invalid"}, envelope["outcome"])
			require.NotEmpty(t, envelope["class"])
			payload := requireJSONObject(t, envelope["payload"])
			if scenario == "partial" {
				require.Equal(t, "finished", payload["status"])
				require.Len(t, payload["outcomes"], 2)
				require.Len(t, fixture.writes, 1)
			} else {
				require.Equal(t, "preparation_failed", payload["status"])
				require.NotContains(t, payload, "outcomes")
				require.Empty(t, fixture.writes)
			}
			require.NotContains(t, stderr, "Do you want to proceed?")
		})
	}
}

// TestUpdateJobBulkCompletionHelper isolates os.Exit paths and real stdin while
// preserving configured streams; an injected reader error exercises preflight failure.
func TestUpdateJobBulkCompletionHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_BATCH_ARGS")), &args))
	root := Root()
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.SetArgs(args)
	// This test-only handshake lets the terminal driver send EOF before parsing.
	if os.Getenv("C8VOLT_BATCH_INPUT_READY") == "1" {
		fmt.Fprintln(os.Stderr, "test input ready")
	}
	if os.Getenv("C8VOLT_BATCH_STDIN_ERROR") == "1" {
		root.SetIn(io.MultiReader(strings.NewReader("1\n"), iotest.ErrReader(fmt.Errorf("input failed"))))
	}
	if _, err := root.ExecuteC(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

// TestUpdateJobBulkCompletionConfirmation uses real terminal stdin with redirected
// stdout to prove confirmation routing, decline/EOF and prompt-free empty completion.
func TestUpdateJobBulkCompletionConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name, scenario, response string
		flags                    []string
		eof, failed              bool
	}{
		{name: "accept", scenario: "normal", response: "y"},
		{name: "decline", scenario: "normal", response: "n", failed: true},
		{name: "EOF", scenario: "normal", eof: true, failed: true},
		{name: "empty", scenario: "empty"},
		{name: "empty preview", scenario: "empty", flags: []string{"--dry-run"}},
		{name: "empty no wait", scenario: "empty", flags: []string{"--no-wait"}},
		{name: "empty quiet", scenario: "empty", flags: []string{"--quiet"}},
		{name: "empty JSON quiet", scenario: "empty", flags: []string{"--json", "--quiet", "--auto-confirm"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture, server := newBulkCompletionFixture(t, "8.8", tc.scenario)
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.8"), "--tenant", "tenant-a", "update", "job", "--type", "cleanup", "--complete"}
			args = append(args, tc.flags...)
			encoded, _ := json.Marshal(args)
			var exchanges []testx.CmdTerminalExchange
			prompt := "You are about to complete 1 selected job(s); selection scope: tenant-a only; filters: type=cleanup. Completing a listener job acknowledges its work without executing its business logic.\nDo you want to proceed? [y/N]: "
			if tc.scenario != "empty" {
				exchanges = []testx.CmdTerminalExchange{{Prompt: prompt, Response: tc.response, EndOfInput: tc.eof}}
			}
			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{ScopeTestName: "TestUpdateJobBulkCompletionHelper", Env: map[string]string{"C8VOLT_BATCH_ARGS": string(encoded)}, Exchanges: exchanges})
			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			if !strings.Contains(strings.Join(tc.flags, " "), "json") {
				prefix := ""
				if strings.HasPrefix(result.Stderr, prompt) {
					prefix = prompt
				}
				var diagnostics string
				result.Stdout, diagnostics = assertBulkCompletionHumanLogs(t, result.Stdout, strings.TrimPrefix(result.Stderr, prefix))
				result.Stderr = prefix + diagnostics
			}
			if tc.failed {
				var exitErr *exec.ExitError
				require.ErrorAs(t, result.Err, &exitErr)
				require.NotZero(t, exitErr.ExitCode())
				require.Contains(t, result.Stdout, "outcome: aborted; accepted 0; observed completed 0; unsatisfied 0; not started 1")
			} else {
				require.NoError(t, result.Err, result.Stderr)
			}
			require.NotContains(t, result.Stdout, "Do you want to proceed?")
			if tc.scenario == "empty" {
				require.Empty(t, result.Stderr)
				require.Zero(t, fixture.reads)
				require.Empty(t, fixture.writes)
				switch tc.name {
				case "empty JSON quiet":
					decodeBulkCompletionEnvelope(t, result.Stdout)
				case "empty quiet":
					require.Empty(t, result.Stdout)
				case "empty preview":
					require.Equal(t, "dry run: job completion\nfilters: type=cleanup\nconfigured tenant: none\n--tenant \"tenant-a\" sets the tenant filter\nselection scope: tenant-a only\nselected job(s): 0\n"+jobCompletionListenerConsequence+"\noutcome: preview; no changes applied\n", result.Stdout)
				default:
					require.Equal(t, "job completion\nfilters: type=cleanup\nconfigured tenant: none\n--tenant \"tenant-a\" sets the tenant filter\nselection scope: tenant-a only\nselected job(s): 0\noutcome: no matching jobs; no changes applied\n", result.Stdout)
				}
			} else {
				require.Equal(t, 1, strings.Count(result.Stderr, prompt))
				if tc.failed {
					require.Empty(t, fixture.writes)
					require.Zero(t, fixture.reads)
				} else {
					require.Len(t, fixture.writes, 1)
				}
			}
			require.NotContains(t, result.Stderr, "next page")
		})
	}
}

// TestUpdateJobBulkCompletionInvalidInputs verifies CLI guards reject before any HTTP call.
func TestUpdateJobBulkCompletionInvalidInputs(t *testing.T) {
	for _, flags := range [][]string{{"--total"}, {"--keys-only"}, {"--json"}, {"--json", "--dry-run", "--verbose"}, {"--key", "2251799813711967"}, {"--vars", "[]"}, {"--type", ""}, {"--limit", "-1"}, {"--complete=false"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			fixture, server := newBulkCompletionFixture(t, "8.8", "normal")
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.8"), "update", "job", "--type", "cleanup", "--complete"}
			encoded, _ := json.Marshal(append(args, flags...))
			_, _, err := testx.RunCmdSubprocessSeparate(t, "TestUpdateJobBulkCompletionHelper", map[string]string{"C8VOLT_BATCH_ARGS": string(encoded)})
			require.Error(t, err)
			require.Empty(t, fixture.calls)
		})
	}
}

// bulkCompletionCommandAPI exposes only prepare and execute, making accidental
// legacy reads or mutations fail while checking the command/facade boundary.
type bulkCompletionCommandAPI struct {
	job.API
	prepare func(context.Context, job.CompletionRequest, ...foptions.FacadeOption) (job.PreparedCompletionBatch, error)
	execute func(context.Context, job.PreparedCompletionBatch, int, ...foptions.FacadeOption) (job.CompletionBatchReport, error)
}

// PrepareJobCompletionBatch records the exact CLI request and options.
func (api bulkCompletionCommandAPI) PrepareJobCompletionBatch(ctx context.Context, request job.CompletionRequest, opts ...foptions.FacadeOption) (job.PreparedCompletionBatch, error) {
	return api.prepare(ctx, request, opts...)
}

// ExecuteJobCompletionBatch verifies unchanged frozen provenance and execution options.
func (api bulkCompletionCommandAPI) ExecuteJobCompletionBatch(ctx context.Context, batch job.PreparedCompletionBatch, workers int, opts ...foptions.FacadeOption) (job.CompletionBatchReport, error) {
	return api.execute(ctx, batch, workers, opts...)
}

// TestUpdateJobBulkCompletionOptions verifies workers and options reach both facade
// operations, and declined confirmation renders all unstarted rows without execution.
func TestUpdateJobBulkCompletionOptions(t *testing.T) {
	for _, decline := range []bool{false, true} {
		t.Run(fmt.Sprint(decline), func(t *testing.T) {
			resetUpdateJobFlagState()
			t.Cleanup(resetUpdateJobFlagState)
			flagViewAsJson = true
			flagNoWait = true
			flagFailFast = true
			flagNoWorkerLimit = true
			flagWorkers = 3
			previous := confirmCmdOrAbortFn
			t.Cleanup(func() { confirmCmdOrAbortFn = previous })
			request := job.CompletionRequest{Selection: job.CompletionSelection{Mode: job.CompletionSelectionSearch, InputForm: job.CompletionInputFilters, Filter: job.CompletionFilter{Type: "cleanup"}}}
			batch := job.PreparedCompletionBatch{Selection: request.Selection, Status: job.CompletionPreparationComplete, DiscoveryTenant: "tenant-a", Candidates: []job.CompletionCandidate{{Key: "2251799813711967", Kind: "TASK_LISTENER"}}}
			cmd := &cobra.Command{Use: "job"}
			parent := &cobra.Command{Use: "update"}
			parent.AddCommand(cmd)
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetContext(context.Background())
			confirmCmdOrAbortFn = func(writer io.Writer, _ bool, prompt string) error {
				require.Same(t, &stderr, writer)
				require.Contains(t, prompt, "1 selected job(s)")
				if decline {
					return localPreconditionError(ErrCmdAborted)
				}
				return nil
			}
			prepares, executes := 0, 0
			api := bulkCompletionCommandAPI{
				prepare: func(_ context.Context, got job.CompletionRequest, opts ...foptions.FacadeOption) (job.PreparedCompletionBatch, error) {
					prepares++
					require.Equal(t, request, got)
					options := foptions.ApplyFacadeOptions(opts)
					require.True(t, options.NoWait)
					require.True(t, options.FailFast)
					require.True(t, options.NoWorkerLimit)
					return batch, nil
				},
				execute: func(_ context.Context, got job.PreparedCompletionBatch, workers int, opts ...foptions.FacadeOption) (job.CompletionBatchReport, error) {
					executes++
					require.Equal(t, batch, got)
					require.Equal(t, 3, workers)
					options := foptions.ApplyFacadeOptions(opts)
					require.True(t, options.NoWait)
					require.True(t, options.FailFast)
					require.True(t, options.NoWorkerLimit)
					return job.CompletionBatchReport{Status: job.CompletionBatchFinished}, nil
				},
			}
			err := runUpdateJobBatch(cmd, api, testx.TestConfig(t), request)
			require.Equal(t, 1, prepares)
			var envelope map[string]any
			require.NoError(t, json.Unmarshal(stdout.Bytes(), &envelope))
			if decline {
				require.ErrorContains(t, err, ErrCmdAborted.Error())
				require.Zero(t, executes)
				payload := requireJSONObject(t, envelope["payload"])
				require.Equal(t, "aborted", payload["status"])
				rows := payload["outcomes"].([]any)
				require.Len(t, rows, 1)
				require.Equal(t, "declined", requireJSONObject(t, rows[0])["reason"])
			} else {
				require.NoError(t, err)
				require.Equal(t, 1, executes)
			}
		})
	}
}

// TestUpdateJobBulkCompletionAllPredicatesAndLimit verifies all nine AND predicates
// reach the adapter, zero retries survive, and the unique cap stays visible in preview.
func TestUpdateJobBulkCompletionAllPredicatesAndLimit(t *testing.T) {
	fixture, server := newBulkCompletionFixture(t, "8.8", "predicates")
	stdout, stderr := executeRootForJobTestWithSeparateOutputs(t, "--config", writeTestConfigForVersion(t, server.URL, "8.8"), "--tenant", "tenant-a", "update", "job", "--complete", "--dry-run", "--verbose", "--state", "created", "--type", "cleanup", "--pi-key", "2251799813711000", "--element-instance-key", "2251799813711001", "--element-id", "task", "--worker", "worker", "--retries", "0", "--kind", "task_listener", "--listener-event-type", "canceling", "--limit", "1", "--batch-size", "10")
	stdout, stderr = assertBulkCompletionHumanLogs(t, stdout, stderr)
	require.Empty(t, stderr)
	require.Contains(t, stdout, "selected job(s): 1")
	require.Contains(t, stdout, "discovery user-limited: limit 1; pages 1; batch size 10")
	require.Contains(t, stdout, "retries=0")
	require.Contains(t, stdout, "2251799813711967")
	require.Equal(t, 1, fixture.discovery)
	require.Zero(t, fixture.reads)
	require.Empty(t, fixture.writes)
}

// TestUpdateJobBulkCompletionPartialHuman keeps compact failures aggregate-first,
// exposes ordered context only under verbose, and retains errors under quiet.
func TestUpdateJobBulkCompletionPartialHuman(t *testing.T) {
	for _, flags := range [][]string{nil, {"--verbose"}, {"--quiet"}, {"--quiet", "--json"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			fixture, server := newBulkCompletionFixture(t, "8.8", "partial")
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.8"), "--tenant", "tenant-a", "update", "job", "--type", "cleanup", "--complete", "--auto-confirm", "--workers", "1"}
			encoded, _ := json.Marshal(append(args, flags...))
			stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestUpdateJobBulkCompletionHelper", map[string]string{"C8VOLT_BATCH_ARGS": string(encoded)})
			require.Error(t, err)
			if !strings.Contains(strings.Join(flags, " "), "json") {
				stdout, stderr = assertBulkCompletionHumanLogs(t, stdout, stderr)
			}
			require.Len(t, fixture.writes, 1)
			mode := strings.Join(flags, " ")
			if strings.Contains(mode, "json") {
				envelope := decodeBulkCompletionEnvelope(t, stdout)
				require.Equal(t, "failed", envelope["outcome"])
				return
			}
			require.Contains(t, stderr, "job completion:")
			if strings.Contains(mode, "quiet") {
				require.Empty(t, stdout)
				return
			}
			require.Contains(t, stdout, "outcome: failed; accepted 1; observed completed 0; unsatisfied 1; not started 0")
			if strings.Contains(mode, "verbose") {
				require.Contains(t, stdout, "disposition:unsatisfied")
				require.Less(t, strings.Index(stdout, "2251799813711967"), strings.Index(stdout, "2251799813711968"))
			} else {
				require.NotContains(t, stdout, "2251799813711967")
			}
		})
	}
}

// TestUpdateJobBulkCompletionExplicitCollections proves combined input order,
// service-owned canonical deduplication, precision, and the stable bulk schema.
func TestUpdateJobBulkCompletionExplicitCollections(t *testing.T) {
	for _, tc := range []struct {
		name, input, form string
		keys, want        []string
	}{
		{"repeated duplicate", "", "collection", []string{"--key", "1", "--key", "1"}, []string{"1"}},
		{"comma duplicate", "", "collection", []string{"--key", "1,01"}, []string{"1"}},
		{"precision", "", "collection", []string{"--key", "9223372036854775807,9007199254740993"}, []string{"9223372036854775807", "9007199254740993"}},
		{"stdin one", "1\n", "stdin", []string{"-"}, []string{"1"}},
		{"flags before stdin", "\n 2 \r\n1\n", "stdin", []string{"-", "--key", "001,1", "--key", "2"}, []string{"1", "2"}},
	} {
		for _, preview := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s preview=%t", tc.name, preview), func(t *testing.T) {
				fixture, server := newBulkCompletionFixture(t, "8.8", "normal")
				args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.8"), "update", "job", "--complete", "--auto-confirm", "--json", "--workers", "1"}
				args = append(args, tc.keys...)
				if preview {
					args = append(args, "--dry-run")
				}
				encoded, _ := json.Marshal(args)
				stdout, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, "TestUpdateJobBulkCompletionHelper", "", map[string]string{"C8VOLT_BATCH_ARGS": string(encoded)}, tc.input)
				require.NoError(t, err, stderr)
				require.Empty(t, stderr)
				envelope := decodeBulkCompletionEnvelope(t, stdout)
				payload := requireJSONObject(t, envelope["payload"])
				selection := requireJSONObject(t, payload["selection"])
				require.Equal(t, "explicit", selection["mode"])
				require.Equal(t, tc.form, selection["inputForm"])
				require.Equal(t, "explicit_inputs_read", payload["completionReason"])
				require.Zero(t, fixture.discovery)
				require.Equal(t, float64(len(tc.want)), requireJSONObject(t, payload["counts"])["selected"])
				field, omitted := "outcomes", "candidates"
				if preview {
					field, omitted = omitted, field
				}
				require.NotContains(t, payload, omitted)
				rows := payload[field].([]any)
				for i, row := range rows {
					candidate := requireJSONObject(t, row)
					if !preview {
						candidate = requireJSONObject(t, candidate["candidate"])
					}
					require.Equal(t, tc.want[i], candidate["key"])
				}
				if preview {
					require.Equal(t, tc.want, fixture.lookupKeys)
					require.Empty(t, fixture.writes)
				} else {
					require.Equal(t, append(append([]string{}, tc.want...), tc.want...), fixture.lookupKeys)
					require.Len(t, fixture.writes, len(tc.want))
				}
			})
		}
	}
}

// TestUpdateJobBulkCompletionExplicitTenants retains authorized keys outside the
// discovery tenant and reports actual context without implying filtered selection.
func TestUpdateJobBulkCompletionExplicitTenants(t *testing.T) {
	for _, flags := range [][]string{{"--dry-run", "--verbose"}, {"--auto-confirm"}, {"--auto-confirm", "--json"}, {"--automation", "--json", "--quiet", "--no-wait"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			fixture, server := newBulkCompletionFixture(t, "8.8", "explicit")
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.8"), "--tenant", "tenant-a", "update", "job", "--key", "1,2", "--complete", "--workers", "1"}
			stdout, stderr := executeRootForJobTestWithSeparateOutputs(t, append(args, flags...)...)
			if !strings.Contains(strings.Join(flags, " "), "json") {
				stdout, stderr = assertBulkCompletionHumanLogs(t, stdout, stderr)
			}
			require.Empty(t, stderr)
			require.Zero(t, fixture.discovery)
			if strings.Contains(strings.Join(flags, " "), "json") {
				envelope := decodeBulkCompletionEnvelope(t, stdout)
				ctx := requireJSONObject(t, envelope["tenantContext"])
				require.Equal(t, "explicit_keys", ctx["mode"])
				require.Equal(t, "not_applied", ctx["filter"])
				require.Equal(t, []any{"tenant-a", "tenant-b"}, ctx["resolvedTenantIds"])
				require.Equal(t, true, ctx["crossTenant"])
			} else {
				require.Contains(t, stdout, "selection: explicit keys\n")
				require.Contains(t, stdout, "affected tenants: tenant-a, tenant-b\n")
				require.Contains(t, stdout, "selected job(s): 2\n")
				require.NotContains(t, stdout, "filters:")
				require.NotContains(t, stdout, "discovery:")
				require.NotContains(t, stdout, "selection scope: tenant-a only")
			}
			if flags[0] == "--dry-run" {
				require.Empty(t, fixture.writes)
			} else {
				require.Len(t, fixture.writes, 2)
			}
		})
	}
}

// TestUpdateJobBulkCompletionExplicitFailures rejects incomplete input or a
// missing explicit target before any mutation and retains preparation evidence.
func TestUpdateJobBulkCompletionExplicitFailures(t *testing.T) {
	for _, tc := range []struct {
		name, scenario, stdin string
		flags                 []string
		readError, prepared   bool
	}{
		{name: "malformed late stdin", stdin: "1\n2\nbad\n", flags: []string{"-"}},
		{name: "read failure", flags: []string{"-"}, readError: true},
		{name: "empty comma", flags: []string{"--key", "1,"}},
		{name: "empty scalar", flags: []string{"--key", ""}},
		{name: "missing target", scenario: "explicit missing", flags: []string{"--key", "1,2"}, prepared: true},
		{name: "listener variables", flags: []string{"--key", "1,2", "--vars", `{"x":1}`}, prepared: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture, server := newBulkCompletionFixture(t, "8.8", tc.scenario)
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.8"), "update", "job", "--complete", "--auto-confirm", "--json"}
			encoded, _ := json.Marshal(append(args, tc.flags...))
			env := map[string]string{"C8VOLT_BATCH_ARGS": string(encoded)}
			if tc.readError {
				env["C8VOLT_BATCH_STDIN_ERROR"] = "1"
			}
			stdout, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, "TestUpdateJobBulkCompletionHelper", "", env, tc.stdin)
			require.Error(t, err)
			envelope := decodeBulkCompletionEnvelope(t, stdout)
			require.Contains(t, []string{"invalid", "failed"}, envelope["outcome"])
			require.Empty(t, fixture.writes)
			require.Zero(t, fixture.discovery)
			require.NotContains(t, stderr, "Do you want to proceed?")
			if tc.prepared {
				payload := requireJSONObject(t, envelope["payload"])
				require.Equal(t, "preparation_failed", payload["status"])
				require.Contains(t, payload, "candidates")
				require.NotContains(t, payload, "outcomes")
			} else {
				require.Empty(t, fixture.calls)
			}
		})
	}
}

// TestUpdateJobBulkCompletionEmptyStdin proves deliberate empty input is an
// authoritative no-op across result modes and never claims accepted work.
func TestUpdateJobBulkCompletionEmptyStdin(t *testing.T) {
	for _, flags := range [][]string{nil, {"--dry-run"}, {"--no-wait"}, {"--quiet"}, {"--json", "--auto-confirm"}, {"--json", "--quiet", "--automation", "--no-wait"}, {"--json", "--quiet", "--dry-run"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			fixture, server := newBulkCompletionFixture(t, "8.8", "normal")
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.8"), "update", "job", "-", "--complete"}
			encoded, _ := json.Marshal(append(args, flags...))
			stdout, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, "TestUpdateJobBulkCompletionHelper", "", map[string]string{"C8VOLT_BATCH_ARGS": string(encoded)}, " \n\t\r\n")
			require.NoError(t, err, stderr)
			if !strings.Contains(strings.Join(flags, " "), "json") {
				stdout, stderr = assertBulkCompletionHumanLogs(t, stdout, stderr)
			}
			require.Empty(t, stderr)
			require.Empty(t, fixture.calls)
			mode := strings.Join(flags, " ")
			if strings.Contains(mode, "json") {
				envelope := decodeBulkCompletionEnvelope(t, stdout)
				require.Equal(t, "succeeded", envelope["outcome"])
				payload := requireJSONObject(t, envelope["payload"])
				field, omitted := "outcomes", "candidates"
				if strings.Contains(mode, "dry-run") {
					field, omitted = omitted, field
				}
				require.Equal(t, []any{}, payload[field])
				require.NotContains(t, payload, omitted)
				for _, count := range requireJSONObject(t, payload["counts"]) {
					require.Equal(t, float64(0), count)
				}
			} else if strings.Contains(mode, "quiet") {
				require.Empty(t, stdout)
			} else {
				want := "job completion\nselection: explicit keys\nselection scope: explicit resource keys; tenant filter not applied\nselected job(s): 0\n"
				if strings.Contains(mode, "dry-run") {
					want = "dry run: " + want + jobCompletionListenerConsequence + "\noutcome: preview; no changes applied\n"
				} else {
					want += "outcome: no selected jobs; no changes applied\n"
				}
				require.Equal(t, want, stdout)
			}
		})
	}
}

// TestUpdateJobBulkCompletionExplicitGuardrails rejects machine-output misuse and
// input-mode conflicts before any lookup, including deliberately empty stdin.
func TestUpdateJobBulkCompletionExplicitGuardrails(t *testing.T) {
	for _, flags := range [][]string{
		{"--key", "1,2", "--json"}, {"--key", "1,2", "--json", "--dry-run", "--verbose"},
		{"--key", "1,2", "--keys-only"}, {"--key", "1,2", "--limit", "0"},
		{"--key", "1,2", "--batch-size", "10"}, {"--key", "1,2", "--type", "cleanup"},
		{"-", "--json"}, {"-", "--retries", "0"}, {"-", "--kind", "TASK_LISTENER"},
		{"--key", "1,2", "--fail"}, {"--key", "1,2", "--timeout", "1m"},
		{"--key", "1,2", "--throw-bpmn-error", "ERROR"},
	} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			fixture, server := newBulkCompletionFixture(t, "8.8", "normal")
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.8"), "update", "job", "--complete"}
			encoded, _ := json.Marshal(append(args, flags...))
			_, _, err := testx.RunCmdSubprocessSeparate(t, "TestUpdateJobBulkCompletionHelper", map[string]string{"C8VOLT_BATCH_ARGS": string(encoded)})
			require.Error(t, err)
			require.Empty(t, fixture.calls)
		})
	}
}

// TestUpdateJobBulkCompletionExplicitConfirmation uses terminal stdin to prove a
// frozen cross-tenant collection gets one accurate stderr confirmation decision.
func TestUpdateJobBulkCompletionExplicitConfirmation(t *testing.T) {
	for _, answer := range []string{"y", "n"} {
		t.Run(answer, func(t *testing.T) {
			fixture, server := newBulkCompletionFixture(t, "8.8", "explicit")
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.8"), "--tenant", "tenant-a", "update", "job", "--key", "1,2,1", "--complete", "--workers", "1"}
			encoded, _ := json.Marshal(args)
			prompt := "You are about to complete 2 explicitly selected job(s); selection scope: explicit resource keys; tenant filter not applied; affected tenants: tenant-a, tenant-b. " + jobCompletionListenerConsequence + "\nDo you want to proceed? [y/N]: "
			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{ScopeTestName: "TestUpdateJobBulkCompletionHelper", Env: map[string]string{"C8VOLT_BATCH_ARGS": string(encoded)}, Exchanges: []testx.CmdTerminalExchange{{Prompt: prompt, Response: answer}}})
			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			if true {
				prefix := ""
				if strings.HasPrefix(result.Stderr, prompt) {
					prefix = prompt
				}
				var diagnostics string
				result.Stdout, diagnostics = assertBulkCompletionHumanLogs(t, result.Stdout, strings.TrimPrefix(result.Stderr, prefix))
				result.Stderr = prefix + diagnostics
			}
			require.NotContains(t, result.Stdout, "Do you want to proceed?")
			require.Equal(t, 1, strings.Count(result.Stderr, prompt))
			if answer == "y" {
				require.NoError(t, result.Err, result.Stderr)
				require.Len(t, fixture.writes, 2)
			} else {
				require.Error(t, result.Err)
				require.Empty(t, fixture.writes)
				require.Equal(t, []string{"1", "2"}, fixture.lookupKeys)
				require.Contains(t, result.Stdout, "not started 2")
			}
		})
	}
}

// TestUpdateJobBulkCompletionEmptyStdinTerminal supplies genuine terminal EOF
// before parsing and proves empty input does not request mutation confirmation.
func TestUpdateJobBulkCompletionEmptyStdinTerminal(t *testing.T) {
	for _, flags := range [][]string{nil, {"--dry-run"}, {"--no-wait"}, {"--quiet", "--json", "--auto-confirm"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			fixture, server := newBulkCompletionFixture(t, "8.8", "normal")
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.8"), "update", "job", "-", "--complete"}
			encoded, _ := json.Marshal(append(args, flags...))
			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{ScopeTestName: "TestUpdateJobBulkCompletionHelper", Env: map[string]string{"C8VOLT_BATCH_ARGS": string(encoded), "C8VOLT_BATCH_INPUT_READY": "1"}, Exchanges: []testx.CmdTerminalExchange{{Prompt: "test input ready\n", EndOfInput: true}}})
			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			if !strings.Contains(strings.Join(flags, " "), "json") {
				prefix := ""
				if strings.HasPrefix(result.Stderr, "test input ready\n") {
					prefix = "test input ready\n"
				}
				var diagnostics string
				result.Stdout, diagnostics = assertBulkCompletionHumanLogs(t, result.Stdout, strings.TrimPrefix(result.Stderr, prefix))
				result.Stderr = prefix + diagnostics
			}
			require.NoError(t, result.Err, result.Stderr)
			require.Equal(t, "test input ready\n", result.Stderr)
			require.NotContains(t, result.Stdout, "Do you want to proceed?")
			require.Empty(t, fixture.calls)
			if len(flags) > 0 && flags[0] == "--quiet" {
				decodeBulkCompletionEnvelope(t, result.Stdout)
			} else {
				require.Contains(t, result.Stdout, "selected job(s): 0\n")
			}
		})
	}
}
