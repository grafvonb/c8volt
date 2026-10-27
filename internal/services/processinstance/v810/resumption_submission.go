// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/httpc"
)

// submitProcessInstanceResumption submits only the supplied key and never reads
// state. Evidence lives outside the retry return value so backoff cancellation
// cannot erase an earlier response. Functional retry chatter is suppressed; the
// configured HTTP transport retains its independent DEBUG diagnostics.
func (s *Service) submitProcessInstanceResumption(ctx context.Context, key string) (d.ProcessInstanceResumptionResult, error) {
	result := d.ProcessInstanceResumptionResult{
		Key: key, Status: d.ProcessInstanceResumptionStatusNotAttempted,
		Submission:   d.ProcessInstanceResumptionSubmissionNotSubmitted,
		Confirmation: d.ProcessInstanceResumptionConfirmationNotRequested,
	}
	var lastErr error
	_, retryErr := services.RetryCamundaMutation(ctx, nil, "resume process instance", func(ctx context.Context) (struct{}, *http.Response, []byte, error) {
		if err := ctx.Err(); err != nil {
			return struct{}{}, nil, nil, err
		}
		attemptCtx, observation := httpc.BeginRequestRateAttempt(ctx)
		resp, callErr := s.cc.ResumeProcessInstanceWithResponse(attemptCtx, camundav810.ProcessInstanceKey(key), camundav810.ResumeProcessInstanceJSONRequestBody{})
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
		validHTTP := validResumptionHTTPResponse(response)
		if validHTTP {
			result.StatusCode = response.StatusCode
		}
		lastErr = resumptionResponseError(response, body, callErr)
		if lastErr == nil {
			result.Status = d.ProcessInstanceResumptionStatusAccepted
			result.Submission = d.ProcessInstanceResumptionSubmissionAccepted
		} else if callErr == nil && validHTTP && resumptionRejected(response, body) {
			// A later rejection cannot prove that an earlier ambiguous attempt failed.
			if result.Submission != d.ProcessInstanceResumptionSubmissionUnknown {
				result.Submission = d.ProcessInstanceResumptionSubmissionRejected
			}
		} else {
			result.Submission = d.ProcessInstanceResumptionSubmissionUnknown
		}
		// Preserve the raw call error and response for the existing retry policy;
		// converting 429/503 into domain errors here would suppress their retries.
		return struct{}{}, response, body, callErr
	})
	if result.Submission == d.ProcessInstanceResumptionSubmissionAccepted {
		return result, nil
	}
	err := lastErr
	if retryErr != nil {
		err = retryErr
	}
	return resumptionFailure(result, err)
}

// validResumptionHTTPResponse requires usable metadata before retaining a status
// or treating a response as authoritative submission evidence.
func validResumptionHTTPResponse(response *http.Response) bool {
	return response != nil && response.StatusCode >= 100 && response.StatusCode <= 599 &&
		response.Request != nil && response.Request.URL != nil
}

// resumptionResponseError accepts only the documented empty 204, never a partial
// response with a call error or another success status.
func resumptionResponseError(response *http.Response, body []byte, callErr error) error {
	if callErr != nil {
		return callErr
	}
	if !validResumptionHTTPResponse(response) {
		return fmt.Errorf("%w: invalid process-instance resumption HTTP response", d.ErrMalformedResponse)
	}
	if response.StatusCode == http.StatusNoContent && len(body) == 0 {
		return nil
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return fmt.Errorf("%w: unexpected process-instance resumption response (status %d)", d.ErrMalformedResponse, response.StatusCode)
	}
	return httpc.HttpStatusErr(response, body)
}

// resumptionRejected recognizes explicit mutation rejection. A generic server or
// proxy failure, including an HTML page mentioning throttling, remains ambiguous.
func resumptionRejected(response *http.Response, body []byte) bool {
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
