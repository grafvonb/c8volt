// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"
	"io"
	"log/slog"
	"testing"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDSelectionFacade retains copied selection evidence, option mapping and original failure classification.
func TestBusinessIDSelectionFacade(t *testing.T) {
	t.Parallel()
	for _, ignore := range []bool{false, true} {
		for _, fail := range []bool{false, true} {
			bid := " exact,* Ω "
			observed := "different"
			calls := 0
			api := stubProcessInstanceAPI{searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
				calls++
				require.Equal(t, ignore, services.ApplyCallOptions(opts).IgnoreTenant)
				require.Equal(t, bid, *filter.BusinessId)
				if calls == 2 {
					return d.ProcessInstancePage{}, d.ErrForbidden
				}
				overflow := d.ProcessInstanceOverflowStateNoMore
				if fail {
					overflow = d.ProcessInstanceOverflowStateHasMore
				}
				return d.ProcessInstancePage{Request: page, Items: []d.ProcessInstance{{Key: "1", BusinessId: &observed, TenantId: "tenant-a", Variables: map[string]any{"nested": map[string]any{"value": "original"}}}}, OverflowState: overflow}, nil
			}}
			cli := New(nil, api, stubIncidentAPI{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			opts := []options.FacadeOption{}
			if ignore {
				opts = append(opts, options.WithIgnoreTenant())
			}
			result, err := cli.ResolveProcessInstanceSelection(context.Background(), ProcessInstanceSelectionRequest{Cardinality: ProcessInstanceSelectionCollection, SearchRequest: ProcessInstanceSearchRequest{Filter: ProcessInstanceFilter{BusinessId: &bid}}}, opts...)
			if fail {
				require.ErrorIs(t, err, ferr.ErrLocalPrecondition)
				require.Equal(t, ProcessInstanceSearchCompletionIncomplete, result.Completion)
			} else {
				require.NoError(t, err)
				require.Equal(t, ProcessInstanceSearchCompletionExhausted, result.Completion)
			}
			bid = "changed"
			observed = "changed"
			require.Equal(t, " exact,* Ω ", *result.Request.SearchRequest.Filter.BusinessId)
			require.Equal(t, "different", *result.Items[0].BusinessId)
			require.Equal(t, "search", result.Origin)
			require.Equal(t, ignore, result.IgnoreTenant)
			require.Equal(t, []string{"tenant-a"}, result.TenantEvidence.ResolvedTenantIDs)
		}
	}
}
