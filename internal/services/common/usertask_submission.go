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

// userTaskSubmissionResult is private response evidence shared only by native user-task mutations.
type userTaskSubmissionResult struct {
	status     string
	submission string
	statusCode int
	errorText  string
}

// submitUserTask retains last-response and sticky ambiguity evidence outside retry return values.
// It never reads or reconciles task state, and passes original transport errors to the retry policy.
func submitUserTask(ctx context.Context, operation, responseName string, log *slog.Logger, submit func(context.Context) (*http.Response, []byte, error)) (userTaskSubmissionResult, error) {
	result := userTaskSubmissionResult{status: "not_attempted", submission: "not_submitted"}
	if err := ctx.Err(); err != nil {
		result.errorText = err.Error()
		return result, err
	}
	var attempted, ambiguous bool
	var lastErr error
	_, retryErr := services.RetryCamundaMutation(ctx, log, operation, func(ctx context.Context) (struct{}, *http.Response, []byte, error) {
		if err := ctx.Err(); err != nil {
			return struct{}{}, nil, nil, err
		}
		attempted = true
		response, body, callErr := submit(ctx)
		if response != nil && response.StatusCode > 0 {
			result.statusCode = response.StatusCode
		}
		lastErr = userTaskSubmissionResponseError(responseName, response, body, callErr)
		if lastErr == nil {
			result.status = "accepted"
			result.submission = "accepted"
		} else {
			// A call error takes precedence over a partial response, even a 204.
			if callErr != nil || !userTaskSubmissionRejected(response, body) {
				ambiguous = true
			}
			result.submission = "rejected"
			if ambiguous {
				result.submission = "unknown"
			}
		}
		// Pass only the original call error: converting HTTP errors here would
		// bypass the retry helper's existing 429/RESOURCE_EXHAUSTED policy.
		return struct{}{}, response, body, callErr
	})
	if result.status == "accepted" {
		return result, nil
	}
	err := lastErr
	if retryErr != nil {
		err = retryErr
	}
	if attempted {
		result.status = "failed"
		var networkErr net.Error
		switch {
		case errors.Is(err, context.Canceled):
			result.status = "canceled"
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, d.ErrGatewayTimeout):
			result.status = "timed_out"
		case errors.As(err, &networkErr) && networkErr.Timeout():
			result.status = "timed_out"
			// Preserve both the transport cause and existing facade timeout classification.
			err = fmt.Errorf("%w: %w", d.ErrGatewayTimeout, err)
		}
	}
	err = fmt.Errorf("%s: %w", operation, err)
	result.errorText = err.Error()
	return result, err
}

// userTaskSubmissionResponseError accepts only documented 204 responses; other
// success codes and missing HTTP metadata are malformed, never proof of acceptance.
func userTaskSubmissionResponseError(responseName string, response *http.Response, body []byte, callErr error) error {
	if callErr != nil {
		return callErr
	}
	if response == nil || response.StatusCode < 100 || response.StatusCode > 599 ||
		response.Request == nil || response.Request.URL == nil {
		return fmt.Errorf("%w: invalid user-task %s HTTP response", d.ErrMalformedResponse, responseName)
	}
	if response.StatusCode == http.StatusNoContent {
		return nil
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return fmt.Errorf("%w: unexpected user-task %s HTTP status %d", d.ErrMalformedResponse, responseName, response.StatusCode)
	}
	return httpc.HttpStatusErr(response, body)
}

// userTaskSubmissionRejected recognizes explicit backend rejection only; generic
// server/proxy failures and malformed responses leave submission uncertain.
func userTaskSubmissionRejected(response *http.Response, body []byte) bool {
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
