// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestOpsAnalyseSlowProcessInstancesRejectsListenersWithKeysOnly verifies local validation runs before remote analysis.
func TestOpsAnalyseSlowProcessInstancesRejectsListenersWithKeysOnly(t *testing.T) {
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	prevKeysOnly := flagViewKeysOnly
	t.Cleanup(func() { flagViewKeysOnly = prevKeysOnly })
	flagViewKeysOnly = true
	flagOpsAnalyseSlowProcessInstanceKeys = []string{"2251799813685249"}
	flagOpsAnalyseSlowProcessInstanceWithListeners = true

	err := validateOpsSlowProcessAnalysisCommandArgs(cmd, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--with-listeners cannot be combined with --keys-only")
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

// TestOpsAnalyseSlowProcessInstancesRejectsInvalidDetailFilters verifies bad timeline filter values fail locally.
func TestOpsAnalyseSlowProcessInstancesRejectsInvalidDetailFilters(t *testing.T) {
	tests := []struct {
		name  string
		setup func()
		want  string
	}{
		{name: "bad root duration", setup: func() { flagOpsAnalyseSlowProcessInstanceDurationLonger = "soon" }, want: "invalid value for --dur-longer"},
		{name: "bad element duration", setup: func() { flagOpsAnalyseSlowProcessInstanceElementDurationLonger = "soon" }, want: "invalid value for --dur-element-longer"},
		{name: "negative element duration", setup: func() { flagOpsAnalyseSlowProcessInstanceElementDurationLonger = "-1s" }, want: "--dur-element-longer must not be negative"},
		{name: "bad type", setup: func() { flagOpsAnalyseSlowProcessInstanceType = "not-a-type" }, want: "invalid value for --type"},
		{name: "bad element state", setup: func() { flagOpsAnalyseSlowProcessInstanceElementState = "waiting" }, want: "invalid value for --element-state"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := resetOpsSlowProcessAnalysisTestFlags(t)
			flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"
			tc.setup()

			err := validateOpsSlowProcessAnalysisCommandArgs(cmd, nil)

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
