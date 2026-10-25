// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package waiter

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// suspensionReaderFunc lets tests script observations without mutation capabilities.
type suspensionReaderFunc func(context.Context, string, ...services.CallOption) (d.ProcessInstance, error)

// GetProcessInstance delegates to the test's exact-key reader.
func (f suspensionReaderFunc) GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error) {
	return f(ctx, key, opts...)
}

// suspensionBackoff keeps observation timing deterministic inside synctest.
func suspensionBackoff() config.BackoffConfig {
	return config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Second, Timeout: 10 * time.Second, MaxRetries: 3}
}

// TestProcessInstanceSuspensionObservations verifies truthful confirmation, terminal
// conflicts, typed absence, identity validation and retention after failed reads.
func TestProcessInstanceSuspensionObservations(t *testing.T) {
	t.Parallel()
	const key = "2251799813685249"
	type step struct {
		state    d.State
		date     string
		err      error
		wrongKey bool
	}
	tests := []struct {
		name    string
		steps   []step
		want    SuspensionObservation
		wantErr error
	}{
		{"active then suspended", []step{{state: d.StateActive}, {state: d.StateSuspended, date: "date"}}, SuspensionObservation{true, d.StateSuspended, "date"}, nil},
		{"suspended without date", []step{{state: d.StateSuspended}}, SuspensionObservation{true, d.StateSuspended, ""}, nil},
		{"completed", []step{{state: d.StateCompleted}}, SuspensionObservation{true, d.StateCompleted, ""}, d.ErrConflict},
		{"canceled", []step{{state: d.StateCanceled}}, SuspensionObservation{true, d.StateCanceled, ""}, d.ErrConflict},
		{"terminated", []step{{state: d.StateTerminated}}, SuspensionObservation{true, d.StateTerminated, ""}, d.ErrConflict},
		{"date alone", []step{{state: d.StateActive, date: "date"}, {state: d.StateActive, date: "date"}, {state: d.StateActive, date: "date"}}, SuspensionObservation{true, d.StateActive, "date"}, d.ErrGatewayTimeout},
		{"unknown", []step{{state: d.StateUnknown}, {err: d.ErrForbidden}}, SuspensionObservation{true, d.StateUnknown, ""}, d.ErrForbidden},
		{"unrecognized", []step{{state: "FUTURE"}, {err: d.ErrForbidden}}, SuspensionObservation{true, d.StateUnknown, ""}, d.ErrForbidden},
		{"first read failure", []step{{err: d.ErrForbidden}}, SuspensionObservation{}, d.ErrForbidden},
		{"false absence text", []step{{err: fmt.Errorf("404 not found: %w", d.ErrForbidden)}}, SuspensionObservation{}, d.ErrForbidden},
		{"denial before absence", []step{{err: errors.Join(d.ErrNotFound, d.ErrUnauthorized)}}, SuspensionObservation{}, d.ErrUnauthorized},
		{"preserve prior date", []step{{state: d.StateActive, date: "prior"}, {err: d.ErrForbidden}}, SuspensionObservation{true, d.StateActive, "prior"}, d.ErrForbidden},
		{"absence clears date", []step{{state: d.StateActive, date: "prior"}, {err: fmt.Errorf("lookup: %w", d.ErrNotFound)}, {err: d.ErrForbidden}}, SuspensionObservation{true, d.StateAbsent, ""}, d.ErrForbidden},
		{"absence continues", []step{{err: d.ErrNotFound}, {state: d.StateSuspended}}, SuspensionObservation{true, d.StateSuspended, ""}, nil},
		{"absence exhausts", []step{{err: d.ErrNotFound}, {err: d.ErrNotFound}, {err: d.ErrNotFound}}, SuspensionObservation{true, d.StateAbsent, ""}, d.ErrGatewayTimeout},
		{"wrong identity", []step{{state: d.StateSuspended, wrongKey: true}}, SuspensionObservation{}, d.ErrMalformedResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				calls := 0
				reader := suspensionReaderFunc(func(ctx context.Context, gotKey string, opts ...services.CallOption) (d.ProcessInstance, error) {
					require.Equal(t, key, gotKey)
					require.True(t, services.ApplyCallOptions(opts).SuppressNestedProcessInstanceLookupLogs)
					require.Less(t, calls, len(tt.steps))
					s := tt.steps[calls]
					calls++
					pi := d.ProcessInstance{Key: key, State: s.state, SuspendedDate: s.date}
					if s.wrongKey {
						pi.Key = ""
					}
					return pi, s.err
				})
				got, err := WaitForProcessInstanceSuspension(context.Background(), reader, suspensionBackoff(), key)
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, tt.want, got)
				require.Equal(t, len(tt.steps), calls)
			})
		})
	}
}

// TestProcessInstanceSuspensionDeadline bounds in-flight reads by the earlier
// caller/configured deadline and retains observations on interruption.
func TestProcessInstanceSuspensionDeadline(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"configured", "caller", "cancel", "already canceled", "invalid timing"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				cfg := suspensionBackoff()
				expected := 10 * time.Second
				wantErr := context.DeadlineExceeded
				switch mode {
				case "caller":
					var stop context.CancelFunc
					ctx, stop = context.WithTimeout(ctx, 2*time.Second)
					defer stop()
					expected = 2 * time.Second
				case "cancel":
					wantErr = context.Canceled
					expected = time.Second
				case "already canceled":
					cancel()
					expected = 0
					wantErr = context.Canceled
				case "invalid timing":
					cfg.Timeout = 0
					expected = 0
					wantErr = d.ErrValidation
				}
				calls := 0
				start := time.Now()
				reader := suspensionReaderFunc(func(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error) {
					calls++
					if mode == "cancel" && calls == 1 {
						return d.ProcessInstance{Key: key, State: d.StateActive, SuspendedDate: "prior"}, nil
					}
					if mode == "cancel" {
						cancel()
					}
					<-ctx.Done()
					return d.ProcessInstance{}, ctx.Err()
				})
				got, err := WaitForProcessInstanceSuspension(ctx, reader, cfg, "2251799813685249")
				require.ErrorIs(t, err, wantErr)
				require.Equal(t, expected, time.Since(start))
				if mode == "cancel" {
					require.Equal(t, SuspensionObservation{true, d.StateActive, "prior"}, got)
				}
				if mode == "already canceled" || mode == "invalid timing" {
					require.Zero(t, calls)
				}
			})
		})
	}
}
