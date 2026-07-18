// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestOpsAnalyseSlowProcessInstancesBuildsExplicitKeyRequests verifies flags and stdin normalize to one keyed request.
func TestOpsAnalyseSlowProcessInstancesBuildsExplicitKeyRequests(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		keys      []string
		stdinKeys typex.Keys
		wantKeys  typex.Keys
	}{
		{name: "repeated --key", keys: []string{"2251799813685249", "2251799813685250"}, wantKeys: typex.Keys{"2251799813685249", "2251799813685250"}},
		{name: "stdin dash", args: []string{"-"}, stdinKeys: typex.Keys{"2251799813685251"}, wantKeys: typex.Keys{"2251799813685251"}},
		{name: "mixed flag and stdin keys", args: []string{"-"}, keys: []string{"2251799813685249"}, stdinKeys: typex.Keys{"2251799813685250"}, wantKeys: typex.Keys{"2251799813685249", "2251799813685250"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := resetOpsSlowProcessAnalysisTestFlags(t)
			flagOpsAnalyseSlowProcessInstanceKeys = append([]string(nil), tc.keys...)
			keys := append(typex.Keys{}, flagOpsAnalyseSlowProcessInstanceKeys...)
			keys = append(keys, tc.stdinKeys...)

			got, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, tc.args, keys.Unique())

			require.NoError(t, err)
			require.Equal(t, ops.SlowProcessAnalysisSelectionModeExplicitKeys, got.Request.SelectionMode)
			require.Equal(t, tc.wantKeys, got.Request.InputKeys)
			require.Equal(t, len(tc.args) == 1 && tc.args[0] == "-", got.StdinRequested)
			require.NotZero(t, got.Request.CapturedNow)
		})
	}
}

// TestOpsAnalyseSlowProcessInstancesKeyFlagHasShortAlias protects the documented -k shorthand.
func TestOpsAnalyseSlowProcessInstancesKeyFlagHasShortAlias(t *testing.T) {
	require.NotNil(t, opsAnalyseSlowProcessInstancesCmd.Flags().ShorthandLookup("k"))
	require.Equal(t, "key", opsAnalyseSlowProcessInstancesCmd.Flags().ShorthandLookup("k").Name)
}

// TestOpsAnalyseSlowProcessInstancesRejectsInvalidExplicitKeyInputs verifies local key-mode validation.
func TestOpsAnalyseSlowProcessInstancesRejectsInvalidExplicitKeyInputs(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*cobra.Command)
		args  []string
		want  string
	}{
		{name: "invalid key", setup: func(*cobra.Command) { flagOpsAnalyseSlowProcessInstanceKeys = []string{"bad"} }, want: "not a valid key"},
		{name: "extra positional args", args: []string{"-", "-"}, want: "unexpected positional arguments"},
		{name: "unexpected positional arg", args: []string{"2251799813685249"}, want: "unexpected positional argument"},
		{name: "key with bpmn selector", setup: func(*cobra.Command) {
			flagOpsAnalyseSlowProcessInstanceKeys = []string{"2251799813685249"}
			flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"
		}, want: "cannot be combined"},
		{name: "key with search filter", setup: func(cmd *cobra.Command) {
			flagOpsAnalyseSlowProcessInstanceKeys = []string{"2251799813685249"}
			flagOpsAnalyseSlowProcessInstanceState = "active"
			require.NoError(t, cmd.Flags().Set("state", "active"))
		}, want: "search filters"},
		{name: "both process definition selectors", setup: func(*cobra.Command) {
			flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"
			flagOpsAnalyseSlowProcessInstancePDKey = "2251799813687001"
		}, want: "cannot be combined"},
		{name: "no selector", want: "select process instances"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := resetOpsSlowProcessAnalysisTestFlags(t)
			if tc.setup != nil {
				tc.setup(cmd)
			}

			err := validateOpsSlowProcessAnalysisCommandArgs(cmd, tc.args)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

// TestOpsAnalyseSlowProcessInstancesRejectsEmptyStdin verifies dash input fails before remote analysis.
func TestOpsAnalyseSlowProcessInstancesRejectsEmptyStdin(t *testing.T) {
	oldStdin := os.Stdin
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = oldStdin
		require.NoError(t, reader.Close())
	})

	got, err := readKeysIfDash([]string{"-"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "stdin contained no keys")
	require.Nil(t, got)
}

// resetOpsSlowProcessAnalysisTestFlags restores command globals and returns a flag-aware test command.
func resetOpsSlowProcessAnalysisTestFlags(t *testing.T) *cobra.Command {
	t.Helper()
	flagOpsAnalyseSlowProcessInstanceKeys = nil
	flagOpsAnalyseSlowProcessInstanceBpmnProcessID = ""
	flagOpsAnalyseSlowProcessInstancePDKey = ""
	flagOpsAnalyseSlowProcessInstanceState = "all"
	flagOpsAnalyseSlowProcessInstanceStartDateAfter = ""
	flagOpsAnalyseSlowProcessInstanceStartDateBefore = ""
	flagOpsAnalyseSlowProcessInstanceEndDateAfter = ""
	flagOpsAnalyseSlowProcessInstanceEndDateBefore = ""
	flagOpsAnalyseSlowProcessInstanceNoIncidentsOnly = false
	flagOpsAnalyseSlowProcessInstanceBatchSize = consts.MaxPISearchSize
	flagOpsAnalyseSlowProcessInstanceLimit = 0
	flagOpsAnalyseSlowProcessInstanceElementID = ""
	flagOpsAnalyseSlowProcessInstanceType = ""
	flagOpsAnalyseSlowProcessInstanceElementState = ""
	flagOpsAnalyseSlowProcessInstanceDurationAfter = ""

	cmd := &cobra.Command{Use: "slow-process-instances"}
	flags := cmd.Flags()
	flags.StringVar(&flagOpsAnalyseSlowProcessInstanceState, "state", "all", "")
	flags.Int32Var(&flagOpsAnalyseSlowProcessInstanceBatchSize, "batch-size", consts.MaxPISearchSize, "")
	flags.Int32Var(&flagOpsAnalyseSlowProcessInstanceLimit, "limit", 0, "")
	return cmd
}
