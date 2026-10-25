// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package search

import (
	"context"
	"errors"
	"math"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

type pageReaderFunc func(context.Context, string, d.ProcessInstanceVariablePageRequest, ...services.CallOption) (d.ProcessInstanceVariablePage, error)

func (f pageReaderFunc) SearchProcessInstanceVariablesPage(ctx context.Context, key string, page d.ProcessInstanceVariablePageRequest, opts ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
	return f(ctx, key, page, opts...)
}

func TestProcessInstanceVariablesUsesCursorAcrossSparsePages(t *testing.T) {
	requests := make([]d.ProcessInstanceVariablePageRequest, 0, 3)
	reader := pageReaderFunc(func(_ context.Context, key string, request d.ProcessInstanceVariablePageRequest, opts ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
		require.Equal(t, "pi-1", key)
		require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
		requests = append(requests, request)
		switch len(requests) {
		case 1:
			return page(request, []d.ProcessInstanceVariable{{Name: "zeta"}}, 2, total(4, d.ProcessInstanceVariableReportedTotalKindExact), "cursor-1"), nil
		case 2:
			return page(request, nil, 1, total(4, d.ProcessInstanceVariableReportedTotalKindExact), "cursor-2"), nil
		default:
			return page(request, []d.ProcessInstanceVariable{{Name: "alpha"}}, 1, total(4, d.ProcessInstanceVariableReportedTotalKindExact), ""), nil
		}
	})

	got, err := ProcessInstanceVariables(context.Background(), reader, "pi-1", services.WithIgnoreTenant())

	require.NoError(t, err)
	require.Equal(t, []string{"alpha", "zeta"}, names(got))
	require.Equal(t, "", requests[0].After)
	require.Equal(t, "cursor-1", requests[1].After)
	require.Equal(t, "cursor-2", requests[2].After)
}

func TestProcessInstanceVariablesUsesRawCountForOffsetAndProbesUnknownTotal(t *testing.T) {
	var requests []d.ProcessInstanceVariablePageRequest
	reader := pageReaderFunc(func(_ context.Context, _ string, request d.ProcessInstanceVariablePageRequest, _ ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
		requests = append(requests, request)
		if len(requests) == 1 {
			return page(request, []d.ProcessInstanceVariable{{Name: "root"}}, 3, nil, ""), nil
		}
		return page(request, nil, 0, nil, ""), nil
	})

	got, err := ProcessInstanceVariables(context.Background(), reader, "pi-1")

	require.NoError(t, err)
	require.Equal(t, []string{"root"}, names(got))
	require.Equal(t, int32(3), requests[1].From)
}

func TestProcessInstanceVariablesRejectsMalformedProgress(t *testing.T) {
	tests := []struct {
		name string
		read pageReaderFunc
	}{
		{name: "repeated cursor", read: func(_ context.Context, _ string, request d.ProcessInstanceVariablePageRequest, _ ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
			return page(request, nil, 0, total(2, d.ProcessInstanceVariableReportedTotalKindExact), "same"), nil
		}},
		{name: "decreasing lower bound", read: func() pageReaderFunc {
			calls := 0
			return func(_ context.Context, _ string, request d.ProcessInstanceVariablePageRequest, _ ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
				calls++
				if calls == 1 {
					return page(request, nil, 1, total(3, d.ProcessInstanceVariableReportedTotalKindLowerBound), "next"), nil
				}
				return page(request, nil, 1, total(2, d.ProcessInstanceVariableReportedTotalKindLowerBound), ""), nil
			}
		}()},
		{name: "exact below observed", read: func(_ context.Context, _ string, request d.ProcessInstanceVariablePageRequest, _ ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
			return page(request, nil, 2, total(1, d.ProcessInstanceVariableReportedTotalKindExact), ""), nil
		}},
		{name: "offset overflow", read: func(_ context.Context, _ string, request d.ProcessInstanceVariablePageRequest, _ ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
			return page(request, nil, math.MaxInt32, nil, ""), nil
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProcessInstanceVariables(context.Background(), tt.read, "pi-1")
			require.ErrorIs(t, err, d.ErrMalformedResponse)
			require.Nil(t, got)
		})
	}
}

func TestProcessInstanceVariablesPropagatesCancellationAndLaterFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	got, err := ProcessInstanceVariables(ctx, pageReaderFunc(func(context.Context, string, d.ProcessInstanceVariablePageRequest, ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
		called = true
		return d.ProcessInstanceVariablePage{}, nil
	}), "pi-1")
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, got)
	require.False(t, called)

	wantErr := errors.New("page two failed")
	calls := 0
	got, err = ProcessInstanceVariables(context.Background(), pageReaderFunc(func(_ context.Context, _ string, request d.ProcessInstanceVariablePageRequest, _ ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
		calls++
		if calls == 1 {
			return page(request, nil, 1, total(2, d.ProcessInstanceVariableReportedTotalKindExact), ""), nil
		}
		return d.ProcessInstanceVariablePage{}, wantErr
	}), "pi-1")
	require.ErrorIs(t, err, wantErr)
	require.Nil(t, got)
}

func page(request d.ProcessInstanceVariablePageRequest, items []d.ProcessInstanceVariable, raw int32, reported *d.ProcessInstanceVariableReportedTotal, cursor string) d.ProcessInstanceVariablePage {
	return d.ProcessInstanceVariablePage{Items: items, Request: request, RawItemCount: raw, ReportedTotal: reported, EndCursor: cursor}
}

func total(count int64, kind d.ProcessInstanceVariableReportedTotalKind) *d.ProcessInstanceVariableReportedTotal {
	return &d.ProcessInstanceVariableReportedTotal{Count: count, Kind: kind}
}

func names(items []d.ProcessInstanceVariable) []string {
	result := make([]string, len(items))
	for i := range items {
		result[i] = items[i].Name
	}
	return result
}

// scopePageReaderFunc adapts a callback without expanding legacy root-only doubles.
type scopePageReaderFunc func(context.Context, d.ScopeVariableQuery, d.ProcessInstanceVariablePageRequest, ...services.CallOption) (d.ProcessInstanceVariablePage, error)

// SearchScopeVariablesPage forwards the query and options for traversal assertions.
func (f scopePageReaderFunc) SearchScopeVariablesPage(ctx context.Context, query d.ScopeVariableQuery, request d.ProcessInstanceVariablePageRequest, opts ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
	return f(ctx, query, request, opts...)
}

// TestScopeVariablesCollectorPreservesProgressChecks proves the callback collector
// retains total, cursor, raw-count, request, and offset validation for local reads.
func TestScopeVariablesCollectorPreservesProgressChecks(t *testing.T) {
	t.Parallel()
	exact := d.ProcessInstanceVariableReportedTotalKindExact
	lower := d.ProcessInstanceVariableReportedTotalKindLowerBound
	tests := []struct {
		name  string
		pages []d.ProcessInstanceVariablePage
	}{
		{"negative total", []d.ProcessInstanceVariablePage{{ReportedTotal: total(-1, exact)}}},
		{"negative raw count", []d.ProcessInstanceVariablePage{{RawItemCount: -1}}},
		{"more selected than raw", []d.ProcessInstanceVariablePage{{Items: []d.ProcessInstanceVariable{{Name: "v"}}}}},
		{"unknown total kind", []d.ProcessInstanceVariablePage{{ReportedTotal: total(0, "other")}}},
		{"total appears", []d.ProcessInstanceVariablePage{{RawItemCount: 1}, {ReportedTotal: total(1, exact)}}},
		{"total disappears", []d.ProcessInstanceVariablePage{{RawItemCount: 1, ReportedTotal: total(2, exact)}, {}}},
		{"exact changes", []d.ProcessInstanceVariablePage{{RawItemCount: 1, ReportedTotal: total(2, exact)}, {ReportedTotal: total(3, exact)}}},
		{"exact becomes lower bound", []d.ProcessInstanceVariablePage{{RawItemCount: 1, ReportedTotal: total(2, exact)}, {ReportedTotal: total(2, lower)}}},
		{"decreasing lower bound", []d.ProcessInstanceVariablePage{{RawItemCount: 1, ReportedTotal: total(3, lower)}, {ReportedTotal: total(2, lower)}}},
		{"exact below retained bound", []d.ProcessInstanceVariablePage{{RawItemCount: 1, ReportedTotal: total(3, lower)}, {ReportedTotal: total(2, exact)}}},
		{"exact below observed", []d.ProcessInstanceVariablePage{{RawItemCount: 2, ReportedTotal: total(1, exact)}}},
		{"empty before exact completion", []d.ProcessInstanceVariablePage{{ReportedTotal: total(1, exact)}}},
		{"empty before lower bound", []d.ProcessInstanceVariablePage{{ReportedTotal: total(1, lower)}}},
		{"repeated cursor", []d.ProcessInstanceVariablePage{{EndCursor: "a"}, {EndCursor: "a"}}},
		{"cursor cycle", []d.ProcessInstanceVariablePage{{EndCursor: "a"}, {EndCursor: "b"}, {EndCursor: "a"}}},
		{"offset overflow", []d.ProcessInstanceVariablePage{{RawItemCount: math.MaxInt32}, {RawItemCount: 1}}},
	}
	query := d.ScopeVariableQuery{ProcessInstanceKey: "owner", ScopeKey: "scope"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			reader := scopePageReaderFunc(func(_ context.Context, got d.ScopeVariableQuery, request d.ProcessInstanceVariablePageRequest, opts ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
				require.Equal(t, query, got)
				require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
				require.Less(t, calls, len(tt.pages))
				p := tt.pages[calls]
				calls++
				p.Request = request
				return p, nil
			})
			got, err := ScopeVariables(context.Background(), reader, query, services.WithIgnoreTenant())
			require.ErrorIs(t, err, d.ErrMalformedResponse)
			require.Nil(t, got)
			require.Equal(t, len(tt.pages), calls)
		})
	}
	t.Run("request mismatch", func(t *testing.T) {
		got, err := ScopeVariables(context.Background(), scopePageReaderFunc(func(context.Context, d.ScopeVariableQuery, d.ProcessInstanceVariablePageRequest, ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
			return d.ProcessInstanceVariablePage{}, nil
		}), query)
		require.ErrorIs(t, err, d.ErrMalformedResponse)
		require.Nil(t, got)
	})
}

// TestScopeVariablesCollectorCursorToOffset retains cumulative raw advancement
// after a cursor disappears, and permits an exhausted lower-bound collection.
func TestScopeVariablesCollectorCursorToOffset(t *testing.T) {
	t.Parallel()
	calls := 0
	reader := scopePageReaderFunc(func(_ context.Context, _ d.ScopeVariableQuery, request d.ProcessInstanceVariablePageRequest, _ ...services.CallOption) (d.ProcessInstanceVariablePage, error) {
		calls++
		require.LessOrEqual(t, calls, 3)
		switch calls {
		case 1:
			return page(request, nil, 2, total(3, d.ProcessInstanceVariableReportedTotalKindLowerBound), "next"), nil
		case 2:
			require.Equal(t, "next", request.After)
			return page(request, []d.ProcessInstanceVariable{{Name: "local"}}, 1, total(3, d.ProcessInstanceVariableReportedTotalKindLowerBound), ""), nil
		default:
			require.Empty(t, request.After)
			require.Equal(t, int32(3), request.From)
			return page(request, nil, 0, total(3, d.ProcessInstanceVariableReportedTotalKindLowerBound), ""), nil
		}
	})
	got, err := ScopeVariables(context.Background(), reader, d.ScopeVariableQuery{ProcessInstanceKey: "owner", ScopeKey: "scope"})
	require.NoError(t, err)
	require.Equal(t, []string{"local"}, names(got))
	require.Equal(t, 3, calls)
}
