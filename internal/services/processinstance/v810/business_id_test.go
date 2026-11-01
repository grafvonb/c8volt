// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	v810 "github.com/grafvonb/c8volt/internal/services/processinstance/v810"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServiceCreateProcessInstanceBusinessID validates exact generated HTTP payloads and rejects invalid intent before I/O.
func TestServiceCreateProcessInstanceBusinessID(t *testing.T) {
	t.Parallel()
	for _, selector := range []string{"id", "key"} {
		for _, tc := range []struct {
			name  string
			value *string
			valid bool
		}{
			{"omitted", nil, true}, {"empty", new(""), false}, {"whitespace", new(" "), true},
			{"opaque", new("  order,AbC  "), true}, {"malformed", new(string([]byte{0xff})), false},
			{"BMP256", new(strings.Repeat("é", 256)), true}, {"BMP257", new(strings.Repeat("é", 257)), false},
			{"supplementary128", new(strings.Repeat("😀", 128)), true}, {"supplementary129", new(strings.Repeat("😀", 129)), false},
			{"mixed256", new(strings.Repeat("a", 254) + "😀"), true}, {"mixed257", new(strings.Repeat("a", 255) + "😀"), false},
		} {
			t.Run(selector+"/"+tc.name, func(t *testing.T) {
				var calls atomic.Int32
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					assert.Equal(t, "POST", r.Method)
					assert.Equal(t, "/v2/process-instances", r.URL.Path)
					var body map[string]any
					if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
						w.WriteHeader(400)
						return
					}
					if tc.value == nil {
						assert.NotContains(t, body, "businessId")
					} else {
						assert.Equal(t, *tc.value, body["businessId"])
					}
					if selector == "id" {
						assert.Equal(t, "demo", body["processDefinitionId"])
						assert.EqualValues(t, 7, body["processDefinitionVersion"])
					} else {
						assert.Equal(t, "9001", body["processDefinitionKey"])
					}
					assert.Equal(t, "tenant-a", body["tenantId"])
					assert.Equal(t, map[string]any{"order": "42"}, body["variables"])
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]any{"processInstanceKey": "123", "businessId": "native"})
				}))
				defer server.Close()
				cfg := testConfig()
				cfg.APIs.Camunda.BaseURL = server.URL + "/v2"
				svc, err := v810.New(cfg, server.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
				require.NoError(t, err)
				data := d.ProcessInstanceData{BusinessId: tc.value, TenantId: "tenant-a", ProcessDefinitionVersion: 7, Variables: map[string]any{"order": "42"}}
				if selector == "id" {
					data.BpmnProcessId = "demo"
				} else {
					data.ProcessDefinitionSpecificId = "9001"
				}
				got, err := svc.CreateProcessInstance(context.Background(), data, services.WithNoWait())
				if !tc.valid {
					require.ErrorIs(t, err, d.ErrValidation)
					require.Zero(t, calls.Load())
					require.Empty(t, got.Key)
					return
				}
				require.NoError(t, err)
				require.EqualValues(t, 1, calls.Load())
				require.Equal(t, new("native"), got.BusinessId)
			})
		}
	}
}

// TestServiceCreateProcessInstanceBusinessIDObservation replaces acceptance identity with the actual wait snapshot, even when absent.
func TestServiceCreateProcessInstanceBusinessIDObservation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name             string
		native, observed *string
		noWait, failWait bool
	}{
		{name: "observed", native: new("accepted"), observed: new("observed")},
		{name: "absent observation", native: new("accepted")},
		{name: "empty observation", native: new("accepted"), observed: new("")},
		{name: "no wait", native: new("accepted"), observed: new("observed"), noWait: true},
		{name: "absent acceptance", observed: new("observed"), noWait: true},
		{name: "failed wait", native: new("accepted"), failWait: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var posts, gets atomic.Int32
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == "POST" && r.URL.Path == "/v2/process-instances":
					posts.Add(1)
					_ = json.NewEncoder(w).Encode(map[string]any{"processInstanceKey": "123", "businessId": tc.native})
				case r.Method == "GET" && r.URL.Path == "/v2/process-instances/123":
					gets.Add(1)
					if tc.failWait {
						w.WriteHeader(http.StatusForbidden)
						_, _ = io.WriteString(w, `{"title":"forbidden"}`)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"processInstanceKey": "123", "state": "ACTIVE", "businessId": tc.observed, "startDate": "2026-10-31T12:00:00Z"})
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
					w.WriteHeader(500)
				}
			}))
			defer server.Close()
			cfg := waitTestConfig()
			cfg.APIs.Camunda.BaseURL = server.URL + "/v2"
			svc, err := v810.New(cfg, server.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			opts := []services.CallOption{services.WithSuppressWorkflowDetailLogs(), services.WithSuppressProcessInstanceDetailLogs()}
			if tc.noWait {
				opts = append(opts, services.WithNoWait())
			}
			got, err := svc.CreateProcessInstance(context.Background(), d.ProcessInstanceData{BpmnProcessId: "demo", BusinessId: new("input")}, opts...)
			require.EqualValues(t, 1, posts.Load())
			if tc.failWait {
				require.ErrorIs(t, err, d.ErrForbidden)
				require.Equal(t, d.ProcessInstanceCreation{}, got)
			} else {
				require.NoError(t, err)
				if tc.noWait {
					require.Equal(t, tc.native, got.BusinessId)
					require.Empty(t, got.State)
				} else {
					require.Equal(t, tc.observed, got.BusinessId)
					require.Equal(t, d.StateActive, got.State)
				}
			}
			if tc.noWait {
				require.Zero(t, gets.Load())
			} else {
				require.EqualValues(t, 1, gets.Load())
			}
		})
	}
}
