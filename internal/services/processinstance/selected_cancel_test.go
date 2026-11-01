// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	tr "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// selectedCancelAPI owns a two-family fixture, with duplicate/sparse seed pages and mutable outcomes.
type selectedCancelAPI struct {
	API
	t         *testing.T
	tenant    string
	pages     int
	expanded  testx.SafeSlice[string]
	canceled  testx.SafeSlice[string]
	failure   error
	failPhase string
	nodes     map[string]d.ProcessInstance
}

// newSelectedCancelAPI constructs the contract graph without any live dependencies.
func newSelectedCancelAPI(t *testing.T) *selectedCancelAPI {
	a := &selectedCancelAPI{t: t, tenant: "tenant-a", nodes: map[string]d.ProcessInstance{}}
	for key, parent := range map[string]string{"RA": "", "A1": "RA", "A1a": "A1", "A2": "RA", "RB": "", "B1": "RB", "B2": "RB"} {
		a.nodes[key] = d.ProcessInstance{Key: key, ParentKey: parent, TenantId: "tenant-a", State: d.StateActive, Variables: map[string]any{"nested": map[string]any{"value": "original"}}}
	}
	for _, key := range []string{"A1", "A2", "B1"} {
		item := a.nodes[key]
		item.BusinessId = toolx.Ptr("ORDER-123")
		a.nodes[key] = item
	}
	return a
}

// ValidateProcessInstanceSearchFilter simulates version rejection before any fixture I/O.
func (a *selectedCancelAPI) ValidateProcessInstanceSearchFilter(d.ProcessInstanceFilter) error {
	if a.failPhase == "version" {
		return d.ErrUnsupported
	}
	return nil
}

// ProcessInstanceSearchTenant exposes the effective tenant for context-mismatch tests.
func (a *selectedCancelAPI) ProcessInstanceSearchTenant(opts ...services.CallOption) string {
	if services.ApplyCallOptions(opts).IgnoreTenant {
		return ""
	}
	return a.tenant
}

// SearchForProcessInstancesPage verifies scoped discovery and supplies duplicate and sparse pages.
func (a *selectedCancelAPI) SearchForProcessInstancesPage(_ context.Context, f d.ProcessInstanceFilter, p d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
	require.False(a.t, services.ApplyCallOptions(opts).IgnoreTenant)
	require.Equal(a.t, "ORDER-123", *f.BusinessId)
	a.pages++
	if a.pages == 1 {
		return d.ProcessInstancePage{Request: p, Items: []d.ProcessInstance{a.nodes["A1"], a.nodes["A1"]}, OverflowState: d.ProcessInstanceOverflowStateHasMore, EndCursor: "second"}, nil
	}
	if a.failPhase == "search" {
		return d.ProcessInstancePage{}, a.failure
	}
	if a.pages == 2 {
		return d.ProcessInstancePage{Request: p, OverflowState: d.ProcessInstanceOverflowStateHasMore, EndCursor: "third"}, nil
	}
	return d.ProcessInstancePage{Request: p, Items: []d.ProcessInstance{a.nodes["A2"], a.nodes["B1"]}, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
}

// AncestryResult resolves each selected seed without bypassing discovery first.
func (a *selectedCancelAPI) AncestryResult(_ context.Context, key string, opts ...services.CallOption) (tr.Result, error) {
	require.True(a.t, services.ApplyCallOptions(opts).IgnoreTenant)
	root := a.nodes[key].ParentKey
	result := tr.Result{StartKey: key, RootKey: root, Keys: []string{key, root}, Chain: map[string]d.ProcessInstance{key: a.nodes[key], root: a.nodes[root]}, Outcome: tr.OutcomeComplete}
	if a.failPhase == "ancestry" {
		result.Outcome = tr.OutcomePartial
		result.MissingAncestors = []tr.MissingAncestor{{Key: "missing", StartKey: key}}
	}
	return result, nil
}

// DescendantsResult records one expansion per root and can return incomplete evidence with an error.
func (a *selectedCancelAPI) DescendantsResult(_ context.Context, root string, opts ...services.CallOption) (tr.Result, error) {
	require.True(a.t, services.ApplyCallOptions(opts).IgnoreTenant)
	a.expanded.Append(root)
	keys := []string{"RA", "A1", "A1a", "A2"}
	if root == "RB" {
		keys = []string{"RB", "B1", "B2"}
	}
	result := tr.Result{RootKey: root, Keys: keys, Chain: a.nodes, Outcome: tr.OutcomeComplete}
	if a.failPhase == "children" {
		result.Outcome = tr.OutcomeUnresolved
		return result, a.failure
	}
	if a.failPhase == "unknown" {
		result.Outcome = ""
	}
	return result, nil
}

// CancelProcessInstance proves frozen root submission and retains partial failures.
func (a *selectedCancelAPI) CancelProcessInstance(_ context.Context, key string, opts ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
	require.True(a.t, services.ApplyCallOptions(opts).IgnoreTenant)
	a.canceled.Append(key)
	if a.failPhase == "cancel" && key == "RB" {
		return d.CancelResponse{}, nil, a.failure
	}
	return d.CancelResponse{Ok: true, StatusCode: 204}, nil, nil
}

// selectedCancelRequest retains the original opaque criterion and bounded collection controls.
func selectedCancelRequest(limit int32) d.ProcessInstanceSelectedMutationRequest {
	return d.ProcessInstanceSelectedMutationRequest{Operation: "cancel", Workers: 1, Selection: d.ProcessInstanceSelectionRequest{Cardinality: d.ProcessInstanceSelectionCollection, SearchRequest: d.ProcessInstanceSearchRequest{Filter: d.ProcessInstanceFilter{BusinessId: toolx.Ptr("ORDER-123")}, Limit: limit}}}
}

// TestBusinessIDCancelFrozenFamilies verifies distinct seeds, global expansion, copies and frozen partial execution.
func TestBusinessIDCancelFrozenFamilies(t *testing.T) {
	for _, limit := range []int32{0, 2} {
		t.Run(string(rune('0'+limit)), func(t *testing.T) {
			a := newSelectedCancelAPI(t)
			request := selectedCancelRequest(limit)
			plan, err := PlanProcessInstanceSelectedMutation(context.Background(), a, nil, request)
			require.NoError(t, err)
			seeds := []string{"A1", "A2", "B1"}
			roots := []string{"RA", "RB"}
			members := []string{"RA", "A1", "A1a", "A2", "RB", "B1", "B2"}
			if limit == 2 {
				seeds = seeds[:2]
				roots = roots[:1]
				members = members[:4]
				require.Equal(t, d.ProcessInstanceSearchCompletionLimitReached, plan.Selection.Completion)
			}
			require.Equal(t, seeds, []string(plan.Selection.Keys))
			require.Equal(t, roots, []string(plan.Expansion.Roots))
			require.Equal(t, members, []string(plan.Expansion.Collected))
			require.Equal(t, roots, a.expanded.Snapshot())
			require.Empty(t, a.canceled.Snapshot())
			*request.Selection.SearchRequest.Filter.BusinessId = "changed"
			*a.nodes["A1"].BusinessId = "changed"
			a.nodes["A1"].Variables["nested"].(map[string]any)["value"] = "changed"
			require.Equal(t, "ORDER-123", *plan.Request.Selection.SearchRequest.Filter.BusinessId)
			require.Equal(t, "ORDER-123", *plan.Selection.Items[0].BusinessId)
			require.Equal(t, "original", plan.Families[0].Items[1].Variables["nested"].(map[string]any)["value"])
			reads := a.pages
			a.failPhase = "cancel"
			a.failure = errors.New("root failure")
			reports, err := ExecuteProcessInstanceCancellationPlan(context.Background(), a, slog.New(slog.NewTextHandler(io.Discard, nil)), plan, services.WithNoWait())
			if limit == 0 {
				require.ErrorIs(t, err, a.failure)
				require.False(t, reports[1].Ok)
			} else {
				require.NoError(t, err)
			}
			require.True(t, reports[0].Ok)
			require.Equal(t, roots, a.canceled.Snapshot())
			require.Equal(t, reads, a.pages)
		})
	}
}

// TestBusinessIDCancelIncompleteNeverExecutes rejects failed discovery and every inconsistent readiness proof.
func TestBusinessIDCancelIncompleteNeverExecutes(t *testing.T) {
	for _, phase := range []string{"search", "children", "ancestry", "unknown"} {
		t.Run(phase, func(t *testing.T) {
			a := newSelectedCancelAPI(t)
			a.failPhase = phase
			a.failure = errors.New("late failure")
			plan, err := PlanProcessInstanceSelectedMutation(context.Background(), a, nil, selectedCancelRequest(0))
			require.Error(t, err)
			if phase == "search" || phase == "children" {
				require.ErrorIs(t, err, a.failure)
			}
			require.NotEmpty(t, plan.Selection.Keys)
			_, err = ExecuteProcessInstanceCancellationPlan(context.Background(), a, nil, plan)
			require.Error(t, err)
			require.Empty(t, a.canceled.Snapshot())
		})
	}
	for _, change := range []string{"operation", "criteria", "tenant", "ignore", "completion", "members", "parent", "roots", "unselected family", "empty", "version"} {
		t.Run(change, func(t *testing.T) {
			a := newSelectedCancelAPI(t)
			plan, err := PlanProcessInstanceSelectedMutation(context.Background(), a, nil, selectedCancelRequest(0))
			require.NoError(t, err)
			var opts []services.CallOption
			switch change {
			case "operation":
				plan.Request.Operation = "delete"
			case "criteria":
				plan.Request.Selection.SearchRequest.Filter.BusinessId = toolx.Ptr("other")
			case "tenant":
				a.tenant = "tenant-b"
			case "ignore":
				opts = append(opts, services.WithIgnoreTenant())
			case "completion":
				plan.Selection.Completion = ""
			case "members":
				plan.Expansion.Collected = plan.Expansion.Collected[:2]
			case "parent":
				plan.Families[0].Items[1].ParentKey = "new-root"
			case "roots":
				plan.Expansion.Roots = append(plan.Expansion.Roots, "new-root")
			case "unselected family":
				plan.Expansion.Roots = append(plan.Expansion.Roots, "RC")
				plan.Expansion.Collected = append(plan.Expansion.Collected, "RC")
				plan.Families = append(plan.Families, d.ProcessInstanceMutationFamily{RootKey: "RC", Keys: []string{"RC"}, Items: []d.ProcessInstance{{Key: "RC", TenantId: "tenant-a"}}})
			case "empty":
				plan.Selection.Keys = nil
				plan.Selection.Items = nil
			case "version":
				a.failPhase = "version"
			}
			_, err = ExecuteProcessInstanceCancellationPlan(context.Background(), a, nil, plan, opts...)
			require.Error(t, err)
			require.Empty(t, a.canceled.Snapshot())
			if change == "version" {
				require.ErrorIs(t, err, d.ErrUnsupported)
			}
		})
	}
}

// ValidateProcessInstanceStates permits fixture states without backend calls.
func (*selectedCancelAPI) ValidateProcessInstanceStates(d.States) error { return nil }
