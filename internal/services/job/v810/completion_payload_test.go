// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	camunda "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestCompletionRequestPayload exercises both service entry points against an
// isolated HTTP server that rejects explicit null optional fields. Backend and
// parsing failures must remain failures after request serialization is fixed.
func TestCompletionRequestPayload(t *testing.T) {
	for _, bulk := range []bool{false, true} {
		for _, tc := range []struct {
			name      string
			variables map[string]any
			status    int
			response  string
			wantErr   error
			malformed bool
		}{
			{name: "nil variables", status: 204},
			{name: "empty variables", variables: map[string]any{}, status: 204},
			{name: "nested null variables", variables: map[string]any{"leaseToken": nil, "businessId": nil, "approved": true, "nested": map[string]any{"unset": nil}}, status: 204},
			{name: "backend rejection", status: 400, response: `{"title":"Bad Request","detail":"a lease token is required for this job"}`, wantErr: d.ErrBadRequest},
			{name: "authorization rejection", status: 401, response: `{"title":"Unauthorized","detail":"completion is not authorized"}`, wantErr: d.ErrUnauthorized},
			{name: "server error", status: 500, response: `{"title":"Internal Server Error","detail":"backend failure"}`, wantErr: d.ErrInternal},
			{name: "malformed rejection", status: 400, response: `{`, malformed: true},
		} {
			mode := "scalar"
			if bulk {
				mode = "bulk"
			}
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				var calls atomic.Int32
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					require.Equal(t, http.MethodPost, r.Method)
					require.Equal(t, "/jobs/123/completion", r.URL.Path)
					require.Equal(t, "application/json", r.Header.Get("Content-Type"))
					raw, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					var body map[string]json.RawMessage
					require.NoError(t, json.Unmarshal(raw, &body))
					w.Header().Set("Content-Type", "application/problem+json")
					for _, field := range []string{"leaseToken", "businessId"} {
						if _, exists := body[field]; exists {
							w.WriteHeader(400)
							io.WriteString(w, `{"title":"Bad Request","detail":"optional field must be omitted, not null"}`)
							return
						}
					}
					require.Len(t, body, 1)
					want := tc.variables
					if want == nil {
						want = map[string]any{}
					}
					expected, err := json.Marshal(want)
					require.NoError(t, err)
					require.JSONEq(t, string(expected), string(body["variables"]))
					w.WriteHeader(tc.status)
					io.WriteString(w, tc.response)
				}))
				t.Cleanup(server.Close)
				cfg := testx.TestConfig(t)
				cfg.APIs.Camunda.BaseURL = server.URL
				svc, err := New(cfg, server.Client(), nil)
				require.NoError(t, err)
				var diagnostic string
				if bulk {
					got, callErr := svc.submitBulkCompletion(context.Background(), d.Job{Key: "123", Kind: d.JobKindExecutionListener}, tc.variables)
					err = callErr
					diagnostic = got.Diagnostic
					if tc.status == 204 {
						require.Equal(t, d.CompletionSubmissionAccepted, got.Status)
					} else if tc.status == 500 || tc.malformed {
						require.Equal(t, d.CompletionSubmissionUncertain, got.Status)
					} else {
						require.Equal(t, d.CompletionSubmissionRejected, got.Status)
					}
					if tc.malformed {
						require.Nil(t, got.HTTPStatus)
					} else {
						require.Equal(t, &tc.status, got.HTTPStatus)
					}
				} else {
					got, callErr := svc.SubmitJobWorkerOutcome(context.Background(), d.JobWorkerOutcomeRequest{Key: "123", Mode: d.JobWorkerOutcomeCompletion, Variables: tc.variables})
					err = callErr
					diagnostic = got.MutationError
					require.Equal(t, tc.status == 204, got.MutationAccepted)
					if tc.status != 204 {
						require.NotEqual(t, "skipped", got.ConfirmationStatus)
					}
				}
				require.Equal(t, int32(1), calls.Load(), "serialization must not add a retry or fallback")
				if tc.status == 204 {
					require.NoError(t, err)
					require.Empty(t, diagnostic)
				} else {
					require.Error(t, err)
					require.Equal(t, err.Error(), diagnostic)
					if tc.malformed {
						require.Contains(t, err.Error(), "unexpected end")
					} else {
						require.ErrorIs(t, err, tc.wantErr)
						require.Contains(t, err.Error(), tc.response)
					}
				}
			})
		}
	}
}

// TestCompletionEncodingFailureIsReturned ensures invalid variables fail before
// HTTP dispatch and are not converted into a successful empty completion.
func TestCompletionEncodingFailureIsReturned(t *testing.T) {
	for _, bulk := range []bool{false, true} {
		t.Run(map[bool]string{false: "scalar", true: "bulk"}[bulk], func(t *testing.T) {
			var calls atomic.Int32
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(204) }))
			t.Cleanup(server.Close)
			cfg := testx.TestConfig(t)
			cfg.APIs.Camunda.BaseURL = server.URL
			svc, err := New(cfg, server.Client(), nil)
			require.NoError(t, err)
			vars := map[string]any{"invalid": make(chan int)}
			if bulk {
				got, err := svc.submitBulkCompletion(context.Background(), d.Job{Key: "123"}, vars)
				require.ErrorIs(t, err, d.ErrValidation)
				require.Equal(t, d.CompletionSubmissionNotSubmitted, got.Status)
				require.Equal(t, err.Error(), got.Diagnostic)
			} else {
				got, err := svc.SubmitJobWorkerOutcome(context.Background(), d.JobWorkerOutcomeRequest{Key: "123", Mode: d.JobWorkerOutcomeCompletion, Variables: vars})
				require.Error(t, err)
				var encodingErr *json.UnsupportedTypeError
				require.ErrorAs(t, err, &encodingErr)
				require.False(t, got.MutationAccepted)
				require.Equal(t, err.Error(), got.MutationError)
			}
			require.Zero(t, calls.Load())
		})
	}
}

// TestCompletionOptionalValuesPreserved prevents the null fix from deleting an
// explicit lease/business ID, changing result data, or stripping null variables.
func TestCompletionOptionalValuesPreserved(t *testing.T) {
	for _, payload := range []string{
		`{"variables":{"leaseToken":null,"nested":{"businessId":null}},"result":{"type":"userTask","denied":false}}`,
		`{"variables":{},"leaseToken":"lease-42","businessId":"business-7","result":{"type":"userTask","denied":false}}`,
		`{"variables":{},"leaseToken":"","businessId":""}`,
		`{"variables":{},"leaseToken":"lease-42"}`,
		`{"variables":{},"businessId":"business-7"}`,
	} {
		t.Run(payload, func(t *testing.T) {
			var calls atomic.Int32
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, "/jobs/123/completion", r.URL.Path)
				actual, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.JSONEq(t, payload, string(actual))
				w.WriteHeader(204)
			}))
			t.Cleanup(server.Close)
			cfg := testx.TestConfig(t)
			cfg.APIs.Camunda.BaseURL = server.URL
			svc, err := New(cfg, server.Client(), nil)
			require.NoError(t, err)
			var body camunda.CompleteJobJSONRequestBody
			require.NoError(t, json.Unmarshal([]byte(payload), &body))
			response, err := svc.completeJobWithResponse(context.Background(), "123", body)
			require.NoError(t, err)
			require.Equal(t, 204, response.StatusCode())
			require.Equal(t, int32(1), calls.Load())
		})
	}
}
