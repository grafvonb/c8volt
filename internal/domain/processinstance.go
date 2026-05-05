// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "github.com/grafvonb/c8volt/toolx"

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

type ProcessInstanceIncidentDetail struct {
	IncidentKey            string
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

func (f ProcessInstanceFilter) String() string {
	parts := make([]string, 0, 13)
	parts = toolx.AppendQuotedField(parts, "key", f.Key)
	parts = toolx.AppendQuotedField(parts, "bpmnProcessId", f.BpmnProcessId)
	parts = toolx.AppendInt32Field(parts, "processVersion", f.ProcessVersion)
	parts = toolx.AppendQuotedField(parts, "processVersionTag", f.ProcessVersionTag)
	parts = toolx.AppendQuotedField(parts, "processDefinitionKey", f.ProcessDefinitionKey)
	parts = toolx.AppendQuotedField(parts, "startDateAfter", f.StartDateAfter)
	parts = toolx.AppendQuotedField(parts, "startDateBefore", f.StartDateBefore)
	parts = toolx.AppendQuotedField(parts, "endDateAfter", f.EndDateAfter)
	parts = toolx.AppendQuotedField(parts, "endDateBefore", f.EndDateBefore)
	parts = toolx.AppendRawField(parts, "state", f.State.String())
	parts = toolx.AppendQuotedField(parts, "parentKey", f.ParentKey)
	parts = toolx.AppendBoolPtrField(parts, "hasParent", f.HasParent)
	parts = toolx.AppendBoolPtrField(parts, "hasIncident", f.HasIncident)
	return toolx.FormatActiveFields(parts)
}

type ProcessInstancePageRequest struct {
	From  int32
	Size  int32
	After string
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
	EndCursor     string
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

type ProcessInstanceExpectationRequest struct {
	States   States
	Incident *bool
}

func (r ProcessInstanceExpectationRequest) HasExpectations() bool {
	return len(r.States) > 0 || r.Incident != nil
}

type ProcessInstanceExpectationResponse struct {
	Key      string
	Ok       bool
	State    State
	Incident *bool
	Status   string
}

type ProcessInstanceExpectationResponses struct {
	Items []ProcessInstanceExpectationResponse
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
