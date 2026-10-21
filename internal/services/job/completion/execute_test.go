// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package completion

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// preparedForExecution obtains genuine frozen plans so execution tests exercise the preparation contract.
func preparedForExecution(t *testing.T, count int) d.PreparedCompletionBatch {
	t.Helper()
	keys := make([]string, count)
	for i := range keys {
		keys[i] = fmt.Sprint(i + 1)
	}
	b, err := Prepare(context.Background(), d.CompletionRequest{Selection: d.CompletionSelection{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputStdin, Keys: keys}}, "tenant", config.BackoffConfig{}, ReadPorts{Get: func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
		return d.Job{Key: key, State: "CREATED", Kind: "BPMN_ELEMENT", Type: "work", TenantId: "tenant"}, nil
	}})
	require.NoError(t, err)
	return b
}

// assertExecutionCounts ensures every frozen key contributes to exactly one ordered execution row.
func assertExecutionCounts(t *testing.T, b d.PreparedCompletionBatch, r d.CompletionBatchReport) {
	t.Helper()
	require.Equal(t, len(b.Candidates), r.Counts.Selected)
	require.Equal(t, r.Counts.Selected, r.Counts.Accepted+r.Counts.Completed+r.Counts.Unsatisfied+r.Counts.NotStarted)
	require.Len(t, r.Outcomes, r.Counts.Selected)
	for i, o := range r.Outcomes {
		require.Equal(t, b.Candidates[i], o.Candidate)
	}
	require.Nil(t, r.Candidates)
}

// TestBulkCompletionExecuteGuards rejects corrupted plans before any read or mutation, even under bypass options.
func TestBulkCompletionExecuteGuards(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*d.PreparedCompletionBatch){
		"incomplete":        func(b *d.PreparedCompletionBatch) { b.Status = d.CompletionPreparationIncomplete },
		"reason":            func(b *d.PreparedCompletionBatch) { b.CompletionReason = d.CompletionReasonExhausted },
		"duplicate":         func(b *d.PreparedCompletionBatch) { b.Candidates = append(b.Candidates, b.Candidates[0]) },
		"noncanonical key":  func(b *d.PreparedCompletionBatch) { b.Candidates[0].Key = "01" },
		"wrong key":         func(b *d.PreparedCompletionBatch) { b.Candidates[0].Key = "9" },
		"missing candidate": func(b *d.PreparedCompletionBatch) { b.Candidates = nil },
		"selection mode":    func(b *d.PreparedCompletionBatch) { b.Selection.Mode = "bad" },
		"mixed selection":   func(b *d.PreparedCompletionBatch) { b.Selection.Filter.Type = "work" },
		"changed tenant":    func(b *d.PreparedCompletionBatch) { b.DiscoveryTenant = "other" },
		"tenant evidence":   func(b *d.PreparedCompletionBatch) { b.TenantIDs = nil },
		"observed count":    func(b *d.PreparedCompletionBatch) { b.ObservedCount = 0 },
		"pages":             func(b *d.PreparedCompletionBatch) { b.PageCount = -1 },
		"limited":           func(b *d.PreparedCompletionBatch) { b.Limited = true },
		"payload":           func(b *d.PreparedCompletionBatch) { b.Variables = map[string]any{"x": make(chan int)} },
		"listener payload": func(b *d.PreparedCompletionBatch) {
			b.Candidates[0].Kind = "TASK_LISTENER"
			b.Variables = map[string]any{"x": true}
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			b := preparedForExecution(t, 1)
			change(&b)
			var calls atomic.Int32
			ports := ExecutionPorts{Get: func(context.Context, string, ...services.CallOption) (d.Job, error) {
				calls.Add(1)
				return d.Job{}, nil
			}, Submit: func(context.Context, d.Job, map[string]any) (d.CompletionSubmission, error) {
				calls.Add(1)
				return d.CompletionSubmission{}, nil
			}}
			r, err := Execute(context.Background(), b, "tenant", config.BackoffConfig{}, ports, 1, services.WithForce(), services.WithNoStateCheck(), services.WithIgnoreTenant())
			require.Error(t, err)
			require.Equal(t, d.CompletionBatchPreparationFailed, r.Status)
			require.Zero(t, calls.Load())
			require.Nil(t, r.Outcomes)
		})
	}
}

// TestBulkCompletionExecutePreviewAndEmpty proves dry-run never touches execution ports and cancellation still fails empty runs.
func TestBulkCompletionExecutePreviewAndEmpty(t *testing.T) {
	t.Parallel()
	b := preparedForExecution(t, 2)
	r, err := Execute(context.Background(), b, "tenant", config.BackoffConfig{}, ExecutionPorts{}, 2, services.WithDryRun())
	require.NoError(t, err)
	require.Equal(t, d.CompletionBatchPreview, r.Status)
	require.Equal(t, b.Candidates, r.Candidates)
	require.Nil(t, r.Outcomes)
	r.Candidates[0].Key = "changed"
	r.Selection.Keys[0] = "changed"
	r.TenantIDs[0] = "changed"
	require.Equal(t, "1", b.Candidates[0].Key)
	require.Equal(t, "1", b.Selection.Keys[0])
	require.Equal(t, "tenant", b.TenantIDs[0])
	b = preparedForExecution(t, 0)
	r, err = Execute(context.Background(), b, "tenant", config.BackoffConfig{}, ExecutionPorts{}, 1, services.WithNoWait())
	require.NoError(t, err)
	require.Equal(t, d.CompletionBatchFinished, r.Status)
	assertExecutionCounts(t, b, r)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, err = Execute(ctx, b, "tenant", config.BackoffConfig{}, ExecutionPorts{}, 1)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, d.CompletionBatchInterrupted, r.Status)
}

// TestBulkCompletionExecuteRechecks keeps state, scope and payload guards independent of force and retries.
func TestBulkCompletionExecuteRechecks(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, state string
		readErr     error
		mutate      func(*d.Job)
		reason      d.CompletionOutcomeReason
		disposition d.CompletionDisposition
		observation d.CompletionObservationStatus
		writes      int
	}{
		{name: "accepted zero retries", state: "CREATED", disposition: d.CompletionDispositionAccepted, observation: d.CompletionObservationSkipped, writes: 1},
		{name: "failed", state: "FAILED", disposition: d.CompletionDispositionAccepted, observation: d.CompletionObservationSkipped, writes: 1},
		{name: "timed out", state: "TIMED_OUT", disposition: d.CompletionDispositionAccepted, observation: d.CompletionObservationSkipped, writes: 1},
		{name: "retries updated", state: "RETRIES_UPDATED", disposition: d.CompletionDispositionAccepted, observation: d.CompletionObservationSkipped, writes: 1},
		{name: "migrated", state: "MIGRATED", disposition: d.CompletionDispositionAccepted, observation: d.CompletionObservationSkipped, writes: 1},
		{name: "completed despite changed scope", state: "COMPLETED", mutate: func(j *d.Job) { j.Type = "other" }, disposition: d.CompletionDispositionCompleted, observation: d.CompletionObservationCompleted},
		{name: "canceled", state: "CANCELED", reason: d.CompletionOutcomeOtherTerminal, observation: d.CompletionObservationNotCompleted},
		{name: "error thrown", state: "ERROR_THROWN", reason: d.CompletionOutcomeOtherTerminal, observation: d.CompletionObservationNotCompleted},
		{name: "unknown", state: "NEW_STATE", reason: d.CompletionOutcomeUnsupportedState, observation: d.CompletionObservationUnknown},
		{name: "scope", state: "CREATED", mutate: func(j *d.Job) { j.Type = "other" }, reason: d.CompletionOutcomeScopeChanged, observation: d.CompletionObservationNotCompleted},
		{name: "tenant", state: "CREATED", mutate: func(j *d.Job) { j.TenantId = "other" }, reason: d.CompletionOutcomeScopeChanged, observation: d.CompletionObservationNotCompleted},
		{name: "listener payload", state: "CREATED", mutate: func(j *d.Job) { j.Kind = "TASK_LISTENER" }, reason: d.CompletionOutcomePayloadIncompatible, observation: d.CompletionObservationNotCompleted},
		{name: "missing", readErr: d.ErrNotFound, reason: d.CompletionOutcomeMissing, observation: d.CompletionObservationMissing},
		{name: "unreadable", readErr: d.ErrUnavailable, reason: d.CompletionOutcomeUnreadable, observation: d.CompletionObservationUnreadable},
		{name: "wrong lookup key", state: "COMPLETED", mutate: func(j *d.Job) { j.Key = "99" }, reason: d.CompletionOutcomeUnreadable, observation: d.CompletionObservationUnreadable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := preparedForExecution(t, 1)
			b.Selection = d.CompletionSelection{Mode: d.CompletionSelectionSearch, InputForm: d.CompletionInputFilters, BatchSize: 10, Filter: d.CompletionFilter{Type: "work"}}
			b.CompletionReason = d.CompletionReasonExhausted
			b.PageCount = 1
			b.Variables = map[string]any{"value": true}
			reads, writes := 0, 0
			ports := ExecutionPorts{Get: func(_ context.Context, key string, opts ...services.CallOption) (d.Job, error) {
				reads++
				require.True(t, services.ApplyCallOptions(opts).Force)
				j := d.Job{Key: key, State: tc.state, Type: "work", TenantId: "tenant", Kind: "BPMN_ELEMENT"}
				if tc.mutate != nil {
					tc.mutate(&j)
				}
				return j, tc.readErr
			}, Submit: func(_ context.Context, j d.Job, v map[string]any) (d.CompletionSubmission, error) {
				writes++
				require.Equal(t, tc.state, j.State)
				require.Zero(t, j.Retries)
				require.Equal(t, b.Variables, v)
				return d.CompletionSubmission{Status: d.CompletionSubmissionAccepted}, nil
			}}
			r, err := Execute(context.Background(), b, "tenant", config.BackoffConfig{}, ports, 1, services.WithForce(), services.WithNoStateCheck(), services.WithIgnoreTenant())
			if tc.reason != "" {
				require.Error(t, err)
				require.Equal(t, d.CompletionDispositionUnsatisfied, r.Outcomes[0].Disposition)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.disposition, r.Outcomes[0].Disposition)
			}
			require.Equal(t, tc.reason, r.Outcomes[0].Reason)
			require.Equal(t, tc.observation, r.Outcomes[0].Observation.Status)
			require.Equal(t, tc.writes, writes)
			require.Equal(t, 1, reads)
			assertExecutionCounts(t, b, r)
		})
	}
}

// TestBulkCompletionExecuteContinueAndFailFast preserves all rows and never substitutes new targets after rejection.
func TestBulkCompletionExecuteContinueAndFailFast(t *testing.T) {
	t.Parallel()
	for _, fast := range []bool{false, true} {
		t.Run(fmt.Sprint(fast), func(t *testing.T) {
			b := preparedForExecution(t, 4)
			var submitted []string
			ports := ExecutionPorts{Get: func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
				return d.Job{Key: key, State: "CREATED"}, nil
			}, Submit: func(_ context.Context, j d.Job, _ map[string]any) (d.CompletionSubmission, error) {
				submitted = append(submitted, j.Key)
				if j.Key == "2" {
					return d.CompletionSubmission{Status: d.CompletionSubmissionRejected}, d.ErrForbidden
				}
				return d.CompletionSubmission{Status: d.CompletionSubmissionAccepted}, nil
			}}
			var opts []services.CallOption
			if fast {
				opts = append(opts, services.WithFailFast())
			}
			r, err := Execute(context.Background(), b, "tenant", config.BackoffConfig{}, ports, 1, opts...)
			require.ErrorIs(t, err, d.ErrForbidden)
			assertExecutionCounts(t, b, r)
			if fast {
				require.Equal(t, []string{"1", "2"}, submitted)
				require.Equal(t, 2, r.Counts.NotStarted)
				require.Equal(t, d.CompletionOutcomeFailFast, r.Outcomes[2].Reason)
			} else {
				require.Equal(t, []string{"1", "2", "3", "4"}, submitted)
				require.Equal(t, 3, r.Counts.Accepted)
			}
		})
	}
}

// TestBulkCompletionExecuteInflight keeps caller-owned in-flight work alive when fail-fast closes scheduling.
func TestBulkCompletionExecuteInflight(t *testing.T) {
	t.Parallel()
	b := preparedForExecution(t, 5)
	bothStarted := make(chan struct{})
	failed := make(chan struct{})
	var started atomic.Int32
	ports := ExecutionPorts{Get: func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
		return d.Job{Key: key, State: "CREATED"}, nil
	}, Submit: func(ctx context.Context, j d.Job, _ map[string]any) (d.CompletionSubmission, error) {
		if started.Add(1) == 2 {
			close(bothStarted)
		}
		select {
		case <-bothStarted:
		case <-ctx.Done():
			return d.CompletionSubmission{}, ctx.Err()
		}
		if j.Key == "1" {
			close(failed)
			return d.CompletionSubmission{Status: d.CompletionSubmissionRejected}, d.ErrForbidden
		}
		<-failed
		// Allow the pool to observe failure before completing the other request.
		time.Sleep(20 * time.Millisecond)
		if ctx.Err() != nil {
			return d.CompletionSubmission{Status: d.CompletionSubmissionUncertain}, ctx.Err()
		}
		return d.CompletionSubmission{Status: d.CompletionSubmissionAccepted}, nil
	}}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := Execute(ctx, b, "tenant", config.BackoffConfig{}, ports, 2, services.WithFailFast())
	require.ErrorIs(t, err, d.ErrForbidden)
	require.NotErrorIs(t, err, context.Canceled)
	require.Equal(t, int32(2), started.Load())
	require.Equal(t, 1, r.Counts.Accepted)
	require.Equal(t, 3, r.Counts.NotStarted)
	assertExecutionCounts(t, b, r)
}

// TestBulkCompletionExecuteConcurrencyAndCancellation verifies bounded work, ordered results and caller failure even after acceptance.
func TestBulkCompletionExecuteConcurrencyAndCancellation(t *testing.T) {
	t.Parallel()
	b := preparedForExecution(t, 30)
	var active, peak, writes atomic.Int32
	ports := ExecutionPorts{Get: func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
		return d.Job{Key: key, State: "CREATED"}, nil
	}, Submit: func(_ context.Context, j d.Job, _ map[string]any) (d.CompletionSubmission, error) {
		n := active.Add(1)
		for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
		}
		defer active.Add(-1)
		time.Sleep(time.Millisecond)
		writes.Add(1)
		return d.CompletionSubmission{Status: d.CompletionSubmissionAccepted}, nil
	}}
	r, err := Execute(context.Background(), b, "tenant", config.BackoffConfig{}, ports, 3, services.WithNoWorkerLimit())
	require.NoError(t, err)
	require.LessOrEqual(t, peak.Load(), int32(3))
	require.Greater(t, peak.Load(), int32(1))
	require.Equal(t, int32(30), writes.Load())
	assertExecutionCounts(t, b, r)
	for _, count := range []int{1, 3} {
		b = preparedForExecution(t, count)
		ctx, cancel := context.WithCancel(context.Background())
		ports.Submit = func(context.Context, d.Job, map[string]any) (d.CompletionSubmission, error) {
			cancel()
			return d.CompletionSubmission{Status: d.CompletionSubmissionAccepted}, nil
		}
		r, err = Execute(ctx, b, "tenant", config.BackoffConfig{}, ports, 1)
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, d.CompletionBatchInterrupted, r.Status)
		require.Equal(t, 1, r.Counts.Accepted)
		require.Equal(t, count-1, r.Counts.NotStarted)
		assertExecutionCounts(t, b, r)
		if count > 1 {
			require.Equal(t, d.CompletionOutcomeInterrupted, r.Outcomes[1].Reason)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, err = Execute(ctx, b, "tenant", config.BackoffConfig{}, ports, 1)
	require.True(t, errors.Is(err, context.Canceled))
	require.Equal(t, len(b.Candidates), r.Counts.NotStarted)
	assertExecutionCounts(t, b, r)
}

// TestBulkCompletionExecuteSearchScope validates genuine exhausted/limited plans and every frozen predicate on recheck.
func TestBulkCompletionExecuteSearchScope(t *testing.T) {
	t.Parallel()
	zero := int32(0)
	original := d.Job{Key: "1", State: "CREATED", Type: "work", ProcessInstanceKey: "2", ElementInstanceKey: "3", ElementId: "task", Worker: "worker", Retries: 0, Kind: "TASK_LISTENER", ListenerEventType: "CANCELING", TenantId: "tenant"}
	filter := d.CompletionFilter{State: original.State, Type: original.Type, ProcessInstanceKey: original.ProcessInstanceKey, ElementInstanceKey: original.ElementInstanceKey, ElementId: original.ElementId, Worker: original.Worker, Retries: &zero, Kind: original.Kind, ListenerEventType: original.ListenerEventType}
	for _, limited := range []bool{false, true} {
		t.Run(fmt.Sprint(limited), func(t *testing.T) {
			selection := d.CompletionSelection{Mode: d.CompletionSelectionSearch, InputForm: d.CompletionInputFilters, Filter: filter, BatchSize: 10}
			if limited {
				selection.Limit = 1
			}
			b, err := Prepare(context.Background(), d.CompletionRequest{Selection: selection}, "tenant", config.BackoffConfig{}, ReadPorts{Search: func(_ context.Context, _ d.JobSearchQuery, p d.JobPageRequest, _ ...services.CallOption) (d.JobSearchPage, error) {
				return d.JobSearchPage{Request: p, Items: []d.Job{original}, RawItemCount: 1, NextFrom: 1, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
			}})
			require.NoError(t, err)
			for name, change := range map[string]func(*d.Job){
				"state": func(j *d.Job) { j.State = "FAILED" }, "type": func(j *d.Job) { j.Type = "other" }, "process": func(j *d.Job) { j.ProcessInstanceKey = "99" }, "element instance": func(j *d.Job) { j.ElementInstanceKey = "99" }, "element ID": func(j *d.Job) { j.ElementId = "other" }, "worker": func(j *d.Job) { j.Worker = "other" }, "retries": func(j *d.Job) { j.Retries = 1 }, "kind": func(j *d.Job) { j.Kind = "BPMN_ELEMENT" }, "event": func(j *d.Job) { j.ListenerEventType = "COMPLETING" }, "tenant": func(j *d.Job) { j.TenantId = "other" }, "unchanged": func(*d.Job) {},
			} {
				t.Run(name, func(t *testing.T) {
					writes := 0
					ports := ExecutionPorts{Get: func(context.Context, string, ...services.CallOption) (d.Job, error) {
						j := original
						change(&j)
						return j, nil
					}, Submit: func(context.Context, d.Job, map[string]any) (d.CompletionSubmission, error) {
						writes++
						return d.CompletionSubmission{Status: d.CompletionSubmissionAccepted}, nil
					}}
					r, err := Execute(context.Background(), b, "tenant", config.BackoffConfig{}, ports, 1, services.WithForce(), services.WithNoStateCheck(), services.WithIgnoreTenant())
					assertExecutionCounts(t, b, r)
					if name == "unchanged" {
						require.NoError(t, err)
						require.Equal(t, 1, writes)
					} else {
						require.Error(t, err)
						require.Zero(t, writes)
						require.Equal(t, d.CompletionOutcomeScopeChanged, r.Outcomes[0].Reason)
					}
				})
			}
			for name, change := range map[string]func(*d.PreparedCompletionBatch){
				"tenant changed": func(b *d.PreparedCompletionBatch) { b.DiscoveryTenant = "other" },
				"candidate out of scope": func(b *d.PreparedCompletionBatch) {
					b.Candidates = append([]d.CompletionCandidate(nil), b.Candidates...)
					b.Candidates[0].Type = "other"
				},
				"unnormalized filter": func(b *d.PreparedCompletionBatch) { b.Selection.Filter.State = "created" },
				"wrong reason":        func(b *d.PreparedCompletionBatch) { b.CompletionReason = d.CompletionReasonExplicitInputsRead },
				"missing pages":       func(b *d.PreparedCompletionBatch) { b.PageCount = 0 },
				"inconsistent limit":  func(b *d.PreparedCompletionBatch) { b.Selection.Limit = 2; b.Limited = true },
			} {
				t.Run(name, func(t *testing.T) {
					altered := b
					change(&altered)
					_, err := Execute(context.Background(), altered, "tenant", config.BackoffConfig{}, ExecutionPorts{}, 1, services.WithDryRun())
					require.ErrorIs(t, err, d.ErrValidation)
				})
			}
		})
	}
}

// TestBulkCompletionExecuteDefaultWorkerPolicy keeps an ordinary cap and honors an explicit finite no-limit request.
func TestBulkCompletionExecuteDefaultWorkerPolicy(t *testing.T) {
	t.Parallel()
	for _, unlimited := range []bool{false, true} {
		t.Run(fmt.Sprint(unlimited), func(t *testing.T) {
			b := preparedForExecution(t, 40)
			expected := toolx.DetermineNoOfWorkers(40, 0, unlimited)
			var active, peak atomic.Int32
			ready := make(chan struct{})
			ports := ExecutionPorts{Get: func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
				return d.Job{Key: key, State: "CREATED"}, nil
			}, Submit: func(ctx context.Context, _ d.Job, _ map[string]any) (d.CompletionSubmission, error) {
				n := active.Add(1)
				defer active.Add(-1)
				for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
				}
				if n == int32(expected) {
					select {
					case <-ready:
					default:
						close(ready)
					}
				}
				select {
				case <-ready:
					return d.CompletionSubmission{Status: d.CompletionSubmissionAccepted}, nil
				case <-ctx.Done():
					return d.CompletionSubmission{Status: d.CompletionSubmissionUncertain}, ctx.Err()
				}
			}}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			var opts []services.CallOption
			if unlimited {
				opts = append(opts, services.WithNoWorkerLimit())
			}
			r, err := Execute(ctx, b, "tenant", config.BackoffConfig{}, ports, 0, opts...)
			require.NoError(t, err)
			require.Equal(t, int32(expected), peak.Load())
			require.Equal(t, 40, r.Counts.Accepted)
			assertExecutionCounts(t, b, r)
		})
	}
}
