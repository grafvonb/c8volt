// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"errors"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// selectedDeleteAPI rejects recursive deletion by leaving that embedded method unset.
type selectedDeleteAPI struct {
	*selectedCancelAPI
	deleted []string
	failKey string
}

// DeleteProcessInstanceExact records only reviewed members, with an optional action-time conflict.
func (a *selectedDeleteAPI) DeleteProcessInstanceExact(_ context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error) {
	require.True(a.t, services.ApplyCallOptions(opts).IgnoreTenant)
	a.deleted = append(a.deleted, key)
	if key == a.failKey {
		return d.DeleteResponse{}, a.failure
	}
	return d.DeleteResponse{Ok: true, StatusCode: 204}, nil
}

// TestBusinessIDDeleteFrozenMembers proves child-first order, immutable scope and preserved partial evidence.
func TestBusinessIDDeleteFrozenMembers(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "partial"}[failure], func(t *testing.T) {
			a := &selectedDeleteAPI{selectedCancelAPI: newSelectedCancelAPI(t)}
			request := selectedCancelRequest(0)
			request.Operation = "delete"
			plan, err := PlanProcessInstanceSelectedMutation(context.Background(), a, nil, request)
			require.NoError(t, err)
			a.nodes["new-child"] = d.ProcessInstance{Key: "new-child", ParentKey: "RA"}
			if failure {
				a.failKey = "A1"
				a.failure = errors.New("action failed")
			}
			result, err := ExecuteProcessInstanceDeletionPlan(context.Background(), a, plan, services.WithForce(), services.WithNoWait())
			if failure {
				require.ErrorIs(t, err, a.failure)
				require.False(t, result.Roots[0].Ok)
			} else {
				require.NoError(t, err)
				require.True(t, result.Roots[0].Ok)
			}
			require.True(t, result.Roots[1].Ok)
			require.True(t, result.MutationSubmitted)
			expected := []string{"A1a", "A1", "A2", "RA", "B1", "B2", "RB"}
			if failure {
				expected = []string{"A1a", "A1", "B1", "B2", "RB"}
			}
			require.Equal(t, expected, a.deleted)
			require.Len(t, result.Members, len(expected))
			require.Equal(t, 3, a.pages)
			require.Equal(t, []string{"RA", "RB"}, a.expanded.Snapshot())
		})
	}
}

// TestBusinessIDDeleteGuards rejects altered context, wrong operation and unforced nonterminal plans before writes.
func TestBusinessIDDeleteGuards(t *testing.T) {
	for _, change := range []string{"force", "operation", "criteria", "tenant", "version", "empty-version", "dry-run"} {
		t.Run(change, func(t *testing.T) {
			a := &selectedDeleteAPI{selectedCancelAPI: newSelectedCancelAPI(t)}
			request := selectedCancelRequest(0)
			request.Operation = "delete"
			plan, err := PlanProcessInstanceSelectedMutation(context.Background(), a, nil, request)
			require.NoError(t, err)
			opts := []services.CallOption{services.WithForce()}
			switch change {
			case "force":
				opts = nil
			case "operation":
				plan.Request.Operation = "cancel"
			case "criteria":
				plan.Selection.Request.SearchRequest.Limit = 2
			case "tenant":
				a.tenant = "other"
			case "version", "empty-version":
				a.failPhase = "version"
				if change == "empty-version" {
					plan.Selection.Keys = nil
					plan.Selection.Items = nil
					plan.Families = nil
					plan.Expansion.Collected = nil
					plan.Expansion.Roots = nil
				}
			case "dry-run":
				opts = append(opts, services.WithDryRun())
			}
			_, err = ExecuteProcessInstanceDeletionPlan(context.Background(), a, plan, opts...)
			if change == "dry-run" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			if change == "version" || change == "empty-version" {
				require.ErrorIs(t, err, d.ErrUnsupported)
			}
			require.Empty(t, a.deleted)
		})
	}
}
