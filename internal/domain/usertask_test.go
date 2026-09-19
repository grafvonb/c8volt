// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUserTaskModelPreservesStringKeys verifies native task identity and association keys never pass through numeric JSON values.
func TestUserTaskModelPreservesStringKeys(t *testing.T) {
	t.Parallel()

	task := UserTask{
		Key:                      "9007199254740993",
		State:                    "CREATED",
		Name:                     "Approve invoice",
		ElementId:                "approve_invoice",
		ElementInstanceKey:       "9007199254740995",
		Assignee:                 "alice",
		CandidateUsers:           []string{"bob"},
		CandidateGroups:          []string{"accounting"},
		ProcessInstanceKey:       "9007199254740997",
		ProcessDefinitionKey:     "9007199254740999",
		ProcessDefinitionId:      "invoice",
		ProcessDefinitionVersion: 7,
		TenantId:                 "tenant-a",
	}

	raw, err := json.Marshal(task)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"Key":"9007199254740993",
		"State":"CREATED",
		"Name":"Approve invoice",
		"ElementId":"approve_invoice",
		"ElementInstanceKey":"9007199254740995",
		"Assignee":"alice",
		"CandidateUsers":["bob"],
		"CandidateGroups":["accounting"],
		"ProcessInstanceKey":"9007199254740997",
		"ProcessDefinitionKey":"9007199254740999",
		"ProcessDefinitionId":"invoice",
		"ProcessDefinitionVersion":7,
		"TenantId":"tenant-a"
	}`, string(raw))
}

// TestUserTaskSearchQueryCarriesEveryVersionNeutralSelector verifies the shared query does not omit a planned backend predicate or bound.
func TestUserTaskSearchQueryCarriesEveryVersionNeutralSelector(t *testing.T) {
	t.Parallel()

	query := UserTaskSearchQuery{
		ProcessInstanceKey:   "2251799813685249",
		ProcessDefinitionKey: "2251799813685250",
		BpmnProcessId:        "invoice",
		ElementId:            "approve_invoice",
		State:                "CREATED",
		Assignee:             "alice",
		CandidateUser:        "bob",
		CandidateGroup:       "accounting",
		BatchSize:            1000,
		Limit:                25,
	}

	require.Equal(t, "2251799813685249", query.ProcessInstanceKey)
	require.Equal(t, "2251799813685250", query.ProcessDefinitionKey)
	require.Equal(t, "invoice", query.BpmnProcessId)
	require.Equal(t, "approve_invoice", query.ElementId)
	require.Equal(t, "CREATED", query.State)
	require.Equal(t, "alice", query.Assignee)
	require.Equal(t, "bob", query.CandidateUser)
	require.Equal(t, "accounting", query.CandidateGroup)
	require.EqualValues(t, 1000, query.BatchSize)
	require.EqualValues(t, 25, query.Limit)
}

// TestUserTaskPageRequestValidateRejectsMixedPositions verifies offset and forward-cursor positions cannot be sent together.
func TestUserTaskPageRequestValidateRejectsMixedPositions(t *testing.T) {
	t.Parallel()

	require.NoError(t, (UserTaskPageRequest{Size: 1000}).Validate())
	require.NoError(t, (UserTaskPageRequest{From: 1000, Size: 1000}).Validate())
	require.NoError(t, (UserTaskPageRequest{After: "cursor-1", Size: 1000}).Validate())
	require.EqualError(t, (UserTaskPageRequest{From: 1000, After: "cursor-1", Size: 1000}).Validate(), "user-task page request cannot combine offset and cursor positions")
}

// TestUserTaskReportedTotalValidationClosesKindSet verifies backend totals distinguish exact values from capped lower bounds.
func TestUserTaskReportedTotalValidationClosesKindSet(t *testing.T) {
	t.Parallel()

	require.NoError(t, (UserTaskReportedTotal{Count: 10, Kind: UserTaskReportedTotalKindExact}).Validate())
	require.NoError(t, (UserTaskReportedTotal{Count: 10, Kind: UserTaskReportedTotalKindLowerBound}).Validate())
	require.EqualError(t, (UserTaskReportedTotal{Count: 10, Kind: UserTaskReportedTotalKind("approximate")}).Validate(), `invalid user-task reported total kind "approximate"`)
	require.EqualError(t, (UserTaskReportedTotal{Count: -1, Kind: UserTaskReportedTotalKindExact}).Validate(), "user-task reported total count cannot be negative")
}

// TestUserTaskContinuationStateValidationClosesStateSet verifies page continuation uses only the three planned service states.
func TestUserTaskContinuationStateValidationClosesStateSet(t *testing.T) {
	t.Parallel()

	for _, state := range []UserTaskContinuationState{
		UserTaskContinuationStateHasMore,
		UserTaskContinuationStateNoMore,
		UserTaskContinuationStateIndeterminate,
	} {
		require.NoError(t, state.Validate())
	}
	require.EqualError(t, UserTaskContinuationState("unknown").Validate(), `invalid user-task continuation state "unknown"`)
}

// TestUserTaskSearchPageActionValidationClosesActionSet verifies visitors can only continue or stop traversal.
func TestUserTaskSearchPageActionValidationClosesActionSet(t *testing.T) {
	t.Parallel()

	require.NoError(t, UserTaskSearchPageActionContinue.Validate())
	require.NoError(t, UserTaskSearchPageActionStop.Validate())
	require.EqualError(t, UserTaskSearchPageAction("skip").Validate(), `invalid user-task search page action "skip"`)
}

// TestUserTaskSearchCompletionValidationClosesDispositionSet verifies completed traversal records why collection stopped.
func TestUserTaskSearchCompletionValidationClosesDispositionSet(t *testing.T) {
	t.Parallel()

	for _, completion := range []UserTaskSearchCompletionDisposition{
		UserTaskSearchCompletionExhausted,
		UserTaskSearchCompletionLimitReached,
		UserTaskSearchCompletionVisitorStopped,
	} {
		require.NoError(t, completion.Validate())
	}
	require.EqualError(t, UserTaskSearchCompletionDisposition("failed").Validate(), `invalid user-task search completion disposition "failed"`)
}

// TestUserTaskTraversalModelsCarryServiceFacts verifies page, visitor, and result models retain raw and selected progress separately.
func TestUserTaskTraversalModelsCarryServiceFacts(t *testing.T) {
	t.Parallel()

	page := UserTaskSearchPage{
		Items:             []UserTask{{Key: "2251799813685249"}},
		Request:           UserTaskPageRequest{After: "cursor-1", Size: 1000},
		RawItemCount:      2,
		EndCursor:         "cursor-2",
		ReportedTotal:     &UserTaskReportedTotal{Count: 10000, Kind: UserTaskReportedTotalKindLowerBound},
		ContinuationState: UserTaskContinuationStateHasMore,
	}
	step := UserTaskSearchPageStep{Page: page, CumulativeCount: 1, LimitReached: true}
	result := UserTaskSearchPagesResult{
		Items:      append([]UserTask(nil), page.Items...),
		Total:      1,
		Pages:      2,
		Completion: UserTaskSearchCompletionLimitReached,
	}

	require.EqualValues(t, 2, step.Page.RawItemCount)
	require.EqualValues(t, 1, step.CumulativeCount)
	require.True(t, step.LimitReached)
	require.EqualValues(t, 1, result.Total)
	require.EqualValues(t, 2, result.Pages)
	require.Equal(t, UserTaskSearchCompletionLimitReached, result.Completion)
}

// TestUserTaskVariablePageRequestValidationRejectsInvalidBounds verifies effective-variable paging stays offset-only with usable bounds.
func TestUserTaskVariablePageRequestValidationRejectsInvalidBounds(t *testing.T) {
	t.Parallel()

	require.NoError(t, (UserTaskVariablePageRequest{From: 0, Size: 1}).Validate())
	require.NoError(t, (UserTaskVariablePageRequest{From: 1000, Size: 1000}).Validate())
	require.EqualError(t, (UserTaskVariablePageRequest{From: -1, Size: 1000}).Validate(), "user-task variable page request offset cannot be negative")
	require.EqualError(t, (UserTaskVariablePageRequest{From: 0, Size: 0}).Validate(), "user-task variable page request size must be positive")
	require.EqualError(t, (UserTaskVariablePageRequest{From: 0, Size: -1}).Validate(), "user-task variable page request size must be positive")
}

// TestUserTaskVariableModelsCarryEnrichmentAndPagingFacts verifies the domain records preserve task order, initialized empty collections, and raw continuation metadata.
func TestUserTaskVariableModelsCarryEnrichmentAndPagingFacts(t *testing.T) {
	t.Parallel()

	variable := ProcessInstanceVariable{
		Name:               "approvalReason",
		Value:              `"accepted"`,
		VariableKey:        "2251799813685250",
		ProcessInstanceKey: "2251799813685249",
		ScopeKey:           "2251799813685251",
		TenantId:           "tenant-a",
		APITruncated:       true,
	}
	page := UserTaskVariablePage{
		Items:                   []ProcessInstanceVariable{variable},
		Request:                 UserTaskVariablePageRequest{From: 1000, Size: 1000},
		RawItemCount:            2,
		ReportedTotal:           UserTaskReportedTotal{Count: 10000, Kind: UserTaskReportedTotalKindLowerBound},
		HasContinuationEvidence: true,
	}
	enriched := VariableEnrichedUserTasks{
		Total: 2,
		Items: []VariableEnrichedUserTask{
			{Item: UserTask{Key: "task-1"}, Variables: []ProcessInstanceVariable{variable}},
			{Item: UserTask{Key: "task-2"}, Variables: make([]ProcessInstanceVariable, 0)},
		},
	}

	require.EqualValues(t, 2, page.RawItemCount)
	require.True(t, page.HasContinuationEvidence)
	require.Equal(t, UserTaskReportedTotalKindLowerBound, page.ReportedTotal.Kind)
	require.EqualValues(t, 2, enriched.Total)
	require.Equal(t, "task-1", enriched.Items[0].Item.Key)
	require.Equal(t, "approvalReason", enriched.Items[0].Variables[0].Name)
	require.NotNil(t, enriched.Items[1].Variables)
	require.Empty(t, enriched.Items[1].Variables)
}
