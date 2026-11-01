// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"testing"

	native "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestJobBusinessIDNativeRead copies backend identity for native search-backed lookups, including nil and empty.
func TestJobBusinessIDNativeRead(t *testing.T) {
	t.Parallel()
	for _, id := range []*string{nil, toolx.Ptr(""), toolx.Ptr("old,\"ID\"\n")} {
		original := toolx.CopyPtr(id)
		calls := 0
		svc := newJobServiceTest(t, &mockJobClient{searchJobsWithResponse: func(context.Context, native.SearchJobsJSONRequestBody, ...native.RequestEditorFn) (*native.SearchJobsResponse, error) {
			calls++
			return &native.SearchJobsResponse{HTTPResponse: okHTTPResponse(), JSON200: &native.JobSearchQueryResult{Items: []native.JobSearchResult{{JobKey: "1", ProcessInstanceKey: "2", BusinessId: id}}}}, nil
		}})
		got, err := svc.GetJob(context.Background(), "1")
		require.NoError(t, err)
		require.Equal(t, 1, calls)
		require.Equal(t, original, got.BusinessId)
		if id != nil {
			*id = "changed"
			require.Equal(t, original, got.BusinessId)
		}
	}
}
