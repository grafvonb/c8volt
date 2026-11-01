// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	tr "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// selectedActionAPI owns selection and state transitions; any unconfigured API call panics.
type selectedActionAPI struct {
	API
	t             *testing.T
	nodes         []d.ProcessInstance
	unsupported   bool
	reads         int
	expanded      testx.SafeSlice[string]
	writes        testx.SafeSlice[string]
	failTraversal bool
}

// ValidateProcessInstanceSearchFilter simulates an authoritative no-I/O adapter gate.
func (a *selectedActionAPI) ValidateProcessInstanceSearchFilter(d.ProcessInstanceFilter) error {
	if a.unsupported {
		return d.ErrUnsupported
	}
	return nil
}

// ProcessInstanceSearchTenant binds prepared actions to the discovery tenant.
func (a *selectedActionAPI) ProcessInstanceSearchTenant(opts ...services.CallOption) string {
	if services.ApplyCallOptions(opts).IgnoreTenant {
		return ""
	}
	return "tenant-a"
}

// SearchForProcessInstancesPage serves copied ordered observations under original tenant options.
func (a *selectedActionAPI) SearchForProcessInstancesPage(_ context.Context, _ d.ProcessInstanceFilter, p d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
	require.False(a.t, services.ApplyCallOptions(opts).IgnoreTenant)
	a.reads++
	return d.ProcessInstancePage{Request: p, Items: a.nodes, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
}

// GetProcessInstances supplies identity facts for exact-root variable planning without selection rediscovery.
func (a *selectedActionAPI) GetProcessInstances(_ context.Context, keys typex.Keys, _ int, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	require.True(a.t, services.ApplyCallOptions(opts).IgnoreTenant)
	out := make([]d.ProcessInstance, len(keys))
	for i, key := range keys {
		out[i] = d.ProcessInstance{Key: key, TenantId: "tenant-a"}
	}
	return out, nil
}

// ValidateProcessInstanceSuspension is a local capability check in the owned adapter.
func (a *selectedActionAPI) ValidateProcessInstanceSuspension() error { return nil }

// ValidateProcessInstanceResumption is a local capability check in the owned adapter.
func (a *selectedActionAPI) ValidateProcessInstanceResumption() error { return nil }

// SuspendProcessInstance records only actual submissions while preserving dry-run semantics.
func (a *selectedActionAPI) SuspendProcessInstance(_ context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceSuspensionResult, error) {
	require.True(a.t, services.ApplyCallOptions(opts).IgnoreTenant)
	if !services.ApplyCallOptions(opts).DryRun {
		a.writes.Append(key)
	}
	return d.ProcessInstanceSuspensionResult{Key: key, Submission: d.ProcessInstanceSuspensionSubmissionAccepted}, nil
}

// ResumeProcessInstance records only actual submissions while preserving dry-run semantics.
func (a *selectedActionAPI) ResumeProcessInstance(_ context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
	require.True(a.t, services.ApplyCallOptions(opts).IgnoreTenant)
	if !services.ApplyCallOptions(opts).DryRun {
		a.writes.Append(key)
	}
	return d.ProcessInstanceResumptionResult{Key: key, Submission: d.ProcessInstanceResumptionSubmissionAccepted}, nil
}

// DescendantsResult returns overlapping scopes with mixed eligibility, never ancestors or siblings.
func (a *selectedActionAPI) DescendantsResult(_ context.Context, key string, opts ...services.CallOption) (tr.Result, error) {
	require.True(a.t, services.ApplyCallOptions(opts).IgnoreTenant)
	a.expanded.Append(key)
	if a.failTraversal {
		return tr.Result{Outcome: tr.OutcomePartial}, d.ErrUpstream
	}
	nodes := a.nodes
	keys := typex.Keys{}
	chain := map[string]d.ProcessInstance{}
	found := false
	for _, node := range nodes {
		if node.Key == key {
			found = true
		}
		if found {
			keys = append(keys, node.Key)
			chain[node.Key] = node
		}
	}
	return tr.Result{Mode: tr.ModeDescendants, StartKey: key, RootKey: key, Keys: keys, Chain: chain, Outcome: tr.OutcomeComplete}, nil
}

// newSelectedActionAPI creates two overlapping selected instances and one terminal descendant.
func newSelectedActionAPI(t *testing.T, state d.State) *selectedActionAPI {
	t.Helper()
	keys := suspensionKeys(3)
	return &selectedActionAPI{t: t, nodes: []d.ProcessInstance{
		{Key: keys[0], TenantId: "tenant-a", State: state, BusinessId: toolx.Ptr("observed")},
		{Key: keys[1], ParentKey: keys[0], TenantId: "tenant-a", State: state},
		{Key: keys[2], ParentKey: keys[1], TenantId: "tenant-a", State: d.StateCompleted},
	}}
}

// TestBusinessIDActionsServiceState freezes exact versus eligible descendant scope and rejects altered evidence.
func TestBusinessIDActionsServiceState(t *testing.T) {
	for _, resume := range []bool{false, true} {
		for _, tree := range []bool{false, true} {
			state := d.StateActive
			if resume {
				state = d.StateSuspended
			}
			a := newSelectedActionAPI(t, state)
			request := selectedCancelRequest(2).Selection
			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			if resume {
				plan, err := PlanSelectedProcessInstanceResumption(context.Background(), a, nil, request, tree)
				require.NoError(t, err)
				require.Equal(t, suspensionKeys(2), plan.TargetKeys)
				bad := plan
				bad.TargetKeys = append(typex.Keys(nil), plan.TargetKeys...)
				bad.TargetKeys[0] = "unreviewed"
				_, err = ExecuteProcessInstanceResumptionPlan(context.Background(), a, log, bad, 1)
				require.ErrorIs(t, err, d.ErrPrecondition)
				_, err = ExecuteProcessInstanceResumptionPlan(context.Background(), a, log, plan, 1, services.WithIgnoreTenant())
				require.ErrorIs(t, err, d.ErrPrecondition)
				result, err := ExecuteProcessInstanceResumptionPlan(context.Background(), a, log, plan, 1, services.WithNoWait())
				require.NoError(t, err)
				require.NotNil(t, result.Selection)
				a.unsupported = true
				_, err = ExecuteProcessInstanceResumptionPlan(context.Background(), a, log, plan, 1, services.WithDryRun())
				require.ErrorIs(t, err, d.ErrUnsupported)
			} else {
				plan, err := PlanSelectedProcessInstanceSuspension(context.Background(), a, nil, request, tree)
				require.NoError(t, err)
				require.Equal(t, suspensionKeys(2), plan.TargetKeys)
				bad := plan
				bad.TargetKeys = append(typex.Keys(nil), plan.TargetKeys...)
				bad.TargetKeys[0] = "unreviewed"
				_, err = ExecuteProcessInstanceSuspensionPlan(context.Background(), a, log, bad, 1)
				require.ErrorIs(t, err, d.ErrPrecondition)
				_, err = ExecuteProcessInstanceSuspensionPlan(context.Background(), a, log, plan, 1, services.WithIgnoreTenant())
				require.ErrorIs(t, err, d.ErrPrecondition)
				result, err := ExecuteProcessInstanceSuspensionPlan(context.Background(), a, log, plan, 1, services.WithNoWait())
				require.NoError(t, err)
				require.NotNil(t, result.Selection)
				a.unsupported = true
				_, err = ExecuteProcessInstanceSuspensionPlan(context.Background(), a, log, plan, 1, services.WithDryRun())
				require.ErrorIs(t, err, d.ErrUnsupported)
			}
			require.Equal(t, 1, a.reads)
			require.Equal(t, suspensionKeys(2), typex.Keys(a.writes.Snapshot()))
			if tree {
				require.Equal(t, []string{suspensionKeys(1)[0]}, a.expanded.Snapshot())
			} else {
				require.Empty(t, a.expanded.Snapshot())
			}
		}
	}
}

// TestBusinessIDActionsServiceVariables preserves copied nested payloads and forbids incomplete or mismatched prepared selection.
func TestBusinessIDActionsServiceVariables(t *testing.T) {
	a := newSelectedActionAPI(t, d.StateActive)
	vars := &processInstanceVariableExecutionAPI{}
	updates := NewVariableUpdates(a, vars, config.New(), nil)
	request := selectedCancelRequest(2).Selection
	values := map[string]any{"foo": map[string]any{"nested": []any{"bar"}}}
	plan, err := PlanSelectedProcessInstanceVariableUpdates(context.Background(), a, nil, updates, request, values)
	require.NoError(t, err)
	values["foo"].(map[string]any)["nested"].([]any)[0] = "changed"
	*request.SearchRequest.Filter.BusinessId = "changed"
	*a.nodes[0].BusinessId = "changed"
	require.Equal(t, "ORDER-123", *plan.Selection.Request.SearchRequest.Filter.BusinessId)
	require.Equal(t, "observed", *plan.Selection.Items[0].BusinessId)
	for _, completion := range []d.ProcessInstanceSearchCompletion{"", d.ProcessInstanceSearchCompletionIncomplete, d.ProcessInstanceSearchCompletionVisitorStopped} {
		bad := copyProcessInstanceVariableUpdatePlan(plan)
		bad.Selection.Completion = completion
		_, err = updates.ExecuteProcessInstanceVariableUpdates(context.Background(), bad, 1)
		require.ErrorIs(t, err, d.ErrPrecondition)
	}
	_, err = updates.ExecuteProcessInstanceVariableUpdates(context.Background(), plan, 1, services.WithIgnoreTenant())
	require.ErrorIs(t, err, d.ErrPrecondition)
	results, err := updates.ExecuteProcessInstanceVariableUpdates(context.Background(), plan, 1, services.WithNoWait())
	require.NoError(t, err)
	require.Len(t, results.Items, 2)
	require.Equal(t, 1, a.reads)
	require.Empty(t, a.expanded.Snapshot())
	require.Len(t, vars.writes, 2)
	for _, write := range vars.writes {
		require.True(t, write.options.IgnoreTenant)
		require.Equal(t, map[string]any{"foo": map[string]any{"nested": []any{"bar"}}}, write.variables)
	}
	a.unsupported = true
	_, err = updates.ExecuteProcessInstanceVariableUpdates(context.Background(), plan, 1, services.WithDryRun())
	require.ErrorIs(t, err, d.ErrUnsupported)
}

// TestBusinessIDActionsServiceEmptyAndIncomplete gates prepared empty plans and failed descendant observations before mutation.
func TestBusinessIDActionsServiceEmptyAndIncomplete(t *testing.T) {
	a := newSelectedActionAPI(t, d.StateActive)
	a.nodes = nil
	request := selectedCancelRequest(0).Selection
	suspend, err := PlanSelectedProcessInstanceSuspension(context.Background(), a, nil, request, true)
	require.NoError(t, err)
	resume, err := PlanSelectedProcessInstanceResumption(context.Background(), a, nil, request, true)
	require.NoError(t, err)
	updates := NewVariableUpdates(a, &processInstanceVariableExecutionAPI{}, config.New(), nil)
	update, err := PlanSelectedProcessInstanceVariableUpdates(context.Background(), a, nil, updates, request, nil)
	require.NoError(t, err)
	sr, err := ExecuteProcessInstanceSuspensionPlan(context.Background(), a, nil, suspend, 1)
	require.NoError(t, err)
	require.NotNil(t, sr.Selection)
	rr, err := ExecuteProcessInstanceResumptionPlan(context.Background(), a, nil, resume, 1)
	require.NoError(t, err)
	require.NotNil(t, rr.Selection)
	ur, err := updates.ExecuteProcessInstanceVariableUpdates(context.Background(), update, 1)
	require.NoError(t, err)
	require.NotNil(t, ur.Selection)
	a.unsupported = true
	_, err = ExecuteProcessInstanceSuspensionPlan(context.Background(), a, nil, suspend, 1)
	require.ErrorIs(t, err, d.ErrUnsupported)
	_, err = ExecuteProcessInstanceResumptionPlan(context.Background(), a, nil, resume, 1)
	require.ErrorIs(t, err, d.ErrUnsupported)
	_, err = updates.ExecuteProcessInstanceVariableUpdates(context.Background(), update, 1)
	require.ErrorIs(t, err, d.ErrUnsupported)
	require.Empty(t, a.writes.Snapshot())
	a = newSelectedActionAPI(t, d.StateActive)
	a.failTraversal = true
	suspend, err = PlanSelectedProcessInstanceSuspension(context.Background(), a, nil, request, true)
	require.ErrorIs(t, err, d.ErrUpstream)
	_, err = ExecuteProcessInstanceSuspensionPlan(context.Background(), a, nil, suspend, 1)
	require.True(t, errors.Is(err, d.ErrPrecondition))
	require.Empty(t, a.writes.Snapshot())
}

// ValidateProcessInstanceStates retains normalized state input for the owned search adapter.
func (a *selectedActionAPI) ValidateProcessInstanceStates(states d.States) error { return nil }
