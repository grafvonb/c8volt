// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package waiter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
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
	observeOptions     func(opts []services.CallOption)
}

// GetProcessInstance observes the supplied options and delegates to the configured fixture; unexpected calls panic.
func (s stubPIWaiter) GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error) {
	if s.observeOptions != nil {
		s.observeOptions(opts)
	}
	if s.getProcessInstance == nil {
		panic("unexpected GetProcessInstance call")
	}
	return s.getProcessInstance(ctx, key)
}

// GetProcessInstanceStateByKey observes the supplied options before executing the configured state lookup.
func (s stubPIWaiter) GetProcessInstanceStateByKey(ctx context.Context, key string, opts ...services.CallOption) (d.State, d.ProcessInstance, error) {
	if s.observeOptions != nil {
		s.observeOptions(opts)
	}
	return s.getStateByKey(ctx, key)
}

// Incident=true requires polling full PI details because the state endpoint cannot prove the incident marker.
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

// Missing process instances must keep retrying for incident=false instead of being treated as incident-free.
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

// Combined expectations protect against separately observed state and incident matches on different polls.
func TestWaitForProcessInstanceExpectation_StateAndIncidentCompatibility(t *testing.T) {
	t.Run("requires state and incident to match on the same present instance", func(t *testing.T) {
		t.Parallel()

		wantIncident := true
		attempts := 0
		waiter := stubPIWaiter{
			getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
				attempts++
				switch attempts {
				case 1:
					return d.ProcessInstance{Key: key, State: d.StateActive, Incident: false}, nil
				case 2:
					return d.ProcessInstance{Key: key, State: d.StateCompleted, Incident: true}, nil
				default:
					return d.ProcessInstance{Key: key, State: d.StateActive, Incident: true}, nil
				}
			},
		}

		got, pi, err := WaitForProcessInstanceExpectation(
			context.Background(),
			waiter,
			testConfig(time.Millisecond, 4, 100*time.Millisecond),
			testLogger(),
			"123",
			d.ProcessInstanceExpectationRequest{
				States:   d.States{d.StateActive},
				Incident: &wantIncident,
			},
		)

		require.NoError(t, err)
		assert.Equal(t, 3, attempts)
		assert.True(t, got.Ok)
		assert.Equal(t, d.StateActive, got.State)
		require.NotNil(t, got.Incident)
		assert.True(t, *got.Incident)
		assert.Equal(t, d.ProcessInstance{Key: "123", State: d.StateActive, Incident: true}, pi)
	})

	t.Run("preserves canceled and terminated compatibility", func(t *testing.T) {
		t.Parallel()

		wantIncident := true
		waiter := stubPIWaiter{
			getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
				return d.ProcessInstance{Key: key, State: d.StateTerminated, Incident: true}, nil
			},
		}

		got, pi, err := WaitForProcessInstanceExpectation(
			context.Background(),
			waiter,
			testConfig(time.Millisecond, 2, 100*time.Millisecond),
			testLogger(),
			"123",
			d.ProcessInstanceExpectationRequest{
				States:   d.States{d.StateCanceled},
				Incident: &wantIncident,
			},
		)

		require.NoError(t, err)
		assert.True(t, got.Ok)
		assert.Equal(t, d.StateTerminated, got.State)
		require.NotNil(t, got.Incident)
		assert.True(t, *got.Incident)
		assert.Equal(t, d.ProcessInstance{Key: "123", State: d.StateTerminated, Incident: true}, pi)
	})

	t.Run("does not let state absent satisfy an incident expectation", func(t *testing.T) {
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
			d.ProcessInstanceExpectationRequest{
				States:   d.States{d.StateAbsent},
				Incident: &wantIncident,
			},
		)

		require.Error(t, err)
		assert.Equal(t, 2, attempts)
		assert.False(t, got.Ok)
		assert.Equal(t, d.StateUnknown, got.State)
		assert.Nil(t, got.Incident)
		assert.Contains(t, got.Status, "exceeded max_retries (2)")
		assert.Equal(t, d.ProcessInstance{}, pi)
	})
}

// TestWaitForProcessInstanceExpectation_ExplicitCanceledCompatibility keeps explicit canceled matching narrower than cancellation cleanup acceptance.
func TestWaitForProcessInstanceExpectation_ExplicitCanceledCompatibility(t *testing.T) {
	wantIncident := true
	tests := []struct {
		name       string
		state      d.State
		incident   bool
		err        error
		wantOK     bool
		wantState  d.State
		wantCalls  int
		wantResult d.ProcessInstance
	}{
		{
			name:       "accepts canceled with the required incident",
			state:      d.StateCanceled,
			incident:   true,
			wantOK:     true,
			wantState:  d.StateCanceled,
			wantCalls:  1,
			wantResult: d.ProcessInstance{Key: "123", State: d.StateCanceled, Incident: true},
		},
		{
			name:       "accepts terminated with the required incident",
			state:      d.StateTerminated,
			incident:   true,
			wantOK:     true,
			wantState:  d.StateTerminated,
			wantCalls:  1,
			wantResult: d.ProcessInstance{Key: "123", State: d.StateTerminated, Incident: true},
		},
		{
			name:      "rejects completed",
			state:     d.StateCompleted,
			incident:  true,
			wantState: d.StateUnknown,
			wantCalls: 2,
		},
		{
			name:      "rejects absent",
			err:       d.ErrNotFound,
			wantState: d.StateUnknown,
			wantCalls: 2,
		},
		{
			name:      "still requires the requested incident",
			state:     d.StateCanceled,
			wantState: d.StateUnknown,
			wantCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			waiter := stubPIWaiter{
				getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
					calls++
					if tt.err != nil {
						return d.ProcessInstance{}, tt.err
					}
					return d.ProcessInstance{Key: key, State: tt.state, Incident: tt.incident}, nil
				},
			}

			got, pi, err := WaitForProcessInstanceExpectation(
				context.Background(),
				waiter,
				testConfig(time.Millisecond, 2, 100*time.Millisecond),
				testLogger(),
				"123",
				d.ProcessInstanceExpectationRequest{
					States:   d.States{d.StateCanceled},
					Incident: &wantIncident,
				},
			)

			assert.Equal(t, tt.wantCalls, calls)
			assert.Equal(t, tt.wantOK, got.Ok)
			assert.Equal(t, tt.wantState, got.State)
			assert.Equal(t, tt.wantResult, pi)
			if tt.wantOK {
				require.NoError(t, err)
				require.NotNil(t, got.Incident)
				assert.True(t, *got.Incident)
				assert.Contains(t, got.Status, "satisfied expectation(s)")
				return
			}
			require.Error(t, err)
			assert.Nil(t, got.Incident)
			assert.Contains(t, got.Status, "exceeded max_retries (2)")
		})
	}
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

		assert.NotContains(t, quietLog, "pi 123 waiting; state ACTIVE")
		assert.Contains(t, verboseLog, "pi 123 waiting; state ACTIVE")
	})

	t.Run("uses command activity while waiting for a single instance", func(t *testing.T) {
		t.Parallel()

		sink := &activitysink.Sink{}
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
		assert.Equal(t, []string{"waiting for pi 123 state"}, msgs)
		assert.Equal(t, []string{"pi 123 waiting; state ACTIVE, attempt 1"}, sink.Updates())
		assert.Equal(t, []activitysink.Start{{
			Message:    "waiting for pi 123 state",
			Importance: logging.ActivityImportanceWait,
		}}, sink.Starts())
		assert.Equal(t, []activitysink.Update{{
			Message:    "pi 123 waiting; state ACTIVE, attempt 1",
			Importance: logging.ActivityImportanceWait,
		}}, sink.PriorityUpdates())
	})
}

// TestWaitForProcessInstanceStateObservations verifies each actual state lookup has one completed DEBUG observation.
func TestWaitForProcessInstanceStateObservations(t *testing.T) {
	t.Run("retry then success includes a delay only on the continued check", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		waiter := stubPIWaiter{
			getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
				attempts++
				if attempts == 1 {
					return d.StateActive, d.ProcessInstance{Key: key, State: d.StateActive}, nil
				}
				return d.StateCompleted, d.ProcessInstance{Key: key, State: d.StateCompleted}, nil
			},
			observeOptions: func(opts []services.CallOption) {
				assert.True(t, services.ApplyCallOptions(opts).SuppressNestedProcessInstanceLookupLogs)
			},
		}
		var output bytes.Buffer

		got, _, err := WaitForProcessInstanceState(
			context.Background(), waiter, testConfig(time.Nanosecond, 3, time.Second), debugLogger(&output),
			"123", d.States{d.StateCompleted},
		)

		require.NoError(t, err)
		assert.True(t, got.Ok)
		lines := observationLines(output.String())
		require.Len(t, lines, 2)
		assert.Contains(t, lines[0], "key=123 attempt=1 state=ACTIVE")
		assert.Contains(t, lines[0], "next_delay=1ns")
		assert.Contains(t, lines[1], "key=123 attempt=2 state=COMPLETED")
		assert.NotContains(t, lines[1], "next_delay=")
		assert.NotContains(t, output.String(), "fetch state")
	})

	t.Run("absence is a terminal observed state", func(t *testing.T) {
		t.Parallel()

		waiter := stubPIWaiter{getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
			return d.StateUnknown, d.ProcessInstance{}, d.ErrNotFound
		}}
		var output bytes.Buffer

		got, _, err := WaitForProcessInstanceState(
			context.Background(), waiter, testConfig(time.Millisecond, 3, time.Second), debugLogger(&output),
			"missing", d.States{d.StateAbsent},
		)

		require.NoError(t, err)
		assert.Equal(t, d.StateAbsent, got.State)
		lines := observationLines(output.String())
		require.Len(t, lines, 1)
		assert.Contains(t, lines[0], "key=missing attempt=1 state=ABSENT")
		assert.NotContains(t, lines[0], "next_delay=")
	})

	t.Run("failed lookup records a concise outcome once", func(t *testing.T) {
		t.Parallel()

		waiter := stubPIWaiter{getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
			return d.StateUnknown, d.ProcessInstance{}, errors.New("lookup failed: nested transport detail")
		}}
		var output bytes.Buffer

		_, _, err := WaitForProcessInstanceState(
			context.Background(), waiter, testConfig(time.Millisecond, 3, time.Second), debugLogger(&output),
			"123", d.States{d.StateCompleted},
		)

		require.Error(t, err)
		lines := observationLines(output.String())
		require.Len(t, lines, 1)
		assert.Contains(t, lines[0], "key=123 attempt=1 lookup_error=lookup failed")
		assert.NotContains(t, lines[0], "nested transport detail")
		assert.NotContains(t, lines[0], "next_delay=")
	})

	t.Run("interrupted sleep omits a delay because no next check occurs", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		waiter := stubPIWaiter{getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
			cancel()
			return d.StateActive, d.ProcessInstance{Key: key, State: d.StateActive}, nil
		}}
		var output bytes.Buffer

		_, _, err := WaitForProcessInstanceState(
			ctx, waiter, testConfig(time.Second, 3, 0), debugLogger(&output), "123", d.States{d.StateCompleted},
		)

		require.ErrorIs(t, err, context.Canceled)
		lines := observationLines(output.String())
		require.Len(t, lines, 1)
		assert.Contains(t, lines[0], "key=123 attempt=1 state=ACTIVE")
		assert.NotContains(t, lines[0], "next_delay=")
	})

	t.Run("max retries emits the final observation without another delay", func(t *testing.T) {
		t.Parallel()

		waiter := stubPIWaiter{getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
			return d.StateActive, d.ProcessInstance{Key: key, State: d.StateActive}, nil
		}}
		var output bytes.Buffer

		_, _, err := WaitForProcessInstanceState(
			context.Background(), waiter, testConfig(time.Nanosecond, 2, time.Second), debugLogger(&output),
			"123", d.States{d.StateCompleted},
		)

		require.Error(t, err)
		lines := observationLines(output.String())
		require.Len(t, lines, 2)
		assert.Contains(t, lines[0], "attempt=1 state=ACTIVE")
		assert.Contains(t, lines[0], "next_delay=1ns")
		assert.Contains(t, lines[1], "attempt=2 state=ACTIVE")
		assert.NotContains(t, lines[1], "next_delay=")
	})

	t.Run("wait timeout records the completed check without advertising continuation", func(t *testing.T) {
		t.Parallel()

		waiter := stubPIWaiter{getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
			return d.StateActive, d.ProcessInstance{Key: key, State: d.StateActive}, nil
		}}
		var output bytes.Buffer

		_, _, err := WaitForProcessInstanceState(
			context.Background(), waiter, testConfig(time.Second, 0, 5*time.Millisecond), debugLogger(&output),
			"123", d.States{d.StateCompleted},
		)

		require.ErrorIs(t, err, context.DeadlineExceeded)
		lines := observationLines(output.String())
		require.Len(t, lines, 1)
		assert.Contains(t, lines[0], "attempt=1 state=ACTIVE")
		assert.NotContains(t, lines[0], "next_delay=")
	})

	t.Run("already canceled context performs and records no check", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var output bytes.Buffer
		called := false
		waiter := stubPIWaiter{getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
			called = true
			return d.StateUnknown, d.ProcessInstance{}, nil
		}}

		_, _, err := WaitForProcessInstanceState(
			ctx, waiter, testConfig(time.Millisecond, 3, 0), debugLogger(&output), "123", d.States{d.StateCompleted},
		)

		require.ErrorIs(t, err, context.Canceled)
		assert.False(t, called)
		assert.Empty(t, observationLines(output.String()))
	})
}

// TestWaitForProcessInstanceExpectationObservations verifies full-instance polling uses the same observation budget.
func TestWaitForProcessInstanceExpectationObservations(t *testing.T) {
	t.Parallel()

	wantIncident := true
	attempts := 0
	waiter := stubPIWaiter{
		getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
			attempts++
			return d.ProcessInstance{Key: key, State: d.StateActive, Incident: attempts == 2}, nil
		},
		observeOptions: func(opts []services.CallOption) {
			assert.True(t, services.ApplyCallOptions(opts).SuppressNestedProcessInstanceLookupLogs)
		},
	}
	var output bytes.Buffer

	got, _, err := WaitForProcessInstanceExpectation(
		context.Background(), waiter, testConfig(time.Nanosecond, 3, time.Second), debugLogger(&output),
		"123", d.ProcessInstanceExpectationRequest{Incident: &wantIncident},
	)

	require.NoError(t, err)
	assert.True(t, got.Ok)
	lines := observationLines(output.String())
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "state=ACTIVE")
	assert.Contains(t, lines[0], "next_delay=1ns")
	assert.Contains(t, lines[1], "state=ACTIVE")
	assert.NotContains(t, lines[1], "next_delay=")
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
	assert.Contains(t, msgs, "waiting for 2 pi state")
	assert.Contains(t, sink.Starts(), activitysink.Start{
		Message:    "waiting for 2 pi state",
		Importance: logging.ActivityImportanceWait,
	})
}

func TestWaitForProcessInstancesStateEmitsCompletionFacts(t *testing.T) {
	t.Parallel()

	waiter := stubPIWaiter{
		getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
			if key == "124" {
				return d.StateActive, d.ProcessInstance{Key: key, State: d.StateActive}, errors.New("still active")
			}
			return d.StateCompleted, d.ProcessInstance{Key: key, State: d.StateCompleted}, nil
		},
	}
	var completions []d.OpsCompletionProgress

	_, err := WaitForProcessInstancesState(
		context.Background(),
		waiter,
		testConfig(time.Nanosecond, 1, 25*time.Millisecond),
		testLogger(),
		typex.Keys{"123", "124"},
		d.States{d.StateCompleted},
		1,
		services.WithProgress(func(event d.OpsProgressEvent) {
			if event.Kind == d.OpsProgressEventKindCompletion && event.Completion != nil {
				completions = append(completions, *event.Completion)
			}
		}),
	)

	require.Error(t, err)
	require.Len(t, completions, 2)
	require.Equal(t, d.OpsCompletionProgress{
		Phase:        "expect process instances",
		CoreResource: "process instance(s)",
		Total:        2,
		Identity:     "123",
		Disposition:  d.OpsCompletionDispositionConfirmed,
	}, completions[0])
	require.Equal(t, "expect process instances", completions[1].Phase)
	require.Equal(t, "process instance(s)", completions[1].CoreResource)
	require.Equal(t, 2, completions[1].Total)
	require.Equal(t, "124", completions[1].Identity)
	require.Equal(t, d.OpsCompletionDispositionFailed, completions[1].Disposition)
	require.Contains(t, completions[1].FailureDetail, "still active")
}

func TestWaitForProcessInstancesExpectationEmitsCompletionFacts(t *testing.T) {
	t.Parallel()

	wantIncident := true
	waiter := stubPIWaiter{
		getProcessInstance: func(ctx context.Context, key string) (d.ProcessInstance, error) {
			return d.ProcessInstance{Key: key, State: d.StateActive, Incident: true}, nil
		},
	}
	var completions []d.OpsCompletionProgress

	_, err := WaitForProcessInstancesExpectation(
		context.Background(),
		waiter,
		testConfig(time.Nanosecond, 1, 25*time.Millisecond),
		testLogger(),
		typex.Keys{"123", "124"},
		d.ProcessInstanceExpectationRequest{Incident: &wantIncident},
		1,
		services.WithProgress(func(event d.OpsProgressEvent) {
			if event.Kind == d.OpsProgressEventKindCompletion && event.Completion != nil {
				completions = append(completions, *event.Completion)
			}
		}),
	)

	require.NoError(t, err)
	require.Equal(t, []d.OpsCompletionProgress{
		{
			Phase:        "expect process instances",
			CoreResource: "process instance(s)",
			Total:        2,
			Identity:     "123",
			Disposition:  d.OpsCompletionDispositionConfirmed,
		},
		{
			Phase:        "expect process instances",
			CoreResource: "process instance(s)",
			Total:        2,
			Identity:     "124",
			Disposition:  d.OpsCompletionDispositionConfirmed,
		},
	}, completions)
}

// TestWaitForProcessInstanceState_SingleTargetPollingIsNotWorkflowProgress
// pins single-key waits as transient waiter activity rather than semantic
// workflow completion progress.
func TestWaitForProcessInstanceState_SingleTargetPollingIsNotWorkflowProgress(t *testing.T) {
	t.Parallel()

	sink := &activitysink.Sink{}
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

	_, _, err := WaitForProcessInstanceState(
		logging.ToActivityContext(context.Background(), sink),
		waiter,
		testConfig(time.Nanosecond, 3, 25*time.Millisecond),
		testLogger(),
		"123",
		d.States{d.StateCompleted},
	)

	require.NoError(t, err)
	assert.Equal(t, []activitysink.Start{{
		Message:    "waiting for pi 123 state",
		Importance: logging.ActivityImportanceWait,
	}}, sink.Starts())
	assert.Equal(t, []activitysink.Update{{
		Message:    "pi 123 waiting; state ACTIVE, attempt 1",
		Importance: logging.ActivityImportanceWait,
	}}, sink.PriorityUpdates())
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

// debugLogger captures DEBUG observations for exact polling-budget assertions.
func debugLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// observationLines extracts only waiter-owned observation records from captured structured logs.
func observationLines(output string) []string {
	var lines []string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "pi state observation:") {
			lines = append(lines, line)
		}
	}
	return lines
}

// TestWaitForProcessInstanceStateMutationOwnsLookupFailure cancels inside a lookup
// to verify that caller-owned failure reporting suppresses the duplicate ERROR
// while retaining one polling observation and the original cancellation cause.
func TestWaitForProcessInstanceStateMutationOwnsLookupFailure(t *testing.T) {
	for _, suppressed := range []bool{false, true} {
		t.Run(fmt.Sprint(suppressed), func(t *testing.T) {
			var output bytes.Buffer
			log := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s := stubPIWaiter{getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
				cancel() // Cancellation happens inside the lookup, never in the sleep branch.
				return d.StateUnknown, d.ProcessInstance{}, ctx.Err()
			}}
			var opts []services.CallOption
			if suppressed {
				opts = append(opts, services.WithSuppressProcessInstanceDetailLogs())
			}
			_, _, err := WaitForProcessInstanceState(ctx, s, testConfig(time.Millisecond, 2, time.Second), log, "root", d.States{d.StateCanceled}, opts...)
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, 1, strings.Count(output.String(), "pi state observation:"))
			require.Equal(t, !suppressed, strings.Contains(output.String(), "level=ERROR"))
		})
	}
}
