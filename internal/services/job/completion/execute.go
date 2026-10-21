// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package completion

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
)

// ExecutionPorts permits key rechecks and one-shot submission, never rediscovery or other mutations.
// Adapters must treat the supplied payload as read-only; it is a private snapshot shared by workers.
type ExecutionPorts struct {
	Get    func(context.Context, string, ...services.CallOption) (d.Job, error)
	Submit func(context.Context, d.Job, map[string]any) (d.CompletionSubmission, error)
}

// executionResult distinguishes unstarted pool slots from completed callback results.
type executionResult struct {
	started bool
	outcome d.CompletionOutcome
	err     error
}

// Execute validates and executes a frozen batch with bounded workers and one attempt per key.
// Caller cancellation remains a failure independently of the final per-key partition.
func Execute(ctx context.Context, batch d.PreparedCompletionBatch, tenant string, backoff config.BackoffConfig, ports ExecutionPorts, wantedWorkers int, opts ...services.CallOption) (d.CompletionBatchReport, error) {
	cfg := services.ApplyCallOptions(opts)
	batch, err := snapshotExecutionBatch(batch)
	report := reportForBatch(batch)
	if err == nil {
		err = validateExecutionBatch(batch, tenant)
	}
	if err != nil {
		return report, err
	}
	if cfg.DryRun && ctx.Err() == nil {
		report.Status = d.CompletionBatchPreview
		return report, nil
	}
	report.Status = d.CompletionBatchFinished
	report.Candidates = nil
	report.Outcomes = make([]d.CompletionOutcome, len(batch.Candidates))
	for i, candidate := range batch.Candidates {
		report.Outcomes[i] = d.CompletionOutcome{Candidate: candidate, Disposition: d.CompletionDispositionNotStarted, Submission: d.CompletionSubmission{Status: d.CompletionSubmissionNotSubmitted}, Observation: d.CompletionObservation{Status: d.CompletionObservationSkipped}}
	}
	if len(batch.Candidates) > 0 && ctx.Err() == nil && (ports.Get == nil || ports.Submit == nil) {
		report = reportForBatch(batch)
		return report, fmt.Errorf("%w: missing job completion execution port", d.ErrUnsupported)
	}
	workers := toolx.DetermineNoOfWorkers(len(batch.Candidates), wantedWorkers, cfg.NoWorkerLimit)
	var stopped atomic.Bool
	// Pool fail-fast cancellation controls scheduling only. Already-started callbacks
	// use the caller context, so another key's rejection cannot erase their evidence.
	results, _ := pool.ExecuteSlice(ctx, batch.Candidates, workers, cfg.FailFast, func(scheduleCtx context.Context, candidate d.CompletionCandidate, _ int) (executionResult, error) {
		if scheduleCtx.Err() != nil || ctx.Err() != nil || stopped.Load() {
			return executionResult{}, nil
		}
		outcome, runErr := executeCandidate(ctx, batch, candidate, backoff, ports, cfg.NoWait, opts...)
		if runErr != nil && cfg.FailFast {
			stopped.Store(true)
		}
		return executionResult{started: true, outcome: outcome, err: runErr}, runErr
	})
	// Aggregate callback errors, not pool-internal fail-fast cancellation errors.
	// The caller error is joined explicitly because the pool can return nil on cancellation.
	for i, result := range results {
		if result.started {
			report.Outcomes[i] = result.outcome
			err = errors.Join(err, result.err)
		}
	}
	callerErr := ctx.Err()
	for i := range report.Outcomes {
		out := &report.Outcomes[i]
		switch out.Disposition {
		case d.CompletionDispositionAccepted:
			report.Counts.Accepted++
		case d.CompletionDispositionCompleted:
			report.Counts.Completed++
		case d.CompletionDispositionUnsatisfied:
			report.Counts.Unsatisfied++
		case d.CompletionDispositionNotStarted:
			report.Counts.NotStarted++
			out.Reason = d.CompletionOutcomeFailFast
			if callerErr != nil {
				out.Reason = d.CompletionOutcomeInterrupted
			}
		}
	}
	if callerErr != nil {
		report.Status = d.CompletionBatchInterrupted
		err = errors.Join(err, callerErr)
	}
	return report, err
}

// reportForBatch preserves diagnostic preparation metadata without inventing execution rows.
func reportForBatch(b d.PreparedCompletionBatch) d.CompletionBatchReport {
	return d.CompletionBatchReport{Status: d.CompletionBatchPreparationFailed, Selection: b.Selection, DiscoveryTenant: b.DiscoveryTenant, TenantIDs: b.TenantIDs, CompletionReason: b.CompletionReason, PageCount: b.PageCount, ObservedCount: b.ObservedCount, Limited: b.Limited, Counts: d.CompletionCounts{Selected: len(b.Candidates)}, Candidates: b.Candidates}
}

// executeCandidate rechecks the original key and preserves submission evidence during reconciliation.
func executeCandidate(ctx context.Context, b d.PreparedCompletionBatch, candidate d.CompletionCandidate, backoff config.BackoffConfig, ports ExecutionPorts, noWait bool, opts ...services.CallOption) (d.CompletionOutcome, error) {
	out := d.CompletionOutcome{Candidate: candidate, Disposition: d.CompletionDispositionUnsatisfied, Submission: d.CompletionSubmission{Status: d.CompletionSubmissionNotSubmitted}}
	job, observation, err := observeJob(ctx, candidate.Key, candidate.State, d.CompletionPhaseRecheck, ports.Get, opts...)
	out.Observation = observation
	if err != nil {
		out.Reason = d.CompletionOutcomeUnreadable
		if errors.Is(err, d.ErrNotFound) {
			out.Reason = d.CompletionOutcomeMissing
		}
		if ctx.Err() != nil {
			out.Reason = d.CompletionOutcomeInterrupted
		}
		return out, err
	}
	if job.State == "COMPLETED" {
		out.Disposition = d.CompletionDispositionCompleted
		return out, nil
	}
	if !Matches(job, b.Selection, b.DiscoveryTenant) {
		out.Reason = d.CompletionOutcomeScopeChanged
		return out, outcomeError(out)
	}
	if err = common.ValidateJobCompletionPayload(job, b.Variables); err != nil {
		out.Reason = d.CompletionOutcomePayloadIncompatible
		return out, err
	}
	switch job.State {
	case "CANCELED", "ERROR_THROWN":
		out.Reason = d.CompletionOutcomeOtherTerminal
		return out, outcomeError(out)
	case "CREATED", "FAILED", "TIMED_OUT", "RETRIES_UPDATED", "MIGRATED":
	default:
		out.Reason = d.CompletionOutcomeUnsupportedState
		return out, outcomeError(out)
	}
	if err = ctx.Err(); err != nil {
		out.Reason = d.CompletionOutcomeInterrupted
		return out, err
	}
	out.Submission, err = ports.Submit(ctx, job, b.Variables)
	switch out.Submission.Status {
	case d.CompletionSubmissionAccepted:
		out.Disposition = d.CompletionDispositionAccepted
		out.Observation = d.CompletionObservation{Status: d.CompletionObservationSkipped}
		return out, nil
	case d.CompletionSubmissionRejected:
		out.Reason = d.CompletionOutcomeRejected
	case d.CompletionSubmissionUncertain:
		out.Reason = d.CompletionOutcomeUncertain
	case d.CompletionSubmissionNotSubmitted:
		out.Reason = d.CompletionOutcomePayloadIncompatible
		if ctx.Err() != nil {
			out.Reason = d.CompletionOutcomeInterrupted
		}
	default:
		out.Submission.Status = d.CompletionSubmissionUncertain
		out.Reason = d.CompletionOutcomeUncertain
		err = errors.Join(err, fmt.Errorf("%w: invalid completion submission evidence", d.ErrMalformedResponse))
	}
	if needsReconciliation(out.Submission) {
		var readErr error
		out, readErr = reconcile(ctx, out, backoff, ports.Get, noWait, opts...)
		if out.Disposition == d.CompletionDispositionCompleted {
			return out, nil
		}
		err = errors.Join(err, readErr)
	}
	if err == nil {
		err = outcomeError(out)
	}
	return out, err
}

// outcomeError classifies a known unsatisfied target without attributing completion to this invocation.
func outcomeError(out d.CompletionOutcome) error {
	return fmt.Errorf("%w: job %s completion unsatisfied: %s", d.ErrPrecondition, out.Candidate.Key, out.Reason)
}
