// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package watch

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestWatchRunExecutesFirstTickBeforeSleeping protects the operator-visible
// invariant that watch mode renders an immediate first snapshot.
func TestWatchRunExecutesFirstTickBeforeSleeping(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	var events []string
	result, err := Run(ctx, Options{
		Interval: time.Second,
		Sleep: func(_ context.Context, interval time.Duration) error {
			require.Equal(t, time.Second, interval)
			events = append(events, "sleep")
			cancel()
			return context.Canceled
		},
	}, func(context.Context, Tick) error {
		events = append(events, "tick")
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, TerminationReasonCanceled, result.Reason)
	require.Equal(t, []string{"tick", "sleep"}, events)
}

// TestWatchRunSleepsPositiveIntervalBetweenTicks verifies successful snapshots use
// the configured fixed interval before the next attempt.
func TestWatchRunSleepsPositiveIntervalBetweenTicks(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	var intervals []time.Duration
	result, err := Run(ctx, Options{
		Interval: 25 * time.Millisecond,
		Sleep: func(_ context.Context, interval time.Duration) error {
			intervals = append(intervals, interval)
			if len(intervals) == 2 {
				cancel()
				return context.Canceled
			}
			return nil
		},
	}, func(_ context.Context, tick Tick) error {
		require.EqualValues(t, len(intervals)+1, tick.Index)
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, TerminationReasonCanceled, result.Reason)
	require.Equal(t, []time.Duration{25 * time.Millisecond, 25 * time.Millisecond}, intervals)
	require.EqualValues(t, 2, result.SuccessfulTicks)
}

// TestWatchRunKeepsTicksSerialWhenTickExceedsInterval verifies a slow tick must
// finish before the runner sleeps and starts the next tick.
func TestWatchRunKeepsTicksSerialWhenTickExceedsInterval(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	var events []string
	activeTicks := 0
	result, err := Run(ctx, Options{
		Interval: time.Nanosecond,
		Sleep: func(_ context.Context, interval time.Duration) error {
			require.Equal(t, time.Nanosecond, interval)
			require.Zero(t, activeTicks)
			events = append(events, "sleep")
			if len(events) >= 4 {
				cancel()
				return context.Canceled
			}
			return nil
		},
	}, func(_ context.Context, tick Tick) error {
		require.Zero(t, activeTicks)
		activeTicks++
		events = append(events, fmt.Sprintf("tick %d start", tick.Index))
		time.Sleep(time.Millisecond)
		events = append(events, fmt.Sprintf("tick %d end", tick.Index))
		activeTicks--
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, TerminationReasonCanceled, result.Reason)
	require.Equal(t, []string{"tick 1 start", "tick 1 end", "sleep", "tick 2 start", "tick 2 end", "sleep"}, events)
	require.EqualValues(t, 2, result.SuccessfulTicks)
}

// TestWatchRunStopsOnContextCancellation verifies cancellation during interval wait
// ends the session cleanly instead of reporting a lookup failure.
func TestWatchRunStopsOnContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	result, err := Run(ctx, Options{
		Interval: time.Second,
		Sleep: func(_ context.Context, _ time.Duration) error {
			cancel()
			return context.Canceled
		},
	}, func(context.Context, Tick) error {
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, TerminationReasonCanceled, result.Reason)
	require.EqualValues(t, 1, result.Ticks)
}

// TestWatchRunStopsWhenTickReturnsContextCancellation verifies tick functions
// can propagate cancellation directly without consuming retry budget.
func TestWatchRunStopsWhenTickReturnsContextCancellation(t *testing.T) {
	t.Parallel()

	result, err := Run(context.Background(), Options{
		Interval: time.Second,
		Sleep: func(context.Context, time.Duration) error {
			t.Fatal("sleep should not run after tick cancellation")
			return nil
		},
	}, func(context.Context, Tick) error {
		return context.Canceled
	})

	require.NoError(t, err)
	require.Equal(t, TerminationReasonCanceled, result.Reason)
	require.EqualValues(t, 1, result.Ticks)
	require.Zero(t, result.FailedTicks)
}

// TestWatchRunStopsOnTimeout verifies an overall watch timeout terminates the
// session without converting the stop into a tick error.
func TestWatchRunStopsOnTimeout(t *testing.T) {
	t.Parallel()

	result, err := Run(context.Background(), Options{
		Interval: time.Second,
		Timeout:  5 * time.Millisecond,
		Sleep: func(ctx context.Context, _ time.Duration) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}, func(context.Context, Tick) error {
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, TerminationReasonTimeout, result.Reason)
	require.EqualValues(t, 1, result.SuccessfulTicks)
}

// TestWatchRunResetsRetryBudgetAfterSuccess verifies consecutive retry failures are
// counted only until a successful snapshot resets the budget.
func TestWatchRunResetsRetryBudgetAfterSuccess(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	transient := errors.New("temporary lookup failure")
	calls := 0
	var retryCounts []int
	result, err := Run(ctx, Options{
		Interval:   time.Millisecond,
		MaxRetries: 1,
		OnRetry: func(event RetryEvent) {
			retryCounts = append(retryCounts, event.ConsecutiveFailures)
		},
		Sleep: func(_ context.Context, _ time.Duration) error {
			if calls == 4 {
				cancel()
				return context.Canceled
			}
			return nil
		},
	}, func(context.Context, Tick) error {
		calls++
		switch calls {
		case 1, 3:
			return transient
		default:
			return nil
		}
	})

	require.NoError(t, err)
	require.Equal(t, TerminationReasonCanceled, result.Reason)
	require.EqualValues(t, 4, result.Ticks)
	require.EqualValues(t, 2, result.SuccessfulTicks)
	require.Equal(t, []int{1, 1}, retryCounts)
}

// TestWatchRunStopsWhenRetryBudgetIsExhausted verifies persistent retryable
// failures stop after the allowed consecutive retry budget is exceeded.
func TestWatchRunStopsWhenRetryBudgetIsExhausted(t *testing.T) {
	t.Parallel()

	transient := errors.New("temporary lookup failure")
	result, err := Run(context.Background(), Options{
		Interval:   time.Millisecond,
		MaxRetries: 2,
		Sleep: func(context.Context, time.Duration) error {
			return nil
		},
	}, func(context.Context, Tick) error {
		return transient
	})

	require.Error(t, err)
	require.ErrorIs(t, err, transient)
	require.Equal(t, TerminationReasonRetryExhausted, result.Reason)
	require.EqualValues(t, 3, result.Ticks)
	require.Equal(t, 3, result.ConsecutiveFailures)
}

// TestWatchRunRequestTimeoutsRetryWithoutSessionDeadline distinguishes a failed
// request from the watch lifetime and preserves consecutive retry accounting.
func TestWatchRunRequestTimeoutsRetryWithoutSessionDeadline(t *testing.T) {
	t.Parallel()
	for _, exhausted := range []bool{false, true} {
		t.Run(fmt.Sprint(exhausted), func(t *testing.T) {
			calls := 0
			retries := 0
			requestErr := fmt.Errorf("request deadline: %w", context.DeadlineExceeded)
			result, err := Run(context.Background(), Options{
				Interval: time.Second, MaxRetries: 1,
				Retryable: func(err error) bool { return errors.Is(err, context.DeadlineExceeded) },
				OnRetry:   func(RetryEvent) { retries++ },
				Sleep: func(context.Context, time.Duration) error {
					if calls == 4 {
						return context.Canceled
					}
					return nil
				},
			}, func(ctx context.Context, tick Tick) error {
				_, hasDeadline := ctx.Deadline()
				require.False(t, hasDeadline)
				calls++
				if exhausted || calls%2 == 1 {
					return requestErr
				}
				return nil
			})
			if exhausted {
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.Equal(t, TerminationReasonRetryExhausted, result.Reason)
				require.Equal(t, 2, calls)
				require.Equal(t, 1, retries)
			} else {
				require.NoError(t, err)
				require.Equal(t, TerminationReasonCanceled, result.Reason)
				require.EqualValues(t, 2, result.SuccessfulTicks)
				require.Equal(t, 2, retries)
			}
		})
	}
}

// TestWatchRunSessionDeadlineDuringRequest stops a blocked request only when
// the session deadline itself expires, without starting a retry.
func TestWatchRunSessionDeadlineDuringRequest(t *testing.T) {
	t.Parallel()
	result, err := Run(context.Background(), Options{
		Interval: time.Second, Timeout: 5 * time.Millisecond,
		OnRetry: func(RetryEvent) { t.Fatal("expired session must not retry") },
	}, func(ctx context.Context, _ Tick) error {
		<-ctx.Done()
		return fmt.Errorf("request: %w", ctx.Err())
	})
	require.NoError(t, err)
	require.Equal(t, TerminationReasonTimeout, result.Reason)
	require.EqualValues(t, 1, result.Ticks)
}
