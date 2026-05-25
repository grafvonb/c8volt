// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import "time"

type Job struct {
	Key                string     `json:"key,omitempty"`
	State              string     `json:"state,omitempty"`
	Retries            int32      `json:"retries"`
	Deadline           *time.Time `json:"deadline,omitempty"`
	Type               string     `json:"type,omitempty"`
	Worker             string     `json:"worker,omitempty"`
	Kind               string     `json:"kind,omitempty"`
	ListenerEventType  string     `json:"listenerEventType,omitempty"`
	ProcessInstanceKey string     `json:"processInstanceKey,omitempty"`
	ElementInstanceKey string     `json:"elementInstanceKey,omitempty"`
	ElementId          string     `json:"elementId,omitempty"`
	ErrorCode          string     `json:"errorCode,omitempty"`
	ErrorMessage       string     `json:"errorMessage,omitempty"`
	TenantId           string     `json:"tenantId,omitempty"`
}

type SearchRequest struct {
	Key                string
	State              string
	Type               string
	ProcessInstanceKey string
	ElementInstanceKey string
	ElementId          string
	Worker             string
	Retries            *int32
	Kind               string
	ListenerEventType  string
	BatchSize          int32
	Limit              int32
}

func (r SearchRequest) HasKey() bool {
	return r.Key != ""
}

func (r SearchRequest) HasSearchFilters() bool {
	return r.State != "" ||
		r.Type != "" ||
		r.ProcessInstanceKey != "" ||
		r.ElementInstanceKey != "" ||
		r.ElementId != "" ||
		r.Worker != "" ||
		r.Retries != nil ||
		r.Kind != "" ||
		r.ListenerEventType != ""
}

type SearchResult struct {
	Items []Job `json:"items"`
	Limit int32 `json:"limit"`
}

type UpdateRequest struct {
	Key              string
	Retries          *int32
	Timeout          *time.Duration
	TimeoutRaw       string
	TimeoutMillis    *int64
	NoWait           bool
	AutoConfirm      bool
	Automation       bool
	DryRun           bool
	UpdatePlan       *UpdatePlan
	ConfirmRetries   bool
	SkipConfirmation bool
	WorkerOutcome    *WorkerOutcomeRequest
}

func (r UpdateRequest) HasRetries() bool {
	return r.Retries != nil
}

func (r UpdateRequest) HasTimeout() bool {
	return r.TimeoutMillis != nil
}

func (r UpdateRequest) HasUpdates() bool {
	return r.HasRetries() || r.HasTimeout()
}

type RetryChangeStatus string

const (
	RetryChangeNotRequested RetryChangeStatus = "not_requested"
	RetryChangeChanged      RetryChangeStatus = "changed"
	RetryChangeUnchanged    RetryChangeStatus = "unchanged"
)

type MutationMode string

const (
	MutationModeUpdate           MutationMode = "update"
	MutationModeTechnicalFailure MutationMode = "technical_failure"
	MutationModeBPMNError        MutationMode = "bpmn_error"
	MutationModeCompletion       MutationMode = "completion"
)

type UpdatePlan struct {
	Key               string            `json:"key,omitempty"`
	Current           Job               `json:"current,omitempty"`
	Mode              MutationMode      `json:"mode,omitempty"`
	RequestedRetries  *int32            `json:"requestedRetries,omitempty"`
	RetryStatus       RetryChangeStatus `json:"retryStatus,omitempty"`
	RequestedTimeout  string            `json:"requestedTimeout,omitempty"`
	TimeoutMillis     *int64            `json:"timeoutMillis,omitempty"`
	Message           string            `json:"message,omitempty"`
	RetryBackoff      string            `json:"retryBackoff,omitempty"`
	RetryBackoffMS    *int64            `json:"retryBackoffMs,omitempty"`
	ErrorCode         string            `json:"errorCode,omitempty"`
	Variables         map[string]any    `json:"variables,omitempty"`
	MaterialChange    bool              `json:"materialChange"`
	DryRun            bool              `json:"dryRun"`
	MutationSubmitted bool              `json:"mutationSubmitted"`
	Items             []UpdatePlanItem  `json:"items,omitempty"`
}

func (p UpdatePlan) HasMaterialChange() bool {
	return p.MaterialChange
}

type UpdatePlanItem struct {
	Name   string `json:"name,omitempty"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
	Status string `json:"status,omitempty"`
}

type UpdateResult struct {
	Key                string      `json:"key,omitempty"`
	Status             string      `json:"status,omitempty"`
	Plan               *UpdatePlan `json:"plan,omitempty"`
	MutationAccepted   bool        `json:"mutationAccepted"`
	ConfirmationStatus string      `json:"confirmationStatus,omitempty"`
	SubmittedRetries   *int32      `json:"submittedRetries,omitempty"`
	SubmittedTimeoutMS *int64      `json:"submittedTimeoutMs,omitempty"`
	ConfirmedRetries   *int32      `json:"confirmedRetries,omitempty"`
	Error              string      `json:"error,omitempty"`
}

type WorkerOutcomeMode string

const (
	WorkerOutcomeTechnicalFailure WorkerOutcomeMode = "technical_failure"
	WorkerOutcomeBPMNError        WorkerOutcomeMode = "bpmn_error"
	WorkerOutcomeCompletion       WorkerOutcomeMode = "completion"
)

type WorkerOutcomeRequest struct {
	Key                string
	Mode               WorkerOutcomeMode
	Message            string
	Variables          map[string]any
	Retries            *int32
	RetryBackoff       *time.Duration
	RetryBackoffRaw    string
	RetryBackoffMillis *int64
	ErrorCode          string
	NoWait             bool
	AutoConfirm        bool
	Automation         bool
	DryRun             bool
	OutcomePlan        *UpdatePlan
}

type WorkerOutcomeResult struct {
	Key                string            `json:"key,omitempty"`
	Mode               WorkerOutcomeMode `json:"mode,omitempty"`
	Status             string            `json:"status,omitempty"`
	Plan               *UpdatePlan       `json:"plan,omitempty"`
	MutationAccepted   bool              `json:"mutationAccepted"`
	ConfirmationStatus string            `json:"confirmationStatus,omitempty"`
	SubmittedRetries   *int32            `json:"submittedRetries,omitempty"`
	SubmittedBackoffMS *int64            `json:"submittedBackoffMs,omitempty"`
	SubmittedErrorCode string            `json:"submittedErrorCode,omitempty"`
	Error              string            `json:"error,omitempty"`
}
