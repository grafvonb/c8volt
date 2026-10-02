// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"context"
	"errors"
	"fmt"
	"testing"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

type facadeVariableUpdateAPI struct {
	plan             d.UserTaskVariableUpdatePlan
	planErr          error
	results          d.UserTaskVariableUpdateResults
	executeErr       error
	plannedKeys      typex.Keys
	plannedVariables map[string]any
	executedPlan     d.UserTaskVariableUpdatePlan
	wantedWorkers    int
	planCfg          *services.CallCfg
	executeCfg       *services.CallCfg
}

// PlanUserTaskVariableUpdates records planning inputs and call options before returning the
// configured plan or error.
func (a *facadeVariableUpdateAPI) PlanUserTaskVariableUpdates(_ context.Context, keys typex.Keys, variables map[string]any, opts ...services.CallOption) (d.UserTaskVariableUpdatePlan, error) {
	a.plannedKeys = append(typex.Keys(nil), keys...)
	a.plannedVariables = variables
	a.planCfg = services.ApplyCallOptions(opts)
	return a.plan, a.planErr
}

// ExecuteUserTaskVariableUpdates records the frozen plan, worker count, and call options before
// returning configured execution outcomes.
func (a *facadeVariableUpdateAPI) ExecuteUserTaskVariableUpdates(_ context.Context, plan d.UserTaskVariableUpdatePlan, wantedWorkers int, opts ...services.CallOption) (d.UserTaskVariableUpdateResults, error) {
	a.executedPlan = plan
	a.wantedWorkers = wantedWorkers
	a.executeCfg = services.ApplyCallOptions(opts)
	return a.results, a.executeErr
}

// TestVariableUpdateFacadeDelegatesAndCopiesNestedValues verifies both facade
// directions own JSON-like mutable values and preserve option configuration.
func TestVariableUpdateFacadeDelegatesAndCopiesNestedValues(t *testing.T) {
	t.Parallel()

	serviceValue := map[string]any{"nested": []any{map[string]any{"enabled": true}}}
	api := &facadeVariableUpdateAPI{plan: d.UserTaskVariableUpdatePlan{
		RequestedKeys: []string{"task-a"},
		UserTasks: []d.UserTaskVariablePlan{{
			UserTaskKey: "task-a", ElementInstanceKey: "element-a", TenantId: "tenant-a",
			Additions:       []d.UserTaskVariablePlannedValue{{Name: "payload", ScopeKey: "element-a", Value: serviceValue}},
			TargetScopeKeys: []string{"element-a"},
		}},
		Targets: []d.ScopeVariableUpdateTarget{{
			ScopeKey: "element-a", TenantId: "tenant-a", Variables: map[string]any{"payload": serviceValue},
			Associations: []d.ScopeVariableUpdateAssociation{{UserTaskKey: "task-a", Names: []string{"payload"}}},
		}},
		RequestedCount: 1, UpdateCount: 1, VariableAddCount: 1,
		TenantContext: d.TenantContext{
			Mode: d.TenantContextModeExplicitKeys, Filter: d.TenantContextFilterNotApplied,
			ResolvedTenantIDs: []string{"tenant-a"},
			Warnings:          []d.TenantContextWarning{{Code: d.TenantContextWarningUnknownTargetTenants, Message: "warning"}},
		},
	}}
	client := NewWithVariableUpdates(nil, nil, nil, api, nil)
	inputValue := map[string]any{"items": []any{map[string]any{"count": float64(1)}}}
	input := map[string]any{"payload": inputValue}

	plan, err := client.PlanUserTaskVariableUpdates(context.Background(), typex.Keys{"task-a"}, input, options.WithFailFast())
	require.NoError(t, err)
	require.True(t, api.planCfg.FailFast)
	require.Equal(t, typex.Keys{"task-a"}, api.plannedKeys)

	inputValue["items"].([]any)[0].(map[string]any)["count"] = float64(2)
	require.Equal(t, float64(1), api.plannedVariables["payload"].(map[string]any)["items"].([]any)[0].(map[string]any)["count"])
	serviceValue["nested"].([]any)[0].(map[string]any)["enabled"] = false
	require.Equal(t, true, plan.UserTasks[0].Additions[0].Value.(map[string]any)["nested"].([]any)[0].(map[string]any)["enabled"])
	require.Equal(t, true, plan.Targets[0].Variables["payload"].(map[string]any)["nested"].([]any)[0].(map[string]any)["enabled"])

	plan.Targets[0].Variables["payload"].(map[string]any)["nested"].([]any)[0].(map[string]any)["enabled"] = "public-change"
	_, err = client.ExecuteUserTaskVariableUpdates(context.Background(), plan, 3, options.WithNoWait(), options.WithNoWorkerLimit())
	require.NoError(t, err)
	require.Equal(t, 3, api.wantedWorkers)
	require.True(t, api.executeCfg.NoWait)
	require.True(t, api.executeCfg.NoWorkerLimit)

	plan.Targets[0].Variables["payload"].(map[string]any)["nested"].([]any)[0].(map[string]any)["enabled"] = "changed-after-call"
	require.Equal(t, "public-change", api.executedPlan.Targets[0].Variables["payload"].(map[string]any)["nested"].([]any)[0].(map[string]any)["enabled"])
	require.Equal(t, []string{"tenant-a"}, plan.TenantContext.ResolvedTenantIDs)
	require.Equal(t, "warning", plan.TenantContext.Warnings[0].Message)
}

// TestVariableUpdateFacadePreservesPartialResultsAndNormalizesErrors verifies
// every execution state survives conversion even when the service also fails.
func TestVariableUpdateFacadePreservesPartialResultsAndNormalizesErrors(t *testing.T) {
	t.Parallel()

	domainErr := fmt.Errorf("submit one target: %w", d.ErrUpstream)
	api := &facadeVariableUpdateAPI{
		results: d.UserTaskVariableUpdateResults{Items: []d.UserTaskVariableUpdateResult{
			{Key: "confirmed", Status: d.UserTaskVariableUpdateStatusConfirmed, MutationAccepted: true, ConfirmationStatus: "confirmed"},
			{Key: "submitted", Status: d.UserTaskVariableUpdateStatusSubmitted, MutationAccepted: true, ConfirmationStatus: "skipped"},
			{Key: "mutation-failed", Status: d.UserTaskVariableUpdateStatusMutationFailed, Error: "write failed", Scopes: []d.ScopeVariableUpdateOutcome{{ScopeKey: "scope-a", Names: []string{"a"}, Status: d.ScopeVariableUpdateStatusMutationFailed, Error: "write failed"}}},
			{Key: "confirmation-failed", Status: d.UserTaskVariableUpdateStatusConfirmationFailed, MutationAccepted: true, ConfirmationStatus: "failed"},
			{Key: "unchanged", Status: d.UserTaskVariableUpdateStatusUnchanged, ConfirmationStatus: "skipped"},
			{Key: "skipped", Status: d.UserTaskVariableUpdateStatusSkipped, ConfirmationStatus: "skipped"},
		}},
		executeErr: domainErr,
	}
	client := NewWithVariableUpdates(nil, nil, nil, api, nil)

	got, err := client.ExecuteUserTaskVariableUpdates(context.Background(), UserTaskVariableUpdatePlan{}, 1)
	require.ErrorIs(t, err, ferr.ErrInternal)
	require.ErrorContains(t, err, domainErr.Error())
	require.Equal(t, []UserTaskVariableUpdateStatus{
		UserTaskVariableUpdateStatusConfirmed,
		UserTaskVariableUpdateStatusSubmitted,
		UserTaskVariableUpdateStatusMutationFailed,
		UserTaskVariableUpdateStatusConfirmationFailed,
		UserTaskVariableUpdateStatusUnchanged,
		UserTaskVariableUpdateStatusSkipped,
	}, []UserTaskVariableUpdateStatus{got.Items[0].Status, got.Items[1].Status, got.Items[2].Status, got.Items[3].Status, got.Items[4].Status, got.Items[5].Status})
	require.Equal(t, ScopeVariableUpdateStatusMutationFailed, got.Items[2].Scopes[0].Status)

	api.results.Items[2].Scopes[0].Names[0] = "service-change"
	require.Equal(t, []string{"a"}, got.Items[2].Scopes[0].Names)
}

// TestLegacyTaskConstructorRejectsVariableUpdates verifies the compatible read
// constructor fails update operations as a normalized local precondition.
func TestLegacyTaskConstructorRejectsVariableUpdates(t *testing.T) {
	t.Parallel()

	client := New(nil, nil, nil, nil)
	_, planErr := client.PlanUserTaskVariableUpdates(context.Background(), typex.Keys{"task-a"}, map[string]any{"a": 1})
	require.ErrorIs(t, planErr, ferr.ErrLocalPrecondition)
	require.ErrorContains(t, planErr, "requires a variable update service")

	_, executeErr := client.ExecuteUserTaskVariableUpdates(context.Background(), UserTaskVariableUpdatePlan{}, 1)
	require.ErrorIs(t, executeErr, ferr.ErrLocalPrecondition)
	require.ErrorContains(t, executeErr, "requires a variable update service")
	require.False(t, errors.Is(planErr, ferr.ErrInternal))
}
