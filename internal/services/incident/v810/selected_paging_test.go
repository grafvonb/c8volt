// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	camunda "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDResolveIncidentPaging proves scoped incident planning exhausts large/sparse pages and rejects cycles or later errors.
func TestBusinessIDResolveIncidentPaging(t *testing.T) {
	for _, mode := range []string{"large", "sparse", "cycle", "offset-cycle", "late-error", "missing-key", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			svc := newTestService(t, mockIncidentClient{searchProcessInstanceIncidents: func(_ context.Context, _ camunda.ProcessInstanceKey, body camunda.SearchProcessInstanceIncidentsJSONRequestBody, _ ...camunda.RequestEditorFn) (*camunda.SearchProcessInstanceIncidentsResponse, error) {
				calls++
				if mode == "late-error" && calls == 2 {
					return nil, d.ErrUnavailable
				}
				items := []camunda.IncidentResult{}
				total := int64(1)
				more := false
				cursor := ""
				if mode == "large" {
					total = 1001
					start, end := 0, 1000
					if calls == 2 {
						start, end = 1000, 1001
						page, err := body.Page.AsCursorForwardPagination()
						require.NoError(t, err)
						require.Equal(t, "second", string(*page.After))
					} else {
						more = true
						cursor = "second"
					}
					for i := start; i < end; i++ {
						items = append(items, camunda.IncidentResult{IncidentKey: fmt.Sprint(i + 1), ProcessInstanceKey: "p", State: camunda.IncidentStateEnumACTIVE})
					}
				} else if mode == "sparse" || mode == "cycle" {
					total = 1
					if calls == 1 || mode == "cycle" {
						more = true
						cursor = "second"
					} else {
						items = append(items, camunda.IncidentResult{IncidentKey: "1", ProcessInstanceKey: "p"})
					}
				} else {
					key := "1"
					if mode == "missing-key" {
						key = ""
					}
					items = append(items, camunda.IncidentResult{IncidentKey: key, ProcessInstanceKey: "p"})
					if mode == "offset-cycle" || mode == "late-error" {
						more = true
					}
				}
				if mode == "cancel" {
					cancel()
				}
				return &camunda.SearchProcessInstanceIncidentsResponse{HTTPResponse: testHTTPResponse(http.StatusOK), JSON200: &camunda.IncidentSearchQueryResult{Items: items, Page: camunda.SearchQueryPageResponse{TotalItems: total, HasMoreTotalItems: more, EndCursor: toolx.Ptr(cursor)}}}, nil
			}})
			result, err := svc.SearchProcessInstanceIncidents(ctx, "p")
			switch mode {
			case "large":
				require.NoError(t, err)
				require.Len(t, result, 1001)
				require.Equal(t, 2, calls)
			case "sparse":
				require.NoError(t, err)
				require.Len(t, result, 1)
				require.Equal(t, 2, calls)
			case "late-error":
				require.ErrorIs(t, err, d.ErrUnavailable)
				require.Len(t, result, 1)
			case "cancel":
				require.ErrorIs(t, err, context.Canceled)
			default:
				require.ErrorIs(t, err, d.ErrMalformedResponse)
			}
		})
	}
}
