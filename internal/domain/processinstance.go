// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"strings"
)

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

func (f ProcessInstanceFilter) String() string {
	parts := make([]string, 0, 13)
	parts = appendStringFilter(parts, "key", f.Key)
	parts = appendStringFilter(parts, "bpmnProcessId", f.BpmnProcessId)
	if f.ProcessVersion != 0 {
		parts = append(parts, fmt.Sprintf("processVersion=%d", f.ProcessVersion))
	}
	parts = appendStringFilter(parts, "processVersionTag", f.ProcessVersionTag)
	parts = appendStringFilter(parts, "processDefinitionKey", f.ProcessDefinitionKey)
	parts = appendStringFilter(parts, "startDateAfter", f.StartDateAfter)
	parts = appendStringFilter(parts, "startDateBefore", f.StartDateBefore)
	parts = appendStringFilter(parts, "endDateAfter", f.EndDateAfter)
	parts = appendStringFilter(parts, "endDateBefore", f.EndDateBefore)
	if f.State != "" {
		parts = append(parts, fmt.Sprintf("state=%s", f.State))
	}
	parts = appendStringFilter(parts, "parentKey", f.ParentKey)
	parts = appendBoolFilter(parts, "hasParent", f.HasParent)
	parts = appendBoolFilter(parts, "hasIncident", f.HasIncident)
	return formatFilterParts(parts)
}

func formatFilterParts(parts []string) string {
	if len(parts) == 0 {
		return "none"
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func appendStringFilter(parts []string, name, value string) []string {
	if value == "" {
		return parts
	}
	return append(parts, fmt.Sprintf("%s=%q", name, value))
}

func appendBoolFilter(parts []string, name string, value *bool) []string {
	if value == nil {
		return parts
	}
	return append(parts, fmt.Sprintf("%s=%t", name, *value))
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
