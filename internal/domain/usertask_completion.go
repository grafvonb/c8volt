// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

// UserTaskCompletionResult preserves execution and submission facts for one selected key.
// Accepted status requires accepted submission; preview and not-attempted require not_submitted.
// Timeout/cancellation alone never establishes rejection. Earlier ambiguity survives a later
// rejection, while a later documented acceptance supersedes earlier uncertainty.
type UserTaskCompletionResult struct {
	// Key: Selected task key; always present
	Key string
	// Status: Local execution result, defined below
	Status UserTaskCompletionStatus
	// Submission: Backend acceptance evidence, independently defined below
	Submission UserTaskCompletionSubmission
	// StatusCode: Last observed HTTP status, omitted when none; not proof of an earlier attempt's outcome
	StatusCode int
	// Error: Existing machine-readable failure evidence; never interpolate into human output
	Error string
}

// UserTaskCompletionStatus describes local execution, independently of backend acceptance.
type UserTaskCompletionStatus string

const (
	// UserTaskCompletionStatusPreview is a valid local selection with no submission requested.
	UserTaskCompletionStatusPreview UserTaskCompletionStatus = "preview"
	// UserTaskCompletionStatusAccepted records acceptance without independent task-state confirmation.
	UserTaskCompletionStatusAccepted UserTaskCompletionStatus = "accepted"
	// UserTaskCompletionStatusFailed is a failure other than timeout or caller cancellation.
	UserTaskCompletionStatusFailed UserTaskCompletionStatus = "failed"
	// UserTaskCompletionStatusTimedOut records an attempted operation's timeout, not rejection.
	UserTaskCompletionStatusTimedOut UserTaskCompletionStatus = "timed_out"
	// UserTaskCompletionStatusCanceled records caller cancellation after an attempt, not rejection.
	UserTaskCompletionStatusCanceled UserTaskCompletionStatus = "canceled"
	// UserTaskCompletionStatusNotAttempted means no native operation began.
	UserTaskCompletionStatusNotAttempted UserTaskCompletionStatus = "not_attempted"
)

// UserTaskCompletionSubmission records backend acceptance evidence across all native attempts.
type UserTaskCompletionSubmission string

const (
	// UserTaskCompletionSubmissionNotSubmitted accompanies preview and not-attempted results.
	UserTaskCompletionSubmissionNotSubmitted UserTaskCompletionSubmission = "not_submitted"
	// UserTaskCompletionSubmissionAccepted means a documented successful response was observed.
	UserTaskCompletionSubmissionAccepted UserTaskCompletionSubmission = "accepted"
	// UserTaskCompletionSubmissionRejected means explicit rejection with no earlier ambiguous attempt.
	UserTaskCompletionSubmissionRejected UserTaskCompletionSubmission = "rejected"
	// UserTaskCompletionSubmissionUnknown means at least one attempt lacks conclusive evidence.
	UserTaskCompletionSubmissionUnknown UserTaskCompletionSubmission = "unknown"
)

// UserTaskCompletionResults retains one item per unique key in selection order,
// including unattempted slots and partial evidence on error. Items is initialized
// to an empty array when no item evidence exists.
type UserTaskCompletionResults struct {
	Items []UserTaskCompletionResult
}
