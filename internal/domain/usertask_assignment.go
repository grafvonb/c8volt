// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

// UserTaskAssignmentResult preserves execution and submission facts for one selected key.
// Accepted status requires accepted submission; preview and not-attempted require not_submitted.
// Timeout/cancellation alone never establishes rejection. Earlier ambiguity survives a later
// rejection, while a later documented acceptance supersedes earlier uncertainty.
type UserTaskAssignmentResult struct {
	// Key: Selected unique task key, always present
	Key string
	// Status: Local operation outcome
	Status UserTaskAssignmentStatus
	// Submission: Evidence about backend acceptance
	Submission UserTaskAssignmentSubmission
	// StatusCode: Last observed submission response code; zero when none
	StatusCode int
	// Error: Existing machine-readable error evidence; not human display text
	Error string
}

// UserTaskAssignmentStatus describes local execution, independently of backend acceptance.
type UserTaskAssignmentStatus string

const (
	// UserTaskAssignmentStatusPreview is a valid local selection with no submission requested.
	UserTaskAssignmentStatusPreview UserTaskAssignmentStatus = "preview"
	// UserTaskAssignmentStatusAccepted records acceptance without independent task-state confirmation.
	UserTaskAssignmentStatusAccepted UserTaskAssignmentStatus = "accepted"
	// UserTaskAssignmentStatusFailed is a failure other than timeout or caller cancellation.
	UserTaskAssignmentStatusFailed UserTaskAssignmentStatus = "failed"
	// UserTaskAssignmentStatusTimedOut records an attempted operation's timeout, not rejection.
	UserTaskAssignmentStatusTimedOut UserTaskAssignmentStatus = "timed_out"
	// UserTaskAssignmentStatusCanceled records caller cancellation after an attempt, not rejection.
	UserTaskAssignmentStatusCanceled UserTaskAssignmentStatus = "canceled"
	// UserTaskAssignmentStatusNotAttempted means no assignment-family submission began.
	UserTaskAssignmentStatusNotAttempted UserTaskAssignmentStatus = "not_attempted"
)

// UserTaskAssignmentSubmission records backend acceptance evidence across all native attempts.
type UserTaskAssignmentSubmission string

const (
	// UserTaskAssignmentSubmissionNotSubmitted accompanies preview and not-attempted results.
	UserTaskAssignmentSubmissionNotSubmitted UserTaskAssignmentSubmission = "not_submitted"
	// UserTaskAssignmentSubmissionAccepted means a documented successful response was observed.
	UserTaskAssignmentSubmissionAccepted UserTaskAssignmentSubmission = "accepted"
	// UserTaskAssignmentSubmissionRejected means explicit rejection with no earlier ambiguous attempt.
	UserTaskAssignmentSubmissionRejected UserTaskAssignmentSubmission = "rejected"
	// UserTaskAssignmentSubmissionUnknown means at least one attempt lacks conclusive evidence.
	UserTaskAssignmentSubmissionUnknown UserTaskAssignmentSubmission = "unknown"
)

// UserTaskAssignmentResults preserves ordered partial results for assignment or unassignment.
type UserTaskAssignmentResults struct {
	// Items: One item per unique validated key; initialized empty for validation failures
	Items []UserTaskAssignmentResult
}
