// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// suspensionPlanStub permits only local preflight and explicit descendant discovery.
type suspensionPlanStub struct {
	suspensionAPIStub
	descendants func(context.Context, string, ...services.CallOption) (traversal.Result, error)
}

// DescendantsResult makes every planning lookup observable to the fixture.
func (s suspensionPlanStub) DescendantsResult(ctx context.Context, key string, opts ...services.CallOption) (traversal.Result, error) {
	return s.descendants(ctx, key, opts...)
}

// suspensionTraversal builds complete ordered observations without deriving state from preview.
func suspensionTraversal(keys typex.Keys, states ...d.State) traversal.Result {
	r := traversal.Result{Mode: traversal.ModeDescendants, StartKey: keys[0], RootKey: keys[0], Outcome: traversal.OutcomeComplete, Keys: append([]string(nil), keys...), Chain: map[string]d.ProcessInstance{}}
	for i, key := range keys {
		r.Chain[key] = d.ProcessInstance{Key: key, State: states[i]}
	}
	return r
}

// TestProcessInstanceSuspensionPlanScope proves stable overlap order, first observation,
// traversal through ineligible parents, independent identities, and owned slices/options.
func TestProcessInstanceSuspensionPlanScope(t *testing.T) {
	t.Parallel()
	keys := suspensionKeys(5)
	for _, childFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(childFirst), func(t *testing.T) {
			parent := suspensionTraversal(keys, d.StateCompleted, d.StateActive, d.StateSuspended, d.StateActive, d.State("FUTURE"))
			parent.Keys = append(parent.Keys, keys[1]) // Repeated traversal keys never duplicate targets.
			child := suspensionTraversal(keys[1:4], d.StateSuspended, d.StateSuspended, d.StateActive)
			selected := typex.Keys{" " + keys[0] + "\n", keys[1], keys[0]}
			wantSeeds, wantCalls, wantOrder := typex.Keys{keys[0], keys[1]}, typex.Keys{keys[0]}, keys
			wantTargets := typex.Keys{keys[1], keys[3]}
			if childFirst {
				selected = typex.Keys{keys[1], keys[0], keys[1]}
				wantSeeds = typex.Keys{keys[1], keys[0]}
				wantCalls = wantSeeds
				wantOrder = typex.Keys{keys[1], keys[2], keys[3], keys[0], keys[4]}
				wantTargets = typex.Keys{keys[3]}
			}
			original := append(typex.Keys(nil), selected...)
			// Spare capacity exposes accidental append into the caller's options backing array.
			opts := make([]services.CallOption, 4, 10)
			opts[0], opts[1], opts[2], opts[3] = services.WithNoWait(), services.WithVerbose(), services.WithFailFast(), services.WithNoWorkerLimit()
			type marker struct{}
			ctx := context.WithValue(context.Background(), marker{}, "caller")
			previews := 0
			var calls typex.Keys
			api := suspensionPlanStub{
				suspensionAPIStub: suspensionAPIStub{call: func(gotCtx context.Context, key string, got ...services.CallOption) (d.ProcessInstanceSuspensionResult, error) {
					require.Same(t, ctx, gotCtx)
					require.Equal(t, wantSeeds[0], key)
					require.True(t, services.ApplyCallOptions(got).DryRun)
					previews++
					return suspensionItem(key, true), nil
				}},
				descendants: func(gotCtx context.Context, key string, got ...services.CallOption) (traversal.Result, error) {
					require.Same(t, ctx, gotCtx)
					cfg := services.ApplyCallOptions(got)
					require.True(t, cfg.IgnoreTenant)
					require.True(t, cfg.SuppressWorkflowDetailLogs)
					require.True(t, cfg.SuppressProcessInstanceDetailLogs)
					require.False(t, cfg.SuppressNestedProcessInstanceLookupLogs, "DEBUG remains independent")
					require.True(t, cfg.NoWait && cfg.Verbose && cfg.FailFast && cfg.NoWorkerLimit)
					require.False(t, cfg.DryRun, "forced preview must not leak")
					calls = append(calls, key)
					if key == keys[1] {
						return child, nil
					}
					require.Equal(t, keys[0], key)
					return parent, nil
				},
			}
			plan, err := PlanProcessInstanceSuspension(ctx, api, selected, opts...)
			require.NoError(t, err)
			require.Equal(t, 1, previews)
			require.Equal(t, wantCalls, calls)
			require.Equal(t, wantSeeds, plan.RequestedKeys)
			require.Equal(t, wantTargets, plan.TargetKeys)
			var order, projection typex.Keys
			for _, item := range plan.Instances {
				order = append(order, item.Key)
				if item.Classification == d.ProcessInstanceSuspensionPlanEligible {
					projection = append(projection, item.Key)
				}
				if item.Key == keys[4] {
					require.Equal(t, d.StateUnknown, item.ObservedState)
					require.Equal(t, d.ProcessInstanceSuspensionPlanExcluded, item.Classification)
				}
			}
			require.Equal(t, wantOrder, order)
			require.Equal(t, plan.TargetKeys, projection)
			require.Equal(t, original, selected)
			for _, extra := range opts[len(opts):cap(opts)] {
				require.Nil(t, extra)
			}
			require.False(t, services.ApplyCallOptions(opts).DryRun)
			plan.RequestedKeys[0], plan.TargetKeys[0], plan.Instances[0].Key = "changed", "changed", "changed"
			require.Equal(t, original, selected)
			require.Equal(t, keys[0], parent.Keys[0])
			require.Equal(t, keys[0], parent.Chain[keys[0]].Key)
			require.Equal(t, keys[1], child.Chain[keys[1]].Key)
		})
	}
}

// TestProcessInstanceSuspensionPlanRejectsInput checks capability before all-input
// validation and cancellation, with zero discovery on any local failure.
func TestProcessInstanceSuspensionPlanRejectsInput(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"unsupported empty", "unsupported invalid", "empty", "invalid later", "canceled", "preflight", "canceled preflight"} {
		t.Run(mode, func(t *testing.T) {
			keys := suspensionKeys(2)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			wantErr := d.ErrValidation
			calls := 0
			api := suspensionPlanStub{suspensionAPIStub: suspensionAPIStub{call: func(context.Context, string, ...services.CallOption) (d.ProcessInstanceSuspensionResult, error) {
				calls++
				if mode == "canceled preflight" {
					cancel()
					return d.ProcessInstanceSuspensionResult{}, nil
				}
				return d.ProcessInstanceSuspensionResult{}, d.ErrValidation
			}}, descendants: func(context.Context, string, ...services.CallOption) (traversal.Result, error) {
				t.Fatal("discovery after local failure")
				return traversal.Result{}, nil
			}}
			switch mode {
			case "unsupported empty":
				api.guard = d.ErrUnsupported
				keys = nil
				wantErr = d.ErrUnsupported
			case "unsupported invalid":
				api.guard = d.ErrUnsupported
				keys = typex.Keys{"bad"}
				wantErr = d.ErrUnsupported
			case "empty":
				keys = nil
			case "invalid later":
				keys = append(keys, "bad")
			case "canceled":
				cancel()
				wantErr = context.Canceled
			case "canceled preflight":
				wantErr = context.Canceled
			}
			got, err := PlanProcessInstanceSuspension(ctx, api, keys)
			require.ErrorIs(t, err, wantErr)
			require.Equal(t, d.ProcessInstanceSuspensionPlan{}, got)
			if mode == "preflight" || mode == "canceled preflight" {
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
			}
		})
	}
}

// TestProcessInstanceSuspensionPlanFailedDiscovery rejects incomplete or malformed
// results, even after a valid seed, without exposing any executable partial plan.
func TestProcessInstanceSuspensionPlanFailedDiscovery(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"forbidden", "missing", "partial", "unresolved", "no outcome", "wrong mode", "wrong seed", "wrong root", "missing seed", "missing record", "wrong identity", "bad key", "noncanonical key", "canceled", "deadline", "overlap malformed"} {
		t.Run(mode, func(t *testing.T) {
			keys := suspensionKeys(3)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			wantErr := d.ErrMalformedResponse
			api := suspensionPlanStub{suspensionAPIStub: suspensionAPIStub{call: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceSuspensionResult, error) {
				require.True(t, services.ApplyCallOptions(opts).DryRun)
				return suspensionItem(key, true), nil
			}}, descendants: func(_ context.Context, key string, _ ...services.CallOption) (traversal.Result, error) {
				calls++
				if calls == 1 {
					return suspensionTraversal(keys[:1], d.StateActive), nil
				}
				r := suspensionTraversal(keys[1:], d.StateActive, d.StateSuspended)
				switch mode {
				case "forbidden":
					wantErr = d.ErrForbidden
					return r, fmt.Errorf("discovery: %w", wantErr)
				case "missing":
					wantErr = d.ErrNotFound
					return r, wantErr
				case "partial":
					wantErr = d.ErrPrecondition
					r.Outcome = traversal.OutcomePartial
				case "unresolved":
					wantErr = d.ErrPrecondition
					r.Outcome = traversal.OutcomeUnresolved
				case "no outcome":
					r.Outcome = ""
				case "wrong mode":
					r.Mode = traversal.ModeFamily
				case "wrong seed":
					r.StartKey = keys[0]
				case "wrong root":
					r.RootKey = keys[0]
				case "missing seed":
					r.Keys = r.Keys[1:]
				case "missing record":
					delete(r.Chain, keys[2])
				case "wrong identity":
					r.Chain[keys[2]] = d.ProcessInstance{Key: keys[0], State: d.StateActive}
				case "bad key":
					r.Keys = append(r.Keys, "bad")
				case "noncanonical key":
					r.Keys = append(r.Keys, " "+keys[2])
				case "canceled":
					cancel()
					wantErr = context.Canceled
				case "deadline":
					wantErr = context.DeadlineExceeded
					return r, wantErr
				case "overlap malformed":
					r.Keys = append(r.Keys, keys[0]) // Even already-seen members must have an observation.
				}
				return r, nil
			}}
			got, err := PlanProcessInstanceSuspension(ctx, api, keys[:2])
			require.ErrorIs(t, err, wantErr)
			require.Equal(t, d.ProcessInstanceSuspensionPlan{}, got)
			require.Equal(t, 2, calls)
		})
	}
}

// TestProcessInstanceSuspensionPlanIneligible preserves each known nonactive state,
// normalizes unrecognized observations and returns a non-nil empty target array.
func TestProcessInstanceSuspensionPlanIneligible(t *testing.T) {
	t.Parallel()
	states := []d.State{d.StateSuspended, d.StateCompleted, d.StateCanceled, d.StateTerminated, d.StateAbsent, d.StateUnknown, "", "future", d.StateAll}
	keys := suspensionKeys(len(states))
	api := suspensionPlanStub{suspensionAPIStub: suspensionAPIStub{call: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceSuspensionResult, error) {
		require.True(t, services.ApplyCallOptions(opts).DryRun)
		return suspensionItem(key, true), nil
	}}, descendants: func(context.Context, string, ...services.CallOption) (traversal.Result, error) {
		return suspensionTraversal(keys, states...), nil
	}}
	plan, err := PlanProcessInstanceSuspension(context.Background(), api, keys[:1], services.WithDryRun())
	require.NoError(t, err)
	require.NotNil(t, plan.RequestedKeys)
	require.NotNil(t, plan.Instances)
	require.NotNil(t, plan.TargetKeys)
	require.Empty(t, plan.TargetKeys)
	for i, item := range plan.Instances {
		expectedState := states[i]
		if i >= 6 {
			expectedState = d.StateUnknown
		}
		require.Equal(t, expectedState, item.ObservedState)
		if i == 0 {
			require.Equal(t, d.ProcessInstanceSuspensionPlanAlreadySuspended, item.Classification)
		} else {
			require.Equal(t, d.ProcessInstanceSuspensionPlanExcluded, item.Classification)
		}
	}
}
