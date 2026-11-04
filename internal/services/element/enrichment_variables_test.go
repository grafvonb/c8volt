// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"errors"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// scopeReaderFunc supplies complete local reads without any live dependency.
type scopeReaderFunc func(context.Context, d.ScopeVariableQuery, ...services.CallOption) ([]d.ProcessInstanceVariable, error)

// SearchScopeVariables delegates to the test-owned complete reader.
func (f scopeReaderFunc) SearchScopeVariables(ctx context.Context, q d.ScopeVariableQuery, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	return f(ctx, q, opts...)
}

// pagedElementStub models the base traversal's copy-before-visitor behavior.
type pagedElementStub struct {
	stubElementAPI
	pages func(context.Context, d.ElementSearchQuery, d.ElementSearchPageVisitor, ...services.CallOption) (d.ElementSearchPagesResult, error)
}

// SearchElementsPages delegates traversal to a test-owned fixture.
func (s pagedElementStub) SearchElementsPages(ctx context.Context, q d.ElementSearchQuery, v d.ElementSearchPageVisitor, opts ...services.CallOption) (d.ElementSearchPagesResult, error) {
	return s.pages(ctx, q, v, opts...)
}

// TestElementScopeVariablesGet verifies exact identity, cross-tenant options, both enrichments and no rediscovery.
func TestElementScopeVariablesGet(t *testing.T) {
	t.Parallel()
	for _, selection := range []d.ElementEnrichment{{}, {Variables: true}, {Listeners: true}, {Variables: true, Listeners: true}} {
		t.Run(enrichmentCaseName(selection), func(t *testing.T) {
			gets, reads, jobs := 0, 0, 0
			api := NewEnrichment(stubElementAPI{get: func(_ context.Context, key string, opts ...services.CallOption) (d.Element, error) {
				gets++
				require.Equal(t, "E", key)
				require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
				return d.Element{ElementInstanceKey: "E", ProcessInstanceKey: "P", TenantId: "other"}, nil
			}}, stubJobAPI{search: func(_ context.Context, q d.JobSearchQuery, opts ...services.CallOption) (d.JobSearchResult, error) {
				jobs++
				require.Equal(t, "P", q.ProcessInstanceKey)
				require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
				return d.JobSearchResult{Items: []d.Job{{Key: q.Kind, Kind: q.Kind, ElementInstanceKey: "E", ProcessInstanceKey: "P"}, {Key: "foreign", Kind: q.Kind, ElementInstanceKey: "sibling", ProcessInstanceKey: "P"}}}, nil
			}}, scopeReaderFunc(func(_ context.Context, q d.ScopeVariableQuery, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
				reads++
				require.Equal(t, "P", q.ProcessInstanceKey)
				require.Equal(t, "E", q.ScopeKey)
				require.Equal(t, "other", *q.ExpectedTenantId)
				require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
				return []d.ProcessInstanceVariable{{Name: "local", Value: `{"large":`, ScopeKey: "E", ProcessInstanceKey: "P", APITruncated: true}}, nil
			}))
			got, err := api.GetElementWithEnrichment(context.Background(), "E", selection, services.WithIgnoreTenant())
			require.NoError(t, err)
			require.Equal(t, 1, gets)
			if selection.Variables {
				require.Equal(t, 1, reads)
				require.NotNil(t, got.Variables)
				require.True(t, (*got.Variables)[0].APITruncated)
			} else {
				require.Zero(t, reads)
				require.Nil(t, got.Variables)
			}
			if selection.Listeners {
				require.Equal(t, 2, jobs)
				require.Len(t, *got.Listeners, 2)
			} else {
				require.Zero(t, jobs)
				require.Nil(t, got.Listeners)
			}
		})
	}
}

// enrichmentCaseName identifies independent enrichment combinations for table reports.
func enrichmentCaseName(e d.ElementEnrichment) string {
	if e.Variables {
		if e.Listeners {
			return "both"
		}
		return "variables"
	}
	if e.Listeners {
		return "listeners"
	}
	return "neither"
}

// TestElementScopeVariablesCollected preserves selection order, totals, empty scopes and listener batching.
func TestElementScopeVariablesCollected(t *testing.T) {
	t.Parallel()
	searches, jobs := 0, 0
	items := []d.Element{{ElementInstanceKey: "E2", ProcessInstanceKey: "P"}, {ElementInstanceKey: "E1", ProcessInstanceKey: "P"}}
	api := NewEnrichment(stubElementAPI{search: func(_ context.Context, q d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error) {
		searches++
		require.Equal(t, int32(2), q.Limit)
		require.False(t, services.ApplyCallOptions(opts).IgnoreTenant)
		return d.ElementSearchResult{Items: items, Total: 2}, nil
	}}, stubJobAPI{search: func(context.Context, d.JobSearchQuery, ...services.CallOption) (d.JobSearchResult, error) {
		jobs++
		return d.JobSearchResult{}, nil
	}}, scopeReaderFunc(func(_ context.Context, q d.ScopeVariableQuery, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
		require.Nil(t, q.ExpectedTenantId)
		return nil, nil
	}))
	got, err := api.SearchElementsWithEnrichment(context.Background(), d.ElementSearchQuery{Limit: 2}, d.ElementEnrichment{Variables: true, Listeners: true})
	require.NoError(t, err)
	require.Equal(t, 1, searches)
	require.Equal(t, 2, jobs)
	require.Equal(t, int32(2), got.Total)
	require.Equal(t, "E2", got.Items[0].ElementInstanceKey)
	require.Equal(t, "E1", got.Items[1].ElementInstanceKey)
	for _, item := range got.Items {
		require.NotNil(t, item.Variables)
		require.NotNil(t, *item.Variables)
		require.Empty(t, *item.Variables)
		require.NotNil(t, item.Listeners)
	}
	require.Nil(t, items[0].Variables)
	require.Nil(t, items[0].Listeners)
}

// TestElementScopeVariablesPages retains enriched aggregates while forwarding sparse pages, limits and stop/error actions.
func TestElementScopeVariablesPages(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"complete", "stop", "limit", "visitor error", "read error", "nil visitor"} {
		t.Run(mode, func(t *testing.T) {
			reads, visits := 0, 0
			sentinel := errors.New("later failure")
			query := d.ElementSearchQuery{BatchSize: 2, Limit: 3}
			api := NewEnrichment(pagedElementStub{pages: func(_ context.Context, q d.ElementSearchQuery, v d.ElementSearchPageVisitor, opts ...services.CallOption) (d.ElementSearchPagesResult, error) {
				require.Equal(t, query, q)
				require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
				result := d.ElementSearchPagesResult{Items: []d.Element{}}
				for i, page := range [][]d.Element{{}, {{ElementInstanceKey: "E1", ProcessInstanceKey: "P"}}, {{ElementInstanceKey: "E2", ProcessInstanceKey: "P"}}} {
					result.Items = append(result.Items, page...)
					result.Pages++
					result.Total = int32(len(result.Items))
					step := d.ElementSearchPageStep{Page: d.ElementSearchPage{Items: page, Request: d.ElementPageRequest{From: int32(i * 2), Size: 2}, OverflowState: d.ProcessInstanceOverflowStateHasMore, ReportedTotal: &d.ElementReportedTotal{Count: 3, Kind: d.ElementReportedTotalKindExact}}, CumulativeCount: result.Total, LimitReached: mode == "limit" && i == 1}
					action, err := v(step)
					if err != nil {
						return d.ElementSearchPagesResult{}, err
					}
					if action == d.ElementSearchPageActionStop || step.LimitReached {
						break
					}
				}
				return result, nil
			}}, nil, scopeReaderFunc(func(_ context.Context, q d.ScopeVariableQuery, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
				reads++
				if mode == "read error" && reads == 2 {
					return nil, sentinel
				}
				return []d.ProcessInstanceVariable{{Name: q.ScopeKey}}, nil
			}))
			var visitor d.ElementSearchPageVisitor
			if mode != "nil visitor" {
				visitor = func(step d.ElementSearchPageStep) (d.ElementSearchPageAction, error) {
					visits++
					require.Equal(t, int32((visits-1)*2), step.Page.Request.From)
					require.Equal(t, int32(2), step.Page.Request.Size)
					require.Equal(t, int64(3), step.Page.ReportedTotal.Count)
					require.Equal(t, d.ProcessInstanceOverflowStateHasMore, step.Page.OverflowState)
					require.Equal(t, int32(visits-1), step.CumulativeCount)
					require.Equal(t, mode == "limit" && visits == 2, step.LimitReached)
					for _, item := range step.Page.Items {
						require.NotNil(t, item.Variables)
					}
					if visits == 2 {
						if mode == "stop" {
							return d.ElementSearchPageActionStop, nil
						}
						if mode == "visitor error" {
							return "", sentinel
						}
					}
					return d.ElementSearchPageActionContinue, nil
				}
			}
			got, err := api.SearchElementsPagesWithEnrichment(context.Background(), query, d.ElementEnrichment{Variables: true}, visitor, services.WithIgnoreTenant())
			if mode == "visitor error" || mode == "read error" {
				require.ErrorIs(t, err, sentinel)
				require.Empty(t, got)
				return
			}
			require.NoError(t, err)
			want := 2
			if mode == "stop" || mode == "limit" {
				want = 1
			}
			require.Len(t, got.Items, want)
			require.Equal(t, want, reads)
			require.Equal(t, int32(want), got.Total)
			require.Equal(t, int32(want+1), got.Pages)
			for _, item := range got.Items {
				require.NotNil(t, item.Variables)
				require.Equal(t, item.ElementInstanceKey, (*item.Variables)[0].Name)
			}
		})
	}
}

// TestElementScopeVariablesFailures never publishes malformed identities or an incomplete scope as empty success.
func TestElementScopeVariablesFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		item    d.Element
		readErr error
		want    error
	}{
		{name: "missing owner", item: d.Element{ElementInstanceKey: "E"}, want: d.ErrMalformedResponse},
		{name: "blank scope", item: d.Element{ElementInstanceKey: " ", ProcessInstanceKey: "P"}, want: d.ErrMalformedResponse},
		{name: "wrong exact key", item: d.Element{ElementInstanceKey: "other", ProcessInstanceKey: "P"}, want: d.ErrMalformedResponse},
		{name: "denied read", item: d.Element{ElementInstanceKey: "E", ProcessInstanceKey: "P"}, readErr: d.ErrForbidden, want: d.ErrForbidden},
		{name: "cancelled", item: d.Element{ElementInstanceKey: "E", ProcessInstanceKey: "P"}, readErr: context.Canceled, want: context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			api := NewEnrichment(stubElementAPI{get: func(context.Context, string, ...services.CallOption) (d.Element, error) { return tc.item, nil }}, nil, scopeReaderFunc(func(context.Context, d.ScopeVariableQuery, ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
				reads++
				return nil, tc.readErr
			}))
			got, err := api.GetElementWithEnrichment(context.Background(), "E", d.ElementEnrichment{Variables: true})
			require.ErrorIs(t, err, tc.want)
			require.Empty(t, got)
			if tc.readErr == nil {
				require.Zero(t, reads)
			}
		})
	}
}

// TestElementScopeVariablesEmptySelection performs no enrichment calls after authoritative empty discovery.
func TestElementScopeVariablesEmptySelection(t *testing.T) {
	t.Parallel()
	api := NewEnrichment(stubElementAPI{search: func(context.Context, d.ElementSearchQuery, ...services.CallOption) (d.ElementSearchResult, error) {
		return d.ElementSearchResult{Items: []d.Element{}}, nil
	}}, stubJobAPI{}, scopeReaderFunc(func(context.Context, d.ScopeVariableQuery, ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
		t.Fatal("unexpected variable read")
		return nil, nil
	}))
	got, err := api.SearchElementsWithEnrichment(context.Background(), d.ElementSearchQuery{}, d.ElementEnrichment{Variables: true, Listeners: true})
	require.NoError(t, err)
	require.NotNil(t, got.Items)
	require.Empty(t, got.Items)
	require.Zero(t, got.Total)
}

// TestElementScopeVariablesDependencyAndReadFailures rejects missing dependencies and failed selected reads without publishing partial enrichment.
func TestElementScopeVariablesDependencyAndReadFailures(t *testing.T) {
	t.Parallel()
	base := stubElementAPI{search: func(context.Context, d.ElementSearchQuery, ...services.CallOption) (d.ElementSearchResult, error) {
		return d.ElementSearchResult{Items: []d.Element{{ElementInstanceKey: "E1", ProcessInstanceKey: "P"}, {ElementInstanceKey: "E2", ProcessInstanceKey: "P"}}, Total: 2}, nil
	}}
	for _, kind := range []string{"missing elements", "missing variables", "missing listeners", "late variables", "listeners"} {
		t.Run(kind, func(t *testing.T) {
			reads := 0
			var elementAPI API = base
			var jobs jobSearcher = stubJobAPI{search: func(context.Context, d.JobSearchQuery, ...services.CallOption) (d.JobSearchResult, error) {
				return d.JobSearchResult{}, d.ErrForbidden
			}}
			var variables scopeVariableSearcher = scopeReaderFunc(func(_ context.Context, q d.ScopeVariableQuery, _ ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
				reads++
				if kind == "late variables" && q.ScopeKey == "E2" {
					return nil, d.ErrForbidden
				}
				return []d.ProcessInstanceVariable{{Name: "local"}}, nil
			})
			switch kind {
			case "missing elements":
				elementAPI = nil
			case "missing variables":
				variables = nil
			case "missing listeners":
				jobs = nil
			}
			api := NewEnrichment(elementAPI, jobs, variables)
			got, err := api.SearchElementsWithEnrichment(context.Background(), d.ElementSearchQuery{}, d.ElementEnrichment{Variables: true, Listeners: true})
			require.Empty(t, got)
			if kind == "late variables" || kind == "listeners" {
				require.ErrorIs(t, err, d.ErrForbidden)
				require.Equal(t, 2, reads)
			} else {
				require.ErrorIs(t, err, d.ErrPrecondition)
				require.Zero(t, reads)
			}
		})
	}
}
