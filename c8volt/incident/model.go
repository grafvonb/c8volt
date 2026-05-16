// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incident

type ProcessInstanceIncidentDetail struct {
	IncidentKey            string `json:"incidentKey,omitempty"`
	CreationTime           string `json:"creationTime,omitempty"`
	ProcessInstanceKey     string `json:"processInstanceKey"`
	TenantId               string `json:"tenantId,omitempty"`
	State                  string `json:"state,omitempty"`
	ErrorType              string `json:"errorType,omitempty"`
	ErrorMessage           string `json:"errorMessage"`
	FlowNodeId             string `json:"flowNodeId,omitempty"`
	FlowNodeInstanceKey    string `json:"flowNodeInstanceKey,omitempty"`
	JobKey                 string `json:"jobKey,omitempty"`
	RootProcessInstanceKey string `json:"rootProcessInstanceKey,omitempty"`
	ProcessDefinitionKey   string `json:"processDefinitionKey,omitempty"`
	ProcessDefinitionId    string `json:"processDefinitionId,omitempty"`
}

type Filter struct {
	Keys                   []string `json:"keys,omitempty"`
	State                  string   `json:"state,omitempty"`
	ErrorType              string   `json:"errorType,omitempty"`
	ErrorMessage           string   `json:"errorMessage,omitempty"`
	ProcessInstanceKey     string   `json:"processInstanceKey,omitempty"`
	RootProcessInstanceKey string   `json:"rootProcessInstanceKey,omitempty"`
	ProcessDefinitionKey   string   `json:"processDefinitionKey,omitempty"`
	ProcessDefinitionId    string   `json:"processDefinitionId,omitempty"`
	FlowNodeId             string   `json:"flowNodeId,omitempty"`
	FlowNodeInstanceKey    string   `json:"flowNodeInstanceKey,omitempty"`
	CreationTimeAfter      string   `json:"creationTimeAfter,omitempty"`
	CreationTimeBefore     string   `json:"creationTimeBefore,omitempty"`
}

type Incidents struct {
	Total int32                           `json:"total,omitempty"`
	Items []ProcessInstanceIncidentDetail `json:"items,omitempty"`
}

type PageRequest struct {
	From  int32  `json:"from,omitempty"`
	Size  int32  `json:"size,omitempty"`
	After string `json:"after,omitempty"`
}

type ReportedTotalKind string

const (
	ReportedTotalKindExact      ReportedTotalKind = "exact"
	ReportedTotalKindLowerBound ReportedTotalKind = "lower_bound"
)

type ReportedTotal struct {
	Count int64             `json:"count,omitempty"`
	Kind  ReportedTotalKind `json:"kind,omitempty"`
}

type OverflowState string

const (
	OverflowStateNoMore        OverflowState = "no_more"
	OverflowStateHasMore       OverflowState = "has_more"
	OverflowStateIndeterminate OverflowState = "indeterminate"
)

type Page struct {
	Request       PageRequest                     `json:"request,omitempty"`
	OverflowState OverflowState                   `json:"overflowState,omitempty"`
	ReportedTotal *ReportedTotal                  `json:"reportedTotal,omitempty"`
	EndCursor     string                          `json:"endCursor,omitempty"`
	Items         []ProcessInstanceIncidentDetail `json:"items,omitempty"`
}

type ResolutionOperation string

const (
	ResolutionOperationIncident        ResolutionOperation = "resolveIncident"
	ResolutionOperationProcessInstance ResolutionOperation = "resolveProcessInstance"
)

type ResolutionStatus string

const (
	ResolutionStatusPlanned            ResolutionStatus = "planned"
	ResolutionStatusSubmitted          ResolutionStatus = "submitted"
	ResolutionStatusConfirmed          ResolutionStatus = "confirmed"
	ResolutionStatusSkipped            ResolutionStatus = "skipped"
	ResolutionStatusMutationFailed     ResolutionStatus = "mutation_failed"
	ResolutionStatusConfirmationFailed ResolutionStatus = "confirmation_failed"
)

type ResolutionResult struct {
	IncidentKey        string                         `json:"incidentKey"`
	ProcessInstanceKey string                         `json:"processInstanceKey,omitempty"`
	MutationAccepted   bool                           `json:"mutationAccepted"`
	Status             ResolutionStatus               `json:"status"`
	ConfirmationStatus string                         `json:"confirmationStatus,omitempty"`
	StatusCode         int                            `json:"statusCode,omitempty"`
	Message            string                         `json:"message,omitempty"`
	Error              string                         `json:"error,omitempty"`
	DryRun             bool                           `json:"dryRun"`
	MutationSubmitted  bool                           `json:"mutationSubmitted"`
	WouldResolve       bool                           `json:"wouldResolve,omitempty"`
	IncidentState      string                         `json:"incidentState,omitempty"`
	Incident           *ProcessInstanceIncidentDetail `json:"incident,omitempty"`
}

func (r ResolutionResult) OK() bool {
	return r.Status != ResolutionStatusMutationFailed && r.Status != ResolutionStatusConfirmationFailed
}

type ResolutionResults struct {
	Operation         ResolutionOperation `json:"operation,omitempty"`
	Items             []ResolutionResult  `json:"items,omitempty"`
	Total             int                 `json:"total"`
	Submitted         int                 `json:"submitted"`
	Confirmed         int                 `json:"confirmed"`
	Skipped           int                 `json:"skipped"`
	Failed            int                 `json:"failed"`
	DryRun            bool                `json:"dryRun"`
	MutationSubmitted bool                `json:"mutationSubmitted"`
}

func (r ResolutionResults) Totals() (total int, oks int, noks int) {
	for _, item := range r.Items {
		if item.OK() {
			oks++
		}
	}
	total = len(r.Items)
	noks = total - oks
	return total, oks, noks
}

type ProcessInstanceResolutionStatus string

const (
	ProcessInstanceResolutionStatusPlanned       ProcessInstanceResolutionStatus = "planned"
	ProcessInstanceResolutionStatusSubmitted     ProcessInstanceResolutionStatus = "submitted"
	ProcessInstanceResolutionStatusConfirmed     ProcessInstanceResolutionStatus = "confirmed"
	ProcessInstanceResolutionStatusSkipped       ProcessInstanceResolutionStatus = "skipped"
	ProcessInstanceResolutionStatusPartialFailed ProcessInstanceResolutionStatus = "partial_failed"
	ProcessInstanceResolutionStatusFailed        ProcessInstanceResolutionStatus = "failed"
)

type ProcessInstanceResolutionResult struct {
	ProcessInstanceKey    string                          `json:"processInstanceKey"`
	AttemptedIncidentKeys []string                        `json:"attemptedIncidentKeys,omitempty"`
	ResolvedIncidentKeys  []string                        `json:"resolvedIncidentKeys,omitempty"`
	SkippedIncidentKeys   []string                        `json:"skippedIncidentKeys,omitempty"`
	FailedIncidentKeys    []string                        `json:"failedIncidentKeys,omitempty"`
	ConfirmationStatus    string                          `json:"confirmationStatus,omitempty"`
	Status                ProcessInstanceResolutionStatus `json:"status"`
	Error                 string                          `json:"error,omitempty"`
	DryRun                bool                            `json:"dryRun"`
	MutationSubmitted     bool                            `json:"mutationSubmitted"`
	Incidents             []ProcessInstanceIncidentDetail `json:"incidents,omitempty"`
}

func (r ProcessInstanceResolutionResult) OK() bool {
	return r.Status != ProcessInstanceResolutionStatusFailed && r.Status != ProcessInstanceResolutionStatusPartialFailed
}

type ProcessInstanceResolutionResults struct {
	Operation         ResolutionOperation               `json:"operation,omitempty"`
	Items             []ProcessInstanceResolutionResult `json:"items,omitempty"`
	Total             int                               `json:"total"`
	Submitted         int                               `json:"submitted"`
	Confirmed         int                               `json:"confirmed"`
	Skipped           int                               `json:"skipped"`
	Failed            int                               `json:"failed"`
	DryRun            bool                              `json:"dryRun"`
	MutationSubmitted bool                              `json:"mutationSubmitted"`
}

func (r ProcessInstanceResolutionResults) Totals() (total int, oks int, noks int) {
	for _, item := range r.Items {
		if item.OK() {
			oks++
		}
	}
	total = len(r.Items)
	noks = total - oks
	return total, oks, noks
}

type ResolutionPlan struct {
	Operation                ResolutionOperation `json:"operation"`
	RequestedKeys            []string            `json:"requestedKeys,omitempty"`
	DryRun                   bool                `json:"dryRun"`
	MutationSubmitted        bool                `json:"mutationSubmitted"`
	Items                    any                 `json:"items,omitempty"`
	WouldResolveIncidentKeys []string            `json:"wouldResolveIncidentKeys,omitempty"`
	SkippedIncidentKeys      []string            `json:"skippedIncidentKeys,omitempty"`
	Errors                   []string            `json:"errors,omitempty"`
}
