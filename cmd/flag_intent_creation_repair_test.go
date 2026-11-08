// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestFlagIntentCreationRepairSurface keeps removed flags out of accepted metadata while
// preserving their diagnostic-only replacements on creation and repair commands.
func TestFlagIntentCreationRepairSurface(t *testing.T) {
	for _, cmd := range []*cobra.Command{runProcessInstanceCmd, opsRepairIncidentCmd, opsRepairProcessInstanceCmd} {
		t.Run(cmd.CommandPath(), func(t *testing.T) {
			names := map[string]string{"retries": "set-retries", "job-timeout": "set-job-timeout"}
			if cmd == runProcessInstanceCmd {
				names = map[string]string{"bid": "set-bid", "vars": "set-vars"}
			}
			capability := commandCapabilityForCommand(cmd)
			for old, canonical := range names {
				require.Nil(t, cmd.Flags().Lookup(old))
				require.NotNil(t, cmd.Flags().Lookup(canonical))
				require.Equal(t, flagMigration{Replacement: "--" + canonical, TakesValue: true}, flagMigrationsForCommand(cmd)[old])
				found := false
				for _, flag := range capability.Flags {
					require.NotEqual(t, old, flag.Name)
					found = found || flag.Name == canonical
				}
				require.True(t, found)
				words := strings.Fields(strings.TrimPrefix(cmd.CommandPath(), "c8volt "))
				output := executeCompletionNoDescForTest(t, append(words, "--")...)
				require.Contains(t, output, "--"+canonical+"\n")
				require.NotContains(t, output, "--"+old+"\n")
			}
		})
	}
	require.NotNil(t, opsRepairProcessInstanceCmd.Flags().Lookup("bid"))
	require.Nil(t, runProcessInstanceCmd.Flags().Lookup("set-vars-file"))
}

// TestFlagIntentCreationRepairRejection exercises actual production leaves before
// bootstrap, including equal mixed input in either order and later JSON controls.
func TestFlagIntentCreationRepairRejection(t *testing.T) {
	var requests atomic.Int64
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(500) }))
	defer server.Close()
	cfg := testx.WriteTestConfig(t, server.URL)
	for _, words := range [][]string{{"run", "pi"}, {"ops", "repair", "incident"}, {"ops", "repair", "pi"}} {
		names := map[string]string{"retries": "set-retries", "job-timeout": "set-job-timeout"}
		if words[0] == "run" {
			names = map[string]string{"bid": "set-bid", "vars": "set-vars"}
		}
		for old, canonical := range names {
			for _, order := range []string{"old", "old-new", "new-old"} {
				t.Run(strings.Join(words, "/")+"/"+old+"/"+order, func(t *testing.T) {
					args := append([]string{}, words...)
					value := "PRIVATE-NOT-READ"
					if old == "retries" {
						value = "0"
					}
					legacy := []string{"--" + old, value}
					current := []string{"--" + canonical, value}
					if order == "new-old" {
						args = append(args, current...)
					}
					args = append(args, legacy...)
					if order == "old-new" {
						args = append(args, current...)
					}
					args = append(args, "--config", cfg, "--json", "--quiet")
					encoded, err := json.Marshal(args)
					require.NoError(t, err)
					stdout, stderr := runCommandErrorEnvelopeSubprocess(t, "TestFlagIntentCreationRepairHelper", t.TempDir(), map[string]string{"C8VOLT_TEST_ARGS": string(encoded)}, "", exitcode.InvalidArgs)
					command := "run process-instance"
					if words[0] == "ops" {
						command = "ops repair incident"
						if words[2] == "pi" {
							command = "ops repair process-instance"
						}
					}
					assertCommandErrorEnvelope(t, stdout, stderr, commandErrorEnvelopeExpectation{Outcome: "invalid", Class: "invalid_input", Command: command, Message: "invalid input: --" + old + " was removed; use --" + canonical})
					require.NotContains(t, stdout+stderr, "PRIVATE-NOT-READ")
				})
			}
		}
	}
	require.Zero(t, requests.Load())
}

// TestFlagIntentCreationRepairHelper executes production flags with unreadable stdin;
// migration errors must finish without input consumption or remote setup.
func TestFlagIntentCreationRepairHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, "TestFlagIntentCreationRepairHelper", os.Getenv(testx.CmdSubprocessNameEnv))
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_ARGS")), &args))
	rootCmd.SetIn(migrationUnreadableInput{})
	os.Args = append([]string{"c8volt"}, args...)
	Execute()
	os.Exit(0)
}

// TestFlagIntentCreationRepairHumanRejection requires configured ERROR records, not
// raw parser text or direct stderr prints, under normal, verbose and quiet modes.
func TestFlagIntentCreationRepairHumanRejection(t *testing.T) {
	for _, words := range [][]string{{"run", "pi"}, {"ops", "repair", "inc"}, {"ops", "repair", "pi"}} {
		old, canonical := "bid", "set-bid"
		if words[0] == "ops" {
			old, canonical = "job-timeout", "set-job-timeout"
		}
		for _, format := range []string{"plain-time", "json"} {
			for _, mode := range []string{"normal", "verbose", "quiet"} {
				t.Run(strings.Join(words, "/")+"/"+format+"/"+mode, func(t *testing.T) {
					cfg := completionLogConfig(t, "http://127.0.0.1:1", format, "warn")
					args := append(append([]string{}, words...), "--"+old, "PRIVATE-IDENTITY", "--config", cfg)
					if mode != "normal" {
						args = append(args, "--"+mode)
					}
					encoded, err := json.Marshal(args)
					require.NoError(t, err)
					stdout, stderr := runCommandErrorEnvelopeSubprocess(t, "TestFlagIntentCreationRepairHelper", t.TempDir(), map[string]string{"C8VOLT_TEST_ARGS": string(encoded)}, "", exitcode.InvalidArgs)
					require.Empty(t, stdout)
					message := "invalid input: --" + old + " was removed; use --" + canonical
					if format == "json" {
						decoder := json.NewDecoder(strings.NewReader(stderr))
						var record map[string]any
						require.NoError(t, decoder.Decode(&record))
						require.Equal(t, "ERROR", record["level"])
						require.Equal(t, message, record["msg"])
						require.ErrorIs(t, decoder.Decode(&record), io.EOF)
					} else {
						require.Regexp(t, `^\S+ ERROR `+message+`\n$`, stderr)
					}
					require.NotContains(t, stderr, "PRIVATE-IDENTITY")
				})
			}
		}
	}
}

// TestFlagIntentCreationRepairPayloads verifies native job updates and the repair
// zero exception through both command runners, without changing report schemas.
func TestFlagIntentCreationRepairPayloads(t *testing.T) {
	for _, noun := range []string{"incident", "process-instance"} {
		for _, tc := range []struct {
			name  string
			flags []string
			want  []map[string]any
		}{
			{name: "default", want: []map[string]any{{"changeset": map[string]any{"retries": float64(1)}}}},
			{name: "explicit", flags: []string{"--set-retries", "3"}, want: []map[string]any{{"changeset": map[string]any{"retries": float64(3)}}}},
			{name: "zero", flags: []string{"--set-retries", "0"}},
			{name: "timeout with zero", flags: []string{"--set-retries", "0", "--set-job-timeout", "1500ms"}, want: []map[string]any{{"changeset": map[string]any{"timeout": float64(1500)}}}},
		} {
			t.Run(noun+"/"+tc.name, func(t *testing.T) {
				var requests testx.SafeSlice[string]
				var bodies testx.SafeSlice[map[string]any]
				fixture := newOpsRepairIncidentServer(t, &requests)
				key := "2251799813685249"
				if noun == "process-instance" {
					fixture.Close()
					fixture = newOpsRepairProcessInstanceServer(t, &requests)
					key = "2251799813685251"
				}
				defer fixture.Close()
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/v2/jobs/") {
						var body map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						bodies.Append(body)
						w.WriteHeader(http.StatusNoContent)
						return
					}
					fixture.Config.Handler.ServeHTTP(w, r)
				}))
				defer server.Close()
				args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.8"), "--json", "ops", "repair", noun, "--key", key, "--no-wait", "--auto-confirm"}
				args = append(args, tc.flags...)
				encoded, err := json.Marshal(args)
				require.NoError(t, err)
				stdout, stderr := runCommandErrorEnvelopeSubprocess(t, "TestFlagIntentCreationRepairHelper", t.TempDir(), map[string]string{"C8VOLT_TEST_ARGS": string(encoded)}, "", 0)
				require.Empty(t, stderr)
				var envelope map[string]any
				decoder := json.NewDecoder(strings.NewReader(stdout))
				require.NoError(t, decoder.Decode(&envelope))
				require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
				require.Equal(t, "ops repair "+noun, envelope["command"])
				require.Equal(t, "succeeded", envelope["outcome"])
				require.Equal(t, len(tc.want), len(bodies.Snapshot()))
				for i, body := range bodies.Snapshot() {
					require.Equal(t, tc.want[i], body)
				}
				require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "POST /v2/incidents/2251799813685249/resolution")
			})
		}
	}
}
