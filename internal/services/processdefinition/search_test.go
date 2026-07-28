// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processdefinition

import (
	"context"
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

type processDefinitionSearchAPIStub struct {
	searchProcessDefinitionsPage func(context.Context, d.ProcessDefinitionFilter, d.ProcessDefinitionPageRequest, ...services.CallOption) (d.ProcessDefinitionPage, error)
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

func (processDefinitionSearchAPIStub) SearchProcessDefinitionsLatest(context.Context, d.ProcessDefinitionFilter, ...services.CallOption) ([]d.ProcessDefinition, error) {
	panic("unexpected SearchProcessDefinitionsLatest call")
}

func (processDefinitionSearchAPIStub) GetProcessDefinition(context.Context, string, ...services.CallOption) (d.ProcessDefinition, error) {
	panic("unexpected GetProcessDefinition call")
}

func (processDefinitionSearchAPIStub) GetProcessDefinitionXML(context.Context, string, ...services.CallOption) (string, error) {
	panic("unexpected GetProcessDefinitionXML call")
}
