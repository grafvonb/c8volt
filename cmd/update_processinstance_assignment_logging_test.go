// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

const assignmentKey = "2251799813711967"
const assignmentOtherKey = "2251799813711968"
const assignmentReady = "assignment test input ready\n"

// assignmentCapture records the actual wire activity of isolated command runs.
type assignmentCapture struct {
	mu            sync.Mutex
	reads, writes int
	values        map[string]string
}

// assignmentServer supports exact-instance requests only, including response loss.
func assignmentServer(t *testing.T, scenario, version string) (*httptest.Server, *assignmentCapture) {
	t.Helper()
	c := &assignmentCapture{values: map[string]string{}}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, version) {
			return
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
		if r.Method == "GET" && (key == assignmentKey || key == assignmentOtherKey) {
			c.reads++
			if scenario != "planning" && !(scenario == "unconfirmed" && c.values[key] != "") {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"processInstanceKey":%q,"state":"ACTIVE","businessId":%q}`, key, c.values[key])
				return
			}
		} else if r.Method == "POST" && strings.HasSuffix(key, "/business-id-assignment") {
			key = strings.TrimSuffix(key, "/business-id-assignment")
			require.Contains(t, []string{assignmentKey, assignmentOtherKey}, key)
			c.writes++
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Len(t, body, 1)
			value, ok := body["businessId"].(string)
			require.True(t, ok)
			if scenario == "success" || scenario == "unconfirmed" || scenario == "partial" && key == assignmentKey {
				c.values[key] = value
				w.WriteHeader(204)
				return
			}
			if scenario == "unknown" {
				conn, _, err := w.(http.Hijacker).Hijack()
				require.NoError(t, err)
				require.NoError(t, conn.Close())
				return
			}
		} else {
			t.Errorf("unexpected endpoint %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		fmt.Fprintf(w, `{"title":"Bad Request","detail":"POST https://private.invalid/assignment body-only-secret leaseToken resource %s"}`, key)
	}))
	t.Cleanup(server.Close)
	return server, c
}

// assignmentConfig gives each command an owned endpoint and explicit log configuration.
func assignmentConfig(t *testing.T, server *httptest.Server, version, format, level string) string {
	t.Helper()
	path := writeTestConfigForVersion(t, server.URL, version)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	require.NoError(t, err)
	_, err = fmt.Fprintf(f, "log:\n  level: %s\n  format: %s\n", level, format)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return path
}

// assignmentCommand exercises production exits and independently captures configured streams.
func assignmentCommand(t *testing.T, args []string, input, destination string, terminal, ready bool, exchanges []testx.CmdTerminalExchange) (string, string, error) {
	t.Helper()
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	env := map[string]string{
		"ASSIGNMENT_ARGS":        string(encoded),
		"ASSIGNMENT_DESTINATION": destination,
		"ASSIGNMENT_CAPTURE":     filepath.Join(t.TempDir(), "stderr"),
	}
	if ready {
		env["ASSIGNMENT_READY"] = "1"
	}
	var stdout, stderr string
	if terminal {
		result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{ScopeTestName: "TestBusinessIDAssignmentHelper", Env: env, Exchanges: exchanges})
		if !result.Supported {
			t.Skip(result.UnsupportedReason)
		}
		stdout, stderr, err = result.Stdout, result.Stderr, result.Err
	} else {
		stdout, stderr, err = testx.RunCmdSubprocessInDirWithSeparateOutputs(t, "TestBusinessIDAssignmentHelper", "", env, input)
	}
	if destination != "" {
		capture, readErr := os.ReadFile(env["ASSIGNMENT_CAPTURE"])
		require.NoError(t, readErr)
		require.Equal(t, stderr, string(capture), "all diagnostic and prompt output must follow the selected writer")
	}
	return stdout, stderr, err
}

// TestBusinessIDAssignmentHelper isolates exits, terminal stdin and writer inheritance.
func TestBusinessIDAssignmentHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	// Explicit test config owns endpoints and credentials; ignore inherited application overrides.
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "C8VOLT_") && name != testx.CmdSubprocessNameEnv {
			os.Unsetenv(name)
		}
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("ASSIGNMENT_ARGS")), &args))
	root := Root()
	root.SetOut(os.Stdout)
	root.SetErr(nil)
	updateProcessInstanceCmd.SetErr(nil)
	if destination := os.Getenv("ASSIGNMENT_DESTINATION"); destination != "" {
		capture, err := os.Create(os.Getenv("ASSIGNMENT_CAPTURE"))
		require.NoError(t, err)
		defer capture.Close()
		writer := io.MultiWriter(os.Stderr, capture)
		if destination == "configured" {
			updateProcessInstanceCmd.SetErr(writer)
		} else {
			root.SetErr(writer)
		}
	}
	if os.Getenv("ASSIGNMENT_READY") == "1" {
		require.True(t, term.IsTerminal(int(os.Stdin.Fd())))
		require.False(t, term.IsTerminal(int(os.Stdout.Fd())))
		fmt.Fprint(updateProcessInstanceCmd.ErrOrStderr(), assignmentReady)
	}
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

// TestBusinessIDAssignmentLogging rejects direct prints, raw joined errors,
// duplicate envelopes and incorrect acceptance, using real service/facade execution.
func TestBusinessIDAssignmentLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, tc := range []struct {
				name, scenario, level      string
				flags                      []string
				machine, quiet, debug, dry bool
			}{
				{name: "success", scenario: "success"},
				{name: "verbose", scenario: "success", flags: []string{"--verbose"}},
				{name: "preview", scenario: "success", flags: []string{"--dry-run"}, dry: true},
				{name: "verbose preview", scenario: "success", flags: []string{"--dry-run", "--verbose"}, dry: true},
				{name: "verbose WARN", scenario: "success", level: "warn", flags: []string{"--verbose"}, quiet: true},
				{name: "verbose quiet", scenario: "success", flags: []string{"--verbose", "--quiet"}, quiet: true},
				{name: "WARN", scenario: "success", level: "warn", quiet: true},
				{name: "quiet", scenario: "success", flags: []string{"--quiet"}, quiet: true},
				{name: "quiet preview", scenario: "success", flags: []string{"--quiet", "--dry-run"}, quiet: true, dry: true},
				{name: "preview WARN", scenario: "success", level: "warn", flags: []string{"--dry-run"}, quiet: true, dry: true},
				{name: "failure", scenario: "reject"},
				{name: "failure verbose", scenario: "reject", flags: []string{"--verbose"}},
				{name: "failure quiet", scenario: "reject", flags: []string{"--quiet"}, quiet: true},
				{name: "debug flag", scenario: "reject", flags: []string{"--debug"}, debug: true},
				{name: "debug configured", scenario: "reject", level: "debug", debug: true},
				{name: "partial JSON", scenario: "partial", flags: []string{"--json"}, machine: true},
				{name: "partial quiet JSON", scenario: "partial", flags: []string{"--quiet", "--json"}, machine: true, quiet: true},
				{name: "planning", scenario: "planning"},
				{name: "planning JSON", scenario: "planning", flags: []string{"--json"}, machine: true},
				{name: "unknown", scenario: "unknown"},
				{name: "unknown JSON", scenario: "unknown", flags: []string{"--json"}, machine: true},
			} {
				t.Run(format+"/"+destination+"/"+tc.name, func(t *testing.T) {
					server, c := assignmentServer(t, tc.scenario, "8.10")
					level := tc.level
					if level == "" {
						level = "info"
					}
					cfg := assignmentConfig(t, server, "8.10", format, level)
					args := []string{"--config", cfg, "update", "pi", "--key", assignmentKey, "--key", assignmentOtherKey, "--set-bid", " Order,42 ", "--auto-confirm", "--no-wait"}
					args = append(args, tc.flags...)
					stdout, stderr, err := assignmentCommand(t, args, "", destination, false, false, nil)
					failed := tc.scenario != "success"
					if failed {
						var exitErr *exec.ExitError
						require.ErrorAs(t, err, &exitErr)
						expected := 2
						if tc.scenario == "unknown" {
							expected = 1
						}
						require.Equal(t, expected, exitErr.ExitCode())
					} else {
						require.NoError(t, err, stderr)
					}
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					var errors, infos, debugs []string
					for _, r := range records {
						switch r[0] {
						case "ERROR":
							errors = append(errors, r[1])
						case "INFO":
							infos = append(infos, r[1])
						case "DEBUG":
							debugs = append(debugs, r[1])
						default:
							t.Errorf("unexpected record %v", r)
						}
					}
					if tc.machine {
						envelope := decodeSingleProcessInstanceOutcomeEnvelope(t, stdout)
						require.Empty(t, errors)
						require.Empty(t, infos)
						require.NotEqual(t, "succeeded", envelope["outcome"])
						payload := requireJSONObject(t, envelope["payload"])
						if tc.scenario == "planning" {
							require.Equal(t, false, payload["ready"])
						} else {
							items := payload["items"].([]any)
							require.Len(t, items, 2)
							first := requireJSONObject(t, items[0])
							expected := "unknown"
							if tc.scenario == "partial" {
								expected = "accepted"
							}
							require.Equal(t, expected, first["status"])
							require.Equal(t, "skipped", first["confirmation"])
						}
					} else {
						require.Empty(t, stdout)
						if failed {
							reason := "invalid input"
							rejected, unknown, unconfirmed := 2, 0, 0
							if tc.scenario == "unknown" {
								rejected, unknown = 0, 2
								reason = "internal"
							}
							expected := fmt.Sprintf("business-ID assignment: 2 selected; %d rejected, %d unknown, %d accepted unconfirmed, 0 not submitted; %s", rejected, unknown, unconfirmed, reason)
							if tc.scenario == "planning" {
								expected = "business-ID assignment: preparation or approval failed; no changes submitted; invalid input"
							}
							require.Equal(t, []string{expected}, errors)
						} else {
							require.Empty(t, errors)
						}
						if tc.quiet || tc.scenario == "planning" {
							require.Empty(t, infos)
						} else {
							require.NotEmpty(t, infos)
							if tc.dry {
								expected := []string{`dry run: business-ID assignment: 2 process instance(s), requested business ID " Order,42 "; backend acceptance is not guaranteed`}
								if tc.name == "verbose preview" {
									expected = append(expected, assignmentKey+`: state "ACTIVE"; previously observed business ID ""`, assignmentOtherKey+`: state "ACTIVE"; previously observed business ID ""`)
								}
								require.Equal(t, expected, infos)
							} else if !failed {
								expected := []string{"business-ID assignment: 2 selected; 2 accepted, 0 confirmed, 0 rejected, 0 unknown, 0 not submitted"}
								if tc.name == "verbose" {
									expected = append(expected, assignmentKey+": accepted; confirmation skipped", assignmentOtherKey+": accepted; confirmation skipped")
								}
								require.Equal(t, expected, infos)
							}
						}
					}
					for _, bad := range []string{"private.invalid", "body-only-secret", "leaseToken"} {
						require.NotContains(t, stderr, bad)
						if !tc.machine {
							require.NotContains(t, stdout, bad)
						}
					}
					if tc.debug {
						require.NotEmpty(t, debugs)
						require.Contains(t, strings.Join(debugs, "\n"), "POST /v2/process-instances/")
						require.Contains(t, strings.Join(debugs, "\n"), "status=400")
					} else {
						require.Empty(t, debugs)
						require.NotContains(t, stderr, "POST")
						require.NotContains(t, stderr, server.URL)
						require.NotContains(t, stderr, "/v2/process-instances/")
					}
					c.mu.Lock()
					defer c.mu.Unlock()
					writes := 2
					if tc.dry || tc.scenario == "planning" {
						writes = 0
					}
					require.Equal(t, writes, c.writes)
					reads := 2
					if tc.scenario == "planning" {
						reads = 1
					}
					require.Equal(t, reads, c.reads)
				})
			}
		}
	}
}
