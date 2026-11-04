// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"slices"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDOperationsCleanupFrozen verifies complete paging, immutable exact targets, skips, and plan rejection before I/O.
func TestBusinessIDOperationsCleanupFrozen(t *testing.T) {
	for _, kind := range []string{"retention", "orphan"} {
		for _, skip := range []bool{false, true} {
			t.Run(fmt.Sprint(kind, skip), func(t *testing.T) {
				reads, searches := 0, 0
				var writes []string
				item := func(k string) d.ProcessInstance {
					p := ""
					if kind == "orphan" {
						p = "missing" + k
					}
					return d.ProcessInstance{Key: k, ParentKey: p, EndDate: "2020-01-01", State: d.StateCompleted, BusinessId: new("observed"), Variables: map[string]any{"nested": map[string]any{"value": "original"}}}
				}
				pi := stubProcessInstanceAPI{searchPage: func(_ context.Context, f d.ProcessInstanceFilter, p d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
					searches++
					require.Equal(t, " ref,*Ω ", *f.BusinessId)
					out := d.ProcessInstancePage{Request: p, OverflowState: d.ProcessInstanceOverflowStateHasMore, EndCursor: fmt.Sprint(searches)}
					switch searches {
					case 1:
						out.Items = []d.ProcessInstance{item("a"), item("a")}
					case 2:
					case 3:
						out.Items = []d.ProcessInstance{item("b")}
						out.OverflowState = d.ProcessInstanceOverflowStateNoMore
					default:
						t.Fatal("selection repeated")
					}
					return out, nil
				}, filterOrphans: func(_ context.Context, items []d.ProcessInstance, _ ...services.CallOption) ([]d.ProcessInstance, error) {
					if skip && items[0].Key == "b" {
						return nil, nil
					}
					return items, nil
				}, ancestryResult: func(_ context.Context, k string, opts ...services.CallOption) (traversal.Result, error) {
					reads++
					require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
					pi := item(k)
					if skip && k == "b" {
						pi.State = d.StateActive
					}
					return traversal.Result{StartKey: k, RootKey: k, Keys: []string{k}, Chain: map[string]d.ProcessInstance{k: pi}, Outcome: traversal.OutcomeComplete}, nil
				}, descendantsResult: func(_ context.Context, k string, opts ...services.CallOption) (traversal.Result, error) {
					reads++
					require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
					child := item(k + "c")
					child.ParentKey = k
					child.BusinessId = nil
					return traversal.Result{RootKey: k, Keys: []string{k, k + "c"}, Chain: map[string]d.ProcessInstance{k: item(k), k + "c": child}, Outcome: traversal.OutcomeComplete}, nil
				}, deleteProcessInstance: func(_ context.Context, k string, opts ...services.CallOption) (d.DeleteResponse, error) {
					require.Equal(t, 3, searches)
					require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
					writes = append(writes, k)
					return d.DeleteResponse{Ok: true, StatusCode: 204, MutationSubmitted: true}, nil
				}}
				api := NewWithRepairDependencies(nil, pi, repairIncidentAPI{}, nil, nil, stubJobAPI{}, toolx.V810)
				filter := d.ProcessInstanceFilter{BusinessId: new(" ref,*Ω ")}
				var scope d.CleanupFrozenScope
				var execute func(d.CleanupFrozenScope) error
				if kind == "retention" {
					req := d.RetentionPolicyRequest{Selection: filter, DryRun: true, Workers: 1, Limit: 2, RetentionDays: 30, DerivedEndDateBoundary: "2026-01-01"}
					planned, err := api.ExecuteRetentionPolicy(context.Background(), req)
					require.NoError(t, err)
					require.NotNil(t, planned.FrozenPlan)
					scope = planned.FrozenPlan.Scope
					if skip {
						require.Equal(t, []string{"b"}, []string(planned.DeletePlan.SkippedSeedKeys))
					}
					req.DryRun = false
					req.NoWait = true
					execute = func(s d.CleanupFrozenScope) error {
						p := *planned.FrozenPlan
						p.Scope = s
						req.FrozenPlan = &p
						_, err := api.ExecuteRetentionPolicy(context.Background(), req)
						return err
					}
				} else {
					req := d.OrphanPurgeRequest{Selection: filter, DryRun: true, Workers: 1, Limit: 2}
					planned, err := api.PurgeOrphanProcessInstances(context.Background(), req)
					require.NoError(t, err)
					require.NotNil(t, planned.FrozenPlan)
					scope = planned.FrozenPlan.Scope
					if skip {
						require.Equal(t, 1, planned.Discovery.Count)
						require.Len(t, scope.Selection.Keys, 2)
					}
					req.DryRun = false
					req.NoWait = true
					execute = func(s d.CleanupFrozenScope) error {
						p := *planned.FrozenPlan
						p.Scope = s
						req.FrozenPlan = &p
						_, err := api.PurgeOrphanProcessInstances(context.Background(), req)
						return err
					}
				}
				before := reads
				require.Empty(t, writes)
				require.Equal(t, d.ProcessInstanceSearchCompletionLimitReached, scope.Selection.Completion)
				for _, change := range []string{"completion", "tenant", "criteria", "members", "parents", "seeds", "expansion", "ancestry"} {
					t.Run(change, func(t *testing.T) {
						invalid := copyCleanupScope(scope)
						switch change {
						case "completion":
							invalid.Selection.Completion = d.ProcessInstanceSearchCompletionIncomplete
						case "tenant":
							invalid.Tenant = "other"
						case "criteria":
							invalid.Selection.Request.SearchRequest.Filter.BusinessId = new("other")
						case "members":
							invalid.Families[0].Keys = append(invalid.Families[0].Keys, "new")
						case "parents":
							invalid.Families[0].Items[1].ParentKey = "outside"
						case "seeds":
							invalid.Seeds = append(invalid.Seeds, "outside")
						case "ancestry":
							if kind != "retention" {
								return
							}
							invalid.Ancestries[len(invalid.Ancestries)-1].Keys = nil
						case "expansion":
							invalid.ExpansionOutcome = d.TraversalOutcomeUnresolved
						}
						require.Error(t, execute(invalid))
						require.Empty(t, writes)
					})
				}
				require.NoError(t, execute(scope))
				expected := []string{"ac", "a", "bc", "b"}
				if skip {
					expected = expected[:2]
				}
				require.Equal(t, expected, writes)
				require.Equal(t, before, reads)
				require.Equal(t, 3, searches)
				copied := copyCleanupScope(scope)
				copied.Selection.Items[0].Variables["nested"].(map[string]any)["value"] = "changed"
				require.Equal(t, "original", scope.Selection.Items[0].Variables["nested"].(map[string]any)["value"])
			})
		}
	}
}

// TestBusinessIDOperationsCleanupVersion rejects supplied criteria and bare-key bypasses before search or mutation.
func TestBusinessIDOperationsCleanupVersion(t *testing.T) {
	for _, v := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		for _, keys := range [][]string{nil, {}, {"a"}} {
			t.Run(fmt.Sprint(v, keys), func(t *testing.T) {
				api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{}, repairIncidentAPI{}, nil, nil, stubJobAPI{}, v)
				filter := d.ProcessInstanceFilter{BusinessId: new("id")}
				want := d.ErrUnsupported
				if v == toolx.V810 {
					want = d.ErrValidation
					if keys == nil {
						return
					}
				}
				_, err := api.ExecuteRetentionPolicy(context.Background(), d.RetentionPolicyRequest{Selection: filter, DiscoveredKeys: keys})
				require.ErrorIs(t, err, want)
				_, err = api.PurgeOrphanProcessInstances(context.Background(), d.OrphanPurgeRequest{Selection: filter, DiscoveredKeys: keys})
				require.ErrorIs(t, err, want)
			})
		}
	}
}

// TestBusinessIDOperationsCleanupFailure keeps partial evidence and stops each failed family before its root.
func TestBusinessIDOperationsCleanupFailure(t *testing.T) {
	for _, failure := range []string{"search", "children", "delete", "wait"} {
		t.Run(failure, func(t *testing.T) {
			var writes []string
			reads := 0
			pi := stubProcessInstanceAPI{searchPage: func(_ context.Context, _ d.ProcessInstanceFilter, p d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
				reads++
				if reads == 2 {
					return d.ProcessInstancePage{}, fmt.Errorf("%w: later search", d.ErrUpstream)
				}
				out := d.ProcessInstancePage{Request: p, Items: []d.ProcessInstance{{Key: "a", ParentKey: "missing"}, {Key: "b", ParentKey: "missing"}}, OverflowState: d.ProcessInstanceOverflowStateNoMore}
				if failure == "search" {
					out.OverflowState = d.ProcessInstanceOverflowStateHasMore
				}
				return out, nil
			}, filterOrphans: func(_ context.Context, items []d.ProcessInstance, _ ...services.CallOption) ([]d.ProcessInstance, error) {
				return items, nil
			}, descendantsResult: func(_ context.Context, k string, _ ...services.CallOption) (traversal.Result, error) {
				if failure == "children" {
					return traversal.Result{}, fmt.Errorf("%w: later child page", d.ErrUpstream)
				}
				return traversal.Result{RootKey: k, Keys: []string{k, k + "c"}, Chain: map[string]d.ProcessInstance{k: {Key: k, ParentKey: "missing", State: d.StateCompleted}, k + "c": {Key: k + "c", ParentKey: k, State: d.StateCompleted}}, Outcome: traversal.OutcomeComplete}, nil
			}, deleteProcessInstance: func(_ context.Context, k string, _ ...services.CallOption) (d.DeleteResponse, error) {
				writes = append(writes, k)
				return d.DeleteResponse{MutationSubmitted: failure == "wait"}, fmt.Errorf("%w: owned mutation failure", d.ErrUpstream)
			}}
			api := NewWithRepairDependencies(nil, pi, repairIncidentAPI{}, nil, nil, stubJobAPI{}, toolx.V810)
			got, err := api.PurgeOrphanProcessInstances(context.Background(), d.OrphanPurgeRequest{Selection: d.ProcessInstanceFilter{BusinessId: new("id")}, Workers: 1})
			require.ErrorIs(t, err, d.ErrUpstream)
			require.NotNil(t, got.Discovery.Selection)
			if slices.Contains([]string{"search", "children"}, failure) {
				require.Empty(t, writes)
			} else {
				require.Equal(t, []string{"ac", "bc"}, writes)
				require.Equal(t, failure == "wait", got.Deletion.Submitted)
			}
		})
	}
}

// TestBusinessIDOperationsCleanupRetentionFamilies verifies overlapping selected children expand one final root and preserve skipped ancestry.
func TestBusinessIDOperationsCleanupRetentionFamilies(t *testing.T) {
	items := map[string]d.ProcessInstance{"r": {Key: "r", State: d.StateCompleted}, "a": {Key: "a", ParentKey: "r", State: d.StateCompleted, EndDate: "2020-01-01"}, "b": {Key: "b", ParentKey: "r", State: d.StateCompleted, EndDate: "2020-01-01"}, "c": {Key: "c", ParentKey: "r", State: d.StateCompleted}, "s": {Key: "s", State: d.StateActive}, "z": {Key: "z", ParentKey: "s", State: d.StateCompleted, EndDate: "2020-01-01"}}
	expansions := 0
	var writes []string
	pi := stubProcessInstanceAPI{searchPage: func(_ context.Context, f d.ProcessInstanceFilter, p d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
		require.Equal(t, "2026-01-01", f.EndDateBefore)
		return d.ProcessInstancePage{Request: p, Items: []d.ProcessInstance{items["a"], items["b"], items["z"]}, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
	}, ancestryResult: func(_ context.Context, k string, _ ...services.CallOption) (traversal.Result, error) {
		root := items[k].ParentKey
		return traversal.Result{RootKey: root, StartKey: k, Keys: []string{k, root}, Chain: map[string]d.ProcessInstance{k: items[k], root: items[root]}, Outcome: traversal.OutcomeComplete}, nil
	}, descendantsResult: func(_ context.Context, k string, _ ...services.CallOption) (traversal.Result, error) {
		expansions++
		require.Equal(t, "r", k)
		return traversal.Result{RootKey: k, Keys: []string{"r", "a", "b", "c"}, Chain: items, Outcome: traversal.OutcomeComplete}, nil
	}, deleteProcessInstance: func(_ context.Context, k string, _ ...services.CallOption) (d.DeleteResponse, error) {
		writes = append(writes, k)
		return d.DeleteResponse{Ok: true, StatusCode: 204, MutationSubmitted: true}, nil
	}}
	api := NewWithRepairDependencies(nil, pi, nil, nil, nil, nil, toolx.V810)
	req := d.RetentionPolicyRequest{Selection: d.ProcessInstanceFilter{BusinessId: new("id")}, RetentionDays: 30, DerivedEndDateBoundary: "2026-01-01", DryRun: true, Workers: 1}
	plan, err := api.ExecuteRetentionPolicy(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, 1, expansions)
	require.Equal(t, []string{"z"}, []string(plan.DeletePlan.SkippedSeedKeys))
	require.Equal(t, []string{"r"}, []string(plan.DeletePlan.DuplicateKeys))
	require.Len(t, plan.DeletePlan.AffectedKeys, 4)
	req.FrozenPlan = plan.FrozenPlan
	req.DryRun = false
	req.DerivedEndDateBoundary = "2026-01-02"
	_, err = api.ExecuteRetentionPolicy(context.Background(), req)
	require.ErrorIs(t, err, d.ErrPrecondition)
	require.Empty(t, writes)
	req.DerivedEndDateBoundary = "2026-01-01"
	got, err := api.ExecuteRetentionPolicy(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b", "c", "r"}, writes)
	require.Equal(t, []string{"r"}, []string(got.Deletion.SubmittedRootKeys))
	require.Equal(t, 1, expansions)
}

// TestBusinessIDOperationsCleanupDryRunOption retains preview outcomes when library callers use the shared dry-run option.
func TestBusinessIDOperationsCleanupDryRunOption(t *testing.T) {
	pi := stubProcessInstanceAPI{searchPage: func(_ context.Context, _ d.ProcessInstanceFilter, p d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
		return d.ProcessInstancePage{Request: p, OverflowState: d.ProcessInstanceOverflowStateNoMore, Items: []d.ProcessInstance{{Key: "a", ParentKey: "missing", State: d.StateCompleted}}}, nil
	}, filterOrphans: func(_ context.Context, items []d.ProcessInstance, _ ...services.CallOption) ([]d.ProcessInstance, error) {
		return items, nil
	}, descendantsResult: func(_ context.Context, k string, _ ...services.CallOption) (traversal.Result, error) {
		return traversal.Result{RootKey: k, Keys: []string{k}, Chain: map[string]d.ProcessInstance{k: {Key: k, ParentKey: "missing", State: d.StateCompleted}}, Outcome: traversal.OutcomeComplete}, nil
	}}
	api := NewWithRepairDependencies(nil, pi, nil, nil, nil, nil, toolx.V810)
	result, err := api.PurgeOrphanProcessInstances(context.Background(), d.OrphanPurgeRequest{Selection: d.ProcessInstanceFilter{BusinessId: new("id")}}, services.WithDryRun())
	require.NoError(t, err)
	require.Equal(t, d.OrphanPurgeOutcomePlanned, result.Outcome)
	require.True(t, result.Request.DryRun)
	require.False(t, result.Deletion.Submitted)
	require.NotNil(t, result.FrozenPlan)
	retained, err := api.ExecuteRetentionPolicy(context.Background(), d.RetentionPolicyRequest{Selection: d.ProcessInstanceFilter{BusinessId: new("id")}}, services.WithDryRun())
	require.NoError(t, err)
	require.Equal(t, d.RetentionPolicyOutcomePlanned, retained.Outcome)
	require.True(t, retained.Request.DryRun)
	require.False(t, retained.Deletion.Submitted)
}
