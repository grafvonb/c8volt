// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incident

import (
	"context"
	"errors"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// reviewedIncidentAPI fails on unconfigured operations, exposing accidental execution-time discovery.
type reviewedIncidentAPI struct {
	API
	reads                         int
	writes                        testx.SafeSlice[string]
	waitKeys                      testx.SafeSlice[string]
	failRead, failWrite, failWait bool
}

// SearchProcessInstanceIncidents returns duplicate owned and unrelated observations.
func (a *reviewedIncidentAPI) SearchProcessInstanceIncidents(_ context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	a.reads++
	if a.failRead && key == "B" {
		return nil, d.ErrUpstream
	}
	item := d.ProcessInstanceIncidentDetail{IncidentKey: "i" + key, ProcessInstanceKey: key, State: "ACTIVE", JobKey: "job", ElementInstanceKey: "scope"}
	return []d.ProcessInstanceIncidentDetail{item, item, {IncidentKey: "foreign", ProcessInstanceKey: "outside", State: "ACTIVE"}}, nil
}

// ResolveIncident preserves a distinctive classified failure for aggregate error tests.
func (a *reviewedIncidentAPI) ResolveIncident(_ context.Context, key string, _ ...services.CallOption) (d.IncidentResolutionResponse, error) {
	a.writes.Append(key)
	if a.failWrite && key == "iA" {
		return d.IncidentResolutionResponse{}, d.ErrForbidden
	}
	return d.IncidentResolutionResponse{Ok: true}, nil
}

// WaitForProcessInstanceIncidentsResolved captures the frozen accepted set without discovering extra targets.
func (a *reviewedIncidentAPI) WaitForProcessInstanceIncidentsResolved(_ context.Context, _ string, keys []string, _ ...services.CallOption) (d.IncidentResolutionResponse, error) {
	for _, key := range keys {
		a.waitKeys.Append(key)
	}
	if a.failWait {
		return d.IncidentResolutionResponse{}, d.ErrUpstream
	}
	return d.IncidentResolutionResponse{Ok: true}, nil
}

// TestBusinessIDResolveIncidentPlan freezes complete, deduplicated owned work and rejects incomplete/malformed evidence.
func TestBusinessIDResolveIncidentPlan(t *testing.T) {
	a := &reviewedIncidentAPI{}
	plan, err := PlanProcessInstancesIncidents(context.Background(), a, typex.Keys{"A", "A", "B"})
	require.NoError(t, err)
	require.Equal(t, 2, a.reads)
	require.Len(t, plan.Incidents, 2)
	plan.Incidents[0].JobKey = "reviewed-job"
	out, err := ExecuteProcessInstancesIncidentPlan(context.Background(), a, plan, 1, services.WithDryRun())
	require.NoError(t, err)
	require.True(t, out.DryRun)
	require.Empty(t, a.writes.Snapshot())
	require.Equal(t, 2, a.reads)
	out, err = ExecuteProcessInstancesIncidentPlan(context.Background(), a, plan, 1)
	require.NoError(t, err)
	require.Equal(t, 2, out.Confirmed)
	require.Equal(t, []string{"iA", "iB"}, a.writes.Snapshot())
	require.Equal(t, a.writes.Snapshot(), a.waitKeys.Snapshot())
	require.Equal(t, 2, a.reads)
	out.Items[0].Incidents[0].JobKey = "changed"
	require.Equal(t, "reviewed-job", plan.Incidents[0].JobKey)
	for _, change := range []func(*d.ProcessInstancesIncidentPlan){func(p *d.ProcessInstancesIncidentPlan) { p.CompletedKeys = nil }, func(p *d.ProcessInstancesIncidentPlan) { p.Keys = append(p.Keys, "A") }, func(p *d.ProcessInstancesIncidentPlan) { p.Incidents = append(p.Incidents, p.Incidents[0]) }, func(p *d.ProcessInstancesIncidentPlan) {
		p.Incidents = []d.ProcessInstanceIncidentDetail{{IncidentKey: "x", ProcessInstanceKey: "outside"}}
	}} {
		altered := plan
		change(&altered)
		_, err = ExecuteProcessInstancesIncidentPlan(context.Background(), a, altered, 1, services.WithDryRun())
		require.ErrorIs(t, err, d.ErrPrecondition)
	}
	a.failRead = true
	partial, err := PlanProcessInstancesIncidents(context.Background(), a, typex.Keys{"A", "B"})
	require.ErrorIs(t, err, d.ErrUpstream)
	require.Equal(t, typex.Keys{"A"}, partial.CompletedKeys)
	_, err = ExecuteProcessInstancesIncidentPlan(context.Background(), a, partial, 1)
	require.ErrorIs(t, err, d.ErrPrecondition)
	require.Len(t, a.writes.Snapshot(), 2)
}

// TestBusinessIDResolveIncidentFailures retains classification, fail-fast identities and acceptance on confirmation failure.
func TestBusinessIDResolveIncidentFailures(t *testing.T) {
	for _, mode := range []string{"partial", "fail-fast", "wait", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			a := &reviewedIncidentAPI{}
			plan, err := PlanProcessInstancesIncidents(context.Background(), a, typex.Keys{"A", "B"})
			require.NoError(t, err)
			opts := []services.CallOption{services.WithNoWait()}
			ctx := context.Background()
			if mode == "wait" {
				a.failWait = true
				opts = nil
			} else {
				a.failWrite = true
			}
			if mode == "fail-fast" {
				opts = append(opts, services.WithFailFast())
			}
			if mode == "cancel" {
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			}
			out, err := ExecuteProcessInstancesIncidentPlan(ctx, a, plan, 1, opts...)
			switch mode {
			case "cancel":
				require.ErrorIs(t, err, context.Canceled)
				require.Empty(t, a.writes.Snapshot())
			case "wait":
				require.ErrorIs(t, err, d.ErrUpstream)
				require.True(t, out.MutationSubmitted)
				require.Equal(t, 2, out.Failed)
			default:
				require.True(t, errors.Is(err, d.ErrForbidden))
				require.Equal(t, 1, out.Failed)
				require.Len(t, out.Items, 2)
				if mode == "fail-fast" {
					require.Equal(t, "B", out.Items[1].ProcessInstanceKey)
					require.Equal(t, "not_attempted", out.Items[1].ConfirmationStatus)
					require.Len(t, a.writes.Snapshot(), 1)
				} else {
					require.Equal(t, 1, out.Submitted)
				}
			}
		})
	}
}

// TestBusinessIDResolveFailFastFinishesInstance preserves the existing flag's process-instance scheduling boundary.
func TestBusinessIDResolveFailFastFinishesInstance(t *testing.T) {
	api := &reviewedIncidentAPI{failWrite: true}
	plan, err := PlanProcessInstancesIncidents(context.Background(), api, typex.Keys{"A", "B"})
	require.NoError(t, err)
	plan.Incidents = append(plan.Incidents, d.ProcessInstanceIncidentDetail{IncidentKey: "another-A", ProcessInstanceKey: "A", State: "ACTIVE"})
	result, err := ExecuteProcessInstancesIncidentPlan(context.Background(), api, plan, 1, services.WithNoWait(), services.WithFailFast())
	require.ErrorIs(t, err, d.ErrForbidden)
	require.Equal(t, []string{"iA", "another-A"}, api.writes.Snapshot())
	require.True(t, result.MutationSubmitted)
	require.Equal(t, []string{"another-A"}, result.Items[0].ResolvedIncidentKeys)
	require.Equal(t, "not_attempted", result.Items[1].ConfirmationStatus)
}
