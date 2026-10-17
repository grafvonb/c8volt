// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance_test

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
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSuspendedUnsupportedEntryPoints verifies all explicit request paths reject
// suspension before backend I/O, including mixed states and empty bulk scopes.
func TestSuspendedUnsupportedEntryPoints(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89} {
		t.Run(string(version), func(t *testing.T) {
			var calls atomic.Int32
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
			}))
			defer server.Close()
			cfg := testConfig()
			cfg.App.CamundaVersion = version
			cfg.App.Backoff = config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Nanosecond, MaxRetries: 1}
			cfg.APIs.Camunda.BaseURL = server.URL
			api, err := processinstance.New(cfg, server.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			ctx := context.Background()
			for _, state := range []d.State{d.StateSuspended, "suspended", "SuSpEnDeD", " SUSPENDED "} {
				t.Run(string(state), func(t *testing.T) {
					filter := d.ProcessInstanceFilter{State: state}
					desired := d.States{d.StateActive, state}
					incident := true
					request := d.ProcessInstanceExpectationRequest{States: desired, Incident: &incident}
					checks := map[string]func() error{
						"page": func() error {
							_, err := api.SearchForProcessInstancesPage(ctx, filter, d.ProcessInstancePageRequest{Size: 10})
							return err
						},
						"list":            func() error { _, err := api.SearchForProcessInstances(ctx, filter, 10); return err },
						"single state":    func() error { _, _, err := api.WaitForProcessInstanceState(ctx, "1", desired); return err },
						"single combined": func() error { _, _, err := api.WaitForProcessInstanceExpectation(ctx, "1", request); return err },
					}
					for name, check := range checks {
						t.Run(name, func(t *testing.T) { require.ErrorIs(t, check(), d.ErrUnsupported); require.Zero(t, calls.Load()) })
					}
					for _, keys := range []typex.Keys{nil, {"1", "2"}} {
						_, err := api.WaitForProcessInstancesState(ctx, keys, desired, 2)
						require.ErrorIs(t, err, d.ErrUnsupported)
						_, err = api.WaitForProcessInstancesExpectation(ctx, keys, request, 2)
						require.ErrorIs(t, err, d.ErrUnsupported)
						require.Zero(t, calls.Load())
					}
					for _, direct := range []bool{false, true} {
						for _, incidentItems := range [][]d.ProcessInstanceIncidentDetail{nil, {{ProcessInstanceKey: "1"}}} {
							inc := &suspendedIncidentIndex{items: incidentItems}
							request := d.ProcessInstanceSearchRequest{Filter: filter, DirectIncidentIndex: direct}
							_, err := processinstance.SearchProcessInstancesPages(ctx, api, inc, request, nil)
							require.ErrorIs(t, err, d.ErrUnsupported)
							require.Contains(t, err.Error(), "SUSPENDED")
							_, err = processinstance.SearchProcessInstancesTotal(ctx, api, inc, request, nil)
							require.ErrorIs(t, err, d.ErrUnsupported)
							require.Contains(t, err.Error(), "SUSPENDED")
							require.Zero(t, calls.Load())
							require.Zero(t, inc.calls)
						}
					}
				})
			}
		})
	}
}

// suspendedIncidentIndex proves validation precedes even empty incident discovery.
type suspendedIncidentIndex struct {
	calls int
	items []d.ProcessInstanceIncidentDetail
}

// SearchIncidentsPage returns a completed incident scope and records discovery.
func (s *suspendedIncidentIndex) SearchIncidentsPage(context.Context, d.IncidentFilter, d.IncidentPageRequest, ...services.CallOption) (d.IncidentPage, error) {
	s.calls++
	return d.IncidentPage{Items: s.items, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
}

// SearchProcessInstanceIncidents records any unexpected enrichment request.
func (s *suspendedIncidentIndex) SearchProcessInstanceIncidents(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	s.calls++
	return s.items, nil
}

// TestSuspendedSearchSelection verifies actual serialized equality filters and
// selected populations; the empty filter is the existing ALL representation.
func TestSuspendedSearchSelection(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		t.Run(string(version), func(t *testing.T) {
			for _, state := range []d.State{"", d.StateActive, d.StateSuspended, d.StateCanceled} {
				if state == d.StateSuspended && version != toolx.V810 {
					continue
				}
				t.Run(string(state), func(t *testing.T) {
					var calls atomic.Int32
					expected := string(state)
					if state == d.StateCanceled {
						expected = "TERMINATED"
					}
					states := []string{expected}
					if state == "" {
						states = []string{"ACTIVE", "SUSPENDED", "COMPLETED"}
					}
					server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						var body struct {
							Filter map[string]any `json:"filter"`
						}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Error(err)
							w.WriteHeader(500)
							return
						}
						if expected == "" {
							assert.NotContains(t, body.Filter, "state")
						} else {
							assert.Equal(t, expected, body.Filter["state"])
						}
						items := make([]map[string]any, 0, len(states))
						for i, state := range states {
							items = append(items, map[string]any{"processInstanceKey": fmt.Sprint(i + 1), "state": state})
						}
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(map[string]any{"items": items, "page": map[string]any{"totalItems": len(items)}})
					}))
					defer server.Close()
					cfg := testConfig()
					cfg.App.CamundaVersion = version
					cfg.APIs.Camunda.BaseURL = server.URL
					api, err := processinstance.New(cfg, server.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
					require.NoError(t, err)
					request := d.ProcessInstanceSearchRequest{Filter: d.ProcessInstanceFilter{State: state}, ReportedTotalAllowed: true}
					result, err := processinstance.SearchProcessInstancesPages(context.Background(), api, nil, request, nil)
					require.NoError(t, err)
					require.Len(t, result.Items, len(states))
					for i, item := range result.Items {
						require.Equal(t, d.State(states[i]), item.State)
					}
					total, err := processinstance.SearchProcessInstancesTotal(context.Background(), api, nil, request, nil)
					require.NoError(t, err)
					require.EqualValues(t, len(states), total)
					require.EqualValues(t, 2, calls.Load())
				})
			}
		})
	}
}

// TestSuspendedStateCapabilityContract limits validation to the new capability,
// preserving unset/ALL and existing state syntax handling on every adapter.
func TestSuspendedStateCapabilityContract(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		t.Run(string(version), func(t *testing.T) {
			cfg := testConfig()
			cfg.App.CamundaVersion = version
			api, err := processinstance.New(cfg, &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			for _, states := range []d.States{nil, {""}, {d.StateAll}, {d.StateActive, d.StateCompleted, d.StateCanceled, d.StateTerminated, d.StateAbsent}, {d.StateUnknown, "future-state"}} {
				require.NoError(t, api.ValidateProcessInstanceStates(states))
			}
			for _, states := range []d.States{{d.StateSuspended}, {d.StateActive, "sUsPeNdEd"}} {
				err := api.ValidateProcessInstanceStates(states)
				if version == toolx.V810 {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, d.ErrUnsupported)
				}
			}
		})
	}
}

// TestSuspendedDirectIncidentSearch preserves exact state selection and empty
// discovery behavior when the capability is supported.
func TestSuspendedDirectIncidentSearch(t *testing.T) {
	t.Parallel()
	for _, state := range []d.State{d.StateSuspended, d.StateActive, ""} {
		for _, empty := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/empty=%t", state, empty), func(t *testing.T) {
				var calls atomic.Int32
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					observed := "SUSPENDED"
					key := "1"
					if strings.HasSuffix(r.URL.Path, "/2") {
						observed = "ACTIVE"
						key = "2"
					}
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprintf(w, `{"processInstanceKey":%q,"state":%q}`, key, observed)
				}))
				defer server.Close()
				cfg := testConfig()
				cfg.App.CamundaVersion = toolx.V810
				cfg.APIs.Camunda.BaseURL = server.URL
				api, err := processinstance.New(cfg, server.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
				require.NoError(t, err)
				inc := &suspendedIncidentIndex{}
				expected := 0
				if !empty {
					inc.items = []d.ProcessInstanceIncidentDetail{{ProcessInstanceKey: "1"}, {ProcessInstanceKey: "2"}}
					expected = 1
					if state == "" {
						expected = 2
					}
				}
				request := d.ProcessInstanceSearchRequest{Filter: d.ProcessInstanceFilter{State: state}, DirectIncidentIndex: true}
				result, err := processinstance.SearchProcessInstancesPages(context.Background(), api, inc, request, nil)
				require.NoError(t, err)
				require.Len(t, result.Items, expected)
				for _, item := range result.Items {
					if state != "" {
						require.Equal(t, state, item.State)
					}
				}
				total, err := processinstance.SearchProcessInstancesTotal(context.Background(), api, inc, request, nil)
				require.NoError(t, err)
				require.EqualValues(t, expected, total)
				require.Equal(t, 2, inc.calls)
				if empty {
					require.Zero(t, calls.Load())
				} else {
					require.EqualValues(t, 4, calls.Load())
				}
			})
		}
	}
}
