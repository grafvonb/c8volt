// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"errors"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/stretchr/testify/require"
)

type stubIncidentSearcher struct {
	search func(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
}

// SearchProcessInstanceIncidents delegates incident lookup to the configured test callback.
func (s stubIncidentSearcher) SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	if s.search == nil {
		return nil, errors.New("unexpected incident search")
	}
	return s.search(ctx, key, opts...)
}

type stubVariableSearcher struct {
	search func(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceVariable, error)
}

// SearchProcessInstanceVariables delegates variable lookup to the configured test callback.
func (s stubVariableSearcher) SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	if s.search == nil {
		return nil, errors.New("unexpected variable search")
	}
	return s.search(ctx, key, opts...)
}

type stubElementSearcher struct {
	search func(context.Context, d.ElementSearchQuery, ...services.CallOption) (d.ElementSearchResult, error)
}

// SearchElements delegates element lookup to the configured test callback.
func (s stubElementSearcher) SearchElements(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error) {
	if s.search == nil {
		return d.ElementSearchResult{}, errors.New("unexpected element search")
	}
	return s.search(ctx, query, opts...)
}

// TestEnrichProcessInstancesWithIncidentsPreservesOrderAndFiltersPerKey verifies service-owned incident association semantics.
func TestEnrichProcessInstancesWithIncidentsPreservesOrderAndFiltersPerKey(t *testing.T) {
	seen := []string{}
	got, err := EnrichProcessInstancesWithIncidents(context.Background(), stubIncidentSearcher{
		search: func(_ context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			seen = append(seen, key)
			return []d.ProcessInstanceIncidentDetail{
				{IncidentKey: "incident-" + key, ProcessInstanceKey: key},
				{IncidentKey: "broad-response-noise", ProcessInstanceKey: "other"},
			}, nil
		},
	}, []d.ProcessInstance{
		{Key: "200", BpmnProcessId: "second"},
		{Key: "100", BpmnProcessId: "first"},
	}, services.WithIgnoreTenant())

	require.NoError(t, err)
	require.Equal(t, []string{"200", "100"}, seen)
	require.Equal(t, int32(2), got.Total)
	require.Equal(t, "200", got.Items[0].Item.Key)
	require.Equal(t, []d.ProcessInstanceIncidentDetail{{IncidentKey: "incident-200", ProcessInstanceKey: "200"}}, got.Items[0].Incidents)
	require.Equal(t, "100", got.Items[1].Item.Key)
	require.Equal(t, []d.ProcessInstanceIncidentDetail{{IncidentKey: "incident-100", ProcessInstanceKey: "100"}}, got.Items[1].Incidents)
}

// TestEnrichProcessInstancesWithVariablesPreservesOrderAndProcessScope verifies service-owned variable filtering and sorting.
func TestEnrichProcessInstancesWithVariablesPreservesOrderAndProcessScope(t *testing.T) {
	got, err := EnrichProcessInstancesWithVariables(context.Background(), stubVariableSearcher{
		search: func(_ context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
			require.True(t, services.ApplyCallOptions(opts).Verbose)
			return []d.ProcessInstanceVariable{
				{Name: "zeta", Value: `"z"`, VariableKey: "3", ProcessInstanceKey: key, ScopeKey: key, APITruncated: true},
				{Name: "ignored-child-scope", Value: `"x"`, VariableKey: "2", ProcessInstanceKey: key, ScopeKey: "child"},
				{Name: "alpha", Value: `"a"`, VariableKey: "1", ProcessInstanceKey: key, ScopeKey: key},
				{Name: "ignored-other-process", Value: `"o"`, VariableKey: "4", ProcessInstanceKey: "other", ScopeKey: "other"},
			}, nil
		},
	}, []d.ProcessInstance{
		{Key: "pi-a"},
		{Key: "pi-b"},
	}, services.WithVerbose())

	require.NoError(t, err)
	require.Equal(t, int32(2), got.Total)
	require.Equal(t, "pi-a", got.Items[0].Item.Key)
	require.Equal(t, []d.ProcessInstanceVariable{
		{Name: "alpha", Value: `"a"`, VariableKey: "1", ProcessInstanceKey: "pi-a", ScopeKey: "pi-a"},
		{Name: "zeta", Value: `"z"`, VariableKey: "3", ProcessInstanceKey: "pi-a", ScopeKey: "pi-a", APITruncated: true},
	}, got.Items[0].Variables)
	require.Equal(t, "pi-b", got.Items[1].Item.Key)
	require.Equal(t, []d.ProcessInstanceVariable{
		{Name: "alpha", Value: `"a"`, VariableKey: "1", ProcessInstanceKey: "pi-b", ScopeKey: "pi-b"},
		{Name: "zeta", Value: `"z"`, VariableKey: "3", ProcessInstanceKey: "pi-b", ScopeKey: "pi-b", APITruncated: true},
	}, got.Items[1].Variables)
}

// TestEnrichProcessInstancesWithElementsPreservesOrderFiltersPerKeyAndSortsElements verifies service-owned element association semantics.
func TestEnrichProcessInstancesWithElementsPreservesOrderFiltersPerKeyAndSortsElements(t *testing.T) {
	seen := []string{}
	got, err := EnrichProcessInstancesWithElements(context.Background(), stubElementSearcher{
		search: func(_ context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error) {
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			seen = append(seen, query.ProcessInstanceKey)
			require.Empty(t, query.Key)
			require.Empty(t, query.ElementId)
			switch query.ProcessInstanceKey {
			case "200":
				return d.ElementSearchResult{Items: []d.Element{
					{ElementInstanceKey: "e-3", ElementId: "ship", Type: "SERVICE_TASK", State: "ACTIVE", StartDate: "2026-07-15T10:12:03Z", ProcessInstanceKey: "200"},
					{ElementInstanceKey: "e-1", ElementId: "start", Type: "START_EVENT", State: "COMPLETED", StartDate: "2026-07-15T10:12:01Z", EndDate: "2026-07-15T10:12:02Z", ProcessInstanceKey: "200"},
					{ElementInstanceKey: "e-2", ElementId: "review", Type: "USER_TASK", State: "ACTIVE", StartDate: "2026-07-15T10:12:03Z", ProcessInstanceKey: "200", HasIncident: true},
					{ElementInstanceKey: "ignored", ElementId: "other", ProcessInstanceKey: "other"},
				}}, nil
			case "100":
				return d.ElementSearchResult{Items: []d.Element{
					{ElementInstanceKey: "e-100", ElementId: "wait", Type: "INTERMEDIATE_CATCH_EVENT", State: "ACTIVE", StartDate: "2026-07-15T10:11:00Z", ProcessInstanceKey: "100"},
				}}, nil
			default:
				t.Fatalf("unexpected element search for process instance %s", query.ProcessInstanceKey)
				return d.ElementSearchResult{}, nil
			}
		},
	}, []d.ProcessInstance{
		{Key: "200", BpmnProcessId: "second"},
		{Key: "100", BpmnProcessId: "first"},
	}, services.WithIgnoreTenant())

	require.NoError(t, err)
	require.Equal(t, []string{"200", "100"}, seen)
	require.Equal(t, int32(2), got.Total)
	require.Equal(t, "200", got.Items[0].Item.Key)
	require.Equal(t, []d.Element{
		{ElementInstanceKey: "e-1", ElementId: "start", Type: "START_EVENT", State: "COMPLETED", StartDate: "2026-07-15T10:12:01Z", EndDate: "2026-07-15T10:12:02Z", ProcessInstanceKey: "200"},
		{ElementInstanceKey: "e-2", ElementId: "review", Type: "USER_TASK", State: "ACTIVE", StartDate: "2026-07-15T10:12:03Z", ProcessInstanceKey: "200", HasIncident: true},
		{ElementInstanceKey: "e-3", ElementId: "ship", Type: "SERVICE_TASK", State: "ACTIVE", StartDate: "2026-07-15T10:12:03Z", ProcessInstanceKey: "200"},
	}, got.Items[0].Elements)
	require.Equal(t, "100", got.Items[1].Item.Key)
	require.Equal(t, []d.Element{
		{ElementInstanceKey: "e-100", ElementId: "wait", Type: "INTERMEDIATE_CATCH_EVENT", State: "ACTIVE", StartDate: "2026-07-15T10:11:00Z", ProcessInstanceKey: "100"},
	}, got.Items[1].Elements)
}

// TestEnrichProcessInstancesWithElementsPropagatesSearchError prevents partial success after element lookup failure.
func TestEnrichProcessInstancesWithElementsPropagatesSearchError(t *testing.T) {
	wantErr := errors.New("element search failed")
	got, err := EnrichProcessInstancesWithElements(context.Background(), stubElementSearcher{
		search: func(_ context.Context, query d.ElementSearchQuery, _ ...services.CallOption) (d.ElementSearchResult, error) {
			require.Equal(t, "pi-a", query.ProcessInstanceKey)
			return d.ElementSearchResult{}, wantErr
		},
	}, []d.ProcessInstance{{Key: "pi-a"}})

	require.ErrorIs(t, err, wantErr)
	require.Empty(t, got)
}

// TestEnrichTraversalWithIncidentsPreservesMetadataAndSelectedKeys verifies traversal enrichment stays scoped to result keys.
func TestEnrichTraversalWithIncidentsPreservesMetadataAndSelectedKeys(t *testing.T) {
	seen := []string{}
	got, err := EnrichTraversalWithIncidents(context.Background(), stubIncidentSearcher{
		search: func(_ context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			seen = append(seen, key)
			return []d.ProcessInstanceIncidentDetail{
				{IncidentKey: "incident-" + key, ProcessInstanceKey: key},
				{IncidentKey: "ignored", ProcessInstanceKey: "other"},
			}, nil
		},
	}, pitraversal.Result{
		Mode:     pitraversal.ModeFamily,
		Outcome:  pitraversal.OutcomePartial,
		StartKey: "start",
		RootKey:  "root",
		Keys:     []string{"root", "child", "missing-from-chain"},
		Edges:    map[string][]string{"root": []string{"child"}},
		Chain: map[string]d.ProcessInstance{
			"root":  {Key: "root"},
			"child": {Key: "child"},
			"extra": {Key: "extra"},
		},
		MissingAncestors: []pitraversal.MissingAncestor{{Key: "parent", StartKey: "start"}},
		Warning:          "partial traversal",
	}, services.WithIgnoreTenant())

	require.NoError(t, err)
	require.Equal(t, []string{"root", "child"}, seen)
	require.Equal(t, "family", got.Mode)
	require.Equal(t, "partial", got.Outcome)
	require.Equal(t, "start", got.StartKey)
	require.Equal(t, "root", got.RootKey)
	require.Equal(t, []string{"root", "child", "missing-from-chain"}, got.Keys)
	require.Equal(t, map[string][]string{"root": []string{"child"}}, got.Edges)
	require.Equal(t, []d.MissingAncestor{{Key: "parent", StartKey: "start"}}, got.MissingAncestors)
	require.Equal(t, "partial traversal", got.Warning)
	require.Equal(t, []d.IncidentEnrichedTraversalItem{
		{Item: d.ProcessInstance{Key: "root"}, Incidents: []d.ProcessInstanceIncidentDetail{{IncidentKey: "incident-root", ProcessInstanceKey: "root"}}},
		{Item: d.ProcessInstance{Key: "child"}, Incidents: []d.ProcessInstanceIncidentDetail{{IncidentKey: "incident-child", ProcessInstanceKey: "child"}}},
	}, got.Items)
}
