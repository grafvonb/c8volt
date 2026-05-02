// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import "github.com/grafvonb/c8volt/toolx"

type ProcessDefinition struct {
	BpmnProcessId     string                       `json:"bpmnProcessId,omitempty"`
	Key               string                       `json:"key,omitempty"`
	Name              string                       `json:"name,omitempty"`
	TenantId          string                       `json:"tenantId,omitempty"`
	ProcessVersion    int32                        `json:"processVersion,omitempty"`
	ProcessVersionTag string                       `json:"processVersionTag,omitempty"`
	Statistics        *ProcessDefinitionStatistics `json:"statistics,omitempty"`
}

type ProcessDefinitionStatistics struct {
	Active                 int64 `json:"active"`
	Canceled               int64 `json:"canceled"`
	Completed              int64 `json:"completed"`
	Incidents              int64 `json:"incidents"`
	IncidentCountSupported bool  `json:"incidentCountSupported,omitempty"`
}

type ProcessDefinitions struct {
	Total int32               `json:"total,omitempty"`
	Items []ProcessDefinition `json:"items,omitempty"`
}

type ProcessDefinitionFilter struct {
	Key               string `json:"key,omitempty"`
	BpmnProcessId     string `json:"bpmnProcessId,omitempty"`
	ProcessVersion    int32  `json:"processVersion,omitempty"`
	ProcessVersionTag string `json:"processVersionTag,omitempty"`
}

func (f ProcessDefinitionFilter) String() string {
	parts := make([]string, 0, 4)
	parts = toolx.AppendQuotedField(parts, "key", f.Key)
	parts = toolx.AppendQuotedField(parts, "bpmnProcessId", f.BpmnProcessId)
	parts = toolx.AppendInt32Field(parts, "processVersion", f.ProcessVersion)
	parts = toolx.AppendQuotedField(parts, "processVersionTag", f.ProcessVersionTag)
	return toolx.FormatActiveFields(parts)
}

type ProcessInstanceData struct {
	BpmnProcessId               string         `json:"bpmnProcessId,omitempty"`               // ProcessDefinitionId in API
	ProcessDefinitionSpecificId string         `json:"processDefinitionSpecificId,omitempty"` // ProcessDefinitionKey in API
	ProcessDefinitionVersion    int32          `json:"processDefinitionVersion,omitempty"`
	Variables                   map[string]any `json:"variables,omitempty"`
	TenantId                    string         `json:"tenantId,omitempty"`
}

type ProcessInstance struct {
	BpmnProcessId             string         `json:"bpmnProcessId,omitempty"`
	EndDate                   string         `json:"endDate,omitempty"`
	Incident                  bool           `json:"incident,omitempty"`
	Key                       string         `json:"key,omitempty"`
	ParentFlowNodeInstanceKey string         `json:"parentFlowNodeInstanceKey,omitempty"`
	ParentKey                 string         `json:"parentKey,omitempty"`
	ParentProcessInstanceKey  string         `json:"parentProcessInstanceKey,omitempty"`
	ProcessDefinitionKey      string         `json:"processDefinitionKey,omitempty"`
	ProcessVersion            int32          `json:"processVersion,omitempty"`
	ProcessVersionTag         string         `json:"processVersionTag,omitempty"`
	StartDate                 string         `json:"startDate,omitempty"`
	State                     State          `json:"state,omitempty"`
	TenantId                  string         `json:"tenantId,omitempty"`
	Variables                 map[string]any `json:"variables,omitempty"`
}

type ProcessInstanceIncidentDetail struct {
	IncidentKey            string `json:"incidentKey,omitempty"`
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

type IncidentEnrichedProcessInstance struct {
	Item      ProcessInstance                 `json:"item"`
	Incidents []ProcessInstanceIncidentDetail `json:"incidents"`
}

type IncidentEnrichedProcessInstances struct {
	Total int32                             `json:"total"`
	Items []IncidentEnrichedProcessInstance `json:"items"`
}

type ProcessInstances struct {
	Total int32             `json:"total,omitempty"`
	Items []ProcessInstance `json:"items,omitempty"`
}

type ProcessInstancePageRequest struct {
	From  int32  `json:"from,omitempty"`
	Size  int32  `json:"size,omitempty"`
	After string `json:"after,omitempty"`
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
	Count int64                            `json:"count,omitempty"`
	Kind  ProcessInstanceReportedTotalKind `json:"kind,omitempty"`
}

type ProcessInstancePage struct {
	Request       ProcessInstancePageRequest    `json:"request,omitempty"`
	OverflowState ProcessInstanceOverflowState  `json:"overflowState,omitempty"`
	ReportedTotal *ProcessInstanceReportedTotal `json:"reportedTotal,omitempty"`
	EndCursor     string                        `json:"endCursor,omitempty"`
	Items         []ProcessInstance             `json:"items,omitempty"`
}

type ProcessInstanceFilter struct {
	Key                  string `json:"key,omitempty"`
	BpmnProcessId        string `json:"bpmnProcessId,omitempty"`
	ProcessVersion       int32  `json:"version,omitempty"`
	ProcessVersionTag    string `json:"versionTag,omitempty"`
	ProcessDefinitionKey string `json:"processDefinitionKey,omitempty"`
	StartDateAfter       string `json:"startDateAfter,omitempty"`
	StartDateBefore      string `json:"startDateBefore,omitempty"`
	EndDateAfter         string `json:"endDateAfter,omitempty"`
	EndDateBefore        string `json:"endDateBefore,omitempty"`
	State                State  `json:"state,omitempty"`
	ParentKey            string `json:"parentKey,omitempty"`
	HasParent            *bool  `json:"hasParent,omitempty"`
	HasIncident          *bool  `json:"hasIncident,omitempty"`
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

type Reporter struct {
	Key        string `json:"key,omitempty"`
	Ok         bool   `json:"ok,omitempty"`
	StatusCode int    `json:"statusCode,omitempty"`
	Status     string `json:"status,omitempty"`
}

func (r Reporter) OK() bool {
	return r.Ok
}

type CancelReport = Reporter

type CancelReports struct {
	Items []CancelReport `json:"items,omitempty"`
}

func (c CancelReports) Totals() (total int, oks int, noks int) {
	return TotalsOf(c.Items)
}

type DeleteReport = Reporter

type DeleteReports struct {
	Items []DeleteReport `json:"items,omitempty"`
}

func (c DeleteReports) Totals() (total int, oks int, noks int) {
	return TotalsOf(c.Items)
}

type OKer interface {
	OK() bool
}

func TotalsOf[T OKer](items []T) (total, oks, noks int) {
	for _, r := range items {
		if r.OK() {
			oks++
		}
	}
	total = len(items)
	noks = total - oks
	return
}

type StateReport struct {
	Key    string `json:"key,omitempty"`
	State  State  `json:"state,omitempty"`
	Status string `json:"status,omitempty"`
}

type StateReports struct {
	Items []StateReport `json:"items,omitempty"`
}
