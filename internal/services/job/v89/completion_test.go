// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	camunda "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestBulkCompletionSubmissionEvidence keeps explicit rejection separate from uncertain dispatch without retrying either.
func TestBulkCompletionSubmissionEvidence(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		status  int
		body    string
		callErr error
		want    d.CompletionSubmissionStatus
		wantErr error
	}{
		{name: "accepted", status: 204, want: d.CompletionSubmissionAccepted},
		{name: "accepted async", status: 202, want: d.CompletionSubmissionAccepted},
		{name: "bad request", status: 400, want: d.CompletionSubmissionRejected, wantErr: d.ErrBadRequest},
		{name: "unauthorized", status: 401, want: d.CompletionSubmissionRejected, wantErr: d.ErrUnauthorized},
		{name: "forbidden", status: 403, want: d.CompletionSubmissionRejected, wantErr: d.ErrForbidden},
		{name: "missing", status: 404, want: d.CompletionSubmissionRejected, wantErr: d.ErrNotFound},
		{name: "conflict", status: 409, want: d.CompletionSubmissionRejected, wantErr: d.ErrConflict},
		{name: "precondition", status: 412, want: d.CompletionSubmissionRejected, wantErr: d.ErrPrecondition},
		{name: "media type", status: 415, want: d.CompletionSubmissionRejected, wantErr: d.ErrUnsupported},
		{name: "validation", status: 422, want: d.CompletionSubmissionRejected, wantErr: d.ErrValidation},
		{name: "rate limit", status: 429, want: d.CompletionSubmissionRejected, wantErr: d.ErrRateLimited},
		{name: "backpressure", status: 503, body: `{"title":"RESOURCE_EXHAUSTED"}`, want: d.CompletionSubmissionRejected, wantErr: d.ErrUnavailable},
		{name: "backpressure detail", status: 503, body: `{"detail":"Command rejected with code RESOURCE_EXHAUSTED"}`, want: d.CompletionSubmissionRejected, wantErr: d.ErrUnavailable},
		{name: "unavailable", status: 503, want: d.CompletionSubmissionUncertain, wantErr: d.ErrUnavailable},
		{name: "proxy with marker", status: 503, body: `<html>RESOURCE_EXHAUSTED</html>`, want: d.CompletionSubmissionUncertain, wantErr: d.ErrUnavailable},
		{name: "internal", status: 500, want: d.CompletionSubmissionUncertain, wantErr: d.ErrInternal},
		{name: "gateway", status: 502, want: d.CompletionSubmissionUncertain, wantErr: d.ErrUpstream},
		{name: "gateway timeout", status: 504, want: d.CompletionSubmissionUncertain, wantErr: d.ErrGatewayTimeout},
		{name: "request timeout", status: 408, want: d.CompletionSubmissionUncertain, wantErr: d.ErrUpstream},
		{name: "redirect", status: 307, want: d.CompletionSubmissionUncertain, wantErr: d.ErrUpstream},
		{name: "unexpected success", status: 206, want: d.CompletionSubmissionUncertain, wantErr: d.ErrUpstream},
		{name: "nil response", want: d.CompletionSubmissionUncertain, wantErr: d.ErrMalformedResponse},
		{name: "timeout", callErr: context.DeadlineExceeded, want: d.CompletionSubmissionUncertain, wantErr: context.DeadlineExceeded},
		{name: "canceled during call", callErr: context.Canceled, want: d.CompletionSubmissionUncertain, wantErr: context.Canceled},
		{name: "connection lost", callErr: io.ErrUnexpectedEOF, want: d.CompletionSubmissionUncertain, wantErr: io.ErrUnexpectedEOF},
		{name: "response and read error", status: 404, callErr: io.ErrUnexpectedEOF, want: d.CompletionSubmissionUncertain, wantErr: io.ErrUnexpectedEOF},
		{name: "success and read error", status: 204, callErr: io.ErrUnexpectedEOF, want: d.CompletionSubmissionUncertain, wantErr: io.ErrUnexpectedEOF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			svc := newJobServiceTest(t, &mockJobClient{
				completeJobWithResponse: func(_ context.Context, key camunda.JobKey, body camunda.CompleteJobJSONRequestBody, _ ...camunda.RequestEditorFn) (*camunda.CompleteJobResponse, error) {
					calls++
					require.Equal(t, camunda.JobKey("123"), key)
					require.NotNil(t, body.Variables)
					require.Equal(t, map[string]any{"approved": true}, *body.Variables)
					if tt.status == 0 {
						return nil, tt.callErr
					}
					response := okJobCompleteHTTPResponse()
					response.StatusCode = tt.status
					return &camunda.CompleteJobResponse{HTTPResponse: response, Body: []byte(tt.body)}, tt.callErr
				},
			})
			got, err := svc.submitBulkCompletion(context.Background(), d.Job{Key: "123", Kind: "BPMN_ELEMENT"}, map[string]any{"approved": true})
			require.Equal(t, 1, calls)
			require.Equal(t, tt.want, got.Status)
			if tt.status == 0 {
				require.Nil(t, got.HTTPStatus)
			} else {
				require.Equal(t, &tt.status, got.HTTPStatus)
			}
			if tt.wantErr == nil {
				require.NoError(t, err)
				require.Empty(t, got.Diagnostic)
			} else {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, err.Error(), got.Diagnostic)
			}
		})
	}
}

// TestBulkCompletionPreflightAndListenerPayload proves local failures are not submissions and allowed listener payloads remain empty objects.
func TestBulkCompletionPreflightAndListenerPayload(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		kind      string
		variables map[string]any
		canceled  bool
		wantErr   error
	}{
		{name: "listener without variables", kind: d.JobKindTaskListener},
		{name: "listener empty variables", kind: d.JobKindTaskListener, variables: map[string]any{}},
		{name: "listener nonempty variables", kind: d.JobKindTaskListener, variables: map[string]any{"approved": false}, wantErr: d.ErrValidation},
		{name: "invalid JSON variables", variables: map[string]any{"invalid": make(chan int)}, wantErr: d.ErrValidation},
		{name: "already canceled", canceled: true, wantErr: context.Canceled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			svc := newJobServiceTest(t, &mockJobClient{
				completeJobWithResponse: func(_ context.Context, _ camunda.JobKey, body camunda.CompleteJobJSONRequestBody, _ ...camunda.RequestEditorFn) (*camunda.CompleteJobResponse, error) {
					calls++
					require.NotNil(t, body.Variables)
					require.Empty(t, *body.Variables)
					return &camunda.CompleteJobResponse{HTTPResponse: okJobCompleteHTTPResponse()}, nil
				},
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tt.canceled {
				cancel()
			}
			got, err := svc.submitBulkCompletion(ctx, d.Job{Key: "123", Kind: tt.kind}, tt.variables)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, d.CompletionSubmissionNotSubmitted, got.Status)
				require.Nil(t, got.HTTPStatus)
				require.Equal(t, err.Error(), got.Diagnostic)
				require.Zero(t, calls)
			} else {
				require.NoError(t, err)
				require.Equal(t, d.CompletionSubmissionAccepted, got.Status)
				require.Equal(t, 1, calls)
			}
		})
	}
}

// TestBulkCompletionHTTP proves the generated client submits once for success, rejection, proxy failure, and undecodable responses.
func TestBulkCompletionHTTP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		status int
		body   string
		want   d.CompletionSubmissionStatus
		parsed bool
	}{
		{"accepted", 204, "", d.CompletionSubmissionAccepted, true},
		{"missing", 404, `{"detail":"job not found"}`, d.CompletionSubmissionRejected, true},
		{"conflict", 409, `{"detail":"job state conflict"}`, d.CompletionSubmissionRejected, true},
		{"rate limit", 429, `{"detail":"rate limited"}`, d.CompletionSubmissionRejected, true},
		{"backpressure", 503, `{"title":"RESOURCE_EXHAUSTED"}`, d.CompletionSubmissionRejected, true},
		{"proxy failure", 502, `{"detail":"gateway failed"}`, d.CompletionSubmissionUncertain, true},
		{"malformed rejection", 404, `{`, d.CompletionSubmissionUncertain, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != "/jobs/123/completion" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				payload, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if string(payload) != `{"variables":{}}` {
					t.Errorf("unexpected payload: %s", payload)
				}
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()
			cfg := testx.TestConfig(t)
			cfg.APIs.Camunda.BaseURL = server.URL
			svc, err := New(cfg, server.Client(), nil)
			require.NoError(t, err)
			got, err := svc.submitBulkCompletion(context.Background(), d.Job{Key: "123", Kind: d.JobKindTaskListener}, nil)
			require.Equal(t, int32(1), calls.Load())
			require.Equal(t, tt.want, got.Status)
			if tt.parsed {
				require.Equal(t, &tt.status, got.HTTPStatus)
			} else {
				require.Nil(t, got.HTTPStatus)
			}
			if tt.want == d.CompletionSubmissionAccepted {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.NotEmpty(t, got.Diagnostic)
			}
		})
	}
}

// TestBulkCompletionHTTPTimeout proves a server-observed completion is not replayed or called rejected after a deadline.
func TestBulkCompletionHTTPTimeout(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(started)
		}
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	defer close(release)
	cfg := testx.TestConfig(t)
	cfg.APIs.Camunda.BaseURL = server.URL
	svc, err := New(cfg, server.Client(), nil)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := svc.submitBulkCompletion(ctx, d.Job{Key: "123"}, nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	select {
	case <-started:
	default:
		t.Fatal("server never observed the submission")
	}
	require.Equal(t, int32(1), calls.Load())
	require.Equal(t, d.CompletionSubmissionUncertain, got.Status)
	require.Nil(t, got.HTTPStatus)
	require.NotEmpty(t, got.Diagnostic)
}

// TestSubmitJobWorkerOutcomeLegacyCompletionRetry preserves automatic backpressure retry on the scalar path only.
func TestSubmitJobWorkerOutcomeLegacyCompletionRetry(t *testing.T) {
	t.Parallel()
	calls := 0
	svc := newJobServiceTest(t, &mockJobClient{
		completeJobWithResponse: func(_ context.Context, _ camunda.JobKey, _ camunda.CompleteJobJSONRequestBody, _ ...camunda.RequestEditorFn) (*camunda.CompleteJobResponse, error) {
			calls++
			response := okJobCompleteHTTPResponse()
			response.Header = http.Header{"Retry-After": []string{"0"}}
			if calls == 1 {
				response.StatusCode = http.StatusTooManyRequests
			}
			return &camunda.CompleteJobResponse{HTTPResponse: response}, nil
		},
	})
	got, err := svc.SubmitJobWorkerOutcome(context.Background(), d.JobWorkerOutcomeRequest{Key: "123", Mode: d.JobWorkerOutcomeCompletion})
	require.NoError(t, err)
	require.True(t, got.MutationAccepted)
	require.Equal(t, "skipped", got.ConfirmationStatus)
	require.Equal(t, 2, calls)
}

// TestBulkCompletionMissingHTTPResponse keeps malformed generated results uncertain instead of panicking.
func TestBulkCompletionMissingHTTPResponse(t *testing.T) {
	t.Parallel()
	svc := newJobServiceTest(t, &mockJobClient{
		completeJobWithResponse: func(context.Context, camunda.JobKey, camunda.CompleteJobJSONRequestBody, ...camunda.RequestEditorFn) (*camunda.CompleteJobResponse, error) {
			return &camunda.CompleteJobResponse{}, nil
		},
	})
	got, err := svc.submitBulkCompletion(context.Background(), d.Job{Key: "123"}, nil)
	require.True(t, errors.Is(err, d.ErrMalformedResponse))
	require.Equal(t, d.CompletionSubmissionUncertain, got.Status)
	require.Nil(t, got.HTTPStatus)
}
