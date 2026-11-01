// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package completion

import (
	"context"
	"testing"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestCompletionBusinessIDSnapshot isolates preparation, preview and execution identities from mutable sources and later observations.
func TestCompletionBusinessIDSnapshot(t *testing.T) {
	t.Parallel()
	for _, id := range []*string{nil, toolx.Ptr(""), toolx.Ptr("old,\"ID\"\n")} {
		for _, search := range []bool{false, true} {
			original := toolx.CopyPtr(id)
			source := d.Job{Key: "1", State: "CREATED", Type: "work", BusinessId: toolx.CopyPtr(id)}
			selection := d.CompletionSelection{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputCollection, Keys: []string{"1"}}
			if search {
				selection = d.CompletionSelection{Mode: d.CompletionSelectionSearch, InputForm: d.CompletionInputFilters, Filter: d.CompletionFilter{Type: "work"}}
			}
			batch, err := Prepare(context.Background(), d.CompletionRequest{Selection: selection}, "", config.BackoffConfig{}, ReadPorts{
				Get: func(context.Context, string, ...services.CallOption) (d.Job, error) { return source, nil },
				Search: func(_ context.Context, _ d.JobSearchQuery, p d.JobPageRequest, _ ...services.CallOption) (d.JobSearchPage, error) {
					return d.JobSearchPage{Request: p, Items: []d.Job{source}, RawItemCount: 1, NextFrom: 1, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
				},
			})
			require.NoError(t, err)
			require.Equal(t, original, batch.Candidates[0].BusinessId)
			if source.BusinessId != nil {
				*source.BusinessId = "source changed"
				require.Equal(t, original, batch.Candidates[0].BusinessId)
			}
			preview, err := Execute(context.Background(), batch, "", config.BackoffConfig{}, ExecutionPorts{}, 1, services.WithDryRun())
			require.NoError(t, err)
			if batch.Candidates[0].BusinessId != nil {
				*batch.Candidates[0].BusinessId = "plan changed"
				require.Equal(t, original, preview.Candidates[0].BusinessId)
				batch.Candidates[0].BusinessId = toolx.CopyPtr(original)
			}
			report, err := Execute(context.Background(), batch, "", config.BackoffConfig{}, ExecutionPorts{
				Get: func(context.Context, string, ...services.CallOption) (d.Job, error) {
					// Execute has already snapshotted the plan before invoking this synchronous port.
					if batch.Candidates[0].BusinessId != nil {
						*batch.Candidates[0].BusinessId = "caller changed"
					}
					return d.Job{Key: "1", State: "CREATED", Type: "work", BusinessId: toolx.Ptr("new")}, nil
				},
				Submit: func(context.Context, d.Job, map[string]any) (d.CompletionSubmission, error) {
					return d.CompletionSubmission{Status: d.CompletionSubmissionAccepted}, nil
				},
			}, 1, services.WithNoWait())
			require.NoError(t, err)
			require.Len(t, report.Outcomes, 1)
			require.Equal(t, original, report.Outcomes[0].Candidate.BusinessId)
		}
	}
}
