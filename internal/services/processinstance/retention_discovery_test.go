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

func TestDiscoverRetentionProcessInstancesUsesPagedEndDateFilterAndFreezesEligibleKeys(t *testing.T) {
	t.Parallel()

	searches := 0
	api := stubRetentionDiscoveryAPI{
		searchPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			searches++
			require.Equal(t, "2026-02-13", filter.EndDateBefore)
			require.EqualValues(t, 2, page.Size)
			switch searches {
			case 1:
				return d.ProcessInstancePage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateHasMore,
					Items: []d.ProcessInstance{
						{Key: "seed-1", EndDate: "2026-02-12"},
						{Key: "running-1"},
					},
				}, nil
			case 2:
				require.EqualValues(t, 2, page.From)
				return d.ProcessInstancePage{
					Request:       page,
					OverflowState: d.ProcessInstanceOverflowStateNoMore,
					Items: []d.ProcessInstance{
						{Key: "seed-2", EndDate: "2026-02-11"},
						{Key: "seed-1", EndDate: "2026-02-12"},
					},
				}, nil
			default:
				t.Fatalf("unexpected search page %d", searches)
				return d.ProcessInstancePage{}, nil
			}
		},
	}

	got, err := DiscoverRetentionProcessInstances(context.Background(), api, RetentionDiscoveryRequest{
		Filter: d.ProcessInstanceFilter{
			EndDateBefore: "2026-02-13",
		},
		BatchSize: 2,
	})

	require.NoError(t, err)
	require.Equal(t, 2, searches)
	require.Equal(t, []d.ProcessInstance{
		{Key: "seed-1", EndDate: "2026-02-12"},
		{Key: "seed-2", EndDate: "2026-02-11"},
		{Key: "seed-1", EndDate: "2026-02-12"},
	}, got.Items)
	require.Equal(t, []string{"seed-1", "seed-2"}, []string(got.Keys))
}

func TestDiscoverRetentionProcessInstancesLimitCountsEligibleSeeds(t *testing.T) {
	t.Parallel()

	api := stubRetentionDiscoveryAPI{
		searchPage: func(_ context.Context, _ d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateHasMore,
				Items: []d.ProcessInstance{
					{Key: "seed-1", EndDate: "2026-02-12"},
					{Key: "seed-2", EndDate: "2026-02-11"},
					{Key: "seed-3", EndDate: "2026-02-10"},
				},
			}, nil
		},
	}

	got, err := DiscoverRetentionProcessInstances(context.Background(), api, RetentionDiscoveryRequest{Limit: 2})

	require.NoError(t, err)
	require.Equal(t, []string{"seed-1", "seed-2"}, []string(got.Keys))
}

func TestDiscoverRetentionProcessInstancesPreservesSelectionFilters(t *testing.T) {
	t.Parallel()

	hasParent := false
	hasIncident := true
	wantFilter := d.ProcessInstanceFilter{
		BpmnProcessId:        "invoice",
		ProcessVersion:       7,
		ProcessVersionTag:    "stable",
		ProcessDefinitionKey: "2251799813685201",
		State:                d.StateCompleted,
		ParentKey:            "2251799813685249",
		HasParent:            &hasParent,
		HasIncident:          &hasIncident,
		EndDateBefore:        "2026-02-13",
	}
	api := stubRetentionDiscoveryAPI{
		searchPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, _ ...services.CallOption) (d.ProcessInstancePage, error) {
			require.Equal(t, wantFilter, filter)
			require.EqualValues(t, 25, page.Size)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateNoMore,
				Items: []d.ProcessInstance{
					{Key: "seed-1", EndDate: "2026-02-12"},
				},
			}, nil
		},
	}

	got, err := DiscoverRetentionProcessInstances(context.Background(), api, RetentionDiscoveryRequest{
		Filter:    wantFilter,
		BatchSize: 25,
		Limit:     1,
	})

	require.NoError(t, err)
	require.Equal(t, wantFilter, got.Filter)
	require.Equal(t, []string{"seed-1"}, []string(got.Keys))
}

type stubRetentionDiscoveryAPI struct {
	searchPage func(context.Context, d.ProcessInstanceFilter, d.ProcessInstancePageRequest, ...services.CallOption) (d.ProcessInstancePage, error)
}

func (s stubRetentionDiscoveryAPI) SearchForProcessInstancesPage(ctx context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
	return s.searchPage(ctx, filter, page, opts...)
}
