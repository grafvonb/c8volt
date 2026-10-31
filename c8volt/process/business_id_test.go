// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClientCreateProcessInstanceBusinessIDVersionPolicy exercises every public creation API through real adapters and owned HTTP.
func TestClientCreateProcessInstanceBusinessIDVersionPolicy(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		for _, selector := range []string{"id", "key"} {
			for _, mode := range []string{"single", "list", "count"} {
				for _, tc := range []struct {
					name    string
					value   *string
					invalid bool
				}{
					{"absent", nil, false}, {"opaque", new("  order,AbC  "), false}, {"empty", new(""), true},
					{"malformed", new(string([]byte{0xff})), true}, {"too long", new(strings.Repeat("x", 257)), true},
				} {
					t.Run(string(version)+"/"+selector+"/"+mode+"/"+tc.name, func(t *testing.T) {
						var calls atomic.Int32
						server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							n := calls.Add(1)
							assert.Equal(t, http.MethodPost, r.Method)
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
							} else {
								assert.Equal(t, "9001", body["processDefinitionKey"])
							}
							w.Header().Set("Content-Type", "application/json")
							// Older adapters deliberately ignore even native fields outside their supported capability.
							_ = json.NewEncoder(w).Encode(map[string]any{"processInstanceKey": fmt.Sprint(n), "businessId": "native"})
						}))
						defer server.Close()
						cfg := &config.Config{App: config.App{CamundaVersion: version}, APIs: config.APIs{Camunda: config.API{BaseURL: server.URL + "/v2"}}}
						log := slog.New(slog.NewTextHandler(io.Discard, nil))
						svc, err := pisvc.New(cfg, server.Client(), log)
						require.NoError(t, err)
						cli := New(nil, svc, nil, log)
						data := ProcessInstanceData{BusinessId: tc.value}
						if selector == "id" {
							data.BpmnProcessId = "demo"
						} else {
							data.ProcessDefinitionSpecificId = "9001"
						}
						var got []ProcessInstance
						switch mode {
						case "single":
							var item ProcessInstance
							item, err = cli.CreateProcessInstance(context.Background(), data, options.WithNoWait())
							if err == nil {
								got = []ProcessInstance{item}
							}
						case "list":
							got, err = cli.CreateProcessInstances(context.Background(), []ProcessInstanceData{data, data}, options.WithNoWait())
						case "count":
							got, err = cli.CreateNProcessInstances(context.Background(), data, 2, 2, options.WithNoWait())
						}
						if tc.value != nil && version != toolx.V810 {
							require.ErrorIs(t, err, ferr.ErrUnsupported)
							require.NotErrorIs(t, err, ferr.ErrInvalidInput)
							require.Zero(t, calls.Load())
							require.Empty(t, got)
						} else if tc.invalid {
							require.ErrorIs(t, err, ferr.ErrInvalidInput)
							require.Zero(t, calls.Load())
							require.Empty(t, got)
						} else {
							require.NoError(t, err)
							want := 2
							if mode == "single" {
								want = 1
							}
							require.Len(t, got, want)
							require.EqualValues(t, want, calls.Load())
							for _, item := range got {
								require.NotEmpty(t, item.Key)
								if version == toolx.V810 {
									require.Equal(t, new("native"), item.BusinessId)
								} else {
									require.Nil(t, item.BusinessId)
								}
							}
						}
					})
				}
			}
		}
	}
}

// TestClientCreateProcessInstancesBusinessIDPartial preserves successful observations and original facade classification after a failure.
func TestClientCreateProcessInstancesBusinessIDPartial(t *testing.T) {
	t.Parallel()
	for _, counted := range []bool{false, true} {
		for _, failure := range []struct {
			name           string
			domain, public error
		}{
			{"conflict", d.ErrConflict, ferr.ErrConflict}, {"validation", d.ErrValidation, ferr.ErrInvalidInput},
			{"unsupported", d.ErrUnsupported, ferr.ErrUnsupported}, {"deadline", context.DeadlineExceeded, ferr.ErrTimeout},
		} {
			t.Run(failure.name+map[bool]string{false: "/list", true: "/count"}[counted], func(t *testing.T) {
				input := new("  order,AbC  ")
				observed := new("native")
				calls := 0
				api := stubProcessInstanceAPI{createProcessInstance: func(_ context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error) {
					calls++
					assert.True(t, services.ApplyCallOptions(opts).NoWait)
					assert.Equal(t, new("  order,AbC  "), data.BusinessId)
					assert.NotSame(t, input, data.BusinessId)
					if calls == 2 {
						return d.ProcessInstanceCreation{Key: "failed", BusinessId: new("not successful")}, failure.domain
					}
					return d.ProcessInstanceCreation{Key: "123", BusinessId: observed}, nil
				}}
				cli := New(nil, api, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
				data := ProcessInstanceData{BpmnProcessId: "demo", BusinessId: input}
				var got []ProcessInstance
				var err error
				if counted {
					got, err = cli.CreateNProcessInstances(context.Background(), data, 3, 1, options.WithNoWait(), options.WithFailFast())
				} else {
					got, err = cli.CreateProcessInstances(context.Background(), []ProcessInstanceData{data, data, data}, options.WithNoWait())
				}
				require.ErrorIs(t, err, failure.public)
				require.Len(t, got, 1)
				require.Equal(t, 2, calls)
				require.Equal(t, "123", got[0].Key)
				require.Equal(t, observed, got[0].BusinessId)
				require.NotSame(t, observed, got[0].BusinessId)
				*observed = "changed"
				require.Equal(t, "native", *got[0].BusinessId)
			})
		}
	}
}

// TestClientCreateProcessInstancesBusinessIDHTTPConflict retains actual successful HTTP responses through native, service, and facade failure handling.
func TestClientCreateProcessInstancesBusinessIDHTTPConflict(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"list", "count sequential", "count concurrent"} {
		for _, noWait := range []bool{false, true} {
			t.Run(mode+map[bool]string{false: "/wait", true: "/no wait"}[noWait], func(t *testing.T) {
				var posts, gets atomic.Int32
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if r.Method == http.MethodGet {
						gets.Add(1)
						assert.Equal(t, "/v2/process-instances/123", r.URL.Path)
						_, _ = io.WriteString(w, `{"processInstanceKey":"123","state":"ACTIVE","businessId":"observed"}`)
						return
					}
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "/v2/process-instances", r.URL.Path)
					var body map[string]any
					if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
						w.WriteHeader(400)
						return
					}
					assert.Equal(t, "  order,AbC  ", body["businessId"])
					n := posts.Add(1)
					if n > 1 {
						w.WriteHeader(http.StatusConflict)
						_, _ = io.WriteString(w, `{"title":"business ID already exists","status":409}`)
						return
					}
					_, _ = io.WriteString(w, `{"processInstanceKey":"123","businessId":"native"}`)
				}))
				defer server.Close()
				cfg := &config.Config{App: config.App{CamundaVersion: toolx.V810}, APIs: config.APIs{Camunda: config.API{BaseURL: server.URL + "/v2"}}}
				log := slog.New(slog.NewTextHandler(io.Discard, nil))
				svc, err := pisvc.New(cfg, server.Client(), log)
				require.NoError(t, err)
				cli := New(nil, svc, nil, log)
				opts := []options.FacadeOption{}
				if noWait {
					opts = append(opts, options.WithNoWait())
				}
				data := ProcessInstanceData{BpmnProcessId: "demo", BusinessId: new("  order,AbC  ")}
				var got []ProcessInstance
				if mode == "list" {
					got, err = cli.CreateProcessInstances(context.Background(), []ProcessInstanceData{data, data, data}, opts...)
				} else {
					workers := 1
					if mode == "count concurrent" {
						workers = 2
					}
					got, err = cli.CreateNProcessInstances(context.Background(), data, 3, workers, opts...)
				}
				require.ErrorIs(t, err, ferr.ErrConflict)
				require.Len(t, got, 1)
				require.Equal(t, "123", got[0].Key)
				if noWait {
					require.Equal(t, new("native"), got[0].BusinessId)
					require.Zero(t, gets.Load())
				} else {
					require.Equal(t, new("observed"), got[0].BusinessId)
					require.EqualValues(t, 1, gets.Load())
				}
				expected := 3
				if mode == "list" {
					expected = 2
				}
				require.EqualValues(t, expected, posts.Load())
			})
		}
	}
}
