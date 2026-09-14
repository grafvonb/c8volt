// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
)

// UserTask represents one version-neutral native user task while retaining the
// identity fields used by the legacy process-instance resolver.
type UserTask struct {
	Key                  string
	State                string
	Name                 string
	ElementId            string
	ElementInstanceKey   string
	Assignee             string
	CandidateUsers       []string
	CandidateGroups      []string
	ProcessInstanceKey   string
	ProcessDefinitionKey string
	ProcessDefinitionId  string
	TenantId             string
}

// UserTaskSearchQuery carries backend predicates and collection bounds without
// embedding tenant scope, which remains a service call option.
type UserTaskSearchQuery struct {
	ProcessInstanceKey   string
	ProcessDefinitionKey string
	BpmnProcessId        string
	ElementId            string
	State                string
	Assignee             string
	CandidateUser        string
	CandidateGroup       string
	BatchSize            int32
	Limit                int32
}

// UserTaskPageRequest describes one offset, forward-cursor, or initial native
// user-task search request.
type UserTaskPageRequest struct {
	From  int32
	Size  int32
	After string
}

// Validate prevents an adapter from combining the mutually exclusive offset
// and forward-cursor request variants.
func (r UserTaskPageRequest) Validate() error {
	if r.From != 0 && r.After != "" {
		return errors.New("user-task page request cannot combine offset and cursor positions")
	}
	return nil
}

// UserTaskReportedTotalKind identifies whether backend total metadata is exact
// or only a capped lower bound.
type UserTaskReportedTotalKind string

const (
	// UserTaskReportedTotalKindExact means the backend reported the complete matching count.
	UserTaskReportedTotalKindExact UserTaskReportedTotalKind = "exact"
	// UserTaskReportedTotalKindLowerBound means the backend count is capped below or at the complete count.
	UserTaskReportedTotalKindLowerBound UserTaskReportedTotalKind = "lower_bound"
)

// Validate rejects total semantics that traversal cannot safely interpret.
func (k UserTaskReportedTotalKind) Validate() error {
	switch k {
	case UserTaskReportedTotalKindExact, UserTaskReportedTotalKindLowerBound:
		return nil
	default:
		return fmt.Errorf("invalid user-task reported total kind %q", k)
	}
}

// UserTaskReportedTotal carries a backend-provided count together with the
// semantics required to decide whether it proves exhaustion.
type UserTaskReportedTotal struct {
	Count int64
	Kind  UserTaskReportedTotalKind
}

// Validate rejects negative counts and unknown total semantics before shared
// traversal relies on backend metadata.
func (t UserTaskReportedTotal) Validate() error {
	if t.Count < 0 {
		return errors.New("user-task reported total count cannot be negative")
	}
	return t.Kind.Validate()
}

// UserTaskContinuationState captures the adapter's normalized evidence about
// whether another backend page exists.
type UserTaskContinuationState string

const (
	// UserTaskContinuationStateHasMore records authoritative continuation evidence.
	UserTaskContinuationStateHasMore UserTaskContinuationState = "has_more"
	// UserTaskContinuationStateNoMore records authoritative exhaustion evidence.
	UserTaskContinuationStateNoMore UserTaskContinuationState = "no_more"
	// UserTaskContinuationStateIndeterminate records metadata that needs traversal-level interpretation.
	UserTaskContinuationStateIndeterminate UserTaskContinuationState = "indeterminate"
)

// Validate rejects continuation values that service traversal cannot interpret.
func (s UserTaskContinuationState) Validate() error {
	switch s {
	case UserTaskContinuationStateHasMore, UserTaskContinuationStateNoMore, UserTaskContinuationStateIndeterminate:
		return nil
	default:
		return fmt.Errorf("invalid user-task continuation state %q", s)
	}
}

// UserTaskSearchPage contains one selected task page and the raw backend facts
// needed for service-owned paging and exact counting.
type UserTaskSearchPage struct {
	Items             []UserTask
	Request           UserTaskPageRequest
	RawItemCount      int32
	EndCursor         string
	ReportedTotal     *UserTaskReportedTotal
	ContinuationState UserTaskContinuationState
}

// UserTaskSearchPageAction tells service-owned traversal whether a caller wants
// to continue after observing the selected page.
type UserTaskSearchPageAction string

const (
	// UserTaskSearchPageActionContinue requests the next available page.
	UserTaskSearchPageActionContinue UserTaskSearchPageAction = "continue"
	// UserTaskSearchPageActionStop ends traversal after the observed page.
	UserTaskSearchPageActionStop UserTaskSearchPageAction = "stop"
)

// Validate rejects visitor actions outside the closed traversal protocol.
func (a UserTaskSearchPageAction) Validate() error {
	switch a {
	case UserTaskSearchPageActionContinue, UserTaskSearchPageActionStop:
		return nil
	default:
		return fmt.Errorf("invalid user-task search page action %q", a)
	}
}

// UserTaskSearchPageStep exposes selected and cumulative progress without
// transferring cursor, offset, or limit arithmetic to the caller.
type UserTaskSearchPageStep struct {
	Page            UserTaskSearchPage
	CumulativeCount int64
	LimitReached    bool
}

// UserTaskSearchPageVisitor observes selected pages and may stop traversal or
// return an interaction/rendering error to the service workflow.
type UserTaskSearchPageVisitor func(UserTaskSearchPageStep) (UserTaskSearchPageAction, error)

// UserTaskSearchCompletionDisposition records the successful reason a paged
// search stopped; failures remain errors rather than completion states.
type UserTaskSearchCompletionDisposition string

const (
	// UserTaskSearchCompletionExhausted means authoritative backend exhaustion was reached.
	UserTaskSearchCompletionExhausted UserTaskSearchCompletionDisposition = "exhausted"
	// UserTaskSearchCompletionLimitReached means the requested overall result limit was reached.
	UserTaskSearchCompletionLimitReached UserTaskSearchCompletionDisposition = "limit_reached"
	// UserTaskSearchCompletionVisitorStopped means the page visitor chose to stop early.
	UserTaskSearchCompletionVisitorStopped UserTaskSearchCompletionDisposition = "visitor_stopped"
)

// Validate rejects completion values that could misrepresent a failed or
// incomplete traversal as successful.
func (d UserTaskSearchCompletionDisposition) Validate() error {
	switch d {
	case UserTaskSearchCompletionExhausted, UserTaskSearchCompletionLimitReached, UserTaskSearchCompletionVisitorStopped:
		return nil
	default:
		return fmt.Errorf("invalid user-task search completion disposition %q", d)
	}
}

// UserTaskSearchPagesResult contains selected tasks, returned count, page
// count, and the successful traversal completion reason.
type UserTaskSearchPagesResult struct {
	Items      []UserTask
	Total      int64
	Pages      int32
	Completion UserTaskSearchCompletionDisposition
}
