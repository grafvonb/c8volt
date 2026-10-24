// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/httpc"
)

// CompleteUserTaskSubmission applies the existing finite mutation retry policy while
// retaining evidence that the retry helper can discard on backoff cancellation.
// Version adapters own generated request mapping; this shared path never reads or
// reconciles task state. A nil logger suppresses functional retry progress only.
func CompleteUserTaskSubmission(ctx context.Context, key string, log *slog.Logger, submit func(context.Context) (*http.Response, []byte, error)) (d.UserTaskCompletionResult, error) {
	result := d.UserTaskCompletionResult{
		Key: key, Status: d.UserTaskCompletionStatusNotAttempted,
		Submission: d.UserTaskCompletionSubmissionNotSubmitted,
	}
	if err := ctx.Err(); err != nil {
		result.Error = err.Error()
		return result, err
	}
	var attempted, ambiguous bool
	var lastErr error
	_, retryErr := services.RetryCamundaMutation(ctx, log, "complete user task", func(ctx context.Context) (struct{}, *http.Response, []byte, error) {
		if err := ctx.Err(); err != nil {
			return struct{}{}, nil, nil, err
		}
		attempted = true
		response, body, callErr := submit(ctx)
		if response != nil && response.StatusCode > 0 {
			result.StatusCode = response.StatusCode
		}
		lastErr = userTaskCompletionResponseError(response, body, callErr)
		if lastErr == nil {
			result.Status = d.UserTaskCompletionStatusAccepted
			result.Submission = d.UserTaskCompletionSubmissionAccepted
		} else {
			// A call error takes precedence over a partial response, even a 204.
			if callErr != nil || !userTaskCompletionRejected(response, body) {
				ambiguous = true
			}
			result.Submission = d.UserTaskCompletionSubmissionRejected
			if ambiguous {
				result.Submission = d.UserTaskCompletionSubmissionUnknown
			}
		}
		// Pass only the original call error: converting HTTP errors here would
		// bypass the retry helper's existing 429/RESOURCE_EXHAUSTED policy.
		return struct{}{}, response, body, callErr
	})
	if result.Status == d.UserTaskCompletionStatusAccepted {
		return result, nil
	}
	err := lastErr
	if retryErr != nil {
		err = retryErr
	}
	if attempted {
		result.Status = d.UserTaskCompletionStatusFailed
		var networkErr net.Error
		switch {
		case errors.Is(err, context.Canceled):
			result.Status = d.UserTaskCompletionStatusCanceled
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, d.ErrGatewayTimeout):
			result.Status = d.UserTaskCompletionStatusTimedOut
		case errors.As(err, &networkErr) && networkErr.Timeout():
			result.Status = d.UserTaskCompletionStatusTimedOut
			// Preserve both the transport cause and existing facade timeout classification.
			err = fmt.Errorf("%w: %w", d.ErrGatewayTimeout, err)
		}
	}
	err = fmt.Errorf("complete user task: %w", err)
	result.Error = err.Error()
	return result, err
}

// userTaskCompletionResponseError accepts only documented 204 responses; other
// success codes and missing HTTP metadata are malformed, never proof of acceptance.
func userTaskCompletionResponseError(response *http.Response, body []byte, callErr error) error {
	if callErr != nil {
		return callErr
	}
	if response == nil || response.StatusCode < 100 || response.StatusCode > 599 ||
		response.Request == nil || response.Request.URL == nil {
		return fmt.Errorf("%w: invalid user-task completion HTTP response", d.ErrMalformedResponse)
	}
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return fmt.Errorf("%w: unexpected user-task completion HTTP status %d", d.ErrMalformedResponse, response.StatusCode)
	}
	return httpc.HttpStatusErr(response, body)
}

// userTaskCompletionRejected recognizes explicit backend rejection only; generic
// server/proxy failures and malformed responses leave submission uncertain.
func userTaskCompletionRejected(response *http.Response, body []byte) bool {
	if response == nil || response.Request == nil || response.Request.URL == nil {
		return false
	}
	switch response.StatusCode {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden,
		http.StatusNotFound, http.StatusConflict, http.StatusPreconditionFailed,
		http.StatusUnsupportedMediaType, http.StatusUnprocessableEntity, http.StatusTooManyRequests:
		return true
	case http.StatusServiceUnavailable:
		return completionResourceExhausted(body)
	default:
		return false
	}
}
