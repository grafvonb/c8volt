// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"
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
