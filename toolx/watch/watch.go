// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

// Package watch provides reusable fixed-interval observation loops for
// read-only CLI commands.
package watch

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// TerminationReason records why a watch session stopped.
type TerminationReason string

const (
	// TerminationReasonCanceled means the caller's context was canceled.
	TerminationReasonCanceled TerminationReason = "canceled"
	// TerminationReasonTimeout means a context deadline or configured timeout stopped the session.
	TerminationReasonTimeout TerminationReason = "timeout"
	// TerminationReasonRetryExhausted means consecutive retryable failures exceeded MaxRetries.
	TerminationReasonRetryExhausted TerminationReason = "retry_exhausted"
	// TerminationReasonFatalError means a non-retryable tick error stopped the session.
	TerminationReasonFatalError TerminationReason = "fatal_error"
)

// Tick describes one snapshot attempt.
type Tick struct {
	Index               int64
	StartedAt           time.Time
	ConsecutiveFailures int
}

// TickFunc collects and emits one watch snapshot.
type TickFunc func(context.Context, Tick) error

// RetryableFunc classifies a tick error as retryable.
type RetryableFunc func(error) bool

// SleepFunc waits between completed tick attempts.
type SleepFunc func(context.Context, time.Duration) error

// RetryEvent records one retryable tick failure.
type RetryEvent struct {
	Tick                int64
	Err                 error
	ConsecutiveFailures int
	MaxRetries          int
}

// RetryObserver receives retryable failure notifications before the next sleep.
type RetryObserver func(RetryEvent)

// Options configure a fixed-interval watch session.
type Options struct {
	Interval   time.Duration
	Timeout    time.Duration
	MaxRetries int
	Retryable  RetryableFunc
	Sleep      SleepFunc
	OnRetry    RetryObserver
}

// Result summarizes a stopped watch session.
type Result struct {
	Ticks               int64
	SuccessfulTicks     int64
	FailedTicks         int64
	ConsecutiveFailures int
	Reason              TerminationReason
	LastError           error
}

// ErrInvalidInterval is returned when a watch session is configured without a positive interval.
var ErrInvalidInterval = errors.New("watch interval must be positive")

// Run starts a fixed-interval watch loop, running tick immediately before the
// first sleep and resetting the consecutive retry budget after every success.
func Run(ctx context.Context, opts Options, tick TickFunc) (Result, error) {
	if tick == nil {
		return Result{Reason: TerminationReasonFatalError}, errors.New("watch tick function is required")
	}
	if opts.Interval <= 0 {
		return Result{Reason: TerminationReasonFatalError}, ErrInvalidInterval
	}
	if opts.Sleep == nil {
		opts.Sleep = sleep
	}
	if opts.Retryable == nil {
		opts.Retryable = func(error) bool { return true }
	}
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	var result Result
	for {
		if err := ctx.Err(); err != nil {
			return terminateForContext(result, err)
		}

		result.Ticks++
		err := tick(ctx, Tick{
			Index:               result.Ticks,
			StartedAt:           time.Now(),
			ConsecutiveFailures: result.ConsecutiveFailures,
		})
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return terminateForContext(result, ctxErr)
			}
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return terminateForContext(result, err)
			}
			result.FailedTicks++
			result.LastError = err
			if !opts.Retryable(err) {
				result.Reason = TerminationReasonFatalError
				return result, err
			}
			result.ConsecutiveFailures++
			if opts.OnRetry != nil {
				opts.OnRetry(RetryEvent{
					Tick:                result.Ticks,
					Err:                 err,
					ConsecutiveFailures: result.ConsecutiveFailures,
					MaxRetries:          opts.MaxRetries,
				})
			}
			if opts.MaxRetries > 0 && result.ConsecutiveFailures > opts.MaxRetries {
				result.Reason = TerminationReasonRetryExhausted
				return result, fmt.Errorf("watch retry exhausted after %d consecutive failure(s): %w", result.ConsecutiveFailures, err)
			}
		} else {
			result.SuccessfulTicks++
			result.ConsecutiveFailures = 0
			result.LastError = nil
		}

		if err := opts.Sleep(ctx, opts.Interval); err != nil {
			return terminateForContext(result, err)
		}
	}
}

// sleep waits for one interval while respecting cancellation.
func sleep(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// terminateForContext converts context completion into a watch termination result.
func terminateForContext(result Result, err error) (Result, error) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		result.Reason = TerminationReasonTimeout
		return result, nil
	case errors.Is(err, context.Canceled):
		result.Reason = TerminationReasonCanceled
		return result, nil
	default:
		result.Reason = TerminationReasonFatalError
		return result, err
	}
}
