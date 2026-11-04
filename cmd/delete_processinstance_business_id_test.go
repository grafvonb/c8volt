// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// newBusinessIDDeleteFixture shares the selection graph and owns exact deletion responses.
func newBusinessIDDeleteFixture(t *testing.T, empty, large bool, failure string) (*businessIDCancelFixture, *httptest.Server) {
	f, server := newBusinessIDCancelFixture(t, empty, large, failure)
	f.deletion = true
	return f, server
}

// businessIDDeleteArgs selects deletion with the same isolated configuration and tenant scope.
func businessIDDeleteArgs(t *testing.T, server *httptest.Server, format, level string, extra ...string) []string {
	args := businessIDCancelArgs(t, server, format, level, extra...)
	for i, arg := range args {
		if arg == "cancel" {
			args[i] = "delete"
			break
		}
	}
	return args
}

// TestBusinessIDDeleteScope proves global union/counts, seed limits, full membership and prompt-free dry-run.
func TestBusinessIDDeleteScope(t *testing.T) {
	for _, limit := range []bool{false, true} {
		for _, mode := range []string{"json", "keys", "verbose"} {
			t.Run(fmt.Sprintf("%t/%s", limit, mode), func(t *testing.T) {
				f, server := newBusinessIDDeleteFixture(t, false, false, "")
				extra := []string{"--dry-run", "--" + mode}
				if mode == "keys" {
					extra[1] = "--keys-only"
				}
				if limit {
					extra = append(extra, "--limit", "2")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, "plain-time", "info", extra...), "configured", false)
				require.NoError(t, err, stderr+out)
				members := []string{"101", "102", "103", "104", "201", "202", "203"}
				seeds := []string{"102", "104", "202"}
				roots := []string{"101", "201"}
				if limit {
					members = members[:4]
					seeds = seeds[:2]
					roots = roots[:1]
				}
				require.Empty(t, f.deleted.Snapshot())
				require.Equal(t, int32(3), f.searches.Load())
				for _, root := range roots {
					require.Equal(t, 1, countBusinessIDCancelKeys(f.children.Snapshot(), root))
				}
				switch mode {
				case "json":
					require.Empty(t, stderr)
					p := requireJSONObject(t, decodeBusinessIDCancelEnvelope(t, out)["payload"])
					require.Equal(t, len(seeds), int(p["requestedCount"].(float64)))
					require.Equal(t, len(roots), int(p["resolvedRootCount"].(float64)))
					require.Equal(t, len(members), int(p["affectedCount"].(float64)))
					require.EqualValues(t, members, interfaceStrings(t, p["affectedFamilyKeys"]))
					require.Equal(t, seeds, interfaceStrings(t, p["requestedKeys"]))
					require.Equal(t, roots, interfaceStrings(t, p["resolvedRoots"]))
					require.Equal(t, true, p["scopeComplete"])
					require.Equal(t, false, p["mutationSubmitted"])
					require.Contains(t, out, `"ORDER-123"`)
				case "keys":
					require.Equal(t, strings.Join(members, "\n")+"\n", out)
					require.Empty(t, stderr)
				case "verbose":
					require.Empty(t, out)
					records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
					require.NotEmpty(t, records)
					for _, r := range records {
						require.Equal(t, "INFO", r[0])
					}
					require.Contains(t, stderr, "in-scope process-instance keys: "+strings.Join(members, ", "))
					require.Contains(t, stderr, fmt.Sprintf("dependency expansion: %d additional", len(members)-len(seeds)))
				}
			})
		}
	}
}

// TestBusinessIDDeleteEmpty preserves every supported empty payload with real terminal stdin and zero prompts/writes.
func TestBusinessIDDeleteEmpty(t *testing.T) {
	for _, dry := range []bool{false, true} {
		for _, mode := range [][]string{nil, {"--quiet"}, {"--keys-only"}, {"--quiet", "--keys-only"}, {"--json"}, {"--quiet", "--json"}} {
			for _, policy := range [][]string{nil, {"--no-wait"}, {"--auto-confirm"}, {"--automation"}} {
				t.Run(fmt.Sprint(dry, mode, policy), func(t *testing.T) {
					f, server := newBusinessIDDeleteFixture(t, true, false, "")
					flags := append(slices.Clone(mode), policy...)
					if dry {
						flags = append(flags, "--dry-run")
					}
					out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, "plain-time", "info", flags...), "inherited", false)
					require.NoError(t, err, stderr+out)
					require.Equal(t, int32(1), f.searches.Load())
					require.Empty(t, f.deleted.Snapshot())
					require.Empty(t, f.children.Snapshot())
					if containsArg(flags, "--json") {
						require.Empty(t, stderr)
						envelope := decodeBusinessIDCancelEnvelope(t, out)
						require.Equal(t, "succeeded", envelope["outcome"])
						requireEmptyProcessInstanceSelectorOutput(t, out, stderr, "delete process-instance", "delete", dry, RenderModeJSON)
					} else {
						require.Empty(t, out)
						records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
						if containsArg(flags, "--quiet") || containsArg(flags, "--keys-only") {
							require.Empty(t, records)
						} else {
							require.Equal(t, [][2]string{{"INFO", "found: 0"}}, records)
						}
					}
				})
			}
		}
	}
}

// TestBusinessIDDeleteLargeAndIncomplete proves exhaustive child paging and no submission on late or malformed discovery.
func TestBusinessIDDeleteLargeAndIncomplete(t *testing.T) {
	for _, failure := range []string{"", "children", "search", "cycle", "unknown"} {
		t.Run(failure, func(t *testing.T) {
			large := failure == "" || failure == "children"
			f, server := newBusinessIDDeleteFixture(t, false, large, failure)
			flags := []string{"--json", "--auto-confirm", "--no-wait"}
			if failure == "" {
				flags = append(flags, "--dry-run")
			}
			out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, "plain-time", "info", flags...), "configured", false)
			require.Empty(t, stderr)
			envelope := decodeBusinessIDCancelEnvelope(t, out)
			p := requireJSONObject(t, envelope["payload"])
			require.Empty(t, f.deleted.Snapshot())
			if failure != "" {
				require.Error(t, err)
				require.NotEqual(t, "succeeded", envelope["outcome"])
				require.Equal(t, false, p["scopeComplete"])
			} else {
				require.NoError(t, err)
				require.Len(t, p["affectedFamilyKeys"], 1003)
				require.Contains(t, p["affectedFamilyKeys"], "2000")
				require.Contains(t, p["affectedFamilyKeys"], "2001")
				require.Equal(t, 2, countBusinessIDCancelKeys(f.children.Snapshot(), "101"))
			}
		})
	}
}

// TestBusinessIDDeleteExecution preserves verified/no-wait results, complete scope and selected-format routing.
func TestBusinessIDDeleteExecution(t *testing.T) {
	for _, noWait := range []bool{false, true} {
		for _, mode := range []string{"json", "keys", "verbose", "automation"} {
			t.Run(fmt.Sprint(noWait, mode), func(t *testing.T) {
				f, server := newBusinessIDDeleteFixture(t, false, false, "")
				flags := []string{"--auto-confirm"}
				if noWait {
					flags = append(flags, "--no-wait")
				}
				if mode == "keys" {
					flags = append(flags, "--keys-only")
				} else {
					flags = append(flags, "--"+mode)
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, "plain-time", "info", flags...), "inherited", false)
				require.NoError(t, err, stderr+out)
				require.Equal(t, []string{"103", "102", "104", "101", "202", "203", "201"}, f.deleted.Snapshot())
				require.Equal(t, int32(3), f.searches.Load())
				switch mode {
				case "json":
					envelope := decodeBusinessIDCancelEnvelope(t, out)
					want := "succeeded"
					if noWait {
						want = "accepted"
					}
					require.Equal(t, want, envelope["outcome"])
					p := requireJSONObject(t, envelope["payload"])
					require.Len(t, p["items"], 2)
					scope := requireJSONObject(t, p["scope"])
					require.Len(t, scope["affectedFamilyKeys"], 7)
					require.Equal(t, true, scope["mutationSubmitted"])
					require.Empty(t, stderr)
				case "keys":
					require.Equal(t, "101\n201\n", out)
					require.Empty(t, stderr)
				default:
					require.Empty(t, out)
					records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
					for _, r := range records {
						require.Equal(t, "INFO", r[0])
					}
					if mode == "verbose" {
						require.Contains(t, stderr, "in-scope process-instance keys: 101, 102, 103, 104, 201, 202, 203")
					}
				}
			})
		}
	}
}

// TestBusinessIDDeleteConfirmation freezes scope before plain configured-stderr approval and preserves decline/EOF.
func TestBusinessIDDeleteConfirmation(t *testing.T) {
	for _, destination := range []string{"configured", "inherited"} {
		for _, answer := range []string{"y", "n", "eof"} {
			t.Run(destination+"/"+answer, func(t *testing.T) {
				f, server := newBusinessIDDeleteFixture(t, false, false, "")
				prompt := "You have requested to delete 3 process instance(s), but due to dependencies, a total of 7 instance(s) with 2 root instance(s) will be deleted.\nDo you want to proceed? [y/N]: "
				exchange := testx.CmdTerminalExchange{Prompt: prompt, Response: answer, EndOfInput: answer == "eof", BeforeResponse: func(out, stderr string) {
					require.Empty(t, out)
					require.Empty(t, f.deleted.Snapshot())
					require.Equal(t, int32(3), f.searches.Load())
					require.Equal(t, 7, len(f.children.Snapshot()))
					f.afterApproval.Store(true)
				}}
				out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, "plain-time", "info", "--no-wait"), destination, false, exchange)
				require.Empty(t, out)
				require.Contains(t, stderr, prompt)
				require.NotContains(t, stderr, "INFO "+prompt)
				if answer == "y" {
					require.NoError(t, err, stderr)
					require.Equal(t, []string{"103", "102", "104", "101", "202", "203", "201"}, f.deleted.Snapshot())
				} else {
					require.Empty(t, f.deleted.Snapshot())
					require.Error(t, err)
				}
				require.Equal(t, int32(3), f.searches.Load())
			})
		}
	}
}

// TestBusinessIDDeleteLogging requires configured INFO/ERROR routing, threshold suppression and independent redacted DEBUG.
func TestBusinessIDDeleteLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, mode := range []string{"normal", "verbose", "quiet", "warn", "debug-flag", "debug-config", "json"} {
				for _, failure := range []string{"", "delete", "partial"} {
					t.Run(format+"/"+destination+"/"+mode+"/"+failure, func(t *testing.T) {
						f, server := newBusinessIDDeleteFixture(t, false, false, failure)
						level := "info"
						if mode == "warn" {
							level = "warn"
						}
						if mode == "debug-config" {
							level = "debug"
						}
						flags := []string{"--auto-confirm", "--no-wait"}
						switch mode {
						case "verbose", "quiet", "json":
							flags = append(flags, "--"+mode)
						case "debug-flag":
							flags = append(flags, "--debug")
						}
						out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, format, level, flags...), destination, false)
						if failure == "" {
							require.NoError(t, err, stderr+out)
						} else {
							assertRunBusinessIDExit(t, err, 2)
						}
						require.Equal(t, []string{"103", "102", "104", "101", "202", "203", "201"}, f.deleted.Snapshot())
						records := bulkCompletionDiagnosticRecords(t, stderr, format)
						var infos, errors, debugs []string
						for _, r := range records {
							switch r[0] {
							case "INFO":
								infos = append(infos, r[1])
							case "ERROR":
								errors = append(errors, r[1])
							case "DEBUG":
								debugs = append(debugs, r[1])
							default:
								t.Fatalf("unexpected severity %s", r[0])
							}
						}
						if mode == "json" {
							require.Empty(t, stderr)
							envelope := decodeBusinessIDCancelEnvelope(t, out)
							p := requireJSONObject(t, envelope["payload"])
							require.Len(t, p["items"], 2)
							if failure != "" {
								require.Equal(t, "invalid_input", envelope["class"])
								require.Contains(t, out, "body-only-secret")
								if failure == "partial" {
									require.Equal(t, true, requireJSONObject(t, p["items"].([]any)[0])["ok"])
								}
							}
						} else {
							require.Empty(t, out)
							if mode == "quiet" || mode == "warn" {
								require.Empty(t, infos)
							} else {
								require.NotEmpty(t, infos)
							}
							if failure == "" {
								require.Empty(t, errors)
							} else {
								success := 0
								if failure == "partial" {
									success = 1
								}
								require.Equal(t, []string{fmt.Sprintf("delete process instances: selected 3, roots 2, successful %d, submitted members %d; invalid input", success, 5+success)}, errors)
							}
						}
						for _, secret := range []string{"private.invalid", "secret-token", "secret-cookie", "body-only-secret", "resource-key-secret"} {
							require.NotContains(t, stderr, secret)
						}
						if strings.HasPrefix(mode, "debug-") {
							require.NotEmpty(t, debugs)
							require.Contains(t, stderr, "POST /v2/")
						} else {
							require.Empty(t, debugs)
							require.NotContains(t, stderr, "POST ")
							require.NotContains(t, stderr, server.URL)
						}
					})
				}
			}
		}
	}
}

// TestBusinessIDDeleteDryRunLogging rejects direct prints and ensures verbose preview suppression at configured thresholds.
func TestBusinessIDDeleteDryRunLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, mode := range []string{"normal", "verbose", "quiet", "warn"} {
				t.Run(format+destination+mode, func(t *testing.T) {
					f, server := newBusinessIDDeleteFixture(t, false, false, "")
					level := "info"
					flags := []string{"--dry-run"}
					if mode == "warn" {
						level = "warn"
					} else if mode != "normal" {
						flags = append(flags, "--"+mode)
					}
					out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, format, level, flags...), destination, false)
					require.NoError(t, err, stderr)
					require.Empty(t, out)
					require.Empty(t, f.deleted.Snapshot())
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					if mode == "quiet" || mode == "warn" {
						require.Empty(t, records)
					} else {
						require.NotEmpty(t, records)
						for _, r := range records {
							require.Equal(t, "INFO", r[0])
						}
					}
				})
			}
		}
	}
}

// TestBusinessIDDeleteRejectedInputs proves unsupported/empty/key/stdin inputs cannot trigger discovery or mutation.
func TestBusinessIDDeleteRejectedInputs(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, mode := range []string{"bid", "empty", "key", "stdin", "definition"} {
			if version == "8.10" && (mode == "bid" || mode == "definition") {
				continue
			}
			t.Run(version+mode, func(t *testing.T) {
				calls := &testx.SafeSlice[string]{}
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if serveVersionCheckTopology(w, r, version) {
						return
					}
					calls.Append(r.URL.Path)
					http.Error(w, "unexpected", 500)
				}))
				defer server.Close()
				args := []string{"delete", "pi", "--bid", "ORDER-123", "--json"}
				input := ""
				switch mode {
				case "empty":
					args[3] = ""
				case "key":
					args = append(args, "--key", "102")
				case "stdin":
					args = append(args, "-")
					input = "102\n"
				case "definition":
					args = append(args, "--bpmn-process-id", "order")
				}
				out, stderr, err := runGetUserTaskCommand(t, testx.WriteTestConfigForVersion(t, server.URL, version), input, args...)
				require.Error(t, err)
				require.Empty(t, stderr)
				decodeBusinessIDCancelEnvelope(t, out)
				require.Empty(t, calls.Snapshot())
			})
		}
	}
}

// TestBusinessIDDeleteOutputFailure makes exactly one JSON write attempt and retains the original operation exit.
func TestBusinessIDDeleteOutputFailure(t *testing.T) {
	for _, failure := range []string{"", "partial", "search"} {
		t.Run(failure, func(t *testing.T) {
			_, server := newBusinessIDDeleteFixture(t, false, false, failure)
			out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, "plain-time", "info", "--json", "--auto-confirm", "--no-wait"), "configured", true)
			require.Error(t, err)
			require.Equal(t, "attempt\n", out)
			require.NotContains(t, stderr, "body-only-secret")
			if failure != "" {
				assertRunBusinessIDExit(t, err, 2)
			}
		})
	}
}

// TestBusinessIDDeleteCombinedFilters retains every AND predicate and quotes opaque values in human previews.
func TestBusinessIDDeleteCombinedFilters(t *testing.T) {
	f, server := newBusinessIDDeleteFixture(t, false, false, "")
	f.bid = " ORDER,零*?\n "
	args := businessIDDeleteArgs(t, server, "plain-time", "info", "--dry-run", "--state", "active", "--bpmn-process-id", "order", "--start-date-before", "2026-05-31")
	for i, arg := range args {
		if arg == "--bid" {
			args[i+1] = f.bid
		}
	}
	out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
	require.NoError(t, err, stderr+out)
	require.Empty(t, out)
	require.Empty(t, f.deleted.Snapshot())
	for _, filter := range f.filters.Snapshot() {
		require.Equal(t, f.bid, filter["businessId"])
		require.Equal(t, "order", filter["processDefinitionId"])
		require.Equal(t, "ACTIVE", filter["state"])
		require.Contains(t, filter, "startDate")
		require.Equal(t, "tenant-a", filter["tenantId"])
	}
	require.Contains(t, stderr, fmt.Sprintf("selection business ID: %q", f.bid))
	require.NotContains(t, stderr, "零*?\n ")
}

// TestBusinessIDDeleteLargeVerbose retains every member beyond one backend page in configured INFO output.
func TestBusinessIDDeleteLargeVerbose(t *testing.T) {
	f, server := newBusinessIDDeleteFixture(t, false, true, "")
	out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, "plain-time", "info", "--dry-run", "--verbose"), "configured", false)
	require.NoError(t, err, stderr)
	require.Empty(t, out)
	require.Empty(t, f.deleted.Snapshot())
	members := []string{"101"}
	for i := 1000; i <= 2001; i++ {
		members = append(members, strconv.Itoa(i))
	}
	records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
	var scope []string
	for _, r := range records {
		require.Equal(t, "INFO", r[0])
		if strings.HasPrefix(r[1], "in-scope process-instance keys:") {
			scope = append(scope, r[1])
		}
	}
	require.Equal(t, []string{"in-scope process-instance keys: " + strings.Join(members, ", ")}, scope)
	require.Equal(t, 2, countBusinessIDCancelKeys(f.children.Snapshot(), "101"))
}

// TestBusinessIDDeleteChangedFamily refuses unplanned children and retains conflicts after plain terminal approval.
func TestBusinessIDDeleteChangedFamily(t *testing.T) {
	f, server := newBusinessIDDeleteFixture(t, false, false, "new-child")
	prompt := "You have requested to delete 3 process instance(s), but due to dependencies, a total of 7 instance(s) with 2 root instance(s) will be deleted.\nDo you want to proceed? [y/N]: "
	exchange := testx.CmdTerminalExchange{Prompt: prompt, Response: "y", BeforeResponse: func(out, stderr string) {
		require.Empty(t, out)
		require.Empty(t, f.deleted.Snapshot())
		f.afterApproval.Store(true)
	}}
	out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, "plain-time", "info", "--no-wait"), "configured", false, exchange)
	require.Error(t, err)
	require.Empty(t, out)
	require.Equal(t, []string{"103", "102", "104", "101", "202", "203", "201"}, f.deleted.Snapshot())
	require.NotContains(t, f.deleted.Snapshot(), "999")
	require.Len(t, f.children.Snapshot(), 7, "execution must not re-expand reviewed membership")
	require.Contains(t, stderr, "successful 1, submitted members 6; local precondition")
}

// TestBusinessIDDeleteForceAndWait preserves the force guard, root cancellation and mandatory prerequisite confirmation.
func TestBusinessIDDeleteForceAndWait(t *testing.T) {
	for _, noWait := range []bool{false, true} {
		for _, force := range []bool{false, true} {
			t.Run(fmt.Sprint(noWait, force), func(t *testing.T) {
				f, server := newBusinessIDDeleteFixture(t, false, false, "force")
				flags := []string{"--auto-confirm", "--json"}
				if noWait {
					flags = append(flags, "--no-wait")
				}
				if force {
					flags = append(flags, "--force")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, "plain-time", "info", flags...), "configured", false)
				require.Empty(t, stderr)
				envelope := decodeBusinessIDCancelEnvelope(t, out)
				if !force {
					require.Error(t, err)
					require.Empty(t, f.deleted.Snapshot())
					require.Empty(t, f.writes.Snapshot())
					return
				}
				require.NoError(t, err, out)
				want := "succeeded"
				if noWait {
					want = "accepted"
				}
				require.Equal(t, want, envelope["outcome"])
				require.Equal(t, []string{"103", "103", "102", "104", "101", "202", "203", "201"}, f.deleted.Snapshot())
				require.Equal(t, []string{"101"}, f.writes.Snapshot(), "existing force escalates a child cancellation to its root")
				require.Len(t, f.deletedSuccessfully.Snapshot(), 7)
				requests := f.requests.Snapshot()
				cancelIndex := slices.Index(requests, "POST /v2/process-instances/101/cancellation")
				require.Greater(t, cancelIndex, 0)
				require.Contains(t, requests[cancelIndex+1:], "GET /v2/process-instances/103", "no-wait still confirms the cancellation prerequisite")
			})
		}
	}
}

// TestBusinessIDDeleteWaitFailures retains submission evidence, original exits and one bounded diagnostic.
func TestBusinessIDDeleteWaitFailures(t *testing.T) {
	for _, failure := range []string{"delete-wait", "cancel-wait"} {
		for _, machine := range []bool{false, true} {
			t.Run(fmt.Sprint(failure, machine), func(t *testing.T) {
				f, server := newBusinessIDDeleteFixture(t, false, false, failure)
				flags := []string{"--auto-confirm", "--force"}
				if machine {
					flags = append(flags, "--json")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, "plain-time", "info", flags...), "configured", false)
				assertRunBusinessIDExit(t, err, 2)
				if machine {
					require.Empty(t, stderr)
					envelope := decodeBusinessIDCancelEnvelope(t, out)
					payload := requireJSONObject(t, envelope["payload"])
					require.Equal(t, true, requireJSONObject(t, payload["scope"])["mutationSubmitted"])
					require.NotEqual(t, "succeeded", envelope["outcome"])
					require.NotEmpty(t, payload["members"])
				} else {
					require.Empty(t, out)
					var failures []string
					for _, r := range bulkCompletionDiagnosticRecords(t, stderr, "plain-time") {
						if r[0] == "ERROR" {
							failures = append(failures, r[1])
						}
					}
					require.Len(t, failures, 1)
					require.NotContains(t, stderr, "body-only-secret")
					require.NotContains(t, stderr, "POST ")
					require.NotContains(t, stderr, "private.invalid")
				}
				require.NotContains(t, f.deleted.Snapshot(), "101", "failed child must prevent deleting its ancestor")
			})
		}
	}
}

// TestBusinessIDDeleteFailFast preserves unattempted roots and never submits an ancestor after a child failure.
func TestBusinessIDDeleteFailFast(t *testing.T) {
	f, server := newBusinessIDDeleteFixture(t, false, false, "delete-wait")
	out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, "plain-time", "info", "--auto-confirm", "--json", "--fail-fast"), "configured", false)
	assertRunBusinessIDExit(t, err, 2)
	require.Empty(t, stderr)
	payload := requireJSONObject(t, decodeBusinessIDCancelEnvelope(t, out)["payload"])
	require.Len(t, payload["items"], 2)
	require.Equal(t, "not attempted", requireJSONObject(t, payload["items"].([]any)[1])["status"])
	require.Equal(t, []string{"103"}, f.deleted.Snapshot())
}

// TestBusinessIDDeleteNoWaitPrerequisiteFailure retains submission evidence, original exits and one bounded diagnostic.
func TestBusinessIDDeleteNoWaitPrerequisiteFailure(t *testing.T) {
	for _, failure := range []string{"cancel-wait"} {
		for _, machine := range []bool{false, true} {
			t.Run(fmt.Sprint(failure, machine), func(t *testing.T) {
				f, server := newBusinessIDDeleteFixture(t, false, false, failure)
				flags := []string{"--auto-confirm", "--force", "--no-wait"}
				if machine {
					flags = append(flags, "--json")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, "plain-time", "info", flags...), "configured", false)
				assertRunBusinessIDExit(t, err, 2)
				if machine {
					require.Empty(t, stderr)
					envelope := decodeBusinessIDCancelEnvelope(t, out)
					payload := requireJSONObject(t, envelope["payload"])
					require.Equal(t, true, requireJSONObject(t, payload["scope"])["mutationSubmitted"])
					require.NotEqual(t, "succeeded", envelope["outcome"])
					require.NotEmpty(t, payload["members"])
				} else {
					require.Empty(t, out)
					var failures []string
					for _, r := range bulkCompletionDiagnosticRecords(t, stderr, "plain-time") {
						if r[0] == "ERROR" {
							failures = append(failures, r[1])
						}
					}
					require.Len(t, failures, 1)
					require.NotContains(t, stderr, "body-only-secret")
					require.NotContains(t, stderr, "POST ")
					require.NotContains(t, stderr, "private.invalid")
				}
				require.NotContains(t, f.deleted.Snapshot(), "101", "failed child must prevent deleting its ancestor")
			})
		}
	}
}

// TestBusinessIDDeleteRetryFailure retains accepted cancellation even when its subsequent delete request fails.
func TestBusinessIDDeleteRetryFailure(t *testing.T) {
	f, server := newBusinessIDDeleteFixture(t, false, false, "retry-fail")
	out, stderr, err := runBusinessIDCommand(t, businessIDDeleteArgs(t, server, "plain-time", "info", "--auto-confirm", "--force", "--no-wait", "--json"), "configured", false)
	assertRunBusinessIDExit(t, err, 2)
	require.Empty(t, stderr)
	payload := requireJSONObject(t, decodeBusinessIDCancelEnvelope(t, out)["payload"])
	require.Equal(t, true, payload["mutationSubmitted"])
	require.Equal(t, true, requireJSONObject(t, payload["scope"])["mutationSubmitted"])
	require.Equal(t, []string{"101"}, f.writes.Snapshot())
	require.Empty(t, f.deletedSuccessfully.Snapshot())
	require.Equal(t, []string{"103", "103", "202"}, f.deleted.Snapshot())
}
