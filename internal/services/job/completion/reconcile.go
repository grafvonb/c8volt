// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package completion

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// needsReconciliation selects only race rejections and ambiguous submissions; it never authorizes replay.
func needsReconciliation(submission d.CompletionSubmission) bool {
	return submission.Status == d.CompletionSubmissionUncertain || (submission.Status == d.CompletionSubmissionRejected && submission.HTTPStatus != nil && (*submission.HTTPStatus == http.StatusNotFound || *submission.HTTPStatus == http.StatusConflict))
}

// observeJob retains the last known state on failed reads and rejects responses for another key.
func observeJob(ctx context.Context, key, lastState string, phase d.CompletionObservationPhase, get func(context.Context, string, ...services.CallOption) (d.Job, error), opts ...services.CallOption) (d.Job, d.CompletionObservation, error) {
	observation := d.CompletionObservation{Status: d.CompletionObservationUnreadable, State: lastState, Phase: phase}
	if err := ctx.Err(); err != nil {
		observation.Diagnostic = err.Error()
		return d.Job{}, observation, err
	}
	job, err := get(ctx, key, opts...)
	if err == nil && job.Key != key {
		err = fmt.Errorf("%w: job lookup returned a different key", d.ErrMalformedResponse)
	}
	if err != nil {
		if errors.Is(err, d.ErrNotFound) {
			observation.Status = d.CompletionObservationMissing
		}
		observation.Diagnostic = err.Error()
		return job, observation, err
	}
	observation.State = job.State
	switch job.State {
	case "COMPLETED":
		observation.Status = d.CompletionObservationCompleted
	case "CREATED", "FAILED", "TIMED_OUT", "RETRIES_UPDATED", "MIGRATED", "CANCELED", "ERROR_THROWN":
		observation.Status = d.CompletionObservationNotCompleted
	default:
		observation.Status = d.CompletionObservationUnknown
	}
	return job, observation, nil
}

// reconcile reads within the existing timeout/backoff budget. Missing or stale reads
// cannot change submission evidence, and no-wait permits only the first read.
func reconcile(ctx context.Context, out d.CompletionOutcome, backoff config.BackoffConfig, get func(context.Context, string, ...services.CallOption) (d.Job, error), noWait bool, opts ...services.CallOption) (d.CompletionOutcome, error) {
	_ = backoff.Normalize()
	readCtx, cancel := context.WithTimeout(ctx, backoff.Timeout)
	defer cancel()
	delay := backoff.InitialDelay
	var lastErr error
	for attempt := 1; ; attempt++ {
		if err := readCtx.Err(); err != nil {
			return out, errors.Join(lastErr, err)
		}
		job, observation, err := observeJob(readCtx, out.Candidate.Key, out.Observation.State, d.CompletionPhaseReconciliation, get, opts...)
		out.Observation = observation
		lastErr = err
		if err == nil {
			switch job.State {
			case "COMPLETED":
				out.Disposition = d.CompletionDispositionCompleted
				out.Reason = ""
				return out, nil
			case "CANCELED", "ERROR_THROWN":
				out.Reason = d.CompletionOutcomeOtherTerminal
				return out, outcomeError(out)
			}
		}
		if noWait || (backoff.MaxRetries > 0 && attempt >= backoff.MaxRetries) {
			return out, lastErr
		}
		timer := time.NewTimer(delay)
		select {
		case <-readCtx.Done():
			timer.Stop()
			return out, errors.Join(lastErr, readCtx.Err())
		case <-timer.C:
			delay = backoff.NextDelay(delay)
		}
	}
}
