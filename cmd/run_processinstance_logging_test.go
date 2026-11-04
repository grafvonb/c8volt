// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// runBusinessIDConfig sets a fake-only endpoint and explicit logger configuration.
func runBusinessIDConfig(t *testing.T, url, format, level string) string {
	t.Helper()
	path := writeTestConfigForVersion(t, url, "8.10")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	require.NoError(t, err)
	_, err = fmt.Fprintf(f, "log:\n  level: %s\n  format: %s\n", level, format)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return path
}

// TestRunProcessInstanceBusinessIDFailureLogging covers repeated backend failures, partial evidence,
// original exits, independent DEBUG controls, redaction, and logger routing through command execution.
func TestRunProcessInstanceBusinessIDFailureLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for i, tc := range []struct {
			name, level   string
			flags         []string
			successes     int
			debug, writer bool
			exit          int
		}{
			{name: "normal partial", successes: 1, exit: 6},
			{name: "verbose partial", flags: []string{"--verbose"}, successes: 1, exit: 6},
			{name: "quiet partial", flags: []string{"--quiet"}, successes: 1, exit: 6},
			{name: "WARN partial", level: "warn", successes: 1, exit: 6},
			{name: "debug flag", flags: []string{"--debug"}, successes: 1, debug: true, exit: 6},
			{name: "debug config", level: "debug", successes: 1, debug: true, exit: 6},
			{name: "quiet overrides debug", flags: []string{"--debug", "--quiet"}, level: "debug", successes: 1, exit: 6},
			{name: "JSON", flags: []string{"--json"}, successes: 1, exit: 6},
			{name: "quiet JSON", flags: []string{"--quiet", "--json"}, successes: 1, exit: 6},
			{name: "debug JSON", flags: []string{"--debug", "--json"}, successes: 1, debug: true, exit: 6},
			{name: "keys", flags: []string{"--keys-only"}, successes: 1, exit: 6},
			{name: "quiet keys", flags: []string{"--quiet", "--keys-only"}, successes: 1, exit: 6},
			{name: "zero keys", flags: []string{"--keys-only"}, exit: 6},
			{name: "zero JSON", flags: []string{"--json"}, exit: 6},
			{name: "zero human", exit: 6},
			{name: "no error codes", flags: []string{"--no-err-codes", "--json"}, successes: 1},
			{name: "writer preserves error", flags: []string{"--json"}, successes: 1, writer: true, exit: 6},
			{name: "writer on success", flags: []string{"--json"}, successes: 4, writer: true, exit: 1},
			{name: "keys writer preserves error", flags: []string{"--keys-only"}, successes: 1, writer: true, exit: 6},
		} {
			t.Run(format+"/"+tc.name, func(t *testing.T) {
				f, srv := newRunBusinessIDFixture(t, "8.10", tc.successes, nil)
				level := tc.level
				if level == "" {
					level = "info"
				}
				args := []string{"--config", runBusinessIDConfig(t, srv.URL, format, level), "run", "pi", "--pd-key", "9001", "--set-bid", "same,ID", "--count", "4", "--workers", "2", "--no-wait"}
				args = append(args, tc.flags...)
				dest := "configured"
				if i%2 == 1 {
					dest = "inherited"
				}
				stdout, stderr, err := runBusinessIDCommand(t, args, dest, tc.writer)
				assertRunBusinessIDExit(t, err, tc.exit)
				require.Equal(t, int32(4), f.writes.Load())
				require.Zero(t, f.reads.Load())
				for _, body := range f.inputs.Snapshot() {
					require.Equal(t, "same,ID", body["businessId"])
				}
				records := bulkCompletionDiagnosticRecords(t, stderr, format)
				var errors, infos, debugs []string
				for _, record := range records {
					switch record[0] {
					case "ERROR":
						errors = append(errors, record[1])
					case "INFO":
						infos = append(infos, record[1])
					case "DEBUG":
						debugs = append(debugs, record[1])
					default:
						t.Errorf("unexpected record: %v", record)
					}
				}
				machine := slices.Contains(tc.flags, "--json")
				if machine {
					require.Empty(t, errors)
					require.Empty(t, infos)
				} else {
					require.Equal(t, []string{fmt.Sprintf("creating process instances: requested 4; successful %d; without successful result %d; conflict", tc.successes, 4-tc.successes)}, errors)
				}
				if slices.Contains(tc.flags, "--quiet") || level == "warn" || slices.Contains(tc.flags, "--keys-only") {
					require.Empty(t, infos)
				}
				if tc.writer {
					require.Equal(t, "attempt\n", stdout)
				} else if machine {
					envelope := decodeRunBusinessIDEnvelope(t, stdout)
					require.Equal(t, "failed", envelope["outcome"])
					require.Equal(t, "conflict", envelope["class"])
					require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "body-only-secret")
					payload := requireJSONObject(t, envelope["payload"])
					if tc.successes == 0 {
						require.Empty(t, payload, "preserve omitempty for zero totals and empty item slices")
					} else {
						require.Equal(t, float64(tc.successes), payload["total"])
						require.Len(t, payload["items"], tc.successes)
						for _, raw := range payload["items"].([]any) {
							item := requireJSONObject(t, raw)
							require.Equal(t, "101", item["key"])
							require.Equal(t, "native,ID", item["businessId"])
						}
					}
				} else if slices.Contains(tc.flags, "--keys-only") {
					want := ""
					if tc.successes > 0 {
						want = "101\n"
					}
					require.Equal(t, want, stdout)
				} else {
					require.Empty(t, stdout)
				}
				for _, bad := range []string{"body-only-secret", "resource-key-secret", "private.invalid", "leaseToken"} {
					require.NotContains(t, stderr, bad)
					if !machine {
						require.NotContains(t, stdout, bad)
					}
				}
				if tc.debug {
					require.NotEmpty(t, debugs)
					require.Contains(t, strings.Join(debugs, "\n"), "POST /v2/process-instances")
					require.Contains(t, strings.Join(debugs, "\n"), "status=409")
				} else {
					require.Empty(t, debugs)
					for _, bad := range []string{srv.URL, "POST", "/v2/process-instances"} {
						require.NotContains(t, stderr, bad)
						if !machine {
							require.NotContains(t, stdout, bad)
						}
					}
				}
			})
		}
	}
}

// TestRunProcessInstanceBusinessIDSummaryLogging requires INFO records for safely quoted observed IDs,
// suppresses them with WARN/quiet, and preserves explicit machine output with terminal stdin.
func TestRunProcessInstanceBusinessIDSummaryLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, level := range []string{"info", "warn"} {
			for _, flags := range [][]string{nil, {"--verbose"}, {"--quiet"}, {"--json"}, {"--quiet", "--json"}, {"--keys-only"}, {"--quiet", "--keys-only"}, {"--automation", "--json"}, {"--auto-confirm"}} {
				t.Run(format+"/"+level+"/"+strings.Join(flags, " "), func(t *testing.T) {
					f, srv := newRunBusinessIDFixture(t, "8.10", -1, toolx.Ptr("old\n\"ID"))
					args := []string{"--config", runBusinessIDConfig(t, srv.URL, format, level), "run", "pi", "--pd-key", "9001", "--set-bid", "new"}
					args = append(args, flags...)
					dest := "configured"
					if format == "json" {
						dest = "inherited"
					}
					stdout, stderr, err := runBusinessIDCommand(t, args, dest, false)
					require.NoError(t, err)
					require.Equal(t, int32(1), f.writes.Load())
					require.Equal(t, int32(1), f.reads.Load())
					require.Zero(t, f.searches.Load())
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					if slices.Contains(flags, "--json") {
						envelope := decodeRunBusinessIDEnvelope(t, stdout)
						require.Equal(t, "succeeded", envelope["outcome"])
						require.Empty(t, records)
						return
					}
					if slices.Contains(flags, "--keys-only") {
						require.Equal(t, "101\n", stdout)
						require.Empty(t, records)
						return
					}
					require.Empty(t, stdout)
					if level == "warn" || slices.Contains(flags, "--quiet") {
						require.Empty(t, records)
						return
					}
					var messages []string
					for _, r := range records {
						require.Equal(t, "INFO", r[0])
						messages = append(messages, r[1])
					}
					require.Contains(t, messages, "found: 1")
					require.Contains(t, strings.Join(messages, "\n"), `bid:"old\n\"ID"`)
					require.Len(t, messages, 3, "tenant, one row and summary only")
				})
			}
		}
	}
}

// TestRunProcessInstanceBusinessIDSequentialPartial preserves the successful prefix and counts unattempted work honestly.
func TestRunProcessInstanceBusinessIDSequentialPartial(t *testing.T) {
	for _, flags := range [][]string{{"--pd-key", "9001,9002,9003"}, {"--pd-key", "9001", "--count", "4", "--workers", "1"}, {"--pd-key", "9001", "--count", "20", "--workers", "1", "--fail-fast"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			f, srv := newRunBusinessIDFixture(t, "8.10", 1, toolx.Ptr("observed"))
			args := []string{"--config", writeTestConfigForVersion(t, srv.URL, "8.10"), "--json", "run", "pi", "--set-bid", "same"}
			args = append(args, flags...)
			stdout, stderr, err := runBusinessIDCommand(t, args, "inherited", false)
			assertRunBusinessIDExit(t, err, 6)
			require.Empty(t, stderr)
			env := decodeRunBusinessIDEnvelope(t, stdout)
			require.Equal(t, "conflict", env["class"])
			p := requireJSONObject(t, env["payload"])
			require.Equal(t, float64(1), p["total"])
			require.Len(t, p["items"], 1)
			item := requireJSONObject(t, p["items"].([]any)[0])
			require.Equal(t, "101", item["key"])
			require.Equal(t, "observed", item["businessId"])
			require.Equal(t, int32(1), f.reads.Load())
			if !slices.Contains(flags, "--count") {
				require.Equal(t, int32(2), f.writes.Load())
			} else if slices.Contains(flags, "--fail-fast") {
				require.Less(t, f.writes.Load(), int32(20))
			} else {
				require.Equal(t, int32(4), f.writes.Load())
			}
		})
	}
}

// TestRunProcessInstanceBusinessIDUnsupportedHuman reports the supported version without mutation or untrusted input.
func TestRunProcessInstanceBusinessIDUnsupportedHuman(t *testing.T) {
	for _, version := range []string{"8.8", "8.9"} {
		t.Run(version, func(t *testing.T) {
			f, srv := newRunBusinessIDFixture(t, version, -1, nil)
			args := []string{"--config", writeTestConfigForVersion(t, srv.URL, version), "--quiet", "run", "pi", "--pd-key", "9001", "--set-bid", "", "--count", "3", "--workers", "2"}
			stdout, stderr, err := runBusinessIDCommand(t, args, "configured", false)
			assertRunBusinessIDExit(t, err, 1)
			require.Empty(t, stdout)
			require.Zero(t, f.writes.Load())
			require.Zero(t, f.reads.Load())
			require.Equal(t, [][2]string{{"ERROR", "creating process instances: requested 3; successful 0; without successful result 3; business ID creation requires Camunda 8.10"}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
		})
	}
}
