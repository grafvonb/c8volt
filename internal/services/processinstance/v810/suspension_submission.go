// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/httpc"
)

// submitProcessInstanceSuspension submits only the supplied key and never reads
// state. Evidence lives outside the retry return value so backoff cancellation
// cannot erase an earlier response. Functional retry chatter is suppressed; the
// configured HTTP transport retains its independent DEBUG diagnostics.
func (s *Service) submitProcessInstanceSuspension(ctx context.Context, key string) (d.ProcessInstanceSuspensionResult, error) {
	result := d.ProcessInstanceSuspensionResult{
		Key: key, Status: d.ProcessInstanceSuspensionStatusNotAttempted,
		Submission:   d.ProcessInstanceSuspensionSubmissionNotSubmitted,
		Confirmation: d.ProcessInstanceSuspensionConfirmationNotRequested,
	}
	var lastErr error
	_, retryErr := services.RetryCamundaMutation(ctx, nil, "suspend process instance", func(ctx context.Context) (struct{}, *http.Response, []byte, error) {
		if err := ctx.Err(); err != nil {
			return struct{}{}, nil, nil, err
		}
		attemptCtx, observation := httpc.BeginRequestRateAttempt(ctx)
		resp, callErr := s.cc.SuspendProcessInstanceWithResponse(attemptCtx, camundav810.ProcessInstanceKey(key), camundav810.SuspendProcessInstanceJSONRequestBody{})
		var response *http.Response
		var body []byte
		if resp != nil {
			response, body = resp.HTTPResponse, resp.Body
		}
		// HTTP Client.Timeout may replace the typed local error. Restore it before
		// retry classification, retaining the original transport cause as evidence.
		if localErr := observation.Failure(); localErr != nil {
			return struct{}{}, response, body, errors.Join(callErr, localErr)
		}
		if httpc.IsRequestRateWaitError(callErr) {
			return struct{}{}, response, body, callErr
		}
		validHTTP := validSuspensionHTTPResponse(response)
		if validHTTP {
			result.StatusCode = response.StatusCode
		}
		lastErr = suspensionResponseError(response, body, callErr)
		if lastErr == nil {
			result.Status = d.ProcessInstanceSuspensionStatusAccepted
			result.Submission = d.ProcessInstanceSuspensionSubmissionAccepted
		} else if callErr == nil && validHTTP && suspensionRejected(response, body) {
			// A later rejection cannot prove that an earlier ambiguous attempt failed.
			if result.Submission != d.ProcessInstanceSuspensionSubmissionUnknown {
				result.Submission = d.ProcessInstanceSuspensionSubmissionRejected
			}
		} else {
			result.Submission = d.ProcessInstanceSuspensionSubmissionUnknown
		}
		// Preserve the raw call error and response for the existing retry policy;
		// converting 429/503 into domain errors here would suppress their retries.
		return struct{}{}, response, body, callErr
	})
	if result.Submission == d.ProcessInstanceSuspensionSubmissionAccepted {
		return result, nil
	}
	err := lastErr
	if retryErr != nil {
		err = retryErr
	}
	result.Status = d.ProcessInstanceSuspensionStatusFailed
	var networkErr net.Error
	switch {
	case errors.Is(err, context.Canceled):
		result.Status = d.ProcessInstanceSuspensionStatusInterrupted
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, d.ErrGatewayTimeout):
		result.Status = d.ProcessInstanceSuspensionStatusTimedOut
	case errors.As(err, &networkErr) && networkErr.Timeout():
		result.Status = d.ProcessInstanceSuspensionStatusTimedOut
		err = fmt.Errorf("%w: %w", d.ErrGatewayTimeout, err)
	}
	err = fmt.Errorf("suspend process instance: %w", err)
	result.Error = err.Error()
	return result, err
}

// validSuspensionHTTPResponse requires usable metadata before retaining a status
// or treating a response as authoritative submission evidence.
func validSuspensionHTTPResponse(response *http.Response) bool {
	return response != nil && response.StatusCode >= 100 && response.StatusCode <= 599 &&
		response.Request != nil && response.Request.URL != nil
}

// suspensionResponseError accepts only the documented empty 204, never a partial
// response with a call error or another success status.
func suspensionResponseError(response *http.Response, body []byte, callErr error) error {
	if callErr != nil {
		return callErr
	}
	if !validSuspensionHTTPResponse(response) {
		return fmt.Errorf("%w: invalid process-instance suspension HTTP response", d.ErrMalformedResponse)
	}
	if response.StatusCode == http.StatusNoContent && len(body) == 0 {
		return nil
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return fmt.Errorf("%w: unexpected process-instance suspension response (status %d)", d.ErrMalformedResponse, response.StatusCode)
	}
	return httpc.HttpStatusErr(response, body)
}

// suspensionRejected recognizes explicit mutation rejection. A generic server or
// proxy failure, including an HTML page mentioning throttling, remains ambiguous.
func suspensionRejected(response *http.Response, body []byte) bool {
	switch response.StatusCode {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden,
		http.StatusNotFound, http.StatusConflict, http.StatusPreconditionFailed,
		http.StatusUnsupportedMediaType, http.StatusUnprocessableEntity, http.StatusTooManyRequests:
		return true
	case http.StatusServiceUnavailable:
		var problem struct {
			Title  string `json:"title"`
			Detail string `json:"detail"`
		}
		if json.Unmarshal(body, &problem) != nil {
			return false
		}
		return strings.Contains(strings.ToUpper(problem.Title), "RESOURCE_EXHAUSTED") ||
			strings.Contains(strings.ToUpper(problem.Detail), "RESOURCE_EXHAUSTED")
	default:
		return false
	}
}
