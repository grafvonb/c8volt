// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "github.com/grafvonb/c8volt/typex"

type Reporter struct {
	Key        string
	Ok         bool
	StatusCode int
	Status     string
}

type Reporters struct {
	Items []Reporter
}

type ProcessInstanceVariableUpdateStatus string

const (
	ProcessInstanceVariableUpdateStatusSubmitted          ProcessInstanceVariableUpdateStatus = "submitted"
	ProcessInstanceVariableUpdateStatusConfirmed          ProcessInstanceVariableUpdateStatus = "confirmed"
	ProcessInstanceVariableUpdateStatusMutationFailed     ProcessInstanceVariableUpdateStatus = "mutation_failed"
	ProcessInstanceVariableUpdateStatusConfirmationFailed ProcessInstanceVariableUpdateStatus = "confirmation_failed"
)

type ProcessInstanceVariableUpdateResult struct {
	Key                string
	Status             ProcessInstanceVariableUpdateStatus
	MutationAccepted   bool
	ConfirmationStatus string
	StatusCode         int
	Message            string
	Error              string
	Variables          map[string]any
}

type ProcessInstanceVariableUpdateResults struct {
	Items []ProcessInstanceVariableUpdateResult
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

type TraversalOutcome string

const (
	TraversalOutcomeComplete   TraversalOutcome = "complete"
	TraversalOutcomePartial    TraversalOutcome = "partial"
	TraversalOutcomeUnresolved TraversalOutcome = "unresolved"
)

type MissingAncestor struct {
	Key      string
	StartKey string
}

type DryRunPIKeyExpansion struct {
	Roots                      typex.Keys
	Collected                  typex.Keys
	SelectedFinalState         []ProcessInstance
	RequiresCancelBeforeDelete []ProcessInstance
	MissingAncestors           []MissingAncestor
	Warning                    string
	Outcome                    TraversalOutcome
}

func (r DryRunPIKeyExpansion) HasActionableResults() bool {
	return len(r.Roots) > 0 || len(r.Collected) > 0
}

type DeleteProcessDefinitionPlan struct {
	Items                 []DeleteProcessDefinitionPlanItem
	StateCheckSkipped     bool
	ProcessDefinitionKeys []string
	Warnings              []string
}

type DeleteProcessDefinitionPlanItem struct {
	Key                        string
	ActiveProcessInstanceCount int64
	ActiveProcessInstanceKeys  []string
	CancellationPlan           DryRunPIKeyExpansion
	Warnings                   []string
}

func (i DeleteProcessDefinitionPlanItem) ActiveProcessInstances() int64 {
	if i.ActiveProcessInstanceCount > 0 {
		return i.ActiveProcessInstanceCount
	}
	return int64(len(i.ActiveProcessInstanceKeys))
}
