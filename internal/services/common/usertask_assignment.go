// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	d "github.com/grafvonb/c8volt/internal/domain"
)

// AssignUserTaskSubmission retains native assignment evidence without reading or reconciling task state.
func AssignUserTaskSubmission(ctx context.Context, key string, log *slog.Logger, submit func(context.Context) (*http.Response, []byte, error)) (d.UserTaskAssignmentResult, error) {
	var alreadyAssigned bool
	evidence, err := submitUserTask(ctx, "assign user task", "assignment", log, func(ctx context.Context) (*http.Response, []byte, error) {
		response, body, callErr := submit(ctx)
		alreadyAssigned = false
		if callErr == nil && response != nil && response.StatusCode == http.StatusConflict {
			var problem struct {
				Detail string `json:"detail"`
			}
			// Match Camunda's claim rejection for this key, including a possible
			// gateway prefix. Other state/listener conflicts must remain errors.
			if json.Unmarshal(body, &problem) == nil {
				alreadyAssigned = strings.HasSuffix(problem.Detail, fmt.Sprintf("Expected to claim user task with key '%s', but it has already been assigned", key))
			}
		}
		return response, body, callErr
	})
	result := assignmentSubmissionResult(key, evidence)
	result.ReassignmentDenied = alreadyAssigned && result.Status == d.UserTaskAssignmentStatusFailed && result.Submission == d.UserTaskAssignmentSubmissionRejected
	return result, err
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
