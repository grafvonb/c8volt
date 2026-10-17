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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

const suspendedInstanceFixture = `{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionVersion":3,"processInstanceKey":"123","state":"SUSPENDED","tenantId":"tenant"}`

// TestSuspendedCommandHelper isolates global Cobra flags and preserves real exit codes and streams.
func TestSuspendedCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_SUSPENDED_ARGS")), &args))
	if os.Getenv("C8VOLT_SUSPENDED_TERMINAL") == "1" {
		require.True(t, term.IsTerminal(int(os.Stdin.Fd())))
	}
	root := Root()
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.SetArgs(args)
	require.NoError(t, root.Execute())
	os.Exit(0)
}

// suspendedCommandEnv serializes arguments without shell parsing for isolated command runs.
func suspendedCommandEnv(t *testing.T, args []string) map[string]string {
	t.Helper()
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	return map[string]string{"C8VOLT_SUSPENDED_ARGS": string(encoded)}
}

// decodeSuspendedEnvelope requires one complete machine result with no extra output.
func decodeSuspendedEnvelope(t *testing.T, stdout string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(stdout))
	var result map[string]any
	require.NoError(t, decoder.Decode(&result))
	require.ErrorIs(t, decoder.Decode(&struct{}{}), io.EOF)
	return result
}

// TestSuspendedGetOutputTerminal proves empty/nonempty read results never prompt even
// when terminal stdin makes interactive paging eligible; all requests are read searches.
func TestSuspendedGetOutputTerminal(t *testing.T) {
	modes := []struct {
		name              string
		flags             []string
		json, keys, quiet bool
	}{
		{name: "human"}, {name: "json", flags: []string{"--json"}, json: true},
		{name: "keys", flags: []string{"--keys-only"}, keys: true},
		{name: "quiet", flags: []string{"--quiet"}, quiet: true},
		{name: "quiet-json", flags: []string{"--quiet", "--json"}, json: true, quiet: true},
		{name: "quiet-keys", flags: []string{"--quiet", "--keys-only"}, keys: true, quiet: true},
	}
	for _, empty := range []bool{true, false} {
		for _, mode := range modes {
			for _, policy := range []string{"", "--auto-confirm", "--automation"} {
				t.Run(fmt.Sprintf("empty=%t/%s/%s", empty, mode.name, policy), func(t *testing.T) {
					var requests atomic.Int32
					srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						requests.Add(1)
						require.Equal(t, http.MethodPost, r.Method)
						require.Equal(t, "/v2/process-instances/search", r.URL.Path)
						var body map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						require.Equal(t, "SUSPENDED", body["filter"].(map[string]any)["state"])
						w.Header().Set("Content-Type", "application/json")
						if empty {
							fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
						} else {
							fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, suspendedInstanceFixture)
						}
					}))
					t.Cleanup(srv.Close)
					args := []string{"--config", writeTestConfigForVersion(t, srv.URL, "8.10"), "--tenant", "tenant", "get", "pi", "--state", "sUsPeNdEd"}
					args = append(args, mode.flags...)
					if policy != "" {
						args = append(args, policy)
					}
					env := suspendedCommandEnv(t, args)
					env["C8VOLT_SUSPENDED_TERMINAL"] = "1"
					result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{ScopeTestName: "TestSuspendedCommandHelper", Env: env, Timeout: 5 * time.Second})
					require.True(t, result.Supported, result.UnsupportedReason)
					require.NoError(t, result.Err, result.Stderr)
					require.EqualValues(t, 1, requests.Load())
					require.Empty(t, result.Stderr)
					switch {
					case mode.json:
						envelope := decodeSuspendedEnvelope(t, result.Stdout)
						require.Equal(t, "succeeded", envelope["outcome"])
						require.Equal(t, "get process-instance", envelope["command"])
						payload := envelope["payload"].(map[string]any)
						if empty {
							require.NotContains(t, payload, "total")
							require.NotContains(t, payload, "items")
						} else {
							require.EqualValues(t, 1, payload["total"])
							items := payload["items"].([]any)
							require.Len(t, items, 1)
							require.Equal(t, "SUSPENDED", items[0].(map[string]any)["state"])
						}
					case mode.keys:
						want := "123\n"
						if empty {
							want = ""
						}
						require.Equal(t, want, result.Stdout)
					case mode.quiet && empty:
						require.Empty(t, result.Stdout)
					case empty:
						require.Equal(t, "found: 0\n", result.Stdout)
					default:
						require.Equal(t, "123 tenant demo v3 SUSPENDED s: p:<root> inc!\nfound: 1\n", result.Stdout)
					}
				})
			}
		}
	}
}

// TestSuspendedUnsupportedCommands proves old-version searches and expectations
// retain unsupported classification and never reach discovery, lookup, or mutation.
func TestSuspendedUnsupportedCommands(t *testing.T) {
	for _, version := range []string{"8.8", "8.9"} {
		for _, selection := range [][]string{
			{"get", "pi", "--state", "suspended"},
			{"get", "pi", "--state", "SUSPENDED", "--total"},
			{"get", "pi", "--state", "sUsPeNdEd", "--direct-incidents-only"},
			{"expect", "pi", "--key", "123", "--state", "suspended"},
			{"expect", "pi", "--key", "123", "--state", "active,SUSPENDED", "--incident", "false"},
		} {
			for _, machine := range []bool{false, true} {
				// --total deliberately excludes --json in the existing flag contract.
				if machine && selection[len(selection)-1] == "--total" {
					continue
				}
				t.Run(version+"/"+strings.Join(selection, " ")+fmt.Sprint(machine), func(t *testing.T) {
					var requests atomic.Int32
					srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						requests.Add(1)
						http.Error(w, "unexpected request", 500)
					}))
					t.Cleanup(srv.Close)
					args := append([]string{"--config", writeTestConfigForVersion(t, srv.URL, version)}, selection...)
					if machine {
						args = append(args, "--json")
					}
					stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestSuspendedCommandHelper", suspendedCommandEnv(t, args))
					var exitErr *exec.ExitError
					require.ErrorAs(t, err, &exitErr)
					require.Equal(t, exitcode.Error, exitErr.ExitCode())
					require.Zero(t, requests.Load())
					if machine {
						result := decodeSuspendedEnvelope(t, stdout)
						require.Equal(t, "failed", result["outcome"])
						require.Equal(t, "unsupported", result["class"])
						require.Contains(t, result["detail"].(map[string]any)["message"], "SUSPENDED")
						require.Empty(t, stderr)
					} else {
						require.Empty(t, stdout)
						require.Contains(t, stderr, "unsupported capability")
					}
				})
			}
		}
	}
}

// TestSuspendedGetExplicitKeyAndExpectations covers case normalization, state-only
// and combined incident success through real adapters while preserving observed state.
func TestSuspendedGetExplicitKeyAndExpectations(t *testing.T) {
	for _, selection := range [][]string{
		{"get", "pi", "--key", "123"},
		{"expect", "pi", "--key", "123", "--state", "SUSPENDED"},
		{"expect", "pi", "--key", "123", "--state", "suspended"},
		{"expect", "pi", "--key", "123", "--state", " sUsPeNdEd ", "--incident", "true"},
	} {
		t.Run(strings.Join(selection, " "), func(t *testing.T) {
			var requests atomic.Int32
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, "/v2/process-instances/123", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, suspendedInstanceFixture)
			}))
			t.Cleanup(srv.Close)
			args := append([]string{"--config", writeTestConfigForVersion(t, srv.URL, "8.10"), "--json"}, selection...)
			stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestSuspendedCommandHelper", suspendedCommandEnv(t, args))
			require.NoError(t, err, stderr)
			result := decodeSuspendedEnvelope(t, stdout)
			require.Equal(t, "succeeded", result["outcome"])
			require.Contains(t, stdout, `"state": "SUSPENDED"`)
			if selection[len(selection)-1] == "true" {
				require.Contains(t, stdout, `"ok": true`)
				require.Contains(t, stdout, `"incident": true`)
			}
			require.Empty(t, stderr)
			require.EqualValues(t, 1, requests.Load())
		})
	}
}

// TestSuspendedStateValidationFeedback keeps invalid-input classification and
// version guidance visible without issuing a request for malformed state syntax.
func TestSuspendedStateValidationFeedback(t *testing.T) {
	for _, verb := range []string{"get", "expect"} {
		t.Run(verb, func(t *testing.T) {
			var requests atomic.Int32
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				http.Error(w, "unexpected request", 500)
			}))
			t.Cleanup(srv.Close)
			args := []string{"--config", writeTestConfigForVersion(t, srv.URL, "8.10"), "--json", verb, "pi", "--state", "not-a-state"}
			if verb == "expect" {
				args = append(args, "--key", "123")
			}
			stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestSuspendedCommandHelper", suspendedCommandEnv(t, args))
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr)
			require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
			envelope := decodeSuspendedEnvelope(t, stdout)
			require.Equal(t, "invalid", envelope["outcome"])
			require.Contains(t, envelope["detail"].(map[string]any)["message"], "suspended requires Camunda 8.10")
			require.Zero(t, requests.Load())
			require.Empty(t, stderr)
		})
	}
}

// TestSuspendedSearchStateSelection verifies canonical suspension and exact ACTIVE
// filters, while ALL remains an omitted predicate and includes suspended results.
func TestSuspendedSearchStateSelection(t *testing.T) {
	for _, state := range []string{"SUSPENDED", " suspended ", "sUsPeNdEd", "active", "all"} {
		t.Run(state, func(t *testing.T) {
			var requests atomic.Int32
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/v2/process-instances/search", r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				filter, _ := body["filter"].(map[string]any)
				items := suspendedInstanceFixture
				count := 1
				switch state {
				case "all":
					require.NotContains(t, filter, "state")
					items += "," + strings.ReplaceAll(strings.ReplaceAll(suspendedInstanceFixture, "SUSPENDED", "ACTIVE"), `"123"`, `"124"`)
					count = 2
				case "active":
					require.Equal(t, "ACTIVE", filter["state"])
					items = strings.ReplaceAll(suspendedInstanceFixture, "SUSPENDED", "ACTIVE")
				default:
					require.Equal(t, "SUSPENDED", filter["state"])
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":%d,"hasMoreTotalItems":false}}`, items, count)
			}))
			t.Cleanup(srv.Close)
			args := []string{"--config", writeTestConfigForVersion(t, srv.URL, "8.10"), "--json", "get", "pi", "--state", state}
			stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestSuspendedCommandHelper", suspendedCommandEnv(t, args))
			require.NoError(t, err, stderr)
			payload := decodeSuspendedEnvelope(t, stdout)["payload"].(map[string]any)
			if state == "all" {
				require.Len(t, payload["items"], 2)
			} else {
				require.Len(t, payload["items"], 1)
			}
			if state == "active" {
				require.NotContains(t, stdout, "SUSPENDED")
			} else {
				require.Contains(t, stdout, "SUSPENDED")
			}
			require.Empty(t, stderr)
			require.EqualValues(t, 1, requests.Load())
		})
	}
}
