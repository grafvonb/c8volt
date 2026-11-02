// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance_test

import (
	"context"
	"errors"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDSelectionCardinality distinguishes distinct keys, exhausted uniqueness and bounded collections.
func TestBusinessIDSelectionCardinality(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		keys        []string
		cardinality d.ProcessInstanceSelectionCardinality
		limit       int32
		want        int
		failure     bool
	}{
		{"empty", nil, d.ProcessInstanceSelectionUniqueStart, 0, 0, false},
		{"unique", []string{"1", "1"}, d.ProcessInstanceSelectionUniqueStart, 0, 1, false},
		{"ambiguous same family", []string{"1", "2"}, d.ProcessInstanceSelectionUniqueStart, 0, 2, true},
		{"limited unique", []string{"1"}, d.ProcessInstanceSelectionUniqueStart, 1, 1, true},
		{"bounded collection", []string{"1", "1", "2", "3"}, d.ProcessInstanceSelectionCollection, 2, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := make([]d.ProcessInstance, 0, len(tc.keys))
			for _, key := range tc.keys {
				items = append(items, d.ProcessInstance{Key: key, ParentKey: "root", TenantId: "tenant-a"})
			}
			api := pagingSearchAPI{fetch: func(context.Context, d.ProcessInstancePageRequest) (d.ProcessInstancePage, error) {
				return d.ProcessInstancePage{Items: items, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
			}}
			result, err := processinstance.ResolveProcessInstanceSelection(context.Background(), api, nil, d.ProcessInstanceSelectionRequest{Cardinality: tc.cardinality, SearchRequest: d.ProcessInstanceSearchRequest{Limit: tc.limit}})
			if tc.failure {
				require.ErrorIs(t, err, d.ErrPrecondition)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, result.Keys, tc.want)
			require.Equal(t, "search", result.Origin)
			require.Len(t, result.TenantEvidence.Targets, tc.want)
		})
	}
}

// TestBusinessIDSelectionPartialFailure keeps observed keys and the original error without implying readiness.
func TestBusinessIDSelectionPartialFailure(t *testing.T) {
	t.Parallel()
	failure := errors.New("later page failure")
	calls := 0
	api := pagingSearchAPI{fetch: func(context.Context, d.ProcessInstancePageRequest) (d.ProcessInstancePage, error) {
		calls++
		if calls == 1 {
			return d.ProcessInstancePage{Items: []d.ProcessInstance{{Key: "1"}}, OverflowState: d.ProcessInstanceOverflowStateHasMore, EndCursor: "next"}, nil
		}
		return d.ProcessInstancePage{}, failure
	}}
	result, err := processinstance.ResolveProcessInstanceSelection(context.Background(), api, nil, d.ProcessInstanceSelectionRequest{Cardinality: d.ProcessInstanceSelectionUniqueStart})
	require.ErrorIs(t, err, failure)
	require.Equal(t, []string{"1"}, []string(result.Keys))
	require.Equal(t, d.ProcessInstanceSearchCompletionIncomplete, result.Completion)
}

// TestBusinessIDSelectionCopies isolates criteria, nested observations and tenant evidence from caller/backend mutation.
func TestBusinessIDSelectionCopies(t *testing.T) {
	t.Parallel()
	bid := " exact,* Ω "
	observed := "different"
	exists := true
	vars := map[string]any{"nested": map[string]any{"value": "original"}}
	request := d.ProcessInstanceSelectionRequest{Cardinality: d.ProcessInstanceSelectionCollection, SearchRequest: d.ProcessInstanceSearchRequest{Filter: d.ProcessInstanceFilter{BusinessId: &bid, States: d.States{d.StateActive}, VariableFilters: d.ProcessInstanceVariableFilterSet{Clauses: []d.ProcessInstanceVariableFilterClause{{Name: "x", Operator: d.ProcessInstanceVariableFilterOperatorExists, Exists: &exists}}}}}}
	api := pagingSearchAPI{fetch: func(context.Context, d.ProcessInstancePageRequest) (d.ProcessInstancePage, error) {
		return d.ProcessInstancePage{Items: []d.ProcessInstance{{Key: "1", BusinessId: &observed, Variables: vars, TenantId: "tenant-a"}}, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
	}}
	result, err := processinstance.ResolveProcessInstanceSelection(context.Background(), api, nil, request)
	require.NoError(t, err)
	bid = "changed"
	observed = "changed"
	exists = false
	request.SearchRequest.Filter.States[0] = d.StateCompleted
	vars["nested"].(map[string]any)["value"] = "changed"
	require.Equal(t, " exact,* Ω ", *result.Request.SearchRequest.Filter.BusinessId)
	require.True(t, *result.Request.SearchRequest.Filter.VariableFilters.Clauses[0].Exists)
	require.Equal(t, d.StateActive, result.Request.SearchRequest.Filter.States[0])
	require.Equal(t, "different", *result.Items[0].BusinessId)
	require.Equal(t, "original", result.Items[0].Variables["nested"].(map[string]any)["value"])
	require.Equal(t, []string{"tenant-a"}, result.TenantEvidence.ResolvedTenantIDs)
	require.False(t, result.IgnoreTenant)
}

// TestBusinessIDSelectionMissingKey rejects malformed observations instead of allowing walk to report an empty success.
func TestBusinessIDSelectionMissingKey(t *testing.T) {
	t.Parallel()
	api := pagingSearchAPI{fetch: func(context.Context, d.ProcessInstancePageRequest) (d.ProcessInstancePage, error) {
		return d.ProcessInstancePage{Items: []d.ProcessInstance{{}}, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
	}}
	result, err := processinstance.ResolveProcessInstanceSelection(context.Background(), api, nil, d.ProcessInstanceSelectionRequest{Cardinality: d.ProcessInstanceSelectionUniqueStart})
	require.ErrorIs(t, err, d.ErrMalformedResponse)
	require.Equal(t, d.ProcessInstanceSearchCompletionIncomplete, result.Completion)
	require.Len(t, result.Items, 1)
}
