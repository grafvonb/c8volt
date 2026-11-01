// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance_test

import (
	"context"
	"encoding/json"
	"fmt"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

// TestBusinessIDSearchPolicyAndNativePayload checks all public service search routes, omitted input, literal serialization and zero-I/O rejection.
func TestBusinessIDSearchPolicyAndNativePayload(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		for _, tc := range []struct {
			name    string
			value   *string
			invalid bool
		}{
			{"omitted", nil, false}, {"opaque", new(" ORDER,零*? "), false}, {"numeric", new("000123"), false},
			{"long", new(strings.Repeat("界", 300)), false}, {"empty", new(""), true}, {"invalid UTF8", new(string([]byte{0xff})), true},
		} {
			t.Run(string(version)+"/"+tc.name, func(t *testing.T) {
				var calls atomic.Int32
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					var body struct {
						Filter map[string]any `json:"filter"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					if tc.value == nil {
						require.NotContains(t, body.Filter, "businessId")
					} else {
						require.Equal(t, map[string]any{"businessId": *tc.value}, body.Filter)
					}
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
				}))
				defer server.Close()
				cfg := testConfig()
				cfg.App.CamundaVersion = version
				cfg.APIs.Camunda.BaseURL = server.URL
				api, err := processinstance.New(cfg, server.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
				require.NoError(t, err)
				var want error
				if tc.value != nil && version != toolx.V810 {
					want = d.ErrUnsupported
				} else if tc.invalid {
					want = d.ErrValidation
				}
				ctx := context.Background()

				filter := d.ProcessInstanceFilter{BusinessId: tc.value}
				require.ErrorIs(t, api.ValidateProcessInstanceSearchFilter(filter), want)
				inc := &suspendedIncidentIndex{}
				request := d.ProcessInstanceSearchRequest{Filter: filter, DirectIncidentIndex: tc.value != nil, Page: d.ProcessInstancePageRequest{Size: 1}}
				checks := []func() error{
					func() error {
						_, e := api.SearchForProcessInstancesPage(ctx, filter, d.ProcessInstancePageRequest{Size: 1})
						return e
					},
					func() error { _, e := api.SearchForProcessInstances(ctx, filter, 1); return e },
					func() error {
						_, e := processinstance.SearchProcessInstancesPages(ctx, api, inc, request, nil)
						return e
					},
					func() error {
						_, e := processinstance.SearchProcessInstancesTotal(ctx, api, inc, request, nil)
						return e
					},
				}

				for _, check := range checks {
					require.ErrorIs(t, check(), want)
				}
				if want != nil {
					require.Zero(t, calls.Load())
				} else {
					require.EqualValues(t, len(checks), calls.Load())
				}
				require.Zero(t, inc.calls)
			})
		}
	}
}

// TestBusinessIDSearchDirectIndexGuard applies the bound after native BID and direct-incident filtering, including a later-page match.
func TestBusinessIDSearchDirectIndexGuard(t *testing.T) {
	t.Parallel()
	for _, total := range []bool{false, true} {
		t.Run(fmt.Sprint(total), func(t *testing.T) {
			var calls atomic.Int32
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				require.Equal(t, "/process-instances/search", r.URL.Path)
				var body struct {
					Filter map[string]any `json:"filter"`
					Page   map[string]any `json:"page"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, "ORDER", body.Filter["businessId"])
				w.Header().Set("Content-Type", "application/json")
				if n == 1 {
					fmt.Fprint(w, `{"items":[{"processInstanceKey":"1"}],"page":{"totalItems":2,"endCursor":"next"}}`)
				} else {
					require.Equal(t, "next", body.Page["after"])
					fmt.Fprint(w, `{"items":[{"processInstanceKey":"2"}],"page":{"totalItems":1}}`)
				}
			}))
			defer server.Close()
			cfg := testConfig()
			cfg.App.CamundaVersion = toolx.V810
			cfg.APIs.Camunda.BaseURL = server.URL
			api, err := processinstance.New(cfg, server.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			inc := &suspendedIncidentIndex{items: []d.ProcessInstanceIncidentDetail{{IncidentKey: "i", ProcessInstanceKey: "2"}}}
			req := d.ProcessInstanceSearchRequest{Filter: d.ProcessInstanceFilter{BusinessId: new("ORDER")}, Page: d.ProcessInstancePageRequest{Size: 1}, Limit: 1, DirectIncidentIndex: true, ReportedTotalAllowed: true}
			if total {
				n, err := processinstance.SearchProcessInstancesTotal(context.Background(), api, inc, req, nil)
				require.NoError(t, err)
				require.EqualValues(t, 1, n)
			} else {
				got, err := processinstance.SearchProcessInstancesPages(context.Background(), api, inc, req, func(d.ProcessInstanceSearchPageStep) (d.ProcessInstanceSearchPageAction, error) {
					*req.Filter.BusinessId = "caller changed"
					return d.ProcessInstanceSearchPageActionContinue, nil
				})
				require.NoError(t, err)
				require.Len(t, got.Items, 1)
				require.Equal(t, "2", got.Items[0].Key)
				require.Equal(t, d.ProcessInstanceSearchCompletionLimitReached, got.Completion)
			}
			require.EqualValues(t, 2, calls.Load())
			require.Equal(t, 2, inc.calls)
		})
	}
}
