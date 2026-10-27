// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

// ProcessInstanceResumptionResult separates submission proof from observation of
// the selected instance. Error is machine evidence, never human diagnostic text.
type ProcessInstanceResumptionResult struct {
	Key          string                                `json:"key"`
	Status       ProcessInstanceResumptionStatus       `json:"status"`
	Submission   ProcessInstanceResumptionSubmission   `json:"submission"`
	Confirmation ProcessInstanceResumptionConfirmation `json:"confirmation"`
	// StatusCode is the last valid submission status, never a read status.
	StatusCode int `json:"statusCode,omitempty"`
	// HasObservation distinguishes no observation from an observed UNKNOWN state.
	HasObservation    bool  `json:"hasObservation"`
	LastObservedState State `json:"lastObservedState,omitempty"`
	// SuspendedDate comes only from the latest successful matching observation.
	SuspendedDate string `json:"suspendedDate,omitempty"`
	Error         string `json:"error,omitempty"`
}

// ProcessInstanceResumptionStatus describes workflow disposition, not engine state.
type ProcessInstanceResumptionStatus string

const (
	// ProcessInstanceResumptionStatusPreview records a local preview without I/O.
	ProcessInstanceResumptionStatusPreview ProcessInstanceResumptionStatus = "preview"
	// ProcessInstanceResumptionStatusAccepted records acceptance without confirmation.
	ProcessInstanceResumptionStatusAccepted ProcessInstanceResumptionStatus = "accepted"
	// ProcessInstanceResumptionStatusConfirmed requires accepted submission and observed ACTIVE or COMPLETED.
	ProcessInstanceResumptionStatusConfirmed ProcessInstanceResumptionStatus = "confirmed"
	// ProcessInstanceResumptionStatusFailed records a failure other than timeout or interruption.
	ProcessInstanceResumptionStatusFailed ProcessInstanceResumptionStatus = "failed"
	// ProcessInstanceResumptionStatusTimedOut retains evidence when a deadline or attempt bound ends work.
	ProcessInstanceResumptionStatusTimedOut ProcessInstanceResumptionStatus = "timed_out"
	// ProcessInstanceResumptionStatusInterrupted records caller cancellation without inferring rejection.
	ProcessInstanceResumptionStatusInterrupted ProcessInstanceResumptionStatus = "interrupted"
	// ProcessInstanceResumptionStatusNotAttempted identifies work that was not started.
	ProcessInstanceResumptionStatusNotAttempted ProcessInstanceResumptionStatus = "not_attempted"
)

// ProcessInstanceResumptionSubmission retains cumulative evidence across submission retries.
type ProcessInstanceResumptionSubmission string

const (
	// ProcessInstanceResumptionSubmissionNotSubmitted means no request was dispatched.
	ProcessInstanceResumptionSubmissionNotSubmitted ProcessInstanceResumptionSubmission = "not_submitted"
	// ProcessInstanceResumptionSubmissionAccepted requires a valid documented acceptance response.
	ProcessInstanceResumptionSubmissionAccepted ProcessInstanceResumptionSubmission = "accepted"
	// ProcessInstanceResumptionSubmissionRejected requires explicit rejection and no earlier ambiguity.
	ProcessInstanceResumptionSubmissionRejected ProcessInstanceResumptionSubmission = "rejected"
	// ProcessInstanceResumptionSubmissionUnknown retains ambiguity until a valid acceptance occurs.
	ProcessInstanceResumptionSubmissionUnknown ProcessInstanceResumptionSubmission = "unknown"
)

// ProcessInstanceResumptionConfirmation records observation independently of submission.
type ProcessInstanceResumptionConfirmation string

const (
	// ProcessInstanceResumptionConfirmationNotRequested covers previews, no-wait, skipped and unaccepted work.
	ProcessInstanceResumptionConfirmationNotRequested ProcessInstanceResumptionConfirmation = "not_requested"
	// ProcessInstanceResumptionConfirmationUnconfirmed means accepted work awaits proof of ACTIVE or COMPLETED.
	ProcessInstanceResumptionConfirmationUnconfirmed ProcessInstanceResumptionConfirmation = "unconfirmed"
	// ProcessInstanceResumptionConfirmationConfirmed requires a matching ACTIVE or COMPLETED observation after acceptance.
	ProcessInstanceResumptionConfirmationConfirmed ProcessInstanceResumptionConfirmation = "confirmed"
)

// ProcessInstanceResumptionResults retains one item per normalized key in input
// order, including skipped work and partial failures. Callers initialize Items
// to an empty array when there are no selected keys.
type ProcessInstanceResumptionResults struct {
	Items []ProcessInstanceResumptionResult `json:"items"`
}
