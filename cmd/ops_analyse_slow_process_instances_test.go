// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
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

// TestOpsAnalyseSlowProcessInstancesWithFullTimelineFlagIsCommandLocal verifies the full detail switch stays out of facade input.
func TestOpsAnalyseSlowProcessInstancesWithFullTimelineFlagIsCommandLocal(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
		flagOpsAnalyseSlowProcessInstanceWithFullTimeline = false
	})

	flag := opsAnalyseSlowProcessInstancesCmd.Flags().Lookup("with-full-timeline")
	require.NotNil(t, flag)
	require.Equal(t, "false", flag.DefValue)
	require.Empty(t, flag.Shorthand)
	require.Contains(t, flag.Usage, "complete chronological element and transition detail")

	aliasCmd, remaining, err := root.Find([]string{"ops", "analyze", "spi", "--with-full-timeline"})
	require.NoError(t, err)
	require.Equal(t, []string{"--with-full-timeline"}, remaining)
	require.Same(t, opsAnalyseSlowProcessInstancesCmd, aliasCmd)
	require.NoError(t, aliasCmd.Flags().Set("with-full-timeline", "true"))

	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"
	flagOpsAnalyseSlowProcessInstanceWithFullTimeline = true

	got, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, nil, nil)

	require.NoError(t, err)
	require.True(t, got.WithFullTimeline)
	require.Equal(t, ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch, got.Request.SelectionMode)
	require.Equal(t, "one-line", got.Request.OutputMode)
}

// TestOpsAnalyseSlowProcessInstancesWithFullTimelineAllowsMachineModes verifies parse output mode remains independent.
func TestOpsAnalyseSlowProcessInstancesWithFullTimelineAllowsMachineModes(t *testing.T) {
	tests := []struct {
		name string
		mode string
		json bool
		keys bool
	}{
		{name: "json", mode: "json", json: true},
		{name: "keys-only", mode: "keys-only", keys: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := resetOpsSlowProcessAnalysisTestFlags(t)
			prevJSON := flagViewAsJson
			prevKeysOnly := flagViewKeysOnly
			t.Cleanup(func() {
				flagViewAsJson = prevJSON
				flagViewKeysOnly = prevKeysOnly
			})
			flagViewAsJson = tc.json
			flagViewKeysOnly = tc.keys
			flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"

			withoutFullTimeline, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, nil, nil)
			require.NoError(t, err)
			flagOpsAnalyseSlowProcessInstanceWithFullTimeline = true
			withFullTimeline, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, nil, nil)

			require.NoError(t, err)
			require.False(t, withoutFullTimeline.WithFullTimeline)
			require.True(t, withFullTimeline.WithFullTimeline)
			require.Equal(t, tc.mode, withFullTimeline.Request.OutputMode)
			require.Equal(t, withoutFullTimeline.Request.SelectionMode, withFullTimeline.Request.SelectionMode)
			require.Equal(t, withoutFullTimeline.Request.ProcessDefinitionSelector, withFullTimeline.Request.ProcessDefinitionSelector)
			require.Equal(t, withoutFullTimeline.Request.ProcessInstanceFilters, withFullTimeline.Request.ProcessInstanceFilters)
			require.Equal(t, withoutFullTimeline.Request.DetailFilters, withFullTimeline.Request.DetailFilters)
			require.Equal(t, withoutFullTimeline.Request.RootDurationLonger, withFullTimeline.Request.RootDurationLonger)
			require.Equal(t, withoutFullTimeline.Request.BatchSize, withFullTimeline.Request.BatchSize)
			require.Equal(t, withoutFullTimeline.Request.Limit, withFullTimeline.Request.Limit)
		})
	}
}

// TestOpsAnalyseSlowProcessInstancesWithListenersMapsRequest verifies listener enrichment is an explicit facade request.
func TestOpsAnalyseSlowProcessInstancesWithListenersMapsRequest(t *testing.T) {
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	flagOpsAnalyseSlowProcessInstanceKeys = []string{"2251799813685249"}
	flagOpsAnalyseSlowProcessInstanceWithListeners = true

	got, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, nil, typex.Keys{"2251799813685249"})

	require.NoError(t, err)
	require.True(t, got.Request.WithListeners)
	require.Equal(t, ops.SlowProcessAnalysisSelectionModeExplicitKeys, got.Request.SelectionMode)
	require.Equal(t, typex.Keys{"2251799813685249"}, got.Request.InputKeys)
}

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

// TestOpsAnalyseSlowProcessInstancesBuildsDetailFilters verifies timeline filters normalize into facade input.
func TestOpsAnalyseSlowProcessInstancesBuildsDetailFilters(t *testing.T) {
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"
	flagOpsAnalyseSlowProcessInstanceElementID = "ReserveStock"
	flagOpsAnalyseSlowProcessInstanceType = "service_task"
	flagOpsAnalyseSlowProcessInstanceElementState = "active"
	flagOpsAnalyseSlowProcessInstanceElementDurationLonger = "2m"

	got, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, nil, nil)

	require.NoError(t, err)
	require.Equal(t, "ReserveStock", got.Request.DetailFilters.ElementID)
	require.Equal(t, "SERVICE_TASK", got.Request.DetailFilters.Type)
	require.Equal(t, "ACTIVE", got.Request.DetailFilters.ElementState)
	require.Equal(t, 2*time.Minute, got.Request.DetailFilters.DurationAfter)
}

// TestOpsAnalyseSlowProcessInstancesBuildsRootDurationFilter verifies root duration filters are separate from detail filters.
func TestOpsAnalyseSlowProcessInstancesBuildsRootDurationFilter(t *testing.T) {
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	flagOpsAnalyseSlowProcessInstanceBpmnProcessID = "OrderProcess"
	flagOpsAnalyseSlowProcessInstanceDurationLonger = "5m"
	flagOpsAnalyseSlowProcessInstanceElementDurationLonger = "30s"

	got, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, nil, nil)

	require.NoError(t, err)
	require.Equal(t, 5*time.Minute, got.Request.RootDurationLonger)
	require.Equal(t, 30*time.Second, got.Request.DetailFilters.DurationAfter)
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

// TestOpsAnalyseSlowProcessInstancesDoesNotExposeIncidentsOnly verifies the unsupported positive incident filter is absent.
func TestOpsAnalyseSlowProcessInstancesDoesNotExposeIncidentsOnly(t *testing.T) {
	require.Nil(t, opsAnalyseSlowProcessInstancesCmd.Flags().Lookup("incidents-only"))
	require.NotContains(t, strings.ReplaceAll(opsAnalyseSlowProcessInstancesCmd.Flags().FlagUsages(), "--no-incidents-only", ""), "--incidents-only")
}

// TestOpsAnalyseSlowProcessInstancesDoesNotExposeDurationAfter verifies the old alias is not user-facing.
func TestOpsAnalyseSlowProcessInstancesDoesNotExposeDurationAfter(t *testing.T) {
	require.Nil(t, opsAnalyseSlowProcessInstancesCmd.Flags().Lookup("duration-after"))
	require.NotContains(t, opsAnalyseSlowProcessInstancesCmd.Long, "--duration-after")
	require.NotContains(t, opsAnalyseSlowProcessInstancesCmd.Example, "--duration-after")
	require.NotContains(t, opsAnalyseSlowProcessInstancesCmd.Flags().FlagUsages(), "--duration-after")
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

// TestOpsAnalyseSlowProcessInstancesPrintsPreflightCertaintyToStderr verifies human preflight wording covers exact, lower-bound, and unknown totals without stdout leakage.
func TestOpsAnalyseSlowProcessInstancesPrintsPreflightCertaintyToStderr(t *testing.T) {
	tests := []struct {
		name      string
		total     *int64
		kind      ops.TotalCertainty
		pageCount *int64
		pageKind  ops.PageCountKind
		want      []string
	}{
		{name: "exact", total: ptrInt64(2000), kind: ops.TotalCertaintyExact, pageCount: ptrInt64(2), pageKind: ops.PageCountKindExact, want: []string{"slow analysis scope: OrderProcess matched 2000 process instances", "discovery pages: 2"}},
		{name: "lower bound", total: ptrInt64(2000), kind: ops.TotalCertaintyLowerBound, pageCount: ptrInt64(2), pageKind: ops.PageCountKindEstimated, want: []string{"slow analysis scope: OrderProcess matched at least 2000 process instances", "discovery pages: at least 2"}},
		{name: "unknown", kind: ops.TotalCertaintyUnknown, pageKind: ops.PageCountKindUnknown, want: []string{"slow analysis scope: OrderProcess matched an unknown number of process instances", "page size: 1000"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := resetOpsSlowProcessAnalysisTestFlags(t)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			printOpsPreflightScope(cmd, ops.PreflightScope{
				Command:         "ops analyse slow-process-instances",
				SelectorSummary: "OrderProcess",
				CoreResource:    "process_instance",
				Total:           tc.total,
				TotalKind:       tc.kind,
				PageSize:        1000,
				PageCount:       tc.pageCount,
				PageCountKind:   tc.pageKind,
			}, ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true})

			require.Empty(t, stdout.String())
			for _, want := range tc.want {
				require.Contains(t, stderr.String(), want)
			}
		})
	}
}

// TestOpsAnalyseSlowProcessInstancesPreflightUsesCommandLogger keeps scope output aligned with other prompt preambles.
func TestOpsAnalyseSlowProcessInstancesPreflightUsesCommandLogger(t *testing.T) {
	total := int64(10000)
	pages := int64(10)
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	var logBuf bytes.Buffer
	cmd.SetContext(logging.ToContext(cmd.Context(), logging.New(logging.LoggerConfig{
		Format: "plain-time",
		Writer: &logBuf,
	})))

	printOpsPreflightScope(cmd, ops.PreflightScope{
		Command:         "ops analyse slow-process-instances",
		SelectorSummary: "OrderProcess",
		CoreResource:    "process_instance",
		Total:           &total,
		TotalKind:       ops.TotalCertaintyLowerBound,
		PageSize:        1000,
		PageCount:       &pages,
		PageCountKind:   ops.PageCountKindEstimated,
		ConsequenceSummary: ops.ConsequenceSummary{
			WorkSummary: "discover all matches and load runtime element timelines",
			RiskSummary: "read-only, expensive",
		},
	}, ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true})

	got := logBuf.String()
	require.Contains(t, got, "INFO slow analysis scope: OrderProcess matched at least 10000 process instances; page size: 1000; discovery pages: at least 10")
	require.Contains(t, got, "WARN slow analysis is expensive: discover all matches and load runtime element timelines")
}

// TestOpsAnalyseSlowProcessInstancesConfiguresBroadPreflightOnlyForSearch verifies explicit-key mode stays concise.
func TestOpsAnalyseSlowProcessInstancesConfiguresBroadPreflightOnlyForSearch(t *testing.T) {
	keyRequest := ops.SlowProcessAnalysisRequest{SelectionMode: ops.SlowProcessAnalysisSelectionModeExplicitKeys}
	configureOpsSlowProcessAnalysisPreflight(resetOpsSlowProcessAnalysisTestFlags(t), &keyRequest)
	require.Nil(t, keyRequest.Progress)
	require.Nil(t, keyRequest.ConfirmPreflight)

	searchRequest := ops.SlowProcessAnalysisRequest{SelectionMode: ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch}
	configureOpsSlowProcessAnalysisPreflight(resetOpsSlowProcessAnalysisTestFlags(t), &searchRequest)
	require.NotNil(t, searchRequest.Progress)
	require.NotNil(t, searchRequest.ConfirmPreflight)
}

// TestOpsAnalyseSlowProcessInstancesRoutesDefaultProgressToActivity verifies human search progress is visible without debug and never touches stdout.
func TestOpsAnalyseSlowProcessInstancesRoutesDefaultProgressToActivity(t *testing.T) {
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	sink := &activitysink.Sink{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetContext(logging.ToActivityContext(cmd.Context(), sink))
	request := ops.SlowProcessAnalysisRequest{SelectionMode: ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch}

	configureOpsSlowProcessAnalysisPreflight(cmd, &request)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPage, Page: &ops.PageProgress{
		Phase:         "discovering process instances",
		CurrentPage:   2,
		PageCount:     ptrInt64(4),
		PageCountKind: ops.PageCountKindExact,
		Seen:          1500,
		Selected:      1498,
	}})
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindFrozenScope, FrozenScope: &ops.FrozenScopeProgress{
		Phase:        "loading runtime elements",
		CoreResource: "process instance(s)",
		Done:         48,
		Total:        800,
	}})
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindETA, ETA: &ops.ETASampleWindow{
		Phase:             "loading runtime elements",
		CompletedSamples:  48,
		Total:             800,
		Elapsed:           2 * time.Minute,
		MinimumSamplesMet: true,
		Rate:              ptrFloat64(4.25),
		Remaining:         ptrDuration(95 * time.Second),
	}})

	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
	require.Equal(t, []string{
		"discovering process instances, page 2/4, 1500 seen, 1498 selected",
		"loading runtime elements, 48/800 process instance(s)",
		"loading runtime elements, 48/800 sample(s), 2m0s elapsed, ~4.2/s, ~1m35s remaining",
	}, sink.Updates())
	require.Equal(t, []activitysink.Update{
		{
			Message:    "discovering process instances, page 2/4, 1500 seen, 1498 selected",
			Importance: logging.ActivityImportanceWorkflow,
		},
		{
			Message:    "loading runtime elements, 48/800 process instance(s)",
			Importance: logging.ActivityImportanceWorkflow,
		},
		{
			Message:    "loading runtime elements, 48/800 sample(s), 2m0s elapsed, ~4.2/s, ~1m35s remaining",
			Importance: logging.ActivityImportanceWorkflow,
		},
	}, sink.PriorityUpdates())
}

// TestOpsAnalyseSlowProcessInstancesVerboseProgressWritesDurableStderr verifies verbose mode keeps an auditable progress trail off stdout.
func TestOpsAnalyseSlowProcessInstancesVerboseProgressWritesDurableStderr(t *testing.T) {
	previousVerbose := flagVerbose
	flagVerbose = true
	t.Cleanup(func() { flagVerbose = previousVerbose })
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	request := ops.SlowProcessAnalysisRequest{SelectionMode: ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch}

	configureOpsSlowProcessAnalysisPreflight(cmd, &request)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPage, Page: &ops.PageProgress{
		Phase:         "discovering process instances",
		CurrentPage:   2,
		PageCount:     ptrInt64(4),
		PageCountKind: ops.PageCountKindExact,
		Seen:          1500,
		Selected:      1498,
	}})
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindFrozenScope, FrozenScope: &ops.FrozenScopeProgress{
		Phase:        "loading listener jobs",
		CoreResource: "process instance(s)",
		Done:         3,
		Total:        3,
	}})

	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "discovering process instances, page 2/4, 1500 seen, 1498 selected")
	require.Contains(t, stderr.String(), "loading listener jobs, 3/3 process instance(s)")
}

// TestOpsAnalyseSlowProcessInstancesJSONProgressKeepsStdoutClean verifies JSON mode suppresses transient and durable progress text.
func TestOpsAnalyseSlowProcessInstancesJSONProgressKeepsStdoutClean(t *testing.T) {
	previousJSON := flagViewAsJson
	flagViewAsJson = true
	t.Cleanup(func() { flagViewAsJson = previousJSON })
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	sink := &activitysink.Sink{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetContext(logging.ToActivityContext(cmd.Context(), sink))
	request := ops.SlowProcessAnalysisRequest{SelectionMode: ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch}

	configureOpsSlowProcessAnalysisPreflight(cmd, &request)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPreflight, Preflight: &ops.PreflightScope{
		SelectorSummary: "OrderProcess",
		CoreResource:    "process_instance",
		Total:           ptrInt64(2000),
		TotalKind:       ops.TotalCertaintyExact,
		PageSize:        1000,
		PageCount:       ptrInt64(2),
		PageCountKind:   ops.PageCountKindExact,
	}})
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPage, Page: &ops.PageProgress{
		Phase:         "discovering process instances",
		CurrentPage:   1,
		PageCount:     ptrInt64(2),
		PageCountKind: ops.PageCountKindExact,
		Seen:          1000,
	}})
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindFrozenScope, FrozenScope: &ops.FrozenScopeProgress{
		Phase:        "loading runtime elements",
		CoreResource: "process instance(s)",
		Done:         1,
		Total:        2,
	}})

	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
	require.Empty(t, sink.Updates())
	require.NotNil(t, request.ConfirmPreflight)
	require.NoError(t, request.ConfirmPreflight(ops.PreflightScope{RequiresConfirmation: true}))
}

// TestOpsAnalyseSlowProcessInstancesKeysOnlyProgressKeepsStdoutClean verifies key pipelines never receive progress or preflight lines.
func TestOpsAnalyseSlowProcessInstancesKeysOnlyProgressKeepsStdoutClean(t *testing.T) {
	previousKeysOnly := flagViewKeysOnly
	flagViewKeysOnly = true
	t.Cleanup(func() { flagViewKeysOnly = previousKeysOnly })
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	sink := &activitysink.Sink{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetContext(logging.ToActivityContext(cmd.Context(), sink))
	request := ops.SlowProcessAnalysisRequest{SelectionMode: ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch}

	configureOpsSlowProcessAnalysisPreflight(cmd, &request)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPreflight, Preflight: &ops.PreflightScope{
		SelectorSummary: "OrderProcess",
		CoreResource:    "process_instance",
		TotalKind:       ops.TotalCertaintyUnknown,
		PageSize:        1000,
	}})
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindFrozenScope, FrozenScope: &ops.FrozenScopeProgress{
		Phase:        "loading runtime elements",
		CoreResource: "process instance(s)",
		Done:         2,
		Total:        2,
	}})

	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
	require.Empty(t, sink.Updates())
	require.NotNil(t, request.ConfirmPreflight)
	require.NoError(t, request.ConfirmPreflight(ops.PreflightScope{RequiresConfirmation: true}))
}

// TestOpsAnalyseSlowProcessInstancesQuietAndAutomationSuppressProgress verifies non-interactive contracts stay silent.
func TestOpsAnalyseSlowProcessInstancesQuietAndAutomationSuppressProgress(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*cobra.Command)
		want  ops.ProgressMode
	}{
		{
			name: "quiet",
			setup: func(*cobra.Command) {
				flagQuiet = true
			},
			want: ops.ProgressModeQuiet,
		},
		{
			name: "automation",
			setup: func(cmd *cobra.Command) {
				flagCmdAutomation = true
				cmd.Flags().Bool("automation", true, "")
			},
			want: ops.ProgressModeAutomation,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			previousQuiet := flagQuiet
			previousAutomation := flagCmdAutomation
			t.Cleanup(func() {
				flagQuiet = previousQuiet
				flagCmdAutomation = previousAutomation
			})
			cmd := resetOpsSlowProcessAnalysisTestFlags(t)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			sink := &activitysink.Sink{}
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetContext(logging.ToActivityContext(cmd.Context(), sink))
			tc.setup(cmd)
			channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
			require.Equal(t, tc.want, channel.Mode)
			request := ops.SlowProcessAnalysisRequest{SelectionMode: ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch}

			configureOpsSlowProcessAnalysisPreflight(cmd, &request)
			request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPreflight, Preflight: &ops.PreflightScope{
				SelectorSummary: "OrderProcess",
				CoreResource:    "process_instance",
				Total:           ptrInt64(10),
				TotalKind:       ops.TotalCertaintyExact,
			}})
			request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPage, Page: &ops.PageProgress{
				Phase:       "discovering process instances",
				CurrentPage: 1,
				Seen:        10,
			}})

			require.Empty(t, stdout.String())
			require.Empty(t, stderr.String())
			require.Empty(t, sink.Updates())
			require.NotNil(t, request.ConfirmPreflight)
			require.NoError(t, request.ConfirmPreflight(ops.PreflightScope{RequiresConfirmation: true}))
		})
	}
}

// TestOpsAnalyseSlowProcessInstancesPreflightConfirmationRespectsModeGating verifies automation-safe modes skip prompts.
func TestOpsAnalyseSlowProcessInstancesPreflightConfirmationRespectsModeGating(t *testing.T) {
	require.True(t, opsSlowProcessAnalysisPreflightConfirmationAllowed(ops.ProgressChannel{Mode: ops.ProgressModeHuman}))
	require.True(t, opsSlowProcessAnalysisPreflightConfirmationAllowed(ops.ProgressChannel{Mode: ops.ProgressModeVerbose}))
	require.False(t, opsSlowProcessAnalysisPreflightConfirmationAllowed(ops.ProgressChannel{Mode: ops.ProgressModeJSON}))
	require.False(t, opsSlowProcessAnalysisPreflightConfirmationAllowed(ops.ProgressChannel{Mode: ops.ProgressModeKeysOnly}))
	require.False(t, opsSlowProcessAnalysisPreflightConfirmationAllowed(ops.ProgressChannel{Mode: ops.ProgressModeQuiet}))
	require.False(t, opsSlowProcessAnalysisPreflightConfirmationAllowed(ops.ProgressChannel{Mode: ops.ProgressModeAutomation}))
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
	flagOpsAnalyseSlowProcessInstanceDurationLonger = ""
	flagOpsAnalyseSlowProcessInstanceElementDurationLonger = ""
	flagOpsAnalyseSlowProcessInstanceWithFullTimeline = false
	flagOpsAnalyseSlowProcessInstanceWithListeners = false

	cmd := &cobra.Command{Use: "slow-process-instances"}
	cmd.SetContext(context.Background())
	flags := cmd.Flags()
	flags.StringVar(&flagOpsAnalyseSlowProcessInstanceState, "state", "all", "")
	flags.Int32Var(&flagOpsAnalyseSlowProcessInstanceBatchSize, "batch-size", consts.MaxPISearchSize, "")
	flags.Int32Var(&flagOpsAnalyseSlowProcessInstanceLimit, "limit", 0, "")
	flags.BoolVar(&flagOpsAnalyseSlowProcessInstanceWithFullTimeline, "with-full-timeline", false, "")
	flags.BoolVar(&flagOpsAnalyseSlowProcessInstanceWithListeners, "with-listeners", false, "")
	return cmd
}

// ptrFloat64 returns a stable pointer for compact progress callback fixtures.
func ptrFloat64(value float64) *float64 {
	return &value
}

// ptrDuration returns a stable pointer for compact progress callback fixtures.
func ptrDuration(value time.Duration) *time.Duration {
	return &value
}
