// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"fmt"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/toolx"
)

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
	IncidentKey        string                         `json:"incidentKey"`
	ProcessInstanceKey string                         `json:"processInstanceKey,omitempty"`
	MutationAccepted   bool                           `json:"mutationAccepted"`
	Status             IncidentResolutionStatus       `json:"status"`
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

func (r IncidentResolutionResult) OK() bool {
	return r.Status != IncidentResolutionStatusMutationFailed && r.Status != IncidentResolutionStatusConfirmationFailed
}

type IncidentResolutionResults struct {
	Items             []IncidentResolutionResult `json:"items,omitempty"`
	Total             int                        `json:"total"`
	Submitted         int                        `json:"submitted"`
	Confirmed         int                        `json:"confirmed"`
	Skipped           int                        `json:"skipped"`
	Failed            int                        `json:"failed"`
	DryRun            bool                       `json:"dryRun"`
	MutationSubmitted bool                       `json:"mutationSubmitted"`
}

func (r IncidentResolutionResults) Totals() (total int, oks int, noks int) {
	return TotalsOf(r.Items)
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
	return TotalsOf(r.Items)
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

type ProcessInstanceVariable struct {
	Name               string `json:"name"`
	Value              string `json:"value"`
	VariableKey        string `json:"variableKey,omitempty"`
	ProcessInstanceKey string `json:"processInstanceKey"`
	ScopeKey           string `json:"scopeKey"`
	TenantId           string `json:"tenantId,omitempty"`
	APITruncated       bool   `json:"apiTruncated"`
}

type ProcessInstanceVariableUpdateStatus string

const (
	ProcessInstanceVariableUpdateStatusSubmitted          ProcessInstanceVariableUpdateStatus = "submitted"
	ProcessInstanceVariableUpdateStatusConfirmed          ProcessInstanceVariableUpdateStatus = "confirmed"
	ProcessInstanceVariableUpdateStatusMutationFailed     ProcessInstanceVariableUpdateStatus = "mutation_failed"
	ProcessInstanceVariableUpdateStatusConfirmationFailed ProcessInstanceVariableUpdateStatus = "confirmation_failed"
)

type ProcessInstanceVariableUpdateRequest struct {
	Key       string         `json:"key"`
	Variables map[string]any `json:"variables"`
}

type ProcessInstanceVariableUpdateResult struct {
	Key                string                              `json:"key"`
	Status             ProcessInstanceVariableUpdateStatus `json:"status"`
	MutationAccepted   bool                                `json:"mutationAccepted"`
	ConfirmationStatus string                              `json:"confirmationStatus,omitempty"`
	StatusCode         int                                 `json:"statusCode,omitempty"`
	Message            string                              `json:"message,omitempty"`
	Error              string                              `json:"error,omitempty"`
	Variables          map[string]any                      `json:"variables,omitempty"`
}

func (r ProcessInstanceVariableUpdateResult) OK() bool {
	return r.MutationAccepted && r.Status != ProcessInstanceVariableUpdateStatusMutationFailed && r.Status != ProcessInstanceVariableUpdateStatusConfirmationFailed
}

type ProcessInstanceVariableUpdateResults struct {
	Items []ProcessInstanceVariableUpdateResult `json:"items,omitempty"`
}

func (r ProcessInstanceVariableUpdateResults) Totals() (total int, oks int, noks int) {
	return TotalsOf(r.Items)
}

type IncidentEnrichedProcessInstance struct {
	Item      ProcessInstance                 `json:"item"`
	Incidents []ProcessInstanceIncidentDetail `json:"incidents"`
}

type IncidentEnrichedProcessInstances struct {
	Total int32                             `json:"total"`
	Items []IncidentEnrichedProcessInstance `json:"items"`
}

type VariableEnrichedProcessInstance struct {
	Item      ProcessInstance           `json:"item"`
	Variables []ProcessInstanceVariable `json:"variables"`
}

type VariableEnrichedProcessInstances struct {
	Total int32                             `json:"total"`
	Items []VariableEnrichedProcessInstance `json:"items"`
}

type IncidentEnrichedTraversalItem struct {
	Item      ProcessInstance                 `json:"item"`
	Incidents []ProcessInstanceIncidentDetail `json:"incidents"`
}

type IncidentEnrichedTraversalResult struct {
	Mode             TraversalMode                   `json:"mode"`
	Outcome          TraversalOutcome                `json:"outcome"`
	StartKey         string                          `json:"startKey,omitempty"`
	RootKey          string                          `json:"rootKey,omitempty"`
	Keys             []string                        `json:"keys,omitempty"`
	Edges            map[string][]string             `json:"edges,omitempty"`
	Items            []IncidentEnrichedTraversalItem `json:"items,omitempty"`
	MissingAncestors []MissingAncestor               `json:"missingAncestors,omitempty"`
	Warning          string                          `json:"warning,omitempty"`
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

type IncidentExpectation bool

const (
	IncidentExpectationFalse IncidentExpectation = false
	IncidentExpectationTrue  IncidentExpectation = true
)

func (e IncidentExpectation) Bool() bool {
	return bool(e)
}

// ParseIncidentExpectation accepts only the CLI contract values so invalid input fails before waiting starts.
func ParseIncidentExpectation(in string) (IncidentExpectation, error) {
	switch in {
	case "true":
		return IncidentExpectationTrue, nil
	case "false":
		return IncidentExpectationFalse, nil
	default:
		return false, fmt.Errorf("%w: %s", ferr.ErrInvalidInput, in)
	}
}

// ValidIncidentExpectationStrings keeps help, validation, and tests aligned on the public boolean surface.
func ValidIncidentExpectationStrings() []string {
	return []string{"true", "false"}
}

type ProcessInstanceExpectationRequest struct {
	States   States               `json:"states,omitempty"`
	Incident *IncidentExpectation `json:"incident,omitempty"`
}

func (r ProcessInstanceExpectationRequest) HasExpectations() bool {
	return len(r.States) > 0 || r.Incident != nil
}

type ProcessInstanceExpectationReport struct {
	Key      string               `json:"key,omitempty"`
	Ok       bool                 `json:"ok,omitempty"`
	State    State                `json:"state,omitempty"`
	Incident *IncidentExpectation `json:"incident,omitempty"`
	Status   string               `json:"status,omitempty"`
}

func (r ProcessInstanceExpectationReport) OK() bool {
	return r.Ok
}

type ProcessInstanceExpectationReports struct {
	Items []ProcessInstanceExpectationReport `json:"items,omitempty"`
}

func (r ProcessInstanceExpectationReports) Totals() (total int, oks int, noks int) {
	return TotalsOf(r.Items)
}
