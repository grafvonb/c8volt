// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"context"
	"log/slog"
	"net/http"

	d "github.com/grafvonb/c8volt/internal/domain"
)

// CompleteUserTaskSubmission preserves completion's existing types, messages and native response semantics.
func CompleteUserTaskSubmission(ctx context.Context, key string, log *slog.Logger, submit func(context.Context) (*http.Response, []byte, error)) (d.UserTaskCompletionResult, error) {
	evidence, err := submitUserTask(ctx, "complete user task", "completion", log, submit)
	return d.UserTaskCompletionResult{
		Key: key, Status: d.UserTaskCompletionStatus(evidence.status), Submission: d.UserTaskCompletionSubmission(evidence.submission),
		StatusCode: evidence.statusCode, Error: evidence.errorText,
	}, err
}
