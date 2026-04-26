// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type ProcessInstance struct {
	BpmnProcessId             string
	EndDate                   string
	Incident                  bool
	Key                       string
	ParentFlowNodeInstanceKey string
	ParentKey                 string
	ProcessDefinitionKey      string
	ProcessVersion            int32
	ProcessVersionTag         string
	StartDate                 string
	State                     State
	TenantId                  string
	Variables                 map[string]any
}

type ProcessInstanceFilter struct {
	Key                  string
	BpmnProcessId        string
	ProcessVersion       int32
	ProcessVersionTag    string
	ProcessDefinitionKey string
	StartDateAfter       string
	StartDateBefore      string
	EndDateAfter         string
	EndDateBefore        string
	State                State
	ParentKey            string
	HasParent            *bool
	HasIncident          *bool
}

type ProcessInstancePageRequest struct {
	From int32
	Size int32
}

type ProcessInstanceOverflowState string

const (
	ProcessInstanceOverflowStateNoMore        ProcessInstanceOverflowState = "no_more"
	ProcessInstanceOverflowStateHasMore       ProcessInstanceOverflowState = "has_more"
	ProcessInstanceOverflowStateIndeterminate ProcessInstanceOverflowState = "indeterminate"
)

type ProcessInstanceReportedTotalKind string

const (
	ProcessInstanceReportedTotalKindExact      ProcessInstanceReportedTotalKind = "exact"
	ProcessInstanceReportedTotalKindLowerBound ProcessInstanceReportedTotalKind = "lower_bound"
)

type ProcessInstanceReportedTotal struct {
	Count int64
	Kind  ProcessInstanceReportedTotalKind
}

type ProcessInstancePage struct {
	Items         []ProcessInstance
	Request       ProcessInstancePageRequest
	OverflowState ProcessInstanceOverflowState
	ReportedTotal *ProcessInstanceReportedTotal
}

type CancelResponse struct {
	Ok         bool
	StatusCode int
	Status     string
}

type DeleteResponse struct {
	Ok         bool
	StatusCode int
	Status     string
}

type StateResponse struct {
	Ok     bool
	State  State
	Status string
}

type StateResponses struct {
	Items []StateResponse
}

type ProcessInstanceData struct {
	BpmnProcessId               string // ProcessDefinitionId in API
	ProcessDefinitionSpecificId string // ProcessDefinitionKey in API
	ProcessDefinitionVersion    int32
	Variables                   map[string]any
	TenantId                    string
}

type ProcessInstanceCreation struct {
	Key                      string         `json:"key,omitempty"`
	BpmnProcessId            string         `json:"bpmnProcessId,omitempty"`        // ProcessDefinitionId in API
	ProcessDefinitionKey     string         `json:"processDefinitionKey,omitempty"` // ProcessDefinitionKey in API
	ProcessDefinitionVersion int32          `json:"processDefinitionVersion,omitempty"`
	TenantId                 string         `json:"tenantId,omitempty"`
	Variables                map[string]any `json:"variables,omitempty"`
	StartDate                string         `json:"startDate,omitempty"`
	StartConfirmedAt         string         `json:"startConfirmedAt,omitempty"`
}
