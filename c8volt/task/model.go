// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
)

// UserTask is the stable public representation of one native user task.
type UserTask struct {
	Key                      string   `json:"key"`
	State                    string   `json:"state"`
	Name                     string   `json:"name,omitempty"`
	ElementId                string   `json:"elementId,omitempty"`
	ElementInstanceKey       string   `json:"elementInstanceKey,omitempty"`
	Assignee                 string   `json:"assignee,omitempty"`
	CandidateUsers           []string `json:"candidateUsers,omitempty"`
	CandidateGroups          []string `json:"candidateGroups,omitempty"`
	ProcessInstanceKey       string   `json:"processInstanceKey"`
	ProcessDefinitionKey     string   `json:"processDefinitionKey,omitempty"`
	ProcessDefinitionId      string   `json:"processDefinitionId,omitempty"`
	ProcessDefinitionVersion int32    `json:"processDefinitionVersion,omitempty"`
	TenantId                 string   `json:"tenantId,omitempty"`
}

// UserTasks contains returned tasks and their returned count.
type UserTasks struct {
	Total int64      `json:"total"`
	Items []UserTask `json:"items"`
}

// UserTaskVariable reuses the public process-variable wire contract for one
// backend-selected effective user-task variable. Name is the backend-selected
// effective name and is unique within each successful task collection. Value
// is serialized as received; empty strings are valid and are not shortened for
// JSON. VariableKey is the identity returned by the backend.
// ProcessInstanceKey is the owning process, not a discovery filter. ScopeKey is
// the actual winning scope and may differ from the process-instance key.
// TenantId is backend tenant metadata and must not be rewritten from discovery
// settings. APITruncated is true when the received value is reported incomplete
// by the backend.
type UserTaskVariable = process.ProcessInstanceVariable

// VariableFilterOperator reuses the established process-instance variable
// filter operator contract for local user-task variable searches.
type VariableFilterOperator = process.ProcessInstanceVariableFilterOperator

const (
	// VariableFilterOperatorEq matches a local variable's serialized value exactly.
	VariableFilterOperatorEq = process.ProcessInstanceVariableFilterOperatorEq
	// VariableFilterOperatorNeq excludes a local variable's serialized value.
	VariableFilterOperatorNeq = process.ProcessInstanceVariableFilterOperatorNeq
	// VariableFilterOperatorExists checks whether a local variable is present.
	VariableFilterOperatorExists = process.ProcessInstanceVariableFilterOperatorExists
	// VariableFilterOperatorIn matches any serialized value in an array.
	VariableFilterOperatorIn = process.ProcessInstanceVariableFilterOperatorIn
	// VariableFilterOperatorNotIn excludes serialized values in an array.
	VariableFilterOperatorNotIn = process.ProcessInstanceVariableFilterOperatorNotIn
	// VariableFilterOperatorLike uses native wildcard matching.
	VariableFilterOperatorLike = process.ProcessInstanceVariableFilterOperatorLike
)

// VariableFilterClause reuses the established ordered predicate record for a
// local user-task variable condition.
type VariableFilterClause = process.ProcessInstanceVariableFilterClause

// VariableFilterSet keeps local user-task variable predicates in caller order.
type VariableFilterSet = process.ProcessInstanceVariableFilterSet

// VariableEnrichedUserTask pairs an unchanged selected task with its effective
// variables. Variables must be initialized even when the collection is empty.
type VariableEnrichedUserTask struct {
	Item      UserTask           `json:"item"`
	Variables []UserTaskVariable `json:"variables"`
}

// VariableEnrichedUserTasks preserves input task order in initialized Items.
// Total equals the number of returned items rather than a backend search total.
type VariableEnrichedUserTasks struct {
	Total int64                      `json:"total"`
	Items []VariableEnrichedUserTask `json:"items"`
}

// SearchRequest carries native task predicates and collection bounds.
type SearchRequest struct {
	ProcessInstanceKey   string            `json:"processInstanceKey,omitempty"`
	ProcessDefinitionKey string            `json:"processDefinitionKey,omitempty"`
	BpmnProcessId        string            `json:"bpmnProcessId,omitempty"`
	ElementId            string            `json:"elementId,omitempty"`
	State                string            `json:"state,omitempty"`
	Assignee             string            `json:"assignee,omitempty"`
	CandidateUser        string            `json:"candidateUser,omitempty"`
	CandidateGroup       string            `json:"candidateGroup,omitempty"`
	VariableFilters      VariableFilterSet `json:"variableFilters,omitempty"`
	BatchSize            int32             `json:"batchSize,omitempty"`
	Limit                int32             `json:"limit,omitempty"`
}

// PageRequest identifies one offset, cursor, or initial native search page.
type PageRequest struct {
	From  int32  `json:"from,omitempty"`
	Size  int32  `json:"size,omitempty"`
	After string `json:"after,omitempty"`
}

// ReportedTotalKind describes whether a backend total is exact or capped.
type ReportedTotalKind string

const (
	// ReportedTotalKindExact marks a complete backend count.
	ReportedTotalKindExact ReportedTotalKind = "exact"
	// ReportedTotalKindLowerBound marks a capped lower-bound count.
	ReportedTotalKindLowerBound ReportedTotalKind = "lower_bound"
)

// ReportedTotal carries backend count metadata exposed to page visitors.
type ReportedTotal struct {
	Count int64             `json:"count"`
	Kind  ReportedTotalKind `json:"kind"`
}

// ContinuationState describes the backend evidence for another page.
type ContinuationState string

const (
	// ContinuationStateHasMore records authoritative continuation evidence.
	ContinuationStateHasMore ContinuationState = "has_more"
	// ContinuationStateNoMore records authoritative exhaustion evidence.
	ContinuationStateNoMore ContinuationState = "no_more"
	// ContinuationStateIndeterminate records metadata requiring service interpretation.
	ContinuationStateIndeterminate ContinuationState = "indeterminate"
)

// SearchPage exposes selected tasks and normalized page metadata to visitors.
type SearchPage struct {
	Items             []UserTask        `json:"items"`
	Request           PageRequest       `json:"request"`
	RawItemCount      int32             `json:"rawItemCount"`
	EndCursor         string            `json:"endCursor,omitempty"`
	ReportedTotal     *ReportedTotal    `json:"reportedTotal,omitempty"`
	ContinuationState ContinuationState `json:"continuationState"`
}

// SearchPageAction tells service traversal whether a visitor wants to continue.
type SearchPageAction string

const (
	// SearchPageActionContinue requests another available page.
	SearchPageActionContinue SearchPageAction = "continue"
	// SearchPageActionStop ends traversal after the observed page.
	SearchPageActionStop SearchPageAction = "stop"
)

// SearchPageStep exposes selected and cumulative traversal progress.
type SearchPageStep struct {
	Page            SearchPage `json:"page"`
	CumulativeCount int64      `json:"cumulativeCount"`
	LimitReached    bool       `json:"limitReached"`
}

// SearchPageVisitor observes selected pages without owning page advancement.
type SearchPageVisitor func(SearchPageStep) (SearchPageAction, error)

// SearchCompletionDisposition identifies why successful traversal stopped.
type SearchCompletionDisposition string

const (
	// SearchCompletionExhausted means authoritative backend exhaustion was reached.
	SearchCompletionExhausted SearchCompletionDisposition = "exhausted"
	// SearchCompletionLimitReached means the requested result limit was reached.
	SearchCompletionLimitReached SearchCompletionDisposition = "limit_reached"
	// SearchCompletionVisitorStopped means the page visitor stopped traversal.
	SearchCompletionVisitorStopped SearchCompletionDisposition = "visitor_stopped"
)

// SearchPagesResult contains selected tasks and successful traversal metadata.
type SearchPagesResult struct {
	Items      []UserTask                  `json:"items"`
	Total      int64                       `json:"total"`
	Pages      int32                       `json:"pages"`
	Completion SearchCompletionDisposition `json:"completion"`
}

// UserTaskVariablePlannedValue describes an addition, unchanged request, or
// untouched effective variable in a frozen update plan.
type UserTaskVariablePlannedValue struct {
	Name         string `json:"name"`
	ScopeKey     string `json:"scopeKey"`
	Inherited    bool   `json:"inherited"`
	Value        any    `json:"value"`
	APITruncated bool   `json:"apiTruncated,omitempty"`
}

// UserTaskVariablePlannedChange preserves the observed and requested values
// at the scope selected during planning.
type UserTaskVariablePlannedChange struct {
	Name         string `json:"name"`
	ScopeKey     string `json:"scopeKey"`
	Inherited    bool   `json:"inherited"`
	Before       any    `json:"before"`
	After        any    `json:"after"`
	APITruncated bool   `json:"apiTruncated,omitempty"`
}

// ScopeVariableUpdateAssociation records which names on one selected task
// depend on a unique scope write.
type ScopeVariableUpdateAssociation struct {
	UserTaskKey string   `json:"userTaskKey"`
	Names       []string `json:"names"`
}

// ScopeVariableUpdateTarget is one frozen local write grouped by scope.
type ScopeVariableUpdateTarget struct {
	ScopeKey     string                           `json:"scopeKey"`
	TenantId     string                           `json:"tenantId,omitempty"`
	Variables    map[string]any                   `json:"variables"`
	Associations []ScopeVariableUpdateAssociation `json:"associations"`
}

// UserTaskVariablePlan classifies the effective-variable view for one task.
type UserTaskVariablePlan struct {
	UserTaskKey        string                          `json:"userTaskKey"`
	ElementInstanceKey string                          `json:"elementInstanceKey"`
	TenantId           string                          `json:"tenantId,omitempty"`
	Additions          []UserTaskVariablePlannedValue  `json:"additions"`
	Changes            []UserTaskVariablePlannedChange `json:"changes"`
	UnchangedRequested []UserTaskVariablePlannedValue  `json:"unchangedRequested"`
	Untouched          []UserTaskVariablePlannedValue  `json:"untouched"`
	TargetScopeKeys    []string                        `json:"targetScopeKeys"`
}

// UserTaskVariableUpdatePlan is a complete frozen plan suitable for preview
// and later execution without recomputing variable scopes.
type UserTaskVariableUpdatePlan struct {
	RequestedKeys          []string                    `json:"requestedKeys"`
	UserTasks              []UserTaskVariablePlan      `json:"userTasks"`
	Targets                []ScopeVariableUpdateTarget `json:"targets"`
	RequestedCount         int                         `json:"requestedCount"`
	UpdateCount            int                         `json:"updateCount"`
	VariableAddCount       int                         `json:"variableAddCount"`
	VariableChangeCount    int                         `json:"variableChangeCount"`
	VariableUnchangedCount int                         `json:"variableUnchangedCount"`
	VariableUntouchedCount int                         `json:"variableUntouchedCount"`
	MutationSubmitted      bool                        `json:"mutationSubmitted"`
	TenantContext          options.TenantContext       `json:"tenantContext"`
}

// ScopeVariableUpdateStatus identifies submission state for one frozen scope.
type ScopeVariableUpdateStatus string

const (
	ScopeVariableUpdateStatusSubmitted      ScopeVariableUpdateStatus = "submitted"
	ScopeVariableUpdateStatusMutationFailed ScopeVariableUpdateStatus = "mutation_failed"
	ScopeVariableUpdateStatusSkipped        ScopeVariableUpdateStatus = "skipped"
)

// ScopeVariableUpdateOutcome retains submission facts for one frozen target.
type ScopeVariableUpdateOutcome struct {
	ScopeKey   string                    `json:"scopeKey"`
	Names      []string                  `json:"names"`
	Status     ScopeVariableUpdateStatus `json:"status"`
	Accepted   bool                      `json:"accepted"`
	StatusCode int                       `json:"statusCode,omitempty"`
	Message    string                    `json:"message,omitempty"`
	Error      string                    `json:"error,omitempty"`
}

// UserTaskVariableUpdateStatus names task-level execution states.
type UserTaskVariableUpdateStatus string

const (
	UserTaskVariableUpdateStatusConfirmed          UserTaskVariableUpdateStatus = "confirmed"
	UserTaskVariableUpdateStatusSubmitted          UserTaskVariableUpdateStatus = "submitted"
	UserTaskVariableUpdateStatusMutationFailed     UserTaskVariableUpdateStatus = "mutation_failed"
	UserTaskVariableUpdateStatusConfirmationFailed UserTaskVariableUpdateStatus = "confirmation_failed"
	UserTaskVariableUpdateStatusUnchanged          UserTaskVariableUpdateStatus = "unchanged"
	UserTaskVariableUpdateStatusSkipped            UserTaskVariableUpdateStatus = "skipped"
)

// UserTaskVariableUpdateResult retains task-level status and dependent scope outcomes.
type UserTaskVariableUpdateResult struct {
	Key                string                       `json:"key"`
	Status             UserTaskVariableUpdateStatus `json:"status"`
	MutationAccepted   bool                         `json:"mutationAccepted"`
	ConfirmationStatus string                       `json:"confirmationStatus,omitempty"`
	Message            string                       `json:"message,omitempty"`
	Error              string                       `json:"error,omitempty"`
	Variables          map[string]any               `json:"variables"`
	Scopes             []ScopeVariableUpdateOutcome `json:"scopes,omitempty"`
}

// UserTaskVariableUpdateResults preserves unique requested task order.
type UserTaskVariableUpdateResults struct {
	Items []UserTaskVariableUpdateResult `json:"items,omitempty"`
}
