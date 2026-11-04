// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDOperationsSlowFacade preserves exact criteria, mode and partial observations through copies and JSON.
func TestBusinessIDOperationsSlowFacade(t *testing.T) {
	for _, mode := range []SlowProcessAnalysisSelectionMode{SlowProcessAnalysisSelectionModeProcessInstanceSearch, SlowProcessAnalysisSelectionModeProcessDefinitionSearch, SlowProcessAnalysisSelectionModeExplicitKeys} {
		for _, bid := range []*string{nil, new(" ref,Ω *\n <tag>| "), new("")} {
			t.Run(fmt.Sprintf("%s/%v", mode, bid), func(t *testing.T) {
				var source d.SlowProcessAnalysisResult
				var original string
				if bid != nil {
					original = *bid
				}
				api := stubOpsService{slowProcessAnalysis: func(_ context.Context, request d.SlowProcessAnalysisRequest, opts ...services.CallOption) (d.SlowProcessAnalysisResult, error) {
					require.True(t, services.ApplyCallOptions(opts).FailFast)
					require.Equal(t, string(mode), string(request.SelectionMode))
					require.Equal(t, bid, request.ProcessInstanceFilters.BusinessId)
					if bid != nil {
						require.NotSame(t, bid, request.ProcessInstanceFilters.BusinessId)
						*bid = "changed caller"
					}
					source = d.SlowProcessAnalysisResult{Request: request, Count: 1, Items: []d.SlowProcessAnalysisProcessInstance{{Key: "101", BusinessId: new("observed")}},
						Selection: &d.ProcessInstanceSelection{Request: d.ProcessInstanceSelectionRequest{Cardinality: d.ProcessInstanceSelectionCollection, SearchRequest: d.ProcessInstanceSearchRequest{Filter: d.ProcessInstanceFilter{BusinessId: request.ProcessInstanceFilters.BusinessId}}},
							Items: []d.ProcessInstance{{Key: "101", Variables: map[string]any{"nested": map[string]any{"value": "original"}}}}, Keys: []string{"101"}, Completion: d.ProcessInstanceSearchCompletionIncomplete},
					}
					return source, fmt.Errorf("%w: partial analysis", d.ErrNotFound)
				}}
				got, err := New(api, nil).AnalyseSlowProcessInstances(context.Background(), SlowProcessAnalysisRequest{SelectionMode: mode, ProcessInstanceFilters: SlowProcessAnalysisProcessInstanceSearchFilters{BusinessId: bid}}, foptions.WithFailFast())
				require.ErrorIs(t, err, ferrors.ErrNotFound)
				require.Equal(t, 1, got.Count)
				require.Equal(t, "observed", *got.Items[0].BusinessId)
				if bid != nil {
					require.Equal(t, original, *got.Request.ProcessInstanceFilters.BusinessId)
					*source.Request.ProcessInstanceFilters.BusinessId = "changed service"
					require.Equal(t, original, *got.Request.ProcessInstanceFilters.BusinessId)
					require.Equal(t, original, *got.Selection.Request.SearchRequest.Filter.BusinessId)
				}
				source.Selection.Keys[0] = "changed"
				source.Selection.Items[0].Variables["nested"].(map[string]any)["value"] = "changed"
				require.Equal(t, []string{"101"}, []string(got.Selection.Keys))
				require.Equal(t, "original", got.Selection.Items[0].Variables["nested"].(map[string]any)["value"])
				data, err := json.Marshal(got)
				require.NoError(t, err)
				var roundtrip SlowProcessAnalysisResult
				require.NoError(t, json.Unmarshal(data, &roundtrip))
				require.Equal(t, got.Request, roundtrip.Request)
				require.Equal(t, got.Selection, roundtrip.Selection)
				require.Equal(t, got.Items, roundtrip.Items)
				if bid == nil {
					require.NotContains(t, string(data), `"processInstanceFilters":{"businessId"`)
				}
			})
		}
	}
}
