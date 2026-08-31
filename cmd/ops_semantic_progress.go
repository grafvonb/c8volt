// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

// opsSemanticProgressScope describes one frozen command-owned completion scope.
type opsSemanticProgressScope struct {
	Phase                     string
	ActivityLabel             string
	CoreResource              string
	Total                     int
	AffectedResource          string
	AffectedCoverageAvailable bool
	SubmittedVerb             string
	ConfirmedVerb             string
	FailedVerb                string
}

// opsSemanticProgressConfig groups the fixed dependencies needed for one
// semantic progress reporter lifetime.
type opsSemanticProgressConfig struct {
	Scope  opsSemanticProgressScope
	Policy opsSemanticProgressOutputPolicy
	Now    func() time.Time
}

// opsSemanticProgressAggregate is the monotonic command-owned projection of
// wording-free completion facts.
type opsSemanticProgressAggregate struct {
	Completed     int
	Failed        int
	Total         int
	Affected      int
	AffectedValid bool
}

// opsSemanticProgressReporter serializes completion facts, transient activity
// updates, and durable diagnostic output for one workflow scope.
type opsSemanticProgressReporter struct {
	mu        sync.Mutex
	cmd       *cobra.Command
	scope     opsSemanticProgressScope
	policy    opsSemanticProgressOutputPolicy
	now       func() time.Time
	startedAt time.Time
	aggregate opsSemanticProgressAggregate
	stop      func()
	closed    bool
}

// newOpsSemanticProgressReporter constructs one reporter and immediately owns
// the workflow activity when the current output policy permits transient output.
func newOpsSemanticProgressReporter(cmd *cobra.Command, cfg opsSemanticProgressConfig) *opsSemanticProgressReporter {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	scope := cfg.Scope
	aggregate := opsSemanticProgressAggregate{
		Total:         scope.Total,
		AffectedValid: scope.AffectedCoverageAvailable,
	}
	reporter := &opsSemanticProgressReporter{
		cmd:       cmd,
		scope:     scope,
		policy:    cfg.Policy,
		now:       now,
		startedAt: now(),
		aggregate: aggregate,
		stop:      func() {},
	}
	if cmd != nil && cfg.Policy.TransientActivity {
		reporter.stop = logging.StartActivityWithImportance(opsSemanticProgressCommandContext(cmd), formatOpsSemanticProgressAggregate(scope, aggregate), logging.ActivityImportanceWorkflow)
	}
	return reporter
}

// Report ingests one completion event and ignores unrelated progress facts so
// existing service callbacks can share a single function safely.
func (r *opsSemanticProgressReporter) Report(event ops.ProgressEvent) {
	if r == nil || event.Kind != ops.ProgressEventKindCompletion || event.Completion == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || !r.completionMatchesScope(*event.Completion) {
		return
	}
	r.ingestCompletionLocked(*event.Completion)
	aggregate := r.aggregate
	if r.policy.TransientActivity && r.cmd != nil {
		logging.UpdateActivityWithImportance(opsSemanticProgressCommandContext(r.cmd), formatOpsSemanticProgressAggregate(r.scope, aggregate), logging.ActivityImportanceWorkflow)
	}
	if r.policy.VerboseItems {
		printOpsDurableLine(r.cmd, formatOpsSemanticProgressCompletion(r.scope, aggregate, *event.Completion), r.policy.FailureWarnings && event.Completion.Disposition == ops.CompletionDispositionFailed)
		return
	}
	if r.policy.FailureWarnings && event.Completion.Disposition == ops.CompletionDispositionFailed {
		printOpsDurableLine(r.cmd, formatOpsSemanticProgressCompletion(r.scope, aggregate, *event.Completion), true)
	}
}

// Aggregate returns a copy of the current completion aggregate for tests and
// command-family adapters that need fallback decisions.
func (r *opsSemanticProgressReporter) Aggregate() opsSemanticProgressAggregate {
	if r == nil {
		return opsSemanticProgressAggregate{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.aggregate
}

// Close ends the owned workflow activity exactly once.
func (r *opsSemanticProgressReporter) Close() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	stop := r.stop
	r.mu.Unlock()
	if stop != nil {
		stop()
	}
}

// ingestCompletionLocked applies one fact while preserving monotonic counters
// and permanently invalidating affected output on unknown or invalid deltas.
func (r *opsSemanticProgressReporter) ingestCompletionLocked(completion ops.CompletionProgress) {
	if r.aggregate.Total <= 0 && completion.Total > 0 {
		r.aggregate.Total = completion.Total
	}
	if r.aggregate.Total > 0 && r.aggregate.Completed >= r.aggregate.Total {
		return
	}
	r.aggregate.Completed++
	if completion.Disposition == ops.CompletionDispositionFailed {
		r.aggregate.Failed++
	}
	if !r.aggregate.AffectedValid {
		return
	}
	if completion.AffectedCount == nil || *completion.AffectedCount < 0 {
		r.aggregate.AffectedValid = false
		return
	}
	r.aggregate.Affected += *completion.AffectedCount
}

// completionMatchesScope avoids mixing unrelated completion phases when a
// command reuses the same facade callback for discovery and mutation.
func (r *opsSemanticProgressReporter) completionMatchesScope(completion ops.CompletionProgress) bool {
	phase := strings.TrimSpace(r.scope.Phase)
	return phase == "" || strings.TrimSpace(completion.Phase) == "" || strings.TrimSpace(completion.Phase) == phase
}

// opsSemanticProgressCommandContext keeps nil command contexts from disabling
// reporter construction in focused unit tests.
func opsSemanticProgressCommandContext(cmd *cobra.Command) context.Context {
	if cmd == nil || cmd.Context() == nil {
		return context.Background()
	}
	return cmd.Context()
}

// processInstanceMutationSemanticProgressScope returns the shared vocabulary
// for process-instance cancel/delete workflow completions.
func processInstanceMutationSemanticProgressScope(operation string, total int, affectedCoverageAvailable bool) opsSemanticProgressScope {
	label, confirmedVerb := processInstanceMutationResultWords(operation, false)
	_, submittedVerb := processInstanceMutationResultWords(operation, true)
	activity := strings.TrimSpace(label)
	if activity == "" {
		activity = "mutation"
	}
	return opsSemanticProgressScope{
		Phase:                     strings.TrimSpace(operation),
		ActivityLabel:             activity + " process-instance trees",
		CoreResource:              "process-instance tree(s)",
		Total:                     total,
		AffectedResource:          "affected process instances",
		AffectedCoverageAvailable: affectedCoverageAvailable,
		SubmittedVerb:             submittedVerb,
		ConfirmedVerb:             confirmedVerb,
		FailedVerb:                "failed",
	}
}
