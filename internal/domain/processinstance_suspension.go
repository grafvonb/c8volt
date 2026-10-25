// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

// ProcessInstanceSuspensionResult separates submission proof from observation of
// the selected instance. Error is machine evidence, never human diagnostic text.
type ProcessInstanceSuspensionResult struct {
	Key          string                                `json:"key"`
	Status       ProcessInstanceSuspensionStatus       `json:"status"`
	Submission   ProcessInstanceSuspensionSubmission   `json:"submission"`
	Confirmation ProcessInstanceSuspensionConfirmation `json:"confirmation"`
	// StatusCode is the last valid submission status, never a read status.
	StatusCode int `json:"statusCode,omitempty"`
	// HasObservation distinguishes no observation from an observed UNKNOWN state.
	HasObservation    bool  `json:"hasObservation"`
	LastObservedState State `json:"lastObservedState,omitempty"`
	// SuspendedDate comes only from the latest successful matching observation.
	SuspendedDate string `json:"suspendedDate,omitempty"`
	Error         string `json:"error,omitempty"`
}

// ProcessInstanceSuspensionStatus describes workflow disposition, not engine state.
type ProcessInstanceSuspensionStatus string

const (
	// ProcessInstanceSuspensionStatusPreview records a local preview without I/O.
	ProcessInstanceSuspensionStatusPreview ProcessInstanceSuspensionStatus = "preview"
	// ProcessInstanceSuspensionStatusAccepted records acceptance without confirmation.
	ProcessInstanceSuspensionStatusAccepted ProcessInstanceSuspensionStatus = "accepted"
	// ProcessInstanceSuspensionStatusConfirmed requires accepted submission and observed SUSPENDED.
	ProcessInstanceSuspensionStatusConfirmed ProcessInstanceSuspensionStatus = "confirmed"
	// ProcessInstanceSuspensionStatusFailed records a failure other than timeout or interruption.
	ProcessInstanceSuspensionStatusFailed ProcessInstanceSuspensionStatus = "failed"
	// ProcessInstanceSuspensionStatusTimedOut retains evidence when a deadline or attempt bound ends work.
	ProcessInstanceSuspensionStatusTimedOut ProcessInstanceSuspensionStatus = "timed_out"
	// ProcessInstanceSuspensionStatusInterrupted records caller cancellation without inferring rejection.
	ProcessInstanceSuspensionStatusInterrupted ProcessInstanceSuspensionStatus = "interrupted"
	// ProcessInstanceSuspensionStatusNotAttempted identifies work that was not started.
	ProcessInstanceSuspensionStatusNotAttempted ProcessInstanceSuspensionStatus = "not_attempted"
)

// ProcessInstanceSuspensionSubmission retains cumulative evidence across submission retries.
type ProcessInstanceSuspensionSubmission string

const (
	// ProcessInstanceSuspensionSubmissionNotSubmitted means no request was dispatched.
	ProcessInstanceSuspensionSubmissionNotSubmitted ProcessInstanceSuspensionSubmission = "not_submitted"
	// ProcessInstanceSuspensionSubmissionAccepted requires a valid documented acceptance response.
	ProcessInstanceSuspensionSubmissionAccepted ProcessInstanceSuspensionSubmission = "accepted"
	// ProcessInstanceSuspensionSubmissionRejected requires explicit rejection and no earlier ambiguity.
	ProcessInstanceSuspensionSubmissionRejected ProcessInstanceSuspensionSubmission = "rejected"
	// ProcessInstanceSuspensionSubmissionUnknown retains ambiguity until a valid acceptance occurs.
	ProcessInstanceSuspensionSubmissionUnknown ProcessInstanceSuspensionSubmission = "unknown"
)

// ProcessInstanceSuspensionConfirmation records observation independently of submission.
type ProcessInstanceSuspensionConfirmation string

const (
	// ProcessInstanceSuspensionConfirmationNotRequested covers previews, no-wait, skipped and unaccepted work.
	ProcessInstanceSuspensionConfirmationNotRequested ProcessInstanceSuspensionConfirmation = "not_requested"
	// ProcessInstanceSuspensionConfirmationUnconfirmed means accepted work awaits proof of SUSPENDED.
	ProcessInstanceSuspensionConfirmationUnconfirmed ProcessInstanceSuspensionConfirmation = "unconfirmed"
	// ProcessInstanceSuspensionConfirmationConfirmed requires a matching SUSPENDED observation after acceptance.
	ProcessInstanceSuspensionConfirmationConfirmed ProcessInstanceSuspensionConfirmation = "confirmed"
)

// ProcessInstanceSuspensionResults retains one item per normalized key in input
// order, including skipped work and partial failures. Callers initialize Items
// to an empty array when there are no selected keys.
type ProcessInstanceSuspensionResults struct {
	Items []ProcessInstanceSuspensionResult `json:"items"`
}
