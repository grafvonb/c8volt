// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

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

type JobSearchQuery struct {
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
	Limit              int32
}

func (q JobSearchQuery) HasKey() bool {
	return q.Key != ""
}

func (q JobSearchQuery) HasSearchFilters() bool {
	return q.State != "" ||
		q.Type != "" ||
		q.ProcessInstanceKey != "" ||
		q.ElementInstanceKey != "" ||
		q.ElementId != "" ||
		q.Worker != "" ||
		q.Retries != nil ||
		q.Kind != "" ||
		q.ListenerEventType != ""
}

type JobSearchResult struct {
	Items []Job `json:"items"`
	Limit int32 `json:"limit"`
}

type JobUpdateRequest struct {
	Key               string
	Retries           *int32
	TimeoutMillis     *int64
	SkipConfirmation  bool
	ConfirmRetries    bool
	RequestedTimeout  string
	RequestedDuration time.Duration
}

func (r JobUpdateRequest) HasRetries() bool {
	return r.Retries != nil
}

func (r JobUpdateRequest) HasTimeout() bool {
	return r.TimeoutMillis != nil
}

func (r JobUpdateRequest) HasUpdates() bool {
	return r.HasRetries() || r.HasTimeout()
}

type JobUpdateResult struct {
	Key                  string `json:"key,omitempty"`
	MutationAccepted     bool   `json:"mutationAccepted"`
	ConfirmationStatus   string `json:"confirmationStatus,omitempty"`
	ConfirmedRetries     *int32 `json:"confirmedRetries,omitempty"`
	SubmittedRetries     *int32 `json:"submittedRetries,omitempty"`
	SubmittedTimeoutMS   *int64 `json:"submittedTimeoutMs,omitempty"`
	ConfirmationError    string `json:"confirmationError,omitempty"`
	MutationError        string `json:"mutationError,omitempty"`
	UnsupportedOperation bool   `json:"unsupportedOperation,omitempty"`
}

type JobWorkerOutcomeMode string

const (
	JobWorkerOutcomeTechnicalFailure JobWorkerOutcomeMode = "technical_failure"
	JobWorkerOutcomeBPMNError        JobWorkerOutcomeMode = "bpmn_error"
	JobWorkerOutcomeCompletion       JobWorkerOutcomeMode = "completion"
)

type JobWorkerOutcomeRequest struct {
	Key                string
	Mode               JobWorkerOutcomeMode
	Message            string
	Variables          map[string]any
	Retries            *int32
	RetryBackoffMillis *int64
	ErrorCode          string
	SkipConfirmation   bool
}

type JobWorkerOutcomeResult struct {
	Key                  string               `json:"key,omitempty"`
	Mode                 JobWorkerOutcomeMode `json:"mode,omitempty"`
	MutationAccepted     bool                 `json:"mutationAccepted"`
	ConfirmationStatus   string               `json:"confirmationStatus,omitempty"`
	SubmittedRetries     *int32               `json:"submittedRetries,omitempty"`
	SubmittedBackoffMS   *int64               `json:"submittedBackoffMs,omitempty"`
	SubmittedErrorCode   string               `json:"submittedErrorCode,omitempty"`
	MutationError        string               `json:"mutationError,omitempty"`
	UnsupportedOperation bool                 `json:"unsupportedOperation,omitempty"`
}
