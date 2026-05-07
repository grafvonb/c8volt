// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package waiter

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// stubVariableWaiter lets tests control variable search responses.
type stubVariableWaiter struct {
	search func(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceVariable, error)
}

func (s stubVariableWaiter) SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	return s.search(ctx, key, opts...)
}

func TestWaitForProcessInstanceVariables_WaitsUntilRequestedValuesAreVisible(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	got, err := WaitForProcessInstanceVariables(
		context.Background(),
		stubVariableWaiter{
			search: func(_ context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
				require.Equal(t, "123", key)
				if attempts.Add(1) == 1 {
					return []d.ProcessInstanceVariable{
						{Name: "hasIncident", Value: `false`, ProcessInstanceKey: "123", ScopeKey: "123"},
					}, nil
				}
				return []d.ProcessInstanceVariable{
					{Name: "hasIncident", Value: `true`, ProcessInstanceKey: "123", ScopeKey: "123"},
				}, nil
			},
		},
		testConfig(time.Millisecond, 3, time.Second),
		"123",
		map[string]any{"hasIncident": true},
	)

	require.NoError(t, err)
	require.Empty(t, got)
	require.Equal(t, int32(2), attempts.Load())
}

func TestMissingRequestedVariables_NormalizedJSONAndScopeFiltering(t *testing.T) {
	t.Parallel()

	got := MissingRequestedVariables("123", map[string]any{
		"foo":    "bar",
		"nested": map[string]any{"count": float64(2)},
	}, []d.ProcessInstanceVariable{
		{Name: "foo", Value: `"bar"`, ProcessInstanceKey: "123", ScopeKey: "123"},
		{Name: "nested", Value: `{"count":2}`, ProcessInstanceKey: "123", ScopeKey: "123"},
		{Name: "foo", Value: `"wrong-scope"`, ProcessInstanceKey: "123", ScopeKey: "element-1"},
	})

	require.Empty(t, got)
}

// testConfig builds a variable waiter config with explicit retry timing.
func testConfig(initialDelay time.Duration, maxRetries int, timeout time.Duration) *config.Config {
	return &config.Config{App: config.App{
		Backoff: config.BackoffConfig{
			Strategy:     config.BackoffFixed,
			InitialDelay: initialDelay,
			MaxDelay:     initialDelay,
			MaxRetries:   maxRetries,
			Timeout:      timeout,
		},
	}}
}
