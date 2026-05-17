// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

func TestDiscoverOrphanProcessInstancesLimitCountsOrphansNotCandidates(t *testing.T) {
	t.Parallel()

	searches := 0
	api := stubOrphanDiscoveryAPI{
		searchPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			searches++
			require.NotNil(t, filter.HasParent)
			require.True(t, *filter.HasParent)
			require.EqualValues(t, 10, page.Size)
			switch searches {
			case 1:
				return d.ProcessInstancePage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateHasMore,
					Items: []d.ProcessInstance{
						{Key: "candidate-1", ParentKey: "parent-1"},
						{Key: "orphan-1", ParentKey: "missing-1"},
						{Key: "candidate-2", ParentKey: "parent-2"},
					},
				}, nil
			case 2:
				require.EqualValues(t, 3, page.From)
				return d.ProcessInstancePage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateNoMore,
					Items: []d.ProcessInstance{
						{Key: "orphan-2", ParentKey: "missing-2"},
						{Key: "orphan-3", ParentKey: "missing-3"},
					},
				}, nil
			default:
				t.Fatalf("unexpected search page %d", searches)
				return d.ProcessInstancePage{}, nil
			}
		},
		filterOrphans: func(_ context.Context, items []d.ProcessInstance, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			orphans := make([]d.ProcessInstance, 0, len(items))
			for _, item := range items {
				if item.Key == "orphan-1" || item.Key == "orphan-2" || item.Key == "orphan-3" {
					orphans = append(orphans, item)
				}
			}
			return orphans, nil
		},
	}

	got, err := DiscoverOrphanProcessInstances(context.Background(), api, OrphanDiscoveryRequest{
		BatchSize: 10,
		Limit:     2,
	})

	require.NoError(t, err)
	require.Equal(t, 2, searches)
	require.Equal(t, []d.ProcessInstance{
		{Key: "orphan-1", ParentKey: "missing-1"},
		{Key: "orphan-2", ParentKey: "missing-2"},
	}, got.Items)
	require.Equal(t, []string{"orphan-1", "orphan-2"}, []string(got.Keys))
}

type stubOrphanDiscoveryAPI struct {
	searchPage    func(context.Context, d.ProcessInstanceFilter, d.ProcessInstancePageRequest, ...services.CallOption) (d.ProcessInstancePage, error)
	filterOrphans func(context.Context, []d.ProcessInstance, ...services.CallOption) ([]d.ProcessInstance, error)
}

func (s stubOrphanDiscoveryAPI) SearchForProcessInstancesPage(ctx context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
	return s.searchPage(ctx, filter, page, opts...)
}

func (s stubOrphanDiscoveryAPI) FilterProcessInstanceWithOrphanParent(ctx context.Context, items []d.ProcessInstance, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	return s.filterOrphans(ctx, items, opts...)
}
