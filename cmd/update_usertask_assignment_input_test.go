// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// newAssignmentInputCommand registers the new modes on a private command only;
// shared legacy flags are restored by the existing input fixture.
func newAssignmentInputCommand(t *testing.T, argv ...string) *cobra.Command {
	t.Helper()
	cmd := newCompletionInputCommand(t)
	cmd.Flags().String("assign", "", "")
	cmd.Flags().Bool("unassign", false, "")
	cmd.Flags().Bool("complete", false, "")
	cmd.Flags().Bool("force", false, "")
	require.NoError(t, cmd.Flags().Parse(argv))
	return cmd
}

// TestUpdateUserTaskAssignmentInputSelection proves complete validation and
// stable flag-before-stdin uniqueness without accessing process stdin or services.
func TestUpdateUserTaskAssignmentInputSelection(t *testing.T) {
	for _, operation := range []string{"assign", "unassign"} {
		selector := "--" + operation
		if operation == "assign" {
			selector += "=alice"
		}
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
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				cmd := newAssignmentInputCommand(t, append([]string{selector}, tc.flags...)...)
				cmd.SetIn(strings.NewReader(tc.stdin))
				request, err := parseUpdateUserTaskAssignmentInput(cmd, tc.args)
				if tc.invalid {
					require.ErrorIs(t, err, ferrors.ErrInvalidInput)
					require.Empty(t, request.Keys)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tc.want, []string(request.Keys))
				require.Equal(t, operation, request.Operation)
				if len(flagUpdateUserTaskKeys) > 0 {
					flagUpdateUserTaskKeys[0] = "changed"
					require.Equal(t, tc.want, []string(request.Keys))
				}
			})
		}
	}
}

// TestUpdateUserTaskAssignmentInputControls verifies presence/value distinctions,
// opaque identities, worker bounds and existing machine confirmation guards.
func TestUpdateUserTaskAssignmentInputControls(t *testing.T) {
	for _, operation := range []string{"assign", "unassign"} {
		selector := "--" + operation
		if operation == "assign" {
			selector += "=  opaque\n\x1bidentity  "
		}
		for _, tc := range []struct {
			flags   []string
			invalid bool
			workers int
		}{
			{},
			{flags: []string{"--workers", "1"}, workers: 1},
			{flags: []string{"--workers", "0"}, invalid: true},
			{flags: []string{"--workers", "-1"}, invalid: true},
			{flags: []string{"--no-wait"}, invalid: true},
			{flags: []string{"--no-wait=false"}},
			{flags: []string{"--vars="}, invalid: true},
			{flags: []string{"--vars-file="}, invalid: true},
			{flags: []string{"--vars", "{}"}, invalid: true},
			{flags: []string{"--vars-file", "not-read.json"}, invalid: true},
			{flags: []string{"--complete"}, invalid: true},
			{flags: []string{"--complete=false"}},
			{flags: []string{"--force"}, invalid: operation == "unassign"},
			{flags: []string{"--force=false"}, invalid: operation == "unassign"},
			{flags: []string{"--json"}, invalid: true},
			{flags: []string{"--json", "--dry-run"}},
			{flags: []string{"--json", "--auto-confirm"}},
			{flags: []string{"--json", "--automation"}},
			{flags: []string{"--json", "--dry-run", "--verbose"}, invalid: true},
			{flags: []string{"--json", "--quiet", "--auto-confirm"}},
		} {
			t.Run(operation+strings.Join(tc.flags, " "), func(t *testing.T) {
				cmd := newAssignmentInputCommand(t, append([]string{selector, "-k", completionKeyA}, tc.flags...)...)
				input, err := parseUpdateUserTaskAssignmentInput(cmd, nil)
				if tc.invalid {
					require.ErrorIs(t, err, ferrors.ErrInvalidInput)
					require.Empty(t, input.Keys)
					return
				}
				require.NoError(t, err)
				require.Equal(t, operation, input.Operation)
				require.Equal(t, tc.workers, input.WantedWorkers)
				if operation == "assign" {
					require.Equal(t, "  opaque\n\x1bidentity  ", input.Assignee)
				} else {
					require.Empty(t, input.Assignee)
				}
				override, err := cmd.Flags().GetBool("force")
				require.NoError(t, err)
				require.Equal(t, override, input.AllowOverride)
				require.NoError(t, cmd.Flags().Set("assign", "changed"))
				require.NoError(t, cmd.Flags().Set("force", "true"))
				flagUpdateUserTaskKeys[0] = "changed"
				require.Equal(t, completionKeyA, input.Keys[0])
				require.Equal(t, override, input.AllowOverride)
				require.NotEqual(t, "changed", input.Assignee)
			})
		}
	}
	for _, flags := range [][]string{{"--assign="}, {"--assign", " \t\n"}, {"--assign=alice", "--unassign"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			cmd := newAssignmentInputCommand(t, append(flags, "-k", completionKeyA)...)
			_, err := parseUpdateUserTaskAssignmentInput(cmd, nil)
			require.ErrorIs(t, err, ferrors.ErrInvalidInput)
		})
	}
}

// TestUpdateUserTaskAssignmentInputModes preserves inactive legacy selectors and
// rejects assignment-only flag presence even when explicitly false.
func TestUpdateUserTaskAssignmentInputModes(t *testing.T) {
	for _, tc := range []struct {
		flags     []string
		operation string
		invalid   bool
	}{
		{},
		{flags: []string{"--unassign=false", "--complete=false", "--vars={}"}},
		{flags: []string{"--complete", "--unassign=false"}},
		{flags: []string{"--assign=", "--complete=false"}, operation: "assign"},
		{flags: []string{"--assign=alice", "--unassign=false"}, operation: "assign"},
		{flags: []string{"--unassign", "--complete=false"}, operation: "unassign"},
		{flags: []string{"--unassign", "--complete"}, invalid: true},
		{flags: []string{"--assign=alice", "--complete"}, invalid: true},
		{flags: []string{"--force=false"}, invalid: true},
		{flags: []string{"--force"}, invalid: true},
		{flags: []string{"--complete", "--force=false"}, invalid: true},
	} {
		t.Run(strings.Join(tc.flags, " "), func(t *testing.T) {
			cmd := newAssignmentInputCommand(t, tc.flags...)
			operation, err := resolveUpdateUserTaskAssignmentMode(cmd)
			if tc.invalid {
				require.ErrorIs(t, err, ferrors.ErrInvalidInput)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.operation, operation)
			if operation == "" {
				_, err = parseUpdateUserTaskAssignmentInput(cmd, nil)
				require.ErrorIs(t, err, ferrors.ErrInvalidInput)
			}
		})
	}
}

// TestUpdateUserTaskAssignmentInputReader bounds scanner failures and discards
// the entire selection after late failure, without reading unsolicited stdin.
func TestUpdateUserTaskAssignmentInputReader(t *testing.T) {
	for _, selector := range []string{"--assign=alice", "--unassign"} {
		cmd := newAssignmentInputCommand(t, selector, "-k", completionKeyA)
		broken := &completionBrokenReader{}
		cmd.SetIn(broken)
		_, err := parseUpdateUserTaskAssignmentInput(cmd, nil)
		require.NoError(t, err)
		require.Zero(t, broken.reads)
		for _, reader := range []io.Reader{io.MultiReader(bytes.NewBufferString(completionKeyB+"\n"), broken), strings.NewReader(strings.Repeat("x", 10*1024*1024+1))} {
			cmd.SetIn(reader)
			input, err := parseUpdateUserTaskAssignmentInput(cmd, []string{"-"})
			require.ErrorIs(t, err, ferrors.ErrInvalidInput)
			require.Contains(t, err.Error(), "could not read user task keys from stdin")
			require.NotContains(t, err.Error(), "distinctive reader failure")
			require.Less(t, len(err.Error()), 150)
			require.Empty(t, input.Keys)
		}
	}
}
