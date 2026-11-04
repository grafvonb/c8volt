// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"encoding/json"
	"mime"
	"net/http"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/httpc"
)

// RecordScopeVariableSubmission accumulates evidence outside the retry callback's
// return value, which can be lost on backoff cancellation. localErr is the fresh
// request-rate observation, including errors replaced by http.Client.Timeout.
// It describes only this attempt, never earlier dispatches. Error normalization
// and the existing PI/UT acceptance mapping remain with the caller.
func RecordScopeVariableSubmission(result *d.ScopeVariableUpdateResponse, response *http.Response, body []byte, callErr, localErr error) {
	if result.SubmissionStatus == "" {
		result.SubmissionStatus = d.ScopeVariableSubmissionNotSent
	}
	if localErr != nil || httpc.IsRequestRateWaitError(callErr) {
		return
	}
	if response != nil && response.StatusCode > 0 {
		result.StatusCode = response.StatusCode
		result.Status = response.Status
		if result.Status == "" {
			result.Status = http.StatusText(response.StatusCode)
		}
	}
	// Acceptance is final. No subsequent observation (including confirmation)
	// can undo proof of a successful mutation.
	if result.SubmissionStatus == d.ScopeVariableSubmissionAccepted {
		return
	}
	validHTTP := response != nil && response.Request != nil && response.Request.URL != nil
	if callErr == nil && validHTTP && response.StatusCode == http.StatusNoContent && len(body) == 0 {
		result.SubmissionStatus = d.ScopeVariableSubmissionAccepted
		result.Accepted = true
		return
	}
	if callErr == nil && validHTTP && scopeVariableInvalidData(response, body) {
		if result.SubmissionStatus != d.ScopeVariableSubmissionUnknown {
			result.SubmissionStatus = d.ScopeVariableSubmissionRejected
		}
		return
	}
	result.SubmissionStatus = d.ScopeVariableSubmissionUnknown
}

// scopeVariableInvalidData accepts only the endpoint's documented 400 problem
// detail schema. Generated decoding alone accepts missing/null required fields;
// server, proxy and malformed failures must retain uncertainty instead.
func scopeVariableInvalidData(response *http.Response, body []byte) bool {
	if response.StatusCode != http.StatusBadRequest {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/problem+json" {
		return false
	}
	var problem struct {
		Type     *string `json:"type"`
		Title    *string `json:"title"`
		Status   *int32  `json:"status"`
		Detail   *string `json:"detail"`
		Instance *string `json:"instance"`
	}
	return json.Unmarshal(body, &problem) == nil && problem.Type != nil && problem.Title != nil &&
		problem.Status != nil && *problem.Status == http.StatusBadRequest && problem.Detail != nil && problem.Instance != nil
}
