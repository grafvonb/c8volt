// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package waiter

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPIWaiter struct {
	getProcessInstance func(ctx context.Context, key string) (d.ProcessInstance, error)
	getStateByKey      func(ctx context.Context, key string) (d.State, d.ProcessInstance, error)
}

func (s stubPIWaiter) GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error) {
	if s.getProcessInstance == nil {
		panic("unexpected GetProcessInstance call")
	}
	return s.getProcessInstance(ctx, key)
}

func (s stubPIWaiter) GetProcessInstanceStateByKey(ctx context.Context, key string, _ ...services.CallOption) (d.State, d.ProcessInstance, error) {
	return s.getStateByKey(ctx, key)
}

func TestWaitForProcessInstanceExpectation_IncidentTrueWaitsAcrossPolling(t *testing.T) {
	t.Parallel()

	wantIncident := true
	attempts := 0
	waiter := stubPIWaiter{
		getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
			attempts++
			return d.ProcessInstance{
				Key:      key,
				State:    d.StateActive,
				Incident: attempts >= 2,
			}, nil
		},
	}

	got, pi, err := WaitForProcessInstanceExpectation(
		context.Background(),
		waiter,
		testConfig(time.Millisecond, 3, 100*time.Millisecond),
		testLogger(),
		"123",
		d.ProcessInstanceExpectationRequest{Incident: &wantIncident},
	)

	require.NoError(t, err)
	assert.Equal(t, 2, attempts)
	assert.True(t, got.Ok)
	assert.Equal(t, "123", got.Key)
	assert.Equal(t, d.StateActive, got.State)
	require.NotNil(t, got.Incident)
	assert.True(t, *got.Incident)
	assert.Contains(t, got.Status, "satisfied expectation(s)")
	assert.Equal(t, d.ProcessInstance{Key: "123", State: d.StateActive, Incident: true}, pi)
}

func TestWaitForProcessInstanceExpectation_IncidentFalseRequiresPresentInstance(t *testing.T) {
	t.Parallel()

	wantIncident := false
	attempts := 0
	waiter := stubPIWaiter{
		getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
			attempts++
			return d.ProcessInstance{}, d.ErrNotFound
		},
	}

	got, pi, err := WaitForProcessInstanceExpectation(
		context.Background(),
		waiter,
		testConfig(time.Millisecond, 2, 100*time.Millisecond),
		testLogger(),
		"missing",
		d.ProcessInstanceExpectationRequest{Incident: &wantIncident},
	)

	require.Error(t, err)
	assert.Equal(t, 2, attempts)
	assert.False(t, got.Ok)
	assert.Equal(t, "missing", got.Key)
	assert.Equal(t, d.StateUnknown, got.State)
	assert.Nil(t, got.Incident)
	assert.Contains(t, got.Status, "exceeded max_retries (2)")
	assert.Equal(t, d.ProcessInstance{}, pi)
}

// TestWaitForProcessInstanceState verifies single-instance wait behavior across success, retry, timeout, and activity paths.
func TestWaitForProcessInstanceState(t *testing.T) {
	t.Run("returns immediately when desired state is already present", func(t *testing.T) {
		t.Parallel()

		waiter := stubPIWaiter{
			getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
				return d.StateActive, d.ProcessInstance{Key: key, State: d.StateActive}, nil
			},
		}

		got, pi, err := WaitForProcessInstanceState(
			context.Background(),
			waiter,
			testConfig(5*time.Millisecond, 3, 25*time.Millisecond),
			testLogger(),
			"123",
			d.States{d.StateActive},
		)

		require.NoError(t, err)
		assert.True(t, got.Ok)
		assert.Equal(t, d.StateActive, got.State)
		assert.Contains(t, got.Status, "already in one of the desired state(s)")
		assert.Equal(t, d.ProcessInstance{Key: "123", State: d.StateActive}, pi)
	})

	t.Run("treats terminated as matching desired canceled", func(t *testing.T) {
		t.Parallel()

		waiter := stubPIWaiter{
			getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
				return d.StateTerminated, d.ProcessInstance{Key: key, State: d.StateTerminated}, nil
			},
		}

		got, pi, err := WaitForProcessInstanceState(
			context.Background(),
			waiter,
			testConfig(5*time.Millisecond, 3, 25*time.Millisecond),
			testLogger(),
			"123",
			d.States{d.StateCanceled},
		)

		require.NoError(t, err)
		assert.True(t, got.Ok)
		assert.Equal(t, d.StateTerminated, got.State)
		assert.Contains(t, got.Status, "already in one of the desired state(s)")
		assert.Equal(t, d.ProcessInstance{Key: "123", State: d.StateTerminated}, pi)
	})

	t.Run("treats canceled as matching desired terminated", func(t *testing.T) {
		t.Parallel()

		waiter := stubPIWaiter{
			getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
				return d.StateCanceled, d.ProcessInstance{Key: key, State: d.StateCanceled}, nil
			},
		}

		got, pi, err := WaitForProcessInstanceState(
			context.Background(),
			waiter,
			testConfig(5*time.Millisecond, 3, 25*time.Millisecond),
			testLogger(),
			"123",
			d.States{d.StateTerminated},
		)

		require.NoError(t, err)
		assert.True(t, got.Ok)
		assert.Equal(t, d.StateCanceled, got.State)
		assert.Contains(t, got.Status, "already in one of the desired state(s)")
		assert.Equal(t, d.ProcessInstance{Key: "123", State: d.StateCanceled}, pi)
	})

	t.Run("treats not found as absent when absent is desired", func(t *testing.T) {
		t.Parallel()

		waiter := stubPIWaiter{
			getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
				return d.StateUnknown, d.ProcessInstance{}, d.ErrNotFound
			},
		}

		got, pi, err := WaitForProcessInstanceState(
			context.Background(),
			waiter,
			testConfig(5*time.Millisecond, 3, 25*time.Millisecond),
			testLogger(),
			"missing",
			d.States{d.StateAbsent},
		)

		require.NoError(t, err)
		assert.True(t, got.Ok)
		assert.Equal(t, d.StateAbsent, got.State)
		assert.Contains(t, got.Status, "reached one of the desired state(s)")
		assert.Equal(t, d.ProcessInstance{}, pi)
	})

	t.Run("keeps not found strict when absent is not desired", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		waiter := stubPIWaiter{
			getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
				attempts++
				return d.StateUnknown, d.ProcessInstance{}, d.ErrNotFound
			},
		}

		got, pi, err := WaitForProcessInstanceState(
			context.Background(),
			waiter,
			testConfig(time.Millisecond, 2, 100*time.Millisecond),
			testLogger(),
			"missing",
			d.States{d.StateCompleted},
		)

		require.Error(t, err)
		assert.Equal(t, 2, attempts)
		assert.False(t, got.Ok)
		assert.Equal(t, d.StateUnknown, got.State)
		assert.Contains(t, got.Status, "exceeded max_retries (2)")
		assert.Equal(t, d.ProcessInstance{}, pi)
	})

	t.Run("stops when max retries are exceeded", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		waiter := stubPIWaiter{
			getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
				attempts++
				return d.StateActive, d.ProcessInstance{Key: key, State: d.StateActive}, nil
			},
		}

		got, pi, err := WaitForProcessInstanceState(
			context.Background(),
			waiter,
			testConfig(time.Millisecond, 2, 100*time.Millisecond),
			testLogger(),
			"123",
			d.States{d.StateCompleted},
		)

		require.Error(t, err)
		assert.Equal(t, 2, attempts)
		assert.False(t, got.Ok)
		assert.Equal(t, d.StateUnknown, got.State)
		assert.Contains(t, got.Status, "exceeded max_retries (2)")
		assert.Equal(t, d.ProcessInstance{}, pi)
	})

	t.Run("honors context cancellation before polling starts", func(t *testing.T) {
		t.Parallel()

		called := false
		waiter := stubPIWaiter{
			getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
				called = true
				return d.StateUnknown, d.ProcessInstance{}, nil
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		got, pi, err := WaitForProcessInstanceState(
			ctx,
			waiter,
			testConfig(5*time.Millisecond, 3, 25*time.Millisecond),
			testLogger(),
			"123",
			d.States{d.StateCompleted},
		)

		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		assert.False(t, called)
		assert.False(t, got.Ok)
		assert.Equal(t, d.StateUnknown, got.State)
		assert.Contains(t, got.Status, "due to context error")
		assert.Equal(t, d.ProcessInstance{}, pi)
	})

	t.Run("treats 404-shaped errors as absent", func(t *testing.T) {
		t.Parallel()

		waiter := stubPIWaiter{
			getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
				return d.StateUnknown, d.ProcessInstance{}, errors.New("operate returned 404")
			},
		}

		got, _, err := WaitForProcessInstanceState(
			context.Background(),
			waiter,
			testConfig(5*time.Millisecond, 3, 25*time.Millisecond),
			testLogger(),
			"missing",
			d.States{d.StateAbsent},
		)

		require.NoError(t, err)
		assert.True(t, got.Ok)
		assert.Equal(t, d.StateAbsent, got.State)
	})

	t.Run("treats not-found text errors as absent", func(t *testing.T) {
		t.Parallel()

		waiter := stubPIWaiter{
			getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
				return d.StateUnknown, d.ProcessInstance{}, errors.New("process instance does not exist anymore")
			},
		}

		got, _, err := WaitForProcessInstanceState(
			context.Background(),
			waiter,
			testConfig(5*time.Millisecond, 3, 25*time.Millisecond),
			testLogger(),
			"missing",
			d.States{d.StateAbsent},
		)

		require.NoError(t, err)
		assert.True(t, got.Ok)
		assert.Equal(t, d.StateAbsent, got.State)
	})

	t.Run("only logs polling attempts when verbose", func(t *testing.T) {
		t.Parallel()

		run := func(t *testing.T, opts ...services.CallOption) string {
			t.Helper()

			attempts := 0
			waiter := stubPIWaiter{
				getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
					attempts++
					if attempts == 1 {
						return d.StateActive, d.ProcessInstance{Key: key, State: d.StateActive}, nil
					}
					return d.StateCompleted, d.ProcessInstance{Key: key, State: d.StateCompleted}, nil
				},
			}

			buf := &bytes.Buffer{}
			logger := slog.New(slog.NewTextHandler(buf, nil))
			got, _, err := WaitForProcessInstanceState(
				context.Background(),
				waiter,
				testConfig(time.Nanosecond, 3, 25*time.Millisecond),
				logger,
				"123",
				d.States{d.StateCompleted},
				opts...,
			)

			require.NoError(t, err)
			assert.True(t, got.Ok)
			return buf.String()
		}

		quietLog := run(t)
		verboseLog := run(t, services.WithVerbose())

		assert.NotContains(t, quietLog, "process instance 123 currently in state ACTIVE")
		assert.Contains(t, verboseLog, "process instance 123 currently in state ACTIVE")
	})

	t.Run("uses command activity while waiting for a single instance", func(t *testing.T) {
		t.Parallel()

		sink := &activitysink.Sink{}
		waiter := stubPIWaiter{
			getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
				return d.StateCompleted, d.ProcessInstance{Key: key, State: d.StateCompleted}, nil
			},
		}

		_, _, err := WaitForProcessInstanceState(
			logging.ToActivityContext(context.Background(), sink),
			waiter,
			testConfig(time.Nanosecond, 3, 25*time.Millisecond),
			testLogger(),
			"123",
			d.States{d.StateCompleted},
		)

		require.NoError(t, err)
		started, stopped, msgs := sink.Snapshot()
		assert.Equal(t, 1, started)
		assert.Equal(t, 1, stopped)
		assert.Equal(t, []string{"waiting for process instance 123 to reach desired state(s)"}, msgs)
	})
}

// TestWaitForProcessInstancesState_UsesAggregateCommandActivity verifies aggregate waits expose one shared activity scope.
func TestWaitForProcessInstancesState_UsesAggregateCommandActivity(t *testing.T) {
	t.Parallel()

	sink := &activitysink.Sink{}
	waiter := stubPIWaiter{
		getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
			return d.StateCompleted, d.ProcessInstance{Key: key, State: d.StateCompleted}, nil
		},
	}

	_, err := WaitForProcessInstancesState(
		logging.ToActivityContext(context.Background(), sink),
		waiter,
		testConfig(time.Nanosecond, 3, 25*time.Millisecond),
		testLogger(),
		typex.Keys{"123", "124"},
		d.States{d.StateCompleted},
		1,
	)

	require.NoError(t, err)
	started, stopped, msgs := sink.Snapshot()
	assert.Equal(t, started, stopped)
	assert.Contains(t, msgs, "waiting for 2 process instance(s) to reach desired state(s)")
}

// testConfig builds a waiter config with explicit retry timing for unit tests.
func testConfig(initialDelay time.Duration, maxRetries int, timeout time.Duration) *config.Config {
	return &config.Config{
		App: config.App{
			Backoff: config.BackoffConfig{
				Strategy:     config.BackoffFixed,
				InitialDelay: initialDelay,
				MaxRetries:   maxRetries,
				Timeout:      timeout,
			},
		},
	}
}

// testLogger returns a discard logger for waiter tests.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
