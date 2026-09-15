// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"encoding/json"
	"errors"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestFromDomainUserTaskMapsEveryFieldAndCopiesCandidates verifies the public boundary preserves identity and owns mutable slices.
func TestFromDomainUserTaskMapsEveryFieldAndCopiesCandidates(t *testing.T) {
	t.Parallel()

	source := d.UserTask{
		Key: "9007199254740993", State: "CREATED", Name: "Approve invoice", ElementId: "approve_invoice",
		ElementInstanceKey: "9007199254740995", Assignee: "alice", CandidateUsers: []string{"bob"},
		CandidateGroups: []string{"accounting"}, ProcessInstanceKey: "9007199254740997",
		ProcessDefinitionKey: "9007199254740999", ProcessDefinitionId: "invoice", ProcessDefinitionVersion: 7, TenantId: "foreign-tenant",
	}

	got := fromDomainUserTask(source)
	require.Equal(t, UserTask{
		Key: "9007199254740993", State: "CREATED", Name: "Approve invoice", ElementId: "approve_invoice",
		ElementInstanceKey: "9007199254740995", Assignee: "alice", CandidateUsers: []string{"bob"},
		CandidateGroups: []string{"accounting"}, ProcessInstanceKey: "9007199254740997",
		ProcessDefinitionKey: "9007199254740999", ProcessDefinitionId: "invoice", ProcessDefinitionVersion: 7, TenantId: "foreign-tenant",
	}, got)

	source.CandidateUsers[0] = "changed-user"
	source.CandidateGroups[0] = "changed-group"
	require.Equal(t, []string{"bob"}, got.CandidateUsers)
	require.Equal(t, []string{"accounting"}, got.CandidateGroups)

	raw, err := json.Marshal(got)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"key":"9007199254740993",
		"state":"CREATED",
		"name":"Approve invoice",
		"elementId":"approve_invoice",
		"elementInstanceKey":"9007199254740995",
		"assignee":"alice",
		"candidateUsers":["bob"],
		"candidateGroups":["accounting"],
		"processInstanceKey":"9007199254740997",
		"processDefinitionKey":"9007199254740999",
		"processDefinitionId":"invoice",
		"processDefinitionVersion":7,
		"tenantId":"foreign-tenant"
	}`, string(raw))
}

// TestUserTaskJSONPreservesRequiredFieldsAndOmitsEmptyOptionals verifies the stable task payload contract.
func TestUserTaskJSONPreservesRequiredFieldsAndOmitsEmptyOptionals(t *testing.T) {
	t.Parallel()

	raw, err := json.Marshal(UserTask{Key: "9007199254740993", State: "CREATED", ProcessInstanceKey: "9007199254740997"})
	require.NoError(t, err)
	require.JSONEq(t, `{"key":"9007199254740993","state":"CREATED","processInstanceKey":"9007199254740997"}`, string(raw))
}

// TestFromDomainUserTasksInitializesEmptyCollection verifies empty facade results have the required non-nil JSON shape.
func TestFromDomainUserTasksInitializesEmptyCollection(t *testing.T) {
	t.Parallel()

	got := fromDomainUserTasks(nil)
	require.NotNil(t, got.Items)
	require.IsType(t, int64(0), got.Total)
	raw, err := json.Marshal(got)
	require.NoError(t, err)
	require.JSONEq(t, `{"total":0,"items":[]}`, string(raw))
}

// TestToDomainSearchRequestMapsEverySelectorAndBound verifies facade conversion is mechanical and tenant-neutral.
func TestToDomainSearchRequestMapsEverySelectorAndBound(t *testing.T) {
	t.Parallel()

	request := SearchRequest{
		ProcessInstanceKey: "1", ProcessDefinitionKey: "2", BpmnProcessId: "invoice", ElementId: "approve_invoice",
		State: "CREATED", Assignee: "Alice", CandidateUser: "Bob", CandidateGroup: "Accounting", BatchSize: 1000, Limit: 25,
	}
	require.Equal(t, d.UserTaskSearchQuery{
		ProcessInstanceKey: "1", ProcessDefinitionKey: "2", BpmnProcessId: "invoice", ElementId: "approve_invoice",
		State: "CREATED", Assignee: "Alice", CandidateUser: "Bob", CandidateGroup: "Accounting", BatchSize: 1000, Limit: 25,
	}, toDomainSearchRequest(request))
}

// TestSearchPageVisitorConversionMapsFactsActionErrorAndCopies verifies visitor adaptation does not reinterpret traversal state.
func TestSearchPageVisitorConversionMapsFactsActionErrorAndCopies(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("visitor failed")
	source := d.UserTaskSearchPageStep{
		Page: d.UserTaskSearchPage{
			Items:   []d.UserTask{{Key: "1", State: "CREATED", ProcessInstanceKey: "2", CandidateUsers: []string{"alice"}}},
			Request: d.UserTaskPageRequest{After: "cursor-1", Size: 1000}, RawItemCount: 2, EndCursor: "cursor-2",
			ReportedTotal:     &d.UserTaskReportedTotal{Count: 10000, Kind: d.UserTaskReportedTotalKindLowerBound},
			ContinuationState: d.UserTaskContinuationStateHasMore,
		},
		CumulativeCount: 1, LimitReached: true,
	}

	visitor := toDomainSearchPageVisitor(func(step SearchPageStep) (SearchPageAction, error) {
		require.EqualValues(t, 1, step.CumulativeCount)
		require.True(t, step.LimitReached)
		require.Equal(t, ContinuationStateHasMore, step.Page.ContinuationState)
		require.Equal(t, ReportedTotalKindLowerBound, step.Page.ReportedTotal.Kind)
		require.Equal(t, []string{"alice"}, step.Page.Items[0].CandidateUsers)
		step.Page.Items[0].CandidateUsers[0] = "changed"
		return SearchPageActionStop, wantErr
	})

	action, err := visitor(source)
	require.Equal(t, d.UserTaskSearchPageActionStop, action)
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, []string{"alice"}, source.Page.Items[0].CandidateUsers)
	require.Nil(t, toDomainSearchPageVisitor(nil))
}

// TestFromDomainSearchPagesResultPreservesInt64CountAndOwnsItems verifies result conversion retains metadata without slice aliasing.
func TestFromDomainSearchPagesResultPreservesInt64CountAndOwnsItems(t *testing.T) {
	t.Parallel()

	source := d.UserTaskSearchPagesResult{
		Items: []d.UserTask{{Key: "1", CandidateGroups: []string{"ops"}}}, Total: int64(1) << 40,
		Pages: 3, Completion: d.UserTaskSearchCompletionLimitReached,
	}
	got := fromDomainSearchPagesResult(source)

	require.Equal(t, int64(1)<<40, got.Total)
	require.EqualValues(t, 3, got.Pages)
	require.Equal(t, SearchCompletionLimitReached, got.Completion)
	source.Items[0].CandidateGroups[0] = "changed"
	require.Equal(t, []string{"ops"}, got.Items[0].CandidateGroups)

	empty := fromDomainSearchPagesResult(d.UserTaskSearchPagesResult{})
	require.NotNil(t, empty.Items)
}
