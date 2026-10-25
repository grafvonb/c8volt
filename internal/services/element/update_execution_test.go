// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// elementExecutionVariables supplies isolated read/write callbacks without backend discovery.
type elementExecutionVariables struct {
	read  scopeReaderFunc
	write func(context.Context, string, map[string]any, ...services.CallOption) (d.ScopeVariableUpdateResponse, error)
}

// SearchScopeVariables delegates a complete read to the test callback.
func (s elementExecutionVariables) SearchScopeVariables(ctx context.Context, q d.ScopeVariableQuery, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	return s.read(ctx, q, opts...)
}

// UpdateScopeVariables delegates one logical write to the test callback.
func (s elementExecutionVariables) UpdateScopeVariables(ctx context.Context, key string, values map[string]any, opts ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
	return s.write(ctx, key, values, opts...)
}

// acceptedElementWrite models documented acceptance, independently of the legacy boolean.
func acceptedElementWrite() d.ScopeVariableUpdateResponse {
	return d.ScopeVariableUpdateResponse{SubmissionStatus: d.ScopeVariableSubmissionAccepted, StatusCode: 204}
}

// elementVerificationRows builds complete exact-local observations from approved values.
func elementVerificationRows(q d.ScopeVariableQuery, values map[string]any) []d.ProcessInstanceVariable {
	rows := make([]d.ProcessInstanceVariable, 0, len(values))
	for name, value := range values {
		raw, _ := json.Marshal(value)
		rows = append(rows, d.ProcessInstanceVariable{Name: name, Value: string(raw), ProcessInstanceKey: q.ProcessInstanceKey, ScopeKey: q.ScopeKey})
	}
	return rows
}

// TestElementVariableUpdatesExecuteFrozen verifies one exact write per changed key,
// all requested values in confirmation, stable order, and isolation before workers.
func TestElementVariableUpdatesExecuteFrozen(t *testing.T) {
	t.Parallel()
	plan := elementValidationPlan(t)
	expected := elementValidationPlan(t)
	writes, reads := []string{}, []string{}
	vars := elementExecutionVariables{
		write: func(_ context.Context, key string, values map[string]any, opts ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
			writes = append(writes, key)
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			require.Equal(t, expected.Elements[len(writes)-1].MutationVariables, values)
			// The service must have frozen every target before the first callback.
			plan.Elements[1].ScopeKey = "parent"
			plan.Elements[1].MutationVariables["add"].(map[string]any)["new"].([]any)[0] = 99
			values["add"].(map[string]any)["new"].([]any)[0] = 88
			return acceptedElementWrite(), nil
		},
		read: func(_ context.Context, q d.ScopeVariableQuery, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
			reads = append(reads, q.ScopeKey)
			require.Equal(t, "P-"+q.ScopeKey, q.ProcessInstanceKey)
			require.Equal(t, q.ScopeKey, *q.ExpectedTenantId)
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			return elementVerificationRows(q, expected.RequestedValues), nil
		}}
	result, err := NewVariableUpdates(nil, vars, nil).ExecuteElementVariableUpdates(context.Background(), plan, 1)
	require.NoError(t, err)
	require.Equal(t, []string{"E2", "E1"}, writes)
	require.Equal(t, writes, reads)
	for i, item := range result.Items {
		require.Equal(t, expected.RequestedKeys[i], item.Key)
		require.Equal(t, "P-"+item.Key, item.ProcessInstanceKey)
		require.Equal(t, item.Key, item.ScopeKey)
		require.Equal(t, d.ElementVariableUpdateStatusConfirmed, item.Status)
		require.Equal(t, d.ScopeVariableSubmissionAccepted, item.SubmissionStatus)
		require.Equal(t, "confirmed", item.ConfirmationStatus)
		require.True(t, item.MutationAccepted)
		require.True(t, item.OK())
		require.Equal(t, expected.Elements[i].MutationVariables, item.Variables)
	}
}

// TestElementVariableUpdatesExecuteEvidence proves strict evidence drives outcomes,
// including legacy-success ambiguity, no-wait, and accepted errors that prevent confirmation.
func TestElementVariableUpdatesExecuteEvidence(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		evidence     d.ScopeVariableSubmissionStatus
		legacy       bool
		failure      error
		noWait       bool
		status       d.ElementVariableUpdateStatus
		confirmation string
		success      bool
	}{
		{"unsent", d.ScopeVariableSubmissionNotSent, false, context.DeadlineExceeded, false, d.ElementVariableUpdateStatusMutationFailed, "not_required", false},
		{"rejected", d.ScopeVariableSubmissionRejected, false, d.ErrValidation, false, d.ElementVariableUpdateStatusMutationFailed, "not_required", false},
		{"unknown", d.ScopeVariableSubmissionUnknown, true, nil, false, d.ElementVariableUpdateStatusMutationFailed, "not_required", false},
		{"accepted no wait", d.ScopeVariableSubmissionAccepted, false, nil, true, d.ElementVariableUpdateStatusSubmitted, "skipped", true},
		{"accepted interrupted", d.ScopeVariableSubmissionAccepted, true, context.Canceled, false, d.ElementVariableUpdateStatusSubmitted, "not_started", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := elementValidationPlan(t)
			var reads atomic.Int32
			vars := elementExecutionVariables{write: func(context.Context, string, map[string]any, ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
				return d.ScopeVariableUpdateResponse{SubmissionStatus: tc.evidence, Accepted: tc.legacy, StatusCode: 503}, tc.failure
			}, read: func(context.Context, d.ScopeVariableQuery, ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
				reads.Add(1)
				return nil, nil
			}}
			opts := []services.CallOption{}
			if tc.noWait {
				opts = append(opts, services.WithNoWait())
			}
			result, err := NewVariableUpdates(nil, vars, nil).ExecuteElementVariableUpdates(context.Background(), plan, 1, opts...)
			if tc.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			if tc.failure != nil {
				require.ErrorIs(t, err, tc.failure)
			}
			require.Zero(t, reads.Load())
			require.Len(t, result.Items, 2)
			for _, item := range result.Items {
				require.Equal(t, tc.status, item.Status)
				require.Equal(t, tc.evidence, item.SubmissionStatus)
				require.Equal(t, tc.confirmation, item.ConfirmationStatus)
				require.Equal(t, tc.evidence == d.ScopeVariableSubmissionAccepted, item.MutationAccepted)
				require.Equal(t, tc.success, item.OK())
				require.Equal(t, 503, item.StatusCode)
			}
		})
	}
}

// TestElementVariableUpdatesExecuteFailFast preserves accepted in-flight writes
// while distinguishing unscheduled work and suppressing a new confirmation phase.
func TestElementVariableUpdatesExecuteFailFast(t *testing.T) {
	t.Parallel()
	plan := elementValidationPlan(t)
	started := make(chan struct{})
	var writes atomic.Int32
	vars := elementExecutionVariables{write: func(ctx context.Context, key string, _ map[string]any, _ ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
		writes.Add(1)
		if key == "E2" {
			<-started
			return d.ScopeVariableUpdateResponse{SubmissionStatus: d.ScopeVariableSubmissionRejected}, d.ErrValidation
		}
		close(started)
		<-ctx.Done()
		return acceptedElementWrite(), nil
	}}
	result, err := NewVariableUpdates(nil, vars, nil).ExecuteElementVariableUpdates(context.Background(), plan, 2, services.WithFailFast())
	require.ErrorIs(t, err, d.ErrValidation)
	require.Equal(t, int32(2), writes.Load())
	require.Equal(t, d.ElementVariableUpdateStatusMutationFailed, result.Items[0].Status)
	require.Equal(t, d.ElementVariableUpdateStatusSubmitted, result.Items[1].Status)
	require.Equal(t, "not_started", result.Items[1].ConfirmationStatus)
	require.True(t, result.Items[1].MutationAccepted)
	require.False(t, result.Items[1].OK())
	require.NotEmpty(t, result.Items[1].Error)
	// One worker cannot start a second destination after the first failure.
	vars.write = func(context.Context, string, map[string]any, ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
		return d.ScopeVariableUpdateResponse{SubmissionStatus: d.ScopeVariableSubmissionNotSent}, d.ErrValidation
	}
	result, err = NewVariableUpdates(nil, vars, nil).ExecuteElementVariableUpdates(context.Background(), plan, 1, services.WithFailFast())
	require.ErrorIs(t, err, d.ErrValidation)
	require.Equal(t, d.ElementVariableUpdateStatusSkipped, result.Items[1].Status)
	require.Equal(t, d.ScopeVariableSubmissionNotSent, result.Items[1].SubmissionStatus)
	require.Equal(t, "not_required", result.Items[1].ConfirmationStatus)
}

// TestElementVariableUpdatesExecuteCancellation retains accepted writes when cancellation
// precedes read-back, and returns ordered skipped outcomes when scheduling never starts.
func TestElementVariableUpdatesExecuteCancellation(t *testing.T) {
	t.Parallel()
	for _, before := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		plan := elementValidationPlan(t)
		writes := 0
		vars := elementExecutionVariables{write: func(context.Context, string, map[string]any, ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
			writes++
			cancel()
			return acceptedElementWrite(), nil
		}}
		if before {
			cancel()
		}
		result, err := NewVariableUpdates(nil, vars, nil).ExecuteElementVariableUpdates(ctx, plan, 1)
		cancel()
		require.ErrorIs(t, err, context.Canceled)
		if before {
			require.Zero(t, writes)
			require.Equal(t, d.ElementVariableUpdateStatusSkipped, result.Items[0].Status)
		} else {
			require.Equal(t, 1, writes)
			require.Equal(t, "not_started", result.Items[0].ConfirmationStatus)
			require.True(t, result.Items[0].MutationAccepted)
			require.False(t, result.Items[0].OK())
		}
		require.Equal(t, d.ElementVariableUpdateStatusSkipped, result.Items[1].Status)
	}
}

// TestElementVariableUpdatesExecuteWorkers bounds both phases and preserves input order
// despite concurrent completion. No worker relies on shared unsynchronized fixture state.
func TestElementVariableUpdatesExecuteWorkers(t *testing.T) {
	t.Parallel()
	for _, workers := range []int{1, 2} {
		plan := elementValidationPlan(t)
		var active, peak, writes, reads atomic.Int32
		work := func() {
			n := active.Add(1)
			for old := peak.Load(); n > old; old = peak.Load() {
				if peak.CompareAndSwap(old, n) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			active.Add(-1)
		}
		vars := elementExecutionVariables{write: func(context.Context, string, map[string]any, ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
			writes.Add(1)
			work()
			return acceptedElementWrite(), nil
		}, read: func(_ context.Context, q d.ScopeVariableQuery, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
			reads.Add(1)
			work()
			return elementVerificationRows(q, plan.RequestedValues), nil
		}}
		result, err := NewVariableUpdates(nil, vars, config.New()).ExecuteElementVariableUpdates(context.Background(), plan, workers, services.WithNoWorkerLimit())
		require.NoError(t, err)
		require.Equal(t, int32(2), writes.Load())
		require.Equal(t, int32(2), reads.Load())
		require.LessOrEqual(t, peak.Load(), int32(workers))
		require.Equal(t, "E2", result.Items[0].Key)
		require.Equal(t, "E1", result.Items[1].Key)
	}
}

// TestElementVariableUpdatesExecuteNoOpAndDryRun never starts remote work for
// authoritative unchanged plans or previews, even when no-wait is also enabled.
func TestElementVariableUpdatesExecuteNoOpAndDryRun(t *testing.T) {
	t.Parallel()
	for _, empty := range []bool{false, true} {
		plan := elementValidationPlan(t)
		for i := range plan.Elements {
			p := &plan.Elements[i]
			if empty {
				p.VerificationValues = map[string]any{}
				p.UnchangedRequested = []d.ElementVariablePlannedValue{}
			} else {
				for _, v := range p.Additions {
					p.UnchangedRequested = append(p.UnchangedRequested, v)
				}
				for _, v := range p.Changes {
					p.UnchangedRequested = append(p.UnchangedRequested, d.ElementVariablePlannedValue{Name: v.Name, Value: v.After})
				}
			}
			p.Additions = nil
			p.Changes = nil
			p.MutationVariables = nil
		}
		plan.UpdateCount = 0
		plan.VariableAddCount = 0
		plan.VariableChangeCount = 0
		plan.VariableUnchangedCount = 6
		if empty {
			plan.RequestedValues = map[string]any{}
			plan.VariableUnchangedCount = 0
		}
		for _, noWait := range []bool{false, true} {
			opts := []services.CallOption{}
			if noWait {
				opts = append(opts, services.WithNoWait())
			}
			result, err := NewVariableUpdates(nil, nil, nil).ExecuteElementVariableUpdates(context.Background(), plan, 2, opts...)
			require.NoError(t, err)
			require.Len(t, result.Items, 2)
			for _, item := range result.Items {
				require.Equal(t, d.ElementVariableUpdateStatusUnchanged, item.Status)
				require.Equal(t, d.ScopeVariableSubmissionNotSent, item.SubmissionStatus)
				require.Equal(t, "not_required", item.ConfirmationStatus)
				require.False(t, item.MutationAccepted)
				require.True(t, item.OK())
			}
		}
	}
	result, err := NewVariableUpdates(nil, nil, nil).ExecuteElementVariableUpdates(context.Background(), elementValidationPlan(t), 1, services.WithDryRun(), services.WithNoWait())
	require.NoError(t, err)
	require.NotNil(t, result.Items)
	require.Empty(t, result.Items)
	_, err = NewVariableUpdates(nil, nil, nil).ExecuteElementVariableUpdates(context.Background(), elementValidationPlan(t), 1)
	require.ErrorIs(t, err, d.ErrPrecondition)
}

// TestElementVariableUpdatesExecutePartial continues independent work after a
// submission failure without fail-fast, retaining unchanged and confirmed outcomes.
func TestElementVariableUpdatesExecutePartial(t *testing.T) {
	t.Parallel()
	executing := false
	gets, writes, reads := 0, []string{}, []string{}
	vars := elementExecutionVariables{
		write: func(_ context.Context, key string, _ map[string]any, _ ...services.CallOption) (d.ScopeVariableUpdateResponse, error) {
			writes = append(writes, key)
			if key == "E1" {
				return d.ScopeVariableUpdateResponse{SubmissionStatus: d.ScopeVariableSubmissionUnknown}, d.ErrUnavailable
			}
			return acceptedElementWrite(), nil
		},
		read: func(_ context.Context, q d.ScopeVariableQuery, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
			if executing {
				reads = append(reads, q.ScopeKey)
			}
			if executing || q.ScopeKey == "E3" {
				return elementVerificationRows(q, map[string]any{"x": true}), nil
			}
			return nil, nil
		},
	}
	api := NewVariableUpdates(stubElementAPI{get: func(_ context.Context, key string, _ ...services.CallOption) (d.Element, error) {
		gets++
		return d.Element{ElementInstanceKey: key, ProcessInstanceKey: "P"}, nil
	}}, vars, nil)
	plan, err := api.PlanElementVariableUpdates(context.Background(), []string{"E1", "E2", "E3", "E1"}, map[string]any{"x": true})
	require.NoError(t, err)
	executing = true
	result, err := api.ExecuteElementVariableUpdates(context.Background(), plan, 1)
	require.ErrorIs(t, err, d.ErrUnavailable)
	require.Equal(t, 3, gets)
	require.Equal(t, []string{"E1", "E2"}, writes)
	require.Equal(t, []string{"E2"}, reads)
	require.Len(t, result.Items, 3)
	require.Equal(t, d.ElementVariableUpdateStatusMutationFailed, result.Items[0].Status)
	require.Equal(t, d.ScopeVariableSubmissionUnknown, result.Items[0].SubmissionStatus)
	require.Equal(t, d.ElementVariableUpdateStatusConfirmed, result.Items[1].Status)
	require.True(t, result.Items[1].OK())
	require.Equal(t, d.ElementVariableUpdateStatusUnchanged, result.Items[2].Status)
	require.True(t, result.Items[2].OK())
	require.False(t, result.Items[2].MutationAccepted)
}
