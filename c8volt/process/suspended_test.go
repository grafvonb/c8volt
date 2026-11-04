// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// TestClient_SuspendedDateSnapshots verifies real adapters and one reused facade
// return fresh lookup/search snapshots; older adapters keep the new field empty.
func TestClient_SuspendedDateSnapshots(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		for _, search := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/search=%t", version, search), func(t *testing.T) {
				var calls atomic.Int32
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					n := calls.Add(1)
					date := ""
					if n == 1 {
						date = `,"suspendedDate":"2026-10-17T10:30:00.123456789+02:00"`
					}
					body := `{"processInstanceKey":"123","state":"SUSPENDED"` + date + `}`
					w.Header().Set("Content-Type", "application/json")
					if search {
						require.Equal(t, http.MethodPost, r.Method)
						require.Equal(t, "/v2/process-instances/search", r.URL.Path)
						fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, body)
					} else {
						require.Equal(t, http.MethodGet, r.Method)
						require.Equal(t, "/v2/process-instances/123", r.URL.Path)
						io.WriteString(w, body)
					}
				}))
				defer server.Close()
				log := slog.New(slog.NewTextHandler(io.Discard, nil))
				cfg := &config.Config{App: config.App{CamundaVersion: version}, APIs: config.APIs{Camunda: config.API{BaseURL: server.URL + "/v2"}, Operate: config.API{BaseURL: server.URL}}}
				api, err := pisvc.New(cfg, server.Client(), log)
				require.NoError(t, err)
				client := New(nil, api, nil, log)
				var got ProcessInstance
				for i := range 2 {
					if search {
						// Unfiltered reads stay valid on older adapters, even with an unfamiliar field.
						result, err := client.SearchProcessInstances(context.Background(), ProcessInstanceFilter{}, 10)
						require.NoError(t, err)
						require.Len(t, result.Items, 1)
						got = result.Items[0]
					} else {
						got, err = client.LookupProcessInstance(context.Background(), "123")
						require.NoError(t, err)
					}
					require.Equal(t, StateSuspended, got.State)
					if i == 0 && version == toolx.V810 {
						require.Equal(t, "2026-10-17T08:30:00.123456789Z", got.SuspendedDate)
					} else {
						require.Empty(t, got.SuspendedDate)
					}
					require.EqualValues(t, i+1, calls.Load())
				}
			})
		}
	}
}

// TestClient_SuspendedCapability verifies public errors retain unsupported
// classification through real adapters, without any instance or incident I/O.
func TestClient_SuspendedCapability(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89} {
		t.Run(string(version), func(t *testing.T) {
			var calls atomic.Int32
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer server.Close()
			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			cfg := &config.Config{App: config.App{CamundaVersion: version, Backoff: config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Nanosecond, MaxRetries: 1}}, APIs: config.APIs{Camunda: config.API{BaseURL: server.URL}, Operate: config.API{BaseURL: server.URL}}}
			api, err := pisvc.New(cfg, server.Client(), log)
			require.NoError(t, err)
			inc := stubIncidentAPI{searchIncidentsPage: func(context.Context, d.IncidentFilter, d.IncidentPageRequest, ...services.CallOption) (d.IncidentPage, error) {
				calls.Add(1)
				return d.IncidentPage{}, nil
			}}
			client := New(nil, api, inc, log)
			ctx := context.Background()
			for _, state := range []State{StateSuspended, "suspended", "SuSpEnDeD"} {
				filter := ProcessInstanceFilter{State: state}
				check := func(err error) {
					t.Helper()
					require.ErrorIs(t, err, ferr.ErrUnsupported)
					require.Equal(t, ferr.ClassUnsupported, ferr.Classify(err))
					require.Equal(t, ferr.ExitCode(ferr.ErrUnsupported), ferr.ExitCode(err))
					require.Zero(t, calls.Load())
				}
				_, err := client.SearchProcessInstances(ctx, filter, 10)
				check(err)
				_, err = client.SearchProcessInstancesPage(ctx, filter, ProcessInstancePageRequest{Size: 10})
				check(err)
				for _, direct := range []bool{false, true} {
					request := ProcessInstanceSearchRequest{Filter: filter, DirectIncidentIndex: direct}
					_, err = client.SearchProcessInstancesPages(ctx, request, nil)
					check(err)
					_, err = client.SearchProcessInstancesTotal(ctx, request, nil)
					check(err)
				}
				desired := States{StateActive, state}
				_, _, err = client.WaitForProcessInstanceState(ctx, "1", desired)
				check(err)
				for _, incident := range []*IncidentExpectation{nil, toolx.Ptr(IncidentExpectationTrue), toolx.Ptr(IncidentExpectationFalse)} {
					request := ProcessInstanceExpectationRequest{States: desired, Incident: incident}
					_, _, err = client.WaitForProcessInstanceExpectation(ctx, "1", request)
					check(err)
					for _, keys := range []typex.Keys{nil, {"1", "2"}} {
						_, err = client.WaitForProcessInstancesState(ctx, keys, desired, 2)
						check(err)
						_, err = client.WaitForProcessInstancesExpectation(ctx, keys, request, 2)
						check(err)
					}
				}
			}
		})
	}
}

// TestClient_SuspendedExpectations proves state and combined waits succeed on
// v810 only when all conditions match, while terminal waits exhaust bounded retries.
func TestClient_SuspendedExpectations(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"processInstanceKey":"1","state":"SUSPENDED","hasIncident":true}`)
	}))
	defer server.Close()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{App: config.App{CamundaVersion: toolx.V810, Backoff: config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Nanosecond, MaxRetries: 1}}, APIs: config.APIs{Camunda: config.API{BaseURL: server.URL}}}
	api, err := pisvc.New(cfg, server.Client(), log)
	require.NoError(t, err)
	client := New(nil, api, nil, log)
	ctx := context.Background()
	for _, desired := range []States{{StateSuspended}, {StateActive, StateSuspended}, {"suspended"}} {
		report, pi, err := client.WaitForProcessInstanceState(ctx, "1", desired)
		require.NoError(t, err)
		require.Equal(t, StateSuspended, report.State)
		require.Equal(t, StateSuspended, pi.State)
		reports, err := client.WaitForProcessInstancesState(ctx, typex.Keys{"1"}, desired, 1)
		require.NoError(t, err)
		require.Len(t, reports.Items, 1)
		require.Equal(t, StateSuspended, reports.Items[0].State)
		request := ProcessInstanceExpectationRequest{States: desired, Incident: toolx.Ptr(IncidentExpectationTrue)}
		combined, _, err := client.WaitForProcessInstanceExpectation(ctx, "1", request)
		require.NoError(t, err)
		require.True(t, combined.Ok)
		require.Equal(t, StateSuspended, combined.State)
		combinedReports, err := client.WaitForProcessInstancesExpectation(ctx, typex.Keys{"1"}, request, 1)
		require.NoError(t, err)
		require.Len(t, combinedReports.Items, 1)
		require.True(t, combinedReports.Items[0].Ok)
		require.Equal(t, StateSuspended, combinedReports.Items[0].State)
	}
	require.EqualValues(t, 12, calls.Load())
	for _, desired := range []States{{StateCompleted}, {StateCanceled, StateTerminated, StateAbsent}} {
		before := calls.Load()
		report, _, err := client.WaitForProcessInstanceState(ctx, "1", desired)
		require.Error(t, err)
		require.Equal(t, StateUnknown, report.State)
		require.EqualValues(t, 1, calls.Load()-before)
	}
	before := calls.Load()
	report, _, err := client.WaitForProcessInstanceExpectation(ctx, "1", ProcessInstanceExpectationRequest{States: States{StateSuspended}, Incident: toolx.Ptr(IncidentExpectationFalse)})
	require.Error(t, err)
	require.False(t, report.Ok)
	require.EqualValues(t, 1, calls.Load()-before)
}
