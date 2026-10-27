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

// resumptionReaderFunc lets tests script observations without mutation capabilities.
type resumptionReaderFunc func(context.Context, string, ...services.CallOption) (d.ProcessInstance, error)

// GetProcessInstance delegates to the test's exact-key reader.
func (f resumptionReaderFunc) GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error) {
	return f(ctx, key, opts...)
}

// resumptionBackoff keeps observation timing deterministic inside synctest.
func resumptionBackoff() config.BackoffConfig {
	return config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Second, Timeout: 10 * time.Second, MaxRetries: 3}
}

// TestProcessInstanceResumptionObservations verifies truthful confirmation, terminal
// conflicts, typed absence, identity validation and retention after failed reads.
func TestProcessInstanceResumptionObservations(t *testing.T) {
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
		want    ResumptionObservation
		wantErr error
	}{
		{"suspended then active", []step{{state: d.StateSuspended}, {state: d.StateActive, date: "date"}}, ResumptionObservation{true, d.StateActive, "date"}, nil},
		{"active without date", []step{{state: d.StateActive}}, ResumptionObservation{true, d.StateActive, ""}, nil},
		{"completed", []step{{state: d.StateCompleted}}, ResumptionObservation{true, d.StateCompleted, ""}, nil},
		{"canceled", []step{{state: d.StateCanceled}}, ResumptionObservation{true, d.StateCanceled, ""}, d.ErrConflict},
		{"terminated", []step{{state: d.StateTerminated}}, ResumptionObservation{true, d.StateTerminated, ""}, d.ErrConflict},
		{"date alone", []step{{state: d.StateSuspended, date: "date"}, {state: d.StateSuspended, date: "date"}, {state: d.StateSuspended, date: "date"}}, ResumptionObservation{true, d.StateSuspended, "date"}, d.ErrGatewayTimeout},
		{"unknown", []step{{state: d.StateUnknown}, {err: d.ErrForbidden}}, ResumptionObservation{true, d.StateUnknown, ""}, d.ErrForbidden},
		{"unrecognized", []step{{state: "FUTURE"}, {err: d.ErrForbidden}}, ResumptionObservation{true, d.StateUnknown, ""}, d.ErrForbidden},
		{"first read failure", []step{{err: d.ErrForbidden}}, ResumptionObservation{}, d.ErrForbidden},
		{"false absence text", []step{{err: fmt.Errorf("404 not found: %w", d.ErrForbidden)}}, ResumptionObservation{}, d.ErrForbidden},
		{"denial before absence", []step{{err: errors.Join(d.ErrNotFound, d.ErrUnauthorized)}}, ResumptionObservation{}, d.ErrUnauthorized},
		{"preserve prior date", []step{{state: d.StateSuspended, date: "prior"}, {err: d.ErrForbidden}}, ResumptionObservation{true, d.StateSuspended, "prior"}, d.ErrForbidden},
		{"absence clears date", []step{{state: d.StateSuspended, date: "prior"}, {err: fmt.Errorf("lookup: %w", d.ErrNotFound)}, {err: d.ErrForbidden}}, ResumptionObservation{true, d.StateAbsent, ""}, d.ErrForbidden},
		{"absence continues", []step{{err: d.ErrNotFound}, {state: d.StateActive}}, ResumptionObservation{true, d.StateActive, ""}, nil},
		{"absence exhausts", []step{{err: d.ErrNotFound}, {err: d.ErrNotFound}, {err: d.ErrNotFound}}, ResumptionObservation{true, d.StateAbsent, ""}, d.ErrGatewayTimeout},
		{"successful omission clears date", []step{{state: d.StateSuspended, date: "prior"}, {state: d.StateSuspended}, {err: d.ErrForbidden}}, ResumptionObservation{true, d.StateSuspended, ""}, d.ErrForbidden},
		{"active clears stale date", []step{{state: d.StateSuspended, date: "prior"}, {state: d.StateActive}}, ResumptionObservation{true, d.StateActive, ""}, nil},
		{"completed clears stale date", []step{{state: d.StateSuspended, date: "prior"}, {state: d.StateCompleted}}, ResumptionObservation{true, d.StateCompleted, ""}, nil},
		{"precondition retains evidence", []step{{state: d.StateSuspended, date: "prior"}, {err: d.ErrPrecondition}}, ResumptionObservation{true, d.StateSuspended, "prior"}, d.ErrPrecondition},
		{"read failure retains evidence", []step{{state: d.StateSuspended, date: "prior"}, {err: d.ErrInternal}}, ResumptionObservation{true, d.StateSuspended, "prior"}, d.ErrInternal},
		{"suspended without date cannot confirm", []step{{state: d.StateSuspended}, {state: d.StateSuspended}, {state: d.StateSuspended}}, ResumptionObservation{true, d.StateSuspended, ""}, d.ErrGatewayTimeout},
		{"unknown continues", []step{{state: d.StateUnknown}, {state: d.StateActive}}, ResumptionObservation{true, d.StateActive, ""}, nil},
		{"wrong identity preserves evidence", []step{{state: d.StateSuspended, date: "prior"}, {state: d.StateActive, wrongKey: true}}, ResumptionObservation{true, d.StateSuspended, "prior"}, d.ErrMalformedResponse},
		{"wrong identity", []step{{state: d.StateActive, wrongKey: true}}, ResumptionObservation{}, d.ErrMalformedResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				calls := 0
				reader := resumptionReaderFunc(func(ctx context.Context, gotKey string, opts ...services.CallOption) (d.ProcessInstance, error) {
					require.Equal(t, key, gotKey)
					require.True(t, services.ApplyCallOptions(opts).SuppressNestedProcessInstanceLookupLogs)
					require.True(t, services.ApplyCallOptions(opts).SuppressProcessInstanceDetailLogs)
					require.Less(t, calls, len(tt.steps))
					s := tt.steps[calls]
					calls++
					pi := d.ProcessInstance{Key: key, State: s.state, SuspendedDate: s.date}
					if s.wrongKey {
						pi.Key = ""
					}
					return pi, s.err
				})
				got, err := WaitForProcessInstanceResumption(context.Background(), reader, resumptionBackoff(), key)
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, tt.want, got)
				require.Equal(t, len(tt.steps), calls)
			})
		})
	}
}

// TestProcessInstanceResumptionDeadline bounds in-flight reads by the earlier
// caller/configured deadline and retains observations on interruption.
func TestProcessInstanceResumptionDeadline(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"configured", "caller", "cancel", "already canceled", "invalid timing"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				cfg := resumptionBackoff()
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
				reader := resumptionReaderFunc(func(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error) {
					calls++
					if mode == "cancel" && calls == 1 {
						return d.ProcessInstance{Key: key, State: d.StateSuspended, SuspendedDate: "prior"}, nil
					}
					if mode == "cancel" {
						cancel()
					}
					<-ctx.Done()
					return d.ProcessInstance{}, ctx.Err()
				})
				got, err := WaitForProcessInstanceResumption(ctx, reader, cfg, "2251799813685249")
				require.ErrorIs(t, err, wantErr)
				require.Equal(t, expected, time.Since(start))
				if mode == "cancel" {
					require.Equal(t, ResumptionObservation{true, d.StateSuspended, "prior"}, got)
				}
				if mode == "already canceled" || mode == "invalid timing" {
					require.Zero(t, calls)
				}
			})
		})
	}
}

// TestProcessInstanceResumptionLateRead rejects even a matching success returned
// after the deadline and retains only observations completed within the bound.
func TestProcessInstanceResumptionLateRead(t *testing.T) {
	t.Parallel()
	for _, state := range []d.State{d.StateActive, d.StateCompleted} {
		t.Run(string(state), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				calls := 0
				reader := resumptionReaderFunc(func(ctx context.Context, key string, _ ...services.CallOption) (d.ProcessInstance, error) {
					calls++
					if calls == 1 {
						return d.ProcessInstance{Key: key, State: d.StateSuspended, SuspendedDate: "prior"}, nil
					}
					<-ctx.Done()
					return d.ProcessInstance{Key: key, State: state}, nil
				})
				got, err := WaitForProcessInstanceResumption(context.Background(), reader, resumptionBackoff(), "2251799813685249")
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.Equal(t, ResumptionObservation{true, d.StateSuspended, "prior"}, got)
				require.Equal(t, 2, calls)
			})
		})
	}
}
