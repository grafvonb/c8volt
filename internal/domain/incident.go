// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type ProcessInstanceIncidentDetail struct {
	IncidentKey            string
	CreationTime           string
	ProcessInstanceKey     string
	TenantId               string
	State                  string
	ErrorType              string
	ErrorMessage           string
	FlowNodeId             string
	FlowNodeInstanceKey    string
	JobKey                 string
	RootProcessInstanceKey string
	ProcessDefinitionKey   string
	ProcessDefinitionId    string
}

type IncidentFilter struct {
	State                  string
	ErrorType              string
	ErrorMessage           string
	ProcessInstanceKey     string
	RootProcessInstanceKey string
	ProcessDefinitionKey   string
	ProcessDefinitionId    string
	FlowNodeId             string
	FlowNodeInstanceKey    string
	CreationTimeAfter      string
	CreationTimeBefore     string
}

type IncidentResolutionResponse struct {
	Key        string
	Ok         bool
	StatusCode int
	Status     string
}

type IncidentPageRequest struct {
	From  int32
	Size  int32
	After string
}

type IncidentReportedTotalKind string

const (
	IncidentReportedTotalKindExact      IncidentReportedTotalKind = "exact"
	IncidentReportedTotalKindLowerBound IncidentReportedTotalKind = "lower_bound"
)

type IncidentReportedTotal struct {
	Count int64
	Kind  IncidentReportedTotalKind
}

type IncidentPage struct {
	Items         []ProcessInstanceIncidentDetail
	Request       IncidentPageRequest
	OverflowState ProcessInstanceOverflowState
	ReportedTotal *IncidentReportedTotal
	EndCursor     string
}

type ResolutionOperation string

const (
	ResolutionOperationIncident        ResolutionOperation = "resolveIncident"
	ResolutionOperationProcessInstance ResolutionOperation = "resolveProcessInstance"
)

type IncidentResolutionStatus string

const (
	IncidentResolutionStatusPlanned            IncidentResolutionStatus = "planned"
	IncidentResolutionStatusSubmitted          IncidentResolutionStatus = "submitted"
	IncidentResolutionStatusConfirmed          IncidentResolutionStatus = "confirmed"
	IncidentResolutionStatusSkipped            IncidentResolutionStatus = "skipped"
	IncidentResolutionStatusMutationFailed     IncidentResolutionStatus = "mutation_failed"
	IncidentResolutionStatusConfirmationFailed IncidentResolutionStatus = "confirmation_failed"
)

type IncidentResolutionResult struct {
	IncidentKey        string
	ProcessInstanceKey string
	MutationAccepted   bool
	Status             IncidentResolutionStatus
	ConfirmationStatus string
	StatusCode         int
	Message            string
	Error              string
	DryRun             bool
	MutationSubmitted  bool
	WouldResolve       bool
	IncidentState      string
	Incident           *ProcessInstanceIncidentDetail
}

type IncidentResolutionResults struct {
	Operation         ResolutionOperation
	Items             []IncidentResolutionResult
	Total             int
	Submitted         int
	Confirmed         int
	Skipped           int
	Failed            int
	DryRun            bool
	MutationSubmitted bool
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
	ProcessInstanceKey    string
	AttemptedIncidentKeys []string
	ResolvedIncidentKeys  []string
	SkippedIncidentKeys   []string
	FailedIncidentKeys    []string
	ConfirmationStatus    string
	Status                ProcessInstanceResolutionStatus
	Error                 string
	DryRun                bool
	MutationSubmitted     bool
	Incidents             []ProcessInstanceIncidentDetail
}

type ProcessInstanceResolutionResults struct {
	Operation         ResolutionOperation
	Items             []ProcessInstanceResolutionResult
	Total             int
	Submitted         int
	Confirmed         int
	Skipped           int
	Failed            int
	DryRun            bool
	MutationSubmitted bool
}
