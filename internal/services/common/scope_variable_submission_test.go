// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package common

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

const scopeInvalidData = `{"type":"about:blank","title":"Bad Request","status":400,"detail":"invalid variables","instance":"/v2/element-instances/456/variables"}`

// scopeSubmissionResponse provides real HTTP metadata without a network dependency.
func scopeSubmissionResponse(code int) *http.Response {
	return &http.Response{StatusCode: code, Status: http.StatusText(code), Header: http.Header{"Content-Type": []string{"application/problem+json"}}, Request: &http.Request{Method: http.MethodPut, URL: &url.URL{Scheme: "https", Host: "fixture.invalid", Path: "/v2/element-instances/456/variables"}}}
}

// TestScopeVariableSubmissionClassification requires documented, well-formed evidence;
// arbitrary HTTP errors and response-plus-error never prove rejection or acceptance.
func TestScopeVariableSubmissionClassification(t *testing.T) {
	t.Parallel()
	transportErr := errors.New("response interrupted")
	type submissionCase struct {
		name              string
		response          *http.Response
		body              string
		callErr, localErr error
		want              d.ScopeVariableSubmissionStatus
	}
	tests := []submissionCase{
		{name: "accepted", response: scopeSubmissionResponse(204), want: d.ScopeVariableSubmissionAccepted},
		{name: "rejected", response: scopeSubmissionResponse(400), body: scopeInvalidData, want: d.ScopeVariableSubmissionRejected},
		{name: "missing response", want: d.ScopeVariableSubmissionUnknown},
		{name: "transport", callErr: transportErr, want: d.ScopeVariableSubmissionUnknown},
		{name: "generic cancellation", callErr: context.Canceled, want: d.ScopeVariableSubmissionUnknown},
		{name: "local admission", callErr: context.DeadlineExceeded, localErr: context.DeadlineExceeded, want: d.ScopeVariableSubmissionNotSent},
		{name: "response and error", response: scopeSubmissionResponse(204), callErr: transportErr, want: d.ScopeVariableSubmissionUnknown},
		{name: "rejection and error", response: scopeSubmissionResponse(400), body: scopeInvalidData, callErr: transportErr, want: d.ScopeVariableSubmissionUnknown},
		{name: "malformed rejection", response: scopeSubmissionResponse(400), body: `{`, want: d.ScopeVariableSubmissionUnknown},
		{name: "missing fields", response: scopeSubmissionResponse(400), body: `{"status":400}`, want: d.ScopeVariableSubmissionUnknown},
		{name: "null fields", response: scopeSubmissionResponse(400), body: `{"type":null,"title":null,"status":400,"detail":null,"instance":null}`, want: d.ScopeVariableSubmissionUnknown},
		{name: "wrong status", response: scopeSubmissionResponse(400), body: `{"type":"about:blank","title":"Bad Request","status":500,"detail":"bad","instance":"/"}`, want: d.ScopeVariableSubmissionUnknown},
		{name: "unexpected success", response: scopeSubmissionResponse(200), want: d.ScopeVariableSubmissionUnknown},
		{name: "unexpected response body", response: scopeSubmissionResponse(204), body: `{}`, want: d.ScopeVariableSubmissionUnknown},
		{name: "missing request", response: &http.Response{StatusCode: 204}, want: d.ScopeVariableSubmissionUnknown},
	}
	for _, code := range []int{401, 403, 404, 409, 429, 500, 502, 503, 504} {
		tests = append(tests, submissionCase{name: http.StatusText(code), response: scopeSubmissionResponse(code), body: scopeInvalidData, want: d.ScopeVariableSubmissionUnknown})
	}
	proxy := scopeSubmissionResponse(400)
	proxy.Header.Set("Content-Type", "text/html")
	tests = append(tests, submissionCase{name: "proxy content", response: proxy, body: scopeInvalidData, want: d.ScopeVariableSubmissionUnknown})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.ScopeVariableUpdateResponse{ScopeKey: "456", SubmissionStatus: d.ScopeVariableSubmissionNotSent}
			RecordScopeVariableSubmission(&result, tt.response, []byte(tt.body), tt.callErr, tt.localErr)
			require.Equal(t, tt.want, result.SubmissionStatus)
			require.Equal(t, tt.want == d.ScopeVariableSubmissionAccepted, result.Accepted)
			require.Equal(t, "456", result.ScopeKey)
		})
	}
}

// TestScopeVariableSubmissionSequences covers artificial sequences without changing
// retry policy: a 400 is not actually retried, but its evidence must remain sticky.
func TestScopeVariableSubmissionSequences(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		attempts []string
		want     d.ScopeVariableSubmissionStatus
		code     int
	}{
		{"unsent", []string{"unsent"}, d.ScopeVariableSubmissionNotSent, 0},
		{"ambiguous then unsent", []string{"unknown", "unsent"}, d.ScopeVariableSubmissionUnknown, 503},
		{"rejected then unsent", []string{"rejected", "unsent"}, d.ScopeVariableSubmissionRejected, 400},
		{"ambiguous then rejected", []string{"unknown", "rejected"}, d.ScopeVariableSubmissionUnknown, 400},
		{"rejected then ambiguous", []string{"rejected", "unknown"}, d.ScopeVariableSubmissionUnknown, 503},
		{"accepted after ambiguity", []string{"unknown", "accepted"}, d.ScopeVariableSubmissionAccepted, 204},
		{"accepted survives", []string{"accepted", "unsent"}, d.ScopeVariableSubmissionAccepted, 204},
	} {
		t.Run(tt.name, func(t *testing.T) {
			result := d.ScopeVariableUpdateResponse{ScopeKey: "456", SubmissionStatus: d.ScopeVariableSubmissionNotSent}
			for _, attempt := range tt.attempts {
				switch attempt {
				case "unsent":
					RecordScopeVariableSubmission(&result, nil, nil, context.DeadlineExceeded, context.DeadlineExceeded)
				case "unknown":
					RecordScopeVariableSubmission(&result, scopeSubmissionResponse(503), []byte(`RESOURCE_EXHAUSTED`), nil, nil)
				case "rejected":
					RecordScopeVariableSubmission(&result, scopeSubmissionResponse(400), []byte(scopeInvalidData), nil, nil)
				case "accepted":
					RecordScopeVariableSubmission(&result, scopeSubmissionResponse(204), nil, nil, nil)
				}
			}
			require.Equal(t, tt.want, result.SubmissionStatus)
			require.Equal(t, tt.code, result.StatusCode)
			require.Equal(t, tt.want == d.ScopeVariableSubmissionAccepted, result.Accepted)
		})
	}
}
