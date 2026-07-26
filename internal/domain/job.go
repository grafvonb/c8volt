// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import "time"

const (
	JobKindExecutionListener = "EXECUTION_LISTENER"
	JobKindTaskListener      = "TASK_LISTENER"
)

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

type RuntimeListenerJob struct {
	JobKey             string     `json:"jobKey,omitempty"`
	Kind               string     `json:"kind,omitempty"`
	ListenerEventType  string     `json:"listenerEventType,omitempty"`
	Type               string     `json:"type,omitempty"`
	State              string     `json:"state,omitempty"`
	Retries            int32      `json:"retries"`
	Worker             string     `json:"worker,omitempty"`
	Deadline           *time.Time `json:"deadline,omitempty"`
	ProcessInstanceKey string     `json:"processInstanceKey,omitempty"`
	ElementInstanceKey string     `json:"elementInstanceKey,omitempty"`
	ElementId          string     `json:"elementId,omitempty"`
	TenantId           string     `json:"tenantId,omitempty"`
	ErrorCode          string     `json:"errorCode,omitempty"`
	ErrorMessage       string     `json:"errorMessage,omitempty"`
}

func RuntimeListenerJobFromJob(job Job) RuntimeListenerJob {
	return RuntimeListenerJob{
		JobKey:             job.Key,
		Kind:               job.Kind,
		ListenerEventType:  job.ListenerEventType,
		Type:               job.Type,
		State:              job.State,
		Retries:            job.Retries,
		Worker:             job.Worker,
		Deadline:           job.Deadline,
		ProcessInstanceKey: job.ProcessInstanceKey,
		ElementInstanceKey: job.ElementInstanceKey,
		ElementId:          job.ElementId,
		TenantId:           job.TenantId,
		ErrorCode:          job.ErrorCode,
		ErrorMessage:       job.ErrorMessage,
	}
}

func IsRuntimeListenerJobKind(kind string) bool {
	return kind == JobKindExecutionListener || kind == JobKindTaskListener
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
	BatchSize          int32
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

// JobSearchPageAction tells service-owned page traversal whether the caller
// needs more pages after observing the current page.
type JobSearchPageAction string

const (
	// JobSearchPageActionContinue keeps collecting the next available page.
	JobSearchPageActionContinue JobSearchPageAction = "continue"
	// JobSearchPageActionStop stops traversal after the current page.
	JobSearchPageActionStop JobSearchPageAction = "stop"
)

type JobReportedTotalKind string

const (
	JobReportedTotalKindExact      JobReportedTotalKind = "exact"
	JobReportedTotalKindLowerBound JobReportedTotalKind = "lower_bound"
)

type JobReportedTotal struct {
	Count int64
	Kind  JobReportedTotalKind
}

type JobPageRequest struct {
	From int32
	Size int32
}

type JobSearchPage struct {
	Items         []Job
	Request       JobPageRequest
	OverflowState ProcessInstanceOverflowState
	ReportedTotal *JobReportedTotal
}

// JobSearchPageStep carries one selected page plus service-owned traversal
// state to callers that still own rendering or prompt policy.
type JobSearchPageStep struct {
	Page            JobSearchPage
	CumulativeCount int32
	LimitReached    bool
}

// JobSearchPageVisitor observes each selected page during service-owned
// traversal and may stop collection without owning offset math.
type JobSearchPageVisitor func(JobSearchPageStep) (JobSearchPageAction, error)

// JobSearchPagesResult captures a full or caller-stopped paged discovery.
type JobSearchPagesResult struct {
	Items []Job
	Limit int32
	Pages int32
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
