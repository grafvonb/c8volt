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

// resumptionPlanStub permits only local preflight and explicit descendant discovery.
type resumptionPlanStub struct {
	resumptionAPIStub
	descendants func(context.Context, string, ...services.CallOption) (traversal.Result, error)
}

// DescendantsResult makes every planning lookup observable to the fixture.
func (s resumptionPlanStub) DescendantsResult(ctx context.Context, key string, opts ...services.CallOption) (traversal.Result, error) {
	return s.descendants(ctx, key, opts...)
}

// TestPlanProcessInstanceResumptionScope proves stable overlap order, first observation,
// traversal through ineligible parents, independent identities, and owned slices/options.
func TestPlanProcessInstanceResumptionScope(t *testing.T) {
	t.Parallel()
	keys := resumptionKeys(5)
	for _, childFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(childFirst), func(t *testing.T) {
			parent := planTraversal(keys, d.StateCompleted, d.StateSuspended, d.StateActive, d.StateSuspended, d.State("FUTURE"))
			parent.Keys = append(parent.Keys, keys[1]) // Repeated traversal keys never duplicate targets.
			child := planTraversal(keys[1:4], d.StateActive, d.StateActive, d.StateSuspended)
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
			api := resumptionPlanStub{
				resumptionAPIStub: resumptionAPIStub{call: func(gotCtx context.Context, key string, got ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
					require.Same(t, ctx, gotCtx)
					require.Equal(t, wantSeeds[0], key)
					require.True(t, services.ApplyCallOptions(got).DryRun)
					previews++
					return resumptionItem(key, true), nil
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
			plan, err := PlanProcessInstanceResumption(ctx, api, selected, opts...)
			require.NoError(t, err)
			require.Equal(t, 1, previews)
			require.Equal(t, wantCalls, calls)
			require.Equal(t, wantSeeds, plan.RequestedKeys)
			require.Equal(t, wantTargets, plan.TargetKeys)
			var order, projection typex.Keys
			for _, item := range plan.Instances {
				order = append(order, item.Key)
				if item.Classification == d.ProcessInstanceResumptionPlanEligible {
					projection = append(projection, item.Key)
				}
				if item.Key == keys[4] {
					require.Equal(t, d.StateUnknown, item.ObservedState)
					require.Equal(t, d.ProcessInstanceResumptionPlanExcluded, item.Classification)
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

// TestPlanProcessInstanceResumptionRejectsInput checks capability before all-input
// validation and cancellation, with zero discovery on any local failure.
func TestPlanProcessInstanceResumptionRejectsInput(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"unsupported empty", "unsupported invalid", "empty", "invalid later", "canceled", "preflight", "canceled preflight"} {
		t.Run(mode, func(t *testing.T) {
			keys := resumptionKeys(2)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			wantErr := d.ErrValidation
			calls := 0
			api := resumptionPlanStub{resumptionAPIStub: resumptionAPIStub{call: func(context.Context, string, ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
				calls++
				if mode == "canceled preflight" {
					cancel()
					return d.ProcessInstanceResumptionResult{}, nil
				}
				return d.ProcessInstanceResumptionResult{}, d.ErrValidation
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
			got, err := PlanProcessInstanceResumption(ctx, api, keys)
			require.ErrorIs(t, err, wantErr)
			require.Equal(t, d.ProcessInstanceResumptionPlan{}, got)
			if mode == "preflight" || mode == "canceled preflight" {
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
			}
		})
	}
}

// TestPlanProcessInstanceResumptionFailedDiscovery rejects incomplete or malformed
// results, even after a valid seed, without exposing any executable partial plan.
func TestPlanProcessInstanceResumptionFailedDiscovery(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"forbidden", "missing", "partial", "unresolved", "no outcome", "wrong mode", "wrong seed", "wrong root", "missing ancestry", "empty keys", "nil chain", "missing seed", "missing record", "wrong identity", "bad key", "noncanonical key", "canceled", "deadline", "overlap malformed", "overlap mismatched"} {
		t.Run(mode, func(t *testing.T) {
			keys := resumptionKeys(3)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			wantErr := d.ErrMalformedResponse
			api := resumptionPlanStub{resumptionAPIStub: resumptionAPIStub{call: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
				require.True(t, services.ApplyCallOptions(opts).DryRun)
				return resumptionItem(key, true), nil
			}}, descendants: func(_ context.Context, key string, _ ...services.CallOption) (traversal.Result, error) {
				calls++
				if calls == 1 {
					return planTraversal(keys[:1], d.StateSuspended), nil
				}
				r := planTraversal(keys[1:], d.StateSuspended, d.StateActive)
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
				case "missing ancestry":
					r.MissingAncestors = []traversal.MissingAncestor{{Key: keys[0], StartKey: key}}
				case "empty keys":
					r.Keys = nil
				case "nil chain":
					r.Chain = nil
				case "missing seed":
					r.Keys = r.Keys[1:]
				case "missing record":
					delete(r.Chain, keys[2])
				case "wrong identity":
					r.Chain[keys[2]] = d.ProcessInstance{Key: keys[0], State: d.StateSuspended}
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
				case "overlap mismatched":
					r.Keys = append(r.Keys, keys[0])
					r.Chain[keys[0]] = d.ProcessInstance{Key: keys[1], State: d.StateSuspended}
				case "overlap malformed":
					r.Keys = append(r.Keys, keys[0]) // Even already-seen members must have an observation.
				}
				return r, nil
			}}
			got, err := PlanProcessInstanceResumption(ctx, api, keys[:2])
			require.ErrorIs(t, err, wantErr)
			require.Equal(t, d.ProcessInstanceResumptionPlan{}, got)
			require.Equal(t, 2, calls)
		})
	}
}

// TestPlanProcessInstanceResumptionIneligible preserves each known nonsuspended state,
// normalizes unrecognized observations and returns a non-nil empty target array.
func TestPlanProcessInstanceResumptionIneligible(t *testing.T) {
	t.Parallel()
	states := []d.State{d.StateActive, d.StateCompleted, d.StateCanceled, d.StateTerminated, d.StateAbsent, d.StateUnknown, "", "future", d.StateAll}
	keys := resumptionKeys(len(states))
	api := resumptionPlanStub{resumptionAPIStub: resumptionAPIStub{call: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
		require.True(t, services.ApplyCallOptions(opts).DryRun)
		return resumptionItem(key, true), nil
	}}, descendants: func(context.Context, string, ...services.CallOption) (traversal.Result, error) {
		return planTraversal(keys, states...), nil
	}}
	plan, err := PlanProcessInstanceResumption(context.Background(), api, keys[:1], services.WithDryRun())
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
		require.Equal(t, d.ProcessInstanceResumptionPlanExcluded, item.Classification)
	}
}
