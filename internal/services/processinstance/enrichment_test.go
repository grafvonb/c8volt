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
	"github.com/grafvonb/c8volt/testx"
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

type stubJobSearcher struct {
	search func(context.Context, d.JobSearchQuery, ...services.CallOption) (d.JobSearchResult, error)
}

// SearchJobs delegates job lookup to the configured test callback.
func (s stubJobSearcher) SearchJobs(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (d.JobSearchResult, error) {
	if s.search == nil {
		return d.JobSearchResult{}, errors.New("unexpected job search")
	}
	return s.search(ctx, query, opts...)
}

// TestEnrichProcessInstancesWithIncidentsPreservesOrderAndFiltersPerKey verifies service-owned incident association semantics.
func TestEnrichProcessInstancesWithIncidentsPreservesOrderAndFiltersPerKey(t *testing.T) {
	var seen testx.SafeSlice[string]
	got, err := EnrichProcessInstancesWithIncidents(context.Background(), stubIncidentSearcher{
		search: func(_ context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			seen.Append(key)
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
	require.ElementsMatch(t, []string{"200", "100"}, seen.Snapshot())
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

func TestEnrichProcessInstancesWithElementsEmitsFrozenProgress(t *testing.T) {
	var events []d.OpsProgressEvent
	got, err := EnrichProcessInstancesWithElements(context.Background(), stubElementSearcher{
		search: func(_ context.Context, query d.ElementSearchQuery, _ ...services.CallOption) (d.ElementSearchResult, error) {
			return d.ElementSearchResult{Items: []d.Element{{ElementInstanceKey: "el-" + query.ProcessInstanceKey, ProcessInstanceKey: query.ProcessInstanceKey}}}, nil
		},
	}, []d.ProcessInstance{{Key: "pi-1"}, {Key: "pi-2"}}, services.WithProgress(func(event d.OpsProgressEvent) {
		events = append(events, event)
	}))

	require.NoError(t, err)
	require.Len(t, got.Items, 2)
	require.Len(t, events, 3)
	require.Equal(t, d.OpsProgressEventKindFrozenScope, events[0].Kind)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading runtime elements", CoreResource: "process instance(s)", Done: 0, Total: 2}, *events[0].FrozenScope)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading runtime elements", CoreResource: "process instance(s)", Done: 1, Total: 2}, *events[1].FrozenScope)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading runtime elements", CoreResource: "process instance(s)", Done: 2, Total: 2}, *events[2].FrozenScope)
}

func TestEnrichProcessInstancesWithElementListenersAttachesByOwnerAndOmitsUnmatched(t *testing.T) {
	elementCalls := []string{}
	jobCalls := []d.JobSearchQuery{}

	got, err := EnrichProcessInstancesWithElementListeners(context.Background(), stubElementSearcher{
		search: func(_ context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error) {
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			elementCalls = append(elementCalls, query.ProcessInstanceKey)
			switch query.ProcessInstanceKey {
			case "pi-2":
				return d.ElementSearchResult{Items: []d.Element{
					{ElementInstanceKey: "el-3", ElementId: "ship", StartDate: "2026-07-15T10:12:03Z", ProcessInstanceKey: "pi-2"},
					{ElementInstanceKey: "el-1", ElementId: "start", StartDate: "2026-07-15T10:12:01Z", ProcessInstanceKey: "pi-2"},
					{ElementInstanceKey: "el-2", ElementId: "review", StartDate: "2026-07-15T10:12:02Z", ProcessInstanceKey: "pi-2"},
					{ElementInstanceKey: "ignored-other-pi", ProcessInstanceKey: "other"},
				}}, nil
			case "pi-1":
				return d.ElementSearchResult{Items: []d.Element{
					{ElementInstanceKey: "el-10", ElementId: "wait", StartDate: "2026-07-15T10:11:00Z", ProcessInstanceKey: "pi-1"},
				}}, nil
			default:
				t.Fatalf("unexpected element search for %s", query.ProcessInstanceKey)
				return d.ElementSearchResult{}, nil
			}
		},
	}, stubJobSearcher{
		search: func(_ context.Context, query d.JobSearchQuery, opts ...services.CallOption) (d.JobSearchResult, error) {
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			jobCalls = append(jobCalls, query)
			require.Contains(t, []string{d.JobKindExecutionListener, d.JobKindTaskListener}, query.Kind)
			switch query.ProcessInstanceKey + "/" + query.Kind {
			case "pi-2/" + d.JobKindExecutionListener:
				return d.JobSearchResult{Items: []d.Job{
					{Key: "job-3", Kind: d.JobKindExecutionListener, ListenerEventType: "END", Type: "ship-listener", State: "FAILED", Retries: 0, ProcessInstanceKey: "pi-2", ElementInstanceKey: "el-3", ElementId: "ship", ErrorCode: "E_SHIP"},
					{Key: "job-unmatched", Kind: d.JobKindExecutionListener, ProcessInstanceKey: "pi-2", ElementInstanceKey: "missing"},
					{Key: "job-wrong-pi", Kind: d.JobKindExecutionListener, ProcessInstanceKey: "other", ElementInstanceKey: "el-3"},
				}}, nil
			case "pi-2/" + d.JobKindTaskListener:
				return d.JobSearchResult{Items: []d.Job{
					{Key: "job-2", Kind: d.JobKindTaskListener, ListenerEventType: "COMPLETING", Type: "review-listener", State: "CREATED", Retries: 3, ProcessInstanceKey: "pi-2", ElementInstanceKey: "el-2", ElementId: "review", Worker: "worker-a"},
					{Key: "job-1", Kind: d.JobKindTaskListener, ListenerEventType: "CREATING", Type: "review-listener", State: "CREATED", Retries: 1, ProcessInstanceKey: "pi-2", ElementInstanceKey: "el-2", ElementId: "review"},
				}}, nil
			case "pi-1/" + d.JobKindExecutionListener, "pi-1/" + d.JobKindTaskListener:
				return d.JobSearchResult{Items: nil}, nil
			default:
				t.Fatalf("unexpected job search for %#v", query)
				return d.JobSearchResult{}, nil
			}
		},
	}, []d.ProcessInstance{
		{Key: "pi-2"},
		{Key: "pi-1"},
	}, services.WithIgnoreTenant())

	require.NoError(t, err)
	require.Equal(t, []string{"pi-2", "pi-1"}, elementCalls)
	require.Equal(t, []d.JobSearchQuery{
		{ProcessInstanceKey: "pi-2", Kind: d.JobKindExecutionListener},
		{ProcessInstanceKey: "pi-2", Kind: d.JobKindTaskListener},
		{ProcessInstanceKey: "pi-1", Kind: d.JobKindExecutionListener},
		{ProcessInstanceKey: "pi-1", Kind: d.JobKindTaskListener},
	}, jobCalls)
	require.Equal(t, "pi-2", got.Items[0].Item.Key)
	require.Len(t, got.Items[0].Elements, 3)
	require.NotNil(t, got.Items[0].Elements[0].Listeners)
	require.Empty(t, *got.Items[0].Elements[0].Listeners)
	require.Equal(t, []d.RuntimeListenerJob{
		{JobKey: "job-1", Kind: d.JobKindTaskListener, ListenerEventType: "CREATING", Type: "review-listener", State: "CREATED", Retries: 1, ProcessInstanceKey: "pi-2", ElementInstanceKey: "el-2", ElementId: "review"},
		{JobKey: "job-2", Kind: d.JobKindTaskListener, ListenerEventType: "COMPLETING", Type: "review-listener", State: "CREATED", Retries: 3, Worker: "worker-a", ProcessInstanceKey: "pi-2", ElementInstanceKey: "el-2", ElementId: "review"},
	}, *got.Items[0].Elements[1].Listeners)
	require.Equal(t, []d.RuntimeListenerJob{
		{JobKey: "job-3", Kind: d.JobKindExecutionListener, ListenerEventType: "END", Type: "ship-listener", State: "FAILED", Retries: 0, ProcessInstanceKey: "pi-2", ElementInstanceKey: "el-3", ElementId: "ship", ErrorCode: "E_SHIP"},
	}, *got.Items[0].Elements[2].Listeners)
	require.NotNil(t, got.Items[1].Elements[0].Listeners)
	require.Empty(t, *got.Items[1].Elements[0].Listeners)
}

func TestEnrichProcessInstancesWithElementListenersEmitsFrozenProgress(t *testing.T) {
	var events []d.OpsProgressEvent
	got, err := EnrichProcessInstancesWithElementListeners(context.Background(), stubElementSearcher{
		search: func(_ context.Context, query d.ElementSearchQuery, _ ...services.CallOption) (d.ElementSearchResult, error) {
			return d.ElementSearchResult{Items: []d.Element{{ElementInstanceKey: "el-" + query.ProcessInstanceKey, ProcessInstanceKey: query.ProcessInstanceKey}}}, nil
		},
	}, stubJobSearcher{
		search: func(_ context.Context, _ d.JobSearchQuery, _ ...services.CallOption) (d.JobSearchResult, error) {
			return d.JobSearchResult{}, nil
		},
	}, []d.ProcessInstance{{Key: "pi-1"}, {Key: "pi-2"}}, services.WithProgress(func(event d.OpsProgressEvent) {
		events = append(events, event)
	}))

	require.NoError(t, err)
	require.Len(t, got.Items, 2)
	require.Len(t, events, 3)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading listener jobs", CoreResource: "process instance(s)", Done: 0, Total: 2}, *events[0].FrozenScope)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading listener jobs", CoreResource: "process instance(s)", Done: 1, Total: 2}, *events[1].FrozenScope)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading listener jobs", CoreResource: "process instance(s)", Done: 2, Total: 2}, *events[2].FrozenScope)
}

// TestEnrichTraversalWithIncidentsPreservesMetadataAndSelectedKeys verifies traversal enrichment stays scoped to result keys.
func TestEnrichTraversalWithIncidentsPreservesMetadataAndSelectedKeys(t *testing.T) {
	var seen testx.SafeSlice[string]
	got, err := EnrichTraversalWithIncidents(context.Background(), stubIncidentSearcher{
		search: func(_ context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			seen.Append(key)
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
	require.ElementsMatch(t, []string{"root", "child"}, seen.Snapshot())
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

// TestEnrichProcessInstancesWithVariablesEmitsFrozenProgress verifies variable enrichment reports exact progress over the walked process-instance set.
func TestEnrichProcessInstancesWithVariablesEmitsFrozenProgress(t *testing.T) {
	var events []d.OpsProgressEvent
	got, err := EnrichProcessInstancesWithVariables(context.Background(), stubVariableSearcher{
		search: func(_ context.Context, key string, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
			return []d.ProcessInstanceVariable{{Name: "status", ProcessInstanceKey: key, ScopeKey: key}}, nil
		},
	}, []d.ProcessInstance{{Key: "pi-1"}, {Key: "pi-2"}}, services.WithProgress(func(event d.OpsProgressEvent) {
		events = append(events, event)
	}))

	require.NoError(t, err)
	require.Len(t, got.Items, 2)
	require.Len(t, events, 3)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading variable details", CoreResource: "process instance(s)", Done: 0, Total: 2}, *events[0].FrozenScope)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading variable details", CoreResource: "process instance(s)", Done: 1, Total: 2}, *events[1].FrozenScope)
	require.Equal(t, d.OpsFrozenScopeProgress{Phase: "loading variable details", CoreResource: "process instance(s)", Done: 2, Total: 2}, *events[2].FrozenScope)
}
