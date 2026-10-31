// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestJobBusinessIDFacade retains observed absence/empty values and detaches domain pointers at the public API.
func TestJobBusinessIDFacade(t *testing.T) {
	t.Parallel()
	for _, id := range []*string{nil, toolx.Ptr(""), toolx.Ptr("old,\"ID\"\n")} {
		original := toolx.CopyPtr(id)
		api := New(fakeJobService{get: func(context.Context, string, ...services.CallOption) (d.Job, error) {
			return d.Job{Key: "1", BusinessId: id}, nil
		}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
		got, err := api.GetJob(context.Background(), "1")
		require.NoError(t, err)
		require.Equal(t, original, got.BusinessId)
		raw, err := json.Marshal(got)
		require.NoError(t, err)
		var fields map[string]any
		require.NoError(t, json.Unmarshal(raw, &fields))
		if id == nil {
			require.NotContains(t, fields, "businessId")
		} else {
			require.Equal(t, *original, fields["businessId"])
			*id = "changed"
			require.Equal(t, original, got.BusinessId)
		}
	}
}

// TestCompletionBusinessIDRoundTrip copies candidate pointers in both plan directions and partial outcome reports.
func TestCompletionBusinessIDRoundTrip(t *testing.T) {
	t.Parallel()
	for _, id := range []*string{nil, toolx.Ptr(""), toolx.Ptr("old")} {
		original := toolx.CopyPtr(id)
		source := d.PreparedCompletionBatch{Candidates: []d.CompletionCandidate{{Key: "1", BusinessId: id}}}
		public, err := fromDomainPreparedCompletionBatch(source)
		require.NoError(t, err)
		roundtrip, err := toDomainPreparedCompletionBatch(public)
		require.NoError(t, err)
		report := fromDomainCompletionBatchReport(d.CompletionBatchReport{Outcomes: []d.CompletionOutcome{{Candidate: source.Candidates[0]}}})
		require.Equal(t, original, public.Candidates[0].BusinessId)
		require.Equal(t, original, roundtrip.Candidates[0].BusinessId)
		require.Equal(t, original, report.Outcomes[0].Candidate.BusinessId)
		if id != nil {
			*id = "source changed"
			require.Equal(t, original, public.Candidates[0].BusinessId)
			require.Equal(t, original, report.Outcomes[0].Candidate.BusinessId)
			*public.Candidates[0].BusinessId = "public changed"
			require.Equal(t, original, roundtrip.Candidates[0].BusinessId)
		}
	}
}
