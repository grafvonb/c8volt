// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

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

// TestOpsAnalyseSlowProcessInstancesSearchDiscoveryStaysTransientOnly documents
// why discovery pages are excluded from semantic completion aggregation.
func TestOpsAnalyseSlowProcessInstancesSearchDiscoveryStaysTransientOnly(t *testing.T) {
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
		CurrentPage:   3,
		PageCount:     ptrInt64(6),
		PageCountKind: ops.PageCountKindExact,
		Seen:          2400,
		Selected:      2397,
	}})

	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
	require.Equal(t, []activitysink.Update{{
		Message:    "discovering process instances, page 3/6, 2400 seen, 2397 selected",
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.PriorityUpdates())
	require.NotContains(t, sink.Updates()[0], "completed")
}

// TestOpsAnalyseSlowProcessInstancesDefaultProgressWritesPacedMilestones verifies broad human runs get sparse durable progress after confirmation.
func TestOpsAnalyseSlowProcessInstancesDefaultProgressWritesPacedMilestones(t *testing.T) {
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	sink := &activitysink.Sink{}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	pacer := newOpsProgressMilestonePacer(func() time.Time { return now })
	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		require.False(t, autoConfirm)
		require.Equal(t, "Continue slow analysis for 2000 process instances?", prompt)
		return nil
	}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetContext(logging.ToActivityContext(cmd.Context(), sink))
	request := ops.SlowProcessAnalysisRequest{SelectionMode: ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch}

	configureOpsSlowProcessAnalysisPreflightWithPacer(cmd, &request, pacer)
	require.NotNil(t, request.ConfirmPreflight)
	require.NoError(t, request.ConfirmPreflight(ops.PreflightScope{
		RequiresConfirmation: true,
		Total:                ptrInt64(2000),
		TotalKind:            ops.TotalCertaintyExact,
		ConsequenceSummary: ops.ConsequenceSummary{
			ConfirmationText: "Continue slow analysis for 2000 process instances?",
		},
	}))

	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPage, Page: &ops.PageProgress{
		Phase:         "discovering process instances",
		CurrentPage:   1,
		PageCount:     ptrInt64(4),
		PageCountKind: ops.PageCountKindExact,
		Seen:          1000,
		Selected:      1000,
	}})
	require.Empty(t, stderr.String())

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPage, Page: &ops.PageProgress{
		Phase:         "discovering process instances",
		CurrentPage:   2,
		PageCount:     ptrInt64(4),
		PageCountKind: ops.PageCountKindExact,
		Seen:          1500,
		Selected:      1498,
	}})
	require.Contains(t, stderr.String(), "discovering process instances, page 2/4, 1500 seen, 1498 selected")

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindFrozenScope, FrozenScope: &ops.FrozenScopeProgress{
		Phase:        "loading runtime elements",
		CoreResource: "process instance(s)",
		Done:         48,
		Total:        800,
	}})

	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "loading runtime elements, 48/800 process instance(s)")
	require.Equal(t, []string{
		"discovering process instances, page 1/4, 1000 seen",
		"discovering process instances, page 2/4, 1500 seen, 1498 selected",
		"loading runtime elements, 48/800 process instance(s)",
	}, sink.Updates())
}

// TestOpsAnalyseSlowProcessInstancesWorkflowActivityOutranksNestedRuntimeWork verifies slow-analysis progress keeps workflow priority.
func TestOpsAnalyseSlowProcessInstancesWorkflowActivityOutranksNestedRuntimeWork(t *testing.T) {
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	sink := &activitysink.Sink{}
	cmd.SetContext(logging.ToActivityContext(cmd.Context(), sink))
	request := ops.SlowProcessAnalysisRequest{SelectionMode: ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch}

	configureOpsSlowProcessAnalysisPreflight(cmd, &request)
	recordHTTPFallbackActivity(cmd.Context(), "loading runtime elements from Camunda")
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindFrozenScope, FrozenScope: &ops.FrozenScopeProgress{
		Phase:        "loading runtime elements",
		CoreResource: "process instance(s)",
		Done:         48,
		Total:        800,
	}})

	require.Equal(t, []activitysink.Update{
		{
			Message:    "loading runtime elements, 48/800 process instance(s)",
			Importance: logging.ActivityImportanceWorkflow,
		},
	}, sink.PriorityUpdates())
	require.Equal(t, []activitysink.Start{
		{
			Message:    "loading runtime elements from Camunda",
			Importance: logging.ActivityImportanceHTTP,
		},
	}, sink.Starts())
}

// TestOpsAnalyseSlowProcessInstancesVerboseProgressWritesDurableStderr verifies verbose mode keeps an auditable progress trail off stdout.
func TestOpsAnalyseSlowProcessInstancesVerboseProgressWritesDurableStderr(t *testing.T) {
	previousVerbose := flagVerbose
	t.Cleanup(func() { flagVerbose = previousVerbose })
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	flagVerbose = true
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

// TestOpsAnalyseSlowProcessInstancesDebugProgressWritesDurableStderr verifies debug mode keeps detailed durable progress off stdout.
func TestOpsAnalyseSlowProcessInstancesDebugProgressWritesDurableStderr(t *testing.T) {
	previousDebug := flagDebug
	t.Cleanup(func() { flagDebug = previousDebug })
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	flagDebug = true
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	request := ops.SlowProcessAnalysisRequest{SelectionMode: ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch}

	configureOpsSlowProcessAnalysisPreflight(cmd, &request)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPage, Page: &ops.PageProgress{
		Phase:         "discovering process instances",
		CurrentPage:   3,
		PageCount:     ptrInt64(7),
		PageCountKind: ops.PageCountKindEstimated,
		Seen:          3000,
		Selected:      2750,
	}})
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindFrozenScope, FrozenScope: &ops.FrozenScopeProgress{
		Phase:        "loading runtime elements",
		CoreResource: "process instance(s)",
		Done:         72,
		Total:        2750,
		Elapsed:      2 * time.Minute,
	}})

	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "discovering process instances, page 3/~7, 3000 seen, 2750 selected")
	require.Contains(t, stderr.String(), "loading runtime elements, 72/2750 process instance(s), 2.6%, 2m0s elapsed")
}

// TestOpsAnalyseSlowProcessInstancesDefaultMilestonesStayCompact verifies default human milestones avoid diagnostic request detail.
func TestOpsAnalyseSlowProcessInstancesDefaultMilestonesStayCompact(t *testing.T) {
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	pacer := newOpsProgressMilestonePacer(func() time.Time { return now })
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	request := ops.SlowProcessAnalysisRequest{SelectionMode: ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch}

	configureOpsSlowProcessAnalysisPreflightWithPacer(cmd, &request, pacer)
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPage, Page: &ops.PageProgress{
		Phase:         "discovering process instances",
		CurrentPage:   4,
		PageCount:     ptrInt64(10),
		PageCountKind: ops.PageCountKindEstimated,
		Seen:          3812,
		Selected:      3800,
	}})
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindFrozenScope, FrozenScope: &ops.FrozenScopeProgress{
		Phase:        "loading runtime elements",
		CoreResource: "process instance(s)",
		Done:         96,
		Total:        3800,
		Elapsed:      3*time.Minute + 5*time.Second,
	}})

	got := stderr.String()
	require.Empty(t, stdout.String())
	require.Contains(t, got, "discovering process instances, page 4/~10, 3812 seen, 3800 selected")
	require.Contains(t, got, "loading runtime elements, 96/3800 process instance(s), 2.5%, 3m5s elapsed")
	for _, diagnostic := range []string{"endpoint", "request", "cursor", "camunda", "/v2", "process instance key", "runtime element key", "element instance key"} {
		require.NotContains(t, strings.ToLower(got), diagnostic)
	}
}

// TestOpsAnalyseSlowProcessInstancesJSONProgressKeepsStdoutClean verifies JSON mode suppresses transient and durable progress text.
func TestOpsAnalyseSlowProcessInstancesJSONProgressKeepsStdoutClean(t *testing.T) {
	previousJSON := flagViewAsJson
	t.Cleanup(func() { flagViewAsJson = previousJSON })
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	flagViewAsJson = true
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	sink := &activitysink.Sink{}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	pacer := newOpsProgressMilestonePacer(func() time.Time { return now })
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetContext(logging.ToActivityContext(cmd.Context(), sink))
	request := ops.SlowProcessAnalysisRequest{SelectionMode: ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch}

	configureOpsSlowProcessAnalysisPreflightWithPacer(cmd, &request, pacer)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPreflight, Preflight: &ops.PreflightScope{
		SelectorSummary: "OrderProcess",
		CoreResource:    "process_instance",
		Total:           ptrInt64(2000),
		TotalKind:       ops.TotalCertaintyExact,
		PageSize:        1000,
		PageCount:       ptrInt64(2),
		PageCountKind:   ops.PageCountKindExact,
	}})
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPage, Page: &ops.PageProgress{
		Phase:         "discovering process instances",
		CurrentPage:   2,
		PageCount:     ptrInt64(2),
		PageCountKind: ops.PageCountKindExact,
		Seen:          2000,
		Selected:      1998,
	}})
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindFrozenScope, FrozenScope: &ops.FrozenScopeProgress{
		Phase:        "loading runtime elements",
		CoreResource: "process instance(s)",
		Done:         48,
		Total:        2000,
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
	t.Cleanup(func() { flagViewKeysOnly = previousKeysOnly })
	cmd := resetOpsSlowProcessAnalysisTestFlags(t)
	flagViewKeysOnly = true
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	sink := &activitysink.Sink{}
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	pacer := newOpsProgressMilestonePacer(func() time.Time { return now })
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetContext(logging.ToActivityContext(cmd.Context(), sink))
	request := ops.SlowProcessAnalysisRequest{SelectionMode: ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch}

	configureOpsSlowProcessAnalysisPreflightWithPacer(cmd, &request, pacer)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPreflight, Preflight: &ops.PreflightScope{
		SelectorSummary: "OrderProcess",
		CoreResource:    "process_instance",
		TotalKind:       ops.TotalCertaintyUnknown,
		PageSize:        1000,
	}})
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPage, Page: &ops.PageProgress{
		Phase:       "discovering process instances",
		CurrentPage: 3,
		Seen:        3000,
		Selected:    2500,
	}})
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindFrozenScope, FrozenScope: &ops.FrozenScopeProgress{
		Phase:        "loading runtime elements",
		CoreResource: "process instance(s)",
		Done:         64,
		Total:        2500,
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
			now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
			pacer := newOpsProgressMilestonePacer(func() time.Time { return now })
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetContext(logging.ToActivityContext(cmd.Context(), sink))
			tc.setup(cmd)
			channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
			require.Equal(t, tc.want, channel.Mode)
			request := ops.SlowProcessAnalysisRequest{SelectionMode: ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch}

			configureOpsSlowProcessAnalysisPreflightWithPacer(cmd, &request, pacer)
			request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPreflight, Preflight: &ops.PreflightScope{
				SelectorSummary: "OrderProcess",
				CoreResource:    "process_instance",
				Total:           ptrInt64(10),
				TotalKind:       ops.TotalCertaintyExact,
			}})
			now = now.Add(opsDurableMilestoneMinimumElapsed)
			request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindPage, Page: &ops.PageProgress{
				Phase:       "discovering process instances",
				CurrentPage: 2,
				Seen:        20,
				Selected:    18,
			}})
			now = now.Add(opsDurableMilestoneMinimumElapsed)
			request.Progress(ops.ProgressEvent{Kind: ops.ProgressEventKindFrozenScope, FrozenScope: &ops.FrozenScopeProgress{
				Phase:        "loading runtime elements",
				CoreResource: "process instance(s)",
				Done:         18,
				Total:        20,
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

// ptrFloat64 returns a stable pointer for compact progress callback fixtures.
func ptrFloat64(value float64) *float64 {
	return &value
}

// ptrDuration returns a stable pointer for compact progress callback fixtures.
func ptrDuration(value time.Duration) *time.Duration {
	return &value
}
