package domain

import (
	"strings"
)

type Incident struct {
	Key                  string            `json:"key,omitempty"`
	CreationTime         string            `json:"creationTime,omitempty"`
	ElementId            string            `json:"elementId,omitempty"`
	ElementInstanceKey   string            `json:"elementInstanceKey,omitempty"`
	ErrorMessage         string            `json:"errorMessage,omitempty"`
	ErrorType            IncidentErrorType `json:"errorType,omitempty"`
	ProcessDefinitionId  string            `json:"processDefinitionId,omitempty"`
	ProcessDefinitionKey string            `json:"processDefinitionKey,omitempty"`
	ProcessInstanceKey   string            `json:"processInstanceKey,omitempty"`
	State                State             `json:"state,omitempty"`
	TenantId             string            `json:"tenantId,omitempty"`
}

type IncidentFilter struct {
	CreationTime         string            `json:"creationTime,omitempty"`
	ElementId            string            `json:"elementId,omitempty"`
	ElementInstanceKey   string            `json:"elementInstanceKey,omitempty"`
	ErrorMessage         string            `json:"errorMessage,omitempty"`
	ErrorType            IncidentErrorType `json:"errorType,omitempty"`
	IncidentKey          string            `json:"incidentKey,omitempty"`
	ProcessDefinitionId  string            `json:"processDefinitionId,omitempty"`
	ProcessDefinitionKey string            `json:"processDefinitionKey,omitempty"`
	ProcessInstanceKey   string            `json:"processInstanceKey,omitempty"`
	State                State             `json:"state,omitempty"`
	TenantId             string            `json:"tenantId,omitempty"`
}

type IncidentResponse struct {
	Ok         bool
	StatusCode int
	Status     string
}

type IncidentErrorType string

const (
	IncidentErrorTypeAll                        IncidentErrorType = "ALL"
	IncidentErrorTypeUnspecified                IncidentErrorType = "UNSPECIFIED"
	IncidentErrorTypeUnknown                    IncidentErrorType = "UNKNOWN"
	IncidentErrorTypeIoMappingError             IncidentErrorType = "IO_MAPPING_ERROR"
	IncidentErrorTypeJobNoRetries               IncidentErrorType = "JOB_NO_RETRIES"
	IncidentErrorTypeExecutionListenerNoRetries IncidentErrorType = "EXECUTION_LISTENER_NO_RETRIES"
	IncidentErrorTypeTaskListenerNoRetries      IncidentErrorType = "TASK_LISTENER_NO_RETRIES"
	IncidentErrorTypeAdHocSubProcessNoRetries   IncidentErrorType = "AD_HOC_SUB_PROCESS_NO_RETRIES"
	IncidentErrorTypeConditionError             IncidentErrorType = "CONDITION_ERROR"
	IncidentErrorTypeExtractValueError          IncidentErrorType = "EXTRACT_VALUE_ERROR"
	IncidentErrorTypeCalledElementError         IncidentErrorType = "CALLED_ELEMENT_ERROR"
	IncidentErrorTypeUnhandledErrorEvent        IncidentErrorType = "UNHANDLED_ERROR_EVENT"
	IncidentErrorTypeMessageSizeExceeded        IncidentErrorType = "MESSAGE_SIZE_EXCEEDED"
	IncidentErrorTypeCalledDecisionError        IncidentErrorType = "CALLED_DECISION_ERROR"
	IncidentErrorTypeDecisionEvaluationError    IncidentErrorType = "DECISION_EVALUATION_ERROR"
	IncidentErrorTypeFormNotFound               IncidentErrorType = "FORM_NOT_FOUND"
	IncidentErrorTypeResourceNotFound           IncidentErrorType = "RESOURCE_NOT_FOUND"
)

func (s IncidentErrorType) String() string { return string(s) }

func (s IncidentErrorType) EqualsIgnoreCase(other IncidentErrorType) bool {
	return strings.EqualFold(s.String(), other.String())
}

func (s IncidentErrorType) In(states ...IncidentErrorType) bool {
	for _, st := range states {
		if s.EqualsIgnoreCase(st) {
			return true
		}
	}
	return false
}

func (s IncidentErrorType) IsUnspecifiedOrUnknown() bool {
	return s.In(IncidentErrorTypeUnspecified, IncidentErrorTypeUnknown)
}

type IncidentErrorTypes []IncidentErrorType

func (sx IncidentErrorTypes) Contains(state IncidentErrorType) bool {
	for _, s := range sx {
		if s.EqualsIgnoreCase(state) {
			return true
		}
	}
	return false
}

func (sx IncidentErrorTypes) Strings() []string {
	out := make([]string, len(sx))
	for i, s := range sx {
		out[i] = s.String()
	}
	return out
}

func (sx IncidentErrorTypes) String() string {
	return strings.Join(sx.Strings(), ", ")
}
