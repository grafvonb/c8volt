// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestUpdateJobAssignmentWire preserves input provenance and independent evidence
// while exercising the actual facade, adapter and one-envelope error exit path.
func TestUpdateJobAssignmentWire(t *testing.T) {
	for _, noWait := range []bool{false, true} {
		t.Run(fmt.Sprint("no-wait=", noWait), func(t *testing.T) { testUpdateJobAssignmentWire(t, noWait) })
	}
}

// testUpdateJobAssignmentWire covers normal and bounded no-wait observation.
func testUpdateJobAssignmentWire(t *testing.T, noWait bool) {
	for _, shape := range []string{"scalar", "collection", "selector"} {
		for _, outcome := range []string{"accepted", "rejected", "unknown", "preview"} {
			t.Run(shape+"/"+outcome, func(t *testing.T) {
				var writes atomic.Int32
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if serveVersionCheckTopology(w, r, "8.10") {
						return
					}
					w.Header().Set("Content-Type", "application/json")
					if r.URL.Path == "/v2/jobs/search" {
						state := "CREATED"
						if writes.Load() > 0 {
							state = "COMPLETED"
						}
						fmt.Fprint(w, strings.Replace(jobSearchResponseWithState(scalarJobKey, 0, state), `"state":`, `"businessId":"PRIOR","state":`, 1))
						return
					}
					require.Equal(t, "/v2/jobs/"+scalarJobKey+"/completion", r.URL.Path)
					writes.Add(1)
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.Equal(t, " Order,007 ", body["businessId"])
					require.Equal(t, map[string]any{"approved": true}, body["variables"])
					switch outcome {
					case "rejected":
						w.WriteHeader(400)
						fmt.Fprint(w, `{"detail":"POST https://private.invalid/secret body-only-secret"}`)
					case "unknown":
						c, _, err := w.(http.Hijacker).Hijack()
						require.NoError(t, err)
						c.Close()
					default:
						w.WriteHeader(204)
					}
				}))
				defer server.Close()
				args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.10"), "update", "job", "--complete", "--set-bid", " Order,007 ", "--vars", `{"approved":true}`, "--json", "--auto-confirm"}
				if noWait {
					args = append(args, "--no-wait")
				}
				switch shape {
				case "scalar":
					args = append(args, "--key", scalarJobKey)
				case "collection":
					args = append(args, "--key", scalarJobKey+","+scalarJobKey)
				default:
					args = append(args, "--retries", "0")
				}
				if outcome == "preview" {
					args = append(args, "--dry-run")
				}
				stdout, stderr, err := runBulkCompletionTerminal(t, args, "configured", true, false, nil)
				if outcome == "unknown" || outcome == "rejected" {
					var e *exec.ExitError
					require.ErrorAs(t, err, &e)
					expected := 2
					if outcome == "unknown" {
						expected = 1
					}
					require.Equal(t, expected, e.ExitCode())
				} else {
					require.NoError(t, err, stderr)
				}
				envelope := decodeBulkCompletionEnvelope(t, stdout)
				payload := requireJSONObject(t, envelope["payload"])
				require.Equal(t, " Order,007 ", payload["requestedBusinessId"])
				if outcome == "preview" {
					require.Zero(t, writes.Load())
					if shape == "scalar" {
						require.Equal(t, "PRIOR", requireJSONObject(t, payload["current"])["businessId"])
					} else {
						require.Equal(t, "PRIOR", requireJSONObject(t, payload["candidates"].([]any)[0])["businessId"])
					}
					return
				}
				require.Equal(t, int32(1), writes.Load())
				evidence := payload
				if shape != "scalar" {
					outcomes := payload["outcomes"].([]any)
					require.Len(t, outcomes, 1)
					evidence = requireJSONObject(t, outcomes[0])
					require.Equal(t, "PRIOR", requireJSONObject(t, evidence["candidate"])["businessId"])
				} else {
					require.NotContains(t, payload, "outcomes")
					require.Equal(t, "PRIOR", requireJSONObject(t, requireJSONObject(t, payload["plan"])["current"])["businessId"])
				}
				require.Equal(t, outcome, requireJSONObject(t, evidence["assignment"])["status"])
				if outcome == "unknown" {
					require.Equal(t, "completed", requireJSONObject(t, evidence["observation"])["status"])
				}
			})
		}
	}
}

// TestUpdateJobAssignmentParsing rejects invalid intent before any client exists.
func TestUpdateJobAssignmentParsing(t *testing.T) {
	for _, flags := range [][]string{
		{"--key", scalarJobKey, "--retries", "1", "--set-bid", "x"},
		{"--key", scalarJobKey, "--timeout", "1m", "--set-bid", "x"},
		{"--key", scalarJobKey, "--fail", "--retries", "0", "--set-bid", "x"},
		{"--key", scalarJobKey, "--throw-bpmn-error", "x", "--set-bid", "x"},
		{"--complete", "--set-bid", "x"},
		{"--key", scalarJobKey, "--complete", "--set-bid", ""},
		{"--key", scalarJobKey, "--complete", "--set-bid", "   "},
		{"--key", scalarJobKey, "--complete", "--set-bid", strings.Repeat("😀", 129)},
	} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			resetUpdateJobFlagState()
			resetCommandTreeFlags(Root())
			t.Cleanup(func() { resetUpdateJobFlagState(); resetCommandTreeFlags(Root()) })
			require.NoError(t, updateJobCmd.ParseFlags(flags))
			_, _, err := parseUpdateJobCommandRequest(updateJobCmd)
			require.Error(t, err)
		})
	}
}

// TestUpdateJobAssignmentIndependentJobs keeps separate jobs in one process,
// including a completed-before-submit target and a native conflict after acceptance.
func TestUpdateJobAssignmentIndependentJobs(t *testing.T) {
	a, b, c := completionAcceptanceKey(0), completionAcceptanceKey(1), completionAcceptanceKey(2)
	f, server := newCompletionAcceptanceServer(t, "8.10", nil, map[string]string{b: "409 completed", c: "completed"})
	f.expectedBID = "ORDER-123"
	stdout, stderr, err := runCompletionAcceptance(t, server, "8.10", []string{"-", "--set-bid", f.expectedBID, "--auto-confirm", "--json", "--no-wait", "--workers", "1"}, a+"\n"+b+"\n"+a+"\n"+c+"\n", false)
	require.Empty(t, stderr)
	payload := requireCompletionAcceptanceFailure(t, stdout, err)
	outcomes := requireCompletionAcceptancePartition(t, payload, []string{a, b, c})
	for i, want := range []string{"accepted", "rejected", "not_submitted"} {
		out := requireJSONObject(t, outcomes[i])
		require.Equal(t, want, requireJSONObject(t, out["assignment"])["status"])
		require.Equal(t, "2251799813711000", requireJSONObject(t, out["candidate"])["processInstanceKey"])
	}
	require.Equal(t, map[string]int{a: 1, b: 1}, f.writes)
	require.Zero(t, f.discovery)
}

// TestUpdateJobAssignmentUnsupported rejects even empty or dry-run old-version
// requests before job discovery and mutation, preserving the unsupported class.
func TestUpdateJobAssignmentUnsupported(t *testing.T) {
	for _, version := range []string{"8.8", "8.9"} {
		for _, selection := range [][]string{{"--key", scalarJobKey}, {"--key", scalarJobKey + "," + scalarJobKey}, {"--retries", "0"}} {
			t.Run(version+strings.Join(selection, " "), func(t *testing.T) {
				url, calls := scalarJobOutcomeFixture(t, version, "CREATED", "completion", 204, false)
				args := []string{"--config", writeTestConfigForVersion(t, url, version), "update", "job", "--complete", "--set-bid", "ORDER-123", "--json", "--dry-run"}
				args = append(args, selection...)
				stdout, _, err := runBulkCompletionTerminal(t, args, "configured", true, false, nil)
				require.Error(t, err)
				require.Equal(t, "unsupported", decodeBulkCompletionEnvelope(t, stdout)["class"])
				require.Empty(t, calls.Snapshot())
			})
		}
	}
}

// TestUpdateJobAssignmentFrozenApproval ensures flag storage edits during scalar
// approval cannot replace the key, exact BID or nested completion variables.
func TestUpdateJobAssignmentFrozenApproval(t *testing.T) {
	previous := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = previous })
	prompted := false
	confirmCmdOrAbortFn = func(_ io.Writer, _ bool, _ string) error {
		prompted = true
		flagUpdateJobSetBID = "changed"
		flagUpdateJobVariables = `{"nested":{"approved":false}}`
		flagUpdateJobKey = "different"
		return nil
	}
	var requests []string
	var bodies []map[string]any
	response := jobSearchResponse(scalarJobKey, 1)
	server := newJobCompleteServer(t, "8.10", &requests, &bodies, []string{response, response, response}, 204)
	defer server.Close()
	stdout, stderr := executeRootForJobTestWithSeparateOutputs(t, "--config", writeTestConfigForVersion(t, server.URL, "8.10"), "update", "job", "--key", scalarJobKey, "--complete", "--set-bid", " Order,007 ", "--vars", `{"nested":{"approved":true}}`)
	require.True(t, prompted)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "assignment accepted")
	require.Equal(t, []map[string]any{{"businessId": " Order,007 ", "variables": map[string]any{"nested": map[string]any{"approved": true}}}}, bodies)
}

// TestUpdateJobAssignmentScalarTerminal verifies plain scalar confirmation with
// terminal stdin and redirected results, including decline and canonical EOF.
func TestUpdateJobAssignmentScalarTerminal(t *testing.T) {
	const prompt = "You are about to update job 2251799813711967.\nDo you want to proceed? [y/N]: "
	for _, destination := range []string{"configured", "inherited"} {
		for _, decision := range []string{"yes", "no", "EOF"} {
			t.Run(destination+"/"+decision, func(t *testing.T) {
				url, calls := scalarJobOutcomeFixture(t, "8.10", "CREATED", "completion", 204, false)
				args := []string{"--config", writeTestConfigForVersion(t, url, "8.10"), "update", "job", "--key", scalarJobKey, "--complete", "--set-bid", "ORDER-123"}
				stdout, stderr, err := runBulkCompletionTerminal(t, args, destination, true, false, []testx.CmdTerminalExchange{{Prompt: prompt, Response: decision, EndOfInput: decision == "EOF"}})
				require.Empty(t, stdout)
				require.Equal(t, 1, strings.Count(stderr, prompt))
				before, after, found := strings.Cut(stderr, prompt)
				require.True(t, found)
				records := bulkCompletionDiagnosticRecords(t, before+after, "plain-time")
				require.NotEmpty(t, records)
				submissions := 0
				for _, call := range calls.Snapshot() {
					if strings.HasSuffix(call, "/completion") {
						submissions++
					}
				}
				if decision == "yes" {
					require.NoError(t, err)
					require.Equal(t, 1, submissions)
				} else {
					require.Error(t, err)
					require.Zero(t, submissions)
				}
			})
		}
	}
}

// TestUpdateJobAssignmentInputGuards exercises full-input and mode guardrails
// with a fake that fails any job lookup or mutation before valid preparation.
func TestUpdateJobAssignmentInputGuards(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		flags       []string
	}{
		{name: "invalid trailing stdin", input: scalarJobKey + "\nbad\n", flags: []string{"-"}},
		{name: "keyed retries", flags: []string{"--key", scalarJobKey, "--retries", "0"}},
		{name: "bulk keys", flags: []string{"--key", scalarJobKey + "," + scalarJobKey, "--keys-only"}},
		{name: "JSON verbose", flags: []string{"--key", scalarJobKey, "--json", "--verbose"}},
		{name: "blank assignment", flags: []string{"--key", scalarJobKey, "--set-bid", " "}},
		{name: "assignment without selection"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveVersionCheckTopology(w, r, "8.10") {
					return
				}
				requests.Add(1)
				t.Errorf("unexpected %s", r.URL.Path)
				w.WriteHeader(400)
			}))
			defer server.Close()
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.10"), "update", "job", "--complete", "--set-bid", "Order", "--auto-confirm"}
			args = append(args, tc.flags...)
			_, _, err := assignmentCommand(t, args, tc.input, "inherited", false, false, nil)
			require.Error(t, err)
			require.Zero(t, requests.Load())
		})
	}
}
