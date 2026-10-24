// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	camunda "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	adapter "github.com/grafvonb/c8volt/internal/services/usertask/v810"
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

// TestCompleteUserTaskDelegation preserves the adapter's response/body/error forwarding.
func TestCompleteUserTaskDelegation(t *testing.T) {
	t.Parallel()
	transportErr := errors.New("adapter transport failure")
	for _, tc := range []struct {
		name             string
		response         *camunda.CompleteUserTaskResponse
		callErr, wantErr error
		submission       d.UserTaskCompletionSubmission
	}{
		{"rejected", completionResponse(409), nil, d.ErrConflict, d.UserTaskCompletionSubmissionRejected},
		{"missing response", nil, nil, d.ErrMalformedResponse, d.UserTaskCompletionSubmissionUnknown},
		{"partial response with error", completionResponse(204), transportErr, transportErr, d.UserTaskCompletionSubmissionUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			svc := completionService(t, nil, func(context.Context, camunda.UserTaskKey, camunda.CompleteUserTaskJSONRequestBody) (*camunda.CompleteUserTaskResponse, error) {
				calls++
				return tc.response, tc.callErr
			})
			got, err := svc.CompleteUserTask(context.Background(), "2251799813685249", nil)
			require.ErrorIs(t, err, tc.wantErr)
			require.Equal(t, 1, calls)
			require.Equal(t, d.UserTaskCompletionStatusFailed, got.Status)
			require.Equal(t, tc.submission, got.Submission)
			require.Equal(t, err.Error(), got.Error)
			if tc.response != nil {
				require.Equal(t, tc.response.HTTPResponse.StatusCode, got.StatusCode)
			}
			if tc.name == "rejected" {
				require.Contains(t, got.Error, "DISTINCTIVE_BODY")
			}
		})
	}
}
