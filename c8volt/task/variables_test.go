// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// facadeVariableUserTaskAPI exposes only effective-variable pages so facade
// tests fail if enrichment re-fetches tasks or selects another workflow.
type facadeVariableUserTaskAPI struct {
	variablePage func(context.Context, string, d.UserTaskVariablePageRequest, ...services.CallOption) (d.UserTaskVariablePage, error)
}

// GetUserTask rejects legacy task re-fetches during selected-task enrichment.
func (facadeVariableUserTaskAPI) GetUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	return d.UserTask{}, errors.New("unexpected legacy user-task read")
}

// GetNativeUserTask rejects native task re-fetches during selected-task enrichment.
func (facadeVariableUserTaskAPI) GetNativeUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	return d.UserTask{}, errors.New("unexpected native user-task read")
}

// SearchUserTasksPage rejects discovery during selected-task enrichment.
func (facadeVariableUserTaskAPI) SearchUserTasksPage(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error) {
	return d.UserTaskSearchPage{}, errors.New("unexpected user-task search")
}

// SearchUserTaskEffectiveVariablesPage delegates the one allowed operation to the fixture.
func (a facadeVariableUserTaskAPI) SearchUserTaskEffectiveVariablesPage(ctx context.Context, key string, page d.UserTaskVariablePageRequest, opts ...services.CallOption) (d.UserTaskVariablePage, error) {
	return a.variablePage(ctx, key, page, opts...)
}

// TestClientEnrichUserTasksWithVariablesMapsOptionsAndFields verifies the
// facade delegates selected tasks once, preserves their order and metadata,
// and maps every effective-variable field without task re-fetches.
func TestClientEnrichUserTasksWithVariablesMapsOptionsAndFields(t *testing.T) {
	t.Parallel()

	candidateUsers := []string{"alice"}
	calls := make([]string, 0, 2)
	api := facadeVariableUserTaskAPI{variablePage: func(_ context.Context, key string, page d.UserTaskVariablePageRequest, opts ...services.CallOption) (d.UserTaskVariablePage, error) {
		calls = append(calls, key)
		require.Equal(t, d.UserTaskVariablePageRequest{Size: 1000}, page)
		cfg := services.ApplyCallOptions(opts)
		require.True(t, cfg.IgnoreTenant)
		require.True(t, cfg.Verbose)

		items := []d.ProcessInstanceVariable{}
		if key == "task-a" {
			items = []d.ProcessInstanceVariable{{
				Name: "local", Value: "{\"approved\":true}", VariableKey: "variable-a",
				ProcessInstanceKey: "process-a", ScopeKey: "task-scope-a",
				TenantId: "foreign-tenant", APITruncated: true,
			}}
		}
		return d.UserTaskVariablePage{
			Items: items, Request: page, RawItemCount: int32(len(items)),
			ReportedTotal: d.UserTaskReportedTotal{Count: int64(len(items)), Kind: d.UserTaskReportedTotalKindExact},
		}, nil
	}}
	client := New(nil, nil, api, nil)
	input := UserTasks{Total: 99, Items: []UserTask{
		{
			Key: "task-a", State: "CREATED", Name: "Approve", ElementId: "approve_invoice",
			ElementInstanceKey: "element-a", Assignee: "bob", CandidateUsers: candidateUsers,
			CandidateGroups: []string{"accounting"}, ProcessInstanceKey: "process-a",
			ProcessDefinitionKey: "definition-a", ProcessDefinitionId: "invoice",
			ProcessDefinitionVersion: 3, TenantId: "task-tenant-a",
		},
		{Key: "task-b", State: "COMPLETED", ProcessInstanceKey: "process-b", ProcessDefinitionVersion: 7},
	}}

	got, err := client.EnrichUserTasksWithVariables(context.Background(), input, options.WithIgnoreTenant(), options.WithVerbose())
	require.NoError(t, err)
	require.Equal(t, []string{"task-a", "task-b"}, calls)
	require.Equal(t, int64(2), got.Total)
	require.Equal(t, input.Items, []UserTask{got.Items[0].Item, got.Items[1].Item})
	require.Equal(t, []UserTaskVariable{{
		Name: "local", Value: "{\"approved\":true}", VariableKey: "variable-a",
		ProcessInstanceKey: "process-a", ScopeKey: "task-scope-a",
		TenantId: "foreign-tenant", APITruncated: true,
	}}, got.Items[0].Variables)
	require.NotNil(t, got.Items[1].Variables)
	require.Empty(t, got.Items[1].Variables)

	candidateUsers[0] = "changed"
	require.Equal(t, []string{"alice"}, got.Items[0].Item.CandidateUsers)
}

// TestClientEnrichUserTasksWithVariablesInitializesEmptyJSON verifies nil input
// produces the stable successful empty collection without service requests.
func TestClientEnrichUserTasksWithVariablesInitializesEmptyJSON(t *testing.T) {
	t.Parallel()

	calls := 0
	api := facadeVariableUserTaskAPI{variablePage: func(context.Context, string, d.UserTaskVariablePageRequest, ...services.CallOption) (d.UserTaskVariablePage, error) {
		calls++
		return d.UserTaskVariablePage{}, errors.New("unexpected variable request")
	}}
	client := New(nil, nil, api, nil)

	got, err := client.EnrichUserTasksWithVariables(context.Background(), UserTasks{})
	require.NoError(t, err)
	require.Equal(t, 0, calls)
	require.Equal(t, int64(0), got.Total)
	require.NotNil(t, got.Items)

	raw, err := json.Marshal(got)
	require.NoError(t, err)
	require.JSONEq(t, `{"total":0,"items":[]}`, string(raw))
}

// TestClientEnrichUserTasksWithVariablesInitializesVariableJSON verifies a
// returned task with no variables uses an initialized variables array.
func TestClientEnrichUserTasksWithVariablesInitializesVariableJSON(t *testing.T) {
	t.Parallel()

	api := facadeVariableUserTaskAPI{variablePage: func(_ context.Context, _ string, page d.UserTaskVariablePageRequest, _ ...services.CallOption) (d.UserTaskVariablePage, error) {
		return d.UserTaskVariablePage{
			Items: []d.ProcessInstanceVariable{}, Request: page,
			ReportedTotal: d.UserTaskReportedTotal{Kind: d.UserTaskReportedTotalKindExact},
		}, nil
	}}
	client := New(nil, nil, api, nil)

	got, err := client.EnrichUserTasksWithVariables(context.Background(), UserTasks{Items: []UserTask{{Key: "task-a", State: "CREATED", ProcessInstanceKey: "process-a"}}})
	require.NoError(t, err)
	raw, err := json.Marshal(got)
	require.NoError(t, err)
	require.JSONEq(t, `{"total":1,"items":[{"item":{"key":"task-a","state":"CREATED","processInstanceKey":"process-a"},"variables":[]}]}`, string(raw))
}

// TestClientEnrichUserTasksWithVariablesConvertsErrors verifies retrieval
// failures cross the public boundary through established facade classes.
func TestClientEnrichUserTasksWithVariablesConvertsErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		domainErr error
		facadeErr error
	}{
		{name: "not found", domainErr: fmt.Errorf("variables disappeared: %w", d.ErrNotFound), facadeErr: ferr.ErrNotFound},
		{name: "malformed", domainErr: fmt.Errorf("variables invalid: %w", d.ErrMalformedResponse), facadeErr: ferr.ErrMalformedResponse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			api := facadeVariableUserTaskAPI{variablePage: func(context.Context, string, d.UserTaskVariablePageRequest, ...services.CallOption) (d.UserTaskVariablePage, error) {
				return d.UserTaskVariablePage{}, tt.domainErr
			}}
			client := New(nil, nil, api, nil)

			got, err := client.EnrichUserTasksWithVariables(context.Background(), UserTasks{Items: []UserTask{{Key: "task-a"}}})
			require.Equal(t, VariableEnrichedUserTasks{}, got)
			require.ErrorIs(t, err, tt.facadeErr)
			require.ErrorContains(t, err, tt.domainErr.Error())
		})
	}
}
