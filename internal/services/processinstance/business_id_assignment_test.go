// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v88 "github.com/grafvonb/c8volt/internal/services/processinstance/v88"
	v89 "github.com/grafvonb/c8volt/internal/services/processinstance/v89"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// assignmentAPI restricts tests to explicit reads and exact-target submissions.
type assignmentAPI struct {
	API
	read   func(context.Context, string) (d.ProcessInstance, error)
	submit func(context.Context, string, *string) (d.BusinessIDAssignmentResult, error)
}

// ValidateBusinessIDAssignment models native input policy without network access.
func (a assignmentAPI) ValidateBusinessIDAssignment(value *string, opts ...services.CallOption) error {
	if value == nil {
		return d.ErrValidation
	}
	return d.ValidateBusinessIDAssignment(value)
}

// GetProcessInstance permits exact-target preview reads only.
func (a assignmentAPI) GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error) {
	return a.read(ctx, key)
}

// AssignProcessInstanceBusinessID delegates a single submission to owned evidence.
func (a assignmentAPI) AssignProcessInstanceBusinessID(ctx context.Context, key string, value *string, opts ...services.CallOption) (d.BusinessIDAssignmentResult, error) {
	return a.submit(ctx, key, value)
}

// assignmentKeys provides valid ordered resource identifiers without consulting an engine.
func assignmentKeys(n int) typex.Keys {
	keys := make(typex.Keys, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("225179981368%04d", i)
	}
	return keys
}

// TestBusinessIDAssignmentPlanning checks whole-input validation, stable deduplication, copying, and incomplete discovery safety.
func TestBusinessIDAssignmentPlanning(t *testing.T) {
	t.Parallel()
	keys := assignmentKeys(2)
	value := "  Order,001  "
	prior := "old"
	reads, writes := 0, 0
	api := assignmentAPI{read: func(_ context.Context, key string) (d.ProcessInstance, error) {
		reads++
		return d.ProcessInstance{Key: key, BusinessId: &prior, State: d.StateActive}, nil
	}, submit: func(_ context.Context, key string, v *string) (d.BusinessIDAssignmentResult, error) {
		writes++
		return d.BusinessIDAssignmentResult{Key: key, Status: "accepted", Confirmation: "skipped"}, nil
	}}
	_, err := PlanProcessInstanceBusinessIDAssignments(context.Background(), api, append(keys, "bad"), &value)
	require.ErrorIs(t, err, d.ErrValidation)
	require.Zero(t, reads)
	input := typex.Keys{" " + keys[0] + " ", keys[1], keys[0]}
	plan, err := PlanProcessInstanceBusinessIDAssignments(context.Background(), api, input, &value)
	require.NoError(t, err)
	require.Equal(t, []string(keys), plan.RequestedKeys)
	require.Len(t, plan.Items, 2)
	input[0] = "changed"
	value = "changed"
	prior = "changed"
	require.Equal(t, "  Order,001  ", *plan.RequestedBusinessID)
	require.Equal(t, "old", *plan.Items[0].PreviouslyObservedBusinessID)
	preview, err := ExecuteProcessInstanceBusinessIDAssignments(context.Background(), api, plan, 1, services.WithDryRun())
	require.NoError(t, err)
	require.Equal(t, "not_submitted", preview.Items[0].Status)
	require.Zero(t, writes)
	*preview.Items[0].RequestedBusinessID = "modified"
	require.Equal(t, "  Order,001  ", *plan.RequestedBusinessID)
	api.read = func(_ context.Context, key string) (d.ProcessInstance, error) {
		if key == keys[1] {
			return d.ProcessInstance{}, d.ErrForbidden
		}
		return d.ProcessInstance{Key: key}, nil
	}
	incomplete, err := PlanProcessInstanceBusinessIDAssignments(context.Background(), api, keys, new("x"))
	require.ErrorIs(t, err, d.ErrForbidden)
	require.False(t, incomplete.Ready)
	require.Len(t, incomplete.Items, 1)
	_, err = ExecuteProcessInstanceBusinessIDAssignments(context.Background(), api, incomplete, 1)
	require.ErrorIs(t, err, d.ErrValidation)
	require.Zero(t, writes)
	for _, modify := range []func(*d.BusinessIDAssignmentPlan){
		func(p *d.BusinessIDAssignmentPlan) { p.Ready = false }, func(p *d.BusinessIDAssignmentPlan) { p.Items[1].Key = p.Items[0].Key }, func(p *d.BusinessIDAssignmentPlan) { p.RequestedKeys[1] = p.RequestedKeys[0] }, func(p *d.BusinessIDAssignmentPlan) { p.RequestedBusinessID = nil },
	} {
		copyPlan := plan
		copyPlan.Items = append([]d.BusinessIDAssignmentItem{}, plan.Items...)
		copyPlan.RequestedKeys = append([]string{}, plan.RequestedKeys...)
		modify(&copyPlan)
		_, err = ExecuteProcessInstanceBusinessIDAssignments(context.Background(), api, copyPlan, 1)
		require.ErrorIs(t, err, d.ErrValidation)
	}
	require.Zero(t, writes)
	empty, err := PlanProcessInstanceBusinessIDAssignments(context.Background(), api, nil, new("x"))
	require.NoError(t, err)
	require.True(t, empty.Ready)
	out, err := ExecuteProcessInstanceBusinessIDAssignments(context.Background(), api, empty, 1, services.WithNoWait())
	require.NoError(t, err)
	require.Empty(t, out.Items)
	require.Zero(t, writes)
}

// TestBusinessIDAssignmentExecution preserves one submission per distinct target, bounded workers, ordered partial failures and unscheduled slots.
func TestBusinessIDAssignmentExecution(t *testing.T) {
	for _, failFast := range []bool{false, true} {
		t.Run(fmt.Sprint(failFast), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				keys := assignmentKeys(6)
				var active, peak, writes atomic.Int32
				api := assignmentAPI{read: func(_ context.Context, key string) (d.ProcessInstance, error) {
					return d.ProcessInstance{Key: key}, nil
				}, submit: func(ctx context.Context, key string, value *string) (d.BusinessIDAssignmentResult, error) {
					writes.Add(1)
					now := active.Add(1)
					defer active.Add(-1)
					for old := peak.Load(); now > old; old = peak.Load() {
						if peak.CompareAndSwap(old, now) {
							break
						}
					}
					time.Sleep(time.Millisecond)
					if key == keys[0] {
						return d.BusinessIDAssignmentResult{Key: key, Status: "rejected", Confirmation: "skipped"}, d.ErrConflict
					}
					return d.BusinessIDAssignmentResult{Key: key, Status: "accepted", Confirmation: "skipped"}, nil
				}}
				plan, err := PlanProcessInstanceBusinessIDAssignments(context.Background(), api, append(keys, keys[0]), new("x"))
				require.NoError(t, err)
				workers := 2
				var opts []services.CallOption
				if failFast {
					opts = append(opts, services.WithFailFast())
					workers = 1
				}
				out, err := ExecuteProcessInstanceBusinessIDAssignments(context.Background(), api, plan, workers, opts...)
				require.ErrorIs(t, err, d.ErrConflict)
				require.Len(t, out.Items, 6)
				require.LessOrEqual(t, peak.Load(), int32(workers))
				for i, item := range out.Items {
					require.Equal(t, keys[i], item.Key)
					require.Equal(t, "x", *item.RequestedBusinessID)
				}
				if failFast {
					require.Equal(t, int32(1), writes.Load())
					require.Equal(t, "not_submitted", out.Items[1].Status)
				} else {
					require.Equal(t, int32(6), writes.Load())
					require.Equal(t, int32(2), peak.Load())
					require.Equal(t, "accepted", out.Items[5].Status)
				}
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				before := writes.Load()
				out, err = ExecuteProcessInstanceBusinessIDAssignments(ctx, api, plan, 2)
				require.ErrorIs(t, err, context.Canceled)
				require.Equal(t, before, writes.Load())
				require.Equal(t, "not_submitted", out.Items[0].Status)
			})
		})
	}
}

// TestBusinessIDAssignmentUnsupported rejects planning, forged execution and direct calls on old adapters before I/O.
func TestBusinessIDAssignmentUnsupported(t *testing.T) {
	t.Parallel()
	for _, api := range []API{&v88.Service{}, &v89.Service{}} {
		_, err := PlanProcessInstanceBusinessIDAssignments(context.Background(), api, assignmentKeys(1), new("x"))
		require.ErrorIs(t, err, d.ErrUnsupported)
		_, err = ExecuteProcessInstanceBusinessIDAssignments(context.Background(), api, d.BusinessIDAssignmentPlan{Ready: true, RequestedBusinessID: new("x")}, 1)
		require.ErrorIs(t, err, d.ErrUnsupported)
		_, err = api.AssignProcessInstanceBusinessID(context.Background(), assignmentKeys(1)[0], new("x"))
		require.ErrorIs(t, err, d.ErrUnsupported)
	}
}

// TestBusinessIDAssignmentCancellation retains already accepted work and explicit not-started slots after cancellation.
func TestBusinessIDAssignmentCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	api := assignmentAPI{read: func(_ context.Context, key string) (d.ProcessInstance, error) {
		return d.ProcessInstance{Key: key}, nil
	}, submit: func(_ context.Context, key string, value *string) (d.BusinessIDAssignmentResult, error) {
		calls++
		cancel()
		return d.BusinessIDAssignmentResult{Key: key, Status: "accepted", Confirmation: "skipped"}, nil
	}}
	plan, err := PlanProcessInstanceBusinessIDAssignments(ctx, api, assignmentKeys(3), new("x"))
	require.NoError(t, err)
	out, err := ExecuteProcessInstanceBusinessIDAssignments(ctx, api, plan, 1)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, calls)
	require.Len(t, out.Items, 3)
	require.Equal(t, "accepted", out.Items[0].Status)
	require.Equal(t, "not_submitted", out.Items[1].Status)
	require.Equal(t, "not_submitted", out.Items[2].Status)
}
