// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/httpc"
)

// ValidateJobCompletionPayload rejects incompatible listener variables and
// non-JSON values before a bulk adapter can dispatch a completion call.
func ValidateJobCompletionPayload(job d.Job, variables map[string]any) error {
	if job.Kind == d.JobKindTaskListener && len(variables) > 0 {
		return fmt.Errorf("%w: task-listener completion cannot include nonempty variables", d.ErrValidation)
	}
	if _, err := json.Marshal(variables); err != nil {
		return fmt.Errorf("%w: invalid completion variables: %v", d.ErrValidation, err)
	}
	return nil
}

// ClassifyJobCompletionSubmission normalizes a single generated-client result
// for the supported adapters. It never retries or consults subsequent job reads.
// A call error takes precedence over HTTP status because incomplete response
// evidence cannot prove either rejection or acceptance.
func ClassifyJobCompletionSubmission(response *http.Response, body []byte, callErr error) (d.CompletionSubmission, error) {
	result := d.CompletionSubmission{Status: d.CompletionSubmissionUncertain}
	if response != nil && response.StatusCode > 0 {
		status := response.StatusCode
		result.HTTPStatus = &status
	}
	if callErr != nil {
		result.Diagnostic = callErr.Error()
		return result, callErr
	}
	if response == nil || response.StatusCode == 0 {
		err := fmt.Errorf("%w: missing completion HTTP response", d.ErrMalformedResponse)
		result.Diagnostic = err.Error()
		return result, err
	}

	err := httpc.HttpStatusErr(response, body)
	if err == nil {
		result.Status = d.CompletionSubmissionAccepted
		return result, nil
	}
	// Only explicit rejection statuses establish rejection. In particular, a
	// timeout, unexpected status, or generic server/proxy failure is ambiguous.
	switch response.StatusCode {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden,
		http.StatusNotFound, http.StatusConflict, http.StatusPreconditionFailed,
		http.StatusUnsupportedMediaType, http.StatusUnprocessableEntity,
		http.StatusTooManyRequests:
		result.Status = d.CompletionSubmissionRejected
	case http.StatusServiceUnavailable:
		if jobCompletionResourceExhausted(body) {
			result.Status = d.CompletionSubmissionRejected
		}
	}
	result.Diagnostic = err.Error()
	return result, err
}

// jobCompletionResourceExhausted requires a structured backend problem, rather
// than treating a generic 503 or a proxy page containing a marker as rejection.
func jobCompletionResourceExhausted(body []byte) bool {
	var problem struct {
		Title  string `json:"title"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(body, &problem); err != nil {
		return false
	}
	return strings.Contains(strings.ToUpper(problem.Title), "RESOURCE_EXHAUSTED") ||
		strings.Contains(strings.ToUpper(problem.Detail), "RESOURCE_EXHAUSTED")
}
