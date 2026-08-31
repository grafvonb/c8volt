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
