// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const completionKeyA = "2251799813685249"
const completionKeyB = "2251799813685250"

// newCompletionInputCommand binds only existing flags on a private command;
// restoring globals keeps preparation tests independent of the real command tree.
func newCompletionInputCommand(t *testing.T, argv ...string) *cobra.Command {
	t.Helper()
	keys, raw, file := flagUpdateUserTaskKeys, flagUpdateUserTaskVars, flagUpdateUserTaskVarsFile
	workers, noWait, dry := flagWorkers, flagNoWait, flagDryRun
	verbose, quiet, asJSON, keysOnly := flagVerbose, flagQuiet, flagViewAsJson, flagViewKeysOnly
	auto, automation := flagCmdAutoConfirm, flagCmdAutomation
	t.Cleanup(func() {
		flagUpdateUserTaskKeys, flagUpdateUserTaskVars, flagUpdateUserTaskVarsFile = keys, raw, file
		flagWorkers, flagNoWait, flagDryRun = workers, noWait, dry
		flagVerbose, flagQuiet, flagViewAsJson, flagViewKeysOnly = verbose, quiet, asJSON, keysOnly
		flagCmdAutoConfirm, flagCmdAutomation = auto, automation
	})
	cmd := &cobra.Command{Use: "user-task"}
	fs := cmd.Flags()
	fs.StringSliceVarP(&flagUpdateUserTaskKeys, "key", "k", nil, "")
	trackUserTaskCompletionKeyInput(fs.Lookup("key"))
	fs.StringVar(&flagUpdateUserTaskVars, "vars", "", "")
	fs.StringVar(&flagUpdateUserTaskVarsFile, "vars-file", "", "")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "")
	fs.BoolVar(&flagNoWait, "no-wait", false, "")
	fs.BoolVar(&flagDryRun, "dry-run", false, "")
	fs.BoolVar(&flagVerbose, "verbose", false, "")
	fs.BoolVar(&flagQuiet, "quiet", false, "")
	fs.BoolVar(&flagViewAsJson, "json", false, "")
	fs.BoolVar(&flagViewKeysOnly, "keys-only", false, "")
	fs.BoolVar(&flagCmdAutoConfirm, "auto-confirm", false, "")
	fs.BoolVar(&flagCmdAutomation, "automation", false, "")
	setContractSupport(cmd, ContractSupportFull)
	setAutomationSupport(cmd, AutomationSupportFull, "")
	require.NoError(t, fs.Parse(argv))
	cmd.SetIn(strings.NewReader(""))
	return cmd
}

// TestUpdateUserTaskCompletionInputSelection proves complete validation and
// stable flag-before-stdin uniqueness without accessing process stdin or services.
func TestUpdateUserTaskCompletionInputSelection(t *testing.T) {
	for _, tc := range []struct {
		name    string
		flags   []string
		args    []string
		stdin   string
		want    []string
		invalid bool
	}{
		{name: "single", flags: []string{"-k", completionKeyA}, want: []string{completionKeyA}},
		{name: "combined", flags: []string{"-k", completionKeyB + "," + completionKeyA, "--key", completionKeyB}, args: []string{"-"}, stdin: "\n " + completionKeyA + " \r\n" + completionKeyB + "\n", want: []string{completionKeyB, completionKeyA}},
		{name: "stdin", args: []string{"-"}, stdin: completionKeyB + "\n" + completionKeyA, want: []string{completionKeyB, completionKeyA}},
		{name: "flags with empty stdin", flags: []string{"-k", completionKeyA}, args: []string{"-"}, want: []string{completionKeyA}},
		{name: "ignored stdin", flags: []string{"-k", completionKeyA}, stdin: "bad", want: []string{completionKeyA}},
		{name: "missing", invalid: true},
		{name: "blank stdin", args: []string{"-"}, stdin: " \n\t\n", invalid: true},
		{name: "empty flag", flags: []string{"--key="}, invalid: true},
		{name: "empty flag before valid", flags: []string{"--key=", "--key", completionKeyA}, invalid: true},
		{name: "empty flag after valid", flags: []string{"--key", completionKeyA, "--key="}, invalid: true},
		{name: "empty flag with valid stdin", flags: []string{"--key="}, args: []string{"-"}, stdin: completionKeyA, invalid: true},
		{name: "empty trailing token", flags: []string{"-k", completionKeyA + ","}, invalid: true},
		{name: "empty middle token", flags: []string{"-k", completionKeyA + ",," + completionKeyB}, invalid: true},
		{name: "late flag", flags: []string{"-k", completionKeyA + ",bad"}, invalid: true},
		{name: "late stdin", flags: []string{"-k", completionKeyA}, args: []string{"-"}, stdin: completionKeyB + "\nbad", invalid: true},
		{name: "stdin comma", args: []string{"-"}, stdin: completionKeyA + "," + completionKeyB, invalid: true},
		{name: "short key", flags: []string{"-k", "123"}, invalid: true},
		{name: "long key", flags: []string{"-k", "12251799813685249"}, invalid: true},
		{name: "signed key", flags: []string{"-k", "+251799813685249"}, invalid: true},
		{name: "unicode digits", flags: []string{"-k", "２２５１７９９８１３６８５２４９"}, invalid: true},
		{name: "invalid positional", flags: []string{"-k", completionKeyA}, args: []string{"unexpected"}, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newCompletionInputCommand(t, tc.flags...)
			cmd.SetIn(strings.NewReader(tc.stdin))
			request, err := parseUpdateUserTaskCompletionInput(cmd, tc.args)
			if tc.invalid {
				require.ErrorIs(t, err, ferrors.ErrInvalidInput)
				require.Empty(t, request.Keys)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, []string(request.Keys))
			require.Nil(t, request.Variables)
			if len(flagUpdateUserTaskKeys) > 0 {
				flagUpdateUserTaskKeys[0] = "changed"
				require.Equal(t, tc.want, []string(request.Keys))
			}
		})
	}
}

// TestUpdateUserTaskCompletionInputVariables covers optional source identity,
// nested JSON values and every invalid source before any runner exists.
func TestUpdateUserTaskCompletionInputVariables(t *testing.T) {
	dir := t.TempDir()
	payload := `{"nested":{"list":[true,2,"secret",null]}}`
	file := filepath.Join(dir, "vars.json")
	empty := filepath.Join(dir, "empty.json")
	require.NoError(t, os.WriteFile(file, []byte(payload), 0600))
	require.NoError(t, os.WriteFile(empty, nil, 0600))
	for _, tc := range []struct {
		name    string
		flags   []string
		want    map[string]any
		invalid bool
	}{
		{name: "omitted"},
		{name: "empty object", flags: []string{"--vars", "{}"}, want: map[string]any{}},
		{name: "nested", flags: []string{"--vars", payload}, want: map[string]any{"nested": map[string]any{"list": []any{true, json.Number("2"), "secret", nil}}}},
		{name: "file", flags: []string{"--vars-file", file}, want: map[string]any{"nested": map[string]any{"list": []any{true, json.Number("2"), "secret", nil}}}},
		{name: "both", flags: []string{"--vars", "{}", "--vars-file", file}, invalid: true},
		{name: "empty explicit source", flags: []string{"--vars="}, invalid: true},
		{name: "empty file path", flags: []string{"--vars-file="}, invalid: true},
		{name: "missing file", flags: []string{"--vars-file", filepath.Join(dir, "missing")}, invalid: true},
		{name: "empty file", flags: []string{"--vars-file", empty}, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newCompletionInputCommand(t, append([]string{"-k", completionKeyA}, tc.flags...)...)
			request, err := parseUpdateUserTaskCompletionInput(cmd, nil)
			if tc.invalid {
				require.ErrorIs(t, err, ferrors.ErrInvalidInput)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, request.Variables)
			// Replacing the original source cannot change the frozen payload.
			flagUpdateUserTaskVars = "{}"
			require.Equal(t, tc.want, request.Variables)
		})
	}
	for _, payload := range []string{" ", "{", "{} {}", "{} trailing", "[]", "null", "true", "12", `"text"`} {
		for _, source := range []string{"--vars", "--vars-file"} {
			t.Run(source+payload, func(t *testing.T) {
				value := payload
				if source == "--vars-file" {
					value = filepath.Join(t.TempDir(), "bad.json")
					require.NoError(t, os.WriteFile(value, []byte(payload), 0600))
				}
				cmd := newCompletionInputCommand(t, "-k", completionKeyA, source, value)
				_, err := parseUpdateUserTaskCompletionInput(cmd, nil)
				require.ErrorIs(t, err, ferrors.ErrInvalidInput)
			})
		}
	}
}

// TestUpdateUserTaskCompletionInputControls retains the existing machine-output
// guardrails while rejecting no-wait and explicit nonpositive worker counts.
func TestUpdateUserTaskCompletionInputControls(t *testing.T) {
	for _, tc := range []struct {
		flags   []string
		invalid bool
		workers int
	}{
		{flags: []string{"--workers", "1"}, workers: 1},
		{flags: []string{"--workers", "0"}, invalid: true},
		{flags: []string{"--workers", "-1"}, invalid: true},
		{flags: []string{"--no-wait"}, invalid: true},
		{flags: []string{"--no-wait=false"}},
		{flags: []string{"--json"}, invalid: true},
		{flags: []string{"--json", "--dry-run"}},
		{flags: []string{"--json", "--auto-confirm"}},
		{flags: []string{"--json", "--automation"}},
		{flags: []string{"--json", "--dry-run", "--verbose"}, invalid: true},
		{flags: []string{"--json", "--quiet", "--auto-confirm"}},
	} {
		t.Run(strings.Join(tc.flags, " "), func(t *testing.T) {
			cmd := newCompletionInputCommand(t, append([]string{"-k", completionKeyA}, tc.flags...)...)
			request, err := parseUpdateUserTaskCompletionInput(cmd, nil)
			if tc.invalid {
				require.ErrorIs(t, err, ferrors.ErrInvalidInput)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.workers, request.WantedWorkers)
		})
	}
}

// completionBrokenReader detects forbidden stdin reads and mid-stream failures.
type completionBrokenReader struct{ reads int }

// Read always fails so an input error cannot become a partial valid selection.
func (r *completionBrokenReader) Read([]byte) (int, error) {
	r.reads++
	return 0, errors.New("distinctive reader failure")
}

// TestUpdateUserTaskCompletionInputReader proves stdin is opt-in and read failures
// invalidate even a previously valid flag or streamed key.
func TestUpdateUserTaskCompletionInputReader(t *testing.T) {
	cmd := newCompletionInputCommand(t, "-k", completionKeyA)
	broken := &completionBrokenReader{}
	cmd.SetIn(broken)
	_, err := parseUpdateUserTaskCompletionInput(cmd, nil)
	require.NoError(t, err)
	require.Zero(t, broken.reads)
	cmd.SetIn(io.MultiReader(bytes.NewBufferString(completionKeyB+"\n"), broken))
	request, err := parseUpdateUserTaskCompletionInput(cmd, []string{"-"})
	require.ErrorIs(t, err, ferrors.ErrInvalidInput)
	require.Empty(t, request.Keys)
}

// TestUpdateUserTaskCompletionInputKeyTrackingPreservesLegacy proves the optional
// wrapper preserves pflag values and reset behavior for ordinary variable mode.
func TestUpdateUserTaskCompletionInputKeyTrackingPreservesLegacy(t *testing.T) {
	cmd := newCompletionInputCommand(t, "--key=", "--key", completionKeyA+","+completionKeyB)
	require.Equal(t, []string{completionKeyA, completionKeyB}, flagUpdateUserTaskKeys)
	values, err := cmd.Flags().GetStringSlice("key")
	require.NoError(t, err)
	require.Equal(t, flagUpdateUserTaskKeys, values)
	_, err = parseUpdateUserTaskCompletionInput(cmd, nil)
	require.ErrorIs(t, err, ferrors.ErrInvalidInput)
	resetCommandTreeFlags(cmd)
	require.NoError(t, cmd.Flags().Set("key", completionKeyB))
	input, err := parseUpdateUserTaskCompletionInput(cmd, nil)
	require.NoError(t, err)
	require.Equal(t, []string{completionKeyB}, []string(input.Keys))
	// Ordinary variable payloads still require a source; completion alone permits omission.
	_, err = parseUpdateVariablesFromFlags(cmd, "", "")
	require.ErrorIs(t, err, ferrors.ErrInvalidInput)
}
