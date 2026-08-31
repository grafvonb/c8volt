// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processdefinition

import (
	"context"
	"strconv"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

func TestSearchProcessDefinitionsPagesUsesCursorTraversal(t *testing.T) {
	t.Parallel()

	var requests []d.ProcessDefinitionPageRequest
	var steps []d.ProcessDefinitionSearchPageStep
	api := processDefinitionSearchAPIStub{
		searchProcessDefinitionsPage: func(_ context.Context, filter d.ProcessDefinitionFilter, page d.ProcessDefinitionPageRequest, _ ...services.CallOption) (d.ProcessDefinitionPage, error) {
			require.Equal(t, d.ProcessDefinitionFilter{BpmnProcessId: "invoice"}, filter)
			requests = append(requests, page)
			switch len(requests) {
			case 1:
				return d.ProcessDefinitionPage{
					Request: page,
					Items: []d.ProcessDefinition{
						{Key: "pd-a", BpmnProcessId: "invoice"},
						{Key: "pd-b", BpmnProcessId: "invoice"},
					},
					OverflowState: d.ProcessInstanceOverflowStateHasMore,
					EndCursor:     "cursor-2",
				}, nil
			case 2:
				require.Equal(t, "cursor-2", page.After)
				return d.ProcessDefinitionPage{
					Request:       page,
					Items:         []d.ProcessDefinition{{Key: "pd-c", BpmnProcessId: "invoice"}},
					OverflowState: d.ProcessInstanceOverflowStateNoMore,
				}, nil
			default:
				t.Fatalf("unexpected process-definition page request %d", len(requests))
				return d.ProcessDefinitionPage{}, nil
			}
		},
	}

	got, err := SearchProcessDefinitionsPages(context.Background(), api, d.ProcessDefinitionSearchRequest{
		Filter: d.ProcessDefinitionFilter{BpmnProcessId: "invoice"},
		Page:   d.ProcessDefinitionPageRequest{Size: 2},
	}, func(step d.ProcessDefinitionSearchPageStep) (d.ProcessDefinitionSearchPageAction, error) {
		steps = append(steps, step)
		return d.ProcessDefinitionSearchPageActionContinue, nil
	})

	require.NoError(t, err)
	require.Len(t, requests, 2)
	require.Len(t, got.Items, 3)
	require.Equal(t, []string{"pd-a", "pd-b", "pd-c"}, []string{got.Items[0].Key, got.Items[1].Key, got.Items[2].Key})
	require.EqualValues(t, 2, got.Pages)
	require.Len(t, steps, 2)
	require.EqualValues(t, 2, steps[0].CumulativeCount)
	require.EqualValues(t, 3, steps[1].CumulativeCount)
}

// TestSearchProcessDefinitionsPagesSortsFinalCollection verifies ordinary
// searches normalize shuffled page arrivals into the canonical collection order.
func TestSearchProcessDefinitionsPagesSortsFinalCollection(t *testing.T) {
	t.Parallel()

	var requests []d.ProcessDefinitionPageRequest
	var steps []d.ProcessDefinitionSearchPageStep
	api := processDefinitionSearchAPIStub{
		searchProcessDefinitionsPage: func(_ context.Context, filter d.ProcessDefinitionFilter, page d.ProcessDefinitionPageRequest, _ ...services.CallOption) (d.ProcessDefinitionPage, error) {
			require.Equal(t, d.ProcessDefinitionFilter{TenantId: "tenant-a"}, filter)
			requests = append(requests, page)
			switch len(requests) {
			case 1:
				return d.ProcessDefinitionPage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateHasMore,
					EndCursor:     "cursor-2",
					Items: []d.ProcessDefinition{
						processDefinitionForSearchOrder("tenant-b", "invoice", 1, "tenant-b-invoice-v1"),
						processDefinitionForSearchOrder("tenant-a", "payment", 1, "tenant-a-payment-v1"),
						processDefinitionForSearchOrder("tenant-a", "invoice", 9, "tenant-a-invoice-v9"),
					},
				}, nil
			case 2:
				require.Equal(t, "cursor-2", page.After)
				return d.ProcessDefinitionPage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateNoMore,
					Items: []d.ProcessDefinition{
						processDefinitionForSearchOrder("tenant-a", "invoice", 10, "2"),
						processDefinitionForSearchOrder("<default>", "invoice", 10, "default-invoice-v10"),
						processDefinitionForSearchOrder("tenant-a", "Invoice", 1, "tenant-a-Invoice-v1"),
						processDefinitionForSearchOrder("tenant-a", "invoice", 10, "10"),
					},
				}, nil
			default:
				t.Fatalf("unexpected process-definition page request %d", len(requests))
				return d.ProcessDefinitionPage{}, nil
			}
		},
	}

	got, err := SearchProcessDefinitionsPages(context.Background(), api, d.ProcessDefinitionSearchRequest{
		Filter: d.ProcessDefinitionFilter{TenantId: "tenant-a"},
		Page:   d.ProcessDefinitionPageRequest{Size: 3},
	}, func(step d.ProcessDefinitionSearchPageStep) (d.ProcessDefinitionSearchPageAction, error) {
		steps = append(steps, step)
		return d.ProcessDefinitionSearchPageActionContinue, nil
	})

	require.NoError(t, err)
	require.Len(t, requests, 2)
	require.EqualValues(t, 2, got.Pages)
	require.EqualValues(t, 3, steps[0].CumulativeCount)
	require.EqualValues(t, 7, steps[1].CumulativeCount)
	require.Equal(t, []string{
		"default-invoice-v10",
		"tenant-a-Invoice-v1",
		"10",
		"2",
		"tenant-a-invoice-v9",
		"tenant-a-payment-v1",
		"tenant-b-invoice-v1",
	}, processDefinitionSearchKeys(got.Items))
}

// TestSearchProcessDefinitionsPagesTraversesCursorAndOffsetPagesAcrossPageSizes
// verifies complete ordinary traversal is exact-once and page-size independent.
func TestSearchProcessDefinitionsPagesTraversesCursorAndOffsetPagesAcrossPageSizes(t *testing.T) {
	t.Parallel()

	source := processDefinitionSearchTraversalFixture()
	wantKeys := []string{
		"default-invoice-v10",
		"tenant-a-Invoice-v1",
		"10",
		"2",
		"tenant-a-invoice-v9",
		"tenant-a-payment-v2",
		"tenant-b-invoice-v3",
	}

	for _, tt := range []struct {
		name      string
		pageSize  int32
		useCursor bool
		wantPages int32
	}{
		{name: "cursor-size-1", pageSize: 1, useCursor: true, wantPages: 7},
		{name: "cursor-size-2", pageSize: 2, useCursor: true, wantPages: 4},
		{name: "cursor-size-1000", pageSize: 1000, useCursor: true, wantPages: 1},
		{name: "offset-size-1", pageSize: 1, wantPages: 7},
		{name: "offset-size-2", pageSize: 2, wantPages: 4},
		{name: "offset-size-1000", pageSize: 1000, wantPages: 1},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var requests []d.ProcessDefinitionPageRequest
			api := processDefinitionSearchAPIStub{
				searchProcessDefinitionsPage: func(_ context.Context, filter d.ProcessDefinitionFilter, page d.ProcessDefinitionPageRequest, _ ...services.CallOption) (d.ProcessDefinitionPage, error) {
					require.Equal(t, d.ProcessDefinitionFilter{}, filter)
					require.Equal(t, tt.pageSize, page.Size)
					requests = append(requests, page)
					start := processDefinitionSearchPageStart(t, page, tt.useCursor)
					end := start + int(page.Size)
					if end > len(source) {
						end = len(source)
					}
					out := d.ProcessDefinitionPage{
						Request: page,
						Items:   append([]d.ProcessDefinition(nil), source[start:end]...),
					}
					if end < len(source) {
						out.OverflowState = d.ProcessInstanceOverflowStateHasMore
						if tt.useCursor {
							out.EndCursor = strconv.Itoa(end)
						}
					} else {
						out.OverflowState = d.ProcessInstanceOverflowStateNoMore
					}
					return out, nil
				},
			}

			got, err := SearchProcessDefinitionsPages(context.Background(), api, d.ProcessDefinitionSearchRequest{
				Page: d.ProcessDefinitionPageRequest{Size: tt.pageSize},
			}, nil)

			require.NoError(t, err)
			require.Equal(t, tt.wantPages, got.Pages)
			require.Len(t, requests, int(tt.wantPages))
			require.Equal(t, wantKeys, processDefinitionSearchKeys(got.Items))
		})
	}
}

// TestSearchProcessDefinitionsPagesLatestReducesExactGroupsBeforeLimit verifies
// latest traversal collects all candidates before exact group reduction and limiting.
func TestSearchProcessDefinitionsPagesLatestReducesExactGroupsBeforeLimit(t *testing.T) {
	t.Parallel()

	source := processDefinitionSearchTraversalFixture()
	var requests []d.ProcessDefinitionPageRequest
	api := processDefinitionSearchAPIStub{
		searchProcessDefinitionsPage: func(_ context.Context, filter d.ProcessDefinitionFilter, page d.ProcessDefinitionPageRequest, _ ...services.CallOption) (d.ProcessDefinitionPage, error) {
			require.Equal(t, d.ProcessDefinitionFilter{IsLatestVersion: true}, filter)
			requests = append(requests, page)
			start := processDefinitionSearchPageStart(t, page, true)
			end := start + int(page.Size)
			if end > len(source) {
				end = len(source)
			}
			out := d.ProcessDefinitionPage{
				Request: page,
				Items:   append([]d.ProcessDefinition(nil), source[start:end]...),
			}
			if end < len(source) {
				out.OverflowState = d.ProcessInstanceOverflowStateHasMore
				out.EndCursor = strconv.Itoa(end)
			} else {
				out.OverflowState = d.ProcessInstanceOverflowStateNoMore
			}
			return out, nil
		},
	}

	got, err := SearchProcessDefinitionsPages(context.Background(), api, d.ProcessDefinitionSearchRequest{
		Page:   d.ProcessDefinitionPageRequest{Size: 2},
		Limit:  3,
		Latest: true,
	}, nil)

	require.NoError(t, err)
	require.EqualValues(t, 4, got.Pages)
	require.Len(t, requests, 4)
	require.Equal(t, []string{
		"default-invoice-v10",
		"tenant-a-Invoice-v1",
		"10",
	}, processDefinitionSearchKeys(got.Items))
}

// TestCollectProcessDefinitionWatchSnapshotCollectsPagedResults verifies watch
// snapshots include every traversed process-definition page for broad searches.
func TestCollectProcessDefinitionWatchSnapshotCollectsPagedResults(t *testing.T) {
	t.Parallel()

	var requests []d.ProcessDefinitionPageRequest
	api := processDefinitionSearchAPIStub{
		searchProcessDefinitionsPage: func(_ context.Context, filter d.ProcessDefinitionFilter, page d.ProcessDefinitionPageRequest, _ ...services.CallOption) (d.ProcessDefinitionPage, error) {
			require.Equal(t, d.ProcessDefinitionFilter{}, filter)
			requests = append(requests, page)
			switch len(requests) {
			case 1:
				return d.ProcessDefinitionPage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateHasMore,
					ReportedTotal: &d.ProcessDefinitionReportedTotal{
						Count: 3,
						Kind:  d.ProcessDefinitionReportedTotalKindExact,
					},
					EndCursor: "cursor-2",
					Items: []d.ProcessDefinition{
						{Key: "pd-a", BpmnProcessId: "invoice"},
						{Key: "pd-b", BpmnProcessId: "payment"},
					},
				}, nil
			case 2:
				require.Equal(t, "cursor-2", page.After)
				return d.ProcessDefinitionPage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateNoMore,
					Items:         []d.ProcessDefinition{{Key: "pd-c", BpmnProcessId: "receipt"}},
				}, nil
			default:
				t.Fatalf("unexpected process-definition page request %d", len(requests))
				return d.ProcessDefinitionPage{}, nil
			}
		},
	}

	got, err := CollectProcessDefinitionWatchSnapshot(context.Background(), api, d.ProcessDefinitionWatchSnapshotRequest{
		WatchAllWhenUnselected: true,
		Page:                   d.ProcessDefinitionPageRequest{Size: 2},
	})

	require.NoError(t, err)
	require.Len(t, requests, 2)
	require.EqualValues(t, 3, got.Total)
	require.EqualValues(t, 2, got.Pages)
	require.False(t, got.Empty)
	require.NotNil(t, got.ReportedTotal)
	require.EqualValues(t, 3, got.ReportedTotal.Count)
	require.Equal(t, []string{"pd-a", "pd-b", "pd-c"}, []string{got.Items[0].Key, got.Items[1].Key, got.Items[2].Key})
}

// TestCollectProcessDefinitionWatchSnapshotSortsBroadPagedResults verifies
// broad watch snapshots reuse the service paged collection's canonical order.
func TestCollectProcessDefinitionWatchSnapshotSortsBroadPagedResults(t *testing.T) {
	t.Parallel()

	var requests []d.ProcessDefinitionPageRequest
	api := processDefinitionSearchAPIStub{
		searchProcessDefinitionsPage: func(_ context.Context, filter d.ProcessDefinitionFilter, page d.ProcessDefinitionPageRequest, _ ...services.CallOption) (d.ProcessDefinitionPage, error) {
			require.Equal(t, d.ProcessDefinitionFilter{}, filter)
			requests = append(requests, page)
			switch len(requests) {
			case 1:
				return d.ProcessDefinitionPage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateHasMore,
					EndCursor:     "cursor-2",
					Items: []d.ProcessDefinition{
						processDefinitionForSearchOrder("tenant-b", "invoice", 1, "tenant-b-invoice-v1"),
						processDefinitionForSearchOrder("tenant-a", "payment", 1, "tenant-a-payment-v1"),
						processDefinitionForSearchOrder("tenant-a", "invoice", 9, "tenant-a-invoice-v9"),
					},
				}, nil
			case 2:
				require.Equal(t, "cursor-2", page.After)
				return d.ProcessDefinitionPage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateNoMore,
					Items: []d.ProcessDefinition{
						processDefinitionForSearchOrder("tenant-a", "invoice", 10, "2"),
						processDefinitionForSearchOrder("<default>", "invoice", 10, "default-invoice-v10"),
						processDefinitionForSearchOrder("tenant-a", "Invoice", 1, "tenant-a-Invoice-v1"),
						processDefinitionForSearchOrder("tenant-a", "invoice", 10, "10"),
					},
				}, nil
			default:
				t.Fatalf("unexpected process-definition page request %d", len(requests))
				return d.ProcessDefinitionPage{}, nil
			}
		},
	}

	got, err := CollectProcessDefinitionWatchSnapshot(context.Background(), api, d.ProcessDefinitionWatchSnapshotRequest{
		WatchAllWhenUnselected: true,
		Page:                   d.ProcessDefinitionPageRequest{Size: 3},
	})

	require.NoError(t, err)
	require.Len(t, requests, 2)
	require.EqualValues(t, 7, got.Total)
	require.EqualValues(t, 2, got.Pages)
	require.Equal(t, []string{
		"default-invoice-v10",
		"tenant-a-Invoice-v1",
		"10",
		"2",
		"tenant-a-invoice-v9",
		"tenant-a-payment-v1",
		"tenant-b-invoice-v1",
	}, processDefinitionSearchKeys(got.Items))
}

// TestCollectProcessDefinitionWatchSnapshotRetainsCanonicalPositionsWhenStatisticsChange
// verifies statistics-only refresh changes update row data without affecting order.
func TestCollectProcessDefinitionWatchSnapshotRetainsCanonicalPositionsWhenStatisticsChange(t *testing.T) {
	t.Parallel()

	refresh := 0
	api := processDefinitionSearchAPIStub{
		searchProcessDefinitionsPage: func(_ context.Context, filter d.ProcessDefinitionFilter, page d.ProcessDefinitionPageRequest, opts ...services.CallOption) (d.ProcessDefinitionPage, error) {
			require.Equal(t, d.ProcessDefinitionFilter{TenantId: "tenant-a"}, filter)
			require.True(t, services.ApplyCallOptions(opts).WithStat)
			refresh++
			activeBase := int64(refresh * 10)
			return d.ProcessDefinitionPage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateNoMore,
				Items: []d.ProcessDefinition{
					processDefinitionForSearchOrderWithStatistics("tenant-a", "invoice", 10, "2", activeBase+4),
					processDefinitionForSearchOrderWithStatistics("tenant-a", "payment", 1, "tenant-a-payment-v1", activeBase+5),
					processDefinitionForSearchOrderWithStatistics("tenant-a", "invoice", 9, "tenant-a-invoice-v9", activeBase+3),
					processDefinitionForSearchOrderWithStatistics("tenant-a", "Invoice", 1, "tenant-a-Invoice-v1", activeBase+1),
					processDefinitionForSearchOrderWithStatistics("tenant-a", "invoice", 10, "10", activeBase+2),
				},
			}, nil
		},
	}

	request := d.ProcessDefinitionWatchSnapshotRequest{
		Filter: d.ProcessDefinitionFilter{TenantId: "tenant-a"},
		Page:   d.ProcessDefinitionPageRequest{Size: 1000},
	}
	first, err := CollectProcessDefinitionWatchSnapshot(context.Background(), api, request, services.WithStat())
	require.NoError(t, err)
	second, err := CollectProcessDefinitionWatchSnapshot(context.Background(), api, request, services.WithStat())
	require.NoError(t, err)

	wantKeys := []string{
		"tenant-a-Invoice-v1",
		"10",
		"2",
		"tenant-a-invoice-v9",
		"tenant-a-payment-v1",
	}
	require.Equal(t, wantKeys, processDefinitionSearchKeys(first.Items))
	require.Equal(t, wantKeys, processDefinitionSearchKeys(second.Items))
	require.Equal(t, map[string]int64{
		"tenant-a-Invoice-v1": 11,
		"10":                  12,
		"tenant-a-invoice-v9": 13,
		"2":                   14,
		"tenant-a-payment-v1": 15,
	}, processDefinitionSearchActiveByKey(first.Items))
	require.Equal(t, map[string]int64{
		"tenant-a-Invoice-v1": 21,
		"10":                  22,
		"tenant-a-invoice-v9": 23,
		"2":                   24,
		"tenant-a-payment-v1": 25,
	}, processDefinitionSearchActiveByKey(second.Items))
}

// TestCollectProcessDefinitionWatchSnapshotTraversesLatestPages verifies latest
// watch snapshots use shared paged traversal and canonical latest reduction.
func TestCollectProcessDefinitionWatchSnapshotTraversesLatestPages(t *testing.T) {
	t.Parallel()

	var requests []d.ProcessDefinitionPageRequest
	api := processDefinitionSearchAPIStub{
		searchProcessDefinitionsPage: func(_ context.Context, filter d.ProcessDefinitionFilter, page d.ProcessDefinitionPageRequest, opts ...services.CallOption) (d.ProcessDefinitionPage, error) {
			require.Equal(t, d.ProcessDefinitionFilter{BpmnProcessId: "invoice", IsLatestVersion: true}, filter)
			require.True(t, services.ApplyCallOptions(opts).WithStat)
			requests = append(requests, page)
			switch len(requests) {
			case 1:
				return d.ProcessDefinitionPage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateHasMore,
					ReportedTotal: &d.ProcessDefinitionReportedTotal{
						Count: 3,
						Kind:  d.ProcessDefinitionReportedTotalKindExact,
					},
					EndCursor: "cursor-2",
					Items: []d.ProcessDefinition{
						processDefinitionForSearchOrder("tenant-a", "invoice", 4, "2"),
						processDefinitionForSearchOrder("tenant-a", "invoice", 4, "10"),
					},
				}, nil
			case 2:
				require.Equal(t, "cursor-2", page.After)
				return d.ProcessDefinitionPage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateNoMore,
					Items: []d.ProcessDefinition{
						processDefinitionForSearchOrder("tenant-b", "invoice", 1, "tenant-b-invoice-v1"),
					},
				}, nil
			default:
				t.Fatalf("unexpected process-definition page request %d", len(requests))
				return d.ProcessDefinitionPage{}, nil
			}
		},
	}

	got, err := CollectProcessDefinitionWatchSnapshot(context.Background(), api, d.ProcessDefinitionWatchSnapshotRequest{
		Filter: d.ProcessDefinitionFilter{BpmnProcessId: "invoice", IsLatestVersion: true},
		Page:   d.ProcessDefinitionPageRequest{Size: 2},
		Latest: true,
	}, services.WithStat())

	require.NoError(t, err)
	require.Len(t, requests, 2)
	require.EqualValues(t, 2, got.Total)
	require.EqualValues(t, 2, got.Pages)
	require.NotNil(t, got.ReportedTotal)
	require.EqualValues(t, 3, got.ReportedTotal.Count)
	require.Equal(t, []string{"10", "tenant-b-invoice-v1"}, processDefinitionSearchKeys(got.Items))
}

// TestCollectProcessDefinitionWatchSnapshotDispatchesKey verifies direct key
// watch snapshots use the exact-key service path and preserve call options.
func TestCollectProcessDefinitionWatchSnapshotDispatchesKey(t *testing.T) {
	t.Parallel()

	api := processDefinitionSearchAPIStub{
		getProcessDefinition: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
			require.Equal(t, "2251799813685255", key)
			require.True(t, services.ApplyCallOptions(opts).WithStat)
			return d.ProcessDefinition{Key: key, BpmnProcessId: "invoice"}, nil
		},
	}

	got, err := CollectProcessDefinitionWatchSnapshot(context.Background(), api, d.ProcessDefinitionWatchSnapshotRequest{
		Key: "2251799813685255",
	}, services.WithStat())

	require.NoError(t, err)
	require.EqualValues(t, 1, got.Total)
	require.EqualValues(t, 1, got.Pages)
	require.Equal(t, "2251799813685255", got.Items[0].Key)
}

type processDefinitionSearchAPIStub struct {
	searchProcessDefinitionsPage   func(context.Context, d.ProcessDefinitionFilter, d.ProcessDefinitionPageRequest, ...services.CallOption) (d.ProcessDefinitionPage, error)
	searchProcessDefinitionsLatest func(context.Context, d.ProcessDefinitionFilter, ...services.CallOption) ([]d.ProcessDefinition, error)
	getProcessDefinition           func(context.Context, string, ...services.CallOption) (d.ProcessDefinition, error)
}

func (s processDefinitionSearchAPIStub) SearchProcessDefinitionsPage(ctx context.Context, filter d.ProcessDefinitionFilter, page d.ProcessDefinitionPageRequest, opts ...services.CallOption) (d.ProcessDefinitionPage, error) {
	if s.searchProcessDefinitionsPage == nil {
		panic("unexpected SearchProcessDefinitionsPage call")
	}
	return s.searchProcessDefinitionsPage(ctx, filter, page, opts...)
}

func (processDefinitionSearchAPIStub) SearchProcessDefinitions(context.Context, d.ProcessDefinitionFilter, int32, ...services.CallOption) ([]d.ProcessDefinition, error) {
	panic("unexpected SearchProcessDefinitions call")
}

func (s processDefinitionSearchAPIStub) SearchProcessDefinitionsLatest(ctx context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	if s.searchProcessDefinitionsLatest == nil {
		panic("unexpected SearchProcessDefinitionsLatest call")
	}
	return s.searchProcessDefinitionsLatest(ctx, filter, opts...)
}

func (s processDefinitionSearchAPIStub) GetProcessDefinition(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
	if s.getProcessDefinition == nil {
		panic("unexpected GetProcessDefinition call")
	}
	return s.getProcessDefinition(ctx, key, opts...)
}

func (processDefinitionSearchAPIStub) GetProcessDefinitionXML(context.Context, string, ...services.CallOption) (string, error) {
	panic("unexpected GetProcessDefinitionXML call")
}

// processDefinitionForSearchOrder keeps the canonical order fields readable in
// service traversal tests.
func processDefinitionForSearchOrder(tenantID, bpmnProcessID string, version int32, key string) d.ProcessDefinition {
	return d.ProcessDefinition{
		TenantId:       tenantID,
		BpmnProcessId:  bpmnProcessID,
		ProcessVersion: version,
		Key:            key,
	}
}

// processDefinitionForSearchOrderWithStatistics adds volatile statistics to a
// canonical-order fixture without making the statistics part of the sort key.
func processDefinitionForSearchOrderWithStatistics(tenantID, bpmnProcessID string, version int32, key string, active int64) d.ProcessDefinition {
	definition := processDefinitionForSearchOrder(tenantID, bpmnProcessID, version, key)
	definition.Statistics = &d.ProcessDefinitionStatistics{
		Active:                 active,
		Incidents:              active + 100,
		IncidentCountSupported: true,
	}
	return definition
}

func processDefinitionSearchTraversalFixture() []d.ProcessDefinition {
	return []d.ProcessDefinition{
		processDefinitionForSearchOrder("tenant-a", "payment", 2, "tenant-a-payment-v2"),
		processDefinitionForSearchOrder("tenant-a", "invoice", 10, "2"),
		processDefinitionForSearchOrder("tenant-a", "invoice", 9, "tenant-a-invoice-v9"),
		processDefinitionForSearchOrder("tenant-b", "invoice", 3, "tenant-b-invoice-v3"),
		processDefinitionForSearchOrder("<default>", "invoice", 10, "default-invoice-v10"),
		processDefinitionForSearchOrder("tenant-a", "Invoice", 1, "tenant-a-Invoice-v1"),
		processDefinitionForSearchOrder("tenant-a", "invoice", 10, "10"),
	}
}

func processDefinitionSearchPageStart(t *testing.T, page d.ProcessDefinitionPageRequest, useCursor bool) int {
	t.Helper()
	if !useCursor {
		require.Empty(t, page.After)
		return int(page.From)
	}
	if page.After == "" {
		require.Zero(t, page.From)
		return 0
	}
	start, err := strconv.Atoi(page.After)
	require.NoError(t, err)
	return start
}

// processDefinitionSearchKeys extracts the returned service collection identity.
func processDefinitionSearchKeys(definitions []d.ProcessDefinition) []string {
	keys := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		keys = append(keys, definition.Key)
	}
	return keys
}

// processDefinitionSearchActiveByKey records the active count attached to each
// returned key so tests can assert enrichment moved with its definition.
func processDefinitionSearchActiveByKey(definitions []d.ProcessDefinition) map[string]int64 {
	activeByKey := make(map[string]int64, len(definitions))
	for _, definition := range definitions {
		if definition.Statistics == nil {
			continue
		}
		activeByKey[definition.Key] = definition.Statistics.Active
	}
	return activeByKey
}
