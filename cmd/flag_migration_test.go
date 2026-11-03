// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestFlagIntentMigrationReporting exercises early rejection through Execute,
// including reporting controls that Cobra never reaches after the removed flag.
func TestFlagIntentMigrationReporting(t *testing.T) {
	var requests atomic.Int64
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	t.Cleanup(func() {
		require.Zero(t, requests.Load(), "migration reporting must not authenticate, check versions, discover or mutate")
	})
	for _, format := range []string{"plain", "json"} {
		for _, destination := range []string{"stderr", "leaf", "inherited"} {
			for _, machine := range []bool{false, true} {
				t.Run(format+"/"+destination+"/"+map[bool]string{false: "human", true: "envelope"}[machine], func(t *testing.T) {
					dir := t.TempDir()
					cfg := filepath.Join(dir, "config.yaml")
					require.NoError(t, os.WriteFile(cfg, []byte("apis:\n  camunda_api:\n    base_url: "+server.URL+"\nlog:\n  level: warn\n  format: "+format+"\nprofiles:\n  diagnostic:\n    app:\n      no_err_codes: false\n"), 0600))
					args := []string{"migration-test", "alias", "--old-value", "SECRET-PAYLOAD", "--config", cfg, "--profile=diagnostic", "-q", "--debug"}
					if machine {
						args = append(args, "-j")
					}
					encoded, err := json.Marshal(args)
					require.NoError(t, err)
					capture := filepath.Join(dir, "diagnostic.log")
					stdout, stderr := runCommandErrorEnvelopeSubprocess(t, "TestFlagIntentMigrationHelper", dir, map[string]string{
						"C8VOLT_TEST_ARGS": string(encoded), "C8VOLT_TEST_DESTINATION": destination, "C8VOLT_TEST_CAPTURE": capture,
					}, "", exitcode.InvalidArgs)
					diagnostic := stderr
					if destination != "stderr" {
						data, err := os.ReadFile(capture)
						require.NoError(t, err)
						diagnostic = string(data)
						require.Empty(t, stderr)
					}
					message := "invalid input: --old-value was removed; use --set-value"
					if machine {
						assertCommandErrorEnvelope(t, stdout, diagnostic, commandErrorEnvelopeExpectation{Outcome: "invalid", Class: "invalid_input", Command: "migration-test resource", Message: message})
					} else {
						require.Empty(t, stdout)
						require.Equal(t, 1, strings.Count(diagnostic, "--old-value was removed"))
						if format == "json" {
							decoder := json.NewDecoder(strings.NewReader(diagnostic))
							var record map[string]any
							require.NoError(t, decoder.Decode(&record))
							require.ErrorIs(t, decoder.Decode(&record), io.EOF)
							require.Equal(t, "ERROR", record["level"])
							require.Equal(t, message, record["msg"])
						} else {
							require.Regexp(t, `^\S+ ERROR invalid input: --old-value was removed; use --set-value\n$`, diagnostic)
						}
					}
					require.NotContains(t, stdout+stderr+diagnostic, "SECRET-PAYLOAD")
				})
			}
		}
	}
}

// migrationUnreadableInput makes any attempted stdin consumption fail the subprocess.
type migrationUnreadableInput struct{}

// Read must never be reached while preparing an early input diagnostic.
func (migrationUnreadableInput) Read([]byte) (int, error) {
	panic("migration diagnostics consumed stdin")
}

// TestFlagIntentMigrationHelper installs a synthetic consumer, keeping production
// flags unchanged until their owning story migrates them.
func TestFlagIntentMigrationHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, "TestFlagIntentMigrationHelper", os.Getenv(testx.CmdSubprocessNameEnv))
	parent := &cobra.Command{Use: "migration-test"}
	leaf := &cobra.Command{Use: "resource", Aliases: []string{"alias"}, Args: cobra.NoArgs,
		Run:         func(*cobra.Command, []string) { panic("rejected input executed") },
		Annotations: map[string]string{"c8volt.flag-migrations": `{"old-value":{"replacement":"--set-value","takes_value":true},"old-switch":{"replacement":"--new-switch","takes_value":false}}`},
	}
	leaf.Flags().String("set-value", "", "opaque value")
	leaf.Flags().StringP("payload", "p", "", "opaque payload")
	leaf.Flags().String("vars-file", "", "must not read this file")
	setFlagMigrations(leaf, flagMigrationsForCommand(leaf))
	setContractSupport(leaf, ContractSupportFull)
	parent.AddCommand(leaf)
	rootCmd.AddCommand(parent)
	leaf.SetIn(migrationUnreadableInput{})
	if destination := os.Getenv("C8VOLT_TEST_DESTINATION"); destination != "" && destination != "stderr" {
		f, err := os.Create(os.Getenv("C8VOLT_TEST_CAPTURE"))
		require.NoError(t, err)
		if destination == "leaf" {
			leaf.SetErr(f)
		} else {
			parent.SetErr(f)
		}
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_ARGS")), &args))
	os.Args = append([]string{"c8volt"}, args...)
	Execute()
}

// TestFlagIntentMigrationTokenBoundaries proves token arity, not substring
// matching, governs both rejection and diagnostic-only control recovery.
func TestFlagIntentMigrationTokenBoundaries(t *testing.T) {
	tests := []struct {
		name                  string
		args                  []string
		rejected, json, quiet bool
		config                string
	}{
		{name: "later controls", args: []string{"--old-value", "secret", "--json", "--config=chosen"}, rejected: true, json: true, config: "chosen"},
		{name: "attached removed value", args: []string{"--old-value=secret", "-qj"}, rejected: true, json: true, quiet: true},
		{name: "removed boolean arity", args: []string{"--old-switch", "-j"}, rejected: true, json: true},
		{name: "removed opaque value", args: []string{"--old-value", "--json", "-q"}, rejected: true, quiet: true},
		{name: "long opaque payload", args: []string{"--payload", "--old-value", "--json"}, json: true},
		{name: "attached payload", args: []string{"--payload=--old-value", "-j=false"}},
		{name: "short attached payload", args: []string{"-p--old-value", "-qj=true"}, quiet: true, json: true},
		{name: "short equals payload", args: []string{"-p=--json", "--old-switch"}, rejected: true},
		{name: "short separate payload", args: []string{"-p", "--json", "--old-switch"}, rejected: true},
		{name: "value after boolean cluster", args: []string{"-qp--json", "--old-switch"}, rejected: true, quiet: true},
		{name: "terminator", args: []string{"--", "--old-value", "--json"}},
		{name: "terminator after rejected", args: []string{"--old-switch", "--", "--json"}, rejected: true},
		{name: "boolean separated value is positional", args: []string{"-j", "false", "--old-switch"}, rejected: true, json: true},
		{name: "last override wins", args: []string{"-j", "--old-switch", "--json=false", "--config", "first", "--config=chosen"}, rejected: true, config: "chosen"},
		{name: "unknown short arity stops recovery", args: []string{"--old-switch", "-x", "--json"}, rejected: true},
		{name: "unknown arity stops recovery", args: []string{"--old-switch", "--unknown", "--json"}, rejected: true},
		{name: "opaque file name", args: []string{"--vars-file", "--json", "--old-switch"}, rejected: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := &cobra.Command{Use: "test"}
			leaf := &cobra.Command{Use: "resource"}
			root.AddCommand(leaf)
			root.PersistentFlags().BoolP("json", "j", false, "")
			root.PersistentFlags().BoolP("quiet", "q", false, "")
			root.PersistentFlags().String("config", "", "")
			leaf.Flags().StringP("payload", "p", "", "")
			leaf.Flags().String("vars-file", "", "")
			setFlagMigrations(leaf, map[string]flagMigration{"old-value": {Replacement: "--set-value", TakesValue: true}, "old-switch": {Replacement: "--new-switch"}})
			recovered, err := migrationReportingInput(leaf, tt.args)
			require.Equal(t, tt.rejected, err != nil)
			require.Equal(t, tt.json, boolFlag(recovered, "json"))
			require.Equal(t, tt.quiet, boolFlag(recovered, "quiet"))
			configPath, getErr := recovered.Flags().GetString("config")
			require.NoError(t, getErr)
			require.Equal(t, tt.config, configPath)
			require.False(t, root.PersistentFlags().Changed("json"))
			require.Nil(t, leaf.Flags().Lookup("old-value"))
			require.Nil(t, recovered.Flags().Lookup("payload"))
		})
	}
}

// TestFlagIntentMigrationEarlyFailures checks Args failures, fallback and exit
// overrides through the real execution boundary, never a renderer-only call.
func TestFlagIntentMigrationEarlyFailures(t *testing.T) {
	for _, tt := range []struct {
		name            string
		input, controls []string
		config          string
		exit            int
		message         string
		human           bool
	}{
		{name: "argument error", input: []string{"PRIVATE-POSITIONAL"}, exit: exitcode.InvalidArgs, message: "invalid input: invalid command arguments; see command help"},
		{name: "opaque payload", input: []string{"--payload", "--old-value", "PRIVATE-POSITIONAL"}, exit: exitcode.InvalidArgs, message: "invalid input: invalid command arguments; see command help"},
		{name: "flag value includes json", input: []string{"--old-value", "--json"}, exit: exitcode.InvalidArgs, message: "invalid input: --old-value was removed; use --set-value"},
		{name: "attached empty", input: []string{"--old-value="}, exit: exitcode.InvalidArgs, message: "invalid input: --old-value was removed; use --set-value"},
		{name: "old and new", input: []string{"--set-value=0", "--old-value=0"}, exit: exitcode.InvalidArgs, message: "invalid input: --old-value was removed; use --set-value"},
		{name: "new and old", input: []string{"--old-value=0", "--set-value=0"}, exit: exitcode.InvalidArgs, message: "invalid input: --old-value was removed; use --set-value"},
		{name: "no error codes flag", input: []string{"--old-switch"}, controls: []string{"--no-err-codes"}, exit: 0, message: "invalid input: --old-switch was removed; use --new-switch"},
		{name: "no error codes profile", input: []string{"--old-switch"}, controls: []string{"--profile=diagnostic"}, config: "profiles:\n  diagnostic:\n    app:\n      no_err_codes: true\n", exit: 0, message: "invalid input: --old-switch was removed; use --new-switch"},
		{name: "explicit exit override", input: []string{"--old-switch"}, controls: []string{"--no-err-codes=false"}, config: "app:\n  no_err_codes: true\n", exit: exitcode.InvalidArgs, message: "invalid input: --old-switch was removed; use --new-switch"},
		{name: "bad local config", input: []string{"--old-switch"}, config: "invalid: [", exit: exitcode.InvalidArgs, message: "invalid input: --old-switch was removed; use --new-switch; local diagnostic configuration could not be loaded"},
		{name: "bad profile", input: []string{"--old-switch"}, controls: []string{"--profile=absent"}, exit: exitcode.InvalidArgs, message: "invalid input: --old-switch was removed; use --new-switch; local diagnostic configuration could not be loaded"},
		{name: "bad local config human fallback", input: []string{"--old-switch"}, config: "invalid: [", exit: exitcode.InvalidArgs, message: "invalid input: --old-switch was removed; use --new-switch; local diagnostic configuration could not be loaded", human: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cfg := filepath.Join(dir, "config.yaml")
			require.NoError(t, os.WriteFile(cfg, []byte(tt.config), 0600))
			args := append([]string{"migration-test", "alias"}, tt.input...)
			args = append(args, "--vars-file", filepath.Join(dir, "must-not-be-read.json"), "--config", cfg)
			args = append(args, tt.controls...)
			if !tt.human {
				args = append(args, "--json")
			}
			encoded, err := json.Marshal(args)
			require.NoError(t, err)
			stdout, stderr := runCommandErrorEnvelopeSubprocess(t, "TestFlagIntentMigrationHelper", dir, map[string]string{"C8VOLT_TEST_ARGS": string(encoded)}, "", tt.exit)
			if tt.human {
				require.Empty(t, stdout)
				require.Regexp(t, `^\S+ \S+ ERROR `, stderr)
				require.Equal(t, 1, strings.Count(stderr, tt.message))
				require.Equal(t, 1, strings.Count(stderr, "\n"))
			} else {
				assertCommandErrorEnvelope(t, stdout, stderr, commandErrorEnvelopeExpectation{Outcome: "invalid", Class: "invalid_input", Command: "migration-test resource", Message: tt.message})
			}
			require.NotContains(t, stdout+stderr, "PRIVATE-POSITIONAL")
			require.NotContains(t, stdout+stderr, "must-not-be-read")
		})
	}
}

// TestFlagIntentMigrationTerminatorExecution keeps post-terminator syntax opaque
// even when those strings resemble both a removed name and an output override.
func TestFlagIntentMigrationTerminatorExecution(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfg, []byte("log:\n  format: json\n"), 0600))
	args := []string{"--config", cfg, "migration-test", "alias", "--", "--old-value", "--json"}
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	stdout, stderr := runCommandErrorEnvelopeSubprocess(t, "TestFlagIntentMigrationHelper", dir, map[string]string{"C8VOLT_TEST_ARGS": string(encoded)}, "", exitcode.InvalidArgs)
	require.Empty(t, stdout)
	decoder := json.NewDecoder(strings.NewReader(stderr))
	var record map[string]any
	require.NoError(t, decoder.Decode(&record))
	require.ErrorIs(t, decoder.Decode(&record), io.EOF)
	require.Equal(t, "ERROR", record["level"])
	require.Equal(t, "invalid input: invalid command arguments; see command help", record["msg"])
}
