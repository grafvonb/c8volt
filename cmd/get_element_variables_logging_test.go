// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestElementScopeVariablesGetDiagnostics proves command-owned/inherited logger routing, bounded errors, independent DEBUG and single failure envelopes.
func TestElementScopeVariablesGetDiagnostics(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, tc := range []struct {
				name                            string
				flags                           []string
				debug, configDebug, json, quiet bool
			}{
				{name: "normal"}, {name: "verbose", flags: []string{"--verbose"}},
				{name: "quiet", flags: []string{"--quiet"}, quiet: true},
				{name: "debug flag", flags: []string{"--debug"}, debug: true},
				{name: "debug config", debug: true, configDebug: true},
				{name: "quiet beats debug", flags: []string{"--quiet", "--debug"}, configDebug: true, quiet: true},
				{name: "JSON", flags: []string{"--json"}, json: true},
				{name: "quiet JSON", flags: []string{"--quiet", "--json"}, json: true, quiet: true},
			} {
				t.Run(format+"/"+destination+"/"+tc.name, func(t *testing.T) {
					// Reads stop at the first failed scope. Earlier complete pages may already be rendered;
					// a repeated transport cause must never be dumped or converted into a successful empty scope.
					marker := fmt.Sprintf("POST https://scope-private.invalid/path?token=scope-token-secret body-only-secret resource=%s", scopeElementB)
					failure := fmt.Sprintf(`{"detail":%q}`, marker+"; "+marker)
					f := newElementScopeFixture(t, "8.10", []string{
						scopePage([]string{scopeElementJSON(scopeElementA)}, 2, true), scopePage([]string{scopeElementJSON(scopeElementB)}, 2, false),
					}, []string{scopePage(nil, 0, false), failure}, 400)
					level := "info"
					if tc.configDebug {
						level = "debug"
					}
					args := append([]string{"--config", completionLogConfig(t, f.server.URL, format, level), "--no-indicator", "--automation", "get", "element", "--batch-size", "1", "--with-vars"}, tc.flags...)
					stdout, stderr, err := runElementScopeTerminal(t, args, destination, nil)
					var exit *exec.ExitError
					require.ErrorAs(t, err, &exit)
					require.Equal(t, 2, exit.ExitCode())
					require.Len(t, f.requests.Snapshot(), 4)
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					var errors, debugs []string
					for _, r := range records {
						switch r[0] {
						case "ERROR":
							errors = append(errors, r[1])
						case "DEBUG":
							debugs = append(debugs, r[1])
						case "INFO":
							require.Equal(t, "verbose", tc.name)
						default:
							t.Errorf("unexpected log: %v", r)
						}
					}
					if tc.json {
						require.Empty(t, errors)
						env := requireSingleJSONObjectDocument(t, stdout)
						require.Equal(t, "invalid_input", env["class"])
						require.Contains(t, requireJSONObject(t, env["detail"])["message"], "body-only-secret")
						require.NotEqual(t, "succeeded", env["outcome"])
					} else {
						require.Equal(t, []string{"get elements: invalid input"}, errors)
						if tc.quiet {
							require.Empty(t, stdout)
						} else {
							require.Equal(t, scopeElementA+" tenant-a SERVICE_TASK task ACTIVE pi:"+scopeOwner+"\n", stdout)
						}
						for _, bad := range []string{"POST", f.server.URL, "/v2/variables/search", "scope-private.invalid", "body-only-secret", scopeElementB, "ERROR", "DEBUG", "found:"} {
							require.NotContains(t, stdout, bad)
						}
					}
					for _, bad := range []string{"scope-private.invalid", "scope-token-secret", "scope-cookie-secret", "body-only-secret"} {
						require.NotContains(t, stderr, bad)
					}
					if tc.debug {
						exchanges := 0
						for _, message := range debugs {
							if strings.Contains(message, "api #") {
								exchanges++
								if exchanges == 1 {
									require.Contains(t, message, "GET /v2/topology")
								} else {
									require.Contains(t, message, "POST /v2/")
								}
							}
						}
						require.Equal(t, 5, exchanges)
					} else {
						require.Empty(t, debugs)
						for _, bad := range []string{"POST", f.server.URL, "/v2/variables/search", scopeElementB} {
							require.NotContains(t, stderr, bad)
						}
					}
				})
			}
		}
	}
}

// TestElementScopeVariablesGetProcess is the isolated subprocess entry for invalid inputs and read failures.
func TestElementScopeVariablesGetProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_ELEMENT_ARGS")), &args))
	os.Args = append([]string{"c8volt"}, args...)
	Execute()
	os.Exit(0)
}

// runElementScopeProcess keeps fatal command paths from exiting the parent test process.
func runElementScopeProcess(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	return testx.RunCmdSubprocessSeparate(t, "TestElementScopeVariablesGetProcess", map[string]string{"C8VOLT_ELEMENT_ARGS": string(encoded)})
}

// TestElementScopeVariablesGetValidation preserves listener/output conflicts and 8.7 bootstrap classification before any requests.
func TestElementScopeVariablesGetValidation(t *testing.T) {
	for _, tc := range []struct {
		name, version string
		flags         []string
		exit          int
	}{
		{"bootstrap", "8.7", nil, 1},
		{"listener keys", "8.8", []string{"--with-listeners", "--keys-only"}, 2},
		{"listener total", "8.8", []string{"--with-listeners", "--total"}, 2},
		{"total JSON", "8.8", []string{"--total", "--json"}, 2},
		{"total keys", "8.8", []string{"--total", "--keys-only"}, 2},
		{"invalid key", "8.8", []string{"--key", "bad-key"}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newElementScopeFixture(t, tc.version, nil, nil, 0)
			args := append([]string{"--config", testx.WriteTestConfigForVersion(t, f.server.URL, tc.version), "get", "element", "--with-vars"}, tc.flags...)
			stdout, stderr, err := runElementScopeProcess(t, args...)
			var exit *exec.ExitError
			require.ErrorAs(t, err, &exit)
			require.Equal(t, tc.exit, exit.ExitCode(), stderr+stdout)
			require.Empty(t, f.requests.Snapshot())
			if strings.Contains(strings.Join(tc.flags, " "), "--json") {
				env := requireSingleJSONObjectDocument(t, stdout)
				require.Equal(t, "invalid_input", env["class"])
			} else {
				require.Empty(t, stdout)
				require.NotEmpty(t, stderr)
			}
		})
	}
}

// TestElementScopeVariablesGetLaterVariableFailure never publishes an incomplete scope as an empty or successful item.
func TestElementScopeVariablesGetLaterVariableFailure(t *testing.T) {
	for _, keyed := range []bool{false, true} {
		t.Run(fmt.Sprintf("keyed=%t", keyed), func(t *testing.T) {
			elements := []string{scopePage([]string{scopeElementJSON(scopeElementA)}, 1, false)}
			if keyed {
				elements = []string{scopeElementJSON(scopeElementA)}
			}
			f := newElementScopeFixture(t, "8.8", elements, []string{
				scopePage([]string{scopeVariableJSON("first", "1", scopeElementA, false)}, 2, true), `{"detail":"later-page-failure"}`,
			}, 400)
			args := []string{"--config", testx.WriteTestConfig(t, f.server.URL), "--json", "get", "element", "--with-vars"}
			if keyed {
				args = append(args, "--key", scopeElementA)
			}
			stdout, stderr, err := runElementScopeProcess(t, args...)
			var exit *exec.ExitError
			require.ErrorAs(t, err, &exit)
			require.Equal(t, 2, exit.ExitCode())
			require.Empty(t, stderr)
			env := requireSingleJSONObjectDocument(t, stdout)
			require.Equal(t, "invalid_input", env["class"])
			require.NotEqual(t, "succeeded", env["outcome"])
			require.NotContains(t, env, "payload")
			require.Len(t, f.requests.Snapshot(), 3)
		})
	}
}
