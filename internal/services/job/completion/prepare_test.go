// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package completion

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

// TestBulkCompletionPrepareSearch freezes first-seen unique matches, retaining raw traversal through sparse pages.
func TestBulkCompletionPrepareSearch(t *testing.T) {
	t.Parallel()
	zero := int32(0)
	request := d.CompletionRequest{Selection: d.CompletionSelection{Mode: d.CompletionSelectionSearch, InputForm: d.CompletionInputFilters, BatchSize: 3, Limit: 2, Filter: d.CompletionFilter{Kind: "task_listener", State: " created ", Retries: &zero}}}
	calls := 0
	ports := ReadPorts{Search: func(_ context.Context, q d.JobSearchQuery, p d.JobPageRequest, opts ...services.CallOption) (d.JobSearchPage, error) {
		require.True(t, q.SortByKey)
		require.Zero(t, q.Limit)
		require.Equal(t, "TASK_LISTENER", q.Kind)
		require.Equal(t, int32(0), *q.Retries)
		require.True(t, services.ApplyCallOptions(opts).DryRun)
		require.Equal(t, int32(calls*3), p.From)
		require.Equal(t, int32(3), p.Size)
		calls++
		item := d.Job{Key: "1", Kind: "TASK_LISTENER", State: "CREATED", TenantId: "tenant"}
		rows := []d.Job{item, item, {Key: "9", Kind: "TASK_LISTENER", State: "CREATED", TenantId: "other"}}
		if calls == 2 {
			rows = []d.Job{{Key: "8", Kind: "BPMN_ELEMENT", State: "CREATED", TenantId: "tenant"}}
		}
		if calls == 3 {
			item.Key = "2"
			item.State = "COMPLETED"
			rows = []d.Job{item}
			item.State = "CREATED"
			rows = append(rows, item)
		}
		return d.JobSearchPage{Request: p, Items: rows, RawItemCount: 3, NextFrom: p.From + 3, OverflowState: d.ProcessInstanceOverflowStateHasMore}, nil
	}}
	batch, err := Prepare(context.Background(), request, "tenant", config.BackoffConfig{}, ports, services.WithDryRun())
	require.NoError(t, err)
	require.Equal(t, 3, calls)
	require.Equal(t, d.CompletionPreparationComplete, batch.Status)
	require.Equal(t, d.CompletionReasonLimitReached, batch.CompletionReason)
	require.True(t, batch.Limited)
	require.Equal(t, int64(9), batch.ObservedCount)
	require.Equal(t, []string{"tenant"}, batch.TenantIDs)
	require.Equal(t, []string{"1", "2"}, []string{batch.Candidates[0].Key, batch.Candidates[1].Key})
	zero = 7
	require.Zero(t, *batch.Selection.Filter.Retries)
}

// TestBulkCompletionPrepareExplicit preserves string keys, ordered deduplication and cross-tenant context without a search.
func TestBulkCompletionPrepareExplicit(t *testing.T) {
	t.Parallel()
	request := d.CompletionRequest{Selection: d.CompletionSelection{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputCollection, Keys: []string{"9007199254740993", "2", "9007199254740993"}}, Variables: map[string]any{"value": "original"}}
	var reads []string
	batch, err := Prepare(context.Background(), request, "configured", config.BackoffConfig{}, ReadPorts{Get: func(_ context.Context, key string, opts ...services.CallOption) (d.Job, error) {
		require.True(t, services.ApplyCallOptions(opts).NoWait)
		reads = append(reads, key)
		return d.Job{Key: key, TenantId: key, State: "CANCELED", Retries: 0}, nil
	}}, services.WithNoWait())
	require.NoError(t, err)
	require.Equal(t, []string{"9007199254740993", "2"}, reads)
	require.Equal(t, reads, batch.TenantIDs)
	require.Equal(t, d.CompletionReasonExplicitInputsRead, batch.CompletionReason)
	require.Equal(t, request.Selection.Keys, batch.Selection.Keys)
	request.Selection.Keys[0] = "99"
	request.Variables["value"] = "changed"
	require.Equal(t, "9007199254740993", batch.Selection.Keys[0])
	require.Equal(t, "original", batch.Variables["value"])
	require.Len(t, batch.Candidates, 2)
}

// TestBulkCompletionPrepareInvalid validates the whole request before any read and never exposes executable partial work.
func TestBulkCompletionPrepareInvalid(t *testing.T) {
	t.Parallel()
	cases := []d.CompletionSelection{
		{},
		{Mode: d.CompletionSelectionSearch, InputForm: d.CompletionInputFilters},
		{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputCollection, Keys: []string{"1", "0"}},
		{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputCollection, Keys: []string{"1", "9223372036854775808"}},
		{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputCollection, Keys: []string{"1", ""}},
		{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputCollection, Keys: []string{"1"}, Filter: d.CompletionFilter{State: "CREATED"}},
		{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputCollection, Keys: []string{"1"}, Limit: 1},
		{Mode: d.CompletionSelectionSearch, InputForm: d.CompletionInputFilters, Filter: d.CompletionFilter{State: "bogus"}},
	}
	for i, selection := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			batch, err := Prepare(context.Background(), d.CompletionRequest{Selection: selection}, "", config.BackoffConfig{}, ReadPorts{})
			require.ErrorIs(t, err, d.ErrValidation)
			require.Equal(t, d.CompletionPreparationIncomplete, batch.Status)
		})
	}
}

// TestBulkCompletionPrepareFailure retains partial evidence but rejects unreadable, incompatible or canceled selections.
func TestBulkCompletionPrepareFailure(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"read", "listener", "canceled", "wrong_key", "invalid_payload"} {
		t.Run(failure, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			request := d.CompletionRequest{Selection: d.CompletionSelection{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputCollection, Keys: []string{"1", "2"}}, Variables: map[string]any{"x": true}}
			if failure == "invalid_payload" {
				request.Variables["x"] = make(chan int)
			}
			batch, err := Prepare(ctx, request, "", config.BackoffConfig{}, ReadPorts{Get: func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
				calls++
				if key == "2" {
					switch failure {
					case "read":
						return d.Job{}, d.ErrNotFound
					case "listener":
						return d.Job{Key: key, Kind: d.JobKindTaskListener}, nil
					case "canceled":
						cancel()
					case "wrong_key":
						return d.Job{Key: "3"}, nil
					}
				}
				return d.Job{Key: key}, nil
			}})
			require.Error(t, err)
			require.Equal(t, d.CompletionPreparationIncomplete, batch.Status)
			require.Empty(t, batch.CompletionReason)
			if failure == "invalid_payload" {
				require.Zero(t, calls)
			} else {
				require.Equal(t, 2, calls)
			}
			if failure == "read" {
				require.ErrorIs(t, err, d.ErrNotFound)
			}
			if failure == "canceled" {
				require.ErrorIs(t, err, context.Canceled)
			}
		})
	}
}

// TestBulkCompletionPrepareEmpty distinguishes authoritative empty sources from absent selection.
func TestBulkCompletionPrepareEmpty(t *testing.T) {
	t.Parallel()
	for _, mode := range []d.CompletionSelectionMode{d.CompletionSelectionExplicit, d.CompletionSelectionSearch} {
		t.Run(string(mode), func(t *testing.T) {
			selection := d.CompletionSelection{Mode: mode, InputForm: d.CompletionInputStdin}
			if mode == d.CompletionSelectionSearch {
				selection.InputForm = d.CompletionInputFilters
				selection.Filter.Type = "none"
			}
			batch, err := Prepare(context.Background(), d.CompletionRequest{Selection: selection}, "", config.BackoffConfig{}, ReadPorts{Search: func(_ context.Context, _ d.JobSearchQuery, p d.JobPageRequest, _ ...services.CallOption) (d.JobSearchPage, error) {
				return d.JobSearchPage{Request: p, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
			}})
			require.NoError(t, err)
			require.Equal(t, d.CompletionPreparationComplete, batch.Status)
			require.NotNil(t, batch.Candidates)
			require.Empty(t, batch.Candidates)
		})
	}
}

// TestBulkCompletionPrepareIncomplete rejects unsafe continuation and propagates failures after selected rows.
func TestBulkCompletionPrepareIncomplete(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"error", "unknown", "nonadvancing", "cycle", "timeout", "caller_deadline"} {
		t.Run(failure, func(t *testing.T) {
			ctx := context.Background()
			if failure == "caller_deadline" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 5*time.Millisecond)
				defer cancel()
			}
			calls := 0
			batch, err := Prepare(ctx, d.CompletionRequest{Selection: d.CompletionSelection{Mode: d.CompletionSelectionSearch, InputForm: d.CompletionInputFilters, Filter: d.CompletionFilter{Type: "wanted"}}}, "", config.BackoffConfig{Timeout: 10 * time.Millisecond}, ReadPorts{Search: func(ctx context.Context, _ d.JobSearchQuery, p d.JobPageRequest, _ ...services.CallOption) (d.JobSearchPage, error) {
				calls++
				page := d.JobSearchPage{Request: p, OverflowState: d.ProcessInstanceOverflowStateHasMore, EndCursor: fmt.Sprint(calls)}
				switch failure {
				case "error":
					if calls == 2 {
						return page, d.ErrNotFound
					}
					page.Items = []d.Job{{Key: "1", Type: "wanted"}}
					page.RawItemCount = 1
					page.NextFrom = 1
				case "unknown":
					page.OverflowState = ""
				case "nonadvancing":
					page.EndCursor = ""
				case "cycle":
					page.EndCursor = "same"
				case "timeout", "caller_deadline":
					if calls > 1 {
						<-ctx.Done()
						return page, ctx.Err()
					}
				}
				return page, nil
			}})
			require.Error(t, err)
			require.Equal(t, d.CompletionPreparationIncomplete, batch.Status)
			require.Empty(t, batch.CompletionReason)
			if failure == "error" {
				require.ErrorIs(t, err, d.ErrNotFound)
				require.Len(t, batch.Candidates, 1)
			} else if failure == "caller_deadline" {
				require.True(t, errors.Is(err, context.DeadlineExceeded))
			} else {
				require.ErrorIs(t, err, d.ErrMalformedResponse)
			}
		})
	}
}

// TestBulkCompletionPreparePredicates checks every AND predicate independently,
// including zero retries and available candidate context without lifecycle dates.
func TestBulkCompletionPreparePredicates(t *testing.T) {
	t.Parallel()
	zero := int32(0)
	filter := d.CompletionFilter{State: "CREATED", Type: "work", ProcessInstanceKey: "11", ElementInstanceKey: "12", ElementId: "element", Worker: "worker", Retries: &zero, Kind: "TASK_LISTENER", ListenerEventType: "CANCELING"}
	match := d.Job{Key: "1", State: filter.State, Type: filter.Type, ProcessInstanceKey: "11", ElementInstanceKey: "12", ElementId: filter.ElementId, Worker: filter.Worker, Kind: filter.Kind, ListenerEventType: filter.ListenerEventType, TenantId: "tenant"}
	changes := []func(*d.Job){
		func(j *d.Job) { j.State = "COMPLETED" }, func(j *d.Job) { j.Type = "other" }, func(j *d.Job) { j.ProcessInstanceKey = "99" }, func(j *d.Job) { j.ElementInstanceKey = "99" }, func(j *d.Job) { j.ElementId = "other" }, func(j *d.Job) { j.Worker = "other" }, func(j *d.Job) { j.Retries = 1 }, func(j *d.Job) { j.Kind = "BPMN_ELEMENT" }, func(j *d.Job) { j.ListenerEventType = "COMPLETING" }, func(j *d.Job) { j.TenantId = "other" },
	}
	rows := []d.Job{}
	for i, change := range changes {
		row := match
		row.Key = fmt.Sprint(i + 2)
		change(&row)
		rows = append(rows, row)
	}
	rows = append(rows, match)
	batch, err := Prepare(context.Background(), d.CompletionRequest{Selection: d.CompletionSelection{Mode: d.CompletionSelectionSearch, InputForm: d.CompletionInputFilters, Filter: filter}}, "tenant", config.BackoffConfig{}, ReadPorts{Search: func(_ context.Context, _ d.JobSearchQuery, p d.JobPageRequest, _ ...services.CallOption) (d.JobSearchPage, error) {
		return d.JobSearchPage{Request: p, Items: rows, RawItemCount: int32(len(rows)), NextFrom: int32(len(rows)), OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
	}})
	require.NoError(t, err)
	require.Len(t, batch.Candidates, 1)
	require.Equal(t, "1", batch.Candidates[0].Key)
	require.Equal(t, "11", batch.Candidates[0].ProcessInstanceKey)
	require.Equal(t, "12", batch.Candidates[0].ElementInstanceKey)
	require.Equal(t, "worker", batch.Candidates[0].Worker)
	require.Equal(t, d.CompletionReasonExhausted, batch.CompletionReason)
}

// TestBulkCompletionPrepareCursorContinuation permits empty advancing pages followed by matches and proves exhaustion.
func TestBulkCompletionPrepareCursorContinuation(t *testing.T) {
	t.Parallel()
	calls := 0
	batch, err := Prepare(context.Background(), d.CompletionRequest{Selection: d.CompletionSelection{Mode: d.CompletionSelectionSearch, InputForm: d.CompletionInputFilters, Filter: d.CompletionFilter{Type: "work"}, BatchSize: 2}}, "", config.BackoffConfig{}, ReadPorts{Search: func(_ context.Context, _ d.JobSearchQuery, p d.JobPageRequest, _ ...services.CallOption) (d.JobSearchPage, error) {
		calls++
		switch calls {
		case 1:
			return d.JobSearchPage{Request: p, OverflowState: d.ProcessInstanceOverflowStateHasMore, EndCursor: "a"}, nil
		case 2:
			require.Equal(t, "a", p.After)
			require.Zero(t, p.From)
			return d.JobSearchPage{Request: p, Items: []d.Job{{Key: "1", Type: "work", TenantId: "a"}, {Key: "2", Type: "work", TenantId: "b"}}, RawItemCount: 2, NextFrom: 2, EndCursor: "b", OverflowState: d.ProcessInstanceOverflowStateHasMore}, nil
		default:
			require.Equal(t, "b", p.After)
			require.Equal(t, int32(2), p.From)
			return d.JobSearchPage{Request: p, NextFrom: 2, OverflowState: d.ProcessInstanceOverflowStateNoMore}, nil
		}
	}})
	require.NoError(t, err)
	require.Equal(t, 3, calls)
	require.Len(t, batch.Candidates, 2)
	require.Equal(t, []string{"a", "b"}, batch.TenantIDs)
	require.Equal(t, d.CompletionReasonExhausted, batch.CompletionReason)
}

// TestBulkCompletionPrepareCanonicalKeys prevents equivalent decimal spellings from creating duplicate targets.
func TestBulkCompletionPrepareCanonicalKeys(t *testing.T) {
	t.Parallel()
	calls := 0
	batch, err := Prepare(context.Background(), d.CompletionRequest{Selection: d.CompletionSelection{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputCollection, Keys: []string{"0001", "1"}}}, "", config.BackoffConfig{}, ReadPorts{Get: func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
		calls++
		require.Equal(t, "1", key)
		return d.Job{Key: "1"}, nil
	}})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Len(t, batch.Candidates, 1)
	require.Equal(t, []string{"1", "1"}, batch.Selection.Keys)
}

// TestBulkCompletionPrepareMalformedRawEvidence blocks an adapter page that contradicts its raw progress or totals.
func TestBulkCompletionPrepareMalformedRawEvidence(t *testing.T) {
	t.Parallel()
	for _, issue := range []string{"no_raw_progress", "exact_total", "lower_bound", "unknown_total"} {
		t.Run(issue, func(t *testing.T) {
			batch, err := Prepare(context.Background(), d.CompletionRequest{Selection: d.CompletionSelection{Mode: d.CompletionSelectionSearch, InputForm: d.CompletionInputFilters, Filter: d.CompletionFilter{Type: "work"}}}, "", config.BackoffConfig{}, ReadPorts{Search: func(_ context.Context, _ d.JobSearchQuery, p d.JobPageRequest, _ ...services.CallOption) (d.JobSearchPage, error) {
				page := d.JobSearchPage{Request: p, Items: []d.Job{{Key: "1", Type: "work"}}, RawItemCount: 1, NextFrom: 1, OverflowState: d.ProcessInstanceOverflowStateNoMore}
				switch issue {
				case "no_raw_progress":
					page.NextFrom = 0
				case "exact_total":
					page.ReportedTotal = &d.JobReportedTotal{Count: 2, Kind: d.JobReportedTotalKindExact}
				case "lower_bound":
					page.ReportedTotal = &d.JobReportedTotal{Count: 1, Kind: d.JobReportedTotalKindLowerBound}
				case "unknown_total":
					page.ReportedTotal = &d.JobReportedTotal{Count: 1}
				}
				return page, nil
			}})
			require.ErrorIs(t, err, d.ErrMalformedResponse)
			require.Equal(t, d.CompletionPreparationIncomplete, batch.Status)
		})
	}
}

// TestBulkCompletionPrepareCanceledEmpty rejects cancellation even when the source would otherwise be empty.
func TestBulkCompletionPrepareCanceledEmpty(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	batch, err := Prepare(ctx, d.CompletionRequest{Selection: d.CompletionSelection{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputStdin}}, "", config.BackoffConfig{}, ReadPorts{})
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, d.CompletionPreparationIncomplete, batch.Status)
	require.Empty(t, batch.CompletionReason)
}
