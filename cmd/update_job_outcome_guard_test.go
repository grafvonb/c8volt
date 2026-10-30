// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

var scalarJobOutcomes = []struct {
	name, path string
	flags      []string
}{
	{"completion", "completion", []string{"--complete"}},
	{"failure", "failure", []string{"--fail", "--retries", "0"}},
	{"BPMN error", "error", []string{"--throw-bpmn-error", "EXPECTED_ERROR"}},
}

const scalarJobKey = "2251799813711967"

// scalarJobOutcomeFixture records every request, making extra lookups and polling
// failures visible while exercising the real facade and versioned adapter.
func scalarJobOutcomeFixture(t *testing.T, version string, state, path string, status int, planning bool) (string, *testx.SafeSlice[string]) {
	t.Helper()
	calls := &testx.SafeSlice[string]{}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, version) {
			return
		}

		calls.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		require.Equal(t, http.MethodPost, r.Method)
		if r.URL.Path == "/v2/jobs/search" && !planning {
			io.WriteString(w, jobSearchResponseWithState(scalarJobKey, 1, state))
			return
		}
		if planning {
			require.Equal(t, "/v2/jobs/search", r.URL.Path)
		} else {
			require.Equal(t, "/v2/jobs/"+scalarJobKey+"/"+path, r.URL.Path)
		}
		w.WriteHeader(status)
		if status != 204 {
			fmt.Fprintf(w, `{"title":"Rejected","detail":"POST https://private.invalid/secret body-only-secret job %s leaseToken"}`, scalarJobKey)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL, calls
}

func assertScalarJobExit(t *testing.T, err error, code int) {
	t.Helper()
	if code == 0 {
		require.NoError(t, err)
		return
	}
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, code, exitErr.ExitCode())
}

// TestScalarJobCompletedGuard proves no approval or mutation for an observed
// completed job, including real terminal stdin, dry-run and machine output.
func TestScalarJobCompletedGuard(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, outcome := range scalarJobOutcomes {
			for i, mode := range []struct {
				name  string
				flags []string
				json  bool
				exit  int
			}{
				{"interactive", nil, false, 6},
				{"dry run", []string{"--dry-run"}, false, 6},
				{"auto confirm", []string{"--auto-confirm"}, false, 6},
				{"automation", []string{"--automation"}, false, 6},
				{"no wait", []string{"--no-wait"}, false, 6},
				{"quiet", []string{"--quiet"}, false, 6},
				{"JSON", []string{"--json", "--auto-confirm"}, true, 6},
				{"quiet JSON preview", []string{"--quiet", "--json", "--dry-run"}, true, 6},
				{"suppressed exit", []string{"--no-err-codes"}, false, 0},
			} {
				t.Run(version+"/"+outcome.name+"/"+mode.name, func(t *testing.T) {
					url, calls := scalarJobOutcomeFixture(t, version, "COMPLETED", outcome.path, 204, false)
					cfg := writeTestConfigForVersion(t, url, version)
					args := append([]string{"--config", cfg, "update", "job", "--key", scalarJobKey}, outcome.flags...)
					args = append(args, mode.flags...)
					destination := "inherited"
					if i%2 == 1 {
						destination = "configured"
					}
					stdout, stderr, err := runBulkCompletionTerminal(t, args, destination, true, false, nil)
					assertScalarJobExit(t, err, mode.exit)
					require.Equal(t, []string{"POST /v2/jobs/search"}, calls.Snapshot())
					expected := "update job: job is already completed; " + outcome.name + " request not sent"
					if mode.json {
						envelope := decodeBulkCompletionEnvelope(t, stdout)
						require.Equal(t, "conflict", envelope["class"])
						require.Equal(t, "conflict: "+expected, requireJSONObject(t, envelope["detail"])["message"])
						require.Empty(t, stderr)
					} else {
						require.Empty(t, stdout)
						require.Equal(t, [][2]string{{"ERROR", expected}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
					}
				})
			}
		}
	}
}

// TestScalarJobCompletedGuardVariables preserves the existing command contract with advisory-aware fixtures.
func TestScalarJobCompletedGuardVariables(t *testing.T) {
	for _, outcome := range scalarJobOutcomes {
		if outcome.name == "failure" {
			continue
		}
		for _, payload := range []string{"{}", `{"approved":true}`} {
			t.Run(outcome.name+"/"+payload, func(t *testing.T) {
				url, calls := scalarJobOutcomeFixture(t, "8.10", "COMPLETED", outcome.path, 204, false)
				args := append([]string{"--config", writeTestConfigForVersion(t, url, "8.10"), "update", "job", "--key", scalarJobKey}, outcome.flags...)
				args = append(args, "--vars", payload)
				stdout, stderr, err := runBulkCompletionTerminal(t, args, "configured", false, false, nil)
				assertScalarJobExit(t, err, 6)
				require.Empty(t, stdout)
				require.Equal(t, [][2]string{{"ERROR", "update job: job is already completed; " + outcome.name + " request not sent; supplied variables were not applied"}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
				require.Equal(t, []string{"POST /v2/jobs/search"}, calls.Snapshot())
			})
		}
	}
}

// TestScalarJobOutcomeErrors preserves backend evidence and exit classes while
// preventing raw transport details or direct prints in human diagnostics.
func TestScalarJobOutcomeErrors(t *testing.T) {
	for _, outcome := range scalarJobOutcomes {
		for _, format := range []string{"plain-time", "json"} {
			for i, tc := range []struct {
				name                                          string
				status, exit                                  int
				class, reason                                 string
				flags                                         []string
				planning, json, debug, configuredDebug, quiet bool
			}{
				{name: "stale lookup", status: 404, exit: 3, class: "not_found", reason: "job not found"},
				{name: "verbose", status: 404, exit: 3, class: "not_found", reason: "job not found", flags: []string{"--verbose"}},
				{name: "quiet", status: 404, exit: 3, class: "not_found", reason: "job not found", flags: []string{"--quiet"}, quiet: true},
				{name: "invalid", status: 400, exit: 2, class: "invalid_input", reason: "invalid input"},
				{name: "conflict", status: 409, exit: 6, class: "conflict", reason: "conflict"},
				{name: "internal", status: 500, exit: 1, class: "internal", reason: "internal"},
				{name: "timeout", status: 504, exit: 4, class: "timeout", reason: "timeout"},
				{name: "lookup failure", status: 404, exit: 3, class: "not_found", reason: "job not found", planning: true},
				{name: "JSON", status: 404, exit: 3, class: "not_found", flags: []string{"--json"}, json: true},
				{name: "JSON conflict", status: 409, exit: 6, class: "conflict", flags: []string{"--json"}, json: true},
				{name: "JSON lookup", status: 400, exit: 2, class: "invalid_input", flags: []string{"--json"}, json: true, planning: true},
				{name: "quiet JSON", status: 404, exit: 3, class: "not_found", flags: []string{"--json", "--quiet"}, json: true, quiet: true},
				{name: "debug flag", status: 404, exit: 3, class: "not_found", reason: "job not found", flags: []string{"--debug"}, debug: true},
				{name: "debug config", status: 404, exit: 3, class: "not_found", reason: "job not found", configuredDebug: true, debug: true},
				{name: "quiet debug", status: 404, exit: 3, class: "not_found", reason: "job not found", flags: []string{"--quiet", "--debug"}, quiet: true, configuredDebug: true},
				{name: "suppressed exit", status: 404, exit: 0, class: "not_found", reason: "job not found", flags: []string{"--no-err-codes"}},
			} {
				t.Run(outcome.name+"/"+format+"/"+tc.name, func(t *testing.T) {
					url, calls := scalarJobOutcomeFixture(t, "8.10", "CREATED", outcome.path, tc.status, tc.planning)
					cfg := writeTestConfigForVersion(t, url, "8.10")
					level := "info"
					if tc.configuredDebug {
						level = "debug"
					}
					f, err := os.OpenFile(cfg, os.O_APPEND|os.O_WRONLY, 0600)
					require.NoError(t, err)
					_, err = fmt.Fprintf(f, "log:\n  level: %s\n  format: %s\n", level, format)
					require.NoError(t, err)
					require.NoError(t, f.Close())
					args := append([]string{"--config", cfg, "update", "job", "--key", scalarJobKey, "--auto-confirm"}, outcome.flags...)
					args = append(args, tc.flags...)
					destination := "inherited"
					if i%2 == 1 {
						destination = "configured"
					}
					stdout, stderr, err := runBulkCompletionTerminal(t, args, destination, false, false, nil)
					assertScalarJobExit(t, err, tc.exit)
					expectedCalls := []string{"POST /v2/jobs/search"}
					if !tc.planning {
						expectedCalls = append(expectedCalls, "POST /v2/jobs/"+scalarJobKey+"/"+outcome.path)
					}
					require.Equal(t, expectedCalls, calls.Snapshot())
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					errors, infos, debugs := []string{}, []string{}, []string{}
					for _, r := range records {
						switch r[0] {
						case "ERROR":
							errors = append(errors, r[1])
						case "INFO":
							infos = append(infos, r[1])
						case "DEBUG":
							debugs = append(debugs, r[1])
						default:
							t.Fatalf("unexpected record: %v", r)
						}
					}
					if tc.json {
						envelope := decodeBulkCompletionEnvelope(t, stdout)
						require.Equal(t, tc.class, envelope["class"])
						detail := requireJSONObject(t, envelope["detail"])["message"]
						require.Contains(t, detail, "body-only-secret")
						require.Contains(t, detail, "POST")
						if tc.planning {
							require.Contains(t, detail, "plan job update:")
						} else {
							require.Contains(t, detail, "update job:")
						}
						require.NotContains(t, envelope, "payload")
						require.Empty(t, errors)
						require.Empty(t, infos)
					} else {
						require.Empty(t, stdout)
						message := "update job: " + outcome.name + " request failed: " + tc.reason
						if tc.planning {
							message = "update job: could not read job: " + tc.reason + "; " + outcome.name + " request not sent"
						}
						require.Equal(t, []string{message}, errors)
						if tc.quiet || tc.planning {
							require.Empty(t, infos)
						}
					}
					for _, bad := range []string{"body-only-secret", "private.invalid", "leaseToken"} {
						require.NotContains(t, stderr, bad)
					}
					require.NotContains(t, stderr, "already completed")
					if tc.debug {
						require.NotEmpty(t, debugs)
						require.Contains(t, strings.Join(debugs, "\n"), "POST /v2/jobs/")
						require.Contains(t, strings.Join(debugs, "\n"), "status=404")
					} else {
						require.Empty(t, debugs)
						for _, bad := range []string{url, "POST", "/v2/jobs/"} {
							require.NotContains(t, stderr, bad)
						}
					}
				})
			}
		}
	}
}

// TestScalarJobOtherStatesRemainBackendOwned prevents an eligibility allowlist
// and proves successful submission still uses exactly the original lookup.
func TestScalarJobOtherStatesRemainBackendOwned(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, outcome := range scalarJobOutcomes {
			for _, state := range []string{"FAILED", "CREATED"} {
				t.Run(version+"/"+outcome.name+"/"+state, func(t *testing.T) {
					url, calls := scalarJobOutcomeFixture(t, version, state, outcome.path, 204, false)
					args := append([]string{"--config", writeTestConfigForVersion(t, url, version), "update", "job", "--key", scalarJobKey, "--auto-confirm", "--json"}, outcome.flags...)
					stdout, stderr, err := runBulkCompletionTerminal(t, args, "configured", false, false, nil)
					require.NoError(t, err)
					require.Empty(t, stderr)
					envelope := decodeBulkCompletionEnvelope(t, stdout)
					require.NotContains(t, envelope, "class")
					payload := requireJSONObject(t, envelope["payload"])
					require.Equal(t, scalarJobKey, payload["key"])
					require.Equal(t, true, payload["mutationAccepted"])
					require.Equal(t, "submitted", payload["status"])
					require.Equal(t, []string{"POST /v2/jobs/search", "POST /v2/jobs/" + scalarJobKey + "/" + outcome.path}, calls.Snapshot())
				})
			}
		}
	}
}

// TestScalarJobOutcomePrompt preserves plain confirmation and proves approval
// reuses the plan instead of fetching state again.
func TestScalarJobOutcomePrompt(t *testing.T) {
	for _, outcome := range scalarJobOutcomes {
		for _, response := range []string{"y", "n"} {
			t.Run(outcome.name+"/"+response, func(t *testing.T) {
				url, calls := scalarJobOutcomeFixture(t, "8.10", "FAILED", outcome.path, 204, false)
				args := append([]string{"--config", writeTestConfigForVersion(t, url, "8.10"), "update", "job", "--key", scalarJobKey}, outcome.flags...)
				prompt := "You are about to update job " + scalarJobKey + ".\nDo you want to proceed? [y/N]: "
				destination := "configured"
				if response == "n" {
					destination = "inherited"
				}
				stdout, stderr, err := runBulkCompletionTerminal(t, args, destination, true, false, []testx.CmdTerminalExchange{{Prompt: prompt, Response: response}})
				require.Empty(t, stdout)
				require.Equal(t, 1, strings.Count(stderr, prompt))
				require.Less(t, strings.Index(stderr, "selection scope:"), strings.Index(stderr, "plan: update job"))
				require.Less(t, strings.Index(stderr, "plan: update job"), strings.Index(stderr, prompt))
				bulkCompletionDiagnosticRecords(t, strings.Replace(stderr, prompt, "", 1), "plain-time")
				expected := []string{"POST /v2/jobs/search"}
				if response == "y" {
					require.NoError(t, err)
					expected = append(expected, "POST /v2/jobs/"+scalarJobKey+"/"+outcome.path)
					require.Contains(t, stderr, "INFO updated job "+scalarJobKey+": submitted")
				} else {
					require.Error(t, err)
					require.NotContains(t, stderr, "INFO updated job")
				}
				require.Equal(t, expected, calls.Snapshot())
			})
		}
	}
}
