// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const updateProcessInstanceOutcomesHelper = "TestUpdateProcessInstanceOutcomesHelper"

// processInstanceOutcomeCapture records the command's distinct planning,
// mutation, and verification phases without relying on rendered output.
type processInstanceOutcomeCapture struct {
	mu            sync.Mutex
	scenario      string
	mutated       map[string]bool
	variableReads map[string]int
	mutations     []string
	mutationBody  []map[string]any
}

// TestUpdateProcessInstanceOutcomeConfirmation exercises delayed and paged
// confirmation through the real command while proving no redundant final read.
func TestUpdateProcessInstanceOutcomeConfirmation(t *testing.T) {
	tests := []struct {
		name      string
		scenario  string
		wantReads int
	}{
		{name: "normal wait", scenario: "normal", wantReads: 2},
		{name: "delayed confirmation", scenario: "delayed", wantReads: 3},
		{name: "multi-page confirmation", scenario: "multi-page", wantReads: 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, capture := newProcessInstanceOutcomeServer(t, test.scenario)
			stdout, stderr, err := runProcessInstanceOutcomeCommand(t, server,
				"--automation", "--json", "--backoff-max-retries", "2",
				"update", "pi", "--key", "101", "--vars", `{"approved":true}`,
			)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			envelope := decodeSingleProcessInstanceOutcomeEnvelope(t, stdout)
			require.Equal(t, "succeeded", envelope["outcome"])
			item := requireJSONObject(t, requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 1)[0])
			require.Equal(t, "confirmed", item["status"])

			capture.mu.Lock()
			defer capture.mu.Unlock()
			require.Equal(t, test.wantReads, capture.variableReads["101"])
			require.Equal(t, []string{"101"}, capture.mutations)
		})
	}
}

// TestUpdateProcessInstanceOutcomeNoWait proves no-wait keeps complete planning
// and selective writes but performs no confirmation search.
func TestUpdateProcessInstanceOutcomeNoWait(t *testing.T) {
	server, capture := newProcessInstanceOutcomeServer(t, "normal")
	stdout, stderr, err := runProcessInstanceOutcomeCommand(t, server,
		"--automation", "--json", "update", "pi", "--key", "101",
		"--vars", `{"approved":true}`, "--no-wait",
	)
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	envelope := decodeSingleProcessInstanceOutcomeEnvelope(t, stdout)
	require.Equal(t, "accepted", envelope["outcome"])
	item := requireJSONObject(t, requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 1)[0])
	require.Equal(t, "submitted", item["status"])

	capture.mu.Lock()
	defer capture.mu.Unlock()
	require.Equal(t, 1, capture.variableReads["101"])
	require.Equal(t, []string{"101"}, capture.mutations)
}

// TestUpdateProcessInstanceOutcomeConfirmationFailures retains acceptance for
// truncated, wrong-scope, and drifted observations in one failure envelope.
func TestUpdateProcessInstanceOutcomeConfirmationFailures(t *testing.T) {
	for _, scenario := range []string{"truncated", "wrong-scope", "unchanged-drift"} {
		t.Run(scenario, func(t *testing.T) {
			server, capture := newProcessInstanceOutcomeServer(t, scenario)
			variables := `{"approved":true}`
			if scenario == "unchanged-drift" {
				variables = `{"approved":true,"stable":"same"}`
			}
			stdout, stderr, err := runProcessInstanceOutcomeCommand(t, server,
				"--automation", "--json", "--backoff-max-retries", "1",
				"update", "pi", "--key", "101", "--vars", variables,
			)
			requireCommandExitCode(t, err, exitcode.Error)
			require.Empty(t, stderr)
			envelope := decodeSingleProcessInstanceOutcomeEnvelope(t, stdout)
			require.Equal(t, "failed", envelope["outcome"])
			item := requireJSONObject(t, requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 1)[0])
			require.Equal(t, "confirmation_failed", item["status"])
			require.Equal(t, true, item["mutationAccepted"])

			capture.mu.Lock()
			defer capture.mu.Unlock()
			require.Equal(t, 2, capture.variableReads["101"])
			require.Equal(t, []string{"101"}, capture.mutations)
			require.Equal(t, map[string]any{"approved": true}, capture.mutationBody[0]["variables"])
		})
	}
}

// TestUpdateProcessInstanceOutcomePartialWriterFailure preserves accepted and
// rejected roots together and emits exactly one JSON failure value.
func TestUpdateProcessInstanceOutcomePartialWriterFailure(t *testing.T) {
	server, capture := newProcessInstanceOutcomeServer(t, "partial-writer")
	stdout, stderr, err := runProcessInstanceOutcomeCommand(t, server,
		"--automation", "--json", "update", "pi", "--key", "101", "--key", "202",
		"--vars", `{"approved":true}`, "--no-wait", "--workers", "1",
	)
	requireCommandExitCode(t, err, exitcode.Error)
	require.Empty(t, stderr)
	envelope := decodeSingleProcessInstanceOutcomeEnvelope(t, stdout)
	require.Equal(t, "failed", envelope["outcome"])
	items := requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 2)
	require.Equal(t, "submitted", requireJSONObject(t, items[0])["status"])
	require.Equal(t, "mutation_failed", requireJSONObject(t, items[1])["status"])

	capture.mu.Lock()
	defer capture.mu.Unlock()
	require.Equal(t, []string{"101", "202"}, capture.mutations)
	require.Equal(t, 1, capture.variableReads["101"])
	require.Equal(t, 1, capture.variableReads["202"])
}

// TestUpdateProcessInstanceOutcomeKeysRetainAcceptedConfirmationFailure proves
// keys-only emits accepted work even when its later confirmation fails.
func TestUpdateProcessInstanceOutcomeKeysRetainAcceptedConfirmationFailure(t *testing.T) {
	server, _ := newProcessInstanceOutcomeServer(t, "truncated")
	stdout, _, err := runProcessInstanceOutcomeCommand(t, server,
		"--automation", "--keys-only", "--backoff-max-retries", "1",
		"update", "pi", "--key", "101", "--vars", `{"approved":true}`,
	)
	requireCommandExitCode(t, err, exitcode.Error)
	require.Equal(t, "101\n", stdout)
}

// TestUpdateProcessInstanceOutcomeFailFastSkipped exposes unstarted work in the
// same failed result as the mutation that stopped scheduling.
func TestUpdateProcessInstanceOutcomeFailFastSkipped(t *testing.T) {
	server, capture := newProcessInstanceOutcomeServer(t, "fail-fast")
	stdout, stderr, err := runProcessInstanceOutcomeCommand(t, server,
		"--automation", "--json", "update", "pi",
		"--key", "101", "--key", "202", "--key", "303",
		"--vars", `{"approved":true}`, "--no-wait", "--workers", "1", "--fail-fast",
	)
	requireCommandExitCode(t, err, exitcode.Error)
	require.Empty(t, stderr)
	envelope := decodeSingleProcessInstanceOutcomeEnvelope(t, stdout)
	items := requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 3)
	require.Equal(t, "mutation_failed", requireJSONObject(t, items[0])["status"])
	require.Equal(t, "skipped", requireJSONObject(t, items[1])["status"])
	require.Equal(t, "skipped", requireJSONObject(t, items[2])["status"])

	capture.mu.Lock()
	defer capture.mu.Unlock()
	require.Equal(t, []string{"101"}, capture.mutations)
}

// TestUpdateProcessInstanceOutcomeHumanModes keeps partial outcome detail in
// normal and debug output while quiet mode suppresses informational rendering.
func TestUpdateProcessInstanceOutcomeHumanModes(t *testing.T) {
	tests := []struct {
		name      string
		rootFlag  string
		wantHuman bool
		wantHTTP  bool
	}{
		{name: "compact", wantHuman: true},
		{name: "quiet", rootFlag: "--quiet"},
		{name: "debug", rootFlag: "--debug", wantHuman: true, wantHTTP: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, _ := newProcessInstanceOutcomeServer(t, "partial-writer")
			args := []string{"--auto-confirm"}
			if test.rootFlag != "" {
				args = append(args, test.rootFlag)
			}
			args = append(args, "update", "pi", "--key", "101", "--key", "202", "--vars", `{"approved":true}`, "--no-wait", "--workers", "1")
			stdout, stderr, err := runProcessInstanceOutcomeCommand(t, server, args...)
			requireCommandExitCode(t, err, exitcode.Error)
			require.Empty(t, stdout)
			require.Equal(t, test.wantHuman, strings.Contains(stderr, "updated process-instance 101: submitted"))
			require.Equal(t, test.wantHuman, strings.Contains(stderr, "updated process-instance 202: mutation failed:"))
			require.Equal(t, test.wantHTTP, strings.Contains(stderr, "DEBUG api #"))
			if test.wantHuman {
				require.Contains(t, stderr, "updated: 2 (mutation accepted: 1, unchanged: 0, failed: 1, skipped: 0)")
				require.NotContains(t, stderr, "ERROR update process-instance variables")
			}
		})
	}
}

// TestRenderUpdateProcessInstanceOutcomeStates verifies every additive state
// and accepted confirmation failure contributes to the truthful human summary.
func TestRenderUpdateProcessInstanceOutcomeStates(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	cmd := &cobra.Command{Use: "process-instance"}
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	results := process.ProcessInstanceVariableUpdateResults{Items: []process.ProcessInstanceVariableUpdateResult{
		{Key: "confirmed", Status: process.ProcessInstanceVariableUpdateStatusConfirmed, MutationAccepted: true},
		{Key: "confirmation-failed", Status: process.ProcessInstanceVariableUpdateStatusConfirmationFailed, MutationAccepted: true, Error: "missing approved"},
		{Key: "unchanged", Status: process.ProcessInstanceVariableUpdateStatusUnchanged},
		{Key: "mutation-failed", Status: process.ProcessInstanceVariableUpdateStatusMutationFailed, Error: "denied"},
		{Key: "skipped", Status: process.ProcessInstanceVariableUpdateStatusSkipped},
	}}
	require.NoError(t, renderUpdateProcessInstanceVariableFailure(cmd, results, fmt.Errorf("partial update")))
	require.Equal(t,
		"updated process-instance confirmed: confirmed\n"+
			"updated process-instance confirmation-failed: confirmation failed: missing approved\n"+
			"updated process-instance unchanged: unchanged\n"+
			"updated process-instance mutation-failed: mutation failed: denied\n"+
			"updated process-instance skipped: skipped\n"+
			"updated: 5 (mutation accepted: 2, unchanged: 1, failed: 2, skipped: 1)\n",
		output.String(),
	)
}

// TestUpdateProcessInstanceOutcomesHelper isolates command paths that terminate
// through the established exit-code implementation.
func TestUpdateProcessInstanceOutcomesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	runUpdateProcessInstanceHelperFromEnv(t)
	os.Exit(0)
}

// runProcessInstanceOutcomeCommand captures stdout and stderr independently
// while allowing production error paths to exit the helper subprocess.
func runProcessInstanceOutcomeCommand(t *testing.T, server *httptest.Server, args ...string) (string, string, error) {
	t.Helper()
	encodedArgs, err := json.Marshal(args)
	require.NoError(t, err)
	return testx.RunCmdSubprocessInDirWithSeparateOutputs(t, updateProcessInstanceOutcomesHelper, "", map[string]string{
		"C8VOLT_TEST_CONFIG":      writeTestConfigForVersion(t, server.URL, "8.10"),
		"C8VOLT_TEST_UPDATE_ARGS": string(encodedArgs),
	}, "")
}

// decodeSingleProcessInstanceOutcomeEnvelope requires exactly one JSON value,
// preventing a generic error envelope from following the partial result.
func decodeSingleProcessInstanceOutcomeEnvelope(t *testing.T, output string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewBufferString(output))
	var envelope map[string]any
	require.NoError(t, decoder.Decode(&envelope))
	require.ErrorIs(t, decoder.Decode(&struct{}{}), io.EOF)
	return envelope
}

// newProcessInstanceOutcomeServer supplies deterministic outcome states while
// preserving actual adapter paging, root filtering, and local write requests.
func newProcessInstanceOutcomeServer(t *testing.T, scenario string) (*httptest.Server, *processInstanceOutcomeCapture) {
	t.Helper()
	capture := &processInstanceOutcomeCapture{
		scenario:      scenario,
		mutated:       make(map[string]bool),
		variableReads: make(map[string]int),
	}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if serveVersionCheckTopology(writer, request, "8.10") {
			return
		}

		switch {
		case request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(request.URL.Path, "/v2/process-instances/")
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"processInstanceKey":%q,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionVersion":1,"state":"ACTIVE"}`, key)
		case request.Method == http.MethodPost && request.URL.Path == "/v2/variables/search":
			serveProcessInstanceOutcomeVariables(t, writer, request, capture)
		case request.Method == http.MethodPut && strings.HasPrefix(request.URL.Path, "/v2/element-instances/") && strings.HasSuffix(request.URL.Path, "/variables"):
			key := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, "/v2/element-instances/"), "/variables")
			var body map[string]any
			require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
			capture.mu.Lock()
			capture.mutations = append(capture.mutations, key)
			capture.mutationBody = append(capture.mutationBody, body)
			capture.mutated[key] = true
			capture.mu.Unlock()
			if scenario == "partial-writer" && key == "202" || scenario == "fail-fast" && key == "101" {
				http.Error(writer, `{"message":"writer rejected update"}`, http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected outcome request: %s %s", request.Method, request.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server, capture
}

// serveProcessInstanceOutcomeVariables changes only post-write observations so
// planning remains stable and verification scenarios stay independently clear.
func serveProcessInstanceOutcomeVariables(t *testing.T, writer http.ResponseWriter, request *http.Request, capture *processInstanceOutcomeCapture) {
	t.Helper()
	var body struct {
		Filter struct {
			ProcessInstanceKey string `json:"processInstanceKey"`
		} `json:"filter"`
		Page struct {
			After *string `json:"after"`
		} `json:"page"`
	}
	require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
	key := body.Filter.ProcessInstanceKey
	require.NotEmpty(t, key)
	capture.mu.Lock()
	capture.variableReads[key]++
	read := capture.variableReads[key]
	mutated := capture.mutated[key]
	scenario := capture.scenario
	capture.mu.Unlock()

	writer.Header().Set("Content-Type", "application/json")
	if !mutated {
		stable := ""
		if scenario == "unchanged-drift" {
			stable = fmt.Sprintf(`,{"name":"stable","value":"\"same\"","variableKey":"stable-%s","processInstanceKey":%q,"scopeKey":%q,"tenantId":"tenant-a"}`, key, key, key)
		}
		_, _ = fmt.Fprintf(writer, `{"items":[{"name":"approved","value":"false","variableKey":"approved-%s","processInstanceKey":%q,"scopeKey":%q,"tenantId":"tenant-a"}%s],"page":{"totalItems":%d,"hasMoreTotalItems":false}}`, key, key, key, stable, 1+boolToInt(stable != ""))
		return
	}

	switch scenario {
	case "delayed":
		value := "true"
		if read == 2 {
			value = "false"
		}
		writeProcessInstanceOutcomePage(writer, key, key, value, false, false)
	case "multi-page":
		if body.Page.After == nil {
			_, _ = fmt.Fprintf(writer, `{"items":[{"name":"approved","value":"true","variableKey":"nested-%s","processInstanceKey":%q,"scopeKey":"child-%s","tenantId":"tenant-a"}],"page":{"totalItems":2,"hasMoreTotalItems":true,"endCursor":"next"}}`, key, key, key)
			return
		}
		_, _ = fmt.Fprintf(writer, `{"items":[{"name":"approved","value":"true","variableKey":"approved-%s","processInstanceKey":%q,"scopeKey":%q,"tenantId":"tenant-a"}],"page":{"totalItems":2,"hasMoreTotalItems":false,"endCursor":"done"}}`, key, key, key)
	case "truncated":
		writeProcessInstanceOutcomePage(writer, key, key, "true", true, false)
	case "wrong-scope":
		writeProcessInstanceOutcomePage(writer, key, "child-"+key, "true", false, false)
	case "unchanged-drift":
		_, _ = fmt.Fprintf(writer, `{"items":[{"name":"approved","value":"true","variableKey":"approved-%s","processInstanceKey":%q,"scopeKey":%q,"tenantId":"tenant-a"},{"name":"stable","value":"\"changed\"","variableKey":"stable-%s","processInstanceKey":%q,"scopeKey":%q,"tenantId":"tenant-a"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`, key, key, key, key, key, key)
	default:
		writeProcessInstanceOutcomePage(writer, key, key, "true", false, false)
	}
}

// writeProcessInstanceOutcomePage emits one raw variable observation with the
// exact scope and truncation evidence needed by the waiter contract.
func writeProcessInstanceOutcomePage(writer http.ResponseWriter, key, scope, value string, truncated, hasMore bool) {
	_, _ = fmt.Fprintf(writer, `{"items":[{"name":"approved","value":%q,"variableKey":"approved-%s","processInstanceKey":%q,"scopeKey":%q,"tenantId":"tenant-a","isTruncated":%t}],"page":{"totalItems":1,"hasMoreTotalItems":%t}}`, value, key, key, scope, truncated, hasMore)
}

// boolToInt keeps fixture total metadata consistent with its emitted items.
func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
