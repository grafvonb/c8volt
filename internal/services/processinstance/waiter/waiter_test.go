package waiter

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPIWaiter struct {
	getStateByKey func(ctx context.Context, key string) (d.State, d.ProcessInstance, error)
}

func (s stubPIWaiter) GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error) {
	panic("unexpected GetProcessInstance call")
}

func (s stubPIWaiter) GetProcessInstanceStateByKey(ctx context.Context, key string, _ ...services.CallOption) (d.State, d.ProcessInstance, error) {
	return s.getStateByKey(ctx, key)
}

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
			testConfig(time.Millisecond, 2, 25*time.Millisecond),
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
}

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

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
