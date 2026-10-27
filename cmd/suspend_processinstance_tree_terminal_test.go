// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestSuspendProcessInstanceTreeTerminal proves approval uses expanded counts and
// a changing hierarchy cannot enlarge the frozen collection after a real PTY prompt.
func TestSuspendProcessInstanceTreeTerminal(t *testing.T) {
	const prompt = "You are about to suspend 2 process instance(s).\nDo you want to proceed? [y/N]: "
	const approval = "plan: suspend 2 process instance(s); 1 selected; 3 in scope; 1 already suspended; 0 excluded"
	const scope = "selected process instances: 1; process instances in scope: 3; active targets: 2; already suspended: 1; excluded: 0"
	for _, destination := range []string{"configured", "inherited"} {
		for _, tc := range []struct {
			name, response                    string
			flags                             []string
			approve, eof, noPrompt, dry, race bool
			code                              int
		}{
			{name: "approve", response: "yes", approve: true},
			{name: "decline", response: "no", code: 1},
			{name: "default", code: 1},
			{name: "EOF", eof: true, code: 1},
			{name: "keys", response: "y", approve: true, flags: []string{"--keys-only"}},
			{name: "quiet", response: "y", approve: true, flags: []string{"--quiet"}},
			{name: "quiet keys", response: "y", approve: true, flags: []string{"--quiet", "--keys-only"}},
			{name: "no-wait", response: "y", approve: true, flags: []string{"--no-wait"}},
			{name: "auto-confirm", approve: true, noPrompt: true, flags: []string{"--auto-confirm"}},
			{name: "automation", approve: true, noPrompt: true, flags: []string{"--automation"}},
			{name: "json", approve: true, noPrompt: true, flags: []string{"--auto-confirm", "--json"}},
			{name: "preview", dry: true, noPrompt: true, flags: []string{"--dry-run"}},
			{name: "preview keys", dry: true, noPrompt: true, flags: []string{"--dry-run", "--no-wait", "--keys-only"}},
			{name: "preview json", dry: true, noPrompt: true, flags: []string{"--dry-run", "--json"}},
			{name: "hierarchy and state race", response: "y", approve: true, race: true, code: 6, flags: []string{"--verbose"}},
		} {
			t.Run(destination+"/"+tc.name, func(t *testing.T) {
				f := newSuspensionTreeFixture(t)
				f.observed[suspendKeyA] = "SUSPENDED"
				args := append([]string{"--config", completionLogConfig(t, f.server.URL, "plain-time", "info"), "suspend", "pi", "--tree", "--key", suspendKeyA, "--workers", "1"}, tc.flags...)
				flags := strings.Join(tc.flags, " ")
				human := !strings.Contains(flags, "--json") && !strings.Contains(flags, "--keys-only") && !strings.Contains(flags, "--quiet")
				var exchanges []testx.CmdTerminalExchange
				if !tc.noPrompt {
					exchanges = []testx.CmdTerminalExchange{{Prompt: prompt, Response: tc.response, EndOfInput: tc.eof, BeforeResponse: func(out, stderr string) {
						require.Empty(t, out)
						calls, _, _ := f.snapshot()
						require.Empty(t, calls, "no execution before approval")
						require.Len(t, f.discoverySnapshot(), 4, "all discovery completes before approval")
						before := strings.TrimSuffix(stderr, prompt)
						if human {
							require.Equal(t, [][2]string{{"INFO", approval}}, bulkCompletionDiagnosticRecords(t, before, "plain-time"))
						} else {
							require.Empty(t, before)
						}
						if tc.race {
							f.mu.Lock()
							defer f.mu.Unlock()
							f.children[suspendKeyC] = []string{"2251799813685252"}
							f.observed["2251799813685252"] = "ACTIVE"
							f.observed[suspendKeyB] = "SUSPENDED"
							f.reject[suspendKeyB] = 409
						}
					}}}
				}
				out, stderr, err := runSuspensionTerminal(t, args, destination, exchanges)
				assertUserTaskCompletionExit(t, err, tc.code)
				require.NotContains(t, out, "proceed?")
				require.NotContains(t, out, "\r")
				if !tc.noPrompt {
					require.Equal(t, 1, strings.Count(stderr, prompt))
					stderr = strings.Replace(stderr, prompt, "", 1)
				} else {
					require.NotContains(t, stderr, "proceed?")
				}
				require.Equal(t, []string{"GET " + suspendKeyA, "POST " + suspendKeyA, "POST " + suspendKeyB, "POST " + suspendKeyC}, f.discoverySnapshot(), "no post-approval rediscovery")
				_, writes, reads := f.snapshot()
				if tc.approve {
					require.Equal(t, map[string]int{suspendKeyB: 1, suspendKeyC: 1}, writes)
					switch {
					case strings.Contains(flags, "--no-wait"):
						require.Empty(t, reads)
					case tc.race:
						require.Equal(t, map[string]int{suspendKeyC: 1}, reads)
					default:
						require.Equal(t, writes, reads)
					}
				} else {
					require.Empty(t, writes)
					require.Empty(t, reads)
				}
				switch {
				case strings.Contains(flags, "--json"):
					env := requireSingleJSONObjectDocument(t, out)
					require.Equal(t, "succeeded", env["outcome"])
					require.Equal(t, []any{suspendKeyB, suspendKeyC}, requireJSONObject(t, requireJSONObject(t, env["payload"])["plan"])["targetKeys"])
					require.Empty(t, stderr)
				case strings.Contains(flags, "--keys-only"):
					require.Equal(t, suspendKeyB+"\n"+suspendKeyC+"\n", out)
					require.Empty(t, stderr)
				default:
					require.Empty(t, out)
					want := [][2]string{}
					if human && !tc.dry {
						want = append(want, [2]string{"INFO", approval})
					}
					if tc.code == 1 {
						want = append(want, [2]string{"ERROR", "suspend process instances: aborted by user"})
					} else if human {
						if tc.dry {
							want = append(want, [2]string{"INFO", "dry run: suspend process-instance tree"})
						}
						want = append(want, [2]string{"INFO", scope})
						summary := "suspension: suspended 2/2 process instance(s)"
						if strings.Contains(flags, "--no-wait") {
							summary = "suspension: submitted 2/2 process instance(s); confirmation not requested"
						}
						if tc.dry {
							summary = "outcome: preview; no changes applied"
						}
						if tc.race {
							summary = "suspension: suspended 1/2 process instance(s); failed 1"
						}
						want = append(want, [2]string{"INFO", summary})
						if tc.race {
							want = append(want,
								[2]string{"INFO", suspendKeyA + ": observed state: SUSPENDED; classification: already_suspended"},
								[2]string{"INFO", suspendKeyB + ": observed state: ACTIVE; classification: eligible"},
								[2]string{"INFO", suspendKeyC + ": observed state: ACTIVE; classification: eligible"},
								[2]string{"INFO", suspendKeyB + ": submission: rejected; confirmation: not requested; last state: not observed"},
								[2]string{"INFO", suspendKeyC + ": submission: accepted; confirmation: confirmed; last state: SUSPENDED"},
								[2]string{"ERROR", "suspend process instances: rejected: 1"})
						}
					}
					require.Equal(t, want, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
				}
			})
		}
	}
}

// TestSuspendProcessInstanceTreeEmptyTerminal exercises prompt-free empty scopes
// with terminal stdin and redirected result streams, including implicit approval modes.
func TestSuspendProcessInstanceTreeEmptyTerminal(t *testing.T) {
	for _, destination := range []string{"configured", "inherited"} {
		for _, dry := range []bool{false, true} {
			for _, noWait := range []bool{false, true} {
				for _, flags := range [][]string{nil, {"--keys-only"}, {"--quiet"}, {"--quiet", "--keys-only"}, {"--auto-confirm"}, {"--automation"}, {"--auto-confirm", "--quiet", "--json"}, {"--automation", "--json", "--keys-only"}} {
					t.Run(fmt.Sprintf("%s/dry=%t/noWait=%t/%v", destination, dry, noWait, flags), func(t *testing.T) {
						f := newSuspensionTreeFixture(t)
						for key := range f.observed {
							f.observed[key] = "SUSPENDED"
						}
						args := append([]string{"--config", completionLogConfig(t, f.server.URL, "plain-time", "info"), "suspend", "pi", "--tree", "--key", suspendKeyA}, flags...)
						if dry {
							args = append(args, "--dry-run")
						}
						if noWait {
							args = append(args, "--no-wait")
						}
						out, stderr, err := runSuspensionTerminal(t, args, destination, nil)
						require.NoError(t, err, stderr)
						require.NotContains(t, stderr, "proceed?")
						mode := strings.Join(flags, " ")
						if strings.Contains(mode, "--json") {
							env := requireSingleJSONObjectDocument(t, out)
							require.Equal(t, "succeeded", env["outcome"])
							payload := requireJSONObject(t, env["payload"])
							require.Equal(t, []any{}, payload["items"])
							require.Equal(t, []any{}, requireJSONObject(t, payload["plan"])["targetKeys"])
							require.Empty(t, stderr)
						} else {
							require.Empty(t, out, "empty keys must be zero bytes")
							if strings.Contains(mode, "--quiet") || strings.Contains(mode, "--keys-only") {
								require.Empty(t, stderr)
							} else {
								want := [][2]string{}
								if dry {
									want = append(want, [2]string{"INFO", "dry run: suspend process-instance tree"})
								}
								want = append(want, [2]string{"INFO", "selected process instances: 1; process instances in scope: 3; active targets: 0; already suspended: 3; excluded: 0"})
								summary := "suspension: no active process instances to suspend; no changes applied"
								if dry {
									summary = "outcome: preview; no changes applied"
								}
								want = append(want, [2]string{"INFO", summary})
								require.Equal(t, want, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
							}
						}
						require.Len(t, f.discoverySnapshot(), 4)
						calls, writes, reads := f.snapshot()
						require.Empty(t, calls)
						require.Empty(t, writes)
						require.Empty(t, reads)
					})
				}
			}
		}
	}
}

// TestSuspendProcessInstanceTreePlanningFailure rejects late discovery errors
// before approval; configured diagnostics remain bounded while JSON retains the cause.
func TestSuspendProcessInstanceTreePlanningFailure(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, tc := range []struct {
				name  string
				flags []string
				level string
				debug bool
			}{
				{name: "normal"}, {name: "verbose", flags: []string{"--verbose"}}, {name: "quiet", flags: []string{"--quiet"}},
				{name: "json", flags: []string{"--auto-confirm", "--json"}}, {name: "keys", flags: []string{"--keys-only"}},
				{name: "debug flag", flags: []string{"--debug"}, debug: true}, {name: "debug config", level: "debug", debug: true},
				{name: "preview", flags: []string{"--dry-run"}}, {name: "no wait", flags: []string{"--no-wait"}},
			} {
				t.Run(format+"/"+destination+"/"+tc.name, func(t *testing.T) {
					f := newSuspensionTreeFixture(t)
					f.children = map[string][]string{}
					f.discoveryStatus[suspendKeyB] = 403
					level := tc.level
					if level == "" {
						level = "info"
					}
					args := append([]string{"--config", completionLogConfig(t, f.server.URL, format, level), "suspend", "pi", "--tree", "--key", suspendKeyA + "," + suspendKeyB}, tc.flags...)
					out, stderr, err := runSuspensionTerminal(t, args, destination, nil)
					assertUserTaskCompletionExit(t, err, 1)
					require.Equal(t, []string{"GET " + suspendKeyA, "POST " + suspendKeyA, "GET " + suspendKeyB}, f.discoverySnapshot())
					calls, _, _ := f.snapshot()
					require.Empty(t, calls)
					require.NotContains(t, stderr, "proceed?")
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					errors, debugs := []string{}, []string{}
					for _, r := range records {
						switch r[0] {
						case "ERROR":
							errors = append(errors, r[1])
						case "DEBUG":
							debugs = append(debugs, r[1])
						default:
							t.Errorf("unexpected diagnostic %v", r)
						}
					}
					if tc.name == "json" {
						require.Empty(t, stderr)
						env := requireSingleJSONObjectDocument(t, out)
						require.Equal(t, "failed", env["outcome"])
						require.NotContains(t, env, "payload")
						require.Contains(t, requireJSONObject(t, env["detail"])["message"], "body-only-secret")
					} else {
						require.Empty(t, out)
						require.Equal(t, []string{"plan process instance suspension: local precondition"}, errors)
					}
					for _, bad := range []string{"private.invalid", "body-only-secret", "private-cursor", "suspension-token-secret"} {
						require.NotContains(t, stderr, bad)
					}
					if tc.debug {
						require.NotEmpty(t, debugs)
					} else {
						require.Empty(t, debugs)
						for _, bad := range []string{"POST", "GET", f.server.URL, "/v2/", suspendKeyA, suspendKeyB} {
							require.NotContains(t, stderr, bad)
						}
					}
				})
			}
		}
	}
}
