// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// selectedResolutionIncidents owns incident observations and makes accidental reads after review fail.
type selectedResolutionIncidents struct {
	incsvc.API
	t      *testing.T
	reads  int
	frozen bool
	fail   bool
	writes testx.SafeSlice[string]
}

// SearchProcessInstanceIncidents enforces the service's deliberate post-selection option transition.
func (a *selectedResolutionIncidents) SearchProcessInstanceIncidents(_ context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	require.False(a.t, a.frozen)
	require.True(a.t, services.ApplyCallOptions(opts).IgnoreTenant)
	a.reads++
	if a.fail && key == "B2" {
		return nil, d.ErrUpstream
	}
	return []d.ProcessInstanceIncidentDetail{{IncidentKey: "i" + key, ProcessInstanceKey: key, State: "ACTIVE"}}, nil
}

// ResolveIncident accepts only frozen keys without planning any job or variable mutation.
func (a *selectedResolutionIncidents) ResolveIncident(_ context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	require.True(a.t, services.ApplyCallOptions(opts).IgnoreTenant)
	a.writes.Append(key)
	return d.IncidentResolutionResponse{Ok: true}, nil
}

// TestBusinessIDResolveSelectedPlan proves global expansion, context validation and zero mutation on late planning failure.
func TestBusinessIDResolveSelectedPlan(t *testing.T) {
	for _, failure := range []bool{false, true} {
		a := newSelectedCancelAPI(t)
		inc := &selectedResolutionIncidents{t: t, fail: failure}
		request := selectedCancelRequest(0).Selection
		plan, err := PlanSelectedProcessInstanceResolution(context.Background(), a, inc, request, 1)
		require.Len(t, plan.Scope.Selection.Keys, 3)
		require.Len(t, plan.Scope.Expansion.Collected, 7)
		require.ElementsMatch(t, []string{"RA", "RB"}, a.expanded.Snapshot())
		require.Empty(t, inc.writes.Snapshot())
		if failure {
			require.ErrorIs(t, err, d.ErrUpstream)
			_, err = ExecuteSelectedProcessInstanceResolution(context.Background(), a, inc, plan)
			require.ErrorIs(t, err, d.ErrPrecondition)
			require.Empty(t, inc.writes.Snapshot())
			continue
		}
		require.NoError(t, err)
		inc.frozen = true
		*request.SearchRequest.Filter.BusinessId = "changed"
		require.Equal(t, "ORDER-123", *plan.Scope.Request.Selection.SearchRequest.Filter.BusinessId)
		_, err = ExecuteSelectedProcessInstanceResolution(context.Background(), a, inc, plan, services.WithIgnoreTenant())
		require.ErrorIs(t, err, d.ErrPrecondition)
		altered := plan
		altered.Incidents.Keys = nil
		_, err = ExecuteSelectedProcessInstanceResolution(context.Background(), a, inc, altered)
		require.ErrorIs(t, err, d.ErrPrecondition)
		a.failPhase = "version"
		_, err = ExecuteSelectedProcessInstanceResolution(context.Background(), a, inc, plan, services.WithDryRun())
		require.ErrorIs(t, err, d.ErrUnsupported)
		a.failPhase = ""
		result, err := ExecuteSelectedProcessInstanceResolution(context.Background(), a, inc, plan, services.WithNoWait())
		require.NoError(t, err)
		require.Equal(t, 7, result.Submitted)
		require.Len(t, inc.writes.Snapshot(), 7)
		require.Equal(t, 7, inc.reads)
		require.Equal(t, 3, a.pages)
	}
}
