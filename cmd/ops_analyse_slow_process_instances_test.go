// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
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

// TestOpsAnalyseSlowProcessInstancesBuildsProcessDefinitionSearchRequests verifies selector and discovery flags normalize to search mode.
func TestOpsAnalyseSlowProcessInstancesBuildsProcessDefinitionSearchRequests(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*cobra.Command)
		wantBPMN   string
		wantPDKey  string
		wantState  process.State
		wantAfter  string
		wantBefore string
	}{
		{
			name: "bpmn selector with all state and date filters",
			setup: func(cmd *cobra.Command) {
				flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"
				flagOpsAnalyseSlowProcessInstanceState = "all"
				require.NoError(t, cmd.Flags().Set("state", "all"))
				flagOpsAnalyseSlowProcessInstanceStartDateAfter = "2026-07-18T10:00:00Z"
				flagOpsAnalyseSlowProcessInstanceStartDateBefore = "2026-07-19"
				flagOpsAnalyseSlowProcessInstanceNoIncidentsOnly = true
				flagOpsAnalyseSlowProcessInstanceBatchSize = 25
				require.NoError(t, cmd.Flags().Set("batch-size", "25"))
				flagOpsAnalyseSlowProcessInstanceLimit = 10
				require.NoError(t, cmd.Flags().Set("limit", "10"))
			},
			wantBPMN:   "OrderProcess",
			wantState:  "",
			wantAfter:  "2026-07-18T10:00:00Z",
			wantBefore: "2026-07-19T23:59:59.999999999Z",
		},
		{
			name: "process-definition-key selector with completed state",
			setup: func(cmd *cobra.Command) {
				flagOpsAnalyseSlowProcessInstancePDKey = "2251799813687001"
				flagOpsAnalyseSlowProcessInstanceState = "completed"
				require.NoError(t, cmd.Flags().Set("state", "completed"))
				flagOpsAnalyseSlowProcessInstanceEndDateAfter = "2026-07-18T10:00:00.123"
			},
			wantPDKey: "2251799813687001",
			wantState: process.StateCompleted,
			wantAfter: "2026-07-18T10:00:00.123Z",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := resetOpsSlowProcessAnalysisTestFlags(t)
			tc.setup(cmd)

			got, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, nil, nil)

			require.NoError(t, err)
			require.Equal(t, ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch, got.Request.SelectionMode)
			require.Equal(t, tc.wantBPMN, got.Request.ProcessDefinitionSelector.BpmnProcessID)
			require.Equal(t, tc.wantPDKey, got.Request.ProcessDefinitionSelector.ProcessDefinitionKey)
			require.Equal(t, tc.wantState, got.Request.ProcessInstanceFilters.State)
			if tc.wantBefore != "" {
				require.Equal(t, tc.wantAfter, got.Request.ProcessInstanceFilters.StartDateAfter)
				require.Equal(t, tc.wantBefore, got.Request.ProcessInstanceFilters.StartDateBefore)
				require.True(t, got.Request.ProcessInstanceFilters.NoIncidentsOnly)
				require.EqualValues(t, 25, got.Request.BatchSize)
				require.EqualValues(t, 10, got.Request.Limit)
			} else {
				require.Equal(t, tc.wantAfter, got.Request.ProcessInstanceFilters.EndDateAfter)
			}
		})
	}
}

// TestOpsAnalyseSlowProcessInstancesRejectsInvalidSearchInputs verifies search-mode validation stays local.
func TestOpsAnalyseSlowProcessInstancesRejectsInvalidSearchInputs(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*cobra.Command)
		want  string
	}{
		{name: "required selector", want: "select process instances"},
		{name: "both selectors", setup: func(*cobra.Command) {
			flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"
			flagOpsAnalyseSlowProcessInstancePDKey = "2251799813687001"
		}, want: "cannot be combined"},
		{name: "bad date", setup: func(*cobra.Command) {
			flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"
			flagOpsAnalyseSlowProcessInstanceStartDateAfter = "2026-02-30"
		}, want: "invalid value for --start-date-after"},
		{name: "reversed date range", setup: func(*cobra.Command) {
			flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"
			flagOpsAnalyseSlowProcessInstanceEndDateAfter = "2026-07-20"
			flagOpsAnalyseSlowProcessInstanceEndDateBefore = "2026-07-19"
		}, want: "invalid range for --end-date-after and --end-date-before"},
		{name: "key with batch size", setup: func(cmd *cobra.Command) {
			flagOpsAnalyseSlowProcessInstanceKeys = []string{"2251799813685249"}
			flagOpsAnalyseSlowProcessInstanceBatchSize = 25
			require.NoError(t, cmd.Flags().Set("batch-size", "25"))
		}, want: "search filters"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := resetOpsSlowProcessAnalysisTestFlags(t)
			if tc.setup != nil {
				tc.setup(cmd)
			}

			err := validateOpsSlowProcessAnalysisCommandArgs(cmd, nil)

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
		})
	}
}

// TestOpsAnalyseSlowProcessInstancesDoesNotExposeIncidentsOnly verifies the unsupported positive incident filter is absent.
func TestOpsAnalyseSlowProcessInstancesDoesNotExposeIncidentsOnly(t *testing.T) {
	require.Nil(t, opsAnalyseSlowProcessInstancesCmd.Flags().Lookup("incidents-only"))
	require.NotContains(t, strings.ReplaceAll(opsAnalyseSlowProcessInstancesCmd.Flags().FlagUsages(), "--no-incidents-only", ""), "--incidents-only")
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
