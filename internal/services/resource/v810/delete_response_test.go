// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	camunda "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// Exercise the generated decoder, not just pre-populated mock response fields.
// Deferred history deletion is valid; missing identity and broken batches are not.
func TestDeleteResponseHTTP(t *testing.T) {
	for _, noWait := range []bool{false, true} {
		mode := "wait"
		if noWait {
			mode = "no_wait"
		}
		for _, tc := range []struct {
			name        string
			status      int
			body        string
			wantErr     error
			decodeError bool
			batch       bool
		}{
			{name: "deferred history", status: 200, body: `{"resourceKey":"123","batchOperation":null}`},
			{name: "batch history", status: 200, body: `{"resourceKey":"123","batchOperation":{"batchOperationKey":"456","batchOperationType":"DELETE_PROCESS_DEFINITION"}}`, batch: true},
			{name: "empty object", status: 200, body: `{}`, wantErr: d.ErrMalformedResponse},
			{name: "null response", status: 200, body: `null`, wantErr: d.ErrMalformedResponse},
			{name: "wrong resource", status: 200, body: `{"resourceKey":"789","batchOperation":null}`, wantErr: d.ErrMalformedResponse},
			{name: "empty batch key", status: 200, body: `{"resourceKey":"123","batchOperation":{}}`, wantErr: d.ErrMalformedResponse},
			{name: "blank batch key", status: 200, body: `{"resourceKey":"123","batchOperation":{"batchOperationKey":" "}}`, wantErr: d.ErrMalformedResponse},
			{name: "unexpected no content", status: 204, wantErr: d.ErrMalformedResponse},
			{name: "unexpected accepted", status: 202, body: `{"resourceKey":"123","batchOperation":null}`, wantErr: d.ErrMalformedResponse},
			{name: "invalid json", status: 200, body: `{`, decodeError: true},
			{name: "bad request", status: 400, body: `{"title":"Bad Request","detail":"delete rejected"}`, wantErr: d.ErrBadRequest},
			{name: "unauthorized", status: 401, body: `{"title":"Unauthorized","detail":"delete rejected"}`, wantErr: d.ErrUnauthorized},
			{name: "forbidden", status: 403, body: `{"title":"Forbidden","detail":"delete rejected"}`, wantErr: d.ErrForbidden},
			{name: "not found", status: 404, body: `{"title":"Not Found","detail":"delete rejected"}`, wantErr: d.ErrNotFound},
			{name: "server error", status: 500, body: `{"title":"Internal Server Error","detail":"delete rejected"}`, wantErr: d.ErrInternal},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				var deletes, polls atomic.Int32
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					switch r.URL.Path {
					case "/resources/123/deletion":
						deletes.Add(1)
						require.Equal(t, http.MethodPost, r.Method)
						var body map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						require.Equal(t, map[string]any{"deleteHistory": true}, body)
						w.WriteHeader(tc.status)
						_, _ = io.WriteString(w, tc.body)
					case "/batch-operations/456":
						polls.Add(1)
						require.Equal(t, http.MethodGet, r.Method)
						_, _ = io.WriteString(w, `{"batchOperationKey":"456","state":"COMPLETED"}`)
					default:
						t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				t.Cleanup(server.Close)
				cfg := testx.TestConfig(t)
				cfg.APIs.Camunda.BaseURL = server.URL
				svc, err := New(cfg, server.Client(), nil)
				require.NoError(t, err)
				var opts []services.CallOption
				if noWait {
					opts = append(opts, services.WithNoWait())
				}
				got, err := svc.Delete(context.Background(), "123", opts...)
				require.Equal(t, int32(1), deletes.Load(), "no fallback delete may hide a response error")
				wantPolls := int32(0)
				if tc.batch && !noWait {
					wantPolls = 1
				}
				require.Equal(t, wantPolls, polls.Load())
				if tc.wantErr != nil || tc.decodeError {
					require.Error(t, err)
					require.False(t, got.Ok)
					if tc.wantErr != nil {
						require.ErrorIs(t, err, tc.wantErr)
					}
					if tc.status >= 400 {
						require.Contains(t, err.Error(), "delete rejected")
					}
					return
				}
				require.NoError(t, err)
				require.True(t, got.Ok)
				require.True(t, got.DeleteHistory)
				require.Equal(t, 200, got.StatusCode)
				if tc.batch {
					require.Equal(t, "456", got.BatchOperationKey)
					if noWait {
						require.Empty(t, got.BatchState)
					} else {
						require.Equal(t, "COMPLETED", got.BatchState)
					}
				} else {
					require.Empty(t, got.BatchOperationKey)
					require.Empty(t, got.BatchState)
					require.NotContains(t, got.Status, "completed")
				}
			})
		}
	}
}

func TestDeleteResponseTransportError(t *testing.T) {
	want := errors.New("connection lost before response")
	svc := newTestService(t, "", &mockResourceClient{deleteResourceWithResponse: func(context.Context, string, camunda.DeleteResourceOpJSONRequestBody, ...camunda.RequestEditorFn) (*camunda.DeleteResourceOpResponse, error) {
		return nil, want
	}}, nil)
	got, err := svc.Delete(context.Background(), "123")
	require.ErrorIs(t, err, want)
	require.False(t, got.Ok)
}

// A returned batch must still be observed; terminal failures and read errors
// cannot be reinterpreted as the valid deferred-history response.
func TestDeleteResponseBatchFailures(t *testing.T) {
	for _, state := range []string{"FAILED", "CANCELED", "PARTIALLY_COMPLETED", "read_error", "canceled_context"} {
		t.Run(state, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var polls atomic.Int32
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/resources/123/deletion":
					_, _ = io.WriteString(w, `{"resourceKey":"123","batchOperation":{"batchOperationKey":"456"}}`)
				case "/batch-operations/456":
					polls.Add(1)
					if state == "read_error" {
						w.WriteHeader(403)
						_, _ = io.WriteString(w, `{"detail":"batch read forbidden"}`)
					} else if state == "canceled_context" {
						cancel()
						// Observe transport cancellation before returning; otherwise the
						// implicit empty 200 response can race cancellation and fail JSON decoding.
						<-r.Context().Done()
					} else {
						_ = json.NewEncoder(w).Encode(map[string]string{"batchOperationKey": "456", "state": state})
					}
				default:
					t.Errorf("unexpected request %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(server.Close)
			cfg := testx.TestConfig(t)
			cfg.APIs.Camunda.BaseURL = server.URL
			svc, err := New(cfg, server.Client(), nil)
			require.NoError(t, err)
			got, err := svc.Delete(ctx, "123")
			require.Error(t, err)
			require.Equal(t, "456", got.BatchOperationKey)
			require.Equal(t, int32(1), polls.Load())
			switch state {
			case "read_error":
				require.ErrorIs(t, err, d.ErrForbidden)
			case "canceled_context":
				require.ErrorIs(t, err, context.Canceled)
			default:
				require.Contains(t, err.Error(), state)
				require.Equal(t, state, got.BatchState)
			}
		})
	}
}
