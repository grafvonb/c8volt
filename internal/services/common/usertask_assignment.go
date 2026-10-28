// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"context"
	"log/slog"
	"net/http"

	d "github.com/grafvonb/c8volt/internal/domain"
)

// AssignUserTaskSubmission retains native assignment evidence without reading or reconciling task state.
func AssignUserTaskSubmission(ctx context.Context, key string, log *slog.Logger, submit func(context.Context) (*http.Response, []byte, error)) (d.UserTaskAssignmentResult, error) {
	evidence, err := submitUserTask(ctx, "assign user task", "assignment", log, submit)
	return assignmentSubmissionResult(key, evidence), err
}

// UnassignUserTaskSubmission retains native removal evidence without suppressing backend rejections.
func UnassignUserTaskSubmission(ctx context.Context, key string, log *slog.Logger, submit func(context.Context) (*http.Response, []byte, error)) (d.UserTaskAssignmentResult, error) {
	evidence, err := submitUserTask(ctx, "unassign user task", "unassignment", log, submit)
	return assignmentSubmissionResult(key, evidence), err
}

// assignmentSubmissionResult maps private response facts to the assignment-specific domain contract.
func assignmentSubmissionResult(key string, evidence userTaskSubmissionResult) d.UserTaskAssignmentResult {
	return d.UserTaskAssignmentResult{
		Key: key, Status: d.UserTaskAssignmentStatus(evidence.status), Submission: d.UserTaskAssignmentSubmission(evidence.submission),
		StatusCode: evidence.statusCode, Error: evidence.errorText,
	}
}
