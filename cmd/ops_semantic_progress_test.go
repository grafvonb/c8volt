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

// TestNewOpsSemanticProgressReporterStartsWorkflowActivity verifies semantic
// progress owns exactly one workflow-priority activity scope.
func TestNewOpsSemanticProgressReporterStartsWorkflowActivity(t *testing.T) {
	cmd, sink, _ := newOpsSemanticProgressTestCommand(t)
	reporter := newOpsSemanticProgressReporter(cmd, opsSemanticProgressConfig{
		Scope: opsSemanticProgressScope{
			ActivityLabel: "deleting process-instance trees",
			CoreResource:  "process-instance tree(s)",
			Total:         2,
		},
		Policy: opsSemanticProgressOutputPolicyForChannel(ops.ProgressChannel{Mode: ops.ProgressModeHuman, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}),
		Now:    func() time.Time { return time.Date(2026, time.August, 31, 17, 0, 0, 0, time.UTC) },
	})

	require.Equal(t, []activitysink.Start{{
		Message:    "deleting process-instance trees, 0/2 process-instance tree(s)",
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.Starts())

	reporter.Close()
	require.Equal(t, 1, sink.Stopped())
}

// TestOpsSemanticProgressOutputPolicyForChannel verifies progress detail is
// available only to stdout-safe human modes and quiet keeps failure warnings.
func TestOpsSemanticProgressOutputPolicyForChannel(t *testing.T) {
	tests := []struct {
		name             string
		channel          ops.ProgressChannel
		wantTransient    bool
		wantPaced        bool
		wantVerboseItems bool
		wantFailures     bool
	}{
		{
			name:             "default human",
			channel:          ops.ProgressChannel{Mode: ops.ProgressModeHuman, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true},
			wantTransient:    true,
			wantPaced:        true,
			wantVerboseItems: false,
			wantFailures:     true,
		},
		{
			name:             "verbose",
			channel:          ops.ProgressChannel{Mode: ops.ProgressModeVerbose, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true},
			wantTransient:    true,
			wantPaced:        false,
			wantVerboseItems: true,
			wantFailures:     true,
		},
		{
			name:         "quiet",
			channel:      ops.ProgressChannel{Mode: ops.ProgressModeQuiet},
			wantFailures: true,
		},
		{name: "json", channel: ops.ProgressChannel{Mode: ops.ProgressModeJSON}},
		{name: "keys", channel: ops.ProgressChannel{Mode: ops.ProgressModeKeysOnly}},
		{name: "automation", channel: ops.ProgressChannel{Mode: ops.ProgressModeAutomation, StructuredReportAllowed: true}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := opsSemanticProgressOutputPolicyForChannel(tc.channel)
			require.Equal(t, tc.wantTransient, got.TransientActivity)
			require.Equal(t, tc.wantPaced, got.PacedAggregate)
			require.Equal(t, tc.wantVerboseItems, got.VerboseItems)
			require.Equal(t, tc.wantFailures, got.FailureWarnings)
			require.False(t, got.Stdout)
		})
	}
}

// TestOpsSemanticProgressReporterAggregatesConcurrentCompletions verifies
// out-of-order worker callbacks cannot make aggregate counters drift.
func TestOpsSemanticProgressReporterAggregatesConcurrentCompletions(t *testing.T) {
	cmd, sink, _ := newOpsSemanticProgressTestCommand(t)
	reporter := newOpsSemanticProgressReporter(cmd, opsSemanticProgressConfig{
		Scope: opsSemanticProgressScope{
			ActivityLabel:             "deleting process-instance trees",
			CoreResource:              "process-instance tree(s)",
			Total:                     64,
			AffectedResource:          "affected process instances",
			AffectedCoverageAvailable: true,
		},
		Policy: opsSemanticProgressOutputPolicyForChannel(ops.ProgressChannel{Mode: ops.ProgressModeHuman, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}),
		Now:    func() time.Time { return time.Date(2026, time.August, 31, 17, 0, 0, 0, time.UTC) },
	})

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			affected := 2
			disposition := ops.CompletionDispositionConfirmed
			if i%7 == 0 {
				disposition = ops.CompletionDispositionFailed
			}
			reporter.Report(ops.ProgressEvent{
				Kind: ops.ProgressEventKindCompletion,
				Completion: &ops.CompletionProgress{
					Phase:         "mutation",
					Identity:      "pi-" + strconv.Itoa(i),
					Disposition:   disposition,
					AffectedCount: &affected,
				},
			})
		}()
	}
	wg.Wait()

	updates := sink.Updates()
	require.Len(t, updates, 64)
	requireOpsSemanticProgressCompletedSequence(t, updates, 64)
	require.Equal(t, opsSemanticProgressAggregate{
		Completed:     64,
		Failed:        10,
		Total:         64,
		Affected:      128,
		AffectedValid: true,
	}, reporter.Aggregate())
	require.Contains(t, strings.Join(updates, "\n"), "64/64 process-instance tree(s)")
	require.Contains(t, strings.Join(updates, "\n"), "10 failed")
	require.Contains(t, strings.Join(updates, "\n"), "affected process instances: 128")
}

// TestOpsSemanticProgressReporterInvalidatesAffectedCoverage verifies one
// unavailable affected count suppresses the aggregate for the whole scope.
func TestOpsSemanticProgressReporterInvalidatesAffectedCoverage(t *testing.T) {
	cmd, sink, _ := newOpsSemanticProgressTestCommand(t)
	reporter := newOpsSemanticProgressReporter(cmd, opsSemanticProgressConfig{
		Scope: opsSemanticProgressScope{
			ActivityLabel:             "deleting process definitions",
			CoreResource:              "process definition(s)",
			Total:                     3,
			AffectedResource:          "affected process instances",
			AffectedCoverageAvailable: true,
		},
		Policy: opsSemanticProgressOutputPolicyForChannel(ops.ProgressChannel{Mode: ops.ProgressModeHuman, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}),
	})
	affected := 0

	reporter.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Identity: "pd-1", Disposition: ops.CompletionDispositionConfirmed, AffectedCount: &affected}})
	reporter.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Identity: "pd-2", Disposition: ops.CompletionDispositionConfirmed}})
	affected = 9
	reporter.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Identity: "pd-3", Disposition: ops.CompletionDispositionConfirmed, AffectedCount: &affected}})

	got := reporter.Aggregate()
	require.False(t, got.AffectedValid)
	require.Equal(t, 0, got.Affected)
	updates := sink.Updates()
	require.NotEmpty(t, updates)
	require.NotContains(t, updates[len(updates)-1], "affected process instances")
}

// TestOpsSemanticProgressReporterIgnoresUnrelatedCompletionPhase verifies a
// shared callback cannot advance a workflow aggregate with another phase's fact.
func TestOpsSemanticProgressReporterIgnoresUnrelatedCompletionPhase(t *testing.T) {
	cmd, sink, _ := newOpsSemanticProgressTestCommand(t)
	reporter := newOpsSemanticProgressReporter(cmd, opsSemanticProgressConfig{
		Scope: opsSemanticProgressScope{
			Phase:                     "delete",
			ActivityLabel:             "deleting process-instance trees",
			CoreResource:              "process-instance tree(s)",
			Total:                     2,
			AffectedResource:          "affected process instances",
			AffectedCoverageAvailable: true,
		},
		Policy: opsSemanticProgressOutputPolicyForChannel(ops.ProgressChannel{Mode: ops.ProgressModeHuman, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}),
	})
	affected := 3

	reporter.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: "cancel", Identity: "root-1", Disposition: ops.CompletionDispositionConfirmed, AffectedCount: &affected}})

	require.Empty(t, sink.Updates())
	require.Equal(t, opsSemanticProgressAggregate{
		Total:         2,
		AffectedValid: true,
	}, reporter.Aggregate())

	reporter.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Phase: "delete", Identity: "root-2", Disposition: ops.CompletionDispositionConfirmed, AffectedCount: &affected}})

	require.Equal(t, []string{"deleting process-instance trees, 1/2 process-instance tree(s), affected process instances: 3"}, sink.Updates())
	require.Equal(t, opsSemanticProgressAggregate{
		Completed:     1,
		Total:         2,
		Affected:      3,
		AffectedValid: true,
	}, reporter.Aggregate())
}

// TestOpsSemanticProgressReporterKeepsCleanSubTenSecondRunsDurablySilent
// verifies fast successful scopes update only transient activity.
func TestOpsSemanticProgressReporterKeepsCleanSubTenSecondRunsDurablySilent(t *testing.T) {
	cmd, sink, stderr := newOpsSemanticProgressTestCommand(t)
	now := time.Date(2026, time.August, 31, 17, 0, 0, 0, time.UTC)
	reporter := newOpsSemanticProgressReporter(cmd, opsSemanticProgressTestConfig(&now, 2))

	now = now.Add(opsDurableMilestoneMinimumElapsed - time.Nanosecond)
	reportOpsSemanticProgressTestCompletion(reporter, "root-1", ops.CompletionDispositionConfirmed, "")
	reporter.Close()

	require.Empty(t, stderr.String())
	require.Equal(t, []string{"deleting process-instance trees, 1/2 process-instance tree(s)"}, sink.Updates())
	require.Equal(t, 1, sink.Stopped())
}

// TestOpsSemanticProgressReporterPrintsFirstTenSecondCompletion verifies the
// first completion crossing the cadence writes one compact aggregate milestone.
func TestOpsSemanticProgressReporterPrintsFirstTenSecondCompletion(t *testing.T) {
	cmd, _, stderr := newOpsSemanticProgressTestCommand(t)
	now := time.Date(2026, time.August, 31, 17, 0, 0, 0, time.UTC)
	reporter := newOpsSemanticProgressReporter(cmd, opsSemanticProgressTestConfig(&now, 2))
	defer reporter.Close()

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	reportOpsSemanticProgressTestCompletion(reporter, "root-1", ops.CompletionDispositionConfirmed, "")

	require.Equal(t, "deleting process-instance trees, 1/2 process-instance tree(s)\n", stderr.String())
}

// TestOpsSemanticProgressReporterSuppressesRapidCompletionDurableLines verifies
// default output does not print one informational line for every completion.
func TestOpsSemanticProgressReporterSuppressesRapidCompletionDurableLines(t *testing.T) {
	cmd, _, stderr := newOpsSemanticProgressTestCommand(t)
	now := time.Date(2026, time.August, 31, 17, 0, 0, 0, time.UTC)
	reporter := newOpsSemanticProgressReporter(cmd, opsSemanticProgressTestConfig(&now, 4))
	defer reporter.Close()

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	reportOpsSemanticProgressTestCompletion(reporter, "root-1", ops.CompletionDispositionConfirmed, "")
	now = now.Add(time.Second)
	reportOpsSemanticProgressTestCompletion(reporter, "root-2", ops.CompletionDispositionConfirmed, "")
	reportOpsSemanticProgressTestCompletion(reporter, "root-3", ops.CompletionDispositionConfirmed, "")

	require.Equal(t, 1, strings.Count(stderr.String(), "deleting process-instance trees"))
}

// TestOpsSemanticProgressReporterFlushesActivatedDurableProgressOnce verifies
// an activated scope flushes later unreported aggregate progress idempotently.
func TestOpsSemanticProgressReporterFlushesActivatedDurableProgressOnce(t *testing.T) {
	cmd, _, stderr := newOpsSemanticProgressTestCommand(t)
	now := time.Date(2026, time.August, 31, 17, 0, 0, 0, time.UTC)
	reporter := newOpsSemanticProgressReporter(cmd, opsSemanticProgressTestConfig(&now, 3))

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	reportOpsSemanticProgressTestCompletion(reporter, "root-1", ops.CompletionDispositionConfirmed, "")
	now = now.Add(time.Second)
	reportOpsSemanticProgressTestCompletion(reporter, "root-2", ops.CompletionDispositionConfirmed, "")
	reporter.Close()
	reporter.Close()

	require.Equal(t, strings.Join([]string{
		"deleting process-instance trees, 1/3 process-instance tree(s)",
		"deleting process-instance trees, 2/3 process-instance tree(s)",
		"",
	}, "\n"), stderr.String())
}

// TestOpsSemanticProgressReporterWarnsImmediatelyForFailures verifies failure
// completions bypass informational pacing and activate durable output.
func TestOpsSemanticProgressReporterWarnsImmediatelyForFailures(t *testing.T) {
	cmd, _, stderr := newOpsSemanticProgressTestCommand(t)
	now := time.Date(2026, time.August, 31, 17, 0, 0, 0, time.UTC)
	reporter := newOpsSemanticProgressReporter(cmd, opsSemanticProgressTestConfig(&now, 2))
	defer reporter.Close()

	now = now.Add(time.Second)
	reportOpsSemanticProgressTestCompletion(reporter, "root-1", ops.CompletionDispositionFailed, "boom")

	got := stderr.String()
	require.Contains(t, got, "root-1 failed: boom")
	require.Contains(t, got, "deleting process-instance trees, 1/2 process-instance tree(s), 1 failed")
	require.Equal(t, 1, strings.Count(got, "root-1 failed"))
}

// TestOpsSemanticProgressReporterVerboseItemsReplacePacedAggregateMilestones
// verifies verbose output emits one identity/outcome line per completion and
// suppresses default aggregate pacing.
func TestOpsSemanticProgressReporterVerboseItemsReplacePacedAggregateMilestones(t *testing.T) {
	cmd, _, stderr := newOpsSemanticProgressTestCommand(t)
	now := time.Date(2026, time.August, 31, 17, 0, 0, 0, time.UTC)
	reporter := newOpsSemanticProgressReporter(cmd, opsSemanticProgressConfig{
		Scope: opsSemanticProgressScope{
			ActivityLabel:             "deleting process-instance trees",
			CoreResource:              "process-instance tree(s)",
			Total:                     3,
			AffectedResource:          "affected process instances",
			AffectedCoverageAvailable: true,
			SubmittedVerb:             "submitted",
			ConfirmedVerb:             "deleted",
			FailedVerb:                "failed",
		},
		Policy: opsSemanticProgressOutputPolicyForChannel(ops.ProgressChannel{Mode: ops.ProgressModeVerbose, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}),
		Now:    func() time.Time { return now },
	})
	defer reporter.Close()
	affected := 2

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	reporter.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Identity: "root-1", Disposition: ops.CompletionDispositionSubmitted, AffectedCount: &affected}})
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	reporter.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Identity: "root-2", Disposition: ops.CompletionDispositionConfirmed, AffectedCount: &affected}})
	reporter.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Identity: "root-3", Disposition: ops.CompletionDispositionFailed, FailureDetail: "boom", AffectedCount: &affected}})

	lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
	require.Equal(t, []string{
		"root-1 submitted (deleting process-instance trees, 1/3 process-instance tree(s), affected process instances: 2)",
		"root-2 deleted (deleting process-instance trees, 2/3 process-instance tree(s), affected process instances: 4)",
		"root-3 failed: boom (deleting process-instance trees, 3/3 process-instance tree(s), 1 failed, affected process instances: 6)",
	}, lines)
	require.NotContains(t, lines, "deleting process-instance trees, 1/3 process-instance tree(s), affected process instances: 2")
}

// TestOpsSemanticProgressReporterQuietFailureBypassesWarnFiltering verifies
// quiet-mode failures remain visible even when the command logger filters warn
// severity records.
func TestOpsSemanticProgressReporterQuietFailureBypassesWarnFiltering(t *testing.T) {
	cmd, _, stderr := newOpsSemanticProgressTestCommand(t)
	var logBuf bytes.Buffer
	cmd.SetContext(logging.ToContext(cmd.Context(), logging.New(logging.LoggerConfig{
		Level:  "error",
		Format: "plain-time",
		Writer: &logBuf,
	})))
	reporter := newOpsSemanticProgressReporter(cmd, opsSemanticProgressConfig{
		Scope: opsSemanticProgressScope{
			ActivityLabel: "deleting process-instance trees",
			CoreResource:  "process-instance tree(s)",
			Total:         1,
			FailedVerb:    "failed",
		},
		Policy: opsSemanticProgressOutputPolicyForChannel(ops.ProgressChannel{Mode: ops.ProgressModeQuiet}),
	})
	defer reporter.Close()

	reporter.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{
		Identity:      "root-1",
		Disposition:   ops.CompletionDispositionFailed,
		FailureDetail: "boom",
	}})

	require.Empty(t, logBuf.String())
	require.Equal(t, "root-1 failed: boom (deleting process-instance trees, 1/1 process-instance tree(s), 1 failed)\n", stderr.String())
}

// TestOpsSemanticProgressReporterCloseIsIdempotent verifies repeated cleanup
// neither double-stops activity nor emits duplicate final records.
func TestOpsSemanticProgressReporterCloseIsIdempotent(t *testing.T) {
	cmd, sink, stderr := newOpsSemanticProgressTestCommand(t)
	reporter := newOpsSemanticProgressReporter(cmd, opsSemanticProgressConfig{
		Scope: opsSemanticProgressScope{
			ActivityLabel: "repairing incidents",
			CoreResource:  "repair(s)",
			Total:         1,
		},
		Policy: opsSemanticProgressOutputPolicyForChannel(ops.ProgressChannel{Mode: ops.ProgressModeVerbose, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}),
	})

	reporter.Report(ops.ProgressEvent{Kind: ops.ProgressEventKindCompletion, Completion: &ops.CompletionProgress{Identity: "inc-1", Disposition: ops.CompletionDispositionConfirmed}})
	reporter.Close()
	reporter.Close()

	require.Equal(t, 1, sink.Stopped())
	require.Equal(t, 1, strings.Count(stderr.String(), "inc-1"))
}

// newOpsSemanticProgressTestCommand returns a command with activity capture and
// stderr capture wired the same way command tests exercise progress rendering.
func newOpsSemanticProgressTestCommand(t *testing.T) (*cobra.Command, *activitysink.Sink, *bytes.Buffer) {
	t.Helper()
	sink := &activitysink.Sink{}
	var stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetErr(&stderr)
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))
	return cmd, sink, &stderr
}

// opsSemanticProgressTestConfig builds a paced default-human reporter fixture
// with deterministic time controlled by the caller.
func opsSemanticProgressTestConfig(now *time.Time, total int) opsSemanticProgressConfig {
	return opsSemanticProgressConfig{
		Scope: opsSemanticProgressScope{
			ActivityLabel: "deleting process-instance trees",
			CoreResource:  "process-instance tree(s)",
			Total:         total,
			ConfirmedVerb: "deleted",
			FailedVerb:    "failed",
		},
		Policy: opsSemanticProgressOutputPolicyForChannel(ops.ProgressChannel{Mode: ops.ProgressModeHuman, TransientAllowed: true, DurableAllowed: true, StderrAllowed: true}),
		Now:    func() time.Time { return *now },
	}
}

// reportOpsSemanticProgressTestCompletion sends one completion fact through the
// reporter without repeating the progress event envelope in each pacing test.
func reportOpsSemanticProgressTestCompletion(reporter *opsSemanticProgressReporter, identity string, disposition ops.CompletionDisposition, detail string) {
	reporter.Report(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Identity:      identity,
			Disposition:   disposition,
			FailureDetail: detail,
		},
	})
}

// requireOpsSemanticProgressCompletedSequence asserts each transient update
// advances completion count exactly once under concurrent reporter calls.
func requireOpsSemanticProgressCompletedSequence(t *testing.T, updates []string, total int) {
	t.Helper()
	completedRe := regexp.MustCompile(`\b(\d+)/` + regexp.QuoteMeta(strconv.Itoa(total)) + `\b`)
	seen := make(map[int]bool, total)
	for _, update := range updates {
		match := completedRe.FindStringSubmatch(update)
		require.Len(t, match, 2, "update does not include completed/total count: %q", update)
		completed, err := strconv.Atoi(match[1])
		require.NoError(t, err)
		require.GreaterOrEqual(t, completed, 1)
		require.LessOrEqual(t, completed, total)
		require.False(t, seen[completed], "duplicate completed count %d in updates %v", completed, updates)
		seen[completed] = true
	}
	for completed := 1; completed <= total; completed++ {
		require.Truef(t, seen[completed], "missing completed count %d", completed)
	}
}
