// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

// CompletionSubmissionStatus describes only submission evidence, independently
// of any later observation of the job or its process instance.
type CompletionSubmissionStatus string

const (
	// CompletionSubmissionNotSubmitted means no completion call was made.
	CompletionSubmissionNotSubmitted CompletionSubmissionStatus = "not_submitted"
	// CompletionSubmissionAccepted means the completion endpoint accepted the call.
	CompletionSubmissionAccepted CompletionSubmissionStatus = "accepted"
	// CompletionSubmissionRejected means an explicit backend rejection was received.
	CompletionSubmissionRejected CompletionSubmissionStatus = "rejected"
	// CompletionSubmissionUncertain means dispatch may have succeeded. A later
	// missing job cannot change this evidence into a rejection.
	CompletionSubmissionUncertain CompletionSubmissionStatus = "uncertain"
)

// CompletionSubmission retains the evidence from at most one intentional
// completion attempt. HTTPStatus is absent when the adapter has no response;
// Diagnostic carries normalized error detail without replacing the returned error.
type CompletionSubmission struct {
	Status     CompletionSubmissionStatus `json:"status"`
	HTTPStatus *int                       `json:"httpStatus,omitempty"`
	Diagnostic string                     `json:"diagnostic,omitempty"`
}
