// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"time"

	"github.com/grafvonb/c8volt/toolx"
)

const (
	JobKindExecutionListener = "EXECUTION_LISTENER"
	JobKindTaskListener      = "TASK_LISTENER"
)

// Job retains the backend observation, including its independently optional business identity.
type Job struct {
	BusinessId         *string    `json:"businessId,omitempty"`
	Key                string     `json:"key,omitempty"`
	State              string     `json:"state,omitempty"`
	Retries            int32      `json:"retries"`
	CreationTime       *time.Time `json:"creationTime,omitempty"`
	EndTime            *time.Time `json:"endTime,omitempty"`
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

// RuntimeListenerJob retains the listener job observation independently of its owning process.
type RuntimeListenerJob struct {
	BusinessId         *string    `json:"businessId,omitempty"`
	JobKey             string     `json:"jobKey,omitempty"`
	Kind               string     `json:"kind,omitempty"`
	ListenerEventType  string     `json:"listenerEventType,omitempty"`
	Type               string     `json:"type,omitempty"`
	State              string     `json:"state,omitempty"`
	Retries            int32      `json:"retries"`
	Worker             string     `json:"worker,omitempty"`
	CreationTime       *time.Time `json:"creationTime,omitempty"`
	EndTime            *time.Time `json:"endTime,omitempty"`
	Deadline           *time.Time `json:"deadline,omitempty"`
	ProcessInstanceKey string     `json:"processInstanceKey,omitempty"`
	ElementInstanceKey string     `json:"elementInstanceKey,omitempty"`
	ElementId          string     `json:"elementId,omitempty"`
	TenantId           string     `json:"tenantId,omitempty"`
	ErrorCode          string     `json:"errorCode,omitempty"`
	ErrorMessage       string     `json:"errorMessage,omitempty"`
}

// RuntimeListenerJobFromJob snapshots the job identity without consulting its parent.
func RuntimeListenerJobFromJob(job Job) RuntimeListenerJob {
	return RuntimeListenerJob{
		BusinessId:         toolx.CopyPtr(job.BusinessId),
		JobKey:             job.Key,
		Kind:               job.Kind,
		ListenerEventType:  job.ListenerEventType,
		Type:               job.Type,
		State:              job.State,
		Retries:            job.Retries,
		Worker:             job.Worker,
		CreationTime:       job.CreationTime,
		EndTime:            job.EndTime,
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

// JobSearchQuery describes predicates and traversal controls for job discovery.
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
	// SortByKey requests stable ascending key order for frozen bulk discovery.
	SortByKey bool
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

// JobPageRequest retains the raw offset even when a cursor drives the request.
type JobPageRequest struct {
	From  int32
	Size  int32
	After string
}

// JobSearchPage preserves raw traversal evidence independently of selected rows.
type JobSearchPage struct {
	Items         []Job
	Request       JobPageRequest
	OverflowState ProcessInstanceOverflowState
	ReportedTotal *JobReportedTotal
	// RawItemCount is the response count before overfull trimming or local filtering.
	RawItemCount int32
	// NextFrom advances by the consumed raw window, before local filtering.
	NextFrom  int32
	EndCursor string
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

// JobWorkerOutcomeRequest carries optional assignment only for completion.
// Nil means absent in existing job operations. Direct assignment requires a value.
// Copy pointers at request conversion and plan preparation boundaries. Do not share mutable caller storage.
type JobWorkerOutcomeRequest struct {
	// SetBusinessID is optional assignment intent, independent of observed identity.
	SetBusinessID      *string `json:"requestedBusinessId,omitempty"`
	Key                string
	Mode               JobWorkerOutcomeMode
	Message            string
	Variables          map[string]any
	Retries            *int32
	RetryBackoffMillis *int64
	ErrorCode          string
	SkipConfirmation   bool
}

// JobWorkerOutcomeResult retains requested identity separately from submission and observation.
// No assignment field is emitted when intent is absent.
type JobWorkerOutcomeResult struct {
	Submission           *CompletionSubmission  `json:"submission,omitempty"`
	Observation          *CompletionObservation `json:"observation,omitempty"`
	RequestedBusinessID  *string                `json:"requestedBusinessId,omitempty"`
	Assignment           *CompletionAssignment  `json:"assignment,omitempty"`
	Key                  string                 `json:"key,omitempty"`
	Mode                 JobWorkerOutcomeMode   `json:"mode,omitempty"`
	MutationAccepted     bool                   `json:"mutationAccepted"`
	ConfirmationStatus   string                 `json:"confirmationStatus,omitempty"`
	SubmittedRetries     *int32                 `json:"submittedRetries,omitempty"`
	SubmittedBackoffMS   *int64                 `json:"submittedBackoffMs,omitempty"`
	SubmittedErrorCode   string                 `json:"submittedErrorCode,omitempty"`
	MutationError        string                 `json:"mutationError,omitempty"`
	UnsupportedOperation bool                   `json:"unsupportedOperation,omitempty"`
}
