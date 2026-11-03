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

// TestFlagIntentVariablesSurface keeps removed flags out of accepted metadata while
// preserving their diagnostic-only replacements on every shared-source consumer.
func TestFlagIntentVariablesSurface(t *testing.T) {
	for _, cmd := range []*cobra.Command{updateElementCmd, updateProcessInstanceCmd, updateUserTaskCmd, opsRepairIncidentCmd, opsRepairProcessInstanceCmd} {
		t.Run(cmd.CommandPath(), func(t *testing.T) {
			names := map[string]string{"vars": "set-vars", "vars-file": "set-vars-file"}
			if cmd == updateUserTaskCmd {
				names["assign"] = "set-assignee"
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
			}
		})
	}
	require.NotNil(t, getUserTaskCmd.Flags().Lookup("assignee"))
}

// TestFlagIntentVariablesRejection exercises actual production leaves before
// bootstrap, including equal mixed input in either order and later JSON controls.
func TestFlagIntentVariablesRejection(t *testing.T) {
	var requests atomic.Int64
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(500) }))
	defer server.Close()
	cfg := testx.WriteTestConfig(t, server.URL)
	for _, words := range [][]string{{"update", "ei"}, {"update", "pi"}, {"update", "ut"}, {"ops", "repair", "incident"}, {"ops", "repair", "pi"}} {
		names := map[string]string{"vars": "set-vars", "vars-file": "set-vars-file"}
		if words[1] == "ut" {
			names["assign"] = "set-assignee"
		}
		for old, canonical := range names {
			for _, order := range []string{"old", "old-new", "new-old"} {
				t.Run(strings.Join(words, "/")+"/"+old+"/"+order, func(t *testing.T) {
					args := append([]string{}, words...)
					legacy := []string{"--" + old, "PRIVATE-NOT-READ"}
					current := []string{"--" + canonical, "PRIVATE-NOT-READ"}
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
					stdout, stderr := runCommandErrorEnvelopeSubprocess(t, "TestFlagIntentVariablesHelper", t.TempDir(), map[string]string{"C8VOLT_TEST_ARGS": string(encoded)}, "", exitcode.InvalidArgs)
					command := "update element"
					switch strings.Join(words, " ") {
					case "update pi":
						command = "update process-instance"
					case "update ut":
						command = "update user-task"
					case "ops repair incident":
						command = "ops repair incident"
					case "ops repair pi":
						command = "ops repair process-instance"
					}
					assertCommandErrorEnvelope(t, stdout, stderr, commandErrorEnvelopeExpectation{Outcome: "invalid", Class: "invalid_input", Command: command, Message: "invalid input: --" + old + " was removed; use --" + canonical})
					require.NotContains(t, stdout+stderr, "PRIVATE-NOT-READ")
				})
			}
		}
	}
	require.Zero(t, requests.Load())
}

// TestFlagIntentVariablesHelper executes production flags with unreadable stdin;
// migration errors must finish without input consumption or remote setup.
func TestFlagIntentVariablesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, "TestFlagIntentVariablesHelper", os.Getenv(testx.CmdSubprocessNameEnv))
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_ARGS")), &args))
	rootCmd.SetIn(migrationUnreadableInput{})
	os.Args = append([]string{"c8volt"}, args...)
	Execute()
}

// TestFlagIntentVariablesCompletion verifies the generated completion candidates
// advertise only canonical inputs, without attempting configuration bootstrap.
func TestFlagIntentVariablesCompletion(t *testing.T) {
	for _, words := range [][]string{{"update", "ei"}, {"update", "pi"}, {"update", "ut"}, {"ops", "repair", "incident"}, {"ops", "repair", "pi"}} {
		t.Run(strings.Join(words, "/"), func(t *testing.T) {
			output := executeCompletionNoDescForTest(t, append(words, "--")...)
			require.Contains(t, output, "--set-vars\n")
			require.Contains(t, output, "--set-vars-file\n")
			require.NotContains(t, output, "--vars\n")
			require.NotContains(t, output, "--vars-file\n")
			if words[1] == "ut" {
				require.Contains(t, output, "--set-assignee\n")
				require.NotContains(t, output, "--assign\n")
			}
		})
	}
}

// TestFlagIntentVariablesHumanRejection requires configured ERROR records, not
// raw parser text or direct stderr prints, under normal, verbose and quiet modes.
func TestFlagIntentVariablesHumanRejection(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, mode := range []string{"normal", "verbose", "quiet"} {
			t.Run(format+"/"+mode, func(t *testing.T) {
				cfg := completionLogConfig(t, "http://127.0.0.1:1", format, "warn")
				args := []string{"update", "ut", "--assign", "PRIVATE-IDENTITY", "--config", cfg}
				if mode != "normal" {
					args = append(args, "--"+mode)
				}
				encoded, err := json.Marshal(args)
				require.NoError(t, err)
				stdout, stderr := runCommandErrorEnvelopeSubprocess(t, "TestFlagIntentVariablesHelper", t.TempDir(), map[string]string{"C8VOLT_TEST_ARGS": string(encoded)}, "", exitcode.InvalidArgs)
				require.Empty(t, stdout)
				message := "invalid input: --assign was removed; use --set-assignee"
				if format == "json" {
					decoder := json.NewDecoder(strings.NewReader(stderr))
					var record map[string]any
					require.NoError(t, decoder.Decode(&record))
					require.Equal(t, "ERROR", record["level"])
					require.Equal(t, message, record["msg"])
					require.ErrorIs(t, decoder.Decode(&record), io.EOF)
				} else {
					require.Regexp(t, `^\S+ ERROR invalid input: --assign was removed; use --set-assignee\n$`, stderr)
				}
				require.NotContains(t, stderr, "PRIVATE-IDENTITY")
			})
		}
	}
}
