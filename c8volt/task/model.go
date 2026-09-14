// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

// UserTask is the stable public representation of one native user task.
type UserTask struct {
	Key                  string   `json:"key"`
	State                string   `json:"state"`
	Name                 string   `json:"name,omitempty"`
	ElementId            string   `json:"elementId,omitempty"`
	ElementInstanceKey   string   `json:"elementInstanceKey,omitempty"`
	Assignee             string   `json:"assignee,omitempty"`
	CandidateUsers       []string `json:"candidateUsers,omitempty"`
	CandidateGroups      []string `json:"candidateGroups,omitempty"`
	ProcessInstanceKey   string   `json:"processInstanceKey"`
	ProcessDefinitionKey string   `json:"processDefinitionKey,omitempty"`
	ProcessDefinitionId  string   `json:"processDefinitionId,omitempty"`
	TenantId             string   `json:"tenantId,omitempty"`
}

// UserTasks contains returned tasks and their returned count.
type UserTasks struct {
	Total int64      `json:"total"`
	Items []UserTask `json:"items"`
}

// SearchRequest carries native task predicates and collection bounds.
type SearchRequest struct {
	ProcessInstanceKey   string `json:"processInstanceKey,omitempty"`
	ProcessDefinitionKey string `json:"processDefinitionKey,omitempty"`
	BpmnProcessId        string `json:"bpmnProcessId,omitempty"`
	ElementId            string `json:"elementId,omitempty"`
	State                string `json:"state,omitempty"`
	Assignee             string `json:"assignee,omitempty"`
	CandidateUser        string `json:"candidateUser,omitempty"`
	CandidateGroup       string `json:"candidateGroup,omitempty"`
	BatchSize            int32  `json:"batchSize,omitempty"`
	Limit                int32  `json:"limit,omitempty"`
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
