// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	camunda "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	adapter "github.com/grafvonb/c8volt/internal/services/usertask/v88"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// completionClient permits only completion; any other generated operation panics.
type completionClient struct {
	adapter.GenUserTaskClientCamunda
	submit func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error)
}

// CompleteUserTaskWithResponse exposes the generated request to each isolated fixture.
func (c completionClient) CompleteUserTaskWithResponse(ctx context.Context, key camunda.UserTaskKey, body camunda.CompleteUserTaskJSONRequestBody, _ ...camunda.RequestEditorFn) (*camunda.CompleteUserTaskResponse, error) {
	return c.submit(ctx, key, body)
}

// completionResponse includes metadata needed by shared HTTP classification and disables real retry delays.
func completionResponse(code int) *camunda.CompleteUserTaskResponse {
	response := newHTTPResponse(http.MethodPost, "https://distinctive.invalid/v2/user-tasks/2251799813685249/completion", code, http.StatusText(code))
	response.Header = http.Header{"Retry-After": []string{"0"}}
	return &camunda.CompleteUserTaskResponse{HTTPResponse: response, Body: []byte(`{"detail":"DISTINCTIVE_BODY RESOURCE_EXHAUSTED"}`)}
}

// completionService blocks every non-completion request, including any accidental legacy fallback.
func completionService(t *testing.T, log *slog.Logger, submit func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error)) *adapter.Service {
	t.Helper()
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	cfg := testConfig()
	cfg.App.Tenant = "discovery-tenant-must-not-filter-explicit-key"
	svc, err := adapter.New(cfg, &http.Client{Transport: completionForbiddenTransport{}}, log, adapter.WithClientCamunda(completionClient{submit: submit}))
	require.NoError(t, err)
	return svc
}

// completionForbiddenTransport fails closed if the adapter bypasses its generated completion mock.
type completionForbiddenTransport struct{}

// RoundTrip makes forbidden reads, legacy fallback, assignment and scope writes fail immediately.
func (completionForbiddenTransport) RoundTrip(*http.Request) (*http.Response, error) {
	panic("completion must not perform other HTTP operations")
}

// TestCompleteUserTaskWirePayload verifies the real generated route, nullable data and no-content acceptance.
func TestCompleteUserTaskWirePayload(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		variables map[string]any
		wire      string
	}{
		{"omitted", nil, `{"action":null,"variables":null}`},
		{"empty", map[string]any{}, `{"action":null,"variables":{}}`},
		{"nested", map[string]any{"object": map[string]any{"values": []any{true, nil, "text", json.Number("9007199254740993")}}}, `{"action":null,"variables":{"object":{"values":[true,null,"text",9007199254740993]}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/v2/user-tasks/2251799813685249/completion", r.URL.Path)
				require.Empty(t, r.URL.RawQuery)
				raw, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.JSONEq(t, tc.wire, string(raw))
				// Preserve large numbers byte-for-byte rather than relying on JSONEq's float conversion.
				if tc.name == "nested" {
					require.Contains(t, string(raw), "9007199254740993")
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			cfg := testConfig()
			cfg.APIs.Camunda.BaseURL = server.URL + "/v2"
			cfg.App.Tenant = "other-tenant"
			svc, err := adapter.New(cfg, server.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			got, err := svc.CompleteUserTask(context.Background(), "2251799813685249", tc.variables)
			require.NoError(t, err)
			require.Equal(t, d.UserTaskCompletionResult{Key: "2251799813685249", Status: d.UserTaskCompletionStatusAccepted, Submission: d.UserTaskCompletionSubmissionAccepted, StatusCode: 204}, got)
			require.Equal(t, 1, calls)
		})
	}
}

// TestCompleteUserTaskFailures preserves classified machine evidence without normal-level diagnostics or extra retries.
func TestCompleteUserTaskFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		code       int
		cause      error
		submission d.UserTaskCompletionSubmission
		status     d.UserTaskCompletionStatus
		attempts   int
	}{
		{400, d.ErrBadRequest, d.UserTaskCompletionSubmissionRejected, d.UserTaskCompletionStatusFailed, 1},
		{401, d.ErrUnauthorized, d.UserTaskCompletionSubmissionRejected, d.UserTaskCompletionStatusFailed, 1},
		{403, d.ErrForbidden, d.UserTaskCompletionSubmissionRejected, d.UserTaskCompletionStatusFailed, 1},
		{404, d.ErrNotFound, d.UserTaskCompletionSubmissionRejected, d.UserTaskCompletionStatusFailed, 1},
		{409, d.ErrConflict, d.UserTaskCompletionSubmissionRejected, d.UserTaskCompletionStatusFailed, 1},
		{429, d.ErrRateLimited, d.UserTaskCompletionSubmissionRejected, d.UserTaskCompletionStatusFailed, 8},
		{503, d.ErrUnavailable, d.UserTaskCompletionSubmissionRejected, d.UserTaskCompletionStatusFailed, 8},
		{500, d.ErrInternal, d.UserTaskCompletionSubmissionUnknown, d.UserTaskCompletionStatusFailed, 1},
		{504, d.ErrGatewayTimeout, d.UserTaskCompletionSubmissionUnknown, d.UserTaskCompletionStatusTimedOut, 1},
	} {
		t.Run(fmt.Sprint(tc.code), func(t *testing.T) {
			var logs bytes.Buffer
			calls := 0
			svc := completionService(t, slog.New(slog.NewTextHandler(&logs, nil)), func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error) {
				calls++
				return completionResponse(tc.code), nil
			})
			got, err := svc.CompleteUserTask(context.Background(), "2251799813685249", nil)
			require.ErrorIs(t, err, tc.cause)
			require.Equal(t, tc.attempts, calls)
			require.Equal(t, tc.status, got.Status)
			require.Equal(t, tc.submission, got.Submission)
			require.Equal(t, tc.code, got.StatusCode)
			require.Equal(t, err.Error(), got.Error)
			require.Contains(t, got.Error, "DISTINCTIVE_BODY")
			require.Contains(t, got.Error, "POST")
			require.Contains(t, got.Error, "https://distinctive.invalid/")
			require.Empty(t, logs.String())
		})
	}
}

// TestCompleteUserTaskMalformedAndTransport never interprets missing, unexpected or incomplete responses as acceptance.
func TestCompleteUserTaskMalformedAndTransport(t *testing.T) {
	t.Parallel()
	transportErr := errors.New("DISTINCTIVE_TRANSPORT")
	for _, tc := range []struct {
		name     string
		response *camunda.CompleteUserTaskResponse
		err      error
		cause    error
		code     int
	}{
		{"nil", nil, nil, d.ErrMalformedResponse, 0},
		{"no HTTP", &camunda.CompleteUserTaskResponse{}, nil, d.ErrMalformedResponse, 0},
		{"zero status", completionResponse(0), nil, d.ErrMalformedResponse, 0},
		{"unexpected 200", completionResponse(200), nil, d.ErrMalformedResponse, 200},
		{"unexpected 202", completionResponse(202), nil, d.ErrMalformedResponse, 202},
		{"transport", nil, transportErr, transportErr, 0},
		{"incomplete 204", completionResponse(204), transportErr, transportErr, 204},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			svc := completionService(t, nil, func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error) {
				calls++
				return tc.response, tc.err
			})
			got, err := svc.CompleteUserTask(context.Background(), "2251799813685249", nil)
			require.ErrorIs(t, err, tc.cause)
			require.Equal(t, 1, calls)
			require.Equal(t, d.UserTaskCompletionStatusFailed, got.Status)
			require.Equal(t, d.UserTaskCompletionSubmissionUnknown, got.Submission)
			require.Equal(t, tc.code, got.StatusCode)
		})
	}
}

// completionNetworkError exercises the retry helper's existing network policy without using network traffic.
type completionNetworkError struct{ timeout bool }

// Error includes sensitive markers that must never enter ordinary retry messages.
func (completionNetworkError) Error() string {
	return "POST https://distinctive.invalid/2251799813685249 DISTINCTIVE_BODY"
}

// Timeout distinguishes timeout classification from temporary connection failures.
func (e completionNetworkError) Timeout() bool { return e.timeout }

// Temporary permits the existing finite retry policy to handle the fixture.
func (completionNetworkError) Temporary() bool { return true }

// TestCompleteUserTaskRetryEvidence keeps earlier ambiguity until a later documented acceptance supersedes it.
func TestCompleteUserTaskRetryEvidence(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		first      int
		transport  error
		final      int
		submission d.UserTaskCompletionSubmission
	}{
		{"throttle then accepted", 429, nil, 204, d.UserTaskCompletionSubmissionAccepted},
		{"resource exhaustion then accepted", 503, nil, 204, d.UserTaskCompletionSubmissionAccepted},
		{"temporary then rejected", 0, completionNetworkError{}, 409, d.UserTaskCompletionSubmissionUnknown},
		{"timeout then accepted", 0, completionNetworkError{timeout: true}, 204, d.UserTaskCompletionSubmissionAccepted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			svc := completionService(t, nil, func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error) {
				calls++
				if calls == 1 {
					return completionResponse(tc.first), tc.transport
				}
				return completionResponse(tc.final), nil
			})
			got, err := svc.CompleteUserTask(context.Background(), "2251799813685249", nil)
			if tc.final == 204 {
				require.NoError(t, err)
				require.Equal(t, d.UserTaskCompletionStatusAccepted, got.Status)
			} else {
				require.ErrorIs(t, err, d.ErrConflict)
				require.Equal(t, d.UserTaskCompletionStatusFailed, got.Status)
			}
			require.Equal(t, 2, calls)
			require.Equal(t, tc.final, got.StatusCode)
			require.Equal(t, tc.submission, got.Submission)
		})
	}
}

// TestCompleteUserTaskCancellation retains submission evidence and the last status even when retry backoff returns no response.
func TestCompleteUserTaskCancellation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		before         bool
		response       int
		transport      bool
		timeout        bool
		wantStatus     d.UserTaskCompletionStatus
		wantSubmission d.UserTaskCompletionSubmission
	}{
		{"before request", true, 0, false, false, d.UserTaskCompletionStatusNotAttempted, d.UserTaskCompletionSubmissionNotSubmitted},
		{"during request", false, 0, true, false, d.UserTaskCompletionStatusCanceled, d.UserTaskCompletionSubmissionUnknown},
		{"after rejection", false, 429, false, false, d.UserTaskCompletionStatusCanceled, d.UserTaskCompletionSubmissionRejected},
		{"after ambiguous timeout", false, 503, true, true, d.UserTaskCompletionStatusCanceled, d.UserTaskCompletionSubmissionUnknown},
		{"accepted despite cancellation", false, 204, false, false, d.UserTaskCompletionStatusAccepted, d.UserTaskCompletionSubmissionAccepted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.before {
				cancel()
			}
			calls := 0
			svc := completionService(t, nil, func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error) {
				calls++
				cancel()
				if tc.timeout {
					return completionResponse(tc.response), context.DeadlineExceeded
				}
				if tc.transport {
					return nil, context.Canceled
				}
				return completionResponse(tc.response), nil
			})
			got, err := svc.CompleteUserTask(ctx, "2251799813685249", nil)
			if tc.response == 204 {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, context.Canceled)
			}
			require.Equal(t, tc.wantStatus, got.Status)
			require.Equal(t, tc.wantSubmission, got.Submission)
			require.Equal(t, tc.response, got.StatusCode)
			if tc.before {
				require.Zero(t, calls)
			} else {
				require.Equal(t, 1, calls)
			}
		})
	}
}

// TestCompleteUserTaskDeadlineBeforeSubmission performs no native request for an expired caller deadline.
func TestCompleteUserTaskDeadlineBeforeSubmission(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	svc := completionService(t, nil, func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error) {
		t.Fatal("must not submit")
		return nil, nil
	})
	got, err := svc.CompleteUserTask(ctx, "2251799813685249", nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, d.UserTaskCompletionStatusNotAttempted, got.Status)
	require.Equal(t, d.UserTaskCompletionSubmissionNotSubmitted, got.Submission)
}

// TestCompleteUserTaskSnapshotsPayload prevents generated-client mutation from changing the caller's nested data.
func TestCompleteUserTaskSnapshotsPayload(t *testing.T) {
	t.Parallel()
	variables := map[string]any{"nested": map[string]any{"value": "original"}}
	svc := completionService(t, nil, func(_ context.Context, key camunda.UserTaskKey, body camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error) {
		require.Equal(t, camunda.UserTaskKey("2251799813685249"), key)
		require.Nil(t, body.Action)
		(*body.Variables)["nested"].(map[string]any)["value"] = "mutated"
		return completionResponse(204), nil
	})
	_, err := svc.CompleteUserTask(context.Background(), "2251799813685249", variables)
	require.NoError(t, err)
	require.Equal(t, "original", variables["nested"].(map[string]any)["value"])
}

// TestCompleteUserTaskRetryLogging proves configured framing and safe verbose progress in a fresh process per log format.
func TestCompleteUserTaskRetryLogging(t *testing.T) {
	format := os.Getenv("C8VOLT_UNIT_COMPLETION_LOG_FORMAT")
	if format == "" {
		for _, format := range []string{"plain", "structured"} {
			t.Run(format, func(t *testing.T) {
				// Retry progress is globally throttled. Fresh test processes ensure both
				// handlers actually emit, without sleeping or changing production globals.
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCompleteUserTaskRetryLogging$", "-test.count=1")
				command.Env = append(os.Environ(), "C8VOLT_UNIT_COMPLETION_LOG_FORMAT="+format)
				output, err := command.CombinedOutput()
				require.NoError(t, err, string(output))
			})
		}
		return
	}
	for _, mode := range []string{"compact", "verbose", "suppressed"} {
		t.Run(mode, func(t *testing.T) {
			var logs bytes.Buffer
			var handler slog.Handler = slog.NewTextHandler(&logs, nil)
			if format == "structured" {
				handler = slog.NewJSONHandler(&logs, nil)
			}
			calls := 0
			svc := completionService(t, slog.New(handler), func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error) {
				calls++
				return completionResponse(429), nil
			})
			var opts []services.CallOption
			if mode != "compact" {
				opts = append(opts, services.WithVerbose())
			}
			if mode == "suppressed" {
				opts = append(opts, services.WithSuppressWorkflowDetailLogs())
			}
			_, err := svc.CompleteUserTask(context.Background(), "2251799813685249", nil, opts...)
			require.ErrorIs(t, err, d.ErrRateLimited)
			require.Equal(t, 8, calls)
			raw := logs.String()
			if mode != "verbose" {
				require.Empty(t, raw)
				return
			}
			require.Equal(t, 1, bytes.Count(logs.Bytes(), []byte("\n")))
			for _, secret := range []string{"POST", "distinctive.invalid", "DISTINCTIVE_BODY", "2251799813685249", "RESOURCE_EXHAUSTED"} {
				require.NotContains(t, raw, secret)
			}
			if format == "structured" {
				var record map[string]any
				decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
				require.NoError(t, decoder.Decode(&record))
				require.Equal(t, "INFO", record["level"])
				require.Contains(t, record["msg"], "Camunda throttled complete user task; rate limited; retrying in")
				require.ErrorIs(t, decoder.Decode(&record), io.EOF)
			} else {
				require.Contains(t, raw, `level=INFO msg="Camunda throttled complete user task; rate limited; retrying in`)
			}
		})
	}
}

// TestCompleteUserTaskInvalidPayload prevents unsupported values from reaching native submission.
func TestCompleteUserTaskInvalidPayload(t *testing.T) {
	t.Parallel()
	svc := completionService(t, nil, func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error) {
		t.Fatal("must not submit")
		return nil, nil
	})
	got, err := svc.CompleteUserTask(context.Background(), "2251799813685249", map[string]any{"invalid": make(chan int)})
	require.ErrorIs(t, err, d.ErrValidation)
	require.Equal(t, d.UserTaskCompletionStatusNotAttempted, got.Status)
	require.Equal(t, d.UserTaskCompletionSubmissionNotSubmitted, got.Submission)
}

// TestCompleteUserTaskTimeoutAndExhaustion retains uncertainty and classifies exhausted temporary network errors accurately.
func TestCompleteUserTaskTimeoutAndExhaustion(t *testing.T) {
	t.Parallel()
	for _, timeout := range []bool{false, true} {
		t.Run(fmt.Sprint(timeout), func(t *testing.T) {
			calls := 0
			networkErr := completionNetworkError{timeout: timeout}
			svc := completionService(t, nil, func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error) {
				calls++
				return completionResponse(0), networkErr
			})
			got, err := svc.CompleteUserTask(context.Background(), "2251799813685249", nil)
			require.ErrorIs(t, err, networkErr)
			require.Equal(t, 8, calls)
			require.Equal(t, d.UserTaskCompletionSubmissionUnknown, got.Submission)
			if timeout {
				require.Equal(t, d.UserTaskCompletionStatusTimedOut, got.Status)
				require.ErrorIs(t, err, d.ErrGatewayTimeout)
			} else {
				require.Equal(t, d.UserTaskCompletionStatusFailed, got.Status)
			}
		})
	}
}

// TestCompleteUserTaskBackoffEvidence cancels after a response exists, before retry waiting can submit another request.
func TestCompleteUserTaskBackoffEvidence(t *testing.T) {
	t.Parallel()
	for _, ambiguous := range []bool{false, true} {
		t.Run(fmt.Sprint(ambiguous), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			svc := completionService(t, nil, func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error) {
				calls++
				if ambiguous && calls == 1 {
					return completionResponse(0), completionNetworkError{}
				}
				// Cancellation occurs after the backend response exists, so backoff must retain it.
				cancel()
				response := completionResponse(429)
				response.HTTPResponse.Header.Set("Retry-After", "3600")
				return response, nil
			})
			got, err := svc.CompleteUserTask(ctx, "2251799813685249", nil)
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, d.UserTaskCompletionStatusCanceled, got.Status)
			require.Equal(t, 429, got.StatusCode)
			if ambiguous {
				require.Equal(t, 2, calls)
				require.Equal(t, d.UserTaskCompletionSubmissionUnknown, got.Submission)
			} else {
				require.Equal(t, 1, calls)
				require.Equal(t, d.UserTaskCompletionSubmissionRejected, got.Submission)
			}
		})
	}
}

// TestCompleteUserTaskDeadlineDuringBackoff retains explicit rejection when the caller's deadline interrupts retry waiting.
func TestCompleteUserTaskDeadlineDuringBackoff(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	calls := 0
	svc := completionService(t, nil, func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error) {
		calls++
		response := completionResponse(429)
		response.HTTPResponse.Header.Set("Retry-After", "3600")
		return response, nil
	})
	got, err := svc.CompleteUserTask(ctx, "2251799813685249", nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, 1, calls)
	require.Equal(t, d.UserTaskCompletionStatusTimedOut, got.Status)
	require.Equal(t, d.UserTaskCompletionSubmissionRejected, got.Submission)
	require.Equal(t, 429, got.StatusCode)
}

// TestCompleteUserTaskLastStatusSurvivesTransportCancellation keeps the most recent actual HTTP evidence without treating it as final rejection.
func TestCompleteUserTaskLastStatusSurvivesTransportCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	svc := completionService(t, nil, func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error) {
		calls++
		if calls == 1 {
			return completionResponse(429), nil
		}
		cancel()
		return nil, context.Canceled
	})
	got, err := svc.CompleteUserTask(ctx, "2251799813685249", nil)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 2, calls)
	require.Equal(t, d.UserTaskCompletionStatusCanceled, got.Status)
	require.Equal(t, d.UserTaskCompletionSubmissionUnknown, got.Submission)
	require.Equal(t, 429, got.StatusCode)
}

// TestCompleteUserTaskMalformedMetadataAndGeneric503 never accepts incomplete metadata or treats a proxy error as definitive rejection.
func TestCompleteUserTaskMalformedMetadataAndGeneric503(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"missing request", "missing URL", "generic 503", "proxy resource marker"} {
		t.Run(name, func(t *testing.T) {
			response := completionResponse(204)
			switch name {
			case "missing request":
				response.HTTPResponse.Request = nil
			case "missing URL":
				response.HTTPResponse.Request.URL = nil
			case "generic 503":
				response = completionResponse(503)
				response.Body = []byte(`{"detail":"proxy unavailable"}`)
			case "proxy resource marker":
				response = completionResponse(503)
				response.Body = []byte("proxy RESOURCE_EXHAUSTED page")
			}
			calls := 0
			svc := completionService(t, nil, func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error) {
				calls++
				return response, nil
			})
			got, err := svc.CompleteUserTask(context.Background(), "2251799813685249", nil)
			if response.HTTPResponse.StatusCode == 204 {
				require.ErrorIs(t, err, d.ErrMalformedResponse)
			} else {
				require.ErrorIs(t, err, d.ErrUnavailable)
			}
			if name == "proxy resource marker" {
				require.Equal(t, 8, calls)
			} else {
				require.Equal(t, 1, calls)
			}
			require.Equal(t, d.UserTaskCompletionStatusFailed, got.Status)
			require.Equal(t, d.UserTaskCompletionSubmissionUnknown, got.Submission)
		})
	}
}
