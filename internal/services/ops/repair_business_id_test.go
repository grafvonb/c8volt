// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDOperationsRepairVersion rejects supplied selectors before keyed or empty shortcuts.
func TestBusinessIDOperationsRepairVersion(t *testing.T) {
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89} {
		for _, mode := range []d.OpsRepairDiscoveryMode{d.OpsRepairDiscoveryModeSearch, d.OpsRepairDiscoveryModeKeyed} {
			t.Run(string(version)+string(mode), func(t *testing.T) {
				bid := "ORDER-123"
				api := NewWithRepairDependencies(nil, stubProcessInstanceAPI{}, repairIncidentAPI{}, nil, nil, stubJobAPI{}, version)
				_, err := api.RepairProcessInstances(context.Background(), d.OpsRepairRequest{DiscoveryMode: mode, ProcessInstanceSelection: d.ProcessInstanceFilter{BusinessId: &bid}}, services.WithDryRun())
				require.ErrorIs(t, err, d.ErrUnsupported)
			})
		}
	}
}

// TestBusinessIDOperationsRepairFrozen proves a bounded distinct plan is reused without fresh PI or incident reads.
func TestBusinessIDOperationsRepairFrozen(t *testing.T) {
	bid := " ref,Ω * "
	searches, reads := 0, 0
	var writes []string
	pi := stubProcessInstanceAPI{validateStates: func(d.States) error { return nil }, searchPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
		searches++
		require.Equal(t, bid, *filter.BusinessId)
		require.True(t, *filter.HasIncident)
		p := d.ProcessInstancePage{Request: page, OverflowState: d.ProcessInstanceOverflowStateHasMore, EndCursor: fmt.Sprint(searches)}
		switch searches {
		case 1:
			p.Items = []d.ProcessInstance{{Key: "101", BusinessId: &bid}, {Key: "101", BusinessId: &bid}}
		case 2:
		case 3:
			p.Items = []d.ProcessInstance{{Key: "102"}}
			p.OverflowState = d.ProcessInstanceOverflowStateNoMore
		default:
			t.Fatal("rediscovered selected work")
		}
		return p, nil
	}}
	inc := repairIncidentAPI{searchProcessInstanceIncidents: func(_ context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
		reads++
		require.LessOrEqual(t, reads, 2)
		return []d.ProcessInstanceIncidentDetail{{IncidentKey: "i" + key, ProcessInstanceKey: key, State: "ACTIVE"}, {IncidentKey: "i" + key, ProcessInstanceKey: key, State: "ACTIVE"}, {IncidentKey: "foreign", ProcessInstanceKey: "outside", State: "ACTIVE"}}, nil
	}, resolveIncident: func(_ context.Context, key string, _ ...services.CallOption) (d.IncidentResolutionResponse, error) {
		writes = append(writes, key)
		return d.IncidentResolutionResponse{Ok: true}, nil
	}}
	api := NewWithRepairDependencies(nil, pi, inc, nil, nil, stubJobAPI{}, toolx.V810)
	request := d.OpsRepairRequest{DiscoveryMode: d.OpsRepairDiscoveryModeSearch, ProcessInstanceSelection: d.ProcessInstanceFilter{BusinessId: &bid}, DryRun: true, Limit: 2, Workers: 1, RequestedRetries: new(int32(0))}
	planned, err := api.RepairProcessInstances(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, 3, searches)
	require.Equal(t, 2, reads)
	require.Empty(t, writes)
	require.NotNil(t, planned.FrozenPlan)
	require.True(t, planned.FrozenSet.Limited)
	require.False(t, planned.FrozenSet.Complete)
	request.DryRun = false
	request.NoWait = true
	request.FrozenPlan = planned.FrozenPlan
	result, err := api.RepairProcessInstances(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, []string{"i101", "i102"}, writes)
	require.Equal(t, 3, searches)
	require.Equal(t, 2, reads)
	require.Equal(t, bid, *result.Request.ProcessInstanceSelection.BusinessId)
	require.Equal(t, bid, *result.Report.FrozenSet.Selection.Items[0].BusinessId)
	require.Nil(t, result.Report.FrozenSet.Selection.Items[1].BusinessId)
	for _, change := range []string{"completion", "criteria", "tenant", "keys", "incidents", "variables", "retries", "timeout", "direct", "bare-keys", "empty-incomplete"} {
		t.Run(change, func(t *testing.T) {
			copy, err := CopyRepairFrozenPlan(*planned.FrozenPlan)
			require.NoError(t, err)
			invalid := request
			invalid.FrozenPlan = &copy
			switch change {
			case "completion":
				copy.Selection.Completion = d.ProcessInstanceSearchCompletionIncomplete
			case "criteria":
				invalid.ProcessInstanceSelection.BusinessId = new("other")
			case "tenant":
				copy.Tenant = "other"
			case "keys":
				copy.Selection.Keys = append(copy.Selection.Keys, "999")
			case "incidents":
				copy.Incidents.Incidents[0].ProcessInstanceKey = "outside"
			case "variables":
				invalid.Variables = map[string]any{"new": true}
			case "retries":
				invalid.RequestedRetries = new(int32(3))
			case "timeout":
				invalid.RequestedJobTimeout = time.Minute
			case "direct":
				invalid.DirectIncidentsOnly = true
			case "bare-keys":
				invalid.InputKeys = []string{"101"}
			case "empty-incomplete":
				copy.Selection.Keys = nil
				copy.Selection.Items = nil
				copy.Selection.Completion = d.ProcessInstanceSearchCompletionIncomplete
				copy.Incidents = d.ProcessInstancesIncidentPlan{}
			}
			_, err = api.RepairProcessInstances(context.Background(), invalid)
			require.Error(t, err)
			require.Len(t, writes, 2)
			require.Equal(t, 3, searches)
			require.Equal(t, 2, reads)
		})
	}
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89} {
		old := NewWithRepairDependencies(nil, stubProcessInstanceAPI{}, repairIncidentAPI{}, nil, nil, stubJobAPI{}, version)
		_, err := old.RepairProcessInstances(context.Background(), request)
		require.ErrorIs(t, err, d.ErrUnsupported)
	}
}

// TestBusinessIDOperationsRepairPartialPlanning retains incomplete evidence and never executes a failed discovery.
func TestBusinessIDOperationsRepairPartialPlanning(t *testing.T) {
	for _, failure := range []string{"search", "incident", "cancel"} {
		t.Run(failure, func(t *testing.T) {
			bid := "order"
			calls := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			pi := stubProcessInstanceAPI{validateStates: func(d.States) error { return nil }, searchPage: func(_ context.Context, _ d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
				calls++
				p := d.ProcessInstancePage{Request: page, Items: []d.ProcessInstance{{Key: "101"}}, OverflowState: d.ProcessInstanceOverflowStateNoMore}
				if failure == "search" {
					if calls == 1 {
						p.OverflowState = d.ProcessInstanceOverflowStateHasMore
						p.EndCursor = "next"
					} else {
						return p, d.ErrForbidden
					}
				}
				if failure == "cancel" {
					cancel()
				}
				return p, nil
			}}
			inc := repairIncidentAPI{searchProcessInstanceIncidents: func(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
				return []d.ProcessInstanceIncidentDetail{{IncidentKey: "i101", ProcessInstanceKey: "101", State: "ACTIVE"}}, d.ErrForbidden
			}}
			api := NewWithRepairDependencies(nil, pi, inc, nil, nil, stubJobAPI{}, toolx.V810)
			result, err := api.RepairProcessInstances(ctx, d.OpsRepairRequest{DiscoveryMode: d.OpsRepairDiscoveryModeSearch, ProcessInstanceSelection: d.ProcessInstanceFilter{BusinessId: &bid}})
			require.Error(t, err)
			require.Equal(t, d.OpsWorkflowStepStatusFailed, result.FrozenSet.Status)
			require.Nil(t, result.FrozenPlan)
			if failure != "cancel" {
				require.ErrorIs(t, err, d.ErrForbidden)
				require.NotEmpty(t, result.FrozenSet.Selection.Keys)
			}
		})
	}
}
