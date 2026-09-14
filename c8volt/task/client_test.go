// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// facadeUserTaskAPI records native and legacy calls made across the public task boundary.
type facadeUserTaskAPI struct {
	mu          sync.Mutex
	tasks       map[string]d.UserTask
	errors      map[string]error
	nativeKeys  []string
	legacyKeys  []string
	callConfigs []*services.CallCfg
}

// GetUserTask records legacy resolver access so direct facade reads can prove they do not select it.
func (a *facadeUserTaskAPI) GetUserTask(_ context.Context, key string, _ ...services.CallOption) (d.UserTask, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.legacyKeys = append(a.legacyKeys, key)
	return d.UserTask{}, errors.New("legacy resolver selected")
}

// GetNativeUserTask returns configured native results and records translated call options.
func (a *facadeUserTaskAPI) GetNativeUserTask(_ context.Context, key string, opts ...services.CallOption) (d.UserTask, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.nativeKeys = append(a.nativeKeys, key)
	a.callConfigs = append(a.callConfigs, services.ApplyCallOptions(opts))
	if err := a.errors[key]; err != nil {
		return d.UserTask{}, err
	}
	return a.tasks[key], nil
}

// SearchUserTasksPage rejects search until the facade search contract is exercised by its dedicated work unit.
func (a *facadeUserTaskAPI) SearchUserTasksPage(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error) {
	panic("unexpected user-task search")
}

// snapshot returns independently owned observations from the facade service stub.
func (a *facadeUserTaskAPI) snapshot() ([]string, []string, []*services.CallCfg) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.nativeKeys...), append([]string(nil), a.legacyKeys...), append([]*services.CallCfg(nil), a.callConfigs...)
}

// TestClientGetUserTaskUsesNativeReadAndCopiesCandidates verifies single reads select the new native method and map all public identity fields.
func TestClientGetUserTaskUsesNativeReadAndCopiesCandidates(t *testing.T) {
	t.Parallel()

	candidateUsers := []string{"alice"}
	candidateGroups := []string{"accounting"}
	api := &facadeUserTaskAPI{tasks: map[string]d.UserTask{
		"task-a": {
			Key: "task-a", State: "CREATED", Name: "Approve invoice", ElementId: "approve_invoice",
			ElementInstanceKey: "element-a", Assignee: "bob", CandidateUsers: candidateUsers,
			CandidateGroups: candidateGroups, ProcessInstanceKey: "process-a", ProcessDefinitionKey: "definition-a",
			ProcessDefinitionId: "invoice", TenantId: "foreign-tenant",
		},
	}}
	client := New(nil, nil, api, nil)

	got, err := client.GetUserTask(context.Background(), "task-a", options.WithFailFast(), options.WithNoWorkerLimit())
	require.NoError(t, err)
	require.Equal(t, UserTask{
		Key: "task-a", State: "CREATED", Name: "Approve invoice", ElementId: "approve_invoice",
		ElementInstanceKey: "element-a", Assignee: "bob", CandidateUsers: []string{"alice"},
		CandidateGroups: []string{"accounting"}, ProcessInstanceKey: "process-a", ProcessDefinitionKey: "definition-a",
		ProcessDefinitionId: "invoice", TenantId: "foreign-tenant",
	}, got)

	candidateUsers[0] = "changed-user"
	candidateGroups[0] = "changed-group"
	require.Equal(t, []string{"alice"}, got.CandidateUsers)
	require.Equal(t, []string{"accounting"}, got.CandidateGroups)

	nativeKeys, legacyKeys, configs := api.snapshot()
	require.Equal(t, []string{"task-a"}, nativeKeys)
	require.Empty(t, legacyKeys)
	require.Len(t, configs, 1)
	require.True(t, configs[0].FailFast)
	require.True(t, configs[0].NoWorkerLimit)
}

// TestClientGetUserTasksDelegatesStrictBulkAndShapesCollections verifies stable bulk delegation, option propagation, and one-or-many collection shape.
func TestClientGetUserTasksDelegatesStrictBulkAndShapesCollections(t *testing.T) {
	t.Parallel()

	api := &facadeUserTaskAPI{tasks: map[string]d.UserTask{
		"task-a": {Key: "task-a", State: "CREATED", ProcessInstanceKey: "process-a", CandidateUsers: []string{"alice"}},
		"task-b": {Key: "task-b", State: "COMPLETED", ProcessInstanceKey: "process-b", CandidateGroups: []string{"ops"}},
	}}
	client := New(nil, nil, api, nil)

	one, err := client.GetUserTasks(context.Background(), typex.Keys{"task-a", "task-a"}, 1, options.WithFailFast())
	require.NoError(t, err)
	require.Equal(t, int64(1), one.Total)
	require.Equal(t, []UserTask{{Key: "task-a", State: "CREATED", ProcessInstanceKey: "process-a", CandidateUsers: []string{"alice"}}}, one.Items)

	many, err := client.GetUserTasks(context.Background(), typex.Keys{"task-b", "task-a", "task-b"}, 1, options.WithNoWorkerLimit())
	require.NoError(t, err)
	require.Equal(t, int64(2), many.Total)
	require.Equal(t, []string{"task-b", "task-a"}, []string{many.Items[0].Key, many.Items[1].Key})

	api.tasks["task-a"].CandidateUsers[0] = "changed-user"
	api.tasks["task-b"].CandidateGroups[0] = "changed-group"
	require.Equal(t, []string{"alice"}, one.Items[0].CandidateUsers)
	require.Equal(t, []string{"ops"}, many.Items[0].CandidateGroups)

	nativeKeys, legacyKeys, configs := api.snapshot()
	require.Equal(t, []string{"task-a", "task-b", "task-a"}, nativeKeys)
	require.Empty(t, legacyKeys)
	require.Len(t, configs, 3)
	require.True(t, configs[0].FailFast)
	require.True(t, configs[1].NoWorkerLimit)
	require.True(t, configs[2].NoWorkerLimit)
}

// TestClientGetUserTasksInitializesEmptyCollection verifies empty bulk reads retain the public JSON collection contract.
func TestClientGetUserTasksInitializesEmptyCollection(t *testing.T) {
	t.Parallel()

	api := &facadeUserTaskAPI{}
	client := New(nil, nil, api, nil)

	got, err := client.GetUserTasks(context.Background(), nil, 0)
	require.NoError(t, err)
	require.Equal(t, int64(0), got.Total)
	require.NotNil(t, got.Items)
	require.Empty(t, got.Items)
	nativeKeys, legacyKeys, configs := api.snapshot()
	require.Empty(t, nativeKeys)
	require.Empty(t, legacyKeys)
	require.Empty(t, configs)
}

// TestClientGetUserTaskErrorsUseFacadeClassification verifies single and bulk failures cross the facade through ferrors conversion without partial results.
func TestClientGetUserTaskErrorsUseFacadeClassification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		domainErr error
		facadeErr error
	}{
		{name: "not found", domainErr: fmt.Errorf("read task: %w", d.ErrNotFound), facadeErr: ferr.ErrNotFound},
		{name: "malformed response", domainErr: fmt.Errorf("read task: %w", d.ErrMalformedResponse), facadeErr: ferr.ErrMalformedResponse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			api := &facadeUserTaskAPI{errors: map[string]error{"missing": tt.domainErr}}
			client := New(nil, nil, api, nil)

			one, err := client.GetUserTask(context.Background(), "missing")
			require.Empty(t, one)
			require.ErrorIs(t, err, tt.facadeErr)
			require.ErrorContains(t, err, tt.domainErr.Error())

			many, err := client.GetUserTasks(context.Background(), typex.Keys{"missing"}, 1)
			require.Equal(t, UserTasks{}, many)
			require.ErrorIs(t, err, tt.facadeErr)
			require.ErrorContains(t, err, tt.domainErr.Error())
		})
	}
}
