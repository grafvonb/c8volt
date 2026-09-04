// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestOpsPurgeAllProcessDefinitionsProgressStagesUseDistinctCounters verifies
// force-cleanup activity enters each actual stage with its own resource unit.
func TestOpsPurgeAllProcessDefinitionsProgressStagesUseDistinctCounters(t *testing.T) {
	progress, sink, _ := newOpsPurgeAllProcessDefinitionsProgressTest(t, ops.ProgressModeHuman)
	progress.Start()
	require.Equal(t, []activitysink.Start{{
		Message:    opsPurgeAllProcessDefinitionsGenericActivity,
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.Starts())

	totalRoots := 2
	plannedAffected := 7
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{
		Phase:                opsPurgeAllProcessDefinitionsCancelPhase,
		CoreResource:         "process-instance tree(s)",
		Total:                &totalRoots,
		PlannedAffectedCount: &plannedAffected,
	}})
	require.Equal(t, "cancelling process-instance root trees, 0/2 process-instance tree(s), affected scope: 7 process instance(s)", lastOpsPurgeAllProcessDefinitionsProgressUpdate(t, sink))

	affected := 3
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{
		Phase:         opsPurgeAllProcessDefinitionsCancelPhase,
		Identity:      "root-1",
		Disposition:   ops.CompletionDispositionConfirmed,
		AffectedCount: &affected,
	}})
	require.Equal(t, "cancelling process-instance root trees, 1/2 process-instance tree(s), affected process instances: 3, affected scope: 7 process instance(s)", lastOpsPurgeAllProcessDefinitionsProgressUpdate(t, sink))

	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: opsPurgeAllProcessDefinitionsDrainPhase}})
	require.Equal(t, "waiting for active process instances to drain", lastOpsPurgeAllProcessDefinitionsProgressUpdate(t, sink))

	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: opsPurgeAllProcessDefinitionsHistoryDeletePhase, Total: &totalRoots}})
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{
		Phase:       opsPurgeAllProcessDefinitionsHistoryDeletePhase,
		Identity:    "root-1",
		Disposition: ops.CompletionDispositionConfirmed,
	}})
	require.Equal(t, "deleting process-instance histories, 1/2 process-instance tree(s)", lastOpsPurgeAllProcessDefinitionsProgressUpdate(t, sink))

	totalDefinitions := 3
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: processDefinitionDeleteCompletionPhase, Total: &totalDefinitions}})
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{
		Phase:       processDefinitionDeleteCompletionPhase,
		Identity:    "pd-1",
		Disposition: ops.CompletionDispositionConfirmed,
	}})
	require.Equal(t, "deleting process definitions, 1/3 process definition(s)", lastOpsPurgeAllProcessDefinitionsProgressUpdate(t, sink))

	cancelAggregate, ok := progress.Aggregate(opsPurgeAllProcessDefinitionsCancelPhase)
	require.True(t, ok)
	require.Equal(t, opsSemanticProgressAggregate{Completed: 1, Total: 2, Affected: 3, AffectedValid: true}, cancelAggregate)
	historyAggregate, ok := progress.Aggregate(opsPurgeAllProcessDefinitionsHistoryDeletePhase)
	require.True(t, ok)
	require.Equal(t, opsSemanticProgressAggregate{Completed: 1, Total: 2, AffectedValid: false}, historyAggregate)
	definitionAggregate, ok := progress.Aggregate(processDefinitionDeleteCompletionPhase)
	require.True(t, ok)
	require.Equal(t, opsSemanticProgressAggregate{Completed: 1, Total: 3}, definitionAggregate)
}

// TestOpsPurgeAllProcessDefinitionsProgressRejectsUnenteredAndUnrelatedFacts
// verifies exact phase routing does not invent stages from stray callbacks.
func TestOpsPurgeAllProcessDefinitionsProgressRejectsUnenteredAndUnrelatedFacts(t *testing.T) {
	progress, sink, _ := newOpsPurgeAllProcessDefinitionsProgressTest(t, ops.ProgressModeHuman)
	progress.Start()

	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: ""}})
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: "download"}})
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{
		Phase:       opsPurgeAllProcessDefinitionsCancelPhase,
		Identity:    "root-1",
		Disposition: ops.CompletionDispositionConfirmed,
	}})
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{
		Phase:       "",
		Identity:    "pd-1",
		Disposition: ops.CompletionDispositionConfirmed,
	}})

	require.Empty(t, sink.PriorityUpdates())
	_, ok := progress.Aggregate(opsPurgeAllProcessDefinitionsCancelPhase)
	require.False(t, ok)

	totalDefinitions := 1
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: processDefinitionDeleteCompletionPhase, Total: &totalDefinitions}})
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{
		Phase:       opsPurgeAllProcessDefinitionsCancelPhase,
		Identity:    "root-2",
		Disposition: ops.CompletionDispositionConfirmed,
	}})

	require.Equal(t, []activitysink.Update{{
		Message:    "deleting process definitions, 0/1 process definition(s)",
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.PriorityUpdates())
}

// TestOpsPurgeAllProcessDefinitionsProgressKeepsPlannedAffectedSeparate
// verifies nil and known-zero affected facts are rendered without inference.
func TestOpsPurgeAllProcessDefinitionsProgressKeepsPlannedAffectedSeparate(t *testing.T) {
	progress, sink, _ := newOpsPurgeAllProcessDefinitionsProgressTest(t, ops.ProgressModeHuman)
	totalRoots := 2

	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{
		Phase: opsPurgeAllProcessDefinitionsCancelPhase,
		Total: &totalRoots,
	}})
	require.Equal(t, "cancelling process-instance root trees, 0/2 process-instance tree(s)", lastOpsPurgeAllProcessDefinitionsProgressUpdate(t, sink))

	affected := 0
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{
		Phase:         opsPurgeAllProcessDefinitionsCancelPhase,
		Identity:      "root-1",
		Disposition:   ops.CompletionDispositionConfirmed,
		AffectedCount: &affected,
	}})
	require.Equal(t, "cancelling process-instance root trees, 1/2 process-instance tree(s), affected process instances: 0", lastOpsPurgeAllProcessDefinitionsProgressUpdate(t, sink))

	plannedAffected := 0
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{
		Phase:                opsPurgeAllProcessDefinitionsHistoryDeletePhase,
		Total:                &totalRoots,
		PlannedAffectedCount: &plannedAffected,
	}})
	require.Equal(t, "deleting process-instance histories, 0/2 process-instance tree(s), affected scope: 0 process instance(s)", lastOpsPurgeAllProcessDefinitionsProgressUpdate(t, sink))
}

// TestOpsPurgeAllProcessDefinitionsProgressHistoricalCompletionDoesNotRollback
// verifies late completions can update old aggregates without repainting the
// current workflow activity.
func TestOpsPurgeAllProcessDefinitionsProgressHistoricalCompletionDoesNotRollback(t *testing.T) {
	progress, sink, _ := newOpsPurgeAllProcessDefinitionsProgressTest(t, ops.ProgressModeHuman)
	totalRoots := 2
	totalDefinitions := 1

	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: opsPurgeAllProcessDefinitionsCancelPhase, Total: &totalRoots}})
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: opsPurgeAllProcessDefinitionsCancelPhase, Identity: "root-1", Disposition: ops.CompletionDispositionConfirmed}})
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: processDefinitionDeleteCompletionPhase, Total: &totalDefinitions}})
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: opsPurgeAllProcessDefinitionsCancelPhase, Identity: "root-2", Disposition: ops.CompletionDispositionConfirmed}})

	require.Equal(t, "deleting process definitions, 0/1 process definition(s)", lastOpsPurgeAllProcessDefinitionsProgressUpdate(t, sink))
	cancelAggregate, ok := progress.Aggregate(opsPurgeAllProcessDefinitionsCancelPhase)
	require.True(t, ok)
	require.Equal(t, 2, cancelAggregate.Completed)
}

// TestOpsPurgeAllProcessDefinitionsProgressSerializesConcurrentCompletions
// verifies out-of-order nested callbacks keep exact stage-local counters.
func TestOpsPurgeAllProcessDefinitionsProgressSerializesConcurrentCompletions(t *testing.T) {
	progress, sink, _ := newOpsPurgeAllProcessDefinitionsProgressTest(t, ops.ProgressModeHuman)
	progress.Start()
	progress.Start()
	totalRoots := 64
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: opsPurgeAllProcessDefinitionsCancelPhase, Total: &totalRoots}})

	var wg sync.WaitGroup
	for i := range totalRoots {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{
				Phase:       opsPurgeAllProcessDefinitionsCancelPhase,
				Identity:    "root-" + strconv.Itoa(i),
				Disposition: ops.CompletionDispositionConfirmed,
			}})
		}(i)
	}
	wg.Wait()

	aggregate, ok := progress.Aggregate(opsPurgeAllProcessDefinitionsCancelPhase)
	require.True(t, ok)
	require.Equal(t, opsSemanticProgressAggregate{Completed: 64, Total: 64, AffectedValid: false}, aggregate)
	require.Equal(t, 1, sink.Started())
	requireOpsPurgeAllProcessDefinitionsProgressCompletedSequence(t, sink.Updates(), totalRoots)

	progress.Close()
	progress.Close()
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: opsPurgeAllProcessDefinitionsCancelPhase, Identity: "root-ignored", Disposition: ops.CompletionDispositionConfirmed}})
	require.Equal(t, 1, sink.Stopped())
	aggregate, ok = progress.Aggregate(opsPurgeAllProcessDefinitionsCancelPhase)
	require.True(t, ok)
	require.Equal(t, 64, aggregate.Completed)
}

// TestOpsPurgeAllProcessDefinitionsProgressPreservesDefinitionMilestones
// verifies the new coordinator keeps the old single-stage deletion cadence.
func TestOpsPurgeAllProcessDefinitionsProgressPreservesDefinitionMilestones(t *testing.T) {
	now := time.Date(2026, time.September, 4, 12, 30, 0, 0, time.UTC)
	progress, _, stderr := newOpsPurgeAllProcessDefinitionsProgressTestWithClock(t, ops.ProgressModeHuman, func() time.Time { return now })
	totalDefinitions := 2

	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: processDefinitionDeleteCompletionPhase, Total: &totalDefinitions}})
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{
		Phase:       processDefinitionDeleteCompletionPhase,
		Identity:    "pd-1",
		Disposition: ops.CompletionDispositionConfirmed,
	}})

	require.Equal(t, "deleting process definitions, 1/2 process definition(s)\n", stderr.String())
}

// TestOpsPurgeAllProcessDefinitionsProgressPacesFromFirstStage verifies
// generic workflow setup and stage transitions do not reset the durable clock.
func TestOpsPurgeAllProcessDefinitionsProgressPacesFromFirstStage(t *testing.T) {
	now := time.Date(2026, time.September, 4, 12, 45, 0, 0, time.UTC)
	progress, _, stderr := newOpsPurgeAllProcessDefinitionsProgressTestWithClock(t, ops.ProgressModeHuman, func() time.Time { return now })
	totalRoots := 2

	progress.Start()
	now = now.Add(time.Minute)
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: opsPurgeAllProcessDefinitionsCancelPhase, Total: &totalRoots}})
	now = now.Add(opsDurableMilestoneMinimumElapsed - time.Nanosecond)
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: opsPurgeAllProcessDefinitionsCancelPhase, Identity: "root-1", Disposition: ops.CompletionDispositionConfirmed}})
	require.Empty(t, stderr.String())

	now = now.Add(time.Nanosecond)
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: opsPurgeAllProcessDefinitionsCancelPhase, Identity: "root-2", Disposition: ops.CompletionDispositionConfirmed}})
	require.Equal(t, "cancelling process-instance root trees, 2/2 process-instance tree(s)\n", stderr.String())

	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: opsPurgeAllProcessDefinitionsDrainPhase}})
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	require.Equal(t, 1, strings.Count(stderr.String(), "\n"))

	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: opsPurgeAllProcessDefinitionsHistoryDeletePhase, Total: &totalRoots}})
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: opsPurgeAllProcessDefinitionsHistoryDeletePhase, Identity: "root-1", Disposition: ops.CompletionDispositionConfirmed}})
	require.Equal(t, strings.Join([]string{
		"cancelling process-instance root trees, 2/2 process-instance tree(s)",
		"deleting process-instance histories, 1/2 process-instance tree(s)",
		"",
	}, "\n"), stderr.String())
}

// TestOpsPurgeAllProcessDefinitionsProgressWarningDoesNotResetClock verifies
// immediate failure evidence leaves the workflow-wide informational cadence intact.
func TestOpsPurgeAllProcessDefinitionsProgressWarningDoesNotResetClock(t *testing.T) {
	now := time.Date(2026, time.September, 4, 12, 50, 0, 0, time.UTC)
	progress, _, stderr := newOpsPurgeAllProcessDefinitionsProgressTestWithClock(t, ops.ProgressModeHuman, func() time.Time { return now })
	totalRoots := 3

	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: opsPurgeAllProcessDefinitionsCancelPhase, Total: &totalRoots}})
	now = now.Add(4 * time.Second)
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{
		Phase:         opsPurgeAllProcessDefinitionsCancelPhase,
		Identity:      "root-1",
		Disposition:   ops.CompletionDispositionFailed,
		FailureDetail: "boom",
	}})
	require.Contains(t, stderr.String(), "root-1 failed: boom")

	now = now.Add(6 * time.Second)
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: opsPurgeAllProcessDefinitionsCancelPhase, Identity: "root-2", Disposition: ops.CompletionDispositionConfirmed}})

	require.Contains(t, stderr.String(), "cancelling process-instance root trees, 2/3 process-instance tree(s), 1 failed")
	require.Equal(t, 1, strings.Count(stderr.String(), "root-1 failed"))
}

// TestOpsPurgeAllProcessDefinitionsProgressCloseFlushesDirtyStagesOnce verifies
// activated default progress retains one historical record across mutation stages.
func TestOpsPurgeAllProcessDefinitionsProgressCloseFlushesDirtyStagesOnce(t *testing.T) {
	now := time.Date(2026, time.September, 4, 12, 55, 0, 0, time.UTC)
	progress, sink, stderr := newOpsPurgeAllProcessDefinitionsProgressTestWithClock(t, ops.ProgressModeHuman, func() time.Time { return now })
	totalRoots := 2
	totalDefinitions := 3

	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: opsPurgeAllProcessDefinitionsCancelPhase, Total: &totalRoots}})
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: opsPurgeAllProcessDefinitionsCancelPhase, Identity: "root-1", Disposition: ops.CompletionDispositionConfirmed}})
	now = now.Add(time.Second)
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: opsPurgeAllProcessDefinitionsCancelPhase, Identity: "root-2", Disposition: ops.CompletionDispositionConfirmed}})
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: opsPurgeAllProcessDefinitionsHistoryDeletePhase, Total: &totalRoots}})
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: opsPurgeAllProcessDefinitionsHistoryDeletePhase, Identity: "root-1", Disposition: ops.CompletionDispositionConfirmed}})
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: processDefinitionDeleteCompletionPhase, Total: &totalDefinitions}})
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: processDefinitionDeleteCompletionPhase, Identity: "pd-1", Disposition: ops.CompletionDispositionConfirmed}})

	progress.Close()
	progress.Close()
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: processDefinitionDeleteCompletionPhase, Identity: "pd-2", Disposition: ops.CompletionDispositionConfirmed}})

	require.Equal(t, strings.Join([]string{
		"cancelling process-instance root trees, 1/2 process-instance tree(s)",
		"stage progress: cancelling process-instance root trees, 2/2 process-instance tree(s); deleting process-instance histories, 1/2 process-instance tree(s); deleting process definitions, 1/3 process definition(s)",
		"",
	}, "\n"), stderr.String())
	require.Equal(t, 1, sink.Stopped())
}

// TestOpsPurgeAllProcessDefinitionsProgressCloseKeepsSingleStageFormat
// verifies a one-stage close flush retains the ordinary aggregate wording.
func TestOpsPurgeAllProcessDefinitionsProgressCloseKeepsSingleStageFormat(t *testing.T) {
	now := time.Date(2026, time.September, 4, 12, 58, 0, 0, time.UTC)
	progress, _, stderr := newOpsPurgeAllProcessDefinitionsProgressTestWithClock(t, ops.ProgressModeHuman, func() time.Time { return now })
	totalDefinitions := 3

	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: processDefinitionDeleteCompletionPhase, Total: &totalDefinitions}})
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: processDefinitionDeleteCompletionPhase, Identity: "pd-1", Disposition: ops.CompletionDispositionConfirmed}})
	now = now.Add(time.Second)
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: processDefinitionDeleteCompletionPhase, Identity: "pd-2", Disposition: ops.CompletionDispositionConfirmed}})

	progress.Close()

	require.Equal(t, strings.Join([]string{
		"deleting process definitions, 1/3 process definition(s)",
		"deleting process definitions, 2/3 process definition(s)",
		"",
	}, "\n"), stderr.String())
}

// TestOpsPurgeAllProcessDefinitionsProgressCleanShortRunStaysSilent verifies
// close does not invent durable evidence for unactivated fast workflows.
func TestOpsPurgeAllProcessDefinitionsProgressCleanShortRunStaysSilent(t *testing.T) {
	now := time.Date(2026, time.September, 4, 13, 0, 0, 0, time.UTC)
	progress, sink, stderr := newOpsPurgeAllProcessDefinitionsProgressTestWithClock(t, ops.ProgressModeHuman, func() time.Time { return now })
	totalDefinitions := 1

	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: processDefinitionDeleteCompletionPhase, Total: &totalDefinitions}})
	now = now.Add(opsDurableMilestoneMinimumElapsed - time.Nanosecond)
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: processDefinitionDeleteCompletionPhase, Identity: "pd-1", Disposition: ops.CompletionDispositionConfirmed}})
	progress.Close()
	progress.Close()

	require.Empty(t, stderr.String())
	require.Equal(t, 1, sink.Stopped())
}

// TestOpsPurgeAllProcessDefinitionsProgressVerboseItemsSuppressMilestones
// verifies verbose stage completions replace paced aggregate milestones.
func TestOpsPurgeAllProcessDefinitionsProgressVerboseItemsSuppressMilestones(t *testing.T) {
	now := time.Date(2026, time.September, 4, 13, 5, 0, 0, time.UTC)
	progress, _, stderr := newOpsPurgeAllProcessDefinitionsProgressTestWithClock(t, ops.ProgressModeVerbose, func() time.Time { return now })
	totalRoots := 2

	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindStage, Stage: &ops.StageProgress{Phase: opsPurgeAllProcessDefinitionsHistoryDeletePhase, Total: &totalRoots}})
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: opsPurgeAllProcessDefinitionsHistoryDeletePhase, Identity: "root-1", Disposition: ops.CompletionDispositionSubmitted}})
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	progress.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: opsPurgeAllProcessDefinitionsHistoryDeletePhase, Identity: "root-2", Disposition: ops.CompletionDispositionConfirmed}})
	progress.Close()

	require.Equal(t, strings.Join([]string{
		"root-1 submitted (deleting process-instance histories, 1/2 process-instance tree(s))",
		"root-2 deleted (deleting process-instance histories, 2/2 process-instance tree(s))",
		"",
	}, "\n"), stderr.String())
}

// newOpsPurgeAllProcessDefinitionsProgressTest returns a coordinator fixture
// with activity and stderr capture using command progress plumbing.
func newOpsPurgeAllProcessDefinitionsProgressTest(t *testing.T, mode ops.ProgressMode) (*opsPurgeAllProcessDefinitionsProgress, *activitysink.Sink, *bytes.Buffer) {
	t.Helper()
	return newOpsPurgeAllProcessDefinitionsProgressTestWithClock(t, mode, time.Now)
}

// newOpsPurgeAllProcessDefinitionsProgressTestWithClock injects deterministic
// time for coordinator pacing tests.
func newOpsPurgeAllProcessDefinitionsProgressTestWithClock(t *testing.T, mode ops.ProgressMode, now func() time.Time) (*opsPurgeAllProcessDefinitionsProgress, *activitysink.Sink, *bytes.Buffer) {
	t.Helper()
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	sink := &activitysink.Sink{}
	var stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetErr(&stderr)
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))
	channel := ops.ProgressChannel{Mode: mode}
	switch mode {
	case ops.ProgressModeHuman, ops.ProgressModeVerbose, ops.ProgressModeDebug:
		channel.TransientAllowed = true
		channel.DurableAllowed = true
		channel.StderrAllowed = true
	}
	return newOpsPurgeAllProcessDefinitionsProgress(cmd, opsPurgeAllProcessDefinitionsProgressConfig{
		Policy: opsSemanticProgressOutputPolicyForChannel(channel),
		Now:    now,
	}), sink, &stderr
}

// lastOpsPurgeAllProcessDefinitionsProgressUpdate returns the latest captured
// activity update and fails the test if no activity was updated.
func lastOpsPurgeAllProcessDefinitionsProgressUpdate(t *testing.T, sink *activitysink.Sink) string {
	t.Helper()
	updates := sink.PriorityUpdates()
	require.NotEmpty(t, updates)
	return updates[len(updates)-1].Message
}

// requireOpsPurgeAllProcessDefinitionsProgressCompletedSequence asserts that
// concurrent completions advanced each exact counter once.
func requireOpsPurgeAllProcessDefinitionsProgressCompletedSequence(t *testing.T, updates []string, total int) {
	t.Helper()
	completedRe := regexp.MustCompile(`\b(\d+)/` + regexp.QuoteMeta(strconv.Itoa(total)) + `\b`)
	seen := make(map[int]bool, total)
	for _, update := range updates {
		if !strings.Contains(update, "cancelling process-instance root trees") {
			continue
		}
		match := completedRe.FindStringSubmatch(update)
		if len(match) != 2 {
			continue
		}
		completed, err := strconv.Atoi(match[1])
		require.NoError(t, err)
		if completed == 0 {
			continue
		}
		require.GreaterOrEqual(t, completed, 1)
		require.LessOrEqual(t, completed, total)
		require.False(t, seen[completed], "duplicate completed count %d in updates %v", completed, updates)
		seen[completed] = true
	}
	for completed := 1; completed <= total; completed++ {
		require.Truef(t, seen[completed], "missing completed count %d", completed)
	}
}
