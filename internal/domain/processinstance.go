// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/grafvonb/c8volt/toolx"
)

type ProcessInstance struct {
	BpmnProcessId            string
	EndDate                  string
	Incident                 bool
	Key                      string
	ParentElementInstanceKey string
	ParentKey                string
	ProcessDefinitionKey     string
	RootProcessInstanceKey   string
	ProcessVersion           int32
	ProcessVersionTag        string
	StartDate                string
	State                    State
	TenantId                 string
	Variables                map[string]any
}

type ProcessInstanceVariable struct {
	Name               string
	Value              string
	VariableKey        string
	ProcessInstanceKey string
	ScopeKey           string
	TenantId           string
	APITruncated       bool
}

type ProcessInstanceVariableUpdateRequest struct {
	Key       string
	Variables map[string]any
}

type ProcessInstanceVariableUpdateResponse struct {
	Key        string
	Ok         bool
	StatusCode int
	Status     string
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
	VariableFilters      ProcessInstanceVariableFilterSet
}

func (f ProcessInstanceFilter) String() string {
	parts := make([]string, 0, 14)
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
	parts = toolx.AppendRawField(parts, "variableFilters", f.VariableFilters.String())
	return toolx.FormatActiveFields(parts)
}

// ProcessInstanceVariableFilterOperator names the native variable operators
// accepted by supported process-instance search endpoints.
type ProcessInstanceVariableFilterOperator string

const (
	// ProcessInstanceVariableFilterOperatorEq matches a variable's serialized value exactly.
	ProcessInstanceVariableFilterOperatorEq ProcessInstanceVariableFilterOperator = "$eq"
	// ProcessInstanceVariableFilterOperatorNeq excludes a variable's serialized value.
	ProcessInstanceVariableFilterOperatorNeq ProcessInstanceVariableFilterOperator = "$neq"
	// ProcessInstanceVariableFilterOperatorExists checks whether a variable is present.
	ProcessInstanceVariableFilterOperatorExists ProcessInstanceVariableFilterOperator = "$exists"
	// ProcessInstanceVariableFilterOperatorIn matches any serialized value in an array.
	ProcessInstanceVariableFilterOperatorIn ProcessInstanceVariableFilterOperator = "$in"
	// ProcessInstanceVariableFilterOperatorNotIn excludes serialized values in an array.
	ProcessInstanceVariableFilterOperatorNotIn ProcessInstanceVariableFilterOperator = "$notIn"
	// ProcessInstanceVariableFilterOperatorLike uses native wildcard matching.
	ProcessInstanceVariableFilterOperatorLike ProcessInstanceVariableFilterOperator = "$like"
)

// ProcessInstanceVariableFilterClause represents one native variable condition
// after command-layer shorthand and operator aliases have been normalized.
type ProcessInstanceVariableFilterClause struct {
	Name     string
	Operator ProcessInstanceVariableFilterOperator
	Value    string
	Exists   *bool
	Source   string
}

// Validate rejects clause shapes that cannot be translated into the native
// variable-search filter contract without guessing operator intent.
func (c ProcessInstanceVariableFilterClause) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("variable filter name must not be blank")
	}
	switch c.Operator {
	case ProcessInstanceVariableFilterOperatorExists:
		if c.Exists == nil {
			return fmt.Errorf("variable filter %q requires an exists value", c.Name)
		}
	case ProcessInstanceVariableFilterOperatorEq, ProcessInstanceVariableFilterOperatorNeq, ProcessInstanceVariableFilterOperatorLike:
		if c.Value == "" {
			return fmt.Errorf("variable filter %q requires a value for %s", c.Name, c.Operator)
		}
	case ProcessInstanceVariableFilterOperatorIn, ProcessInstanceVariableFilterOperatorNotIn:
		if !isArrayShapedVariableValue(c.Value) {
			return fmt.Errorf("variable filter %q requires an array value for %s", c.Name, c.Operator)
		}
	default:
		return fmt.Errorf("unsupported variable filter operator %q", c.Operator)
	}
	return nil
}

// String renders a clause for debug output without losing the normalized operator.
func (c ProcessInstanceVariableFilterClause) String() string {
	if c.Exists != nil {
		return fmt.Sprintf("%s.%s=%t", c.Name, c.Operator, *c.Exists)
	}
	return fmt.Sprintf("%s.%s=%s", c.Name, c.Operator, strconv.Quote(c.Value))
}

// ProcessInstanceVariableFilterSet keeps variable clauses ordered so command
// parsing, facade mapping, and service request construction remain testable.
type ProcessInstanceVariableFilterSet struct {
	Clauses []ProcessInstanceVariableFilterClause
}

// Validate applies clause-level validation while allowing an empty set to
// preserve existing non-variable process-instance search behavior.
func (s ProcessInstanceVariableFilterSet) Validate() error {
	for i, clause := range s.Clauses {
		if err := clause.Validate(); err != nil {
			return fmt.Errorf("variable filter clause %d: %w", i+1, err)
		}
	}
	return nil
}

// String renders all variable clauses in order for filter debug output.
func (s ProcessInstanceVariableFilterSet) String() string {
	if len(s.Clauses) == 0 {
		return ""
	}
	parts := make([]string, 0, len(s.Clauses))
	for _, clause := range s.Clauses {
		parts = append(parts, clause.String())
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// isArrayShapedVariableValue performs a local shape check without parsing or reserializing native values.
func isArrayShapedVariableValue(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) >= 2 && value[0] == '[' && value[len(value)-1] == ']'
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
	State                    State          `json:"state,omitempty"`
	TenantId                 string         `json:"tenantId,omitempty"`
	Variables                map[string]any `json:"variables,omitempty"`
	StartDate                string         `json:"startDate,omitempty"`
	StartConfirmedAt         string         `json:"startConfirmedAt,omitempty"`
}
